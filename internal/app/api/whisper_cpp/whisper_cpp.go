package whisper_cpp

import (
	"bytes"
	"fmt"
	"os/exec"
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
	// Check if the input file is a 16kHz WAV file
	is16kHzWav, err := audio.Is16kHzWavFile(inputFilePath)
	if err != nil {
		return "", fmt.Errorf("error checking input file: %v", err)
	}

	// Convert the input file to a 16kHz WAV file if necessary
	if !is16kHzWav {
		inputFilePath, err = audio.ConvertTo16kHzWav(inputFilePath)
		if err != nil {
			return "", fmt.Errorf("error converting input file: %v", err)
		}
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

	cmd := exec.Command(lt.binaryPath, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err = cmd.Run()
	if err != nil {
		return "", fmt.Errorf("command execution error: %v, stderr: %s", err, stderr.String())
	}

	output, err := files.ReadOutputFile(outputFile + ".txt")
	if err != nil {
		return "", fmt.Errorf("failed to read output file: %v", err)
	}

	return output, nil
}
