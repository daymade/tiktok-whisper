package downloader

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
)

// TestGetEpisodeInfo_Unit tests getEpisodeInfo with mocked HTTP response
func TestGetEpisodeInfo_Unit(t *testing.T) {
	// Create a mock HTTP server
	mockHTML := `
<!DOCTYPE html>
<html>
<head>
    <meta property="og:audio" content="https://media.xyzcdn.net/test_audio.m4a">
    <meta property="og:title" content="Test Episode Title">
</head>
<body>
    <div class="podcast-title">Test Podcast Name</div>
</body>
</html>
`
	
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(mockHTML))
	}))
	defer server.Close()

	// Test with mock server URL
	audioUrl, episodeTitle, podcastName, err := getEpisodeInfo(server.URL)
	
	if err != nil {
		t.Errorf("getEpisodeInfo() error = %v, want nil", err)
		return
	}
	
	expectedAudioUrl := "https://media.xyzcdn.net/test_audio.m4a"
	if audioUrl != expectedAudioUrl {
		t.Errorf("getEpisodeInfo() audioUrl = %v, want %v", audioUrl, expectedAudioUrl)
	}
	
	expectedTitle := "Test Episode Title"
	if episodeTitle != expectedTitle {
		t.Errorf("getEpisodeInfo() episodeTitle = %v, want %v", episodeTitle, expectedTitle)
	}
	
	expectedPodcastName := "Test Podcast Name"
	if podcastName != expectedPodcastName {
		t.Errorf("getEpisodeInfo() podcastName = %v, want %v", podcastName, expectedPodcastName)
	}
}

// TestGetEpisodeInfo_Unit_MissingContent tests error handling when content is missing
func TestGetEpisodeInfo_Unit_MissingContent(t *testing.T) {
	mockHTML := `
<!DOCTYPE html>
<html>
<head>
    <!-- No og:audio or og:title meta tags -->
</head>
<body>
</body>
</html>
`
	
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(mockHTML))
	}))
	defer server.Close()

	_, _, _, err := getEpisodeInfo(server.URL)
	
	if err == nil {
		t.Errorf("getEpisodeInfo() expected error for missing content, got nil")
	}
}

// TestValidPath_Unit tests path sanitization logic
func TestValidPath_Unit(t *testing.T) {
	tests := []struct {
		name string
		path string
		want string
	}{
		{
			name: "normal_path",
			path: "Normal Episode Title",
			want: "Normal Episode Title",
		},
		{
			name: "path_with_illegal_chars",
			path: "Episode: Title|With<Bad>Chars",
			want: "Episode- Title-With-Bad-Chars",
		},
		{
			name: "path_with_slash",
			path: "Episode/Part1",
			want: "Episode-Part1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := validPath(tt.path); got != tt.want {
				t.Errorf("validPath() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestIsValidXiaoyuzhouEpisodeUrl_Unit tests URL validation logic
func TestIsValidXiaoyuzhouEpisodeUrl_Unit(t *testing.T) {
	tests := []struct {
		name string
		url  string
		want bool
	}{
		{
			name: "valid_http_url",
			url:  "http://www.xiaoyuzhoufm.com/episode/64411602a79cc81470055c96",
			want: true,
		},
		{
			name: "valid_https_url",
			url:  "https://www.xiaoyuzhoufm.com/episode/64411602a79cc81470055c96",
			want: true,
		},
		{
			name: "valid_url_without_www",
			url:  "https://xiaoyuzhoufm.com/episode/64411602a79cc81470055c96",
			want: true,
		},
		{
			name: "invalid_domain",
			url:  "https://example.com/episode/64411602a79cc81470055c96",
			want: false,
		},
		{
			name: "invalid_path",
			url:  "https://www.xiaoyuzhoufm.com/podcast/64411602a79cc81470055c96",
			want: false,
		},
		{
			name: "invalid_episode_id",
			url:  "https://www.xiaoyuzhoufm.com/episode/invalid_id",
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isValidXiaoyuzhouEpisodeUrl(tt.url); got != tt.want {
				t.Errorf("isValidXiaoyuzhouEpisodeUrl() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestGetAudioFileExtension_Unit tests audio file extension detection
func TestGetAudioFileExtension_Unit(t *testing.T) {
	tests := []struct {
		name         string
		audioFileUrl string
		want         string
	}{
		{
			name:         "mp3_extension",
			audioFileUrl: "https://example.com/audio.mp3",
			want:         ".mp3",
		},
		{
			name:         "m4a_extension",
			audioFileUrl: "https://example.com/audio.m4a",
			want:         ".m4a",
		},
		{
			name:         "wav_extension",
			audioFileUrl: "https://example.com/audio.wav",
			want:         ".wav",
		},
		{
			name:         "unsupported_extension",
			audioFileUrl: "https://example.com/audio.txt",
			want:         "",
		},
		{
			name:         "no_extension",
			audioFileUrl: "https://example.com/audio",
			want:         "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getAudioFileExtension(tt.audioFileUrl); got != tt.want {
				t.Errorf("getAudioFileExtension() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestBuildEpisodeFilePath_Unit tests file path building logic
func TestBuildEpisodeFilePath_Unit(t *testing.T) {
	tests := []struct {
		name          string
		dir           string
		podcastName   string
		episodeTitle  string
		fileExtension string
		wantBasename  string
	}{
		{
			name:          "basic_path",
			dir:           "test_dir",
			podcastName:   "Test Podcast",
			episodeTitle:  "Episode 1",
			fileExtension: ".m4a",
			wantBasename:  "Episode 1.m4a",
		},
		{
			name:          "path_with_illegal_chars",
			dir:           "test_dir",
			podcastName:   "Test|Podcast",
			episodeTitle:  "Episode: 1",
			fileExtension: ".mp3",
			wantBasename:  "Episode- 1.mp3",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildEpisodeFilePath(tt.dir, tt.podcastName, tt.episodeTitle, tt.fileExtension)
			
			// Extract the basename to verify the filename is correct
			gotBasename := filepath.Base(got)
			if gotBasename != tt.wantBasename {
				t.Errorf("buildEpisodeFilePath() basename = %v, want %v", gotBasename, tt.wantBasename)
			}
			
			// Check that the path contains the sanitized podcast name
			sanitizedPodcastName := validPath(tt.podcastName)
			if !strings.Contains(got, sanitizedPodcastName) {
				t.Errorf("buildEpisodeFilePath() should contain sanitized podcast name %v in path %v", sanitizedPodcastName, got)
			}
			
			// Check that the path structure is correct (should end with podcast/episode.ext)
			expectedSuffix := filepath.Join(sanitizedPodcastName, tt.wantBasename)
			if !strings.HasSuffix(got, expectedSuffix) {
				t.Errorf("buildEpisodeFilePath() = %v, should end with %v", got, expectedSuffix)
			}
		})
	}
}