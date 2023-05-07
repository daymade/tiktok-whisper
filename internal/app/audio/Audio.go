package audio

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	model2 "tiktok-whisper/internal/app/model"
)

func GetAudioDuration(filePath string) (int, error) {
	cmd := exec.Command("ffprobe", "-v", "error", "-show_entries", "format=duration", "-of", "default=noprint_wrappers=1:nokey=1", filePath)
	output, err := cmd.Output()
	if err != nil {
		return 0, err
	}
	durationFloat, err := strconv.ParseFloat(strings.TrimSpace(string(output)), 64)
	if err != nil {
		return 0, err
	}
	duration := int(math.Round(durationFloat))
	return duration, nil
}

func ConvertToMp3(file model2.FileInfo, mp3FilePath string) error {
	if _, err := os.Stat(mp3FilePath); os.IsNotExist(err) {
		fmt.Printf("Processing file: %s\n", file.Name)

		// Convert MP4 to MP3
		cmd := exec.Command("ffmpeg", "-i", file.FullPath, "-vn", "-acodec", "libmp3lame", mp3FilePath)
		err = cmd.Run()
		if err != nil {
			return fmt.Errorf("FFmpeg error: %v\n", err)
		}
		fmt.Printf("MP4 to MP3 conversion completed: '%s'\n", mp3FilePath)
	} else {
		fmt.Printf("MP3 file already exists for '%s', skipping conversion.\n", file.Name)
	}
	return nil
}

func Is16kHzWavFile(filePath string) (bool, error) {
	cmd := exec.Command("ffprobe", "-v", "quiet", "-print_format", "json", "-show_streams", filePath)
	output, err := cmd.Output()
	if err != nil {
		return false, err
	}

	var probeOutput model2.FFProbeOutput
	err = json.Unmarshal(output, &probeOutput)
	if err != nil {
		return false, err
	}

	for _, stream := range probeOutput.Streams {
		if stream.CodecType == "audio" && stream.CodecName == "pcm_s16le" && stream.SampleRate == 16000 {
			return true, nil
		}
	}

	return false, nil
}

func ConvertTo16kHzWav(inputFilePath string) (string, error) {
	outputFilePath := strings.TrimSuffix(inputFilePath, filepath.Ext(inputFilePath)) + "_16khz.wav"
	err := convertTo16kHzWav(inputFilePath, outputFilePath)
	if err != nil {
		return "", err
	}

	return outputFilePath, nil
}

func convertTo16kHzWav(inputMp3Path, outputWavPath string) error {
	if _, err := os.Stat(outputWavPath); os.IsNotExist(err) {
		fmt.Printf("Processing file: %s\n", inputMp3Path)

		// Convert MP3 to 16kHz WAV
		cmd := exec.Command("ffmpeg", "-i", inputMp3Path, "-vn", "-acodec", "pcm_s16le", "-ar", "16000", "-ac", "2", outputWavPath)
		err = cmd.Run()
		if err != nil {
			return fmt.Errorf("FFmpeg error: %v\n", err)
		}
		fmt.Printf("MP3 to 16kHz WAV conversion completed: '%s'\n", outputWavPath)
	} else {
		fmt.Printf("16kHz WAV file already exists for '%s', skipping conversion.\n", inputMp3Path)
	}
	return nil
}
