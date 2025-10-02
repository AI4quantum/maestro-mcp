package vectordb

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"
)

// MockMilvusClient implements MilvusClient for testing
type MockMilvusClient struct {
	collections map[string]map[string]interface{}
	documents   map[string][]Document
	mutex       sync.RWMutex
	logger      *zap.Logger
}

// NewMockMilvusClient creates a new mock Milvus client
func NewMockMilvusClient() *MockMilvusClient {
	logger, _ := zap.NewProduction()
	return &MockMilvusClient{
		collections: make(map[string]map[string]interface{}),
		documents:   make(map[string][]Document),
		logger:      logger,
	}
}

// Connect simulates connecting to Milvus
func (m *MockMilvusClient) Connect(ctx context.Context) error {
	m.logger.Info("Mock Milvus client connected")
	return nil
}

// CreateCollection simulates creating a collection
func (m *MockMilvusClient) CreateCollection(ctx context.Context, name string, schema map[string]interface{}) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.collections[name] = schema
	m.documents[name] = make([]Document, 0)

	m.logger.Info("Mock Milvus collection created", zap.String("name", name))
	return nil
}

// Insert simulates inserting documents
func (m *MockMilvusClient) Insert(ctx context.Context, collectionName string, documents []Document) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if _, exists := m.documents[collectionName]; !exists {
		return fmt.Errorf("collection '%s' does not exist", collectionName)
	}

	// Add IDs to documents if not present
	for i := range documents {
		if documents[i].ID == "" {
			documents[i].ID = fmt.Sprintf("doc_%d_%d", time.Now().UnixNano(), i)
		}
	}

	m.documents[collectionName] = append(m.documents[collectionName], documents...)

	m.logger.Info("Mock Milvus documents inserted",
		zap.String("collection", collectionName),
		zap.Int("count", len(documents)))

	return nil
}

// Search simulates vector search
func (m *MockMilvusClient) Search(ctx context.Context, collectionName string, query string, limit int) ([]SearchResult, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	docs, exists := m.documents[collectionName]
	if !exists {
		return nil, fmt.Errorf("collection '%s' does not exist", collectionName)
	}

	results := make([]SearchResult, 0, limit)
	for i, doc := range docs {
		if i >= limit {
			break
		}
		results = append(results, SearchResult{
			Document: doc,
			Score:    0.9 - float64(i)*0.1, // Mock decreasing scores
		})
	}

	m.logger.Info("Mock Milvus search executed",
		zap.String("collection", collectionName),
		zap.String("query", query),
		zap.Int("limit", limit),
		zap.Int("results", len(results)))

	return results, nil
}

// Query simulates natural language query
func (m *MockMilvusClient) Query(ctx context.Context, collectionName string, query string, limit int) (interface{}, error) {
	results, err := m.Search(ctx, collectionName, query, limit)
	if err != nil {
		return nil, err
	}

	// Convert to natural language response
	response := fmt.Sprintf("Found %d relevant documents for query '%s':\n", len(results), query)
	for i, result := range results {
		response += fmt.Sprintf("%d. %s (Score: %.2f)\n", i+1, result.Document.Text[:min(100, len(result.Document.Text))], result.Score)
	}

	return response, nil
}

// ListDocuments simulates listing documents
func (m *MockMilvusClient) ListDocuments(ctx context.Context, collectionName string, limit, offset int) ([]Document, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	docs, exists := m.documents[collectionName]
	if !exists {
		return nil, fmt.Errorf("collection '%s' does not exist", collectionName)
	}

	start := offset
	end := offset + limit
	if start >= len(docs) {
		return []Document{}, nil
	}
	if end > len(docs) {
		end = len(docs)
	}

	result := docs[start:end]

	m.logger.Info("Mock Milvus documents listed",
		zap.String("collection", collectionName),
		zap.Int("limit", limit),
		zap.Int("offset", offset),
		zap.Int("count", len(result)))

	return result, nil
}

// CountDocuments simulates counting documents
func (m *MockMilvusClient) CountDocuments(ctx context.Context, collectionName string) (int, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	docs, exists := m.documents[collectionName]
	if !exists {
		return 0, fmt.Errorf("collection '%s' does not exist", collectionName)
	}

	count := len(docs)

	m.logger.Info("Mock Milvus documents counted",
		zap.String("collection", collectionName),
		zap.Int("count", count))

	return count, nil
}

// DeleteDocument simulates deleting a document
func (m *MockMilvusClient) DeleteDocument(ctx context.Context, collectionName string, documentID string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	docs, exists := m.documents[collectionName]
	if !exists {
		return fmt.Errorf("collection '%s' does not exist", collectionName)
	}

	for i, doc := range docs {
		if doc.ID == documentID {
			m.documents[collectionName] = append(docs[:i], docs[i+1:]...)
			m.logger.Info("Mock Milvus document deleted",
				zap.String("collection", collectionName),
				zap.String("document_id", documentID))
			return nil
		}
	}

	return fmt.Errorf("document '%s' not found", documentID)
}

// DeleteDocuments simulates deleting multiple documents
func (m *MockMilvusClient) DeleteDocuments(ctx context.Context, collectionName string, documentIDs []string) error {
	for _, id := range documentIDs {
		if err := m.DeleteDocument(ctx, collectionName, id); err != nil {
			return err
		}
	}
	return nil
}

// ListCollections simulates listing collections
func (m *MockMilvusClient) ListCollections(ctx context.Context) ([]string, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	collections := make([]string, 0, len(m.collections))
	for name := range m.collections {
		collections = append(collections, name)
	}

	m.logger.Info("Mock Milvus collections listed", zap.Int("count", len(collections)))

	return collections, nil
}

// GetCollectionInfo simulates getting collection info
func (m *MockMilvusClient) GetCollectionInfo(ctx context.Context, collectionName string) (map[string]interface{}, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	schema, exists := m.collections[collectionName]
	if !exists {
		return nil, fmt.Errorf("collection '%s' does not exist", collectionName)
	}

	docs, _ := m.documents[collectionName]

	info := map[string]interface{}{
		"name":           collectionName,
		"schema":         schema,
		"document_count": len(docs),
		"created_at":     time.Now().Format(time.RFC3339),
	}

	m.logger.Info("Mock Milvus collection info retrieved", zap.String("collection", collectionName))

	return info, nil
}

// DeleteCollection simulates deleting a collection
func (m *MockMilvusClient) DeleteCollection(ctx context.Context, collectionName string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	delete(m.collections, collectionName)
	delete(m.documents, collectionName)

	m.logger.Info("Mock Milvus collection deleted", zap.String("collection", collectionName))

	return nil
}

// Close simulates closing the client
func (m *MockMilvusClient) Close() error {
	m.logger.Info("Mock Milvus client closed")
	return nil
}

// MockWeaviateClient implements WeaviateClient for testing
type MockWeaviateClient struct {
	collections map[string]map[string]interface{}
	documents   map[string][]Document
	mutex       sync.RWMutex
	logger      *zap.Logger
}

// NewMockWeaviateClient creates a new mock Weaviate client
func NewMockWeaviateClient() *MockWeaviateClient {
	logger, _ := zap.NewProduction()
	return &MockWeaviateClient{
		collections: make(map[string]map[string]interface{}),
		documents:   make(map[string][]Document),
		logger:      logger,
	}
}

// Connect simulates connecting to Weaviate
func (m *MockWeaviateClient) Connect(ctx context.Context) error {
	m.logger.Info("Mock Weaviate client connected")
	return nil
}

// CreateCollection simulates creating a collection
func (m *MockWeaviateClient) CreateCollection(ctx context.Context, name string, schema map[string]interface{}) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.collections[name] = schema
	m.documents[name] = make([]Document, 0)

	m.logger.Info("Mock Weaviate collection created", zap.String("name", name))
	return nil
}

// Insert simulates inserting documents
func (m *MockWeaviateClient) Insert(ctx context.Context, collectionName string, documents []Document) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if _, exists := m.documents[collectionName]; !exists {
		return fmt.Errorf("collection '%s' does not exist", collectionName)
	}

	// Add IDs to documents if not present
	for i := range documents {
		if documents[i].ID == "" {
			documents[i].ID = fmt.Sprintf("doc_%d_%d", time.Now().UnixNano(), i)
		}
	}

	m.documents[collectionName] = append(m.documents[collectionName], documents...)

	m.logger.Info("Mock Weaviate documents inserted",
		zap.String("collection", collectionName),
		zap.Int("count", len(documents)))

	return nil
}

// Search simulates vector search
func (m *MockWeaviateClient) Search(ctx context.Context, collectionName string, query string, limit int) ([]SearchResult, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	docs, exists := m.documents[collectionName]
	if !exists {
		return nil, fmt.Errorf("collection '%s' does not exist", collectionName)
	}

	results := make([]SearchResult, 0, limit)
	for i, doc := range docs {
		if i >= limit {
			break
		}
		results = append(results, SearchResult{
			Document: doc,
			Score:    0.9 - float64(i)*0.1, // Mock decreasing scores
		})
	}

	m.logger.Info("Mock Weaviate search executed",
		zap.String("collection", collectionName),
		zap.String("query", query),
		zap.Int("limit", limit),
		zap.Int("results", len(results)))

	return results, nil
}

// Query simulates natural language query
func (m *MockWeaviateClient) Query(ctx context.Context, collectionName string, query string, limit int) (interface{}, error) {
	results, err := m.Search(ctx, collectionName, query, limit)
	if err != nil {
		return nil, err
	}

	// Convert to natural language response
	response := fmt.Sprintf("Found %d relevant documents for query '%s':\n", len(results), query)
	for i, result := range results {
		response += fmt.Sprintf("%d. %s (Score: %.2f)\n", i+1, result.Document.Text[:min(100, len(result.Document.Text))], result.Score)
	}

	return response, nil
}

// ListDocuments simulates listing documents
func (m *MockWeaviateClient) ListDocuments(ctx context.Context, collectionName string, limit, offset int) ([]Document, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	docs, exists := m.documents[collectionName]
	if !exists {
		return nil, fmt.Errorf("collection '%s' does not exist", collectionName)
	}

	start := offset
	end := offset + limit
	if start >= len(docs) {
		return []Document{}, nil
	}
	if end > len(docs) {
		end = len(docs)
	}

	result := docs[start:end]

	m.logger.Info("Mock Weaviate documents listed",
		zap.String("collection", collectionName),
		zap.Int("limit", limit),
		zap.Int("offset", offset),
		zap.Int("count", len(result)))

	return result, nil
}

// CountDocuments simulates counting documents
func (m *MockWeaviateClient) CountDocuments(ctx context.Context, collectionName string) (int, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	docs, exists := m.documents[collectionName]
	if !exists {
		return 0, fmt.Errorf("collection '%s' does not exist", collectionName)
	}

	count := len(docs)

	m.logger.Info("Mock Weaviate documents counted",
		zap.String("collection", collectionName),
		zap.Int("count", count))

	return count, nil
}

// DeleteDocument simulates deleting a document
func (m *MockWeaviateClient) DeleteDocument(ctx context.Context, collectionName string, documentID string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	docs, exists := m.documents[collectionName]
	if !exists {
		return fmt.Errorf("collection '%s' does not exist", collectionName)
	}

	for i, doc := range docs {
		if doc.ID == documentID {
			m.documents[collectionName] = append(docs[:i], docs[i+1:]...)
			m.logger.Info("Mock Weaviate document deleted",
				zap.String("collection", collectionName),
				zap.String("document_id", documentID))
			return nil
		}
	}

	return fmt.Errorf("document '%s' not found", documentID)
}

// DeleteDocuments simulates deleting multiple documents
func (m *MockWeaviateClient) DeleteDocuments(ctx context.Context, collectionName string, documentIDs []string) error {
	for _, id := range documentIDs {
		if err := m.DeleteDocument(ctx, collectionName, id); err != nil {
			return err
		}
	}
	return nil
}

// ListCollections simulates listing collections
func (m *MockWeaviateClient) ListCollections(ctx context.Context) ([]string, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	collections := make([]string, 0, len(m.collections))
	for name := range m.collections {
		collections = append(collections, name)
	}

	m.logger.Info("Mock Weaviate collections listed", zap.Int("count", len(collections)))

	return collections, nil
}

// GetCollectionInfo simulates getting collection info
func (m *MockWeaviateClient) GetCollectionInfo(ctx context.Context, collectionName string) (map[string]interface{}, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	schema, exists := m.collections[collectionName]
	if !exists {
		return nil, fmt.Errorf("collection '%s' does not exist", collectionName)
	}

	docs, _ := m.documents[collectionName]

	info := map[string]interface{}{
		"name":           collectionName,
		"schema":         schema,
		"document_count": len(docs),
		"created_at":     time.Now().Format(time.RFC3339),
	}

	m.logger.Info("Mock Weaviate collection info retrieved", zap.String("collection", collectionName))

	return info, nil
}

// DeleteCollection simulates deleting a collection
func (m *MockWeaviateClient) DeleteCollection(ctx context.Context, collectionName string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	delete(m.collections, collectionName)
	delete(m.documents, collectionName)

	m.logger.Info("Mock Weaviate collection deleted", zap.String("collection", collectionName))

	return nil
}

// Close simulates closing the client
func (m *MockWeaviateClient) Close() error {
	m.logger.Info("Mock Weaviate client closed")
	return nil
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
