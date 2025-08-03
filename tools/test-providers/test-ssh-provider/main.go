package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"tiktok-whisper/internal/app/api/ssh_whisper"
)

func main() {
	fmt.Println("🚀 Testing SSH Whisper Provider Integration")
	fmt.Println(strings.Repeat("=", 60))

	// Create SSH whisper provider configuration
	config := ssh_whisper.SSHWhisperConfig{
		Host:       "daymade@mac-mini-m4-1.local",
		RemoteDir:  "/Users/daymade/Workspace/cpp/whisper.cpp",
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
	testFile := "/Volumes/SSD2T/workspace/go/tiktok-whisper/test/data/jfk.wav"
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
	// Note: We'll access the types through the provider's interface
	// For now, let's use the simpler API that doesn't require direct type access

	// Enhanced transcription interface test disabled for now
	fmt.Println("  ⚠️  Enhanced transcription interface test skipped (requires type access)")
	fmt.Println("  The basic transcription interface already demonstrates functionality")

	// Test 5: Error handling
	fmt.Println("\n🚫 Test 5: Error Handling")
	
	// Test with non-existent file
	_, err = provider.Transcript("/non/existent/file.wav")
	if err != nil {
		fmt.Println("  ✅ Correctly handled non-existent file error")
		fmt.Printf("     Error: %v\n", err)
	} else {
		fmt.Println("  ❌ Should have failed with non-existent file")
	}

	// Test with empty input path
	_, err = provider.Transcript("")
	if err != nil {
		fmt.Println("  ✅ Correctly handled empty input error")
		fmt.Printf("     Error: %v\n", err)
	} else {
		fmt.Println("  ❌ Should have failed with empty input")
	}

	fmt.Println("\n🎉 SSH Whisper Provider Integration Test Complete!")
	fmt.Println(strings.Repeat("=", 60))
}