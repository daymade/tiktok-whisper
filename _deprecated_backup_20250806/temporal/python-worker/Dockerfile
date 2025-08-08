FROM python:3.11-slim

# Install system dependencies
RUN apt-get update && apt-get install -y \
    ffmpeg \
    git \
    curl \
    && rm -rf /var/lib/apt/lists/*

# Create non-root user
RUN useradd -m -u 1000 v2t

# Set working directory
WORKDIR /app

# Copy requirements first for better caching
COPY requirements.txt .

# Install Python dependencies
RUN pip install --no-cache-dir -r requirements.txt

# Copy worker code
COPY worker.py .
COPY .env* ./

# Create necessary directories
RUN mkdir -p /tmp/v2t-temporal && \
    chown -R v2t:v2t /app /tmp/v2t-temporal

# Switch to non-root user
USER v2t

# Set environment variables
ENV PYTHONUNBUFFERED=1
ENV TEMPORAL_HOST=temporal:7233
ENV TASK_QUEUE=v2t-transcription-queue

# Run the worker
CMD ["python", "worker.py"]