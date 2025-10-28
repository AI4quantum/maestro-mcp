#!/usr/bin/env bash

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

print_status() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

print_header() {
    echo -e "${BLUE}[DOCKER]${NC} $1"
}

# Build the Docker image
build_image() {
    print_header "Building Maestro MCP Docker image..."
    docker build -t maestro-mcp:latest .
    
    if [ $? -eq 0 ]; then
        print_status "Docker image built successfully!"
    else
        print_error "Failed to build Docker image"
        exit 1
    fi
}

# Run the Docker container
run_container() {
    print_header "Running Maestro MCP Docker container..."
    
    # Check if container is already running
    if docker ps | grep -q maestro-mcp; then
        print_warning "Maestro MCP container is already running"
        print_status "Stopping existing container..."
        docker stop maestro-mcp
    fi
    
    # Remove existing container if it exists
    if docker ps -a | grep -q maestro-mcp; then
        print_status "Removing existing container..."
        docker rm maestro-mcp
    fi
    
    # Run the container
    print_status "Starting new container..."
    docker run -d \
        --name maestro-mcp \
        -p 8030:8030 \
        -v "$(pwd)/config.yaml:/app/config.yaml" \
        maestro-mcp:latest
    
    if [ $? -eq 0 ]; then
        print_status "Container started successfully!"
        print_status "Maestro MCP is running at http://localhost:8030"
    else
        print_error "Failed to start container"
        exit 1
    fi
}

# Main execution
print_header "Maestro MCP Docker Build & Run"
echo "====================================="

# Build the image
build_image

# Run the container
run_container

print_header "Done! 🎉"
echo "====================================="
print_status "Maestro MCP is now running in Docker"
print_status "Access the server at: http://localhost:8030"
print_status "To stop the container: docker stop maestro-mcp"
print_status "To view logs: docker logs maestro-mcp"

# Made with Bob
