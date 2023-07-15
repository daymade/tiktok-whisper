import os
from transcribe_utils import transcribe_file

# Supported audio formats
audio_extensions = ['.m4a', '.wav', '.mp3', '.flac']

def process_files(input_dir, output_dir):
    # 获取目录下所有支持的音频文件
    audio_files = [os.path.join(input_dir, file) for file in os.listdir(input_dir) 
                   if any(file.endswith(ext) for ext in audio_extensions)]

    # If no audio files are found, print an error message and exit
    if not audio_files:
        print(f"No audio files found in directory {input_dir}. Please check the directory and try again.")
        exit()

    # 顺序处理文件
    for i, audio_file in enumerate(audio_files, 1):
        print(f"Start processing file {i} out of {len(audio_files)}: {audio_file}")
        transcribe_file(audio_file, output_dir)
        print(f"Finished processing file {i} out of {len(audio_files)}: {audio_file}")

if __name__ == "__main__":
    import argparse
    # 创建命令行参数解析器
    parser = argparse.ArgumentParser()
    parser.add_argument("--input_dir", type=str, required=True, help="Input directory containing the audio files")
    parser.add_argument("--output_dir", type=str, required=True, help="Output directory to save the transcripts")
    args = parser.parse_args()
    
    # 创建输出目录，如果不存在的话
    if not os.path.exists(args.output_dir):
        os.makedirs(args.output_dir)
    process_files(args.input_dir, args.output_dir)
