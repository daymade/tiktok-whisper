package downloader

import (
	"testing"
)

func Test_buildEpisodeFilePath(t *testing.T) {
	type args struct {
		dir           string
		podcastName   string
		episodeTitle  string
		fileExtension string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "dir",
			args: args{
				dir:           "",
				podcastName:   "虎言乱语",
				episodeTitle:  "EP1",
				fileExtension: ".m4a",
			},
			want: "虎言乱语/EP1.m4a",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := buildEpisodeFilePath(tt.args.dir, tt.args.podcastName, tt.args.episodeTitle, tt.args.fileExtension); got != tt.want {
				t.Errorf("buildEpisodeFilePath() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_validPath(t *testing.T) {
	type args struct {
		path string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "normal",
			args: args{path: "Vol.13 从职场作家到创办企业，她用IP思维追求人生梦想| 对话职场作家七芊.m4a"},
			want: "Vol.13 从职场作家到创办企业，她用IP思维追求人生梦想- 对话职场作家七芊.m4a",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := validPath(tt.args.path); got != tt.want {
				t.Errorf("validPath() = %v, want %v", got, tt.want)
			}
		})
	}
}