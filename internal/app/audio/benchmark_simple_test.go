package audio

import (
	"fmt"
	"math/rand"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// BenchmarkPathOperations benchmarks path manipulation operations
func BenchmarkPathOperations(b *testing.B) {
	testPaths := []string{
		"audio.mp3",
		"/simple/path/audio.mp3",
		"/very/long/path/with/many/directories/and/subdirectories/audio.mp3",
		"/path/with spaces/and (special) chars/audio.mp3",
		"C:\\Windows\\Path\\With\\Backslashes\\audio.mp3",
		"/Users/username/Documents/Projects/Audio/Recordings/2023/January/Session1/audio.mp3",
	}
	
	b.Run("TrimSuffix", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			for _, path := range testPaths {
				_ = strings.TrimSuffix(path, filepath.Ext(path))
			}
		}
	})
	
	b.Run("FilePathExt", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			for _, path := range testPaths {
				_ = filepath.Ext(path)
			}
		}
	})
	
	b.Run("CombineOperations", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			for _, path := range testPaths {
				_ = strings.TrimSuffix(path, filepath.Ext(path)) + "_16khz.wav"
			}
		}
	})
}

// BenchmarkStringOperations benchmarks string operations used in audio processing
func BenchmarkStringOperations(b *testing.B) {
	testStrings := []string{
		"30.123456",
		"1234.567890",
		"0.000000",
		"3600.750000",
		"invalid_duration",
		"  \t 120.5  \n",
	}
	
	b.Run("StringsTrimSpace", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			for _, s := range testStrings {
				_ = strings.TrimSpace(s)
			}
		}
	})
	
	b.Run("StringsToLower", func(b *testing.B) {
		extensions := []string{".MP3", ".WAV", ".M4A", ".FLAC", ".OGG"}
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			for _, ext := range extensions {
				_ = strings.ToLower(ext)
			}
		}
	})
	
	b.Run("StringsReplace", func(b *testing.B) {
		filenames := []string{"audio.mp4", "video.mp4", "test.mp4", "movie.mp4"}
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			for _, filename := range filenames {
				_ = strings.Replace(filename, ".mp4", ".mp3", 1)
			}
		}
	})
}

// BenchmarkFormatValidation benchmarks format validation logic
func BenchmarkFormatValidation(b *testing.B) {
	testFiles := []string{
		"audio.mp3",
		"audio.m4a", 
		"audio.wav",
		"audio.flac",
		"audio.ogg",
		"audio.aac",
		"audio.MP3",
		"audio.M4A",
		"audio.WAV",
		"audio.txt",
		"audio",
		"audio.",
	}
	
	b.Run("ExtensionCheck", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			for _, file := range testFiles {
				ext := strings.ToLower(filepath.Ext(file))
				_ = ext == ".mp3" || ext == ".m4a" || ext == ".wav"
			}
		}
	})
	
	b.Run("MapLookup", func(b *testing.B) {
		supportedFormats := map[string]bool{
			".mp3": true,
			".m4a": true,
			".wav": true,
		}
		
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			for _, file := range testFiles {
				ext := strings.ToLower(filepath.Ext(file))
				_ = supportedFormats[ext]
			}
		}
	})
}

// BenchmarkDurationParsing benchmarks duration parsing logic
func BenchmarkDurationParsing(b *testing.B) {
	durations := []string{
		"30.123456",
		"1234.567890",
		"0.000000",
		"3600.750000",
		"120.5",
		"5.0",
		"45.678",
		"29.4",
		"29.5",
		"7200.0",
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, dur := range durations {
			_, _ = parseDurationOutput(dur + "\n")
		}
	}
}

// BenchmarkRandomWorkload benchmarks with randomized workloads
func BenchmarkRandomWorkload(b *testing.B) {
	if testing.Short() {
		b.Skip("skipping randomized benchmark in short mode")
	}
	
	rand.Seed(time.Now().UnixNano())
	
	formats := []string{".mp3", ".wav", ".m4a", ".flac", ".ogg"}
	operations := []string{"path_transform", "format_check", "duration_parse"}
	
	// Pre-generate random test data
	testData := make([]struct {
		filename  string
		format    string
		operation string
		duration  string
	}, 1000)
	
	for i := range testData {
		testData[i] = struct {
			filename  string
			format    string
			operation string
			duration  string
		}{
			filename:  fmt.Sprintf("audio_%d%s", i, formats[rand.Intn(len(formats))]),
			format:    formats[rand.Intn(len(formats))],
			operation: operations[rand.Intn(len(operations))],
			duration:  fmt.Sprintf("%.6f", rand.Float64()*3600),
		}
	}
	
	b.Run("RandomOperations", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			data := testData[i%len(testData)]
			
			switch data.operation {
			case "path_transform":
				_ = strings.TrimSuffix(data.filename, filepath.Ext(data.filename)) + "_16khz.wav"
			case "format_check":
				ext := strings.ToLower(filepath.Ext(data.filename))
				_ = ext == ".mp3" || ext == ".m4a" || ext == ".wav"
			case "duration_parse":
				_, _ = parseDurationOutput(data.duration + "\n")
			}
		}
	})
}

// BenchmarkConcurrentPathOperations benchmarks concurrent path operations
func BenchmarkConcurrentPathOperations(b *testing.B) {
	testPaths := []string{
		"audio1.mp3", "audio2.wav", "audio3.m4a", "audio4.flac",
		"video1.mp4", "video2.avi", "video3.mkv", "video4.mov",
	}
	
	b.Run("Sequential", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			for _, path := range testPaths {
				_ = strings.TrimSuffix(path, filepath.Ext(path)) + "_16khz.wav"
			}
		}
	})
	
	b.Run("Concurrent", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			done := make(chan string, len(testPaths))
			for _, path := range testPaths {
				go func(p string) {
					result := strings.TrimSuffix(p, filepath.Ext(p)) + "_16khz.wav"
					done <- result
				}(path)
			}
			
			// Collect results
			for range testPaths {
				<-done
			}
		}
	})
}

// BenchmarkMemoryAllocation benchmarks memory allocation patterns
func BenchmarkMemoryAllocation(b *testing.B) {
	b.Run("StringConcatenation", func(b *testing.B) {
		filenames := []string{"audio1", "audio2", "audio3", "audio4", "audio5"}
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			for _, name := range filenames {
				_ = name + "_16khz.wav"
			}
		}
	})
	
	b.Run("StringsBuilder", func(b *testing.B) {
		filenames := []string{"audio1", "audio2", "audio3", "audio4", "audio5"}
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			for _, name := range filenames {
				var builder strings.Builder
				builder.WriteString(name)
				builder.WriteString("_16khz.wav")
				_ = builder.String()
			}
		}
	})
}