#!/usr/bin/env python3
"""
Simplified Python Worker for v2t using faster-whisper
Activities are imported separately to avoid workflow sandbox issues
"""

import asyncio
import json
import logging
import os
from datetime import timedelta
from pathlib import Path
from typing import Any, Dict, Optional

from dotenv import load_dotenv
from faster_whisper import WhisperModel
from minio import Minio
from temporalio import activity, workflow
from temporalio.client import Client
from temporalio.worker import Worker

# Import activities that use restricted libraries
from activities import (
    download_video_activity,
    convert_to_audio_activity,
    cleanup_temp_files_activity
)

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

# Whisper model (global for reuse)
whisper_model = None


def get_whisper_model():
    """Get or create whisper model instance."""
    global whisper_model
    if whisper_model is None:
        model_size = os.getenv("WHISPER_MODEL_SIZE", "large-v3")
        device = os.getenv("WHISPER_DEVICE", "cpu")
        compute_type = os.getenv("WHISPER_COMPUTE_TYPE", "int8")
        
        logger.info(f"Loading Whisper model: {model_size} on {device} with {compute_type}")
        whisper_model = WhisperModel(model_size, device=device, compute_type=compute_type)
    
    return whisper_model


@activity.defn
async def transcribe_with_faster_whisper(file_path: str, language: str = "auto") -> dict:
    """Transcribe audio file using faster-whisper."""
    logger.info(f"Transcribing file: {file_path} with language: {language}")
    
    model = get_whisper_model()
    
    # Transcribe
    segments, info = model.transcribe(
        file_path,
        language=None if language == "auto" else language,
        beam_size=5,
        best_of=5,
        patience=1.0,
        length_penalty=1.0,
        repetition_penalty=1.0,
        no_repeat_ngram_size=0,
        temperature=0.0,
        compression_ratio_threshold=2.4,
        log_prob_threshold=-1.0,
        no_speech_threshold=0.6,
        condition_on_previous_text=True,
        prompt_reset_on_temperature=0.5,
        initial_prompt=None,
        prefix=None,
        suppress_blank=True,
        suppress_tokens=[-1],
        without_timestamps=False,
        max_initial_timestamp=1.0,
        word_timestamps=False,
        prepend_punctuations="\"'([{-",
        append_punctuations="\"'.。,!?:：）]}、",
        vad_filter=True,
        vad_parameters=dict(
            threshold=0.5,
            min_speech_duration_ms=250,
            max_speech_duration_s=float("inf"),
            min_silence_duration_ms=2000,
            window_size_samples=1024,
            speech_pad_ms=400,
        ),
    )
    
    # Collect transcription text
    full_text = ""
    for segment in segments:
        full_text += segment.text + " "
    
    logger.info(f"Transcription completed. Detected language: {info.language}")
    
    return {
        "text": full_text.strip(),
        "language": info.language,
        "duration": info.duration,
        "provider": "faster-whisper",
    }


@activity.defn
async def upload_to_minio_activity(local_path: str, object_key: str, metadata: dict = None) -> dict:
    """Upload file to MinIO."""
    logger.info(f"Uploading {local_path} to MinIO as {object_key}")
    
    bucket_name = "v2t-transcriptions"
    
    # Ensure bucket exists
    if not minio_client.bucket_exists(bucket_name):
        minio_client.make_bucket(bucket_name)
    
    # Upload file
    result = minio_client.fput_object(
        bucket_name,
        object_key,
        local_path,
        metadata=metadata,
    )
    
    return {
        "object_key": object_key,
        "etag": result.etag,
        "size": result.size,
        "url": f"minio://{bucket_name}/{object_key}",
    }


@activity.defn
async def download_from_minio_activity(object_key: str) -> dict:
    """Download file from MinIO."""
    logger.info(f"Downloading {object_key} from MinIO")
    
    bucket_name = "v2t-transcriptions"
    local_path = f"/tmp/v2t-temporal/{object_key}"
    
    # Ensure directory exists
    os.makedirs(os.path.dirname(local_path), exist_ok=True)
    
    # Download file
    minio_client.fget_object(bucket_name, object_key, local_path)
    
    return {
        "local_path": local_path,
        "size": os.path.getsize(local_path),
    }


@workflow.defn
class SimpleTranscriptionWorkflow:
    """Simple workflow for transcribing a single file."""
    
    @workflow.run
    async def run(self, request: Dict[str, Any]) -> Dict[str, Any]:
        """Execute simple transcription workflow."""
        file_path = request["file_path"]
        language = request.get("language", "auto")
        file_id = request.get("file_id", "unknown")
        
        workflow.logger.info(f"Starting simple transcription for {file_id}")
        
        # Transcribe file
        result = await workflow.execute_activity(
            transcribe_with_faster_whisper,
            file_path,
            language,
            start_to_close_timeout=timedelta(minutes=30),
        )
        
        # Save transcription
        output_path = f"{file_path}_transcription.txt"
        with open(output_path, "w", encoding="utf-8") as f:
            f.write(result["text"])
        
        return {
            "file_id": file_id,
            "transcription_url": output_path,
            "provider": result["provider"],
            "language": result["language"],
            "duration": result["duration"],
        }


@workflow.defn
class TranscriptionETLWorkflow:
    """ETL workflow: download -> convert -> transcribe."""
    
    @workflow.run
    async def run(self, request: Dict[str, Any]) -> Dict[str, Any]:
        """Execute ETL workflow."""
        url = request["source_url"]
        language = request.get("language", "auto")
        
        workflow.logger.info(f"Starting ETL workflow for URL: {url}")
        
        temp_files = []
        
        try:
            # Step 1: Download video (using activity to avoid sandbox issues)
            download_result = await workflow.execute_activity(
                download_video_activity,
                url,
                request.get("quality", "720p"),
                start_to_close_timeout=timedelta(minutes=10),
            )
            temp_files.append(download_result["file_path"])
            
            # Step 2: Convert to audio
            audio_result = await workflow.execute_activity(
                convert_to_audio_activity,
                download_result["file_path"],
                "wav",
                start_to_close_timeout=timedelta(minutes=5),
            )
            temp_files.append(audio_result["output_path"])
            
            # Step 3: Transcribe
            transcription_result = await workflow.execute_activity(
                transcribe_with_faster_whisper,
                audio_result["output_path"],
                language,
                start_to_close_timeout=timedelta(minutes=30),
            )
            
            # Step 4: Upload to MinIO
            transcription_key = f"etl/{download_result['video_id']}/transcription.txt"
            upload_result = await workflow.execute_activity(
                upload_to_minio_activity,
                audio_result["output_path"],
                transcription_key,
                {
                    "video_id": download_result["video_id"],
                    "title": download_result["title"],
                    "language": transcription_result["language"],
                },
                start_to_close_timeout=timedelta(minutes=2),
            )
            
            return {
                "video_id": download_result["video_id"],
                "title": download_result["title"],
                "transcription_url": upload_result["url"],
                "provider": transcription_result["provider"],
                "language": transcription_result["language"],
            }
            
        finally:
            # Cleanup temp files
            if temp_files:
                await workflow.execute_activity(
                    cleanup_temp_files_activity,
                    temp_files,
                    start_to_close_timeout=timedelta(minutes=1),
                )


async def main():
    """Main entry point."""
    # Create Temporal client
    temporal_host = os.getenv("TEMPORAL_HOST", "127.0.0.1:7233")
    client = await Client.connect(temporal_host)
    
    # Import Runner for custom sandbox restrictions
    from temporalio.worker.workflow_sandbox import Runner, SandboxedWorkflowRunner, SandboxRestrictions
    
    # Create custom restrictions that pass through problematic modules
    restrictions = SandboxRestrictions.default.with_passthrough_modules("yt_dlp", "urllib3", "ffmpeg")
    
    # Create worker with custom runner
    task_queue = os.getenv("TASK_QUEUE", "v2t-transcription-queue")
    worker = Worker(
        client,
        task_queue=task_queue,
        workflows=[SimpleTranscriptionWorkflow, TranscriptionETLWorkflow],
        activities=[
            transcribe_with_faster_whisper,
            upload_to_minio_activity,
            download_from_minio_activity,
            download_video_activity,
            convert_to_audio_activity,
            cleanup_temp_files_activity,
        ],
        max_concurrent_activities=int(os.getenv("MAX_CONCURRENT_ACTIVITIES", "5")),
        workflow_runner=SandboxedWorkflowRunner(restrictions=restrictions),
    )
    
    logger.info(f"Starting Python worker on task queue: {task_queue}")
    logger.info(f"Temporal host: {temporal_host}")
    logger.info(f"Whisper model: {os.getenv('WHISPER_MODEL_SIZE', 'large-v3')}")
    logger.info(f"Device: {os.getenv('WHISPER_DEVICE', 'cpu')}")
    
    # Run worker
    await worker.run()


if __name__ == "__main__":
    asyncio.run(main())