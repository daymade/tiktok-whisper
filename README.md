# tiktok-whisper: tiktok-whisper-video-to-text-go

##### Translate to: [简体中文](README_zh.md)

## About tiktok-whisper-video-to-text-go
Batch convert video to text using openai's whisper or the local coreML whisper.cpp

The "tiktok-whisper" tool can batch convert video to text using OpenAI's Whisper or the local coreML Whisper.cpp. It has features like exporting copy as Excel, saving conversion results to SQLite or PostgreSQL, video duration statistics, and keyword search to locate videos. The tool also provides options to use whisper_cpp + coreML for local transcription and pgvector for vectorized search(yet to be implemented).

## Features
- [x] Batch convert videos to text
- [x] Save conversion results to SQLite or PostgreSQL
- [x] Video duration statistics
- [x] Export copy as Excel
- [x] Use whisper_cpp + coreML for local transcription
- [x] batch download xiaoyuzhou podcasts with a simple url

## Quick Start

```shell
cd ./internal/app
go install github.com/google/wire
# modify binaryPath and modelPath manually
wire

cd tiktok-whisper
go build -o v2t ./cmd/v2t/main.go
./v2t help
```

windows

```cmd
cd tiktok-whisper
go build -o v2t.exe .\cmd\v2t\main.go
.\v2t.exe help
```

## Usage

### convert video/audio to text

```shell
# Convert only one file
./v2t convert -audio --input ./test/data/test.mp3

# Convert all files in directory with specified file extension
./v2t convert -audio --directory ./test/data --type m4a

# Convert all mp4 files in the specified directory to text
./v2t convert --video --directory "./test/data/mp4" --userNickname "testUser" 
```

### download audio from xiaoyuzhou or video from tiktok

```shell
# download xiaoyuzhou with single episode url
./v2t download xiaoyuzhou -e "https://www.xiaoyuzhoufm.com/episode/6398c6ae3a2b7eba5ceb462f"

# or an episode list
./v2t download xiaoyuzhou -e "https://www.xiaoyuzhoufm.com/episode/6398c6ae3a2b7eba5ceb462f,https://www.xiaoyuzhoufm.com/episode/6445559d420fc63f0b9e5747"

# download all episodes from xiaoyuzhou with podcast url
./v2t download xiaoyuzhou -p "https://www.xiaoyuzhoufm.com/podcast/61e389402454b42a2b06177c"
```

### Run faster-whisper with a Python script
There are two Python scripts for audio transcription:

- whisperToText.py: Transcribes a single audio file or audio files in a single directory.
- whisperToTextParallel.py: Transcribes audio files in multiple subdirectories in parallel.

Before running the scripts, please install the required Python packages by running the following command in the project root directory:

```shell
pip install -r requirements.txt
```

The usage is as follows:

For transcribing a single file:
```shell
python scripts/python/whisperToText.py --input_file /path/to/audiofile.mp3 --output_dir /path/to/output
```

For transcribing a single directory:
```shell
python scripts/python/whisperToText.py --input_dir /path/to/input --output_dir /path/to/output
```

For transcribing multiple subdirectories in parallel:
```shell
python scripts/python/whisperToTextParallel.py --base_input_dir /path/to/base/input --base_output_dir /path/to/base/output --processes 4
```

## TODO

- [ ] Keyword search to locate videos
- [ ] Original video jump link
- [ ] Like, share, and comment statistics
- [ ] Use pgvector for vectorized search
