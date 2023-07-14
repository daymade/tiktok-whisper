import os
import time
from faster_whisper import WhisperModel

model_size = "large-v2"
model = WhisperModel(model_size, device="cuda", compute_type="float16")

input_dir = r"/home/daymade/download/youtube"
output_dir = r"/home/daymade/download/youtube_output"

def transcribe_file(file_path):
    print(f"Start transcribing {file_path}")
    segments, info = model.transcribe(file_path, beam_size=5, language="zh")
    print("Detected language '%s' with probability %f" % (info.language, info.language_probability))

    episode = os.path.splitext(os.path.basename(file_path))[0]
    output_file_path = os.path.join(output_dir, os.path.splitext(os.path.basename(file_path))[0] + '.txt')

    # If output file already exists, skip this file
    if os.path.exists(output_file_path):
        print(f"Output file {output_file_path} already exists, skipping...")
        return

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

# 创建输出目录，如果不存在的话
if not os.path.exists(output_dir):
    os.makedirs(output_dir)

# 获取目录下所有wav文件
wav_files = [os.path.join(input_dir, file) for file in os.listdir(input_dir) if file.endswith('.m4a')]

# 顺序处理文件
for i, wav_file in enumerate(wav_files, 1):
    print(f"Start processing file {i} out of {len(wav_files)}: {wav_file}")
    transcribe_file(wav_file)
    print(f"Finished processing file {i} out of {len(wav_files)}: {wav_file}")
