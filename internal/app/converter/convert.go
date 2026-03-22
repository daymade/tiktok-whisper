package converter

import (
	"fmt"
	"log"
	"path/filepath"
	"strings"
	"sync"
	"tiktok-whisper/internal/app/api"
	"tiktok-whisper/internal/app/api/provider"
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

	providerType := provider.ResolveProviderType()

	log.Printf("Starting transcription of file %s\n", audioAbsPath)

	transcription, err := c.transcriber.Transcript(audioAbsPath)
	if err != nil {
		log.Printf("Transcription error: %v\n", err)
		fileName := filepath.Base(audioAbsPath)
		c.db.RecordToDB(repository.RecordInput{
			User:               userNickname,
			InputDir:           audioAbsPath,
			FileName:           fileName,
			Mp3FileName:        fileName,
			AudioDuration:      0,
			Transcription:      "",
			LastConversionTime: time.Now(),
			HasError:           1,
			ErrorMessage:       fmt.Sprintf("Transcription error: %v", err),
			ProviderType:       providerType,
		})
		return
	}

	fileName := filepath.Base(audioAbsPath)
	fileNameWithoutExt := strings.TrimSuffix(fileName, filepath.Ext(fileName))
	transcriptionFileName := fileNameWithoutExt + ".txt"
	transcriptionFilepath := filepath.Join(transcriptionDirectory, transcriptionFileName)

	err = files.WriteToFile(transcription, transcriptionFilepath)
	if err != nil {
		log.Printf("Error writing to audioAbsPath: %v\n", err)
		c.db.RecordToDB(repository.RecordInput{
			User:               userNickname,
			InputDir:           audioAbsPath,
			FileName:           fileName,
			Mp3FileName:        fileName,
			AudioDuration:      0,
			Transcription:      transcription,
			LastConversionTime: time.Now(),
			HasError:           1,
			ErrorMessage:       fmt.Sprintf("File write error: %v", err),
			ProviderType:       providerType,
		})
		return
	}

	duration, err := audio.GetAudioDuration(audioAbsPath)
	if err != nil {
		log.Printf("Failed to get audio duration: %v\n", err)
		duration = 0
	}

	c.db.RecordToDB(repository.RecordInput{
		User:               userNickname,
		InputDir:           audioAbsPath,
		FileName:           fileName,
		Mp3FileName:        fileName,
		AudioDuration:      duration,
		Transcription:      transcription,
		LastConversionTime: time.Now(),
		HasError:           0,
		ErrorMessage:       "",
		ProviderType:       providerType,
	})
	
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
	// Check if force re-transcription is requested
	forceMode := false
	if cfg := provider.GetRuntimeConfig(); cfg != nil && cfg.ForceRetranscribe {
		forceMode = true
		log.Printf("Force mode: will re-transcribe already processed files")
	}

	filesToProcess := make([]model.FileInfo, 0, convertCount)

	for _, fileInfo := range fileInfos {
		// Check if the file has been processed
		id, err := c.db.CheckIfFileProcessed(fileInfo.Name)
		if err == nil {
			if forceMode {
				log.Printf("File '%s' (id=%d) already processed, will re-transcribe (--force)\n", fileInfo.Name, id)
				// Delete old record so the new one can be inserted
				c.db.DeleteByID(id)
			} else {
				log.Printf("File '%s' with '%d' has already been processed, skipping...\n", fileInfo.Name, id)
				continue
			}
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

	providerType := provider.ResolveProviderType()

	mp3FileName := strings.TrimSuffix(fileName, ".mp4") + ".mp3"
	mp3FilePath := filepath.Join(files.GetUserMp3Dir(userNickname), mp3FileName)

	err := audio.ConvertToMp3(fileName, fileFullPath, mp3FilePath)
	if err != nil {
		c.db.RecordToDB(repository.RecordInput{
			User:               userNickname,
			InputDir:           fileFullPath,
			FileName:           fileName,
			Mp3FileName:        mp3FileName,
			AudioDuration:      0,
			Transcription:      "",
			LastConversionTime: time.Now(),
			HasError:           1,
			ErrorMessage:       fmt.Sprintf("FFmpeg error: %v", err),
			ProviderType:       providerType,
		})
		return fmt.Errorf("FFmpeg error: %v", err)
	}

	duration, err := audio.GetAudioDuration(mp3FilePath)
	if err != nil {
		c.db.RecordToDB(repository.RecordInput{
			User:               userNickname,
			InputDir:           fileFullPath,
			FileName:           fileName,
			Mp3FileName:        mp3FileName,
			AudioDuration:      0,
			Transcription:      "",
			LastConversionTime: time.Now(),
			HasError:           1,
			ErrorMessage:       fmt.Sprintf("Failed to get audio duration: %v", err),
			ProviderType:       providerType,
		})
		return fmt.Errorf("failed to get audio duration: %v", err)
	}

	transcription, err := c.transcriber.Transcript(mp3FilePath)
	if err != nil {
		log.Printf("transcripting failed for %v, err: %v", fileName, err)

		c.db.RecordToDB(repository.RecordInput{
			User:               userNickname,
			InputDir:           fileFullPath,
			FileName:           fileName,
			Mp3FileName:        mp3FileName,
			AudioDuration:      duration,
			Transcription:      "",
			LastConversionTime: time.Now(),
			HasError:           1,
			ErrorMessage:       fmt.Sprintf("Transcription error: %v", err),
			ProviderType:       providerType,
		})

		return fmt.Errorf("transcription error: %v", err)
	}

	c.db.RecordToDB(repository.RecordInput{
		User:               userNickname,
		InputDir:           fileFullPath,
		FileName:           fileName,
		Mp3FileName:        mp3FileName,
		AudioDuration:      duration,
		Transcription:      transcription,
		LastConversionTime: time.Now(),
		HasError:           0,
		ErrorMessage:       "",
		ProviderType:       providerType,
	})

	log.Println("transcription completed for file: ", fileName)
	fmt.Println(transcription)
	return nil
}
