import os
import time
from faster_whisper import WhisperModel

# Shared variables
model_size = "large-v2"
model = WhisperModel(model_size, device="cuda", compute_type="float16")

# Transcribe a file
def transcribe_file(file_path, output_dir):
    print(f"Start transcribing {file_path}")
    start_time = time.time()  # Start time for transcription
    
    segments, info = model.transcribe(file_path, 
                                      beam_size=5, 
                                      language="zh", 
                                      initial_prompt="以下是简体中文普通话:")

    episode = os.path.splitext(os.path.basename(file_path))[0]
    output_file_path = os.path.join(output_dir, episode + '.txt')

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

    # Transcription end time
    end_time = time.time()
    transcription_time = end_time - start_time  # Total time taken for transcription

    # Log the length of the audio file and the time taken for transcription
    print(f"Audio length: {info.duration} seconds")
    print(f"Transcription time: {transcription_time} seconds")

    # Calculate and log the speedup factor
    speedup_factor = info.duration / transcription_time
    print(f"Speedup factor: {speedup_factor}X")
