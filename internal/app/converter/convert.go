package converter

import (
	"fmt"
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

	"github.com/samber/lo"
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
// It takes the user nickname, directory, the file extension of the audios, the output directory,
// and the number of parallel conversions as parameters.
func (c *Converter) ConvertAudioDir(userNickname string,
	directory string,
	extension string,
	outputDirectory string,
	convertCount int,
	parallel int) error {
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

	filesToProcess := c.filterUnProcessedFiles(fileInfos, convertCount)

	files := lo.Map(filesToProcess, func(f model.FileInfo, i int) string {
		return f.FullPath
	})

	log.Printf("Found %d files to convert\n", len(files))

	err = c.ConvertAudios(files, outputDirectory, userNickname, parallel)
	if err != nil {
		log.Printf("Error converting audio files: %v\n", err)
		return err
	}

	log.Printf("Successfully converted all audio files\n")

	return nil
}

func (c *Converter) ConvertAudios(audioFiles []string, outputDirectory string, userNickname string, parallel int) error {
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
			c.processFile(file, transcriptionDirectory, userNickname)
			<-sem
		}(file)
	}
	wg.Wait()
	return nil
}

func (c *Converter) processFile(audioAbsPath string, transcriptionDirectory string, userNickname string) {
	log.Printf("Start to process %s\n", audioAbsPath)

	// Start transcription
	log.Printf("Starting transcription of file %s\n", audioAbsPath)
	
	transcription, err := c.transcriber.Transcript(audioAbsPath)
	if err != nil {
		log.Printf("Transcription error: %v\n", err)
		// Record error to database
		fileName := filepath.Base(audioAbsPath)
		c.db.RecordToDB(userNickname, audioAbsPath, fileName, fileName, 0, "", 
			time.Now(), 1, fmt.Sprintf("Transcription error: %v", err))
		return
	}

	fileName := filepath.Base(audioAbsPath)
	fileNameWithoutExt := strings.TrimSuffix(fileName, filepath.Ext(fileName))
	transcriptionFileName := fileNameWithoutExt + ".txt"
	transcriptionFilepath := filepath.Join(transcriptionDirectory, transcriptionFileName)

	err = files.WriteToFile(transcription, transcriptionFilepath)
	if err != nil {
		log.Printf("Error writing to audioAbsPath: %v\n", err)
		// Record error to database
		c.db.RecordToDB(userNickname, audioAbsPath, fileName, fileName, 0, transcription,
			time.Now(), 1, fmt.Sprintf("File write error: %v", err))
		return
	}
	
	// Get audio duration
	duration, err := audio.GetAudioDuration(audioAbsPath)
	if err != nil {
		log.Printf("Failed to get audio duration: %v\n", err)
		duration = 0 // Use 0 if we can't get duration
	}
	
	// Save successful transcription to database
	c.db.RecordToDB(userNickname, audioAbsPath, fileName, fileName, duration, transcription, 
		time.Now(), 0, "")
	
	log.Printf("Transcription saved to: %s\n", transcriptionFilepath)
}

// ConvertVideoDir converts videos in a directory to text in parallel.
// It takes the user's nickname, the input directory, the file extension of the videos,
// the maximum number of videos to convert, and the number of parallel conversions as parameters.
func (c *Converter) ConvertVideoDir(userNickname string, inputDir string, fileExtension string, convertCount int, parallel int) error {
	// Get all MP4 files in the input directory and sort them by old and new
	fileInfos, err := files.GetAllFiles(inputDir, fileExtension)
	if err != nil {
		log.Fatalln(err)
	}

	filesToProcess := c.filterUnProcessedFiles(fileInfos, convertCount)
	if len(filesToProcess) == 0 {
		return nil
	}

	fileFullpaths := lo.Map(filesToProcess, func(f model.FileInfo, i int) string {
		return f.FullPath
	})

	err = c.ConvertVideos(fileFullpaths, userNickname, convertCount, parallel)
	if err != nil {
		log.Printf("Error converting video files: %v\n", err)
		return err
	}

	log.Printf("Successfully converted all video files\n")

	return nil
}

func (c *Converter) ConvertVideos(fileFullpaths []string, userNickname string, convertCount int, parallel int) error {
	// Check and create the data/mp3/userNickname subdirectory
	convertedMp3Dir := files.GetUserMp3Dir(userNickname)
	files.CheckAndCreateMP3Directory(convertedMp3Dir)

	var wg sync.WaitGroup
	sem := make(chan bool, parallel)

	for _, fileAbsPath := range fileFullpaths {
		wg.Add(1)
		go func(fileAbsPath string) {
			defer wg.Done()

			fileName := filepath.Base(fileAbsPath)

			sem <- true
			err := c.convertToText(userNickname, fileName, fileAbsPath)
			<-sem

			if err != nil {
				log.Fatalf("Error converting file %s: %v\n", fileName, err)
			} else {
				log.Printf("Successfully converted file %s\n", fileName)
			}
		}(fileAbsPath)
	}
	wg.Wait()
	return nil
}

func (c *Converter) filterUnProcessedFiles(fileInfos []model.FileInfo, convertCount int) []model.FileInfo {
	filesToProcess := make([]model.FileInfo, 0, convertCount)

	for _, fileInfo := range fileInfos {
		// Check if the file has been processed
		id, err := c.db.CheckIfFileProcessed(fileInfo.Name)
		if err == nil {
			log.Printf("File '%s' with '%d' has already been processed, skipping...\n", fileInfo.Name, id)
			continue
		}

		filesToProcess = append(filesToProcess, fileInfo)
		if len(filesToProcess) >= convertCount {
			break
		}
	}
	return filesToProcess
}

func (c *Converter) convertToText(userNickname string, fileName string, fileFullPath string) error {
	log.Printf("Processing file '%s'\n", fileName)

	// Convert MP4 to MP3 using FFmpeg
	mp3FileName := strings.TrimSuffix(fileName, ".mp4") + ".mp3"
	mp3FilePath := filepath.Join(files.GetUserMp3Dir(userNickname), mp3FileName)

	// Check if the MP3 file already exists
	err := audio.ConvertToMp3(fileName, fileFullPath, mp3FilePath)
	if err != nil {
		c.db.RecordToDB(userNickname, fileFullPath, fileName, mp3FileName, 0, "",
			time.Now(), 1, fmt.Sprintf("FFmpeg error: %v", err))
		return fmt.Errorf("FFmpeg error: %v", err)
	}

	// Get audio duration
	duration, err := audio.GetAudioDuration(mp3FilePath)
	if err != nil {
		c.db.RecordToDB(userNickname, fileFullPath, fileName, mp3FileName, 0, "",
			time.Now(), 1, fmt.Sprintf("Failed to get audio duration: %v", err))
		return fmt.Errorf("failed to get audio duration: %v", err)
	}

	// Call Whisper with a new MP3 file path
	transcription, err := c.transcriber.Transcript(mp3FilePath)
	if err != nil {
		log.Printf("transcripting failed for %v, err: %v", fileName, err)

		c.db.RecordToDB(userNickname, fileFullPath, fileName, mp3FileName, duration, "",
			time.Now(), 1, fmt.Sprintf("Transcription error: %v", err))

		return fmt.Errorf("transcription error: %v", err)
	}

	// Save conversion results to database
	c.db.RecordToDB(userNickname, fileFullPath, fileName, mp3FileName, duration, transcription, time.Now(), 0, "")

	log.Println("transcription completed for file: ", fileName)
	fmt.Println(transcription)
	return nil
}
