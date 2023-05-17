import os
import subprocess
import argparse
from pathlib import Path

def convert_files(input_dir, output_dir, extension='m4a'):
    input_dir = Path(input_dir)
    output_dir = Path(output_dir)

    # Ensure the output directory exists
    output_dir.mkdir(parents=True, exist_ok=True)

    # Loop over all files in the input directory
    for audio_file in input_dir.glob(f'*.{extension}'):
        # Define the output file name
        output_file = output_dir / audio_file.with_suffix('.wav').name

        # Use ffmpeg to convert the audio
        subprocess.run(['ffmpeg', '-i', str(audio_file), '-ar', '16000', str(output_file)])

if __name__ == "__main__":
    parser = argparse.ArgumentParser(description="Convert audio files to 16khz wav format")
    parser.add_argument("input_dir", help="Directory containing the input audio files")
    parser.add_argument("output_dir", help="Directory to save the converted audio files")
    args = parser.parse_args()

    convert_files(args.input_dir, args.output_dir)
