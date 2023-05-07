# tiktok-whisper: tiktok-whisper-video-to-text-go

##### Translate to: [简体中文](README_zh.md)

## About tiktok-whisper-video-to-text-go
Batch convert video to text using openai's whisper or the local coreML whisper.cpp

## Features
- [x] Batch convert videos to text
- [x] Save conversion results to SQLite or PostgreSQL
- [x] Video duration statistics

## Usage

```shell
cd ./internal/app
go install github.com/google/wire
# modify binaryPath and modelPath manually
wire

cd tiktok-whisper
go build -o v2t ./cmd/v2t/main.go
./v2t help
```

## TODO
- [x] Export copy as Excel
- [x] Use whisper_cpp + coreML for local transcription
- [ ] Keyword search to locate videos
- [ ] Original video jump link
- [ ] Like, share, and comment statistics
- [ ] Use pgvector for vectorized search
