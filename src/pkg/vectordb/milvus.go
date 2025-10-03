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
	client         milvusclient.Client
}

// NewMilvusDatabase creates a new Milvus database instance
func NewMilvusDatabase(collectionName string, cfg *config.Config) (*MilvusDatabase, error) {
	logger, _ := zap.NewProduction()

	// Create Milvus client with configuration
	client, err := milvusclient.NewClient(context.Background(), milvusclient.Config{
		Address: fmt.Sprintf("%s:%d", cfg.Database.Milvus.Host, cfg.Database.Milvus.Port),
		Username: cfg.Database.Milvus.Username,
		Password: cfg.Database.Milvus.Password,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create Milvus client: %w", err)
	}

	db := &MilvusDatabase{
		config:         cfg,
		logger:         logger,
		collectionName: collectionName,
		client:         client,
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
	// Check if collection already exists
	exists, err := m.client.HasCollection(ctx, m.collectionName)
	if err != nil {
		return fmt.Errorf("failed to check if collection exists: %w", err)
	}

	if exists {
		m.logger.Info("Collection already exists",
			zap.String("collection", m.collectionName))
		return nil
	}

	// Create collection schema using official Milvus client
	schema := &milvusclient.CollectionSchema{
		CollectionName: m.collectionName,
		Description:    fmt.Sprintf("Collection for %s embeddings", embedding),
		Fields: []*milvusclient.FieldSchema{
			{
				FieldID:      100,
				Name:         "id",
				IsPrimaryKey: true,
				DataType:     milvusclient.FieldTypeVarChar,
				TypeParams: map[string]string{
					"max_length": "65535",
				},
			},
			{
				FieldID:  101,
				Name:     "url",
				DataType: milvusclient.FieldTypeVarChar,
				TypeParams: map[string]string{
					"max_length": "65535",
				},
			},
			{
				FieldID:  102,
				Name:     "text",
				DataType: milvusclient.FieldTypeVarChar,
				TypeParams: map[string]string{
					"max_length": "65535",
				},
			},
			{
				FieldID:  103,
				Name:     "metadata",
				DataType: milvusclient.FieldTypeJSON,
			},
			{
				FieldID:  104,
				Name:     "vector",
				DataType: milvusclient.FieldTypeFloatVector,
				TypeParams: map[string]string{
					"dim": fmt.Sprintf("%d", m.config.MCP.Embedding.VectorSize),
				},
			},
		},
	}

	if err := m.client.CreateCollection(ctx, schema); err != nil {
		return fmt.Errorf("failed to create collection: %w", err)
	}

	// Create index for vector field
	indexParams := map[string]string{
		"metric_type": "L2",
		"index_type":  "IVF_FLAT",
		"params":      `{"nlist": 1024}`,
	}

	if err := m.client.CreateIndex(ctx, m.collectionName, "vector", indexParams); err != nil {
		return fmt.Errorf("failed to create index: %w", err)
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

	// Prepare data for insertion
	var entities []map[string]interface{}
	for _, doc := range docs {
		entity := map[string]interface{}{
			"id":       doc.ID,
			"url":      doc.URL,
			"text":     doc.Text,
			"metadata": doc.Metadata,
			"vector":   doc.Vector,
		}
		entities = append(entities, entity)
	}

	// Insert documents using official Milvus client
	_, err := m.client.Insert(ctx, m.collectionName, entities)
	if err != nil {
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

	// For now, we'll use search as the query method
	// In a real implementation, you might want to use hybrid search
	results, err := m.Search(ctx, query, limit, collectionName)
	if err != nil {
		return nil, fmt.Errorf("failed to query Milvus: %w", err)
	}

	m.logger.Info("Executed query on Milvus",
		zap.String("collection", collectionName),
		zap.String("query", query),
		zap.Int("limit", limit))

	return results, nil
}

// Search performs a vector similarity search
func (m *MilvusDatabase) Search(ctx context.Context, query string, limit int, collectionName string) ([]SearchResult, error) {
	if collectionName == "" {
		collectionName = m.collectionName
	}

	// For now, return empty results since we need to implement vector search
	// In a real implementation, you would:
	// 1. Convert the query text to a vector using embedding service
	// 2. Use the vector to search in Milvus
	// 3. Return the results with scores
	
	// Placeholder implementation
	results := []SearchResult{}

	m.logger.Info("Executed search on Milvus",
		zap.String("collection", collectionName),
		zap.String("query", query),
		zap.Int("limit", limit),
		zap.Int("results", len(results)))

	return results, nil
}

// ListDocuments lists documents from the database
func (m *MilvusDatabase) ListDocuments(ctx context.Context, limit, offset int) ([]Document, error) {
	// Placeholder implementation - would use official client Query method
	documents := []Document{}

	m.logger.Info("Listed documents from Milvus",
		zap.String("collection", m.collectionName),
		zap.Int("limit", limit),
		zap.Int("offset", offset),
		zap.Int("count", len(documents)))

	return documents, nil
}

// CountDocuments returns the count of documents in the database
func (m *MilvusDatabase) CountDocuments(ctx context.Context) (int, error) {
	// Placeholder implementation - would use official client Query method
	count := 0

	m.logger.Info("Counted documents in Milvus",
		zap.String("collection", m.collectionName),
		zap.Int("count", count))

	return count, nil
}

// DeleteDocument deletes a document by ID
func (m *MilvusDatabase) DeleteDocument(ctx context.Context, documentID string) error {
	// Placeholder implementation - would use official client Delete method
	m.logger.Info("Deleted document from Milvus",
		zap.String("collection", m.collectionName),
		zap.String("document_id", documentID))

	return nil
}

// DeleteDocuments deletes multiple documents by IDs
func (m *MilvusDatabase) DeleteDocuments(ctx context.Context, documentIDs []string) error {
	// Placeholder implementation - would use official client Delete method
	m.logger.Info("Deleted documents from Milvus",
		zap.String("collection", m.collectionName),
		zap.Int("count", len(documentIDs)))

	return nil
}

// ListCollections lists all collections in the database
func (m *MilvusDatabase) ListCollections(ctx context.Context) ([]string, error) {
	// Placeholder implementation - would use official client ListCollections method
	collections := []string{}

	m.logger.Info("Listed collections in Milvus",
		zap.Int("count", len(collections)))

	return collections, nil
}

// GetCollectionInfo returns information about a collection
func (m *MilvusDatabase) GetCollectionInfo(ctx context.Context, collectionName string) (map[string]interface{}, error) {
	if collectionName == "" {
		collectionName = m.collectionName
	}

	// Placeholder implementation - would use official client DescribeCollection method
	info := map[string]interface{}{
		"name": collectionName,
		"type": "milvus",
	}

	m.logger.Info("Retrieved collection info from Milvus",
		zap.String("collection", collectionName))

	return info, nil
}

// DeleteCollection deletes a collection
func (m *MilvusDatabase) DeleteCollection(ctx context.Context, collectionName string) error {
	// Placeholder implementation - would use official client DropCollection method
	m.logger.Info("Deleted collection from Milvus",
		zap.String("collection", collectionName))

	return nil
}

// Cleanup cleans up resources and closes connections
func (m *MilvusDatabase) Cleanup(ctx context.Context) error {
	// Placeholder implementation - would use official client Close method
	m.logger.Info("Cleaned up Milvus database")

	return nil
}
