package converter

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"sync"
	"tiktok-whisper/internal/app/util/files"
	"time"

	"github.com/vbauerster/mpb/v8"
	"github.com/vbauerster/mpb/v8/decor"
)

type ProgressConfig struct {
	Enabled bool
	Writer  io.Writer
}

type ProgressManager struct {
	container *mpb.Progress
	enabled   bool
	mu        sync.Mutex
}

type ProgressBar struct {
	bar     *mpb.Bar
	enabled bool
}

func NewProgressManager(config ProgressConfig) *ProgressManager {
	if !config.Enabled {
		return &ProgressManager{enabled: false}
	}

	writer := config.Writer
	if writer == nil {
		writer = os.Stderr
	}

	container := mpb.New(
		mpb.WithOutput(writer),
		mpb.WithRefreshRate(120*time.Millisecond),
		mpb.WithWaitGroup(&sync.WaitGroup{}),
	)

	return &ProgressManager{
		container: container,
		enabled:   true,
	}
}

func (pm *ProgressManager) CreateBar(total int, description string) *ProgressBar {
	if !pm.enabled || pm.container == nil {
		return &ProgressBar{enabled: false}
	}

	pm.mu.Lock()
	defer pm.mu.Unlock()

	bar := pm.container.AddBar(int64(total),
		mpb.PrependDecorators(
			decor.Name(description+" ", decor.WC{W: len(description) + 1, C: decor.DindentRight}),
			decor.CountersNoUnit("(%d/%d)", decor.WCSyncWidth),
		),
		mpb.AppendDecorators(
			decor.NewPercentage("%.1f", decor.WCSyncSpace),
			decor.OnComplete(
				decor.EwmaETA(decor.ET_STYLE_GO, 30, decor.WCSyncWidth), " ✓ ",
			),
			decor.OnComplete(
				decor.EwmaSpeed(0, "%.1f files/s", 30, decor.WCSyncSpace), "",
			),
		),
	)

	return &ProgressBar{
		bar:     bar,
		enabled: true,
	}
}

func (pb *ProgressBar) Increment() {
	if pb.enabled && pb.bar != nil {
		pb.bar.Increment()
	}
}

func (pb *ProgressBar) IncrementWithMessage(message string) {
	if pb.enabled && pb.bar != nil {
		pb.bar.Increment()
		if message != "" {
			pb.bar.SetCurrent(pb.bar.Current())
		}
	}
}

func (pb *ProgressBar) SetTotal(total int64) {
	if pb.enabled && pb.bar != nil {
		pb.bar.SetTotal(total, false)
	}
}

func (pb *ProgressBar) Complete() {
	if pb.enabled && pb.bar != nil {
		pb.bar.SetTotal(pb.bar.Current(), true)
	}
}

func (pm *ProgressManager) Wait() {
	if pm.enabled && pm.container != nil {
		pm.container.Wait()
	}
}

func (pm *ProgressManager) Shutdown() {
	if pm.enabled && pm.container != nil {
		pm.container.Shutdown()
	}
}

func IsTTY(writer io.Writer) bool {
	if writer == nil {
		return false
	}
	
	if file, ok := writer.(*os.File); ok {
		stat, err := file.Stat()
		if err != nil {
			return false
		}
		return (stat.Mode() & os.ModeCharDevice) != 0
	}
	return false
}

func ShouldShowProgress(forced bool) bool {
	if forced {
		return true
	}
	
	return IsTTY(os.Stderr) || IsTTY(os.Stdout)
}

type ProgressAwareConverter struct {
	*Converter
	progressManager *ProgressManager
}

func NewProgressAwareConverter(converter *Converter, config ProgressConfig) *ProgressAwareConverter {
	return &ProgressAwareConverter{
		Converter:       converter,
		progressManager: NewProgressManager(config),
	}
}

func (pac *ProgressAwareConverter) Close() error {
	if pac.progressManager != nil {
		pac.progressManager.Shutdown()
	}
	return pac.Converter.Close()
}

func (pac *ProgressAwareConverter) createProgressBar(total int, description string) *ProgressBar {
	if pac.progressManager == nil {
		return &ProgressBar{enabled: false}
	}
	return pac.progressManager.CreateBar(total, description)
}

func (pac *ProgressAwareConverter) waitForProgress() {
	if pac.progressManager != nil {
		pac.progressManager.Wait()
	}
}

func FormatProgressDescription(action string, userNickname string) string {
	if userNickname != "" {
		return fmt.Sprintf("%s (%s)", action, userNickname)
	}
	return action
}

func (pac *ProgressAwareConverter) ConvertVideosWithProgress(fileFullpaths []string, userNickname string, convertCount int, parallel int) error {
	if len(fileFullpaths) == 0 {
		return nil
	}

	description := FormatProgressDescription("Converting videos", userNickname)
	progressBar := pac.createProgressBar(len(fileFullpaths), description)
	defer pac.waitForProgress()

	convertedMp3Dir := files.GetUserMp3Dir(userNickname)
	files.CheckAndCreateMP3Directory(convertedMp3Dir)

	var wg sync.WaitGroup
	sem := make(chan bool, parallel)

	for _, fileAbsPath := range fileFullpaths {
		wg.Add(1)
		go func(fileAbsPath string) {
			defer wg.Done()
			defer progressBar.Increment()

			fileName := filepath.Base(fileAbsPath)

			sem <- true
			err := pac.convertToText(userNickname, fileName, fileAbsPath)
			<-sem

			if err != nil {
				log.Printf("Error converting file %s: %v\n", fileName, err)
			} else {
				log.Printf("Successfully converted file %s\n", fileName)
			}
		}(fileAbsPath)
	}
	wg.Wait()
	return nil
}

func (pac *ProgressAwareConverter) ConvertAudiosWithProgress(audioFiles []string, outputDirectory string, userNickname string, parallel int) error {
	if len(audioFiles) == 0 {
		return nil
	}

	progressBar := pac.createProgressBar(len(audioFiles), "Converting audios")
	defer pac.waitForProgress()

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
			defer progressBar.Increment()
			
			sem <- true
			pac.processFile(file, transcriptionDirectory, userNickname)
			<-sem
		}(file)
	}
	wg.Wait()
	return nil
}

func (pac *ProgressAwareConverter) ConvertVideoDirWithProgress(userNickname string, inputDir string, fileExtension string, convertCount int, parallel int) error {
	fileInfos, err := files.GetAllFiles(inputDir, fileExtension)
	if err != nil {
		return err
	}

	filesToProcess := pac.filterUnProcessedFiles(fileInfos, convertCount)
	if len(filesToProcess) == 0 {
		return nil
	}

	fileFullpaths := make([]string, len(filesToProcess))
	for i, f := range filesToProcess {
		fileFullpaths[i] = f.FullPath
	}

	return pac.ConvertVideosWithProgress(fileFullpaths, userNickname, convertCount, parallel)
}

func (pac *ProgressAwareConverter) ConvertAudioDirWithProgress(userNickname string, directory string, extension string, outputDirectory string, convertCount int, parallel int) error {
	absDir, err := files.GetAbsolutePath(directory)
	if err != nil {
		log.Printf("Error getting absolute path of directory %s: %v\n", directory, err)
		return err
	}

	log.Printf("Starting to convert audio files in directory %s\n", absDir)

	fileInfos, err := files.GetAllFiles(absDir, extension)
	if err != nil {
		log.Printf("Error getting all files in directory %s: %v\n", absDir, err)
		return err
	}

	filesToProcess := pac.filterUnProcessedFiles(fileInfos, convertCount)

	audioFiles := make([]string, len(filesToProcess))
	for i, f := range filesToProcess {
		audioFiles[i] = f.FullPath
	}

	log.Printf("Found %d files to convert\n", len(audioFiles))

	err = pac.ConvertAudiosWithProgress(audioFiles, outputDirectory, userNickname, parallel)
	if err != nil {
		log.Printf("Error converting audio files: %v\n", err)
		return err
	}

	log.Printf("Successfully converted all audio files\n")
	return nil
}