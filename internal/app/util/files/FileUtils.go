package files

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"tiktok-whisper/internal/app/model"
)

func GetProjectRoot() (string, error) {
	_, filename, _, _ := runtime.Caller(0)
	return findGoModRoot(filename)
}

func GetUserMp3Dir(userNickname string) string {
	root, err := GetProjectRoot()
	if err != nil {
		log.Fatalf("GetUserMp3Dir failed: %v\n", err)
	}
	return filepath.Join(root, "data/mp3", userNickname)
}

func CheckAndCreateMP3Directory(mp3Dir string) {
	if _, err := os.Stat(mp3Dir); os.IsNotExist(err) {
		fmt.Printf("Creating MP3 directory: %s\n", mp3Dir)
		if err := os.MkdirAll(mp3Dir, os.ModePerm); err != nil {
			log.Fatalf("Failed to create MP3 directory: %v\n", err)
		}
	}
}

func GetAllMP4Files(inputDir string) []model.FileInfo {
	files, err := ioutil.ReadDir(inputDir)
	if err != nil {
		log.Fatalf("Failed to read input directory: %v\n", err)
	}

	var fileInfos []model.FileInfo
	for _, file := range files {
		if strings.ToLower(filepath.Ext(file.Name())) == ".mp4" {
			fileInfos = append(fileInfos, model.FileInfo{
				FullPath: filepath.Join(inputDir, file.Name()),
				ModTime:  file.ModTime(),
				Name:     file.Name(),
			})
		}
	}

	sort.Slice(fileInfos, func(i, j int) bool {
		return fileInfos[i].ModTime.Before(fileInfos[j].ModTime)
	})

	return fileInfos
}

// ReadOutputFile reads the specified output file and returns its text content.
func ReadOutputFile(filePath string) (string, error) {
	content, err := ioutil.ReadFile(filePath)
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(content)), nil
}

func findGoModRoot(path string) (string, error) {
	for {
		if _, err := os.Stat(filepath.Join(path, "go.mod")); err == nil {
			return path, nil
		}
		newPath := filepath.Dir(path)
		if newPath == path {
			return "", fmt.Errorf("go.mod not found")
		}
		path = newPath
	}
}
