import os
import argparse
from multiprocessing import Pool, freeze_support
from whisperToText import process_files

# 创建命令行参数解析器
parser = argparse.ArgumentParser()
parser.add_argument("--base_input_dir", type=str, required=True, help="Base input directory containing the audio files")
parser.add_argument("--base_output_dir", type=str, required=True, help="Base output directory to save the transcripts")
parser.add_argument("--processes", type=int, default=1, help="Number of parallel processes")
args = parser.parse_args()

def process_directory(subdir):
    input_dir = os.path.join(args.base_input_dir, subdir)
    output_dir = os.path.join(args.base_output_dir, subdir)

    # 创建输出目录，如果不存在的话
    os.makedirs(output_dir, exist_ok=True)

    process_files(input_dir, output_dir)

if __name__ == '__main__':
    freeze_support()

    # 获取所有子目录
    subdirs = [name for name in os.listdir(args.base_input_dir) if os.path.isdir(os.path.join(args.base_input_dir, name))]

    # 处理每个子目录
    for subdir in subdirs:
        process_directory(subdir)
