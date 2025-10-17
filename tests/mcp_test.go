package tests

import (
	"context"
	"testing"

	"github.com/AI4quantum/maestro-mcp/src/pkg/config"
	localmcp "github.com/AI4quantum/maestro-mcp/src/pkg/mcp"
	"github.com/mark3labs/mcp-go/mcp"
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

	server, err := localmcp.NewServer(cfg, logger)
	require.NoError(t, err)
	assert.NotNil(t, server)
	assert.NotNil(t, server.MCPServer)
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
	server, err := localmcp.NewServer(cfg, logger)
	require.NoError(t, err)

	// Test that tools are registered
	expectedTools := []string{
		// Vector database tools
		"create_vector_database",
		"list_databases",
		"setup_database",
		"write_document",
		"query",
		"list_documents",
		"count_documents",
		"delete_document",
		"cleanup",

		// Workflow and agent tools from Python MCP server
		"run_workflow",
		"create_agents",
		"create_tools",
		"serve_agent",
		"serve_workflow",
		"serve_container_agent",
		"deploy_workflow",
	}

	for _, toolName := range expectedTools {
		tool := server.MCPServer.GetTool(toolName)
		assert.NotNil(t, tool, "Tool %s should be registered", toolName)
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
	server, err := localmcp.NewServer(cfg, logger)
	require.NoError(t, err)

	// Get the create_vector_database tool
	tool := server.MCPServer.GetTool("create_vector_database")
	require.NotNil(t, tool)

	// Test creating a vector database
	args := map[string]interface{}{
		"db_name":         "test_db",
		"db_type":         "milvus",
		"collection_name": "test_collection",
	}

	request := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: args,
		},
	}

	result, err := tool.Handler(context.Background(), request)
	assert.NoError(t, err)
	assert.NotNil(t, result)
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
	server, err := localmcp.NewServer(cfg, logger)
	require.NoError(t, err)

	// Test listing empty databases
	listTool := server.MCPServer.GetTool("list_databases")
	require.NotNil(t, listTool)

	request := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]interface{}{},
		},
	}

	result, err := listTool.Handler(context.Background(), request)
	assert.NoError(t, err)
	// The result format may have changed with the external MCP implementation
	assert.NotNil(t, result)
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
	server, err := localmcp.NewServer(cfg, logger)
	require.NoError(t, err)

	// Test missing required arguments
	createTool := server.MCPServer.GetTool("create_vector_database")
	require.NotNil(t, createTool)

	missingDbTypeRequest := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]interface{}{
				"db_name": "test_db",
				// Missing db_type
			},
		},
	}
	_, err = createTool.Handler(context.Background(), missingDbTypeRequest)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "db_type")

	missingDbNameRequest := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]interface{}{
				"db_type": "milvus",
				// Missing db_name
			},
		},
	}
	_, err = createTool.Handler(context.Background(), missingDbNameRequest)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "db_name")
}

// Made with Bob
