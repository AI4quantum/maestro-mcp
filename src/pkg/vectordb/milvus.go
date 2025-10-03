package vectordb

import (
	"context"
	"fmt"
	"time"

	"github.com/AI4quantum/maestro-mcp/src/pkg/config"
	"go.uber.org/zap"
)

// MilvusDatabase implements VectorDatabase for Milvus
type MilvusDatabase struct {
	config         *config.Config
	logger         *zap.Logger
	collectionName string
	client         MilvusClient
}

// MilvusClient defines the interface for Milvus client operations
type MilvusClient interface {
	Connect(ctx context.Context) error
	CreateCollection(ctx context.Context, name string, schema map[string]interface{}) error
	Insert(ctx context.Context, collectionName string, documents []Document) error
	Search(ctx context.Context, collectionName string, query string, limit int) ([]SearchResult, error)
	Query(ctx context.Context, collectionName string, query string, limit int) (interface{}, error)
	ListDocuments(ctx context.Context, collectionName string, limit, offset int) ([]Document, error)
	CountDocuments(ctx context.Context, collectionName string) (int, error)
	DeleteDocument(ctx context.Context, collectionName string, documentID string) error
	DeleteDocuments(ctx context.Context, collectionName string, documentIDs []string) error
	ListCollections(ctx context.Context) ([]string, error)
	GetCollectionInfo(ctx context.Context, collectionName string) (map[string]interface{}, error)
	DeleteCollection(ctx context.Context, collectionName string) error
	Close() error
}

// NewMilvusDatabase creates a new Milvus database instance
func NewMilvusDatabase(collectionName string, cfg *config.Config) (*MilvusDatabase, error) {
	logger, _ := zap.NewProduction()

	db := &MilvusDatabase{
		config:         cfg,
		logger:         logger,
		collectionName: collectionName,
		client:         NewMockMilvusClient(), // Use mock for now
	}

	return db, nil
}

// Type returns the database type
func (m *MilvusDatabase) Type() string {
	return "milvus"
}

// CollectionName returns the current collection name
func (m *MilvusDatabase) CollectionName() string {
	return m.collectionName
}

// Setup initializes the database and creates collections
func (m *MilvusDatabase) Setup(ctx context.Context, embedding string) error {
	if err := m.client.Connect(ctx); err != nil {
		return fmt.Errorf("failed to connect to Milvus: %w", err)
	}

	// Create collection schema
	schema := map[string]interface{}{
		"name": m.collectionName,
		"fields": []map[string]interface{}{
			{
				"name":    "id",
				"type":    "string",
				"primary": true,
			},
			{
				"name": "url",
				"type": "string",
			},
			{
				"name": "text",
				"type": "string",
			},
			{
				"name": "metadata",
				"type": "json",
			},
			{
				"name":      "vector",
				"type":      "float_vector",
				"dimension": m.config.MCP.Embedding.VectorSize,
			},
		},
		"embedding": embedding,
	}

	if err := m.client.CreateCollection(ctx, m.collectionName, schema); err != nil {
		return fmt.Errorf("failed to create collection: %w", err)
	}

	m.logger.Info("Set up Milvus collection",
		zap.String("collection", m.collectionName),
		zap.String("embedding", embedding))

	return nil
}

// WriteDocument writes a single document to the database
func (m *MilvusDatabase) WriteDocument(ctx context.Context, doc Document) (WriteStats, error) {
	start := time.Now()

	stats, err := m.WriteDocuments(ctx, []Document{doc})
	if err != nil {
		return WriteStats{}, err
	}

	stats.ProcessingTime = time.Since(start).String()
	return stats, nil
}

// WriteDocuments writes multiple documents to the database
func (m *MilvusDatabase) WriteDocuments(ctx context.Context, docs []Document) (WriteStats, error) {
	start := time.Now()

	if err := m.client.Insert(ctx, m.collectionName, docs); err != nil {
		return WriteStats{}, fmt.Errorf("failed to insert documents: %w", err)
	}

	processingTime := time.Since(start)

	m.logger.Info("Wrote documents to Milvus",
		zap.String("collection", m.collectionName),
		zap.Int("count", len(docs)),
		zap.Duration("processing_time", processingTime))

	return WriteStats{
		DocumentsWritten: len(docs),
		ProcessingTime:   processingTime.String(),
	}, nil
}

// Query performs a natural language query on the database
func (m *MilvusDatabase) Query(ctx context.Context, query string, limit int, collectionName string) (interface{}, error) {
	if collectionName == "" {
		collectionName = m.collectionName
	}

	result, err := m.client.Query(ctx, collectionName, query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query Milvus: %w", err)
	}

	m.logger.Info("Executed query on Milvus",
		zap.String("collection", collectionName),
		zap.String("query", query),
		zap.Int("limit", limit))

	return result, nil
}

// Search performs a vector similarity search
func (m *MilvusDatabase) Search(ctx context.Context, query string, limit int, collectionName string) ([]SearchResult, error) {
	if collectionName == "" {
		collectionName = m.collectionName
	}

	results, err := m.client.Search(ctx, collectionName, query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to search Milvus: %w", err)
	}

	m.logger.Info("Executed search on Milvus",
		zap.String("collection", collectionName),
		zap.String("query", query),
		zap.Int("limit", limit),
		zap.Int("results", len(results)))

	return results, nil
}

// ListDocuments lists documents from the database
func (m *MilvusDatabase) ListDocuments(ctx context.Context, limit, offset int) ([]Document, error) {
	documents, err := m.client.ListDocuments(ctx, m.collectionName, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list documents from Milvus: %w", err)
	}

	m.logger.Info("Listed documents from Milvus",
		zap.String("collection", m.collectionName),
		zap.Int("limit", limit),
		zap.Int("offset", offset),
		zap.Int("count", len(documents)))

	return documents, nil
}

// CountDocuments returns the count of documents in the database
func (m *MilvusDatabase) CountDocuments(ctx context.Context) (int, error) {
	count, err := m.client.CountDocuments(ctx, m.collectionName)
	if err != nil {
		return 0, fmt.Errorf("failed to count documents in Milvus: %w", err)
	}

	m.logger.Info("Counted documents in Milvus",
		zap.String("collection", m.collectionName),
		zap.Int("count", count))

	return count, nil
}

// DeleteDocument deletes a document by ID
func (m *MilvusDatabase) DeleteDocument(ctx context.Context, documentID string) error {
	if err := m.client.DeleteDocument(ctx, m.collectionName, documentID); err != nil {
		return fmt.Errorf("failed to delete document from Milvus: %w", err)
	}

	m.logger.Info("Deleted document from Milvus",
		zap.String("collection", m.collectionName),
		zap.String("document_id", documentID))

	return nil
}

// DeleteDocuments deletes multiple documents by IDs
func (m *MilvusDatabase) DeleteDocuments(ctx context.Context, documentIDs []string) error {
	if err := m.client.DeleteDocuments(ctx, m.collectionName, documentIDs); err != nil {
		return fmt.Errorf("failed to delete documents from Milvus: %w", err)
	}

	m.logger.Info("Deleted documents from Milvus",
		zap.String("collection", m.collectionName),
		zap.Int("count", len(documentIDs)))

	return nil
}

// ListCollections lists all collections in the database
func (m *MilvusDatabase) ListCollections(ctx context.Context) ([]string, error) {
	collections, err := m.client.ListCollections(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list collections in Milvus: %w", err)
	}

	m.logger.Info("Listed collections in Milvus",
		zap.Int("count", len(collections)))

	return collections, nil
}

// GetCollectionInfo returns information about a collection
func (m *MilvusDatabase) GetCollectionInfo(ctx context.Context, collectionName string) (map[string]interface{}, error) {
	if collectionName == "" {
		collectionName = m.collectionName
	}

	info, err := m.client.GetCollectionInfo(ctx, collectionName)
	if err != nil {
		return nil, fmt.Errorf("failed to get collection info from Milvus: %w", err)
	}

	m.logger.Info("Retrieved collection info from Milvus",
		zap.String("collection", collectionName))

	return info, nil
}

// DeleteCollection deletes a collection
func (m *MilvusDatabase) DeleteCollection(ctx context.Context, collectionName string) error {
	if err := m.client.DeleteCollection(ctx, collectionName); err != nil {
		return fmt.Errorf("failed to delete collection from Milvus: %w", err)
	}

	m.logger.Info("Deleted collection from Milvus",
		zap.String("collection", collectionName))

	return nil
}

// Cleanup cleans up resources and closes connections
func (m *MilvusDatabase) Cleanup(ctx context.Context) error {
	if err := m.client.Close(); err != nil {
		return fmt.Errorf("failed to close Milvus client: %w", err)
	}

	m.logger.Info("Cleaned up Milvus database")

	return nil
}
