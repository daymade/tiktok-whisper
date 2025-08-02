package model

import "time"

type FileInfo struct {
	FullPath string
	ModTime  time.Time
	Name     string
}
