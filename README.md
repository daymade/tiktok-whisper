# tiktok-whisper: tiktok-whisper-video-to-text-go

##### Translate to: [简体中文](README_zh.md)

## About tiktok-whisper-video-to-text-go

![demo_download_xiaoyuzhou](doc/demo/download_xiaoyuzhou.gif)

Batch convert videos to text using OpenAI's Whisper or the local coreML whisper.cpp.

The tiktok-whisper tool allows batch conversion of videos to text using either OpenAI's cloud-based Whisper API or local coreML's Whisper.cpp. It includes features such as exporting copies to Excel, saving conversion results to SQLite or PostgreSQL, video duration statistics, and keyword search to locate videos. It addresses the original whisper's limitations by offering solutions for macOS compatibility and speed enhancement.

## Features
- [x] Input Xiaoyuzhou podcast links for batch audio downloading
- [x] Batch recognize audio or video, outputting text with timestamps
- [x] Save recognition results to SQLite or PostgreSQL
- [x] Use whisper_cpp + coreML for local transcription on macOS
- [x] Export historical recognition results

## Quick Start

### macOS

Tiktok-whipser is based on two whisper engines: local whisper_cpp and remote openai whisper API. 

For local conversion using coreML on macOS, you need to modify `binaryPath` and `modelPath` direct to your local whisper_cpp. 

If you have an API KEY, you can use OpenAI's cloud API for conversion; skip step 1,2,3 to step 4 for compilation.

1. Generate coreML's model:
```shell
mkdir -p ~/workspace/cpp/ && cd ~/workspace/cpp/
git clone git@github.com:ggerganov/whisper.cpp.git
cd whisper.cpp
bash ./models/download-ggml-model.sh large
conda create -n whisper-cpp python=3.10 -y
conda activate whisper-cpp 
pip install -U ane_transformers openai-whisper coremltools
bash ./models/generate-coreml-model.sh large
make clean
WHISPER_COREML=1 make -j
```

2. for using local whisper_cpp, you should modify the binaryPath and modelPath in `tiktok-whisper/internal/app/wire.go` manually.
```go
func provideLocalTranscriber() api.Transcriber {
    // Modify binaryPath and modelPath to your paths here!
    binaryPath := "~/workspace/cpp/whisper.cpp/main"
    modelPath := "~/workspace/cpp/whisper.cpp/models/ggml-large-v2.bin"
    return whisper_cpp.NewLocalTranscriber(binaryPath, modelPath)
}
```

3. Generate wire configuration and compile the executable:
```shell
cd ./internal/app
go install github.com/google/wire/cmd/wire@latest
wire
```

4. Compile tiktok-whisper with CGO_ENABLED
```shell
cd tiktok-whisper
CGO_ENABLED=1 go build -o v2t ./cmd/v2t/main.go
./v2t help
```

### Windows

The procedure is similar to macOS.

```cmd
cd tiktok-whisper
go build -o v2t.exe .\cmd\v2t\main.go
.\v2t.exe help
```

## Usage

### Download audio from Xiaoyuzhou or video from TikTok

```shell
# Download Xiaoyuzhou audio using a single episode URL
./v2t download xiaoyuzhou -e "https://www.xiaoyuzhoufm.com/episode/6398c6ae3a2b7eba5ceb462f"

# Or using multiple episode URLs
./v2t download xiaoyuzhou -e "https://www.xiaoyuzhoufm.com/episode/6398c6ae3a2b7eba5ceb462f,https://www.xiaoyuzhoufm.com/episode/6445559d420fc63f0b9e5747"

# Download all episodes from a Xiaoyuzhou podcast URL
./v2t download xiaoyuzhou -p "https://www.xiaoyuzhoufm.com/podcast/61e389402454b42a2b06177c"
```

After downloading, you can find the files in the data directory:
```shell
$ tree data/
data/
└── xiaoyuzhou
    └── 硬地骇客
        └── EP21 程序员的职场晋升究竟与什么有关？漂亮的代码？.mp3
```

### Use yt-dlp to download YouTube videos

To download only audio without video, use the following command:
```shell
yt-dlp --extract-audio --audio-format mp3 "https://www.youtube.com/watch?v=tWmNN87VvcE"
```

### Convert videos/audios to text

On macOS, you can use whisper.cpp for audio conversion, ensuring the correct setup of `binaryPath` and `modelPath` in `wire.go`:
```shell
# Convert an

 audio file
./v2t convert -audio --input ./test/data/test.mp3

# Convert all files in a directory with a specified file extension
./v2t convert -audio --directory ./test/data --type m4a

# Convert all mp4 files in a specified directory to text, -n specifies the maximum number of files to convert, default n=1
./v2t convert --video --directory "./test/data/mp4" --userNickname "testUser" -n 100

# Export all recognition history of a specified user as excel
./v2t export --userNickname "testUser" --outputFilePath ./data/testUser.xlsx
```

To use OpenAI's API KEY for audio conversion, ensure `OPENAI_API_KEY` is set correctly in your environment variables and modify `wire.go` to use `provideRemoteTranscriber`:
```diff
func InitializeConverter() *converter.Converter {
-   wire.Build(converter.NewConverter, provideLocalTranscriber, provideTranscriptionDAO)
+   wire.Build(converter.NewConverter, provideRemoteTranscriber, provideTranscriptionDAO)
    return &converter.Converter{}
}
```

### Using Python scripts for faster-whisper

If you are on Windows and have a dedicated GPU, you can use Python's faster-whisper for CUDA processing. There are two Python scripts for batch audio transcription:

- `whisperToText.py`: Transcribes a single file or all files in a single directory.
- `whisperToTextParallel.py`: Transcribes files in multiple subdirectories in parallel.

Before running the scripts, install the required Python packages:
```shell
pip install -r requirements.txt
```

For single file or directory transcription, and parallel transcription of multiple subdirectories, follow the provided commands in the documentation.

## Recent Features ✨

- [x] **Dual Embedding System**: OpenAI (1536D) + Gemini (768D) embedding support
- [x] **pgvector Integration**: Vector similarity search with PostgreSQL
- [x] **User-Specific Processing**: Generate embeddings for specific users with targeted batch processing
- [x] **3D Visualization**: Interactive 3D clustering visualization with Three.js
- [x] **Natural Trackpad Gestures**: Jon Ive-level touch interaction system
- [x] **Real-time Search**: Vector-based similarity search with live results
- [x] **Batch Embedding Generation**: CLI tools for large-scale embedding processing

## Embedding & Vector Search

Generate embeddings and perform similarity search:

```shell
# Generate embeddings for all transcriptions
./v2t embed generate

# Generate embeddings for specific user
./v2t embed generate --user "username" --provider gemini

# Check embedding status and user distribution
./v2t embed status

# Search for similar content
./v2t embed search --text "your search query" --limit 10

# Calculate similarity between transcriptions
./v2t embed similarity --id1 123 --id2 456

# Find potential duplicates for specific user
./v2t embed duplicates --user "username" --threshold 0.95

# Start 3D visualization server
go run web-main.go
# Visit http://localhost:8080 for interactive clustering visualization
```

## TODO

- [x] Video duration statistics  
- [x] Use pgvector for vectorized search
- [x] 3D visualization with clustering
- [x] Natural trackpad gesture support
- [ ] Keyword search to locate videos
- [ ] Original video jump link
- [ ] Like, share, and comment statistics