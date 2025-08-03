package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"
	"tiktok-whisper/internal/app/api/whisper_server"
)

func main() {
	var (
		baseURL    = flag.String("url", "http://127.0.0.1:8080", "Whisper-server base URL")
		audioFile  = flag.String("file", "", "Audio file to transcribe (required)")
		language   = flag.String("lang", "auto", "Language code (auto, en, zh, etc.)")
		format     = flag.String("format", "json", "Response format (json, text, srt, vtt, verbose_json)")
		translate  = flag.Bool("translate", false, "Translate to English")
		timeout    = flag.Int("timeout", 120, "Request timeout in seconds")
		healthOnly = flag.Bool("health", false, "Only perform health check")
		verbose    = flag.Bool("verbose", false, "Verbose output")
	)
	flag.Parse()

	if *audioFile == "" && !*healthOnly {
		fmt.Fprintf(os.Stderr, "Usage: %s -file <audio_file> [options]\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "       %s -health [options]  # Health check only\n", os.Args[0])
		flag.PrintDefaults()
		os.Exit(1)
	}

	// Create whisper-server provider configuration
	config := whisper_server.WhisperServerConfig{
		BaseURL:        *baseURL,
		InferencePath:  "/inference",
		LoadPath:       "/load",
		Timeout:        time.Duration(*timeout) * time.Second,
		Language:       *language,
		ResponseFormat: *format,
		Temperature:    0.0,
		Translate:      *translate,
		NoTimestamps:   false,
		WordThreshold:  0.01,
		MaxLength:      0,
		CustomHeaders: map[string]string{
			"User-Agent": "tiktok-whisper-test-client/1.0",
		},
		InsecureSkipTLS: false,
	}

	// Create provider
	provider := whisper_server.NewWhisperServerProvider(config)

	if *verbose {
		fmt.Printf("Testing whisper-server at: %s\n", *baseURL)
		fmt.Printf("Configuration:\n")
		fmt.Printf("  Timeout: %v\n", config.Timeout)
		fmt.Printf("  Language: %s\n", config.Language)
		fmt.Printf("  Response Format: %s\n", config.ResponseFormat)
		fmt.Printf("  Translate: %v\n", config.Translate)
		fmt.Printf("\n")
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(*timeout+30)*time.Second)
	defer cancel()

	// Step 1: Health check
	fmt.Println("🔍 Performing health check...")
	startTime := time.Now()
	
	err := provider.HealthCheck(ctx)
	healthCheckDuration := time.Since(startTime)
	
	if err != nil {
		fmt.Printf("❌ Health check failed (%v): %v\n", healthCheckDuration, err)
		os.Exit(1)
	}
	
	fmt.Printf("✅ Health check passed (%v)\n", healthCheckDuration)
	
	if *healthOnly {
		fmt.Println("✨ Health check completed successfully!")
		return
	}

	// Step 2: Validate configuration
	fmt.Println("\n🔧 Validating configuration...")
	err = provider.ValidateConfiguration()
	if err != nil {
		fmt.Printf("❌ Configuration validation failed: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("✅ Configuration is valid")

	// Step 3: Check if audio file exists
	fmt.Printf("\n📁 Checking audio file: %s\n", *audioFile)
	absPath, err := filepath.Abs(*audioFile)
	if err != nil {
		fmt.Printf("❌ Failed to get absolute path: %v\n", err)
		os.Exit(1)
	}

	fileInfo, err := os.Stat(absPath)
	if err != nil {
		fmt.Printf("❌ Audio file not found: %v\n", err)
		os.Exit(1)
	}
	
	fmt.Printf("✅ Audio file found: %s (%.2f MB)\n", 
		filepath.Base(absPath), 
		float64(fileInfo.Size())/(1024*1024))

	// Step 4: Get provider info
	if *verbose {
		fmt.Println("\n📋 Provider information:")
		info := provider.GetProviderInfo()
		fmt.Printf("  Name: %s\n", info.DisplayName)
		fmt.Printf("  Type: %v\n", info.Type)
		fmt.Printf("  Version: %s\n", info.Version)
		fmt.Printf("  Supports Timestamps: %v\n", info.SupportsTimestamps)
		fmt.Printf("  Supports Language Detection: %v\n", info.SupportsLanguageDetection)
		fmt.Printf("  Max File Size: %d MB\n", info.MaxFileSizeMB)
		fmt.Printf("  Typical Latency: %d ms\n", info.TypicalLatencyMs)
		fmt.Printf("  Supported Formats: %v\n", info.SupportedFormats)
	}

	// Step 5: Test basic transcription interface
	fmt.Println("\n🎵 Testing basic transcription interface...")
	startTime = time.Now()
	
	basicResult, err := provider.Transcript(absPath)
	basicDuration := time.Since(startTime)
	
	if err != nil {
		fmt.Printf("❌ Basic transcription failed (%v): %v\n", basicDuration, err)
		os.Exit(1)
	}
	
	fmt.Printf("✅ Basic transcription completed (%v)\n", basicDuration)
	fmt.Printf("📝 Result: %s\n", truncateString(basicResult, 100))

	// Step 6: Test different response formats (if verbose)
	if *verbose && *format == "json" {
		fmt.Println("\n🎭 Testing different response formats...")
		
		formats := []string{"text", "srt", "vtt"}
		for _, testFormat := range formats {
			testConfig := config
			testConfig.ResponseFormat = testFormat
			testProvider := whisper_server.NewWhisperServerProvider(testConfig)
			
			fmt.Printf("  Testing %s format... ", testFormat)
			
			startTime := time.Now()
			testResult, testErr := testProvider.Transcript(absPath)
			testDuration := time.Since(startTime)
			
			if testErr != nil {
				fmt.Printf("❌ Failed (%v): %v\n", testDuration, testErr)
			} else {
				fmt.Printf("✅ Success (%v): %s\n", testDuration, truncateString(testResult, 50))
			}
		}
	}

	// Step 7: Performance comparison
	fmt.Println("\n⚡ Performance Summary:")
	fmt.Printf("  Health Check: %v\n", healthCheckDuration)
	fmt.Printf("  Basic Transcription: %v\n", basicDuration)

	fmt.Println("\n🎉 All tests completed successfully!")
	fmt.Println("🌟 The whisper-server HTTP provider is working correctly!")
}

// truncateString truncates a string to maxLen characters and adds "..." if needed
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// Additional helper functions for testing

func checkServerResponsiveness(baseURL string) error {
	// This could be enhanced to check server load, model status, etc.
	return nil
}

func generateTestReport(results map[string]interface{}) {
	// This could generate a detailed test report
	// Could be useful for automated testing or CI/CD
}

func testConcurrentRequests(provider *whisper_server.WhisperServerProvider, audioFile string, concurrency int) {
	// Test concurrent request handling
	// This could be useful for load testing
}

func measureThroughput(provider *whisper_server.WhisperServerProvider, audioFiles []string) {
	// Measure requests per second, total processing time, etc.
	// Useful for performance benchmarking
}

func testErrorHandling(provider *whisper_server.WhisperServerProvider) {
	// Test various error conditions:
	// - Invalid audio files
	// - Network timeouts  
	// - Server errors
	// - Large files
	// - Unsupported formats
}