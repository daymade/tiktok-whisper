# tiktok-whisper 抖音视频转文字

##### Translate to: [English](README.md)

## About tiktok-whisper-video-to-text-go
使用 openai 的 whisper 或者本地 coreML 的 whisper.cpp 批量将视频转换为文字

## Feature

- [x] 批量转换视频为文字
- [x] 保存转换结果到 sqlite 或 postgres
- [x] 视频时长统计

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
- [x] 导出文案为 excel
- [x] 使用 whisper_cpp + coreML 本地转录
- [ ] 关键词搜索定位到视频
- [ ] 原始视频跳转链接
- [ ] 转赞评统计
- [ ] 使用 pgvector 向量化搜索
