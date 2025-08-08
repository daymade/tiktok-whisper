# Docker Compose Profiles Guide

The unified `docker-compose.yml` uses Docker profiles to support different deployment modes.

## Available Profiles

- **single-node**: Minimal setup for development/testing (default)
- **distributed**: Full multi-node setup with MinIO cluster
- **monitoring**: Adds Prometheus and Grafana

## Usage Examples

### Single Node Setup (Development)

```bash
# Start single-node setup
docker-compose --profile single-node up -d

# Or use the shorthand (single-node is default)
docker-compose up -d
```

Services started:
- PostgreSQL (port 5434)
- Temporal Server (port 7233)
- Temporal UI (port 8088)
- MinIO single instance (ports 9000, 9001)

### Distributed Setup (Production)

```bash
# Start distributed setup
docker-compose --profile distributed up -d
```

Services started:
- PostgreSQL (port 5434)
- Temporal Server (port 7233)
- Temporal UI (port 8088)
- MinIO 3-node cluster
- Nginx load balancer for MinIO (ports 9000, 9001)

### With Monitoring

```bash
# Single node with monitoring
docker-compose --profile single-node --profile monitoring up -d

# Distributed with monitoring
docker-compose --profile distributed --profile monitoring up -d
```

Additional services:
- Prometheus (port 9090)
- Grafana (port 3000)

## Environment Variables

```bash
# PostgreSQL port (default: 5434)
POSTGRES_PORT=5434

# MinIO credentials
MINIO_ROOT_USER=minioadmin
MINIO_ROOT_PASSWORD=minioadmin

# Grafana admin password
GRAFANA_PASSWORD=admin
```

## Stopping Services

```bash
# Stop all services
docker-compose down

# Stop and remove volumes
docker-compose down -v
```