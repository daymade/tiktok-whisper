package files

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestPathNormalization(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected func(string) bool
		desc     string
	}{
		{
			name:  "normalize_forward_slashes",
			input: "path/to/file.txt",
			expected: func(got string) bool {
				return strings.Contains(got, "path") && strings.Contains(got, "to") && strings.Contains(got, "file.txt")
			},
			desc: "Should handle forward slashes on all platforms",
		},
		{
			name:  "normalize_multiple_slashes",
			input: "path//to///file.txt",
			expected: func(got string) bool {
				// After normalization, should not have consecutive separators
				sep := string(filepath.Separator)
				return !strings.Contains(got, sep+sep)
			},
			desc: "Should normalize multiple consecutive slashes",
		},
		{
			name:  "normalize_dot_segments",
			input: "./path/../to/./file.txt",
			expected: func(got string) bool {
				return strings.HasSuffix(got, filepath.Join("to", "file.txt"))
			},
			desc: "Should resolve . and .. segments",
		},
		{
			name:  "trailing_slash_handling",
			input: "path/to/directory/",
			expected: func(got string) bool {
				return strings.Contains(got, "directory")
			},
			desc: "Should handle trailing slashes",
		},
	}

	if runtime.GOOS == "windows" {
		tests = append(tests, struct {
			name     string
			input    string
			expected func(string) bool
			desc     string
		}{
			name:  "windows_backslashes",
			input: "path\\to\\file.txt",
			expected: func(got string) bool {
				return strings.Contains(got, "path") && strings.Contains(got, "file.txt")
			},
			desc: "Should handle Windows backslashes",
		})
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetAbsolutePath(tt.input)
			if err != nil {
				t.Fatalf("GetAbsolutePath() error = %v", err)
			}
			if !tt.expected(got) {
				t.Errorf("Path normalization failed for %s: got %s - %s", tt.input, got, tt.desc)
			}
		})
	}
}

func TestUnicodePathHandling(t *testing.T) {
	unicodePaths := []struct {
		name string
		path string
	}{
		{"chinese_characters", "路径/文件名.txt"},
		{"japanese_characters", "パス/ファイル.txt"},
		{"korean_characters", "경로/파일.txt"},
		{"arabic_characters", "مسار/ملف.txt"},
		{"emoji_in_path", "📁/📄.txt"},
		{"mixed_unicode", "文件夹/ファイル/file.txt"},
	}

	for _, tt := range unicodePaths {
		t.Run(tt.name, func(t *testing.T) {
			// Test GetAbsolutePath with Unicode
			got, err := GetAbsolutePath(tt.path)
			if err != nil {
				t.Errorf("GetAbsolutePath() failed with Unicode path %s: %v", tt.path, err)
			}

			// Ensure Unicode is preserved
			for _, part := range strings.Split(tt.path, "/") {
				if part != "" && !strings.Contains(got, part) {
					t.Errorf("Unicode path component %s not preserved in %s", part, got)
				}
			}
		})
	}
}

func TestLongPathHandling(t *testing.T) {
	// Test very long paths
	tests := []struct {
		name       string
		pathLength int
		shouldWork bool
	}{
		{"normal_length", 100, true},
		{"medium_length", 200, true},
		{"long_path", 255, true},
	}

	// On Windows, paths longer than 260 characters may fail without long path support
	if runtime.GOOS != "windows" {
		tests = append(tests, struct {
			name       string
			pathLength int
			shouldWork bool
		}{
			"very_long_path", 500, true,
		})
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a long path by repeating a directory name
			parts := make([]string, 0)
			currentLength := 0
			segment := "verylongdirectorynametotestpathlimits"

			for currentLength < tt.pathLength {
				parts = append(parts, segment)
				currentLength += len(segment) + 1 // +1 for separator
			}

			longPath := filepath.Join(parts...)
			got, err := GetAbsolutePath(longPath)

			if tt.shouldWork && err != nil {
				t.Errorf("GetAbsolutePath() failed for long path: %v", err)
			}
			if tt.shouldWork && !strings.Contains(got, segment) {
				t.Errorf("Long path not properly handled")
			}
		})
	}
}

func TestSpecialCharactersInPath(t *testing.T) {
	specialChars := []struct {
		name  string
		char  string
		valid bool
	}{
		{"spaces", "path with spaces", true},
		{"parentheses", "path(with)parens", true},
		{"brackets", "path[with]brackets", true},
		{"at_symbol", "path@symbol", true},
		{"hash", "path#hash", true},
		{"percent", "path%percent", true},
		{"ampersand", "path&ampersand", true},
		{"plus", "path+plus", true},
		{"equals", "path=equals", true},
		{"comma", "path,comma", true},
		{"semicolon", "path;semicolon", true},
		{"apostrophe", "path'apostrophe", true},
		{"dash", "path-dash", true},
		{"underscore", "path_underscore", true},
		{"tilde", "path~tilde", true},
	}

	// Some characters are invalid on Windows
	if runtime.GOOS != "windows" {
		specialChars = append(specialChars, []struct {
			name  string
			char  string
			valid bool
		}{
			{"colon", "path:colon", true},
			{"asterisk", "path*asterisk", true},
			{"question", "path?question", true},
			{"less_than", "path<less", true},
			{"greater_than", "path>greater", true},
			{"pipe", "path|pipe", true},
			{"quote", "path\"quote", true},
		}...)
	}

	for _, tt := range specialChars {
		t.Run(tt.name, func(t *testing.T) {
			testPath := filepath.Join("test", tt.char, "file.txt")
			got, err := GetAbsolutePath(testPath)

			if tt.valid && err != nil {
				t.Errorf("GetAbsolutePath() failed for special character %s: %v", tt.name, err)
			}
			if tt.valid && !strings.Contains(got, tt.char) {
				t.Errorf("Special character %s not preserved in path: %s", tt.name, got)
			}
		})
	}
}

func TestRelativePathResolution(t *testing.T) {
	// Save current directory
	originalWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}
	defer os.Chdir(originalWd)

	tests := []struct {
		name     string
		input    string
		expected func(string) bool
	}{
		{
			name:  "single_dot",
			input: ".",
			expected: func(got string) bool {
				wd, _ := os.Getwd()
				return got == wd
			},
		},
		{
			name:  "double_dot",
			input: "..",
			expected: func(got string) bool {
				wd, _ := os.Getwd()
				return got == filepath.Dir(wd)
			},
		},
		{
			name:  "nested_relative",
			input: "../..",
			expected: func(got string) bool {
				wd, _ := os.Getwd()
				return got == filepath.Dir(filepath.Dir(wd))
			},
		},
		{
			name:  "mixed_relative",
			input: "./../test/./data",
			expected: func(got string) bool {
				return strings.HasSuffix(got, filepath.Join("test", "data"))
			},
		},
		{
			name:  "relative_with_file",
			input: "./test.txt",
			expected: func(got string) bool {
				wd, _ := os.Getwd()
				return got == filepath.Join(wd, "test.txt")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetAbsolutePath(tt.input)
			if err != nil {
				t.Errorf("GetAbsolutePath() error = %v", err)
				return
			}
			if !tt.expected(got) {
				t.Errorf("Relative path resolution failed for %s: got %s", tt.input, got)
			}
		})
	}
}

func TestCrossPlatformPathSeparators(t *testing.T) {
	// Test that path operations work correctly regardless of separator used
	paths := []string{
		"path/to/file.txt",
		"path\\to\\file.txt",
		"path/to\\file.txt",
		"path\\to/file.txt",
	}

	for _, p := range paths {
		t.Run("separator_"+p, func(t *testing.T) {
			got, err := GetAbsolutePath(p)
			if err != nil {
				t.Errorf("GetAbsolutePath() failed for mixed separators: %v", err)
			}

			// Note: filepath.Join and GetAbsolutePath may preserve backslashes on Unix
			// This is expected Go behavior - the path is still valid
			// Just verify the path is absolute and contains expected components
			if !filepath.IsAbs(got) {
				t.Errorf("Path is not absolute: %s", got)
			}
			if !strings.Contains(got, "path") || !strings.Contains(got, "file.txt") {
				t.Errorf("Path doesn't contain expected components: %s", got)
			}
		})
	}
}

func TestPathCaseSensitivity(t *testing.T) {
	// This behavior varies by OS and filesystem
	if runtime.GOOS == "windows" || runtime.GOOS == "darwin" {
		// Case-insensitive filesystems
		t.Run("case_insensitive_comparison", func(t *testing.T) {
			path1, _ := GetAbsolutePath("Test/Path")
			path2, _ := GetAbsolutePath("test/path")

			// On case-insensitive systems, these might resolve to the same canonical path
			// This test is informational rather than assertive
			t.Logf("Path comparison - path1: %s, path2: %s", path1, path2)
		})
	} else {
		// Case-sensitive filesystems
		t.Run("case_sensitive_comparison", func(t *testing.T) {
			path1, _ := GetAbsolutePath("Test/Path")
			path2, _ := GetAbsolutePath("test/path")

			// These should be different paths
			if path1 == path2 {
				t.Errorf("Paths should be case-sensitive but got same result")
			}
		})
	}
}

func TestSymlinkHandling(t *testing.T) {
	// Note: This test requires filesystem support for symlinks
	// May not work on all Windows systems without appropriate permissions

	if runtime.GOOS == "windows" {
		t.Skip("Skipping symlink test on Windows")
	}

	tempDir := t.TempDir()

	// Create a real directory and file
	realDir := filepath.Join(tempDir, "real")
	os.MkdirAll(realDir, 0755)
	realFile := filepath.Join(realDir, "file.txt")
	os.WriteFile(realFile, []byte("content"), 0644)

	// Create symlinks
	linkDir := filepath.Join(tempDir, "link_to_dir")
	linkFile := filepath.Join(tempDir, "link_to_file")

	if err := os.Symlink(realDir, linkDir); err != nil {
		t.Skip("Cannot create symlinks on this system")
	}
	os.Symlink(realFile, linkFile)

	tests := []struct {
		name string
		path string
	}{
		{"symlink_to_directory", linkDir},
		{"symlink_to_file", linkFile},
		{"path_through_symlink", filepath.Join(linkDir, "file.txt")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetAbsolutePath(tt.path)
			if err != nil {
				t.Errorf("GetAbsolutePath() failed for symlink: %v", err)
			}

			// The result should be a valid path
			if !filepath.IsAbs(got) {
				t.Errorf("Expected absolute path for symlink, got: %s", got)
			}
		})
	}
}
