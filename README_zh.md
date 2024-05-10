# tiktok-whisper 抖音视频转文字

##### Translate to: [English](README.md)

## About tiktok-whisper-video-to-text-go

![demo_download_xiaoyuzhou](doc/demo/download_xiaoyuzhou.gif)

使用 openai 的 whisper 或者本地 coreML 的 whisper.cpp 批量将视频转换为文字

tiktok-whisper 工具可以使用 OpenAI 云端的 Whisper API 或本地 coreML 的 Whisper.cpp 批量转换视频为文本。它的功能包括将拷贝导出为 Excel，将转换结果保存为 SQLite 或 PostgreSQL，视频时长统计，以及关键词搜索来定位视频。

原始的 whisper 有两个缺点: 
1. 必须要求 CUDA 独显, 不能在 macOS 上运行
2. 特别慢

解决方法: 

1. 用 macOS 上的 coreML 调用 whisper.cpp, 比独显慢不了多少
2. 换 fast-whisper, 下文有使用说明

## Feature

- [x] 输入小宇宙播客链接, 批量下载音频
- [x] 批量识别音频或视频, 输出带时间轴的文字
- [x] 保存识别结果到 sqlite 或 postgres
- [x] 在 macOS 上使用 whisper_cpp + coreML 本地转录
- [x] 导出历史识别结果

## 快速开始

### macOS

用 macOS 本地的 coreML 转换, 需要修改 binaryPath 和 modelPath, 如果你有 API KEY, 可以用 OpenAI 远程 API 转换, 请直接跳到第3步开始编译.

0. 生成 coreML 的 model
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

1. 修改
```go
# 修改 binaryPath 和 modelPath
func provideNewLocalTranscriber() api.Transcriber {
	binaryPath := "~/workspace/cpp/whisper.cpp/main"
	modelPath := "~/workspace/cpp/whisper.cpp/models/ggml-large-v2.bin"
	return whisper_cpp.NewLocalTranscriber(binaryPath, modelPath)
}
```

2. 生成 wire 配置
```shell
cd ./internal/app
go install github.com/google/wire/cmd/wire@latest
wire
```

3. 编译出可执行程序
```shell
cd tiktok-whisper
go build -o v2t ./cmd/v2t/main.go
./v2t help
```

### Windows

操作基本和 macOS 相同

```cmd
cd tiktok-whisper
go build -o v2t.exe .\cmd\v2t\main.go
.\v2t.exe help
```

## 基本使用

### 从小宇宙下载音频或从 TikTok 下载视频

```shell
# 使用单集 URL 下载小宇宙音频
./v2t download xiaoyuzhou -e "https://www.xiaoyuzhoufm.com/episode/6398c6ae3a2b7eba5ceb462f"

# 或者使用多集 URL 下载小宇宙音频
./v2t download xiaoyuzhou -e "https://www.xiaoyuzhoufm.com/episode/6398c6ae3a2b7eba5ceb462f,https://www.xiaoyuzhoufm.com/episode/6445559d420fc63f0b9e5747"

# 从小宇宙的播客 URL 下载所有集数的音频
./v2t download xiaoyuzhou -p "https://www.xiaoyuzhoufm.com/podcast/61e389402454b42a2b06177c"
```

下载完成后可以在 data 目录看到文件
```shell
$ tree data/
data/
└── xiaoyuzhou
    └── 硬地骇客
        └── EP21 程序员的职场晋升究竟与什么有关？漂亮的代码？.mp3
```

### 使用 yt-dlp 下载 youtube 视频

只下载音频就够了, 不需要视频, 请使用以下命令
```shell
yt-dlp --extract-audio --audio-format mp3 "https://www.youtube.com/watch?v=tWmNN87VvcE"
```

### 将视频/音频转换为文本

在 macOS 上可以使用 whisper.cpp 来转换音频, 请确保你已经正确设置了 whisper.cpp 的路径(wire.go 中的 binaryPath 和 modelPath)

```shell
# 转换音频文件
./v2t convert -audio --input ./test/data/test.mp3

# 转换指定文件扩展名的目录中的所有文件
./v2t convert -audio --directory ./test/data --type m4a

# 将指定目录中的所有 mp4 文件转换为文本, -n 指定最多转换多少个，默认 n=1
./v2t convert --video --directory "./test/data/mp4" --userNickname "testUser" -n 100

# 将指定用户的识别历史全部导出为 excel
./v2t export --userNickname "testUser" --outputFilePath ./data/testUser.xlsx
```

使用 OpenAI 的 API KEY 来转换音频, 请确保你已经正确设置了环境变量 `OPENAI_API_KEY`, 修改 wire.go 使用 provideRemoteTranscriber
```diff
func InitializeConverter() *converter.Converter {
-   wire.Build(converter.NewConverter, provideLocalTranscriber, provideTranscriptionDAO)
+   wire.Build(converter.NewConverter, provideRemoteTranscriber, provideTranscriptionDAO)
	return &converter.Converter{}
}
```


### 使用 Python 脚本运行 faster-whisper

假如你是在 Windows 平台并且有独立显卡, 那么你可以使用 python 的 faster-whisper 来调用 CUDA 处理, 有两个 Python 脚本用于批量音频转录：

- whisperToText.py: 转录单个文件或单个目录中的音频文件。
- whisperToTextParallel.py: 并行转录多个子目录中的音频文件。

在运行脚本之前，请在项目根目录运行以下命令安装所需的 Python 包：

```shell
pip install -r requirements.txt
```

使用方法如下：

对单个文件进行转录：
```shell
python scripts/python/whisperToText.py --input_file /path/to/audiofile.mp3 --output_dir /path/to/output
```

对单个目录进行转录：
```shell
python scripts/python/whisperToText.py --input_dir /path/to/input --output_dir /path/to/output
```

对多个子目录进行并行转录：
```shell
python scripts/python/whisperToTextParallel.py --base_input_dir /path/to/base/input --base_output_dir /path/to/base/output --processes 4
```

例如:
```shell
python scripts/python/whisperToText.py --input_dir ./data/xiaoyuzhou/硬地骇客/ --output_dir ./data/output
python scripts/python/whisperToText.py --input_dir ./data/youtube/ --output_dir ./data/output
```


## TODO

- [x] 视频时长统计
- [ ] 关键词搜索定位到视频
- [ ] 原始视频跳转链接
- [ ] 转赞评统计
- [ ] 使用 pgvector 向量化搜索
