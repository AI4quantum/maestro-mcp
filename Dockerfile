# Build stage
FROM golang:1.24 AS builder

# Set working directory
WORKDIR /app

# Copy go.mod and go.sum files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy the source code
COPY . .

# Build the application
RUN ./build.sh

# Final stage
FROM debian:bookworm-slim

# Set working directory
WORKDIR /app

# Install necessary runtime dependencies
RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates \
    && rm -rf /var/lib/apt/lists/*

# Copy the binary from the builder stage
COPY --from=builder /app/bin/maestro-mcp /app/maestro-mcp

# Copy configuration files
COPY --from=builder /app/config.yaml /app/config.yaml

# Create directory for database files if needed
RUN mkdir -p /app/data

# Set environment variables
ENV MAESTRO_MCP_SERVER_HOST=0.0.0.0
ENV MAESTRO_MCP_SERVER_PORT=8030

# Expose the port
EXPOSE 8030

# Set the entry point
ENTRYPOINT ["/app/maestro-mcp"]