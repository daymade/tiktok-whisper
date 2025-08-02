package files

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestFileExtensionValidation(t *testing.T) {
	tests := []struct {
		name        string
		filename    string
		extension   string
		shouldMatch bool
	}{
		// Audio formats
		{"mp3_lowercase", "audio.mp3", "mp3", true},
		{"mp3_uppercase", "audio.MP3", "mp3", true},
		{"mp3_mixed_case", "audio.Mp3", "mp3", true},
		{"wav_file", "audio.wav", "wav", true},
		{"m4a_file", "audio.m4a", "m4a", true},
		{"flac_file", "audio.flac", "flac", true},
		{"ogg_file", "audio.ogg", "ogg", true},
		{"aac_file", "audio.aac", "aac", true},
		{"wma_file", "audio.wma", "wma", true},

		// Edge cases
		{"no_extension", "audiofile", "mp3", false},
		{"multiple_dots", "audio.test.mp3", "mp3", true},
		{"dot_in_name", "audio.v2.final.mp3", "mp3", true},
		{"wrong_extension", "audio.txt", "mp3", false},
		{"empty_extension", "audio.", "", true},
		{"hidden_file", ".audio.mp3", "mp3", true},
		{"extension_only", ".mp3", "mp3", true},

		// Special cases
		{"space_in_extension", "audio.mp3 ", "mp3", false},
		{"tab_in_extension", "audio.mp3\t", "mp3", false},
		{"newline_in_extension", "audio.mp3\n", "mp3", false},
		{"similar_extension", "audio.mp3x", "mp3", false},
		{"partial_match", "audio.amp3", "mp3", false},
	}

	tempDir := t.TempDir()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test file
			filePath := filepath.Join(tempDir, tt.filename)
			ioutil.WriteFile(filePath, []byte("test"), 0644)

			// Test with GetAllFiles
			files, err := GetAllFiles(tempDir, tt.extension)
			if err != nil {
				t.Errorf("GetAllFiles() error = %v", err)
				return
			}

			found := false
			for _, f := range files {
				if f.Name == tt.filename {
					found = true
					break
				}
			}

			if found != tt.shouldMatch {
				t.Errorf("Extension matching failed for %s with extension %s: got %v, want %v",
					tt.filename, tt.extension, found, tt.shouldMatch)
			}

			// Cleanup
			os.Remove(filePath)
		})
	}
}

func TestFileValidationWithSize(t *testing.T) {
	tempDir := t.TempDir()

	tests := []struct {
		name        string
		filename    string
		size        int64
		expectValid bool
	}{
		{"empty_file", "empty.mp3", 0, true},
		{"small_file", "small.mp3", 1, true},
		{"normal_file", "normal.mp3", 1024 * 1024, true},     // 1MB
		{"large_file", "large.mp3", 100 * 1024 * 1024, true}, // 100MB
		{"huge_file", "huge.mp3", 1024 * 1024 * 1024, true},  // 1GB
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filePath := filepath.Join(tempDir, tt.filename)

			// Create file with specific size
			file, err := os.Create(filePath)
			if err != nil {
				t.Fatalf("Failed to create file: %v", err)
			}

			if tt.size > 0 {
				// For large files, just set the size without writing all data
				file.Truncate(tt.size)
			}
			file.Close()

			// Verify file size
			info, err := os.Stat(filePath)
			if err != nil {
				t.Errorf("Failed to stat file: %v", err)
			}

			if info.Size() != tt.size {
				t.Errorf("File size mismatch: got %d, want %d", info.Size(), tt.size)
			}

			// Test that GetAllFiles can handle files of various sizes
			files, err := GetAllFiles(tempDir, "mp3")
			if err != nil {
				t.Errorf("GetAllFiles() failed with %s: %v", tt.name, err)
			}

			found := false
			for _, f := range files {
				if f.Name == tt.filename {
					found = true
					break
				}
			}

			if found != tt.expectValid {
				t.Errorf("File validation failed for %s", tt.name)
			}

			// Cleanup
			os.Remove(filePath)
		})
	}
}

func TestFilePermissionValidation(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping Unix permission tests on Windows")
	}

	tempDir := t.TempDir()

	tests := []struct {
		name        string
		filename    string
		permissions os.FileMode
		canRead     bool
	}{
		{"readable_file", "readable.mp3", 0644, true},
		{"write_only_file", "writeonly.mp3", 0200, false},
		{"execute_only_file", "execonly.mp3", 0111, false},
		{"no_permissions", "noperms.mp3", 0000, false},
		{"all_permissions", "allperms.mp3", 0777, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filePath := filepath.Join(tempDir, tt.filename)

			// Create file with specific permissions
			ioutil.WriteFile(filePath, []byte("test"), tt.permissions)

			// Test reading the file
			content, err := ReadOutputFile(filePath)
			couldRead := err == nil

			if couldRead != tt.canRead {
				t.Errorf("Permission validation failed for %s: could read = %v, expected = %v",
					tt.name, couldRead, tt.canRead)
			}

			if tt.canRead && content != "test" {
				t.Errorf("Content mismatch for readable file %s", tt.name)
			}

			// Cleanup
			os.Chmod(filePath, 0644) // Reset permissions to allow deletion
			os.Remove(filePath)
		})
	}
}

func TestSpecialFileTypes(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping special file tests on Windows")
	}

	tempDir := t.TempDir()

	// Test handling of special file types
	t.Run("symlink_file", func(t *testing.T) {
		// Create a real file
		realFile := filepath.Join(tempDir, "real.mp3")
		ioutil.WriteFile(realFile, []byte("content"), 0644)

		// Create a symlink
		linkFile := filepath.Join(tempDir, "link.mp3")
		os.Symlink(realFile, linkFile)

		// GetAllFiles should find both
		files, err := GetAllFiles(tempDir, "mp3")
		if err != nil {
			t.Errorf("GetAllFiles() error = %v", err)
		}

		if len(files) != 2 {
			t.Errorf("Expected 2 files (real + symlink), got %d", len(files))
		}

		// Cleanup
		os.Remove(linkFile)
		os.Remove(realFile)
	})

	t.Run("broken_symlink", func(t *testing.T) {
		// Create a symlink to non-existent file
		linkFile := filepath.Join(tempDir, "broken.mp3")
		os.Symlink("/non/existent/file.mp3", linkFile)

		// GetAllFiles should handle gracefully
		files, err := GetAllFiles(tempDir, "mp3")
		if err != nil {
			t.Errorf("GetAllFiles() error with broken symlink = %v", err)
		}

		// The broken symlink might or might not be included depending on implementation
		t.Logf("Found %d files with broken symlink", len(files))

		// Cleanup
		os.Remove(linkFile)
	})
}

func TestFileNameValidation(t *testing.T) {
	tempDir := t.TempDir()

	// Test files with various naming patterns
	testFiles := []struct {
		name       string
		filename   string
		shouldWork bool
	}{
		{"normal_name", "normal_audio.mp3", true},
		{"numbers_in_name", "track01.mp3", true},
		{"dash_underscore", "my-audio_file.mp3", true},
		{"unicode_name", "音楽ファイル.mp3", true},
		{"emoji_in_name", "🎵music.mp3", true},
		{"spaces_in_name", "my audio file.mp3", true},
		{"parentheses", "audio (remix).mp3", true},
		{"brackets", "audio [version 2].mp3", true},
		{"special_chars", "audio@2x.mp3", true},
		{"dots_in_name", "audio.v2.final.mp3", true},
		{"very_long_name", string(make([]byte, 200)) + ".mp3", true},
	}

	// Initialize very long name
	for i, tf := range testFiles {
		if tf.name == "very_long_name" {
			longName := ""
			for j := 0; j < 200; j++ {
				longName += "a"
			}
			testFiles[i].filename = longName + ".mp3"
		}
	}

	for _, tt := range testFiles {
		t.Run(tt.name, func(t *testing.T) {
			if !tt.shouldWork {
				return
			}

			filePath := filepath.Join(tempDir, tt.filename)

			// Try to create the file
			err := ioutil.WriteFile(filePath, []byte("test"), 0644)
			if err != nil {
				if tt.shouldWork {
					t.Errorf("Failed to create file with name %s: %v", tt.name, err)
				}
				return
			}

			// Try to find it with GetAllFiles
			files, err := GetAllFiles(tempDir, "mp3")
			if err != nil {
				t.Errorf("GetAllFiles() error = %v", err)
				return
			}

			found := false
			for _, f := range files {
				if f.Name == tt.filename {
					found = true
					break
				}
			}

			if !found && tt.shouldWork {
				t.Errorf("File with name pattern %s not found", tt.name)
			}

			// Cleanup
			os.Remove(filePath)
		})
	}
}

func TestFileContentValidation(t *testing.T) {
	tempDir := t.TempDir()

	tests := []struct {
		name            string
		content         string
		expectedContent string
		trimSpace       bool
	}{
		{
			name:            "normal_text",
			content:         "Hello, World!",
			expectedContent: "Hello, World!",
			trimSpace:       false,
		},
		{
			name:            "text_with_newlines",
			content:         "Line 1\nLine 2\nLine 3",
			expectedContent: "Line 1\nLine 2\nLine 3",
			trimSpace:       false,
		},
		{
			name:            "text_with_whitespace",
			content:         "  \n\tContent\n\t  ",
			expectedContent: "Content",
			trimSpace:       true,
		},
		{
			name:            "unicode_content",
			content:         "Hello 世界 🌍",
			expectedContent: "Hello 世界 🌍",
			trimSpace:       false,
		},
		{
			name:            "binary_content",
			content:         string([]byte{0x00, 0x01, 0x02, 0x03}),
			expectedContent: string([]byte{0x00, 0x01, 0x02, 0x03}),
			trimSpace:       false,
		},
		{
			name:            "empty_content",
			content:         "",
			expectedContent: "",
			trimSpace:       false,
		},
		{
			name:            "only_whitespace",
			content:         "   \n\t\r\n   ",
			expectedContent: "",
			trimSpace:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filename := tt.name + ".txt"
			filePath := filepath.Join(tempDir, filename)

			// Write content
			err := WriteToFile(tt.content, filePath)
			if err != nil {
				t.Errorf("WriteToFile() error = %v", err)
				return
			}

			// Read content back
			gotContent, err := ReadOutputFile(filePath)
			if err != nil {
				t.Errorf("ReadOutputFile() error = %v", err)
				return
			}

			// ReadOutputFile always trims space
			expectedAfterRead := tt.expectedContent
			if tt.trimSpace || true { // ReadOutputFile uses TrimSpace
				expectedAfterRead = tt.expectedContent
			}

			if gotContent != expectedAfterRead {
				t.Errorf("Content mismatch:\ngot:      %q\nexpected: %q",
					gotContent, expectedAfterRead)
			}

			// Cleanup
			os.Remove(filePath)
		})
	}
}

func TestConcurrentFileOperations(t *testing.T) {
	tempDir := t.TempDir()

	// Test concurrent reads and writes
	t.Run("concurrent_writes", func(t *testing.T) {
		done := make(chan bool, 10)

		for i := 0; i < 10; i++ {
			go func(index int) {
				filename := filepath.Join(tempDir, fmt.Sprintf("concurrent_%d.txt", index))
				content := fmt.Sprintf("Content for file %d", index)

				err := WriteToFile(content, filename)
				if err != nil {
					t.Errorf("Concurrent write failed for file %d: %v", index, err)
				}

				done <- true
			}(i)
		}

		// Wait for all writes to complete
		for i := 0; i < 10; i++ {
			<-done
		}

		// Verify all files were created
		files, err := GetAllFiles(tempDir, "txt")
		if err != nil {
			t.Errorf("GetAllFiles() error = %v", err)
		}

		if len(files) != 10 {
			t.Errorf("Expected 10 files from concurrent writes, got %d", len(files))
		}
	})

	t.Run("concurrent_reads", func(t *testing.T) {
		// Create a test file
		testFile := filepath.Join(tempDir, "read_test.txt")
		WriteToFile("Test content for concurrent reads", testFile)

		done := make(chan bool, 10)

		for i := 0; i < 10; i++ {
			go func() {
				content, err := ReadOutputFile(testFile)
				if err != nil {
					t.Errorf("Concurrent read failed: %v", err)
				}
				if content != "Test content for concurrent reads" {
					t.Errorf("Concurrent read returned wrong content: %s", content)
				}
				done <- true
			}()
		}

		// Wait for all reads to complete
		for i := 0; i < 10; i++ {
			<-done
		}
	})
}
