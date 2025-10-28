# Maestro MCP Docker Guide

This guide explains how to build and run Maestro MCP using Docker.

## Prerequisites

- Docker installed on your system
- Git repository cloned locally

## Quick Start

The easiest way to build and run Maestro MCP in Docker is to use the provided script:

```bash
./docker-build-run.sh
```

This script will:
1. Build the Docker image
2. Stop and remove any existing Maestro MCP container
3. Start a new container with the appropriate settings
4. Mount your local `config.yaml` into the container

Once running, you can access Maestro MCP at: http://localhost:8030

## Manual Docker Commands

If you prefer to run the Docker commands manually:

### Build the Docker image

```bash
docker build -t maestro-mcp:latest .
```

### Run the Docker container

```bash
docker run -d \
  --name maestro-mcp \
  -p 8030:8030 \
  -v "$(pwd)/config.yaml:/app/config.yaml" \
  maestro-mcp:latest
```

### Stop the container

```bash
docker stop maestro-mcp
```

### View container logs

```bash
docker logs maestro-mcp
```

## Configuration

The Docker container uses the `config.yaml` file for configuration. By default, it mounts your local `config.yaml` file into the container.

You can also override configuration settings using environment variables. For example:

```bash
docker run -d \
  --name maestro-mcp \
  -p 8030:8030 \
  -e MAESTRO_MCP_SERVER_PORT=8030 \
  -e MAESTRO_MCP_LOGGING_LEVEL=debug \
  -v "$(pwd)/config.yaml:/app/config.yaml" \
  maestro-mcp:latest
```

See the `.env.example` file for all available environment variables.

## Customizing the Docker Image

If you need to customize the Docker image, you can modify the `Dockerfile` and rebuild:

1. Edit the `Dockerfile`
2. Rebuild the image: `docker build -t maestro-mcp:custom .`
3. Run with your custom image: `docker run -d --name maestro-mcp -p 8030:8030 maestro-mcp:custom`

## Troubleshooting

### Container fails to start

Check the logs for errors:

```bash
docker logs maestro-mcp
```

### Port conflicts

If port 8030 is already in use, you can map to a different port:

```bash
docker run -d --name maestro-mcp -p 8031:8030 maestro-mcp:latest
```

Then access Maestro MCP at http://localhost:8031