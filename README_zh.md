# tiktok-whisper 抖音视频转文字

##### Translate to: [English](README.md)

## About tiktok-whisper-video-to-text-go
使用 openai 的 whisper 或者本地 coreML 的 whisper.cpp 批量将视频转换为文字

tiktok-whisper 工具可以使用OpenAI的Whisper或本地coreML的Whisper.cpp批量转换视频为文本。它的功能包括将拷贝导出为Excel，将转换结果保存为SQLite或PostgreSQL，视频时长统计，以及关键词搜索来定位视频。

## Feature

- [x] 批量转换视频为文字
- [x] 保存转换结果到 sqlite 或 postgres
- [x] 视频时长统计
- [x] 导出文案为 excel
- [x] 使用 whisper_cpp + coreML 本地转录
- [x] 输入小宇宙播客链接批量下载音频

## 快速开始

```shell
cd ./internal/app
go install github.com/google/wire
# 如果用 OpenAI 远程转换就不用这一步, 如果用本地 coreML 转换, 需要修改 binaryPath 和 modelPath
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

## 基本使用

### 将视频/音频转换为文本

```shell
# 转换音频文件
./v2t convert -audio --input ./test/data/test.mp3

# 转换指定文件扩展名的目录中的所有文件
./v2t convert -audio --directory ./test/data --type m4a

# 将指定目录中的所有 mp4 文件转换为文本
./v2t convert --video --directory "./test/data/mp4" --userNickname "testUser"
```

### 从小宇宙下载音频或从 TikTok 下载视频

```shell
# 使用单集 URL 下载小宇宙音频
./v2t download xiaoyuzhou -e "https://www.xiaoyuzhoufm.com/episode/6398c6ae3a2b7eba5ceb462f"

# 或者使用多集 URL 下载小宇宙音频
./v2t download xiaoyuzhou -e "https://www.xiaoyuzhoufm.com/episode/6398c6ae3a2b7eba5ceb462f,https://www.xiaoyuzhoufm.com/episode/6445559d420fc63f0b9e5747"

# 从小宇宙的播客 URL 下载所有集数的音频
./v2t download xiaoyuzhou -p "https://www.xiaoyuzhoufm.com/podcast/61e389402454b42a2b06177c"
```

## TODO

- [ ] 关键词搜索定位到视频
- [ ] 原始视频跳转链接
- [ ] 转赞评统计
- [ ] 使用 pgvector 向量化搜索
