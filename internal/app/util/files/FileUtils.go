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

// GetAbsolutePath returns the absolute path based on the input path.
// If the input path is relative, it returns the path relative to the current working directory.
// If the input path is absolute, it returns the path as is.
func GetAbsolutePath(path string) (string, error) {
	if filepath.IsAbs(path) {
		return path, nil
	} else {
		wd, err := os.Getwd()
		if err != nil {
			return "", err
		}
		return filepath.Join(wd, path), nil
	}
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

func GetAllFiles(directory string, extension string) ([]model.FileInfo, error) {
	absoluteDir, err := GetAbsolutePath(directory)
	if err != nil {
		return nil, err
	}

	files, err := ioutil.ReadDir(absoluteDir)
	if err != nil {
		log.Fatalf("Failed to read input directory: %v, err: %v\n", directory, err)
	}

	var fileInfos []model.FileInfo
	for _, file := range files {
		if strings.ToLower(filepath.Ext(file.Name())) == "."+strings.ToLower(extension) {
			fileInfos = append(fileInfos, model.FileInfo{
				FullPath: filepath.Join(absoluteDir, file.Name()),
				ModTime:  file.ModTime(),
				Name:     file.Name(),
			})
		}
	}

	sort.Slice(fileInfos, func(i, j int) bool {
		return fileInfos[i].ModTime.Before(fileInfos[j].ModTime)
	})

	return fileInfos, nil
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
