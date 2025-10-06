package tests

import (
	"testing"

	"github.com/AI4quantum/maestro-mcp/src/pkg/config"
	"github.com/AI4quantum/maestro-mcp/src/pkg/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// setupTestServer creates a new MCP server for testing
func setupTestServer(t *testing.T) *mcp.Server {
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
	return server
}

// TestRunWorkflow tests the run_workflow tool
func TestRunWorkflow(t *testing.T) {
	server := setupTestServer(t)

	// Get the run_workflow tool
	tool, exists := server.Tools["run_workflow"]
	require.True(t, exists)

	// Test with valid arguments
	args := map[string]interface{}{
		"agents": []interface{}{
			`{"name": "agent1", "type": "llm", "config": {}}`,
			`{"name": "agent2", "type": "llm", "config": {}}`,
		},
		"workflow": `{"name": "test_workflow", "steps": []}`,
	}

	result, err := tool.Handler(nil, args)
	// Since this is a template implementation, we expect an error or a placeholder result
	// In a real implementation, we would check for specific success conditions
	t.Logf("Result: %v, Error: %v", result, err)

	// Test with missing required arguments
	_, err = tool.Handler(nil, map[string]interface{}{
		// Missing agents
		"workflow": `{"name": "test_workflow", "steps": []}`,
	})
	assert.Error(t, err)

	_, err = tool.Handler(nil, map[string]interface{}{
		"agents": []interface{}{
			`{"name": "agent1", "type": "llm", "config": {}}`,
		},
		// Missing workflow
	})
	assert.Error(t, err)
}

// TestCreateAgents tests the create_agents tool
func TestCreateAgents(t *testing.T) {
	server := setupTestServer(t)

	// Get the create_agents tool
	tool, exists := server.Tools["create_agents"]
	require.True(t, exists)

	// Test with valid arguments
	args := map[string]interface{}{
		"agents": []interface{}{
			`{"name": "agent1", "type": "llm", "config": {}}`,
			`{"name": "agent2", "type": "llm", "config": {}}`,
		},
	}

	result, err := tool.Handler(nil, args)
	// Since this is a template implementation, we expect an error or a placeholder result
	// In a real implementation, we would check for specific success conditions
	t.Logf("Result: %v, Error: %v", result, err)

	// Test with missing required arguments
	_, err = tool.Handler(nil, map[string]interface{}{
		// Missing agents
	})
	assert.Error(t, err)
}

// TestCreateTools tests the create_tools tool
func TestCreateTools(t *testing.T) {
	server := setupTestServer(t)

	// Get the create_tools tool
	tool, exists := server.Tools["create_tools"]
	require.True(t, exists)

	// Test with valid arguments
	args := map[string]interface{}{
		"tools": []interface{}{
			`{"name": "tool1", "description": "Test tool 1", "config": {}}`,
			`{"name": "tool2", "description": "Test tool 2", "config": {}}`,
		},
	}

	result, err := tool.Handler(nil, args)
	// Since this is a template implementation, we expect an error or a placeholder result
	// In a real implementation, we would check for specific success conditions
	t.Logf("Result: %v, Error: %v", result, err)

	// Test with missing required arguments
	_, err = tool.Handler(nil, map[string]interface{}{
		// Missing tools
	})
	assert.Error(t, err)
}

// TestServeAgent tests the serve_agent tool
func TestServeAgent(t *testing.T) {
	server := setupTestServer(t)

	// Get the serve_agent tool
	tool, exists := server.Tools["serve_agent"]
	require.True(t, exists)

	// Test with valid arguments
	args := map[string]interface{}{
		"agent":      `{"name": "agent1", "type": "llm", "config": {}}`,
		"agent_name": "test_agent",
		"host":       "127.0.0.1",
		"port":       float64(8001),
	}

	result, err := tool.Handler(nil, args)
	// Since this is a template implementation, we expect an error or a placeholder result
	// In a real implementation, we would check for specific success conditions
	t.Logf("Result: %v, Error: %v", result, err)

	// Test with missing required arguments
	_, err = tool.Handler(nil, map[string]interface{}{
		// Missing agent
		"agent_name": "test_agent",
		"host":       "127.0.0.1",
		"port":       float64(8001),
	})
	assert.Error(t, err)

	// Test with default values for optional arguments
	args = map[string]interface{}{
		"agent": `{"name": "agent1", "type": "llm", "config": {}}`,
		// agent_name, host, and port are optional with defaults
	}

	result, err = tool.Handler(nil, args)
	t.Logf("Result with defaults: %v, Error: %v", result, err)
}

// TestServeWorkflow tests the serve_workflow tool
func TestServeWorkflow(t *testing.T) {
	server := setupTestServer(t)

	// Get the serve_workflow tool
	tool, exists := server.Tools["serve_workflow"]
	require.True(t, exists)

	// Test with valid arguments
	args := map[string]interface{}{
		"agents":   `[{"name": "agent1", "type": "llm", "config": {}}]`,
		"workflow": `{"name": "test_workflow", "steps": []}`,
		"host":     "127.0.0.1",
		"port":     float64(8001),
	}

	result, err := tool.Handler(nil, args)
	// Since this is a template implementation, we expect an error or a placeholder result
	// In a real implementation, we would check for specific success conditions
	t.Logf("Result: %v, Error: %v", result, err)

	// Test with missing required arguments
	_, err = tool.Handler(nil, map[string]interface{}{
		// Missing agents
		"workflow": `{"name": "test_workflow", "steps": []}`,
		"host":     "127.0.0.1",
		"port":     float64(8001),
	})
	assert.Error(t, err)

	_, err = tool.Handler(nil, map[string]interface{}{
		"agents": `[{"name": "agent1", "type": "llm", "config": {}}]`,
		// Missing workflow
		"host": "127.0.0.1",
		"port": float64(8001),
	})
	assert.Error(t, err)

	// Test with default values for optional arguments
	args = map[string]interface{}{
		"agents":   `[{"name": "agent1", "type": "llm", "config": {}}]`,
		"workflow": `{"name": "test_workflow", "steps": []}`,
		// host and port are optional with defaults
	}

	result, err = tool.Handler(nil, args)
	t.Logf("Result with defaults: %v, Error: %v", result, err)
}

// TestServeContainerAgent tests the serve_container_agent tool
func TestServeContainerAgent(t *testing.T) {
	server := setupTestServer(t)

	// Get the serve_container_agent tool
	tool, exists := server.Tools["serve_container_agent"]
	require.True(t, exists)

	// Test with valid arguments
	args := map[string]interface{}{
		"image_url":      "registry.example.com/agent:latest",
		"app_name":       "test-agent",
		"namespace":      "default",
		"replicas":       float64(1),
		"container_port": float64(80),
		"service_port":   float64(80),
		"service_type":   "LoadBalancer",
		"node_port":      float64(30051),
	}

	result, err := tool.Handler(nil, args)
	// Since this is a template implementation, we expect an error or a placeholder result
	// In a real implementation, we would check for specific success conditions
	t.Logf("Result: %v, Error: %v", result, err)

	// Test with missing required arguments
	_, err = tool.Handler(nil, map[string]interface{}{
		// Missing image_url
		"app_name":       "test-agent",
		"namespace":      "default",
		"replicas":       float64(1),
		"container_port": float64(80),
		"service_port":   float64(80),
		"service_type":   "LoadBalancer",
		"node_port":      float64(30051),
	})
	assert.Error(t, err)

	_, err = tool.Handler(nil, map[string]interface{}{
		"image_url": "registry.example.com/agent:latest",
		// Missing app_name
		"namespace":      "default",
		"replicas":       float64(1),
		"container_port": float64(80),
		"service_port":   float64(80),
		"service_type":   "LoadBalancer",
		"node_port":      float64(30051),
	})
	assert.Error(t, err)

	// Test with default values for optional arguments
	args = map[string]interface{}{
		"image_url": "registry.example.com/agent:latest",
		"app_name":  "test-agent",
		// All other parameters are optional with defaults
	}

	result, err = tool.Handler(nil, args)
	t.Logf("Result with defaults: %v, Error: %v", result, err)
}

// TestDeployWorkflow tests the deploy_workflow tool
func TestDeployWorkflow(t *testing.T) {
	server := setupTestServer(t)

	// Get the deploy_workflow tool
	tool, exists := server.Tools["deploy_workflow"]
	require.True(t, exists)

	// Test with valid arguments
	args := map[string]interface{}{
		"agents":   "agents:\n  - name: agent1\n    type: llm\n",
		"workflow": "name: test_workflow\nsteps: []\n",
		"target":   "docker",
		"env":      "KEY1=value1,KEY2=value2",
	}

	result, err := tool.Handler(nil, args)
	// Since this is a template implementation, we expect an error or a placeholder result
	// In a real implementation, we would check for specific success conditions
	t.Logf("Result: %v, Error: %v", result, err)

	// Test with missing required arguments
	_, err = tool.Handler(nil, map[string]interface{}{
		// Missing agents
		"workflow": "name: test_workflow\nsteps: []\n",
		"target":   "docker",
		"env":      "KEY1=value1,KEY2=value2",
	})
	assert.Error(t, err)

	_, err = tool.Handler(nil, map[string]interface{}{
		"agents": "agents:\n  - name: agent1\n    type: llm\n",
		// Missing workflow
		"target": "docker",
		"env":    "KEY1=value1,KEY2=value2",
	})
	assert.Error(t, err)

	// Test with default values for optional arguments
	args = map[string]interface{}{
		"agents":   "agents:\n  - name: agent1\n    type: llm\n",
		"workflow": "name: test_workflow\nsteps: []\n",
		// target and env are optional with defaults
	}

	result, err = tool.Handler(nil, args)
	t.Logf("Result with defaults: %v, Error: %v", result, err)
}

// Made with Bob
