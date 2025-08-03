# 01 - System Architecture

This section contains documentation about the overall system architecture and design.

## Documents

- **[PROVIDER_FRAMEWORK_ARCHITECTURE.md](PROVIDER_FRAMEWORK_ARCHITECTURE.md)** - Comprehensive provider framework design and implementation
- **[SSH_WHISPER_PROVIDER.md](SSH_WHISPER_PROVIDER.md)** - SSH-based remote whisper provider architecture
- **[WHISPER_SERVER_PROVIDER.md](WHISPER_SERVER_PROVIDER.md)** - HTTP whisper-server provider architecture

## Overview

The tiktok-whisper system uses a flexible provider framework that supports multiple transcription backends:
- Local whisper.cpp execution
- Remote SSH-based whisper execution
- HTTP-based whisper server integration
- Cloud APIs (OpenAI, ElevenLabs)

Each provider implements a common interface, allowing seamless switching between different transcription methods.