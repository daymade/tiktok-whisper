import os
from faster_whisper import WhisperModel

model_size = "large-v2"
model = WhisperModel(model_size, device="cuda", compute_type="float16")

input_dir = r"G:\daymade\whisper\tiktok-whisper\data\xiaoyuzhou\output\虎言乱语"
output_dir = r"G:\daymade\whisper\tiktok-whisper\data\xiaoyuzhou\text_output\虎言乱语"

def transcribe_file(file_path):
    print(f"Start transcribing {file_path}")
    segments, info = model.transcribe(file_path, beam_size=5, language="zh")
    print("Detected language '%s' with probability %f" % (info.language, info.language_probability))

    output_file_path = os.path.join(output_dir, os.path.splitext(os.path.basename(file_path))[0] + '.txt')

    # If output file already exists, skip this file
    if os.path.exists(output_file_path):
        print(f"Output file {output_file_path} already exists, skipping...")
        return

    with open(output_file_path, 'w', encoding='utf-8') as f:
        for i, segment in enumerate(segments, 1):
            f.write("[%.2fs -> %.2fs] %s\n" % (segment.start, segment.end, segment.text))
            print(f"Written segment {i} out of {len(segments)}")

# 创建输出目录，如果不存在的话
if not os.path.exists(output_dir):
    os.makedirs(output_dir)

# 获取目录下所有wav文件
wav_files = [os.path.join(input_dir, file) for file in os.listdir(input_dir) if file.endswith('.wav')]

# 顺序处理文件
for i, wav_file in enumerate(wav_files, 1):
    print(f"Start processing file {i} out of {len(wav_files)}: {wav_file}")
    transcribe_file(wav_file)
    print(f"Finished processing file {i} out of {len(wav_files)}: {wav_file}")
