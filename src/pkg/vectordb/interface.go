package vectordb

import (
	"context"
	"fmt"

	"github.com/AI4quantum/maestro-mcp/src/pkg/config"
)

// VectorDatabase defines the interface for vector database operations
type VectorDatabase interface {
	// Type returns the database type (e.g., "milvus", "weaviate")
	Type() string

	// CollectionName returns the current collection name
	CollectionName() string

	// Setup initializes the database and creates collections
	Setup(ctx context.Context, embedding string) error

	// WriteDocument writes a single document to the database
	WriteDocument(ctx context.Context, doc Document) (WriteStats, error)

	// WriteDocuments writes multiple documents to the database
	WriteDocuments(ctx context.Context, docs []Document) (WriteStats, error)

	// Query performs a natural language query on the database
	Query(ctx context.Context, query string, limit int, collectionName string) (interface{}, error)

	// Search performs a vector similarity search
	Search(ctx context.Context, query string, limit int, collectionName string) ([]SearchResult, error)

	// ListDocuments lists documents from the database
	ListDocuments(ctx context.Context, limit, offset int) ([]Document, error)

	// CountDocuments returns the count of documents in the database
	CountDocuments(ctx context.Context) (int, error)

	// DeleteDocument deletes a document by ID
	DeleteDocument(ctx context.Context, documentID string) error

	// DeleteDocuments deletes multiple documents by IDs
	DeleteDocuments(ctx context.Context, documentIDs []string) error

	// ListCollections lists all collections in the database
	ListCollections(ctx context.Context) ([]string, error)

	// GetCollectionInfo returns information about a collection
	GetCollectionInfo(ctx context.Context, collectionName string) (map[string]interface{}, error)

	// DeleteCollection deletes a collection
	DeleteCollection(ctx context.Context, collectionName string) error

	// Cleanup cleans up resources and closes connections
	Cleanup(ctx context.Context) error
}

// Document represents a document in the vector database
type Document struct {
	ID       string                 `json:"id,omitempty"`
	URL      string                 `json:"url"`
	Text     string                 `json:"text"`
	Metadata map[string]interface{} `json:"metadata"`
	Vector   []float64              `json:"vector,omitempty"`
}

// SearchResult represents a search result
type SearchResult struct {
	Document Document `json:"document"`
	Score    float64  `json:"score"`
}

// WriteStats represents statistics from a write operation
type WriteStats struct {
	DocumentsWritten int      `json:"documents_written"`
	ProcessingTime   string   `json:"processing_time"`
	Errors           []string `json:"errors,omitempty"`
}

// CreateVectorDatabase creates a new vector database instance
func CreateVectorDatabase(dbType, collectionName string, cfg *config.Config) (VectorDatabase, error) {
	switch dbType {
	case "milvus":
		return NewMilvusDatabase(collectionName, cfg)
	case "weaviate":
		return NewWeaviateDatabase(collectionName, cfg)
	default:
		return nil, fmt.Errorf("unsupported vector database type: %s", dbType)
	}
}
