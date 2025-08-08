#!/usr/bin/env python3
"""
Python Worker for v2t Distributed Transcription System using faster-whisper
"""

import asyncio
import json
import logging
import os
import subprocess
import tempfile
from datetime import timedelta
from pathlib import Path
from typing import Any, Dict, Optional

import ffmpeg
import yt_dlp
from dotenv import load_dotenv
from faster_whisper import WhisperModel
from minio import Minio
from temporalio import activity, workflow
from temporalio.client import Client
from temporalio.worker import Worker

# Load environment variables
load_dotenv()

# Configure logging
logging.basicConfig(
    level=logging.INFO,
    format="%(asctime)s - %(name)s - %(levelname)s - %(message)s",
)
logger = logging.getLogger(__name__)

# MinIO configuration
minio_client = Minio(
    os.getenv("MINIO_ENDPOINT", "localhost:9000"),
    access_key=os.getenv("MINIO_ACCESS_KEY", "minioadmin"),
    secret_key=os.getenv("MINIO_SECRET_KEY", "minioadmin"),
    secure=False,
)

# Global model cache
whisper_models: Dict[str, WhisperModel] = {}


def get_whisper_model(model_size: str = "large-v3", device: str = "auto") -> WhisperModel:
    """Get or create a WhisperModel instance with caching."""
    cache_key = f"{model_size}_{device}"
    
    if cache_key not in whisper_models:
        logger.info(f"Loading faster-whisper model: {model_size} on device: {device}")
        
        # Determine compute type based on device
        if device == "cuda":
            compute_type = "float16"
        else:
            compute_type = "int8"  # More efficient for CPU
        
        whisper_models[cache_key] = WhisperModel(
            model_size,
            device=device,
            compute_type=compute_type,
            num_workers=4,  # Parallel processing
            download_root="/tmp/whisper-models",
        )
        
    return whisper_models[cache_key]


@activity.defn
async def download_from_youtube(url: str) -> Dict[str, Any]:
    """Download audio from YouTube using yt-dlp."""
    logger.info(f"Downloading from YouTube: {url}")
    
    output_dir = Path("/tmp/v2t-downloads")
    output_dir.mkdir(exist_ok=True)
    
    # Configure yt-dlp
    ydl_opts = {
        "format": "bestaudio/best",
        "postprocessors": [{
            "key": "FFmpegExtractAudio",
            "preferredcodec": "wav",
            "preferredquality": "192",
        }],
        "outtmpl": str(output_dir / "%(id)s.%(ext)s"),
        "quiet": True,
        "no_warnings": True,
    }
    
    try:
        with yt_dlp.YoutubeDL(ydl_opts) as ydl:
            info = ydl.extract_info(url, download=True)
            video_id = info["id"]
            output_file = output_dir / f"{video_id}.wav"
            
            return {
                "status": "success",
                "file_path": str(output_file),
                "title": info.get("title", "Unknown"),
                "duration": info.get("duration", 0),
                "uploader": info.get("uploader", "Unknown"),
            }
    except Exception as e:
        logger.error(f"YouTube download failed: {e}")
        return {
            "status": "error",
            "error": str(e),
        }


@activity.defn
async def convert_audio_format(
    input_path: str, output_format: str = "wav", sample_rate: int = 16000
) -> Dict[str, Any]:
    """Convert audio to optimal format for transcription using ffmpeg."""
    logger.info(f"Converting audio: {input_path} to {output_format}")
    
    try:
        output_path = input_path.rsplit(".", 1)[0] + f"_converted.{output_format}"
        
        # Use ffmpeg-python for conversion
        stream = ffmpeg.input(input_path)
        stream = ffmpeg.output(
            stream,
            output_path,
            acodec="pcm_s16le",
            ac=1,  # Mono
            ar=sample_rate,
        )
        ffmpeg.run(stream, overwrite_output=True, quiet=True)
        
        # Clean up original if different
        if output_path != input_path:
            os.remove(input_path)
        
        return {
            "status": "success",
            "file_path": output_path,
            "format": output_format,
            "sample_rate": sample_rate,
        }
    except Exception as e:
        logger.error(f"Audio conversion failed: {e}")
        return {
            "status": "error",
            "error": str(e),
            "file_path": input_path,  # Return original path
        }


@activity.defn
async def transcribe_with_faster_whisper(request: Dict[str, Any]) -> Dict[str, Any]:
    """Transcribe audio using faster-whisper with GPU acceleration when available."""
    file_path = request["file_path"]
    language = request.get("language", None)
    model_size = request.get("model_size", "large-v3")
    
    logger.info(f"Starting faster-whisper transcription for {file_path}")
    
    try:
        # Download from MinIO if needed
        local_path = file_path
        if file_path.startswith("minio://"):
            local_path = await download_from_minio(file_path)
        
        # Get model with caching
        model = get_whisper_model(model_size)
        
        # Transcribe with faster-whisper
        segments, info = model.transcribe(
            local_path,
            language=language,
            beam_size=5,
            best_of=5,
            patience=1,
            length_penalty=1,
            temperature=[0.0, 0.2, 0.4, 0.6, 0.8, 1.0],
            compression_ratio_threshold=2.4,
            log_prob_threshold=-1.0,
            no_speech_threshold=0.6,
            condition_on_previous_text=True,
            initial_prompt=None,
            word_timestamps=True,
            prepend_punctuations="\"'([{-",
            append_punctuations="\"'.。,，!！?？:：)]}、",
            vad_filter=True,  # Voice activity detection
            vad_parameters=dict(
                threshold=0.5,
                min_speech_duration_ms=250,
                max_speech_duration_s=float("inf"),
                min_silence_duration_ms=2000,
                window_size_samples=1024,
                speech_pad_ms=400,
            ),
        )
        
        # Process segments
        full_text = []
        word_segments = []
        
        for segment in segments:
            full_text.append(segment.text)
            
            # Include word-level timestamps if available
            if hasattr(segment, "words") and segment.words:
                for word in segment.words:
                    word_segments.append({
                        "word": word.word,
                        "start": word.start,
                        "end": word.end,
                        "probability": word.probability,
                    })
        
        result = {
            "text": "".join(full_text),
            "language": info.language,
            "language_probability": info.language_probability,
            "duration": info.duration,
            "segments": [{
                "start": s.start,
                "end": s.end,
                "text": s.text,
                "avg_logprob": s.avg_logprob,
                "no_speech_prob": s.no_speech_prob,
            } for s in segments],
            "words": word_segments,
            "provider": "faster_whisper",
            "model": model_size,
        }
        
        # Clean up temp file if downloaded
        if local_path != file_path and os.path.exists(local_path):
            os.remove(local_path)
        
        return result
        
    except Exception as e:
        logger.error(f"Faster-whisper transcription failed: {e}")
        raise


@activity.defn
async def download_from_minio(minio_url: str) -> str:
    """Helper to download file from MinIO."""
    parts = minio_url.replace("minio://", "").split("/", 1)
    bucket = parts[0] if len(parts) > 0 else "v2t-transcriptions"
    object_key = parts[1] if len(parts) > 1 else ""
    
    temp_file = tempfile.NamedTemporaryFile(delete=False, suffix=Path(object_key).suffix)
    temp_path = temp_file.name
    temp_file.close()
    
    minio_client.fget_object(bucket, object_key, temp_path)
    return temp_path


@activity.defn
async def upload_to_minio(
    local_path: str, object_key: str, bucket: str = "v2t-transcriptions"
) -> Dict[str, Any]:
    """Upload file to MinIO."""
    try:
        with open(local_path, "rb") as f:
            file_size = os.path.getsize(local_path)
            minio_client.put_object(bucket, object_key, f, file_size)
        
        return {
            "status": "success",
            "url": f"minio://{bucket}/{object_key}",
            "size": file_size,
        }
    except Exception as e:
        logger.error(f"MinIO upload failed: {e}")
        return {
            "status": "error",
            "error": str(e),
        }


# DAG Workflow for ETL Pipeline
@workflow.defn
class TranscriptionETLWorkflow:
    """ETL workflow: Download → Convert → Transcribe → Store"""
    
    @workflow.run
    async def run(self, request: Dict[str, Any]) -> Dict[str, Any]:
        """Execute the ETL pipeline."""
        source_url = request["source_url"]
        workflow_id = workflow.info().workflow_id
        
        # Step 1: Download from source (YouTube, URL, etc.)
        if "youtube.com" in source_url or "youtu.be" in source_url:
            download_result = await workflow.execute_activity(
                download_from_youtube,
                source_url,
                start_to_close_timeout=timedelta(minutes=30),
            )
            if download_result["status"] == "error":
                return download_result
            
            file_path = download_result["file_path"]
        else:
            # Assume it's already a file path
            file_path = source_url
        
        # Step 2: Convert audio format for optimal transcription
        convert_result = await workflow.execute_activity(
            convert_audio_format,
            file_path,
            "wav",
            16000,
            start_to_close_timeout=timedelta(minutes=10),
        )
        
        if convert_result["status"] == "success":
            file_path = convert_result["file_path"]
        
        # Step 3: Transcribe with faster-whisper
        transcribe_request = {
            "file_path": file_path,
            "language": request.get("language"),
            "model_size": request.get("model_size", "large-v3"),
        }
        
        transcription = await workflow.execute_activity(
            transcribe_with_faster_whisper,
            transcribe_request,
            start_to_close_timeout=timedelta(minutes=60),
            heartbeat_timeout=timedelta(seconds=30),
        )
        
        # Step 4: Store results in MinIO
        # Save transcription as JSON
        result_path = f"/tmp/{workflow_id}_result.json"
        with open(result_path, "w", encoding="utf-8") as f:
            json.dump(transcription, f, ensure_ascii=False, indent=2)
        
        upload_result = await workflow.execute_activity(
            upload_to_minio,
            result_path,
            f"results/{workflow_id}/transcription.json",
            start_to_close_timeout=timedelta(minutes=5),
        )
        
        # Clean up local files
        for path in [file_path, result_path]:
            if os.path.exists(path):
                os.remove(path)
        
        return {
            "workflow_id": workflow_id,
            "status": "completed",
            "transcription_url": upload_result.get("url"),
            "duration": transcription.get("duration"),
            "language": transcription.get("language"),
            "provider": "faster_whisper",
        }


async def main():
    """Main worker entry point."""
    # Temporal connection
    temporal_host = os.getenv("TEMPORAL_HOST", "localhost:7233")
    client = await Client.connect(temporal_host)
    
    # Create worker
    worker = Worker(
        client,
        task_queue="v2t-transcription-queue",
        workflows=[TranscriptionETLWorkflow],
        activities=[
            download_from_youtube,
            convert_audio_format,
            transcribe_with_faster_whisper,
            download_from_minio,
            upload_to_minio,
        ],
    )
    
    logger.info(f"Python worker started, connecting to {temporal_host}")
    logger.info("Using faster-whisper for transcription")
    logger.info("Workflows: TranscriptionETLWorkflow")
    logger.info("Activities: YouTube download, Audio conversion, Transcription, Storage")
    
    # Run worker
    await worker.run()


if __name__ == "__main__":
    asyncio.run(main())