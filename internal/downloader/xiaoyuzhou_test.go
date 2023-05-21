package downloader

import (
	"fmt"
	"log"
	"path/filepath"
	"testing"
	"tiktok-whisper/internal/app/util/files"
)

func Test_download(t *testing.T) {
	projectRoot, err := files.GetProjectRoot()
	if err != nil {
		log.Fatalf("Failed to get project root: %v\n", err)
	}
	testOutputDir := filepath.Join(projectRoot, "test", "data", "xiaoyuzhou")

	tests := []struct {
		name string
		pid  string
	}{
		{
			name: "123",
			pid:  "61a9f093ca6141933d1a1c63",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			DownloadPodcast(tt.pid, testOutputDir)
		})
	}
}

func Test_getAudioUrlAndTitle(t *testing.T) {
	type args struct {
		url string
	}
	tests := []struct {
		name    string
		args    args
		want    string
		want1   string
		want2   string
		wantErr bool
	}{
		{
			name: "test1",
			args: args{
				url: "https://www.xiaoyuzhoufm.com/episode/64411602a79cc81470055c96",
			},
			want:    "https://media.xyzcdn.net/nuIZxCLBCRPQfnvMwy_A47tglnOa.mp3",
			want1:   "35｜本土品牌攻打高端市场：闻献如何成为闻献",
			want2:   "DTC Lab",
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1, got2, err := getEpisodeInfo(tt.args.url)
			if (err != nil) != tt.wantErr {
				t.Errorf("getEpisodeInfo() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("getEpisodeInfo() got = %v, want %v", got, tt.want)
			}
			if got1 != tt.want1 {
				t.Errorf("getEpisodeInfo() got1 = %v, want %v", got1, tt.want1)
			}
			if got2 != tt.want2 {
				t.Errorf("getEpisodeInfo() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}

func Test_getJsonScriptFromUrl(t *testing.T) {
	type args struct {
		url string
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{
			name: "profile1",
			args: args{
				url: buildPodcastUrl("61a9f093ca6141933d1a1c63"),
			},
			wantErr: false,
		},
		{
			name: "episode",
			args: args{
				url: buildEpisodeUrl("64411602a79cc81470055c96"),
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := getJsonScriptFromUrl(tt.args.url)
			if (err != nil) != tt.wantErr {
				t.Errorf("getJsonScriptFromUrl() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			fmt.Println(data)
		})
	}
}

func TestDownloadEpisode(t *testing.T) {
	type args struct {
		url string
		dir string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "test1",
			args: args{
				url: "https://www.xiaoyuzhoufm.com/episode/6398c6ae3a2b7eba5ceb462f",
				dir: "/Users/tiansheng/workspace/go/tiktok-whisper/data/xiaoyuzhou/运营狗工作日记",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := DownloadEpisode(tt.args.url, tt.args.dir); (err != nil) != tt.wantErr {
				t.Errorf("DownloadEpisode() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

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
