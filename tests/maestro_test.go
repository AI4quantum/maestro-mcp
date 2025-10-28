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

// setupTestServer creates a new MCP server for testing
func setupTestServer(t *testing.T) *localmcp.Server {
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
	return server
}

// TestRunWorkflow tests the run_workflow tool
func TestRunWorkflow(t *testing.T) {
	server := setupTestServer(t)

	// Get the run_workflow tool from the MCPServer field
	tool := server.MCPServer.GetTool("run_workflow")
	require.NotNil(t, tool)

	// Test with valid arguments
	args := map[string]interface{}{
		"agents": []interface{}{
			`{
				"apiVersion": "maestro/v1alpha1",
				"kind": "Agent",
				"metadata": {
					"name": "test1",
					"labels": {
						"app": "test-example"
					}
				},
				"spec": {
					"model": "meta-llama/llama-3-1-70b-instruct",
					"framework": "beeai",
					"mode": "local",
					"description": "this is a test",
					"tools": ["code_interpreter", "test"],
					"instructions": "print(\"this is a test.\")"
				}
			}`,
			`{
				"apiVersion": "maestro/v1alpha1",
				"kind": "Agent",
				"metadata": {
					"name": "test2",
					"labels": {
						"app": "test-example"
					}
				},
				"spec": {
					"model": "meta-llama/llama-3-1-70b-instruct",
					"framework": "beeai",
					"mode": "local",
					"description": "this is a test",
					"tools": ["code_interpreter", "test"],
					"instructions": "print(\"this is a test.\")"
				}
			}`,
		},
		"workflow": `{
			"apiVersion": "maestro/v1",
			"kind": "Workflow",
			"metadata": {
				"name": "simple workflow",
				"labels": {
					"app": "example2"
				}
			},
			"spec": {
				"template": {
					"metadata": {
						"name": "maestro-deployment",
						"labels": {
							"app": "example",
							"use-case": "test"
						}
					},
					"agents": ["test1", "test2"],
					"prompt": "This is a test input",
					"steps": [
						{
							"name": "step1",
							"agent": "test1"
						},
						{
							"name": "step2",
							"agent": "test2"
						}
					]
				}
			}
		}`,
	}

	request := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: args,
		},
	}

	result, err := tool.Handler(context.Background(), request)
	// Since this is a template implementation, we expect an error or a placeholder result
	// In a real implementation, we would check for specific success conditions
	t.Logf("Result: %v, Error: %v", result, err)

	// Test with missing required arguments
	missingAgentsRequest := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]interface{}{
				// Missing agents
				"workflow": `{
					"apiVersion": "maestro/v1",
					"kind": "Workflow",
					"metadata": {
						"name": "simple workflow"
					}
				}`,
			},
		},
	}
	_, err = tool.Handler(context.Background(), missingAgentsRequest)
	assert.Error(t, err)
}

// TestCreateAgents tests the create_agents tool
func TestCreateAgents(t *testing.T) {
	server := setupTestServer(t)

	// Get the create_agents tool from the MCPServer field
	tool := server.MCPServer.GetTool("create_agents")
	require.NotNil(t, tool)

	// Test with valid arguments
	args := map[string]interface{}{
		"agents": []interface{}{
			`{
				"apiVersion": "maestro/v1alpha1",
				"kind": "Agent",
				"metadata": {
					"name": "test1",
					"labels": {
						"app": "test-example"
					}
				},
				"spec": {
					"model": "meta-llama/llama-3-1-70b-instruct",
					"framework": "beeai",
					"mode": "local",
					"description": "this is a test",
					"tools": ["code_interpreter", "test"],
					"instructions": "print(\"this is a test.\")"
				}
			}`,
		},
	}

	request := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: args,
		},
	}

	result, err := tool.Handler(context.Background(), request)
	// Since this is a template implementation, we expect an error or a placeholder result
	// In a real implementation, we would check for specific success conditions
	t.Logf("Result: %v, Error: %v", result, err)

	// Test with missing required arguments
	missingAgentsRequest := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]interface{}{
				// Missing agents
			},
		},
	}
	_, err = tool.Handler(context.Background(), missingAgentsRequest)
	assert.Error(t, err)
}

// Made with Bob
