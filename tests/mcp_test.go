package tests

import (
	"testing"

	"github.com/AI4quantum/maestro-mcp/src/pkg/config"
	"github.com/AI4quantum/maestro-mcp/src/pkg/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestMCPServerCreation(t *testing.T) {
	cfg := &config.Config{
		MCP: config.MCPConfig{
			ToolTimeout: 15,
			VectorDB: config.VectorDBConfig{
				Type: "milvus",
				Milvus: config.MilvusConfig{
					Host: "localhost",
					Port: 19530,
				},
			},
		},
	}
	
	logger, _ := zap.NewProduction()
	
	server, err := mcp.NewServer(cfg, logger)
	require.NoError(t, err)
	assert.NotNil(t, server)
}

func TestMCPServerToolsRegistration(t *testing.T) {
	cfg := &config.Config{
		MCP: config.MCPConfig{
			ToolTimeout: 15,
			VectorDB: config.VectorDBConfig{
				Type: "milvus",
				Milvus: config.MilvusConfig{
					Host: "localhost",
					Port: 19530,
				},
			},
		},
	}
	
	logger, _ := zap.NewProduction()
	server, err := mcp.NewServer(cfg, logger)
	require.NoError(t, err)
	
	// Test that tools are registered
	expectedTools := []string{
		"create_vector_database",
		"list_databases",
		"setup_database",
		"write_document",
		"query",
		"list_documents",
		"count_documents",
		"delete_document",
		"cleanup",
	}
	
	for _, toolName := range expectedTools {
		_, exists := server.Tools[toolName]
		assert.True(t, exists, "Tool %s should be registered", toolName)
	}
}

func TestMCPServerCreateVectorDatabase(t *testing.T) {
	cfg := &config.Config{
		MCP: config.MCPConfig{
			ToolTimeout: 15,
			VectorDB: config.VectorDBConfig{
				Type: "milvus",
				Milvus: config.MilvusConfig{
					Host: "localhost",
					Port: 19530,
				},
			},
		},
	}
	
	logger, _ := zap.NewProduction()
	server, err := mcp.NewServer(cfg, logger)
	require.NoError(t, err)
	
	// Get the create_vector_database tool
	tool, exists := server.Tools["create_vector_database"]
	require.True(t, exists)
	
	// Test creating a vector database
	args := map[string]interface{}{
		"db_name":        "test_db",
		"db_type":        "milvus",
		"collection_name": "test_collection",
	}
	
	result, err := tool.Handler(nil, args)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Contains(t, result.(string), "Successfully created")
}

func TestMCPServerListDatabasesEmpty(t *testing.T) {
	cfg := &config.Config{
		MCP: config.MCPConfig{
			ToolTimeout: 15,
			VectorDB: config.VectorDBConfig{
				Type: "milvus",
				Milvus: config.MilvusConfig{
					Host: "localhost",
					Port: 19530,
				},
			},
		},
	}
	
	logger, _ := zap.NewProduction()
	server, err := mcp.NewServer(cfg, logger)
	require.NoError(t, err)
	
	// Test listing empty databases
	listTool, exists := server.Tools["list_databases"]
	require.True(t, exists)
	
	result, err := listTool.Handler(nil, map[string]interface{}{})
	assert.NoError(t, err)
	assert.Equal(t, "No vector databases are currently active", result)
}

func TestMCPServerInvalidArguments(t *testing.T) {
	cfg := &config.Config{
		MCP: config.MCPConfig{
			ToolTimeout: 15,
			VectorDB: config.VectorDBConfig{
				Type: "milvus",
				Milvus: config.MilvusConfig{
					Host: "localhost",
					Port: 19530,
				},
			},
		},
	}
	
	logger, _ := zap.NewProduction()
	server, err := mcp.NewServer(cfg, logger)
	require.NoError(t, err)
	
	// Test missing required arguments
	createTool, exists := server.Tools["create_vector_database"]
	require.True(t, exists)
	
	_, err = createTool.Handler(nil, map[string]interface{}{
		"db_name": "test_db",
		// Missing db_type
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "db_type is required")
	
	_, err = createTool.Handler(nil, map[string]interface{}{
		"db_type": "milvus",
		// Missing db_name
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "db_name is required")
}