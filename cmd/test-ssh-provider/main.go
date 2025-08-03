package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"tiktok-whisper/internal/app/api/provider"
	"tiktok-whisper/internal/app/api/ssh_whisper"
)

func main() {
	fmt.Println("🚀 Testing SSH Whisper Provider Integration")
	fmt.Println(strings.Repeat("=", 60))

	// Create SSH whisper provider configuration
	config := ssh_whisper.SSHWhisperConfig{
		Host:       "user@example-host.local",
		RemoteDir:  "/path/to/whisper.cpp",
		BinaryPath: "./build/bin/whisper-cli",
		ModelPath:  "models/ggml-base.en.bin",
		Language:   "en",
		Threads:    4,
	}

	// Create provider instance
	provider := ssh_whisper.NewSSHWhisperProvider(config)
	
	// Test 1: Get provider info
	fmt.Println("\n📋 Test 1: Provider Information")
	info := provider.GetProviderInfo()
	fmt.Printf("  Name: %s\n", info.Name)
	fmt.Printf("  Display Name: %s\n", info.DisplayName)
	fmt.Printf("  Type: %s\n", info.Type)
	fmt.Printf("  Supported Formats: %v\n", info.SupportedFormats)
	fmt.Printf("  Max File Size: %d MB\n", info.MaxFileSizeMB)
	fmt.Printf("  Typical Latency: %d ms\n", info.TypicalLatencyMs)
	fmt.Printf("  Cost: %s\n", info.CostPerMinute)

	// Test 2: Configuration validation
	fmt.Println("\n🔧 Test 2: Configuration Validation")
	if err := provider.ValidateConfiguration(); err != nil {
		fmt.Printf("  ❌ Configuration validation failed: %v\n", err)
		fmt.Println("  This is expected if SSH is not configured properly")
		return
	} else {
		fmt.Println("  ✅ Configuration validation passed")
	}

	// Test 3: Health check
	fmt.Println("\n🏥 Test 3: Health Check")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := provider.HealthCheck(ctx); err != nil {
		fmt.Printf("  ❌ Health check failed: %v\n", err)
		fmt.Println("  This is expected if SSH/remote whisper is not available")
		return
	} else {
		fmt.Println("  ✅ Health check passed - SSH connection and remote whisper.cpp verified")
	}

	// Test 4: Transcription with test file
	fmt.Println("\n🎤 Test 4: Audio Transcription")
	
	// Check if test file exists
	testFile := "/path/to/test/data/jfk.wav"
	if _, err := os.Stat(testFile); os.IsNotExist(err) {
		fmt.Printf("  ⚠️  Test file not found: %s\n", testFile)
		fmt.Println("  Skipping transcription test")
		return
	}

	// Test basic transcription interface
	fmt.Println("  📄 Testing basic transcription interface...")
	start := time.Now()
	result, err := provider.Transcript(testFile)
	duration := time.Since(start)

	if err != nil {
		fmt.Printf("  ❌ Basic transcription failed: %v\n", err)
		return
	}

	fmt.Println("  ✅ Basic transcription succeeded")
	fmt.Printf("  📝 Result: %s\n", result)
	fmt.Printf("  ⏱️  Duration: %v\n", duration)

	// Test enhanced transcription interface
	fmt.Println("\n  📄 Testing enhanced transcription interface...")
	request := &provider.TranscriptionRequest{
		InputFilePath: testFile,
		Language:      "en",
		Prompt:        "This is a speech by President Kennedy.",
	}

	ctx2, cancel2 := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel2()

	start = time.Now()
	response, err := provider.TranscriptWithOptions(ctx2, request)
	duration = time.Since(start)

	if err != nil {
		fmt.Printf("  ❌ Enhanced transcription failed: %v\n", err)
		return
	}

	fmt.Println("  ✅ Enhanced transcription succeeded")
	fmt.Printf("  📝 Text: %s\n", response.Text)
	fmt.Printf("  🌐 Language: %s\n", response.Language)
	fmt.Printf("  ⏱️  Processing Time: %v\n", response.ProcessingTime)
	fmt.Printf("  🤖 Model Used: %s\n", response.ModelUsed)
	fmt.Printf("  📊 Metadata: %+v\n", response.ProviderMetadata)

	// Test 5: Error handling
	fmt.Println("\n🚫 Test 5: Error Handling")
	
	// Test with non-existent file
	_, err = provider.Transcript("/non/existent/file.wav")
	if err != nil {
		fmt.Println("  ✅ Correctly handled non-existent file error")
		if transcriptionErr, ok := err.(*provider.TranscriptionError); ok {
			fmt.Printf("     Code: %s\n", transcriptionErr.Code)
			fmt.Printf("     Provider: %s\n", transcriptionErr.Provider)
			fmt.Printf("     Retryable: %t\n", transcriptionErr.Retryable)
		}
	} else {
		fmt.Println("  ❌ Should have failed with non-existent file")
	}

	// Test with empty input
	_, err = provider.TranscriptWithOptions(context.Background(), &provider.TranscriptionRequest{
		InputFilePath: "",
	})
	if err != nil {
		fmt.Println("  ✅ Correctly handled empty input error")
		if transcriptionErr, ok := err.(*provider.TranscriptionError); ok {
			fmt.Printf("     Code: %s\n", transcriptionErr.Code)
			fmt.Printf("     Message: %s\n", transcriptionErr.Message)
		}
	} else {
		fmt.Println("  ❌ Should have failed with empty input")
	}

	fmt.Println("\n🎉 SSH Whisper Provider Integration Test Complete!")
	fmt.Println(strings.Repeat("=", 60))
}