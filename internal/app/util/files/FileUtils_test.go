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

func TestGetProjectRoot(t *testing.T) {
	// Test should work from any directory in the project
	got, err := GetProjectRoot()
	if err != nil {
		t.Fatalf("GetProjectRoot() error = %v", err)
	}
	
	// Verify that go.mod exists at the returned path
	goModPath := filepath.Join(got, "go.mod")
	if _, err := os.Stat(goModPath); os.IsNotExist(err) {
		t.Errorf("GetProjectRoot() returned path without go.mod: %v", got)
	}
	
	// Should end with the project name
	if !strings.HasSuffix(got, "tiktok-whisper") {
		t.Errorf("GetProjectRoot() path doesn't end with project name: %v", got)
	}
}

func TestGetAbsolutePath(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		wantErr bool
		check   func(t *testing.T, got string)
	}{
		{
			name: "absolute_path_unchanged",
			path: "/usr/local/bin",
			check: func(t *testing.T, got string) {
				if got != "/usr/local/bin" {
					t.Errorf("Expected unchanged absolute path, got %v", got)
				}
			},
		},
		{
			name: "relative_path_resolved",
			path: "test/data",
			check: func(t *testing.T, got string) {
				if !filepath.IsAbs(got) {
					t.Errorf("Expected absolute path, got %v", got)
				}
				if !strings.HasSuffix(got, filepath.Join("test", "data")) {
					t.Errorf("Path doesn't end with expected suffix: %v", got)
				}
			},
		},
		{
			name: "current_directory",
			path: ".",
			check: func(t *testing.T, got string) {
				wd, _ := os.Getwd()
				if got != wd {
					t.Errorf("Expected current directory %v, got %v", wd, got)
				}
			},
		},
		{
			name: "parent_directory",
			path: "..",
			check: func(t *testing.T, got string) {
				wd, _ := os.Getwd()
				expected := filepath.Dir(wd)
				if got != expected {
					t.Errorf("Expected parent directory %v, got %v", expected, got)
				}
			},
		},
		{
			name: "empty_path",
			path: "",
			check: func(t *testing.T, got string) {
				wd, _ := os.Getwd()
				if got != wd {
					t.Errorf("Expected current directory for empty path, got %v", got)
				}
			},
		},
	}
	
	// Platform-specific tests
	if runtime.GOOS == "windows" {
		tests = append(tests, struct {
			name    string
			path    string
			wantErr bool
			check   func(t *testing.T, got string)
		}{
			name: "windows_absolute_path",
			path: "C:\\Windows\\System32",
			check: func(t *testing.T, got string) {
				if got != "C:\\Windows\\System32" {
					t.Errorf("Expected unchanged Windows path, got %v", got)
				}
			},
		})
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetAbsolutePath(tt.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetAbsolutePath() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.check != nil {
				tt.check(t, got)
			}
		})
	}
}

func TestGetUserMp3Dir(t *testing.T) {
	tests := []struct {
		name         string
		userNickname string
		check        func(t *testing.T, got string)
	}{
		{
			name:         "normal_username",
			userNickname: "testuser",
			check: func(t *testing.T, got string) {
				if !strings.HasSuffix(got, filepath.Join("data", "mp3", "testuser")) {
					t.Errorf("Path doesn't end with expected suffix: %v", got)
				}
			},
		},
		{
			name:         "unicode_username",
			userNickname: "用户名",
			check: func(t *testing.T, got string) {
				if !strings.HasSuffix(got, filepath.Join("data", "mp3", "用户名")) {
					t.Errorf("Path doesn't handle Unicode correctly: %v", got)
				}
			},
		},
		{
			name:         "special_characters",
			userNickname: "user@test.com",
			check: func(t *testing.T, got string) {
				if !strings.Contains(got, "user@test.com") {
					t.Errorf("Path doesn't preserve special characters: %v", got)
				}
			},
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetUserMp3Dir(tt.userNickname)
			if tt.check != nil {
				tt.check(t, got)
			}
		})
	}
}

func TestCheckAndCreateMP3Directory(t *testing.T) {
	// Create temporary test directory
	tempDir, err := ioutil.TempDir("", "test_mp3_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)
	
	tests := []struct {
		name  string
		setup func() string
		check func(t *testing.T, mp3Dir string)
	}{
		{
			name: "create_new_directory",
			setup: func() string {
				return filepath.Join(tempDir, "new_mp3_dir")
			},
			check: func(t *testing.T, mp3Dir string) {
				if _, err := os.Stat(mp3Dir); os.IsNotExist(err) {
					t.Errorf("Directory was not created: %v", mp3Dir)
				}
			},
		},
		{
			name: "existing_directory",
			setup: func() string {
				existingDir := filepath.Join(tempDir, "existing_mp3")
				os.MkdirAll(existingDir, os.ModePerm)
				return existingDir
			},
			check: func(t *testing.T, mp3Dir string) {
				if _, err := os.Stat(mp3Dir); err != nil {
					t.Errorf("Existing directory check failed: %v", err)
				}
			},
		},
		{
			name: "nested_directory",
			setup: func() string {
				return filepath.Join(tempDir, "level1", "level2", "mp3")
			},
			check: func(t *testing.T, mp3Dir string) {
				if _, err := os.Stat(mp3Dir); os.IsNotExist(err) {
					t.Errorf("Nested directory was not created: %v", mp3Dir)
				}
			},
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mp3Dir := tt.setup()
			CheckAndCreateMP3Directory(mp3Dir)
			if tt.check != nil {
				tt.check(t, mp3Dir)
			}
		})
	}
}

func TestGetAllFiles(t *testing.T) {
	// Create temporary test directory
	tempDir, err := ioutil.TempDir("", "test_files_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)
	
	// Create test files with different extensions and modification times
	testFiles := []struct {
		name    string
		content string
		modTime time.Time
	}{
		{"file1.mp3", "content1", time.Now().Add(-3 * time.Hour)},
		{"file2.MP3", "content2", time.Now().Add(-2 * time.Hour)},
		{"file3.mp3", "content3", time.Now().Add(-1 * time.Hour)},
		{"file4.wav", "content4", time.Now()},
		{"file5.txt", "content5", time.Now()},
		{"file6.Mp3", "content6", time.Now().Add(-4 * time.Hour)},
	}
	
	for _, tf := range testFiles {
		filePath := filepath.Join(tempDir, tf.name)
		err := ioutil.WriteFile(filePath, []byte(tf.content), 0644)
		if err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}
		os.Chtimes(filePath, tf.modTime, tf.modTime)
	}
	
	tests := []struct {
		name      string
		directory string
		extension string
		wantCount int
		checkOrder bool
	}{
		{
			name:       "mp3_files_case_insensitive",
			directory:  tempDir,
			extension:  "mp3",
			wantCount:  4,
			checkOrder: true,
		},
		{
			name:      "wav_files",
			directory: tempDir,
			extension: "wav",
			wantCount: 1,
		},
		{
			name:      "txt_files",
			directory: tempDir,
			extension: "txt",
			wantCount: 1,
		},
		{
			name:      "non_existent_extension",
			directory: tempDir,
			extension: "xyz",
			wantCount: 0,
		},
		{
			name:      "relative_path",
			directory: ".",
			extension: "go",
			wantCount: -1, // Don't check count, just ensure it works
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetAllFiles(tt.directory, tt.extension)
			if err != nil {
				t.Errorf("GetAllFiles() error = %v", err)
				return
			}
			
			if tt.wantCount >= 0 && len(got) != tt.wantCount {
				t.Errorf("GetAllFiles() returned %d files, want %d", len(got), tt.wantCount)
			}
			
			// Check sorting order (oldest to newest)
			if tt.checkOrder && len(got) > 1 {
				for i := 1; i < len(got); i++ {
					if got[i].ModTime.Before(got[i-1].ModTime) {
						t.Errorf("Files not sorted by modification time: %v before %v",
							got[i].Name, got[i-1].Name)
					}
				}
			}
			
			// Verify all returned files have correct extension
			for _, f := range got {
				ext := strings.TrimPrefix(strings.ToLower(filepath.Ext(f.Name)), ".")
				if ext != strings.ToLower(tt.extension) {
					t.Errorf("File %s has wrong extension: got %s, want %s",
						f.Name, ext, tt.extension)
				}
			}
		})
	}
}

func TestReadOutputFile(t *testing.T) {
	// Create temporary test file
	tempFile, err := ioutil.TempFile("", "test_output_*.txt")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tempFile.Name())
	
	tests := []struct {
		name     string
		setup    func() string
		want     string
		wantErr  bool
	}{
		{
			name: "read_normal_content",
			setup: func() string {
				content := "Hello, World!"
				ioutil.WriteFile(tempFile.Name(), []byte(content), 0644)
				return tempFile.Name()
			},
			want: "Hello, World!",
		},
		{
			name: "read_with_whitespace",
			setup: func() string {
				content := "  \n\tHello, World!\n\t  "
				ioutil.WriteFile(tempFile.Name(), []byte(content), 0644)
				return tempFile.Name()
			},
			want: "Hello, World!",
		},
		{
			name: "read_empty_file",
			setup: func() string {
				ioutil.WriteFile(tempFile.Name(), []byte(""), 0644)
				return tempFile.Name()
			},
			want: "",
		},
		{
			name: "read_unicode_content",
			setup: func() string {
				content := "你好，世界！🌍"
				ioutil.WriteFile(tempFile.Name(), []byte(content), 0644)
				return tempFile.Name()
			},
			want: "你好，世界！🌍",
		},
		{
			name: "read_non_existent_file",
			setup: func() string {
				return "/non/existent/file.txt"
			},
			wantErr: true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filePath := tt.setup()
			got, err := ReadOutputFile(filePath)
			if (err != nil) != tt.wantErr {
				t.Errorf("ReadOutputFile() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("ReadOutputFile() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestWriteToFile(t *testing.T) {
	// Create temporary directory
	tempDir, err := ioutil.TempDir("", "test_write_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)
	
	tests := []struct {
		name     string
		content  string
		filePath string
		wantErr  bool
		check    func(t *testing.T)
	}{
		{
			name:     "write_simple_file",
			content:  "Hello, World!",
			filePath: filepath.Join(tempDir, "simple.txt"),
			check: func(t *testing.T) {
				content, err := ioutil.ReadFile(filepath.Join(tempDir, "simple.txt"))
				if err != nil {
					t.Errorf("Failed to read written file: %v", err)
				}
				if string(content) != "Hello, World!" {
					t.Errorf("File content mismatch: got %s", string(content))
				}
			},
		},
		{
			name:     "write_with_new_directory",
			content:  "Nested content",
			filePath: filepath.Join(tempDir, "new", "nested", "dir", "file.txt"),
			check: func(t *testing.T) {
				filePath := filepath.Join(tempDir, "new", "nested", "dir", "file.txt")
				if _, err := os.Stat(filepath.Dir(filePath)); os.IsNotExist(err) {
					t.Errorf("Directory was not created")
				}
				content, _ := ioutil.ReadFile(filePath)
				if string(content) != "Nested content" {
					t.Errorf("File content mismatch in nested dir")
				}
			},
		},
		{
			name:     "write_unicode_content",
			content:  "Unicode: 你好世界 🌍 émojis",
			filePath: filepath.Join(tempDir, "unicode.txt"),
			check: func(t *testing.T) {
				content, _ := ioutil.ReadFile(filepath.Join(tempDir, "unicode.txt"))
				if string(content) != "Unicode: 你好世界 🌍 émojis" {
					t.Errorf("Unicode content not preserved")
				}
			},
		},
		{
			name:     "overwrite_existing_file",
			content:  "New content",
			filePath: filepath.Join(tempDir, "existing.txt"),
			check: func(t *testing.T) {
				// Pre-create file with old content
				oldFile := filepath.Join(tempDir, "existing.txt")
				ioutil.WriteFile(oldFile, []byte("Old content"), 0644)
				
				// Write new content
				WriteToFile("New content", oldFile)
				
				// Verify new content
				content, _ := ioutil.ReadFile(oldFile)
				if string(content) != "New content" {
					t.Errorf("File was not overwritten")
				}
			},
		},
		{
			name:     "write_empty_content",
			content:  "",
			filePath: filepath.Join(tempDir, "empty.txt"),
			check: func(t *testing.T) {
				content, _ := ioutil.ReadFile(filepath.Join(tempDir, "empty.txt"))
				if len(content) != 0 {
					t.Errorf("Empty file should have no content")
				}
			},
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := WriteToFile(tt.content, tt.filePath)
			if (err != nil) != tt.wantErr {
				t.Errorf("WriteToFile() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.check != nil {
				tt.check(t)
			}
		})
	}
}

func TestFindGoModRoot(t *testing.T) {
	// Create a temporary directory structure
	tempDir, err := ioutil.TempDir("", "test_gomod_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)
	
	// Create directory structure:
	// tempDir/
	//   go.mod
	//   subdir1/
	//     subdir2/
	//       testfile.go
	goModPath := filepath.Join(tempDir, "go.mod")
	ioutil.WriteFile(goModPath, []byte("module test\n"), 0644)
	
	subdir1 := filepath.Join(tempDir, "subdir1")
	os.MkdirAll(subdir1, 0755)
	
	subdir2 := filepath.Join(subdir1, "subdir2")
	os.MkdirAll(subdir2, 0755)
	
	tests := []struct {
		name    string
		path    string
		want    string
		wantErr bool
	}{
		{
			name: "from_root_directory",
			path: tempDir,
			want: tempDir,
		},
		{
			name: "from_subdirectory",
			path: subdir1,
			want: tempDir,
		},
		{
			name: "from_nested_subdirectory",
			path: subdir2,
			want: tempDir,
		},
		{
			name:    "no_go_mod_found",
			path:    "/",
			wantErr: true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := findGoModRoot(tt.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("findGoModRoot() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("findGoModRoot() = %v, want %v", got, tt.want)
			}
		})
	}
}
