# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is a Go-based CLI tool called `tiktok-whisper` that batch converts videos/audio to text transcriptions using either local whisper.cpp (with coreML acceleration on macOS) or remote OpenAI Whisper API. The project supports downloading content from sources like Xiaoyuzhou podcasts and YouTube, then transcribing them with timestamp-aligned text output.

## Network Configuration

- 请求http时必须使用 ip 访问局域网而不是mac-mini-m4-1.local

## Environment Setup

### API Key Configuration

**Security-first approach using .env files:**
```bash
# Copy the example file
cp .env.example .env

# Edit .env file with your API keys
# Note: .env files are automatically ignored by git for security
```

**Required API Keys:**
- `OPENAI_API_KEY` - For OpenAI text-embedding-ada-002 (1536 dimensions) and Whisper transcription
- `GEMINI_API_KEY` - For Google Gemini embedding-001 (768 dimensions)
- `ELEVENLABS_API_KEY` - For ElevenLabs Speech-to-Text API (optional)

[Rest of the file remains unchanged...]