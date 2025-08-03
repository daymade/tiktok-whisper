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
	fmt.Println("🚀 Testing SSH Whisper Provider")
	fmt.Println(strings.Repeat("=", 50))

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
	
	fmt.Println("\n📋 Provider Information")
	info := provider.GetProviderInfo()
	fmt.Printf("  Name: %s\n", info.Name)
	fmt.Printf("  Display Name: %s\n", info.DisplayName)
	fmt.Printf("  Type: %s\n", info.Type)

	fmt.Println("\n🔧 Configuration Validation")
	if err := provider.ValidateConfiguration(); err != nil {
		fmt.Printf("  ❌ Validation failed: %v\n", err)
		if strings.Contains(err.Error(), "SSH connection test failed") {
			fmt.Println("  ℹ️  This is expected if SSH keys are not configured")
		}
		return
	}
	fmt.Println("  ✅ Configuration valid")

	fmt.Println("\n🏥 Health Check")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := provider.HealthCheck(ctx); err != nil {
		fmt.Printf("  ❌ Health check failed: %v\n", err)
		return
	}
	fmt.Println("  ✅ Health check passed")

	fmt.Println("\n🎤 Transcription Test")
	testFile := "/Volumes/SSD2T/workspace/go/tiktok-whisper/test/data/jfk.wav"
	if _, err := os.Stat(testFile); os.IsNotExist(err) {
		fmt.Printf("  ⚠️  Test file not found: %s\n", testFile)
		fmt.Println("  Creating a dummy test to verify connectivity...")
		
		// Just test the basic interface without a real file
		fmt.Println("  📄 Testing basic error handling...")
		_, err := provider.Transcript("/non/existent/file.wav")
		if err != nil {
			fmt.Printf("  ✅ Correctly handled error: %v\n", err)
		}
		return
	}

	fmt.Printf("  📁 Using test file: %s\n", testFile)
	start := time.Now()
	result, err := provider.Transcript(testFile)
	duration := time.Since(start)

	if err != nil {
		fmt.Printf("  ❌ Transcription failed: %v\n", err)
		return
	}

	fmt.Println("  ✅ Transcription successful!")
	fmt.Printf("  📝 Result: %s\n", result)
	fmt.Printf("  ⏱️  Duration: %v\n", duration)

	fmt.Println("\n🎉 SSH Whisper Provider Test Complete!")
	fmt.Println(strings.Repeat("=", 50))
}