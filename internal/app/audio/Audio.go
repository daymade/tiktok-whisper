package audio

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
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

func ConvertToMp3(fileName string, fileFullPath string, mp3FilePath string) error {
	if _, err := os.Stat(mp3FilePath); os.IsNotExist(err) {
		log.Printf("converting to mp3: %s\n", fileName)

		// Convert MP4 to MP3
		cmd := exec.Command("ffmpeg", "-i", fileFullPath, "-vn", "-acodec", "libmp3lame", mp3FilePath)

		// 创建一个 buffer 来捕获标准错误输出
		var stderr bytes.Buffer
		cmd.Stderr = &stderr

		err := cmd.Run()
		if err != nil {
			// 输出标准错误输出的内容，以便了解详细的错误原因
			return fmt.Errorf("FFmpeg error: %v, stderr: %s", err, stderr.String())
		}

		log.Printf("MP4 to MP3 conversion completed: '%s'\n", mp3FilePath)
	} else {
		log.Printf("MP3 file already exists for '%s', skipping conversion.\n", fileName)
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

func convertTo16kHzWav(inputAudioFilePath, outputWavPath string) error {
	if _, err := os.Stat(outputWavPath); !os.IsNotExist(err) {
		log.Printf("16kHz WAV file already exists for '%s', skipping conversion.\n", inputAudioFilePath)
		return nil
	}

	ext := strings.ToLower(filepath.Ext(inputAudioFilePath))
	if ext != ".mp3" && ext != ".m4a" && ext != ".wav" {
		return fmt.Errorf("unsupported audio format not in [mp3,m4a,wav]: %s", ext)
	}

	log.Printf("convert to 16kHz wav: %s\n", inputAudioFilePath)

	// Convert audio to 16kHz WAV
	cmd := exec.Command("ffmpeg", "-i", inputAudioFilePath, "-vn", "-acodec", "pcm_s16le", "-ar", "16000", "-ac", "2", outputWavPath)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("FFmpeg error: %v", err)
	}

	log.Printf("Audio to 16kHz WAV conversion completed: '%s'\n", outputWavPath)
	return nil
}
