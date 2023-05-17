package whisper_cpp

import (
	"bytes"
	"fmt"
	"log"
	"os/exec"
	"strings"
	"tiktok-whisper/internal/app/audio"
	"tiktok-whisper/internal/app/util/files"
)

// LocalTranscriber implements local transcription, using local binary commands.
type LocalTranscriber struct {
	binaryPath string
	modelPath  string
}

// NewLocalTranscriber creates a new instance of LocalTranscriber.
func NewLocalTranscriber(binaryPath, modelPath string) *LocalTranscriber {
	return &LocalTranscriber{
		binaryPath: binaryPath,
		modelPath:  modelPath,
	}
}

// Transcript encapsulates native binary commands, takes the MP3 file path as input and returns the transcribed text and errors (if any).
func (lt *LocalTranscriber) Transcript(inputFilePath string) (string, error) {
	log.Printf("Starting transcription of file %s\n", inputFilePath)

	// Check if the input file is a 16kHz WAV file
	is16kHzWav, err := audio.Is16kHzWavFile(inputFilePath)
	if err != nil {
		log.Printf("Error checking if input file is a 16kHz WAV file: %v\n", err)
		return "", fmt.Errorf("error checking input file: %v", err)
	}

	// Convert the input file to a 16kHz WAV file if necessary
	if !is16kHzWav {
		log.Printf("Input file is not a 16kHz WAV file, converting...\n")
		inputFilePath, err = audio.ConvertTo16kHzWav(inputFilePath)
		if err != nil {
			log.Printf("Error converting input file to a 16kHz WAV file: %v\n", err)
			return "", fmt.Errorf("error converting input file: %v", err)
		}
		log.Printf("Successfully converted input file to a 16kHz WAV file\n")
	}

	outputFile := "./1"

	args := []string{
		"-m", lt.modelPath,
		"--print-colors",
		"-l", "zh",
		"--prompt", "以下是简体中文普通话:",
		"-otxt",
		"-f", inputFilePath,
		"-of", outputFile,
	}

	command := exec.Command(lt.binaryPath, args...)
	var stdout, stderr bytes.Buffer
	command.Stdout = &stdout
	command.Stderr = &stderr

	log.Printf("Running transcription command...\n command: %s %s", lt.binaryPath, strings.Join(args, " "))

	err = command.Run()
	if err != nil {
		log.Printf("Error running transcription command: %v\n", err)
		return "", fmt.Errorf("command execution error: %v, stderr: %s", err, stderr.String())
	}

	log.Printf("Successfully ran transcription command\n")

	output, err := files.ReadOutputFile(outputFile + ".txt")
	if err != nil {
		log.Printf("Error reading output file: %v\n", err)
		return "", fmt.Errorf("failed to read output file: %v", err)
	}

	log.Printf("Successfully read output file\n")

	return output, nil
}
