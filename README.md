# Maestro MCP Server

A native Golang implementation of the Model Context Protocol (MCP) server for
Maestro and Maestro Knowledge. This server provides vector database operations
through a standardized MCP interface, supporting both Milvus and Weaviate vector
databases.

## Features

- **Vector Database Support**: Milvus and Weaviate integration
- **MCP Protocol**: Full Model Context Protocol implementation
- **Embedding Support**: OpenAI and custom local embedding services
- **Mock Database**: Built-in mock for testing and development
- **Comprehensive Testing**: Unit tests, integration tests, and end-to-end tests
- **Production Ready**: Graceful shutdown, health checks, and logging
- **Configuration**: YAML and environment variable support

## Quick Start

### Prerequisites

- Go 1.21 or later
- Git

### Installation

1. **Clone the repository**:

   ```bash
   git clone https://github.com/maximilien/maestro-mcp.git
   cd maestro-mcp
   ```

2. **Set up environment**:

   ```bash
   cp .env.example .env
   # Edit .env with your configuration
   ```

3. **Build the server**:

   ```bash
   ./build.sh
   ```

4. **Run tests**:

   ```bash
   ./test.sh
   ```

5. **Start the server**:

   ```bash
   ./start.sh
   ```

The server will start on `http://localhost:8030` by default.

## Configuration

### Environment Variables

The server uses environment variables with the `MAESTRO_MCP_` prefix. Key
configuration options:

```bash
# Server Configuration
MAESTRO_MCP_SERVER_HOST=localhost
MAESTRO_MCP_SERVER_PORT=8030

# Vector Database Configuration
MAESTRO_MCP_VECTOR_DB_TYPE=milvus  # or weaviate

# Milvus Configuration
MAESTRO_MCP_VECTOR_DB_MILVUS_HOST=localhost
MAESTRO_MCP_VECTOR_DB_MILVUS_PORT=19530

# Weaviate Configuration
MAESTRO_MCP_VECTOR_DB_WEAVIATE_URL=http://localhost:8080
MAESTRO_MCP_VECTOR_DB_WEAVIATE_API_KEY=your_api_key

# Embedding Configuration
MAESTRO_MCP_EMBEDDING_PROVIDER=openai
MAESTRO_MCP_EMBEDDING_MODEL=text-embedding-ada-002
MAESTRO_MCP_EMBEDDING_API_KEY=your_openai_api_key

# Logging Configuration
MAESTRO_MCP_LOGGING_LEVEL=info
MAESTRO_MCP_LOGGING_FORMAT=json
```

### Configuration File

You can also use a `config.yaml` file:

```yaml
version: "0.0.1"

server:
  host: "localhost"
  port: 8030

mcp:
  tool_timeout: "15s"
  vector_db:
    type: "milvus"
    milvus:
      host: "localhost"
      port: 19530
  embedding:
    provider: "openai"
    model: "text-embedding-ada-002"
```

## Available Tools

The MCP server provides the following tools:

### Database Management

- `create_vector_database`: Create a new vector database instance
- `list_databases`: List all available vector database instances
- `setup_database`: Set up a vector database and create collections
- `cleanup`: Clean up resources and close connections

### Document Operations

- `write_document`: Write a single document to a vector database
- `write_documents`: Write multiple documents to a vector database
- `list_documents`: List documents from a vector database
- `count_documents`: Get the count of documents in a collection
- `delete_document`: Delete a single document by ID
- `delete_documents`: Delete multiple documents by IDs

### Query Operations

- `query`: Query documents using natural language
- `search`: Perform vector similarity search

### Collection Management

- `list_collections`: List all collections in a vector database
- `get_collection_info`: Get information about a collection
- `create_collection`: Create a new collection
- `delete_collection`: Delete a collection

## Usage Examples

### Using curl

1. **List available tools**:

   ```bash
   curl http://localhost:8030/mcp/tools/list
   ```

2. **Create a vector database**:

   ```bash
   curl -X POST http://localhost:8030/mcp/tools/call \
     -H "Content-Type: application/json" \
     -d '{
       "name": "create_vector_database",
       "arguments": {
         "db_name": "my_db",
         "db_type": "milvus",
         "collection_name": "documents"
       }
     }'
   ```

3. **Write a document**:

   ```bash
   curl -X POST http://localhost:8030/mcp/tools/call \
     -H "Content-Type: application/json" \
     -d '{
       "name": "write_document",
       "arguments": {
         "db_name": "my_db",
         "url": "https://example.com/doc1",
         "text": "This is a test document about machine learning.",
         "metadata": {
           "author": "John Doe",
           "category": "AI"
         }
       }
     }'
   ```

4. **Query documents**:

   ```bash
   curl -X POST http://localhost:8030/mcp/tools/call \
     -H "Content-Type: application/json" \
     -d '{
       "name": "query",
       "arguments": {
         "db_name": "my_db",
         "query": "What is machine learning?",
         "limit": 5
       }
     }'
   ```

### Using MCP Client

Add to your MCP client configuration:

```json
{
  "mcpServers": {
    "maestro-mcp": {
      "command": "./bin/maestro-mcp",
      "args": [],
      "env": {
        "MAESTRO_MCP_SERVER_HOST": "localhost",
        "MAESTRO_MCP_SERVER_PORT": "8030"
      }
    }
  }
}
```

## Development

### Project Structure

```text
maestro-mcp/
├── src/                    # Source code
│   ├── main.go            # Main entry point
│   └── pkg/               # Packages
│       ├── config/        # Configuration management
│       ├── mcp/           # MCP server implementation
│       ├── server/         # HTTP server
│       └── vectordb/       # Vector database implementations
├── tests/                 # Test files
├── .github/               # GitHub Actions workflows
│   └── workflows/
│       ├── ci.yml         # Continuous Integration
│       └── release.yml    # Release automation
├── config.yaml           # Configuration file
├── .env.example          # Environment variables example
├── CHANGELOG.md          # Changelog
├── build.sh              # Build script
├── test.sh               # Test script
├── start.sh              # Start script
├── stop.sh               # Stop script
├── lint.sh               # Linting script
└── e2e.sh                # End-to-end test script
```

### Building

```bash
# Build the server
./build.sh

# Build with clean
./build.sh clean
```

### Testing

```bash
# Run unit tests
./test.sh unit

# Run integration tests
./test.sh integration

# Run all tests
./test.sh all

# Run tests with coverage
./test.sh coverage
```

### Linting

```bash
# Run linter
./lint.sh

# Skip security checks
./lint.sh --skip-security
```

### End-to-End Testing

```bash
# Run full E2E test suite
./e2e.sh
```

## CI/CD

This project uses GitHub Actions for continuous integration and deployment.

### Continuous Integration

The CI pipeline runs on every push and pull request and includes:

- **Linting**: Code quality checks with golangci-lint, gofmt, and shellcheck
- **Building**: Compilation for multiple platforms (Linux, macOS, Windows)
- **Testing**: Unit tests, integration tests, and end-to-end tests
- **Security**: Vulnerability scanning with gosec and govulncheck
- **Matrix Testing**: Cross-platform testing with multiple Go versions

### Release Automation

The release workflow automatically:

- **Triggers**: On git tags (e.g., `v1.0.0`) or manual dispatch
- **Tests**: Runs full test suite before release
- **Builds**: Creates binaries for multiple platforms
- **Packages**: Generates checksums and release notes
- **Publishes**: Creates GitHub release with downloadable assets

### Workflow Files

- `.github/workflows/ci.yml`: Main CI pipeline
- `.github/workflows/release.yml`: Release automation

### Status Badges

[![CI](https://github.com/maximilien/maestro-mcp/workflows/CI/badge.svg)](https://github.com/maximilien/maestro-mcp/actions)
[![Release](https://github.com/maximilien/maestro-mcp/workflows/Release/badge.svg)](https://github.com/maximilien/maestro-mcp/actions)

### Server Management

```bash
# Start server
./start.sh

# Start in daemon mode
./start.sh --daemon

# Start with build
./start.sh --build

# Check server status
./stop.sh status

# Stop server
./stop.sh

# Restart server
./stop.sh restart
```

## Vector Database Support

### Milvus

Milvus is a vector database designed for scalable similarity search and AI
applications.

**Configuration**:

```bash
MAESTRO_MCP_VECTOR_DB_TYPE=milvus
MAESTRO_MCP_VECTOR_DB_MILVUS_HOST=localhost
MAESTRO_MCP_VECTOR_DB_MILVUS_PORT=19530
MAESTRO_MCP_VECTOR_DB_MILVUS_USERNAME=root
MAESTRO_MCP_VECTOR_DB_MILVUS_PASSWORD=password
```

### Weaviate

Weaviate is an open-source vector database that allows you to store data objects
and vector embeddings.

**Configuration**:

```bash
MAESTRO_MCP_VECTOR_DB_TYPE=weaviate
MAESTRO_MCP_VECTOR_DB_WEAVIATE_URL=http://localhost:8080
MAESTRO_MCP_VECTOR_DB_WEAVIATE_API_KEY=your_api_key
```

### Mock Database

For testing and development, the server includes a mock vector database that
simulates all operations without requiring external dependencies.

## Embedding Support

### OpenAI Embeddings

```bash
MAESTRO_MCP_EMBEDDING_PROVIDER=openai
MAESTRO_MCP_EMBEDDING_MODEL=text-embedding-ada-002
MAESTRO_MCP_EMBEDDING_API_KEY=your_openai_api_key
```

### Custom Local Embeddings

```bash
MAESTRO_MCP_EMBEDDING_PROVIDER=custom_local
MAESTRO_MCP_EMBEDDING_URL=http://localhost:8000/embed
MAESTRO_MCP_EMBEDDING_MODEL=nomic-embed-text
MAESTRO_MCP_EMBEDDING_VECTOR_SIZE=768
MAESTRO_MCP_EMBEDDING_API_KEY=your_custom_api_key
```

## API Endpoints

### Health Check

```http
GET /health
```

Returns server health status and active vector databases.

### List Tools

```http
GET /mcp/tools/list
```

Returns all available MCP tools with their schemas.

### Call Tool

```http
POST /mcp/tools/call
Content-Type: application/json

{
  "name": "tool_name",
  "arguments": {
    "param1": "value1",
    "param2": "value2"
  }
}
```

Executes an MCP tool with the provided arguments.

## Error Handling

The server provides comprehensive error handling:

- **Validation Errors**: Invalid arguments or missing required parameters
- **Database Errors**: Connection failures, query errors, etc.
- **Timeout Errors**: Operations that exceed configured timeouts
- **Resource Errors**: Memory, disk, or network issues

All errors are returned in a consistent JSON format:

```json
{
  "error": "Error message describing what went wrong"
}
```

## Logging

The server uses structured JSON logging with configurable levels:

- `debug`: Detailed debugging information
- `info`: General information about operations
- `warn`: Warning messages for non-critical issues
- `error`: Error messages for failures

Log output can be configured to stdout, stderr, or files.

## Performance

The server is designed for high performance:

- **Concurrent Operations**: Multiple vector databases can be managed
  simultaneously
- **Connection Pooling**: Efficient database connection management
- **Timeout Protection**: All operations have configurable timeouts
- **Memory Efficient**: Minimal memory footprint with proper cleanup

## Security

Security features include:

- **Input Validation**: All inputs are validated before processing
- **SQL Injection Protection**: Parameterized queries and input sanitization
- **Rate Limiting**: Configurable rate limits for API endpoints
- **Authentication**: Support for API keys and custom authentication

## Releases

### Download

Download the latest release from the
[Releases page](https://github.com/maximilien/maestro-mcp/releases).

### Supported Platforms

- **Linux**: amd64, arm64
- **macOS**: amd64, arm64 (Apple Silicon)
- **Windows**: amd64

### Binary Installation

1. Download the appropriate binary for your platform
2. Make it executable: `chmod +x maestro-mcp-*`
3. Run: `./maestro-mcp-*`

### Verification

Verify the integrity of your download using the checksums provided in each
release:

```bash
sha256sum -c checksums.txt
```

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests for new functionality
5. Run the test suite: `./test.sh all`
6. Run the linter: `./lint.sh`
7. Ensure CI passes
8. Submit a pull request

## License

This project is licensed under the MIT License - see the
[LICENSE](LICENSE) file for details.

## Support

For support and questions:

- Create an issue on GitHub
- Check the documentation
- Review the test examples

## Changelog

### v0.0.3

- Official Milvus client integration
- Official Weaviate client integration
- Updated client initialization with configuration
- Updated Setup methods to use official client schemas

### v0.0.2

- Removed main executable since we have bin directory

### v0.0.1

- Initial release
- Milvus and Weaviate support
- Mock database for testing
- Comprehensive test suite
- MCP protocol implementation
- Configuration management
- Production-ready features
