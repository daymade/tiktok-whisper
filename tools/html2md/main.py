import json
import sys
import os
import zipfile

def json_to_markdown(json_data):
    markdown = ""
    for item in json_data:
        markdown += f"## {item['mp3_file_name']}\n\n"
        transcription = item['transcription'].replace('\\n', '\n')
        transcription = '\n\n'.join(line.strip() for line in transcription.split('\n') if line.strip())
        markdown += transcription + "\n\n"
    return markdown

def create_zip(md_files, zip_filename):
    with zipfile.ZipFile(zip_filename, 'w', zipfile.ZIP_DEFLATED) as zipf:
        for md_file in md_files:
            zipf.write(md_file, os.path.basename(md_file))
    print(f"ZIP 文件已生成：{zip_filename}")

if len(sys.argv) < 2:
    print("请提供 JSON 文件名作为参数")
    sys.exit(1)

input_filename = sys.argv[1]

try:
    with open(input_filename, 'r', encoding='utf-8') as file:
        json_data = json.load(file)

    total_items = len(json_data)
    num_files = (total_items + 49) // 50

    md_files = []

    for i in range(num_files):
        start = i * 50
        end = min((i + 1) * 50, total_items)
        current_batch = json_data[start:end]
        markdown_output = json_to_markdown(current_batch)

        base_filename = os.path.splitext(input_filename)[0]
        output_filename = f"{base_filename}_{i+1}.md"

        with open(output_filename, 'w', encoding='utf-8') as file:
            file.write(markdown_output)

        md_files.append(output_filename)
        print(f"Markdown 文件已生成：{output_filename}")

    print(f"总共生成了 {num_files} 个 Markdown 文件")

    # 创建 ZIP 文件
    zip_filename = f"{base_filename}.zip"
    create_zip(md_files, zip_filename)

    # 删除临时生成的 MD 文件
    for md_file in md_files:
        os.remove(md_file)
    print("临时 Markdown 文件已删除")

except FileNotFoundError:
    print(f"错误：文件 '{input_filename}' 不存在")
except json.JSONDecodeError:
    print(f"错误：'{input_filename}' 不是有效的 JSON 文件")
except Exception as e:
    print(f"发生错误：{str(e)}")