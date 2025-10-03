# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.0.3] - 2025-01-02

### Added

- Official Milvus client integration (github.com/milvus-io/milvus/client/v2/milvusclient)
- Official Weaviate client integration (github.com/weaviate/weaviate-go-client/v5/weaviate)
- Proper client initialization with configuration
- Updated Setup methods to use official client schemas

### Changed

- MilvusDatabase now uses milvusclient.Client instead of custom interface
- WeaviateDatabase now uses weaviate.Client instead of custom interface
- Updated go.mod with official client dependencies

### Technical Notes

- Using placeholder implementations for now due to dependency conflicts
- Official client APIs are integrated and ready for full implementation

## [0.0.2] - 2025-01-02

### Changes

- Removed main executable since we have bin directory

## [0.0.1] - 2025-01-02

### Features Added

- Initial release of Maestro MCP Server
- Support for Milvus and Weaviate vector databases
- Mock database for testing and development
- Comprehensive MCP tool set for vector database operations
- Configuration management with YAML and environment variables
- Comprehensive test suite with unit, integration, and end-to-end tests
- Production-ready features including health checks, logging, and graceful shutdown
- Build, test, lint, and deployment scripts
- GitHub Actions CI/CD workflows

### Features

- **Vector Database Support**: Milvus and Weaviate integration
- **MCP Protocol**: Full Model Context Protocol implementation
- **Embedding Support**: OpenAI and custom local embedding services
- **Mock Database**: Built-in mock for testing and development
- **Comprehensive Testing**: Unit tests, integration tests, and end-to-end tests
- **Production Ready**: Graceful shutdown, health checks, and logging
- **Configuration**: YAML and environment variable support

### Tools Available

- `create_vector_database`: Create a new vector database instance
- `list_databases`: List all available vector database instances
- `setup_database`: Set up a vector database and create collections
- `write_document`: Write a single document to a vector database
- `write_documents`: Write multiple documents to a vector database
- `list_documents`: List documents from a vector database
- `count_documents`: Get the count of documents in a collection
- `query`: Query documents using natural language
- `search`: Perform vector similarity search
- `delete_document`: Delete a single document by ID
- `delete_documents`: Delete multiple documents by IDs
- `list_collections`: List all collections in a vector database
- `get_collection_info`: Get information about a collection
- `create_collection`: Create a new collection
- `delete_collection`: Delete a collection
- `cleanup`: Clean up resources and close connections
