package converter

import (
	"fmt"
	"github.com/samber/lo"
	"log"
	"path/filepath"
	"strings"
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

func (c *Converter) ConvertAudioDir(directory string, extension string) error {
	absDir, err := files.GetAbsolutePath(directory)
	if err != nil {
		return err
	}

	// Get all files with specified extension in directory and sort them by old and new
	fileInfos, err := files.GetAllFiles(absDir, extension)
	if err != nil {
		return err
	}

	files := lo.Map(fileInfos, func(f model.FileInfo, i int) string {
		return f.FullPath
	})

	return c.ConvertAudios(files)
}

func (c *Converter) ConvertAudios(files []string) error {
	for _, file := range files {
		transcription, err := c.transcriber.Transcript(file)
		if err != nil {
			return fmt.Errorf("Transcription error: %v\n", err)
		}
		fmt.Printf("%s Transcripted: %s\n", file, transcription)
	}
	return nil
}

// ConvertVideoDir Enter the directory and the number of conversions as parameters
func (c *Converter) ConvertVideoDir(userNickname string, inputDir string, fileExtension string, convertCount int) {
	// Check and create the data/mp3/userNickname subdirectory
	convertedMp3Dir := files.GetUserMp3Dir(userNickname)
	files.CheckAndCreateMP3Directory(convertedMp3Dir)

	// Get all MP4 files in the input directory and sort them by old and new
	fileInfos, err := files.GetAllFiles(inputDir, fileExtension)
	if err != nil {
		log.Fatalln(err)
	}

	filesToProcess := c.filterUnProcessedFiles(fileInfos, convertCount)
	for _, file := range filesToProcess {
		err := c.convertToText(userNickname, file)

		if err != nil {
			log.Fatalln(err)
		}
	}
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
