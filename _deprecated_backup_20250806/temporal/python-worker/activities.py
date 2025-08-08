"""
Activity implementations for v2t Python worker
Activities run outside the workflow sandbox and can use any libraries
"""

import asyncio
import logging
import os
import tempfile
from pathlib import Path

import ffmpeg
import yt_dlp
from temporalio import activity

logger = logging.getLogger(__name__)


@activity.defn
async def download_video_activity(url: str, quality: str = "720p") -> dict:
    """Download video using yt-dlp (runs outside workflow sandbox)."""
    logger.info(f"Downloading video from URL: {url}")
    
    # Create temp directory for downloads
    temp_dir = tempfile.mkdtemp(prefix="v2t-download-")
    
    # Configure yt-dlp options
    ydl_opts = {
        'format': f'bestvideo[height<={quality[:-1]}]+bestaudio/best[height<={quality[:-1]}]',
        'outtmpl': os.path.join(temp_dir, '%(title)s.%(ext)s'),
        'quiet': True,
        'no_warnings': True,
        'extract_flat': False,
        'force_generic_extractor': False,
    }
    
    try:
        with yt_dlp.YoutubeDL(ydl_opts) as ydl:
            info = ydl.extract_info(url, download=True)
            
            # Get downloaded file path
            filename = ydl.prepare_filename(info)
            if not os.path.exists(filename):
                # Try with different extension
                base = os.path.splitext(filename)[0]
                for ext in ['.mp4', '.webm', '.mkv', '.avi']:
                    if os.path.exists(base + ext):
                        filename = base + ext
                        break
            
            return {
                "video_id": info.get('id', 'unknown'),
                "title": info.get('title', 'Unknown Title'),
                "file_path": filename,
                "duration": info.get('duration', 0),
                "format": info.get('ext', 'unknown'),
            }
    except Exception as e:
        logger.error(f"Failed to download video: {e}")
        raise


@activity.defn
async def convert_to_audio_activity(video_path: str, output_format: str = "wav") -> dict:
    """Convert video to audio using ffmpeg."""
    logger.info(f"Converting video to audio: {video_path}")
    
    input_path = Path(video_path)
    output_path = input_path.with_suffix(f".{output_format}")
    
    try:
        # Use ffmpeg to extract audio
        stream = ffmpeg.input(str(input_path))
        stream = ffmpeg.output(stream, str(output_path), 
                              acodec='pcm_s16le' if output_format == 'wav' else 'libmp3lame',
                              ar='16000',  # 16kHz sample rate for whisper
                              ac=1)  # Mono audio
        ffmpeg.run(stream, overwrite_output=True, capture_stdout=True, capture_stderr=True)
        
        return {
            "output_path": str(output_path),
            "size": output_path.stat().st_size,
            "duration": 0,  # Could extract with ffprobe if needed
        }
    except ffmpeg.Error as e:
        logger.error(f"FFmpeg error: {e.stderr.decode()}")
        raise


@activity.defn
async def cleanup_temp_files_activity(file_paths: list) -> None:
    """Clean up temporary files."""
    for path in file_paths:
        try:
            if os.path.exists(path):
                os.remove(path)
                logger.info(f"Cleaned up file: {path}")
                
            # Also try to remove parent directory if empty
            parent = os.path.dirname(path)
            if os.path.exists(parent) and not os.listdir(parent):
                os.rmdir(parent)
                logger.info(f"Cleaned up directory: {parent}")
        except Exception as e:
            logger.warning(f"Failed to clean up {path}: {e}")