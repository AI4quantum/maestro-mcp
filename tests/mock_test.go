package tests

import (
	"context"
	"testing"

	"github.com/maximilien/maestro-mcp/src/pkg/config"
	"github.com/maximilien/maestro-mcp/src/pkg/vectordb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMockMilvusClient(t *testing.T) {
	client := vectordb.NewMockMilvusClient()
	ctx := context.Background()
	
	// Test connection
	err := client.Connect(ctx)
	assert.NoError(t, err)
	
	// Test collection creation
	schema := map[string]interface{}{
		"name": "test_collection",
		"fields": []map[string]interface{}{
			{"name": "id", "type": "string", "primary": true},
			{"name": "text", "type": "string"},
		},
	}
	
	err = client.CreateCollection(ctx, "test_collection", schema)
	assert.NoError(t, err)
	
	// Test document insertion
	documents := []vectordb.Document{
		{
			URL:      "https://example.com/doc1",
			Text:     "This is a test document",
			Metadata: map[string]interface{}{"author": "test"},
		},
		{
			URL:      "https://example.com/doc2",
			Text:     "This is another test document",
			Metadata: map[string]interface{}{"author": "test"},
		},
	}
	
	err = client.Insert(ctx, "test_collection", documents)
	assert.NoError(t, err)
	
	// Test document listing
	docs, err := client.ListDocuments(ctx, "test_collection", 10, 0)
	assert.NoError(t, err)
	assert.Len(t, docs, 2)
	
	// Test document counting
	count, err := client.CountDocuments(ctx, "test_collection")
	assert.NoError(t, err)
	assert.Equal(t, 2, count)
	
	// Test search
	results, err := client.Search(ctx, "test_collection", "test document", 5)
	assert.NoError(t, err)
	assert.Len(t, results, 2)
	assert.Greater(t, results[0].Score, results[1].Score) // Scores should be decreasing
	
	// Test query
	queryResult, err := client.Query(ctx, "test_collection", "test document", 5)
	assert.NoError(t, err)
	assert.NotEmpty(t, queryResult)
	
	// Test collection listing
	collections, err := client.ListCollections(ctx)
	assert.NoError(t, err)
	assert.Contains(t, collections, "test_collection")
	
	// Test collection info
	info, err := client.GetCollectionInfo(ctx, "test_collection")
	assert.NoError(t, err)
	assert.Equal(t, "test_collection", info["name"])
	assert.Equal(t, 2, info["document_count"])
	
	// Test document deletion
	if len(docs) > 0 {
		err = client.DeleteDocument(ctx, "test_collection", docs[0].ID)
		assert.NoError(t, err)
		
		// Verify deletion
		newCount, err := client.CountDocuments(ctx, "test_collection")
		assert.NoError(t, err)
		assert.Equal(t, 1, newCount)
	}
	
	// Test collection deletion
	err = client.DeleteCollection(ctx, "test_collection")
	assert.NoError(t, err)
	
	// Test close
	err = client.Close()
	assert.NoError(t, err)
}

func TestMockWeaviateClient(t *testing.T) {
	client := vectordb.NewMockWeaviateClient()
	ctx := context.Background()
	
	// Test connection
	err := client.Connect(ctx)
	assert.NoError(t, err)
	
	// Test collection creation
	schema := map[string]interface{}{
		"class": "TestClass",
		"properties": []map[string]interface{}{
			{"name": "url", "dataType": []string{"string"}},
			{"name": "text", "dataType": []string{"text"}},
		},
	}
	
	err = client.CreateCollection(ctx, "TestClass", schema)
	assert.NoError(t, err)
	
	// Test document insertion
	documents := []vectordb.Document{
		{
			URL:      "https://example.com/doc1",
			Text:     "This is a test document for Weaviate",
			Metadata: map[string]interface{}{"category": "test"},
		},
	}
	
	err = client.Insert(ctx, "TestClass", documents)
	assert.NoError(t, err)
	
	// Test document listing
	docs, err := client.ListDocuments(ctx, "TestClass", 10, 0)
	assert.NoError(t, err)
	assert.Len(t, docs, 1)
	
	// Test document counting
	count, err := client.CountDocuments(ctx, "TestClass")
	assert.NoError(t, err)
	assert.Equal(t, 1, count)
	
	// Test search
	results, err := client.Search(ctx, "TestClass", "test document", 5)
	assert.NoError(t, err)
	assert.Len(t, results, 1)
	assert.Greater(t, results[0].Score, 0.0)
	
	// Test query
	queryResult, err := client.Query(ctx, "TestClass", "test document", 5)
	assert.NoError(t, err)
	assert.NotEmpty(t, queryResult)
	
	// Test collection listing
	collections, err := client.ListCollections(ctx)
	assert.NoError(t, err)
	assert.Contains(t, collections, "TestClass")
	
	// Test collection info
	info, err := client.GetCollectionInfo(ctx, "TestClass")
	assert.NoError(t, err)
	assert.Equal(t, "TestClass", info["name"])
	assert.Equal(t, 1, info["document_count"])
	
	// Test close
	err = client.Close()
	assert.NoError(t, err)
}

func TestMockVectorDatabaseIntegration(t *testing.T) {
	// Create a test configuration
	cfg := &config.Config{
		MCP: config.MCPConfig{
			Embedding: config.EmbeddingConfig{
				VectorSize: 1536,
			},
		},
	}
	
	// Test Milvus database
	milvusDB, err := vectordb.NewMilvusDatabase("test_milvus", cfg)
	require.NoError(t, err)
	assert.Equal(t, "milvus", milvusDB.Type())
	assert.Equal(t, "test_milvus", milvusDB.CollectionName())
	
	ctx := context.Background()
	
	// Test setup
	err = milvusDB.Setup(ctx, "default")
	assert.NoError(t, err)
	
	// Test document operations
	doc := vectordb.Document{
		URL:      "https://example.com/test",
		Text:     "Test document for integration testing",
		Metadata: map[string]interface{}{"test": true},
	}
	
	stats, err := milvusDB.WriteDocument(ctx, doc)
	assert.NoError(t, err)
	assert.Equal(t, 1, stats.DocumentsWritten)
	
	count, err := milvusDB.CountDocuments(ctx)
	assert.NoError(t, err)
	assert.Equal(t, 1, count)
	
	// Test Weaviate database
	weaviateDB, err := vectordb.NewWeaviateDatabase("test_weaviate", cfg)
	require.NoError(t, err)
	assert.Equal(t, "weaviate", weaviateDB.Type())
	assert.Equal(t, "test_weaviate", weaviateDB.CollectionName())
	
	// Test setup
	err = weaviateDB.Setup(ctx, "default")
	assert.NoError(t, err)
	
	// Test document operations
	stats, err = weaviateDB.WriteDocument(ctx, doc)
	assert.NoError(t, err)
	assert.Equal(t, 1, stats.DocumentsWritten)
	
	count, err = weaviateDB.CountDocuments(ctx)
	assert.NoError(t, err)
	assert.Equal(t, 1, count)
	
	// Test cleanup
	err = milvusDB.Cleanup(ctx)
	assert.NoError(t, err)
	
	err = weaviateDB.Cleanup(ctx)
	assert.NoError(t, err)
}

func TestMockErrorHandling(t *testing.T) {
	client := vectordb.NewMockMilvusClient()
	ctx := context.Background()
	
	// Test operations on non-existent collection
	_, err := client.ListDocuments(ctx, "non_existent", 10, 0)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "does not exist")
	
	_, err = client.CountDocuments(ctx, "non_existent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "does not exist")
	
	_, err = client.Search(ctx, "non_existent", "query", 5)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "does not exist")
	
	// Test document deletion on non-existent collection
	err = client.DeleteDocument(ctx, "non_existent", "doc_id")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "does not exist")
	
	// Test document deletion with non-existent document ID
	err = client.CreateCollection(ctx, "test_collection", map[string]interface{}{"name": "test_collection"})
	assert.NoError(t, err)
	
	err = client.DeleteDocument(ctx, "test_collection", "non_existent_doc")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}