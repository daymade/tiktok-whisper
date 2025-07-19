package files

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"
	"testing"
)

func BenchmarkGetAbsolutePath(b *testing.B) {
	testCases := []struct {
		name string
		path string
	}{
		{"absolute_path", "/usr/local/bin/test"},
		{"relative_path", "test/data/file.txt"},
		{"current_dir", "."},
		{"parent_dir", ".."},
		{"nested_relative", "../../test/deep/path"},
	}

	for _, tc := range testCases {
		b.Run(tc.name, func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, err := GetAbsolutePath(tc.path)
				if err != nil {
					b.Errorf("GetAbsolutePath failed: %v", err)
				}
			}
		})
	}
}

func BenchmarkGetAllFiles(b *testing.B) {
	// Create a temporary directory structure for benchmarking
	tempDir, err := ioutil.TempDir("", "benchmark_*")
	if err != nil {
		b.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create test files with different counts
	fileCounts := []int{10, 100, 1000}
	
	for _, count := range fileCounts {
		benchDir := filepath.Join(tempDir, fmt.Sprintf("files_%d", count))
		os.MkdirAll(benchDir, 0755)

		// Create files with different extensions
		for i := 0; i < count; i++ {
			var ext string
			switch i % 4 {
			case 0:
				ext = "mp3"
			case 1:
				ext = "wav"
			case 2:
				ext = "m4a"
			case 3:
				ext = "txt"
			}
			
			fileName := fmt.Sprintf("file_%04d.%s", i, ext)
			filePath := filepath.Join(benchDir, fileName)
			ioutil.WriteFile(filePath, []byte("test content"), 0644)
		}

		b.Run(fmt.Sprintf("files_%d", count), func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, err := GetAllFiles(benchDir, "mp3")
				if err != nil {
					b.Errorf("GetAllFiles failed: %v", err)
				}
			}
		})
	}
}

func BenchmarkReadOutputFile(b *testing.B) {
	tempDir, err := ioutil.TempDir("", "read_benchmark_*")
	if err != nil {
		b.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create files with different sizes
	fileSizes := []struct {
		name string
		size int
	}{
		{"small_1kb", 1024},
		{"medium_100kb", 100 * 1024},
		{"large_1mb", 1024 * 1024},
		{"xlarge_10mb", 10 * 1024 * 1024},
	}

	for _, fs := range fileSizes {
		content := make([]byte, fs.size)
		for i := range content {
			content[i] = byte('a' + (i % 26))
		}

		filePath := filepath.Join(tempDir, fs.name+".txt")
		ioutil.WriteFile(filePath, content, 0644)

		b.Run(fs.name, func(b *testing.B) {
			b.SetBytes(int64(fs.size))
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, err := ReadOutputFile(filePath)
				if err != nil {
					b.Errorf("ReadOutputFile failed: %v", err)
				}
			}
		})
	}
}

func BenchmarkWriteToFile(b *testing.B) {
	tempDir, err := ioutil.TempDir("", "write_benchmark_*")
	if err != nil {
		b.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Test writing files of different sizes
	contentSizes := []struct {
		name string
		size int
	}{
		{"small_1kb", 1024},
		{"medium_100kb", 100 * 1024},
		{"large_1mb", 1024 * 1024},
		{"xlarge_10mb", 10 * 1024 * 1024},
	}

	for _, cs := range contentSizes {
		content := make([]byte, cs.size)
		for i := range content {
			content[i] = byte('a' + (i % 26))
		}
		contentStr := string(content)

		b.Run(cs.name, func(b *testing.B) {
			b.SetBytes(int64(cs.size))
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				filePath := filepath.Join(tempDir, fmt.Sprintf("%s_%d.txt", cs.name, i))
				err := WriteToFile(contentStr, filePath)
				if err != nil {
					b.Errorf("WriteToFile failed: %v", err)
				}
			}
		})
	}
}

func BenchmarkCheckAndCreateMP3Directory(b *testing.B) {
	tempDir, err := ioutil.TempDir("", "create_benchmark_*")
	if err != nil {
		b.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	b.Run("create_new_directory", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			dirPath := filepath.Join(tempDir, fmt.Sprintf("mp3_dir_%d", i))
			CheckAndCreateMP3Directory(dirPath)
		}
	})

	// Create a directory once and then test repeated calls
	existingDir := filepath.Join(tempDir, "existing_mp3")
	CheckAndCreateMP3Directory(existingDir)

	b.Run("existing_directory", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			CheckAndCreateMP3Directory(existingDir)
		}
	})
}

func BenchmarkFindGoModRoot(b *testing.B) {
	// Create a deep directory structure with go.mod at the root
	tempDir, err := ioutil.TempDir("", "gomod_benchmark_*")
	if err != nil {
		b.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create go.mod at root
	goModPath := filepath.Join(tempDir, "go.mod")
	ioutil.WriteFile(goModPath, []byte("module test\n"), 0644)

	// Create deep directory structure
	depths := []int{1, 5, 10, 20}
	
	for _, depth := range depths {
		deepPath := tempDir
		for i := 0; i < depth; i++ {
			deepPath = filepath.Join(deepPath, fmt.Sprintf("level_%d", i))
		}
		os.MkdirAll(deepPath, 0755)

		b.Run(fmt.Sprintf("depth_%d", depth), func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, err := findGoModRoot(deepPath)
				if err != nil {
					b.Errorf("findGoModRoot failed: %v", err)
				}
			}
		})
	}
}

func BenchmarkGetUserMp3Dir(b *testing.B) {
	usernames := []string{
		"simple",
		"user_with_underscores",
		"user-with-dashes",
		"用户名带中文",
		"user@domain.com",
		"very_long_username_that_might_be_used_in_some_systems",
	}

	for _, username := range usernames {
		b.Run(username, func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = GetUserMp3Dir(username)
			}
		})
	}
}

func BenchmarkConcurrentFileOperations(b *testing.B) {
	tempDir, err := ioutil.TempDir("", "concurrent_benchmark_*")
	if err != nil {
		b.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Benchmark concurrent writes
	b.Run("concurrent_writes", func(b *testing.B) {
		b.RunParallel(func(pb *testing.PB) {
			i := 0
			for pb.Next() {
				filePath := filepath.Join(tempDir, fmt.Sprintf("concurrent_write_%d.txt", i))
				content := fmt.Sprintf("Content for concurrent write %d", i)
				err := WriteToFile(content, filePath)
				if err != nil {
					b.Errorf("Concurrent write failed: %v", err)
				}
				i++
			}
		})
	})

	// Create a test file for concurrent reads
	testFile := filepath.Join(tempDir, "read_test.txt")
	WriteToFile("Test content for concurrent reads", testFile)

	b.Run("concurrent_reads", func(b *testing.B) {
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				_, err := ReadOutputFile(testFile)
				if err != nil {
					b.Errorf("Concurrent read failed: %v", err)
				}
			}
		})
	})

	// Benchmark concurrent directory operations
	b.Run("concurrent_directory_creation", func(b *testing.B) {
		b.RunParallel(func(pb *testing.PB) {
			i := 0
			for pb.Next() {
				dirPath := filepath.Join(tempDir, fmt.Sprintf("concurrent_dir_%d", i))
				CheckAndCreateMP3Directory(dirPath)
				i++
			}
		})
	})
}

func BenchmarkMemoryUsage(b *testing.B) {
	tempDir, err := ioutil.TempDir("", "memory_benchmark_*")
	if err != nil {
		b.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create many files to test memory usage of GetAllFiles
	fileCount := 10000
	for i := 0; i < fileCount; i++ {
		fileName := fmt.Sprintf("file_%05d.mp3", i)
		filePath := filepath.Join(tempDir, fileName)
		ioutil.WriteFile(filePath, []byte("content"), 0644)
	}

	b.Run("get_all_files_memory", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			files, err := GetAllFiles(tempDir, "mp3")
			if err != nil {
				b.Errorf("GetAllFiles failed: %v", err)
			}
			if len(files) != fileCount {
				b.Errorf("Expected %d files, got %d", fileCount, len(files))
			}
		}
	})
}

func BenchmarkPathOperations(b *testing.B) {
	paths := []string{
		"simple/path",
		"very/deep/nested/path/structure/that/goes/many/levels/down",
		"path/with/unicode/文件名/测试",
		"path\\with\\windows\\separators",
		"./relative/../path/./to/file",
		"/absolute/path/to/file",
	}

	b.Run("path_resolution", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			for _, path := range paths {
				_, err := GetAbsolutePath(path)
				if err != nil {
					b.Errorf("GetAbsolutePath failed for %s: %v", path, err)
				}
			}
		}
	})
}

func BenchmarkFileSystemStress(b *testing.B) {
	tempDir, err := ioutil.TempDir("", "stress_benchmark_*")
	if err != nil {
		b.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	b.Run("stress_test", func(b *testing.B) {
		b.RunParallel(func(pb *testing.PB) {
			i := 0
			for pb.Next() {
				// Mix of operations
				switch i % 4 {
				case 0:
					// Write file
					filePath := filepath.Join(tempDir, fmt.Sprintf("stress_%d.txt", i))
					WriteToFile(fmt.Sprintf("Content %d", i), filePath)
				case 1:
					// Read directory
					GetAllFiles(tempDir, "txt")
				case 2:
					// Create directory
					dirPath := filepath.Join(tempDir, fmt.Sprintf("stress_dir_%d", i))
					CheckAndCreateMP3Directory(dirPath)
				case 3:
					// Get absolute path
					GetAbsolutePath(fmt.Sprintf("stress/path/%d", i))
				}
				i++
			}
		})
	})
}

// Benchmark with different file system types (if available)
func BenchmarkFileSystemTypes(b *testing.B) {
	// This benchmark tests performance on different mount points
	// Results may vary significantly based on the underlying filesystem
	
	testDirs := []struct {
		name string
		path string
	}{
		{"temp_dir", ""}, // Will use system temp
		{"current_dir", "."},
	}

	for _, td := range testDirs {
		b.Run(td.name, func(b *testing.B) {
			var baseDir string
			var cleanup func()

			if td.path == "" {
				tempDir, err := ioutil.TempDir("", "fs_benchmark_*")
				if err != nil {
					b.Skip("Cannot create temp directory")
				}
				baseDir = tempDir
				cleanup = func() { os.RemoveAll(tempDir) }
			} else {
				tempDir, err := ioutil.TempDir(td.path, "fs_benchmark_*")
				if err != nil {
					b.Skip("Cannot create directory in specified path")
				}
				baseDir = tempDir
				cleanup = func() { os.RemoveAll(tempDir) }
			}
			defer cleanup()

			// Run a mix of operations
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				filePath := filepath.Join(baseDir, fmt.Sprintf("test_%d.txt", i))
				WriteToFile("test content", filePath)
				ReadOutputFile(filePath)
				os.Remove(filePath)
			}
		})
	}
}

// Test performance with different concurrency levels
func BenchmarkConcurrencyLevels(b *testing.B) {
	tempDir, err := ioutil.TempDir("", "concurrency_benchmark_*")
	if err != nil {
		b.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	concurrencyLevels := []int{1, 2, 4, 8, 16, 32}

	for _, level := range concurrencyLevels {
		b.Run(fmt.Sprintf("goroutines_%d", level), func(b *testing.B) {
			b.ResetTimer()
			b.RunParallel(func(pb *testing.PB) {
				var wg sync.WaitGroup
				sem := make(chan struct{}, level)

				i := 0
				for pb.Next() {
					wg.Add(1)
					sem <- struct{}{}
					
					go func(index int) {
						defer wg.Done()
						defer func() { <-sem }()
						
						filePath := filepath.Join(tempDir, fmt.Sprintf("concurrent_%d_%d.txt", level, index))
						WriteToFile(fmt.Sprintf("Content %d", index), filePath)
					}(i)
					i++
				}
				wg.Wait()
			})
		})
	}
}