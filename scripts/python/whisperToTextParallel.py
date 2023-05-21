import os
import time
import argparse
from multiprocessing import Pool, freeze_support
from faster_whisper import WhisperModel

# 创建命令行参数解析器
parser = argparse.ArgumentParser()
parser.add_argument("--processes", type=int, default=1, help="Number of parallel processes")
args = parser.parse_args()

model_size = "large-v2"
model = WhisperModel(model_size, device="cuda", compute_type="float16")

base_input_dir = r"G:\daymade\whisper\tiktok-whisper\data\xiaoyuzhou"
base_output_dir = r"G:\daymade\whisper\tiktok-whisper\data\text"

def transcribe_file(args):
    file_path, output_dir = args
    start_time = time.time()
    print(f"Start transcribing {file_path}")
    segments, info = model.transcribe(file_path, beam_size=5, language="zh", initial_prompt="以下是简体中文")
    transcribe_time = time.time()
    print("Detected language '%s' with probability %f, transcribe time: %.2f seconds" % (info.language, info.language_probability, transcribe_time - start_time))

    episode = os.path.splitext(os.path.basename(file_path))[0]
    output_file_path = os.path.join(output_dir, episode + '.txt')

    # If output file already exists, skip this file
    if os.path.exists(output_file_path):
        print(f"Output file {output_file_path} already exists, skipping...")
        return

    write_start_time = time.time()

    print_frequency = 100  # 输出频率，即每隔多少次输出一次
    print_interval = 30  # 输出间隔，即每隔多少秒输出一次
    last_print_time = time.time()  # 上次输出的时间

    with open(output_file_path, 'w', encoding='utf-8') as f:
        for i, segment in enumerate(segments, 1):
            f.write("[%.2fs -> %.2fs] %s\n" % (segment.start, segment.end, segment.text))
            # 检查是否满足输出条件：已经过去了 print_interval 秒或者已经处理了 print_frequency 个段落
            if i % print_frequency == 0 or time.time() - last_print_time >= print_interval:
                print(f"Progress: {round(segment.end)}s of {round(info.duration)}s for {episode}")
                last_print_time = time.time()  # 更新上次输出的时间

    write_end_time = time.time()
    print(f"Transcript time: {write_end_time - write_start_time}s for {round(info.duration)}s")


def process_directory(subdir):
    input_dir = os.path.join(base_input_dir, subdir)
    output_dir = os.path.join(base_output_dir, subdir)

    # 创建输出目录，如果不存在的话
    os.makedirs(output_dir, exist_ok=True)

    # 获取目录下所有wav文件
    wav_files = [(os.path.join(input_dir, file), output_dir) for file in os.listdir(input_dir) if file.endswith('.m4a')]

    # 创建一个进程池
    with Pool(processes=args.processes) as pool:
        # 使用进程池处理文件
        pool.map(transcribe_file, wav_files)

if __name__ == '__main__':
    freeze_support()

    # 获取所有子目录
    subdirs = [name for name in os.listdir(base_input_dir) if os.path.isdir(os.path.join(base_input_dir, name))]

    # 处理每个子目录
    for subdir in subdirs:
        process_directory(subdir)