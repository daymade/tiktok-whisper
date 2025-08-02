package files

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestIntegrationWithRealAudioFiles(t *testing.T) {
	// Get project root to find test audio files
	projectRoot, err := GetProjectRoot()
	if err != nil {
		t.Fatalf("Failed to get project root: %v", err)
	}

	audioDir := filepath.Join(projectRoot, "test", "data", "audio")

	tests := []struct {
		name      string
		extension string
		minCount  int
	}{
		{"mp3_files", "mp3", 1},
		{"wav_files", "wav", 1},
		{"m4a_files", "m4a", 1},
		{"flac_files", "flac", 0}, // Optional
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			files, err := GetAllFiles(audioDir, tt.extension)
			if err != nil {
				t.Errorf("GetAllFiles() error = %v", err)
				return
			}

			if len(files) < tt.minCount {
				t.Errorf("Expected at least %d %s files, got %d", tt.minCount, tt.extension, len(files))
			}

			// Verify all files have correct extension and valid paths
			for _, file := range files {
				if !strings.HasSuffix(strings.ToLower(file.Name), "."+strings.ToLower(tt.extension)) {
					t.Errorf("File %s doesn't have correct extension %s", file.Name, tt.extension)
				}

				if !filepath.IsAbs(file.FullPath) {
					t.Errorf("File path is not absolute: %s", file.FullPath)
				}

				// Verify the path actually points to the audio directory
				if !strings.Contains(file.FullPath, "test/data/audio") {
					t.Errorf("File path doesn't point to audio directory: %s", file.FullPath)
				}
			}

			t.Logf("Found %d %s files in test audio directory", len(files), tt.extension)
		})
	}
}

func TestIntegrationUserDirectoryCreation(t *testing.T) {
	// Test creating user directories similar to real usage
	testUsers := []string{
		"test_user_1",
		"测试用户",
		"test user with spaces",
		"user@domain.com",
	}

	for _, user := range testUsers {
		t.Run("user_"+user, func(t *testing.T) {
			// Get the directory path
			userDir := GetUserMp3Dir(user)

			// Verify the path is constructed correctly
			if !strings.Contains(userDir, "data/mp3") {
				t.Errorf("User directory doesn't contain expected path: %s", userDir)
			}

			if !strings.Contains(userDir, user) {
				t.Errorf("User directory doesn't contain username: %s", userDir)
			}

			// Note: We don't actually create the directory in integration tests
			// as that would modify the project structure
			t.Logf("User %s would have directory: %s", user, userDir)
		})
	}
}

func TestIntegrationProjectStructure(t *testing.T) {
	// Verify our understanding of the project structure
	projectRoot, err := GetProjectRoot()
	if err != nil {
		t.Fatalf("Failed to get project root: %v", err)
	}

	expectedDirs := []string{
		"cmd",
		"internal",
		"test",
		"data",
		"scripts",
	}

	for _, dir := range expectedDirs {
		t.Run("check_"+dir, func(t *testing.T) {
			dirPath := filepath.Join(projectRoot, dir)
			absPath, err := GetAbsolutePath(dirPath)
			if err != nil {
				t.Errorf("GetAbsolutePath failed for %s: %v", dir, err)
			}

			if !strings.HasSuffix(absPath, dir) {
				t.Errorf("Absolute path doesn't end with expected directory: %s", absPath)
			}

			t.Logf("Directory %s resolves to: %s", dir, absPath)
		})
	}
}

func TestIntegrationGoModDetection(t *testing.T) {
	// Test that go.mod detection works from various subdirectories
	projectRoot, err := GetProjectRoot()
	if err != nil {
		t.Fatalf("Failed to get project root: %v", err)
	}

	// Test from different subdirectories
	testDirs := []string{
		"internal/app/util/files",
		"cmd/v2t",
		"test/data",
		"scripts",
	}

	for _, subDir := range testDirs {
		t.Run("from_"+strings.ReplaceAll(subDir, "/", "_"), func(t *testing.T) {
			fullPath := filepath.Join(projectRoot, subDir)

			// findGoModRoot should find the same root from any subdirectory
			foundRoot, err := findGoModRoot(fullPath)
			if err != nil {
				t.Errorf("findGoModRoot failed from %s: %v", subDir, err)
				return
			}

			if foundRoot != projectRoot {
				t.Errorf("Different root found from %s: got %s, want %s", subDir, foundRoot, projectRoot)
			}

			t.Logf("From %s, found root: %s", subDir, foundRoot)
		})
	}
}

func TestIntegrationFileOperationsWorkflow(t *testing.T) {
	// Test a complete workflow similar to real application usage

	// 1. Get project root
	projectRoot, err := GetProjectRoot()
	if err != nil {
		t.Fatalf("Failed to get project root: %v", err)
	}

	// 2. Create user directory path
	testUser := "integration_test_user"
	userMp3Dir := GetUserMp3Dir(testUser)

	// 3. Verify path is constructed correctly
	if !strings.HasPrefix(userMp3Dir, projectRoot) {
		t.Errorf("User MP3 directory not under project root: %s", userMp3Dir)
	}

	// 4. Get absolute path of audio directory
	audioPath := filepath.Join(projectRoot, "test", "data", "audio")
	absolutePath, err := GetAbsolutePath(audioPath)
	if err != nil {
		t.Errorf("GetAbsolutePath failed: %v", err)
	}

	if !filepath.IsAbs(absolutePath) {
		t.Errorf("Returned path is not absolute: %s", absolutePath)
	}

	// 5. List audio files
	audioFiles, err := GetAllFiles(absolutePath, "mp3")
	if err != nil {
		t.Errorf("GetAllFiles failed: %v", err)
	}

	t.Logf("Integration test workflow completed successfully:")
	t.Logf("  Project root: %s", projectRoot)
	t.Logf("  User MP3 dir: %s", userMp3Dir)
	t.Logf("  Audio directory: %s", absolutePath)
	t.Logf("  Found MP3 files: %d", len(audioFiles))
}
