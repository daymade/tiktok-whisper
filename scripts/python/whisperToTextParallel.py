import os
import time
import argparse
from multiprocessing import Pool
from faster_whisper import WhisperModel

# 创建命令行参数解析器
parser = argparse.ArgumentParser()
parser.add_argument("--processes", type=int, default=1, help="Number of parallel processes")
args = parser.parse_args()

model_size = "large-v2"
model = WhisperModel(model_size, device="cuda", compute_type="float16")

input_dir = r"G:\daymade\whisper\tiktok-whisper\data\xiaoyuzhou\output\虎言乱语"
output_dir = r"G:\daymade\whisper\tiktok-whisper\data\xiaoyuzhou\text_output\虎言乱语"

def transcribe_file(file_path):
    start_time = time.time()
    print(f"Start transcribing {file_path}")
    segments, info = model.transcribe(file_path, beam_size=5, language="zh")
    transcribe_time = time.time()
    print("Detected language '%s' with probability %f, transcribe time: %.2f seconds" % (info.language, info.language_probability, transcribe_time - start_time))

    output_file_path = os.path.join(output_dir, os.path.splitext(os.path.basename(file_path))[0] + '.txt')

    # If output file already exists, skip this file
    if os.path.exists(output_file_path):
        print(f"Output file {output_file_path} already exists, skipping...")
        return

    write_start_time = time.time()
    with open(output_file_path, 'w', encoding='utf-8') as f:
        for i, segment in enumerate(segments, 1):
            f.write("[%.2fs -> %.2fs] %s\n" % (segment.start, segment.end, segment.text))
            print(f"Written segment {i} out of {len(segments)}")
    write_end_time = time.time()
    print(f"Writing time: {write_end_time - write_start_time} seconds")

# 创建输出目录，如果不存在的话
if not os.path.exists(output_dir):
    os.makedirs(output_dir)

# 获取目录下所有wav文件
wav_files = [os.path.join(input_dir, file) for file in os.listdir(input_dir) if file.endswith('.wav')]

# python main.py --processes 4
# 创建一个进程池
with Pool(processes=args.processes) as pool:
    # 使用进程池处理文件
    pool.map(transcribe_file, wav_files)
