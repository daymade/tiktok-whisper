#!/usr/bin/env python3
"""
Activities-only Python Worker (no workflows to avoid sandbox issues)
"""

import asyncio
import logging
import os
from datetime import timedelta

from dotenv import load_dotenv
from faster_whisper import WhisperModel
from minio import Minio
from temporalio import activity
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


async def main():
    """Main entry point."""
    # Create Temporal client
    temporal_host = os.getenv("TEMPORAL_HOST", "127.0.0.1:7233")
    client = await Client.connect(temporal_host)
    
    # Create worker with activities only
    task_queue = os.getenv("TASK_QUEUE", "v2t-transcription-queue")
    worker = Worker(
        client,
        task_queue=task_queue,
        workflows=[],  # No workflows, only activities
        activities=[transcribe_with_faster_whisper],
        max_concurrent_activities=int(os.getenv("MAX_CONCURRENT_ACTIVITIES", "5")),
    )
    
    logger.info(f"Starting Python activities-only worker on task queue: {task_queue}")
    logger.info(f"Temporal host: {temporal_host}")
    logger.info(f"Whisper model: {os.getenv('WHISPER_MODEL_SIZE', 'large-v3')}")
    logger.info(f"Device: {os.getenv('WHISPER_DEVICE', 'cpu')}")
    
    # Run worker
    await worker.run()


if __name__ == "__main__":
    asyncio.run(main())