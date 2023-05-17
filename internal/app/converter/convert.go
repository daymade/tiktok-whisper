package converter

import (
	"fmt"
	"github.com/samber/lo"
	"log"
	"path/filepath"
	"strings"
	"sync"
	"tiktok-whisper/internal/app/api"
	"tiktok-whisper/internal/app/audio"
	"tiktok-whisper/internal/app/model"
	"tiktok-whisper/internal/app/repository"
	"tiktok-whisper/internal/app/util/files"
	"time"
)

type Converter struct {
	transcriber api.Transcriber
	db          repository.TranscriptionDAO
}

func NewConverter(transcriber api.Transcriber, transcriptionDAO repository.TranscriptionDAO) *Converter {
	return &Converter{
		transcriber: transcriber,
		db:          transcriptionDAO,
	}
}

func (c *Converter) Close() error {
	return c.db.Close()
}

// ConvertAudioDir converts audio files in a directory to text in parallel.
// It takes the directory, the file extension of the audios, the output directory,
// and the number of parallel conversions as parameters.
func (c *Converter) ConvertAudioDir(directory string, extension string, outputDirectory string, parallel int) error {
	absDir, err := files.GetAbsolutePath(directory)
	if err != nil {
		log.Printf("Error getting absolute path of directory %s: %v\n", directory, err)
		return err
	}

	log.Printf("Starting to convert audio files in directory %s\n", absDir)

	// Get all files with specified extension in directory and sort them by old and new
	fileInfos, err := files.GetAllFiles(absDir, extension)
	if err != nil {
		log.Printf("Error getting all files in directory %s: %v\n", absDir, err)
		return err
	}

	files := lo.Map(fileInfos, func(f model.FileInfo, i int) string {
		return f.FullPath
	})

	log.Printf("Found %d files to convert\n", len(files))

	err = c.ConvertAudios(files, outputDirectory, parallel)
	if err != nil {
		log.Printf("Error converting audio files: %v\n", err)
		return err
	}

	log.Printf("Successfully converted all audio files\n")

	return nil
}

func (c *Converter) ConvertAudios(audioFiles []string, outputDirectory string, parallel int) error {
	transcriptionDirectory, err := filepath.Abs(outputDirectory)
	if err != nil {
		return err
	}

	var wg sync.WaitGroup
	sem := make(chan bool, parallel)

	for _, file := range audioFiles {
		wg.Add(1)
		go func(file string) {
			defer wg.Done()
			sem <- true
			c.processFile(file, transcriptionDirectory)
			<-sem
		}(file)
	}
	wg.Wait()
	return nil
}

func (c *Converter) processFile(audioAbsPath string, transcriptionDirectory string) {
	log.Printf("Start to process %s\n", audioAbsPath)

	transcription, err := c.transcriber.Transcript(audioAbsPath)
	if err != nil {
		log.Printf("Transcription error: %v\n", err)
		return
	}

	fileName := filepath.Base(audioAbsPath)
	fileNameWithoutExt := strings.TrimSuffix(fileName, filepath.Ext(fileName))
	transcriptionFileName := fileNameWithoutExt + ".txt"
	transcriptionFilepath := filepath.Join(transcriptionDirectory, transcriptionFileName)

	err = files.WriteToFile(transcription, transcriptionFilepath)
	if err != nil {
		log.Printf("Error writing to audioAbsPath: %v\n", err)
		return
	}
	log.Printf("Transcription saved to: %s\n", transcriptionFilepath)
}

// ConvertVideoDir converts videos in a directory to text in parallel.
// It takes the user's nickname, the input directory, the file extension of the videos,
// the maximum number of videos to convert, and the number of parallel conversions as parameters.
func (c *Converter) ConvertVideoDir(userNickname string, inputDir string, fileExtension string, convertCount int, parallel int) {
	// Check and create the data/mp3/userNickname subdirectory
	convertedMp3Dir := files.GetUserMp3Dir(userNickname)
	files.CheckAndCreateMP3Directory(convertedMp3Dir)

	// Get all MP4 files in the input directory and sort them by old and new
	fileInfos, err := files.GetAllFiles(inputDir, fileExtension)
	if err != nil {
		log.Fatalln(err)
	}

	filesToProcess := c.filterUnProcessedFiles(fileInfos, convertCount)

	var wg sync.WaitGroup
	sem := make(chan bool, parallel)

	for _, file := range filesToProcess {
		wg.Add(1)
		go func(file model.FileInfo) {
			defer wg.Done()
			sem <- true
			err := c.convertToText(userNickname, file)
			<-sem

			if err != nil {
				log.Printf("Error converting file %s: %v\n", file.Name, err)
			} else {
				log.Printf("Successfully converted file %s\n", file.Name)
			}
		}(file)
	}
	wg.Wait()
}

func (c *Converter) filterUnProcessedFiles(fileInfos []model.FileInfo, convertCount int) []model.FileInfo {
	filesToProcess := make([]model.FileInfo, 0, convertCount)

	for _, fileInfo := range fileInfos {
		// Check if the file has been processed
		id, err := c.db.CheckIfFileProcessed(fileInfo.Name)
		if err == nil {
			fmt.Printf("File '%s' with '%d' has already been processed, skipping...\n", fileInfo.Name, id)
			continue
		}

		filesToProcess = append(filesToProcess, fileInfo)
		if len(filesToProcess) >= convertCount {
			break
		}
	}
	return filesToProcess
}

func (c *Converter) convertToText(userNickname string, file model.FileInfo) error {
	fmt.Printf("Processing file '%s'\n", file.Name)

	// Convert MP4 to MP3 using FFmpeg
	mp3FileName := strings.TrimSuffix(file.Name, ".mp4") + ".mp3"
	mp3FilePath := filepath.Join(files.GetUserMp3Dir(userNickname), mp3FileName)

	// Check if the MP3 file already exists
	err := audio.ConvertToMp3(file, mp3FilePath)
	if err != nil {
		c.db.RecordToDB(userNickname, file.FullPath, file.Name, mp3FileName, 0, "",
			time.Now(), 1, fmt.Sprintf("FFmpeg error: %v", err))
		return fmt.Errorf("FFmpeg error: %v", err)
	}

	// Get audio duration
	duration, err := audio.GetAudioDuration(mp3FilePath)
	if err != nil {
		c.db.RecordToDB(userNickname, file.FullPath, file.Name, mp3FileName, 0, "",
			time.Now(), 1, fmt.Sprintf("Failed to get audio duration: %v", err))
		return fmt.Errorf("Failed to get audio duration: %v\n", err)
	}

	// Call Whisper with a new MP3 file path
	transcription, err := c.transcriber.Transcript(mp3FilePath)
	if err != nil {
		c.db.RecordToDB(userNickname, file.FullPath, file.Name, mp3FileName, duration, "",
			time.Now(), 1, fmt.Sprintf("Transcription error: %v", err))
		return fmt.Errorf("Transcription error: %v\n", err)
	}

	// Save conversion results to database
	c.db.RecordToDB(userNickname, file.FullPath, file.Name, mp3FileName, duration, transcription, time.Now(), 0, "")

	fmt.Printf("Transcription completed for file '%s':\n%s\n", file.Name, transcription)
	return nil
}
