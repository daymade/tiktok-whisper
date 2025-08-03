#!/bin/bash
# 主迁移脚本 - 适配本地环境

set -e

# 设置实际的数据库路径
export DB_PATH="/Volumes/SSD2T/workspace/go/tiktok-whisper/data/transcription.db"
export MIGRATION_BASE_DIR="/Volumes/SSD2T/workspace/go/tiktok-whisper/data/migration"

echo "=== TikTok-Whisper 数据库迁移 ==="
echo "数据库路径: $DB_PATH"
echo "迁移目录: $MIGRATION_BASE_DIR"
echo ""

# 创建迁移目录
mkdir -p "$MIGRATION_BASE_DIR"

# 执行迁移步骤
cd /Volumes/SSD2T/workspace/go/tiktok-whisper/scripts/migration

echo "步骤 1: 执行迁移前检查..."
./01_pre_migration_check.sh

echo ""
echo "准备就绪。是否继续执行迁移？"
read -p "输入 'yes' 继续: " confirm
if [ "$confirm" != "yes" ]; then
    echo "迁移已取消。"
    exit 0
fi

echo ""
echo "步骤 2: 执行数据库迁移..."
./02_execute_migration.sh

echo ""
echo "步骤 3: 执行迁移后验证..."
./03_post_migration_check.sh

echo ""
echo "=== 迁移完成 ==="