# Tiktok-Whisper 数据导出到 Markdown 工具

这个工具自动化了从 tiktok-whisper SQLite 数据库导出转录数据并转换为 Markdown 文件的完整流程。

## 🚀 快速开始

### 最简单的使用方式

```bash
# 1. 进入工具目录
cd tools/export-to-md

# 2. 初始化 uv 环境
uv sync

# 3. 查看所有用户
uv run python export_to_md.py list-users

# 4. 导出指定用户的数据
uv run uv run python export_to_md.py export --user "经纬第二期"
```

就是这么简单！工具会自动：
- 从数据库查询数据
- 导出为 JSON 格式
- 调用 html2md 工具转换为 Markdown
- 生成 ZIP 文件包含所有 Markdown 文件
- 清理临时文件

## 📚 文档导航

### 🚀 新手必读
- **[⚡ 快速开始](快速开始.md)** - 3分钟从安装到成功导出
- **[📚 小白操作指南](小白操作指南.md)** - 超详细的零基础教程
- **[🛠️ 故障排除指南](故障排除指南.md)** - 遇到问题？这里有答案

### 📖 详细文档
- [安装配置](#安装配置)
- [命令参考](#命令参考)
- [使用案例](#使用案例)
- [配置文件](#配置文件)
- [故障排除](#故障排除)
- [技术细节](#技术细节)

## 🔧 安装配置

### 前置要求

1. **uv** (Python 包管理器)
2. **html2md 工具** (应该已经存在)
3. **SQLite 数据库** (tiktok-whisper 的转录数据)

**安装 uv (如果未安装)**：
```bash
curl -LsSf https://astral.sh/uv/install.sh | sh
```

### 初次设置

1. **进入工具目录**：
   ```bash
   cd tools/export-to-md
   ```

2. **初始化 uv 环境**：
   ```bash
   uv sync
   ```

3. **检查配置文件**：
   ```bash
   uv run uv run python export_to_md.py config --show
   ```

4. **如果路径不正确，更新配置**：
   ```bash
   # 更新数据库路径
   uv run uv run python export_to_md.py config --set database_path="../../data/transcription.db"
   
   # 更新 html2md 工具路径
   uv run uv run python export_to_md.py config --set html2md_path="/path/to/html2md/main.py"
   ```

5. **测试工具**：
   ```bash
   uv run python export_to_md.py list-users
   ```

## 📖 命令参考

### `list-users` - 列出所有用户

```bash
uv run python export_to_md.py list-users
```

**输出示例**：
```
==================================================
用户列表 (按记录数排序)
==================================================
 1. 亮哥留学                        (1153 条记录)
 2. 经纬第二期                      ( 265 条记录)
 3. 经纬2024                       ( 271 条记录)
 4. 许朝军                         ( 261 条记录)
 ...

总计: 20 个用户, 3850 条记录
```

### `export` - 导出指定用户

```bash
uv run python export_to_md.py export --user "用户名" [选项]
```

**选项**：
- `--user "用户名"` (必需) - 要导出的用户名
- `--output "目录"` (可选) - 输出目录，默认 `./output`
- `--limit 数量` (可选) - 限制导出的记录数

**示例**：
```bash
# 基本导出
uv run python export_to_md.py export --user "经纬第二期"

# 导出到指定目录
uv run python export_to_md.py export --user "经纬第二期" --output "/path/to/output"

# 只导出最新的 100 条记录
uv run python export_to_md.py export --user "经纬第二期" --limit 100
```

### `export-all` - 导出所有用户

```bash
uv run uv run python export_to_md.py export-all [--output "目录"]
```

这个命令会：
- 为每个用户创建独立的子目录
- 并行处理提高效率
- 生成详细的进度报告

**示例**：
```bash
# 导出所有用户到默认目录
uv run uv run python export_to_md.py export-all

# 导出到指定目录
uv run uv run python export_to_md.py export-all --output "/path/to/exports"
```

### `config` - 配置管理

```bash
# 显示当前配置
uv run python export_to_md.py config --show

# 设置配置项
uv run python export_to_md.py config --set key=value
```

**配置项说明**：
- `database_path` - SQLite 数据库路径
- `html2md_path` - html2md 工具路径
- `default_output_dir` - 默认输出目录
- `keep_json` - 是否保留 JSON 文件 (true/false)
- `keep_md_files` - 是否保留独立的 MD 文件 (true/false)

## 💡 使用案例

### 案例 1: 导出单个用户的最新内容

```bash
# 只要最新的 50 条记录，快速预览
uv run python export_to_md.py export --user "经纬第二期" --limit 50 --output "./preview"
```

### 案例 2: 为所有用户创建备份

```bash
# 导出所有用户到备份目录
uv run uv run python export_to_md.py export-all --output "./backup_$(date +%Y%m%d)"
```

### 案例 3: 定制化配置

```bash
# 保留中间文件用于调试
uv run python export_to_md.py config --set keep_json=true
uv run python export_to_md.py config --set keep_md_files=true

# 导出数据
uv run python export_to_md.py export --user "经纬第二期"
```

### 案例 4: 批处理脚本

创建 `export_favorite_users.sh`：
```bash
#!/bin/bash
users=("经纬第二期" "亮哥留学" "许朝军")

for user in "${users[@]}"; do
    echo "导出用户: $user"
    uv run python export_to_md.py export --user "$user" --output "./favorites/$user"
done
```

## ⚙️ 配置文件

配置文件 `config.json` 的完整结构：

```json
{
  "database_path": "../../data/transcription.db",
  "html2md_path": "/path/to/html2md/main.py",
  "default_output_dir": "./output",
  "keep_json": false,
  "keep_md_files": false,
  "batch_size": 50,
  "max_records": null,
  "date_format": "%Y-%m-%d %H:%M:%S"
}
```

**配置项详解**：

| 配置项 | 说明 | 默认值 |
|--------|------|--------|
| `database_path` | SQLite 数据库文件路径 | `../../data/transcription.db` |
| `html2md_path` | html2md 工具的 main.py 路径 | 需要配置 |
| `default_output_dir` | 默认输出目录 | `./output` |
| `keep_json` | 是否保留导出的 JSON 文件 | `false` |
| `keep_md_files` | 是否保留独立的 MD 文件 | `false` |
| `batch_size` | 每个 MD 文件包含的记录数 | `50` |
| `max_records` | 最大导出记录数限制 | `null` (无限制) |
| `date_format` | 日期格式 | `%Y-%m-%d %H:%M:%S` |

## 🔍 故障排除

### 常见问题

**1. 数据库文件不存在**
```
错误: 数据库文件不存在: ../../data/transcription.db
```
**解决**：检查数据库路径是否正确
```bash
uv run python export_to_md.py config --set database_path="/正确的/路径/transcription.db"
```

**2. html2md 工具不存在**
```
错误: html2md 工具不存在: /path/to/html2md/main.py
```
**解决**：更新 html2md 工具路径
```bash
uv run python export_to_md.py config --set html2md_path="/正确的/路径/html2md/main.py"
```

**3. 用户不存在**
```
错误: 用户不存在: 用户名
```
**解决**：先列出所有用户，确认用户名的准确拼写
```bash
uv run python export_to_md.py list-users
```

**4. 权限问题**
```
错误: Permission denied
```
**解决**：检查输出目录的写权限，或使用具有写权限的目录

**5. Python 版本问题**
```
错误: SyntaxError
```
**解决**：确保使用 Python 3.6 或更高版本
```bash
uv run python export_to_md.py list-users
```

### 调试模式

如果遇到问题，可以开启调试模式：

1. **保留中间文件**：
   ```bash
   uv run python export_to_md.py config --set keep_json=true
   uv run python export_to_md.py config --set keep_md_files=true
   ```

2. **检查生成的 JSON**：
   导出后检查 JSON 文件格式是否正确

3. **手动测试 html2md**：
   ```bash
   python /path/to/html2md/main.py test.json
   ```

## 🔧 技术细节

### 数据流程

```
SQLite 数据库 
    ↓ (SQL 查询)
JSON 文件 
    ↓ (html2md 工具)
Markdown 文件 (批次处理)
    ↓ (打包)
ZIP 文件
```

### 查询逻辑

工具使用以下 SQL 查询导出数据：

```sql
SELECT mp3_file_name, transcription 
FROM transcriptions 
WHERE has_error = 0 
  AND transcription IS NOT NULL 
  AND transcription != '' 
  AND user = ? 
ORDER BY last_conversion_time DESC
```

### 文件命名规则

- **输出 ZIP 文件**: `{用户名}_transcriptions.zip`
- **MD 文件**: `{用户名}_{批次号}.md` (批次内部)
- **临时文件**: 自动生成和清理

### 性能考虑

- **内存使用**: 数据分批处理，避免大量数据同时加载到内存
- **并发处理**: export-all 命令支持并行导出多个用户
- **磁盘空间**: 自动清理临时文件，可配置保留策略

## 🚀 高级功能

### 集成到现有工作流

可以将此工具集成到 tiktok-whisper 的主命令中：

```bash
# 将来可能的集成方式
./v2t export-md --user "经纬第二期"
```

### 自动化任务

使用 cron 任务定期备份：
```bash
# 每天凌晨 2 点备份所有数据
0 2 * * * cd /path/to/tiktok-whisper/tools/export-to-md && uv run uv run python export_to_md.py export-all --output "/backup/$(date +\%Y\%m\%d)"
```

### 扩展功能

工具设计考虑了扩展性，可以轻松添加：
- 日期范围过滤
- 关键词搜索
- 自定义输出格式
- 云存储上传
- 数据分析报告

---

## 📄 许可证

本工具作为 tiktok-whisper 项目的一部分，使用相同的许可证。

## 🤝 贡献

欢迎提交 Issue 和 Pull Request 来改进这个工具！