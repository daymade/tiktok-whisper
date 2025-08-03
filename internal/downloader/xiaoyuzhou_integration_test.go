//go:build integration
// +build integration

package downloader

import (
	"fmt"
	"log"
	"path/filepath"
	"testing"
	"tiktok-whisper/internal/app/util/files"
)

func Test_download_Integration(t *testing.T) {
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

func Test_getAudioUrlAndTitle_Integration(t *testing.T) {
	type args struct {
		url string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "test1",
			args: args{
				url: "https://www.xiaoyuzhoufm.com/episode/64411602a79cc81470055c96",
			},
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
			// For integration tests, we just verify that we got non-empty results
			// instead of checking specific hardcoded values that may change
			if !tt.wantErr {
				if got == "" {
					t.Errorf("getEpisodeInfo() audioUrl is empty")
				}
				if got1 == "" {
					t.Errorf("getEpisodeInfo() episodeTitle is empty")
				}
				if got2 == "" {
					t.Errorf("getEpisodeInfo() podcastName is empty")
				}
				t.Logf("Integration test results - audioUrl: %s, title: %s, podcast: %s", got, got1, got2)
			}
		})
	}
}

func Test_getJsonScriptFromUrl_Integration(t *testing.T) {
	type args struct {
		url string
	}
	tests := []struct {
		name    string
		args    args
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
			// For integration tests, just verify we got some data
			if !tt.wantErr && data == "" {
				t.Errorf("getJsonScriptFromUrl() returned empty data")
			}
			if !tt.wantErr {
				t.Logf("Integration test got %d bytes of JSON data", len(data))
			}
		})
	}
}

func TestDownloadEpisode_Integration(t *testing.T) {
	// Skip this test in CI environments or if INTEGRATION_TEST_DOWNLOADS is not set
	if testing.Short() {
		t.Skip("Skipping download integration test in short mode")
	}

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
				dir: "/tmp/test_downloads/xiaoyuzhou",
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