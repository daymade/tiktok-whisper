# 02 - Features

This section documents the main features and their implementations.

## Documents

- **[DUAL_EMBEDDING_TDD_PLAN.md](DUAL_EMBEDDING_TDD_PLAN.md)** - Dual embedding system (OpenAI + Gemini) for similarity search
- **[TRACKPAD_GESTURE_SYSTEM.md](TRACKPAD_GESTURE_SYSTEM.md)** - Advanced trackpad gesture system for 3D visualization
- **[IMPLEMENTATION_SUMMARY.md](IMPLEMENTATION_SUMMARY.md)** - Summary of implemented features and their status

## Key Features

### 1. Dual Embedding System
- OpenAI text-embedding-3-small (1536 dimensions)
- Google Gemini text-embedding-004 (768 dimensions)
- PostgreSQL pgvector for efficient similarity search
- Duplicate detection and clustering capabilities

### 2. Advanced UI/UX
- Jon Ive-level trackpad gestures
- 3D visualization of embeddings
- Real-time similarity search
- Natural momentum and physics-based animations

### 3. Transcription Capabilities
- Batch processing of audio/video files
- Multiple provider support
- Progress tracking and error handling
- Database persistence with full-text search