# Multi-stage build for v2t API
FROM golang:1.23-alpine AS builder

# Install build dependencies
RUN apk add --no-cache gcc musl-dev sqlite-dev

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the binary for native platform
RUN CGO_ENABLED=1 go build -o v2t ./cmd/v2t/main.go

# Runtime stage
FROM alpine:latest

# Install runtime dependencies
RUN apk add --no-cache ca-certificates ffmpeg

# Create app user
RUN adduser -D -g '' appuser

WORKDIR /app

# Copy binary and required files from builder
COPY --from=builder /app/v2t /app/v2t
COPY --from=builder /app/providers.docker.yaml /app/providers.yaml
COPY --from=builder /app/go.mod /app/go.mod

# Create necessary directories
RUN mkdir -p /app/data /app/logs && \
    chown -R appuser:appuser /app

USER appuser

# Expose API port
EXPOSE 8085

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD wget --no-verbose --tries=1 --spider http://localhost:8085/health || exit 1

# Run the API server
CMD ["./v2t", "api", "--port", "8085", "--host", "0.0.0.0"]