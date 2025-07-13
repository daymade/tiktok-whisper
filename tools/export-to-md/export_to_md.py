#!/usr/bin/env python3
"""
Tiktok-Whisper 数据导出到 Markdown 工具

这个脚本自动化了从 SQLite 数据库导出转录数据到 Markdown 文件的整个流程。

用法:
    python export_to_md.py list-users
    python export_to_md.py export --user "用户名" [选项]
    python export_to_md.py export-all [选项]
    python export_to_md.py config --set key=value
"""

import argparse
import json
import os
import sqlite3
import subprocess
import sys
import tempfile
from pathlib import Path
from typing import Dict, List, Optional, Tuple

# 颜色输出
class Colors:
    RED = '\033[91m'
    GREEN = '\033[92m'
    YELLOW = '\033[93m'
    BLUE = '\033[94m'
    PURPLE = '\033[95m'
    CYAN = '\033[96m'
    WHITE = '\033[97m'
    ENDC = '\033[0m'
    BOLD = '\033[1m'

def colorprint(text: str, color: str = Colors.WHITE, bold: bool = False):
    """彩色打印"""
    prefix = (Colors.BOLD if bold else '') + color
    print(f"{prefix}{text}{Colors.ENDC}")

class Config:
    """配置管理类"""
    
    def __init__(self, config_path: str = "config.json"):
        self.config_path = config_path
        self.config = self.load_config()
    
    def load_config(self) -> Dict:
        """加载配置文件"""
        try:
            if os.path.exists(self.config_path):
                with open(self.config_path, 'r', encoding='utf-8') as f:
                    return json.load(f)
            else:
                colorprint(f"配置文件不存在: {self.config_path}", Colors.YELLOW)
                return self.get_default_config()
        except Exception as e:
            colorprint(f"加载配置失败: {e}", Colors.RED)
            return self.get_default_config()
    
    def get_default_config(self) -> Dict:
        """默认配置"""
        return {
            "database_path": "../../data/transcription.db",
            "html2md_path": "/path/to/html2md/main.py",
            "default_output_dir": "./output",
            "keep_json": False,
            "keep_md_files": False,
            "batch_size": 50,
            "max_records": None,
            "date_format": "%Y-%m-%d %H:%M:%S"
        }
    
    def save_config(self):
        """保存配置文件"""
        try:
            with open(self.config_path, 'w', encoding='utf-8') as f:
                json.dump(self.config, f, ensure_ascii=False, indent=2)
            colorprint(f"配置已保存到: {self.config_path}", Colors.GREEN)
        except Exception as e:
            colorprint(f"保存配置失败: {e}", Colors.RED)
    
    def get(self, key: str, default=None):
        """获取配置值"""
        return self.config.get(key, default)
    
    def set(self, key: str, value):
        """设置配置值"""
        self.config[key] = value

class DatabaseManager:
    """数据库管理类"""
    
    def __init__(self, db_path: str):
        self.db_path = db_path
        self.verify_database()
    
    def verify_database(self):
        """验证数据库文件"""
        if not os.path.exists(self.db_path):
            raise FileNotFoundError(f"数据库文件不存在: {self.db_path}")
    
    def get_connection(self) -> sqlite3.Connection:
        """获取数据库连接"""
        try:
            conn = sqlite3.connect(self.db_path)
            conn.row_factory = sqlite3.Row  # 使结果可以按列名访问
            return conn
        except Exception as e:
            raise Exception(f"连接数据库失败: {e}")
    
    def list_users(self) -> List[Tuple[str, int]]:
        """列出所有用户及其记录数"""
        try:
            with self.get_connection() as conn:
                cursor = conn.execute("""
                    SELECT user, COUNT(*) as count 
                    FROM transcriptions 
                    WHERE has_error = 0 
                      AND transcription IS NOT NULL 
                      AND transcription != '' 
                    GROUP BY user 
                    ORDER BY count DESC
                """)
                return cursor.fetchall()
        except Exception as e:
            raise Exception(f"查询用户列表失败: {e}")
    
    def user_exists(self, username: str) -> bool:
        """检查用户是否存在"""
        try:
            with self.get_connection() as conn:
                cursor = conn.execute("""
                    SELECT COUNT(*) as count 
                    FROM transcriptions 
                    WHERE user = ? AND has_error = 0
                """, (username,))
                result = cursor.fetchone()
                return result['count'] > 0
        except Exception as e:
            raise Exception(f"检查用户存在性失败: {e}")
    
    def export_user_data(self, username: str, limit: Optional[int] = None) -> List[Dict]:
        """导出用户数据"""
        try:
            with self.get_connection() as conn:
                query = """
                    SELECT mp3_file_name, transcription 
                    FROM transcriptions 
                    WHERE has_error = 0 
                      AND transcription IS NOT NULL 
                      AND transcription != '' 
                      AND user = ? 
                    ORDER BY last_conversion_time DESC
                """
                params = (username,)
                
                if limit:
                    query += " LIMIT ?"
                    params = (username, limit)
                
                cursor = conn.execute(query, params)
                return [dict(row) for row in cursor.fetchall()]
        except Exception as e:
            raise Exception(f"导出用户数据失败: {e}")

class ExportManager:
    """导出管理类"""
    
    def __init__(self, config: Config):
        self.config = config
        self.db_manager = DatabaseManager(config.get('database_path'))
        self.verify_html2md_tool()
    
    def verify_html2md_tool(self):
        """验证 html2md 工具"""
        html2md_path = self.config.get('html2md_path')
        if not os.path.exists(html2md_path):
            raise FileNotFoundError(f"html2md 工具不存在: {html2md_path}")
    
    def list_users(self):
        """列出所有用户"""
        try:
            users = self.db_manager.list_users()
            if not users:
                colorprint("未找到任何用户数据", Colors.YELLOW)
                return
            
            colorprint(f"\n{'='*50}", Colors.CYAN, bold=True)
            colorprint("用户列表 (按记录数排序)", Colors.CYAN, bold=True)
            colorprint(f"{'='*50}", Colors.CYAN, bold=True)
            
            total_records = 0
            for i, (username, count) in enumerate(users, 1):
                total_records += count
                colorprint(f"{i:2d}. {username:<30} ({count:>4d} 条记录)", Colors.WHITE)
            
            colorprint(f"\n总计: {len(users)} 个用户, {total_records} 条记录", Colors.GREEN, bold=True)
            
        except Exception as e:
            colorprint(f"列出用户失败: {e}", Colors.RED)
            sys.exit(1)
    
    def export_user(self, username: str, output_dir: str, limit: Optional[int] = None):
        """导出单个用户的数据"""
        try:
            # 验证用户存在
            if not self.db_manager.user_exists(username):
                colorprint(f"用户不存在: {username}", Colors.RED)
                sys.exit(1)
            
            # 创建输出目录
            output_path = Path(output_dir)
            output_path.mkdir(parents=True, exist_ok=True)
            
            colorprint(f"开始导出用户: {username}", Colors.BLUE, bold=True)
            
            # 导出数据
            data = self.db_manager.export_user_data(username, limit)
            if not data:
                colorprint(f"用户 {username} 没有有效的转录数据", Colors.YELLOW)
                return
            
            colorprint(f"找到 {len(data)} 条记录", Colors.GREEN)
            
            # 创建临时 JSON 文件在输出目录中
            temp_json_name = f"temp_export_{username.replace(' ', '_')}.json"
            temp_json_path = output_path / temp_json_name
            with open(temp_json_path, 'w', encoding='utf-8') as temp_file:
                json.dump(data, temp_file, ensure_ascii=False, indent=2)
            
            try:
                # 调用 html2md 工具
                colorprint("正在转换为 Markdown...", Colors.BLUE)
                result = subprocess.run([
                    'python', 
                    self.config.get('html2md_path'), 
                    temp_json_name
                ], capture_output=True, text=True, cwd=str(output_path))
                
                if result.returncode != 0:
                    raise Exception(f"html2md 工具执行失败: {result.stderr}")
                
                # 查找生成的 ZIP 文件
                json_filename = temp_json_path.stem
                zip_filename = f"{json_filename}.zip"
                zip_path = output_path / zip_filename
                
                if zip_path.exists():
                    # 重命名为更有意义的文件名
                    final_zip_name = f"{username.replace(' ', '_')}_transcriptions.zip"
                    final_zip_path = output_path / final_zip_name
                    zip_path.rename(final_zip_path)
                    
                    colorprint(f"✅ 导出成功!", Colors.GREEN, bold=True)
                    colorprint(f"📁 输出文件: {final_zip_path}", Colors.CYAN)
                    colorprint(f"📊 记录数量: {len(data)}", Colors.CYAN)
                else:
                    colorprint("未找到生成的 ZIP 文件", Colors.RED)
                
            finally:
                # 清理临时文件
                if not self.config.get('keep_json'):
                    temp_json_path.unlink(missing_ok=True)
                
        except Exception as e:
            colorprint(f"导出失败: {e}", Colors.RED)
            sys.exit(1)
    
    def export_all_users(self, output_dir: str):
        """导出所有用户的数据"""
        try:
            users = self.db_manager.list_users()
            if not users:
                colorprint("未找到任何用户数据", Colors.YELLOW)
                return
            
            colorprint(f"准备导出 {len(users)} 个用户的数据", Colors.BLUE, bold=True)
            
            base_output_path = Path(output_dir)
            base_output_path.mkdir(parents=True, exist_ok=True)
            
            success_count = 0
            for i, (username, count) in enumerate(users, 1):
                colorprint(f"\n[{i}/{len(users)}] 导出用户: {username} ({count} 条记录)", Colors.PURPLE)
                
                try:
                    user_output_dir = base_output_path / username.replace(' ', '_')
                    self.export_user(username, str(user_output_dir))
                    success_count += 1
                except Exception as e:
                    colorprint(f"导出用户 {username} 失败: {e}", Colors.RED)
                    continue
            
            colorprint(f"\n✅ 批量导出完成! 成功: {success_count}/{len(users)}", Colors.GREEN, bold=True)
            colorprint(f"📁 输出目录: {base_output_path}", Colors.CYAN)
            
        except Exception as e:
            colorprint(f"批量导出失败: {e}", Colors.RED)
            sys.exit(1)

def main():
    parser = argparse.ArgumentParser(
        description="Tiktok-Whisper 数据导出到 Markdown 工具",
        formatter_class=argparse.RawDescriptionHelpFormatter,
        epilog="""
示例用法:
  %(prog)s list-users                           # 列出所有用户
  %(prog)s export --user "经纬第二期"            # 导出指定用户
  %(prog)s export --user "经纬第二期" --limit 100 # 限制导出记录数
  %(prog)s export-all                          # 导出所有用户
  %(prog)s config --set html2md_path="/path/to/html2md/main.py"
        """
    )
    
    subparsers = parser.add_subparsers(dest='command', help='可用命令')
    
    # list-users 命令
    subparsers.add_parser('list-users', help='列出所有用户及其记录数')
    
    # export 命令
    export_parser = subparsers.add_parser('export', help='导出指定用户的数据')
    export_parser.add_argument('--user', required=True, help='用户名')
    export_parser.add_argument('--output', default=None, help='输出目录 (默认: 配置文件中的设置)')
    export_parser.add_argument('--limit', type=int, help='最大导出记录数')
    
    # export-all 命令
    export_all_parser = subparsers.add_parser('export-all', help='导出所有用户的数据')
    export_all_parser.add_argument('--output', default=None, help='输出目录 (默认: 配置文件中的设置)')
    
    # config 命令
    config_parser = subparsers.add_parser('config', help='配置管理')
    config_parser.add_argument('--set', help='设置配置项 (格式: key=value)')
    config_parser.add_argument('--show', action='store_true', help='显示当前配置')
    
    args = parser.parse_args()
    
    if not args.command:
        parser.print_help()
        return
    
    # 加载配置
    script_dir = Path(__file__).parent
    config_path = script_dir / "config.json"
    config = Config(str(config_path))
    
    if args.command == 'config':
        if args.show:
            colorprint("当前配置:", Colors.CYAN, bold=True)
            print(json.dumps(config.config, ensure_ascii=False, indent=2))
        elif args.set:
            try:
                key, value = args.set.split('=', 1)
                # 尝试解析为 JSON 值
                try:
                    value = json.loads(value)
                except json.JSONDecodeError:
                    pass  # 保持字符串值
                config.set(key.strip(), value)
                config.save_config()
                colorprint(f"已设置 {key} = {value}", Colors.GREEN)
            except ValueError:
                colorprint("格式错误，请使用: key=value", Colors.RED)
                sys.exit(1)
        else:
            config_parser.print_help()
        return
    
    # 创建导出管理器
    try:
        export_manager = ExportManager(config)
    except Exception as e:
        colorprint(f"初始化失败: {e}", Colors.RED)
        colorprint("\n请检查配置文件中的路径设置:", Colors.YELLOW)
        colorprint(f"  database_path: {config.get('database_path')}", Colors.WHITE)
        colorprint(f"  html2md_path: {config.get('html2md_path')}", Colors.WHITE)
        colorprint(f"\n使用 'python {sys.argv[0]} config --set key=value' 来修改配置", Colors.CYAN)
        sys.exit(1)
    
    # 执行命令
    if args.command == 'list-users':
        export_manager.list_users()
    
    elif args.command == 'export':
        output_dir = args.output or config.get('default_output_dir')
        export_manager.export_user(args.user, output_dir, args.limit)
    
    elif args.command == 'export-all':
        output_dir = args.output or config.get('default_output_dir')
        export_manager.export_all_users(output_dir)

if __name__ == '__main__':
    main()