# TikTok Whisper RESTful API

This RESTful API provides HTTP endpoints that mirror the CLI functionality of tiktok-whisper, following DDD/TDD/SOLID principles.

## Architecture

The API follows Domain-Driven Design (DDD) with clear separation of concerns:

```
internal/api/
├── server/          # API server setup and configuration
├── v1/              # Version 1 API implementation
│   ├── handlers/    # HTTP handlers (Presentation Layer)
│   ├── services/    # Business logic (Application Layer)
│   ├── dto/         # Data Transfer Objects
│   └── routes/      # Route definitions
├── middleware/      # Cross-cutting concerns
│   ├── cors.go      # CORS handling
│   ├── error_handler.go  # Centralized error handling
│   ├── logging.go   # Structured logging
│   ├── request_id.go     # Request ID tracking
│   └── validation.go     # Request validation
└── errors/          # Structured error types
```

## Running the API Server

```bash
# Start with default settings (port 8080)
v2t api

# Custom configuration
v2t api --port 3000 --host 0.0.0.0 --env production

# With timeouts
v2t api --read-timeout 60 --write-timeout 60 --idle-timeout 300
```

## API Endpoints

### Health Check
- `GET /health` - Server health status

### Transcriptions (maps to `convert` command)
- `POST /api/v1/transcriptions` - Create new transcription job
- `GET /api/v1/transcriptions/:id` - Get transcription by ID
- `GET /api/v1/transcriptions` - List transcriptions with filtering
- `DELETE /api/v1/transcriptions/:id` - Delete transcription

### Providers (maps to `providers` command)
- `GET /api/v1/providers` - List all providers
- `GET /api/v1/providers/:id` - Get provider details
- `GET /api/v1/providers/:id/status` - Health check specific provider
- `GET /api/v1/providers/:id/stats` - Get provider usage statistics
- `POST /api/v1/providers/:id/test` - Test provider functionality

## Request/Response Examples

### Create Transcription
```bash
curl -X POST http://localhost:8080/api/v1/transcriptions \
  -H "Content-Type: application/json" \
  -d '{
    "file_path": "/path/to/audio.mp3",
    "provider": "openai/whisper",
    "language": "en",
    "output_format": "json"
  }'
```

Response:
```json
{
  "id": 123,
  "user_id": "",
  "file_path": "/path/to/audio.mp3",
  "status": "pending",
  "provider": "openai/whisper",
  "language": "en",
  "created_at": "2024-01-01T12:00:00Z",
  "updated_at": "2024-01-01T12:00:00Z"
}
```

### List Providers
```bash
curl http://localhost:8080/api/v1/providers
```

Response:
```json
{
  "providers": [
    {
      "id": "openai/whisper",
      "name": "OpenAI Whisper",
      "description": "OpenAI's Whisper API for speech recognition",
      "type": "remote",
      "available": true,
      "health_status": "healthy",
      "is_default": true,
      "capabilities": {
        "supports_streaming": false,
        "supports_languages": ["en", "es", "fr", "de", "it", "pt", "ru", "ja", "ko"],
        "max_file_size_mb": 25,
        "max_duration_sec": 7200
      }
    }
  ]
}
```

## Error Handling

The API uses structured error responses:

```json
{
  "kind": "validation",
  "message": "Validation failed",
  "details": {
    "file_path": "is required"
  },
  "request_id": "550e8400-e29b-41d4-a716-446655440000"
}
```

Error kinds:
- `validation` - Input validation errors (422)
- `bad_request` - Bad request format (400)
- `not_found` - Resource not found (404)
- `unauthorized` - Authentication required (401)
- `forbidden` - Insufficient permissions (403)
- `conflict` - Resource conflict (409)
- `internal` - Internal server error (500)
- `service_unavailable` - Service unavailable (503)

## Middleware

### Request ID
Every request is assigned a unique ID for tracking:
- Header: `X-Request-ID`
- Included in all error responses
- Logged with all requests

### CORS
Default CORS configuration allows all origins. Configure as needed for production.

### Structured Logging
All requests are logged with:
- Request ID
- Method and path
- Status code
- Response time
- Client IP
- User agent

## Design Principles

### SOLID Principles
- **Single Responsibility**: Each handler manages one resource type
- **Open/Closed**: Extensible through interfaces (new providers, exporters)
- **Liskov Substitution**: All providers implement TranscriptionProvider interface
- **Interface Segregation**: Focused interfaces for different capabilities
- **Dependency Inversion**: Handlers depend on service interfaces, not implementations

### Test-Driven Development (TDD)
- Integration tests written before implementation
- Mock services for isolated testing
- Table-driven test approach
- High test coverage

### Domain-Driven Design (DDD)
- Clear separation between layers
- Rich domain models
- DTOs for API contracts
- Repository pattern for data access

## Future Enhancements

The following endpoints are planned but not yet implemented:
- Downloads API (`/api/v1/downloads`)
- Embeddings API (`/api/v1/embeddings`)
- Export API (`/api/v1/exports`)
- Configuration API (`/api/v1/config`)
- Authentication and authorization middleware
- Rate limiting
- API key management
- WebSocket support for real-time transcription status
- OpenAPI/Swagger documentation