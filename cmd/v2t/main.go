package main

import (
	"fmt"
	"os"

	"tiktok-whisper/cmd/v2t/cmd"
	"tiktok-whisper/internal/config"
	
	// Import providers to register them
	_ "tiktok-whisper/internal/app/api/whisper_cpp"
	_ "tiktok-whisper/internal/app/api/openai/whisper"
	_ "tiktok-whisper/internal/app/api/elevenlabs"
	_ "tiktok-whisper/internal/app/api/ssh_whisper"
	_ "tiktok-whisper/internal/app/api/whisper_server"
	_ "tiktok-whisper/internal/app/api/custom_http"
)

func main() {
	// Initialize configuration (non-blocking - only warns about missing keys)
	apiKeys, err := config.InitializeConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "⚠️  Configuration Warning: %v\n", err)
		fmt.Fprintf(os.Stderr, "💡 To enable embedding features, copy .env.example to .env and add your API keys\n")
		// Continue execution - don't exit
	} else {
		// Store API keys globally for the application
		// This allows other parts of the application to access them
		if apiKeys.OpenAI != "" {
			os.Setenv("OPENAI_API_KEY", apiKeys.OpenAI)
		}
		if apiKeys.Gemini != "" {
			os.Setenv("GEMINI_API_KEY", apiKeys.Gemini)
		}
	}

	// Execute the CLI command
	cmd.Execute()
}
