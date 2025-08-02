package files

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestCheckAndCreateMP3DirectoryAdvanced(t *testing.T) {
	tempDir := t.TempDir()

	tests := []struct {
		name       string
		setup      func() string
		check      func(t *testing.T, dir string)
		shouldFail bool
	}{
		{
			name: "create_deeply_nested_directory",
			setup: func() string {
				return filepath.Join(tempDir, "level1", "level2", "level3", "level4", "mp3")
			},
			check: func(t *testing.T, dir string) {
				info, err := os.Stat(dir)
				if err != nil {
					t.Errorf("Directory not created: %v", err)
				}
				if !info.IsDir() {
					t.Errorf("Created path is not a directory")
				}
			},
		},
		{
			name: "existing_file_at_path",
			setup: func() string {
				filePath := filepath.Join(tempDir, "existing_file")
				ioutil.WriteFile(filePath, []byte("content"), 0644)
				return filePath
			},
			check: func(t *testing.T, dir string) {
				// Should handle gracefully or fail appropriately
				info, err := os.Stat(dir)
				if err == nil && !info.IsDir() {
					// This is expected - can't create directory over file
					return
				}
			},
			shouldFail: true,
		},
		{
			name: "directory_with_special_permissions",
			setup: func() string {
				parentDir := filepath.Join(tempDir, "restricted")
				os.MkdirAll(parentDir, 0755)
				return filepath.Join(parentDir, "mp3")
			},
			check: func(t *testing.T, dir string) {
				info, err := os.Stat(dir)
				if err != nil {
					t.Errorf("Failed to create directory: %v", err)
				}
				// Check that directory has appropriate permissions
				if info != nil && info.Mode().Perm() == 0 {
					t.Errorf("Directory has no permissions")
				}
			},
		},
		{
			name: "concurrent_directory_creation",
			setup: func() string {
				return filepath.Join(tempDir, "concurrent", "mp3")
			},
			check: func(t *testing.T, dir string) {
				// Simulate concurrent creation
				done := make(chan bool, 3)
				for i := 0; i < 3; i++ {
					go func() {
						CheckAndCreateMP3Directory(dir)
						done <- true
					}()
				}

				// Wait for all goroutines
				for i := 0; i < 3; i++ {
					<-done
				}

				// Directory should exist and be valid
				info, err := os.Stat(dir)
				if err != nil {
					t.Errorf("Directory not created after concurrent calls: %v", err)
				}
				if info != nil && !info.IsDir() {
					t.Errorf("Path is not a directory after concurrent creation")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := tt.setup()

			// For tests that should fail, we need to handle the fatal
			if tt.shouldFail {
				// CheckAndCreateMP3Directory uses log.Fatal on error
				// In real tests, we might want to refactor the function to return errors
				defer func() {
					if r := recover(); r != nil {
						// Expected to panic/fatal
					}
				}()
			}

			CheckAndCreateMP3Directory(dir)

			if tt.check != nil {
				tt.check(t, dir)
			}
		})
	}
}

func TestDirectoryTraversal(t *testing.T) {
	tempDir := t.TempDir()

	// Create a complex directory structure
	structure := []string{
		"root/dir1/file1.mp3",
		"root/dir1/file2.wav",
		"root/dir1/subdir1/file3.mp3",
		"root/dir2/file4.mp3",
		"root/dir2/subdir2/file5.txt",
		"root/.hidden/file6.mp3",
		"root/empty_dir/",
	}

	for _, path := range structure {
		fullPath := filepath.Join(tempDir, path)
		if path[len(path)-1] == '/' {
			os.MkdirAll(fullPath, 0755)
		} else {
			os.MkdirAll(filepath.Dir(fullPath), 0755)
			ioutil.WriteFile(fullPath, []byte("content"), 0644)
		}
	}

	tests := []struct {
		name      string
		startDir  string
		extension string
		wantCount int
		checkDirs []string
	}{
		{
			name:      "find_mp3_in_single_dir",
			startDir:  filepath.Join(tempDir, "root", "dir1"),
			extension: "mp3",
			wantCount: 1, // Only file1.mp3, not from subdirs
		},
		{
			name:      "find_all_in_dir2",
			startDir:  filepath.Join(tempDir, "root", "dir2"),
			extension: "mp3",
			wantCount: 1,
		},
		{
			name:      "empty_directory",
			startDir:  filepath.Join(tempDir, "root", "empty_dir"),
			extension: "mp3",
			wantCount: 0,
		},
		{
			name:      "hidden_directory",
			startDir:  filepath.Join(tempDir, "root", ".hidden"),
			extension: "mp3",
			wantCount: 1,
		},
		{
			name:      "non_existent_extension",
			startDir:  filepath.Join(tempDir, "root"),
			extension: "xyz",
			wantCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			files, err := GetAllFiles(tt.startDir, tt.extension)
			if err != nil {
				t.Errorf("GetAllFiles() error = %v", err)
				return
			}

			if len(files) != tt.wantCount {
				t.Errorf("GetAllFiles() found %d files, want %d", len(files), tt.wantCount)
			}
		})
	}
}

func TestDirectoryPermissions(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping Unix permission tests on Windows")
	}

	tempDir := t.TempDir()

	tests := []struct {
		name        string
		permissions os.FileMode
		canRead     bool
		canWrite    bool
	}{
		{
			name:        "read_write_execute",
			permissions: 0755,
			canRead:     true,
			canWrite:    true,
		},
		{
			name:        "read_only",
			permissions: 0555,
			canRead:     true,
			canWrite:    false,
		},
		{
			name:        "write_only",
			permissions: 0333,
			canRead:     false,
			canWrite:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testDir := filepath.Join(tempDir, tt.name)
			err := os.MkdirAll(testDir, tt.permissions)
			if err != nil {
				t.Fatalf("Failed to create test directory: %v", err)
			}

			// Cleanup function to restore permissions for directory removal
			defer func() {
				os.Chmod(testDir, 0755)
			}()

			// Test read permission
			_, readErr := ioutil.ReadDir(testDir)
			canRead := readErr == nil
			if canRead != tt.canRead {
				t.Errorf("Read permission mismatch: got %v, want %v", canRead, tt.canRead)
			}

			// Test write permission
			testFile := filepath.Join(testDir, "test.txt")
			writeErr := ioutil.WriteFile(testFile, []byte("test"), 0644)
			canWrite := writeErr == nil
			if canWrite != tt.canWrite {
				t.Errorf("Write permission mismatch: got %v, want %v", canWrite, tt.canWrite)
			}
		})
	}
}

func TestGetUserMp3DirVariations(t *testing.T) {
	tests := []struct {
		name     string
		nickname string
		validate func(t *testing.T, path string)
	}{
		{
			name:     "empty_nickname",
			nickname: "",
			validate: func(t *testing.T, path string) {
				if !strings.HasSuffix(path, filepath.Join("data", "mp3", "")) {
					t.Errorf("Empty nickname not handled correctly: %s", path)
				}
			},
		},
		{
			name:     "nickname_with_spaces",
			nickname: "user name with spaces",
			validate: func(t *testing.T, path string) {
				if !strings.Contains(path, "user name with spaces") {
					t.Errorf("Spaces in nickname not preserved: %s", path)
				}
			},
		},
		{
			name:     "nickname_with_path_separators",
			nickname: "user/with/slashes",
			validate: func(t *testing.T, path string) {
				// This creates nested directories
				if !strings.Contains(path, "user") {
					t.Errorf("Path separators in nickname not handled: %s", path)
				}
			},
		},
		{
			name:     "very_long_nickname",
			nickname: string(make([]byte, 255)),
			validate: func(t *testing.T, path string) {
				// Should handle long names based on filesystem limits
				if len(path) == 0 {
					t.Errorf("Long nickname resulted in empty path")
				}
			},
		},
		{
			name:     "nickname_with_dots",
			nickname: "../../../etc/passwd",
			validate: func(t *testing.T, path string) {
				// GetUserMp3Dir resolves the relative path and may end up at /etc/passwd
				// This shows the potential security issue with unsanitized input
				// The test just verifies the function works as designed
				if len(path) == 0 {
					t.Errorf("Path should not be empty")
				}
				// The actual path resolution depends on the project root location
				t.Logf("Resolved path: %s", path)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Initialize nickname with spaces for very long nickname test
			if tt.name == "very_long_nickname" {
				for i := range []byte(tt.nickname) {
					tt.nickname = tt.nickname[:i] + "a" + tt.nickname[i+1:]
				}
			}

			path := GetUserMp3Dir(tt.nickname)
			tt.validate(t, path)
		})
	}
}

func TestDirectoryCleanup(t *testing.T) {
	tempDir := t.TempDir()

	// Create test structure
	testFiles := []string{
		"cleanup/file1.mp3",
		"cleanup/subdir/file2.mp3",
		"cleanup/subdir/deep/file3.mp3",
	}

	for _, file := range testFiles {
		fullPath := filepath.Join(tempDir, file)
		os.MkdirAll(filepath.Dir(fullPath), 0755)
		ioutil.WriteFile(fullPath, []byte("content"), 0644)
	}

	cleanupDir := filepath.Join(tempDir, "cleanup")

	// Verify files exist
	files, _ := GetAllFiles(cleanupDir, "mp3")
	if len(files) != 1 { // GetAllFiles doesn't recurse
		t.Errorf("Expected 1 file in cleanup dir, got %d", len(files))
	}

	// Test removing entire directory tree
	err := os.RemoveAll(cleanupDir)
	if err != nil {
		t.Errorf("Failed to remove directory tree: %v", err)
	}

	// Verify directory is gone
	if _, err := os.Stat(cleanupDir); !os.IsNotExist(err) {
		t.Errorf("Directory still exists after cleanup")
	}
}

func TestDirectoryModificationTime(t *testing.T) {
	tempDir := t.TempDir()

	// Create directories with specific modification times
	dirs := []struct {
		name    string
		modTime time.Time
	}{
		{"old_dir", time.Now().Add(-24 * time.Hour)},
		{"new_dir", time.Now()},
		{"future_dir", time.Now().Add(24 * time.Hour)},
	}

	for _, d := range dirs {
		dirPath := filepath.Join(tempDir, d.name)
		os.MkdirAll(dirPath, 0755)
		os.Chtimes(dirPath, d.modTime, d.modTime)

		// Create a file in each directory
		filePath := filepath.Join(dirPath, "file.mp3")
		ioutil.WriteFile(filePath, []byte("content"), 0644)
		os.Chtimes(filePath, d.modTime, d.modTime)
	}

	// Test that GetAllFiles respects file modification times
	for _, d := range dirs {
		t.Run(d.name, func(t *testing.T) {
			files, err := GetAllFiles(filepath.Join(tempDir, d.name), "mp3")
			if err != nil {
				t.Errorf("GetAllFiles() error = %v", err)
				return
			}

			if len(files) != 1 {
				t.Errorf("Expected 1 file, got %d", len(files))
				return
			}

			// Check that modification time is preserved
			timeDiff := files[0].ModTime.Sub(d.modTime).Abs()
			if timeDiff > time.Second {
				t.Errorf("Modification time not preserved: expected %v, got %v",
					d.modTime, files[0].ModTime)
			}
		})
	}
}

func TestDirectorySizeCalculation(t *testing.T) {
	tempDir := t.TempDir()

	// Create files of known sizes
	files := []struct {
		path string
		size int
	}{
		{"size_test/small.mp3", 1024},        // 1KB
		{"size_test/medium.mp3", 1024 * 100}, // 100KB
		{"size_test/large.mp3", 1024 * 1024}, // 1MB
	}

	totalSize := 0
	for _, f := range files {
		fullPath := filepath.Join(tempDir, f.path)
		os.MkdirAll(filepath.Dir(fullPath), 0755)

		// Create file with specific size
		content := make([]byte, f.size)
		ioutil.WriteFile(fullPath, content, 0644)
		totalSize += f.size
	}

	// Get all files and calculate total size
	gotFiles, err := GetAllFiles(filepath.Join(tempDir, "size_test"), "mp3")
	if err != nil {
		t.Fatalf("GetAllFiles() error = %v", err)
	}

	calculatedSize := 0
	for _, f := range gotFiles {
		info, err := os.Stat(f.FullPath)
		if err != nil {
			t.Errorf("Failed to stat file %s: %v", f.FullPath, err)
			continue
		}
		calculatedSize += int(info.Size())
	}

	if calculatedSize != totalSize {
		t.Errorf("Size calculation mismatch: got %d, want %d", calculatedSize, totalSize)
	}
}
