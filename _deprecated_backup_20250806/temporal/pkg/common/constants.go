package common

const (
	// Temporal constants
	DefaultTaskQueue = "v2t-transcription-queue"
	DefaultNamespace = "default"
	
	// Default service addresses
	DefaultTemporalHost = "127.0.0.1:7233"
	DefaultMinIOEndpoint = "localhost:9000"
	
	// Default credentials
	DefaultMinIOAccessKey = "minioadmin"
	DefaultMinIOSecretKey = "minioadmin"
	DefaultMinIOBucket = "v2t-transcriptions"
	
	// Whisper defaults
	DefaultWhisperBinary = "/Volumes/SSD2T/workspace/cpp/whisper.cpp-updated/build/bin/whisper-cli"
	DefaultWhisperModel = "/Volumes/SSD2T/workspace/cpp/whisper.cpp-updated/models/ggml-base.en.bin"
)