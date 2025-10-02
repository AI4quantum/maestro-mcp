package vectordb

import (
	"context"
	"fmt"
	"time"

	"github.com/maximilien/maestro-mcp/src/pkg/config"
	"go.uber.org/zap"
)

// WeaviateDatabase implements VectorDatabase for Weaviate
type WeaviateDatabase struct {
	config         *config.Config
	logger         *zap.Logger
	collectionName string
	client         WeaviateClient
}

// WeaviateClient defines the interface for Weaviate client operations
type WeaviateClient interface {
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

// NewWeaviateDatabase creates a new Weaviate database instance
func NewWeaviateDatabase(collectionName string, cfg *config.Config) (*WeaviateDatabase, error) {
	logger, _ := zap.NewProduction()

	db := &WeaviateDatabase{
		config:         cfg,
		logger:         logger,
		collectionName: collectionName,
		client:         NewMockWeaviateClient(), // Use mock for now
	}

	return db, nil
}

// Type returns the database type
func (w *WeaviateDatabase) Type() string {
	return "weaviate"
}

// CollectionName returns the current collection name
func (w *WeaviateDatabase) CollectionName() string {
	return w.collectionName
}

// Setup initializes the database and creates collections
func (w *WeaviateDatabase) Setup(ctx context.Context, embedding string) error {
	if err := w.client.Connect(ctx); err != nil {
		return fmt.Errorf("failed to connect to Weaviate: %w", err)
	}

	// Create collection schema
	schema := map[string]interface{}{
		"class": w.collectionName,
		"properties": []map[string]interface{}{
			{
				"name":     "url",
				"dataType": []string{"string"},
			},
			{
				"name":     "text",
				"dataType": []string{"text"},
			},
			{
				"name":     "metadata",
				"dataType": []string{"object"},
			},
		},
		"vectorizer": embedding,
	}

	if err := w.client.CreateCollection(ctx, w.collectionName, schema); err != nil {
		return fmt.Errorf("failed to create collection: %w", err)
	}

	w.logger.Info("Set up Weaviate collection",
		zap.String("collection", w.collectionName),
		zap.String("embedding", embedding))

	return nil
}

// WriteDocument writes a single document to the database
func (w *WeaviateDatabase) WriteDocument(ctx context.Context, doc Document) (WriteStats, error) {
	start := time.Now()

	stats, err := w.WriteDocuments(ctx, []Document{doc})
	if err != nil {
		return WriteStats{}, err
	}

	stats.ProcessingTime = time.Since(start).String()
	return stats, nil
}

// WriteDocuments writes multiple documents to the database
func (w *WeaviateDatabase) WriteDocuments(ctx context.Context, docs []Document) (WriteStats, error) {
	start := time.Now()

	if err := w.client.Insert(ctx, w.collectionName, docs); err != nil {
		return WriteStats{}, fmt.Errorf("failed to insert documents: %w", err)
	}

	processingTime := time.Since(start)

	w.logger.Info("Wrote documents to Weaviate",
		zap.String("collection", w.collectionName),
		zap.Int("count", len(docs)),
		zap.Duration("processing_time", processingTime))

	return WriteStats{
		DocumentsWritten: len(docs),
		ProcessingTime:   processingTime.String(),
	}, nil
}

// Query performs a natural language query on the database
func (w *WeaviateDatabase) Query(ctx context.Context, query string, limit int, collectionName string) (interface{}, error) {
	if collectionName == "" {
		collectionName = w.collectionName
	}

	result, err := w.client.Query(ctx, collectionName, query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query Weaviate: %w", err)
	}

	w.logger.Info("Executed query on Weaviate",
		zap.String("collection", collectionName),
		zap.String("query", query),
		zap.Int("limit", limit))

	return result, nil
}

// Search performs a vector similarity search
func (w *WeaviateDatabase) Search(ctx context.Context, query string, limit int, collectionName string) ([]SearchResult, error) {
	if collectionName == "" {
		collectionName = w.collectionName
	}

	results, err := w.client.Search(ctx, collectionName, query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to search Weaviate: %w", err)
	}

	w.logger.Info("Executed search on Weaviate",
		zap.String("collection", collectionName),
		zap.String("query", query),
		zap.Int("limit", limit),
		zap.Int("results", len(results)))

	return results, nil
}

// ListDocuments lists documents from the database
func (w *WeaviateDatabase) ListDocuments(ctx context.Context, limit, offset int) ([]Document, error) {
	documents, err := w.client.ListDocuments(ctx, w.collectionName, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list documents from Weaviate: %w", err)
	}

	w.logger.Info("Listed documents from Weaviate",
		zap.String("collection", w.collectionName),
		zap.Int("limit", limit),
		zap.Int("offset", offset),
		zap.Int("count", len(documents)))

	return documents, nil
}

// CountDocuments returns the count of documents in the database
func (w *WeaviateDatabase) CountDocuments(ctx context.Context) (int, error) {
	count, err := w.client.CountDocuments(ctx, w.collectionName)
	if err != nil {
		return 0, fmt.Errorf("failed to count documents in Weaviate: %w", err)
	}

	w.logger.Info("Counted documents in Weaviate",
		zap.String("collection", w.collectionName),
		zap.Int("count", count))

	return count, nil
}

// DeleteDocument deletes a document by ID
func (w *WeaviateDatabase) DeleteDocument(ctx context.Context, documentID string) error {
	if err := w.client.DeleteDocument(ctx, w.collectionName, documentID); err != nil {
		return fmt.Errorf("failed to delete document from Weaviate: %w", err)
	}

	w.logger.Info("Deleted document from Weaviate",
		zap.String("collection", w.collectionName),
		zap.String("document_id", documentID))

	return nil
}

// DeleteDocuments deletes multiple documents by IDs
func (w *WeaviateDatabase) DeleteDocuments(ctx context.Context, documentIDs []string) error {
	if err := w.client.DeleteDocuments(ctx, w.collectionName, documentIDs); err != nil {
		return fmt.Errorf("failed to delete documents from Weaviate: %w", err)
	}

	w.logger.Info("Deleted documents from Weaviate",
		zap.String("collection", w.collectionName),
		zap.Int("count", len(documentIDs)))

	return nil
}

// ListCollections lists all collections in the database
func (w *WeaviateDatabase) ListCollections(ctx context.Context) ([]string, error) {
	collections, err := w.client.ListCollections(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list collections in Weaviate: %w", err)
	}

	w.logger.Info("Listed collections in Weaviate",
		zap.Int("count", len(collections)))

	return collections, nil
}

// GetCollectionInfo returns information about a collection
func (w *WeaviateDatabase) GetCollectionInfo(ctx context.Context, collectionName string) (map[string]interface{}, error) {
	if collectionName == "" {
		collectionName = w.collectionName
	}

	info, err := w.client.GetCollectionInfo(ctx, collectionName)
	if err != nil {
		return nil, fmt.Errorf("failed to get collection info from Weaviate: %w", err)
	}

	w.logger.Info("Retrieved collection info from Weaviate",
		zap.String("collection", collectionName))

	return info, nil
}

// DeleteCollection deletes a collection
func (w *WeaviateDatabase) DeleteCollection(ctx context.Context, collectionName string) error {
	if err := w.client.DeleteCollection(ctx, collectionName); err != nil {
		return fmt.Errorf("failed to delete collection from Weaviate: %w", err)
	}

	w.logger.Info("Deleted collection from Weaviate",
		zap.String("collection", collectionName))

	return nil
}

// Cleanup cleans up resources and closes connections
func (w *WeaviateDatabase) Cleanup(ctx context.Context) error {
	if err := w.client.Close(); err != nil {
		return fmt.Errorf("failed to close Weaviate client: %w", err)
	}

	w.logger.Info("Cleaned up Weaviate database")

	return nil
}
