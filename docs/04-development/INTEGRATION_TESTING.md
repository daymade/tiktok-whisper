# 集成测试指南

## 概述

本项目提供了完整的集成测试套件，确保所有组件正常工作。测试包括数据库验证、性能测试、Provider框架测试等。

## 测试方式

### 1. 使用 Makefile（推荐）

```bash
# 运行所有测试
make test-all

# 仅运行集成测试
make test-integration

# 运行Go集成测试
make test-integration-go

# 快速测试
./scripts/test/quick_test.sh
```

### 2. 手动运行测试脚本

```bash
# 运行完整集成测试
./scripts/test/integration_test.sh

# 禁用清理（用于调试）
./scripts/test/integration_test.sh --no-cleanup

# 启用详细输出
./scripts/test/integration_test.sh --verbose
```

### 3. 使用 Go test

```bash
# 运行带 integration 标签的测试
CGO_ENABLED=1 go test -v -tags=integration ./test/...
```

## 测试内容

### 数据库测试
- ✓ Schema验证（18个列）
- ✓ 索引验证（7个索引）
- ✓ 查询性能测试
- ✓ 新字段功能测试

### Provider框架测试
- ✓ Provider列表命令
- ✓ Provider配置检查
- ✓ 基本转换功能

### 应用功能测试
- ✓ 构建测试
- ✓ 基本命令测试
- ✓ Web接口测试
- ✓ 导出功能测试
- ✓ 嵌入系统测试

### 性能测试
- ✓ 查询响应时间 < 100ms
- ✓ 索引使用验证

## CI/CD 集成

项目包含 GitHub Actions 工作流：

```yaml
# .github/workflows/integration-tests.yml
- 多平台测试（Ubuntu, macOS）
- 多Go版本测试（1.21, 1.22）
- 自动化测试运行
- 代码覆盖率报告
```

## 测试文件结构

```
scripts/test/
├── integration_test.sh   # Shell集成测试脚本
└── quick_test.sh        # 快速验证脚本

test/
└── integration_test.go  # Go集成测试

.github/workflows/
└── integration-tests.yml # CI/CD配置
```

## 测试环境要求

- Go 1.21+
- SQLite3
- FFmpeg（可选，用于音频测试）
- CGO支持

## 测试结果示例

```
=== Quick Integration Test ===

1. Testing build...
✓ Build successful

2. Testing database...
✓ Database exists
✓ Found 18 columns
✓ Found 7 indexes

3. Testing commands...
✓ Help command works
✓ Version command works
✓ Providers command works

4. Testing query performance...
✓ Query time: 0.005s

=== All tests passed! ===
```

## 故障排除

### 构建失败
确保设置了 `CGO_ENABLED=1`：
```bash
CGO_ENABLED=1 go build -o v2t ./cmd/v2t/main.go
```

### 数据库测试失败
检查数据库文件是否存在：
```bash
ls -la ./data/transcription.db
```

### Provider测试失败
检查配置文件：
```bash
cat ~/.tiktok-whisper/providers.yaml
```

## 扩展测试

要添加新的测试：

1. **Shell测试**：编辑 `scripts/test/integration_test.sh`
2. **Go测试**：在 `test/integration_test.go` 添加新的测试函数
3. **CI测试**：更新 `.github/workflows/integration-tests.yml`

## 最佳实践

1. 每次提交前运行 `make test-all`
2. 重大更改后运行完整集成测试
3. 保持测试独立性，避免相互依赖
4. 使用有意义的测试名称和错误信息
5. 定期检查CI/CD测试结果