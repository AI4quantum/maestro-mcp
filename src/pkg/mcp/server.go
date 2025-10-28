package mcp

import (
	"sync"

	"github.com/AI4quantum/maestro-mcp/src/pkg/config"
	"github.com/AI4quantum/maestro-mcp/src/pkg/vectordb"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"go.uber.org/zap"
)

// Server represents the MCP server implementation
type Server struct {
	config    *config.Config
	logger    *zap.Logger
	vectorDBs map[string]vectordb.VectorDatabase
	MCPServer *server.MCPServer
}

// NewServer creates a new MCP server
func NewServer(cfg *config.Config, logger *zap.Logger) (*Server, error) {
	s := &Server{
		config:    cfg,
		logger:    logger,
		vectorDBs: make(map[string]vectordb.VectorDatabase),
		MCPServer: server.NewMCPServer("maestro", "1.0.0"),
	}

	// Initialize the global server state for handlers
	GlobalServerState = &ServerState{
		Config:    cfg,
		Logger:    logger,
		VectorDBs: s.vectorDBs,
		DBMutex:   sync.RWMutex{},
	}

	// Register tools
	s.registerTools()

	return s, nil
}

// registerTools registers all available MCP tools
func (s *Server) registerTools() {
	// Database management tools
	s.MCPServer.AddTool(mcp.Tool{
		Name:        "create_vector_database",
		Description: "Create a new vector database instance",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"db_name": map[string]interface{}{
					"type":        "string",
					"description": "Unique name for the vector database instance",
				},
				"db_type": map[string]interface{}{
					"type":        "string",
					"description": "Type of vector database to create",
					"enum":        []string{"weaviate", "milvus"},
				},
				"collection_name": map[string]interface{}{
					"type":        "string",
					"description": "Name of the collection to use",
					"default":     "MaestroDocs",
				},
			},
			Required: []string{"db_name", "db_type"},
		},
	},
		handleCreateVectorDatabase,
	)

	s.MCPServer.AddTool(mcp.Tool{
		Name:        "list_databases",
		Description: "List all available vector database instances",
		InputSchema: mcp.ToolInputSchema{
			Type:       "object",
			Properties: map[string]interface{}{},
		},
	},
		handleListDatabases,
	)

	s.MCPServer.AddTool(mcp.Tool{
		Name:        "setup_database",
		Description: "Set up a vector database and create collections",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"db_name": map[string]interface{}{
					"type":        "string",
					"description": "Name of the vector database instance to set up",
				},
				"embedding": map[string]interface{}{
					"type":        "string",
					"description": "Embedding model to use for the collection",
					"default":     "default",
				},
			},
			Required: []string{"db_name"},
		},
	},
		handleSetupDatabase,
	)

	// Document operations
	s.MCPServer.AddTool(mcp.Tool{
		Name:        "write_document",
		Description: "Write a single document to a vector database",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"db_name": map[string]interface{}{
					"type":        "string",
					"description": "Name of the vector database instance",
				},
				"url": map[string]interface{}{
					"type":        "string",
					"description": "URL of the document",
				},
				"text": map[string]interface{}{
					"type":        "string",
					"description": "Text content of the document",
				},
				"metadata": map[string]interface{}{
					"type":        "object",
					"description": "Additional metadata for the document",
					"default":     map[string]interface{}{},
				},
				"vector": map[string]interface{}{
					"type":        "array",
					"description": "Pre-computed vector embedding (optional)",
					"items": map[string]interface{}{
						"type": "number",
					},
				},
			},
			Required: []string{"db_name", "url", "text"},
		},
	},
		handleWriteDocument,
	)

	s.MCPServer.AddTool(mcp.Tool{
		Name:        "query",
		Description: "Query a vector database using natural language",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"db_name": map[string]interface{}{
					"type":        "string",
					"description": "Name of the vector database instance",
				},
				"query": map[string]interface{}{
					"type":        "string",
					"description": "The query string to search for",
				},
				"limit": map[string]interface{}{
					"type":        "integer",
					"description": "Maximum number of results to consider",
					"default":     5,
				},
				"collection_name": map[string]interface{}{
					"type":        "string",
					"description": "Optional collection name to search in",
				},
			},
			Required: []string{"db_name", "query"},
		},
	},
		handleQuery,
	)

	s.MCPServer.AddTool(mcp.Tool{
		Name:        "list_documents",
		Description: "List documents from a vector database",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"db_name": map[string]interface{}{
					"type":        "string",
					"description": "Name of the vector database instance",
				},
				"limit": map[string]interface{}{
					"type":        "integer",
					"description": "Maximum number of documents to return",
					"default":     10,
				},
				"offset": map[string]interface{}{
					"type":        "integer",
					"description": "Number of documents to skip",
					"default":     0,
				},
			},
			Required: []string{"db_name"},
		},
	},
		handleListDocuments,
	)

	s.MCPServer.AddTool(mcp.Tool{
		Name:        "count_documents",
		Description: "Get the current count of documents in a collection",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"db_name": map[string]interface{}{
					"type":        "string",
					"description": "Name of the vector database instance",
				},
			},
			Required: []string{"db_name"},
		},
	},
		handleCountDocuments,
	)

	s.MCPServer.AddTool(mcp.Tool{
		Name:        "delete_document",
		Description: "Delete a single document from a vector database",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"db_name": map[string]interface{}{
					"type":        "string",
					"description": "Name of the vector database instance",
				},
				"document_id": map[string]interface{}{
					"type":        "string",
					"description": "Document ID to delete",
				},
			},
			Required: []string{"db_name", "document_id"},
		},
	},
		handleDeleteDocument,
	)

	s.MCPServer.AddTool(mcp.Tool{
		Name:        "cleanup",
		Description: "Clean up resources and close connections for a vector database",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"db_name": map[string]interface{}{
					"type":        "string",
					"description": "Name of the vector database instance to clean up",
				},
			},
			Required: []string{"db_name"},
		},
	},
		handleCleanup,
	)

	// Workflow and agent tools from Python MCP server
	s.MCPServer.AddTool(mcp.Tool{
		Name:        "run_workflow",
		Description: "Run workflow with specified agents and workflow definitions",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"agents": map[string]interface{}{
					"type":        "array",
					"description": "List of agent definitions",
					"items": map[string]interface{}{
						"type": "string",
					},
				},
				"workflow": map[string]interface{}{
					"type":        "string",
					"description": "Workflow definition",
				},
			},
			Required: []string{"agents", "workflow"},
		},
	},
		handleRunWorkflow,
	)

	s.MCPServer.AddTool(mcp.Tool{
		Name:        "create_agents",
		Description: "Create agents from a list of agent definitions",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"agents": map[string]interface{}{
					"type":        "array",
					"description": "List of agent definitions",
					"items": map[string]interface{}{
						"type": "string",
					},
				},
			},
			Required: []string{"agents"},
		},
	},
		handleCreateAgents,
	)

	s.MCPServer.AddTool(mcp.Tool{
		Name:        "create_tools",
		Description: "Create tools from a list of tool definitions",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"tools": map[string]interface{}{
					"type":        "array",
					"description": "List of tool definitions",
					"items": map[string]interface{}{
						"type": "string",
					},
				},
			},
			Required: []string{"tools"},
		},
	},
		handleCreateTools,
	)

	s.MCPServer.AddTool(mcp.Tool{
		Name:        "serve_agent",
		Description: "Serve an agent on a specified host and port",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"agent": map[string]interface{}{
					"type":        "string",
					"description": "Agent definition",
				},
				"agent_name": map[string]interface{}{
					"type":        "string",
					"description": "Agent name in agent_definitions",
				},
				"host": map[string]interface{}{
					"type":        "string",
					"description": "Server IP",
					"default":     "127.0.0.1",
				},
				"port": map[string]interface{}{
					"type":        "integer",
					"description": "Server port",
					"default":     8001,
				},
			},
			Required: []string{"agent"},
		},
	},
		handleServeAgent,
	)

	s.MCPServer.AddTool(mcp.Tool{
		Name:        "serve_workflow",
		Description: "Serve a workflow on a specified host and port",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"agents": map[string]interface{}{
					"type":        "string",
					"description": "List of agent definitions",
				},
				"workflow": map[string]interface{}{
					"type":        "string",
					"description": "Workflow definition",
				},
				"host": map[string]interface{}{
					"type":        "string",
					"description": "Server IP",
					"default":     "127.0.0.1",
				},
				"port": map[string]interface{}{
					"type":        "integer",
					"description": "Server port",
					"default":     8001,
				},
			},
			Required: []string{"agents", "workflow"},
		},
	},
		handleServeWorkflow,
	)

	s.MCPServer.AddTool(mcp.Tool{
		Name:        "serve_container_agent",
		Description: "Serve a containerized agent in a Kubernetes cluster",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"image_url": map[string]interface{}{
					"type":        "string",
					"description": "Agent container image registry URL",
				},
				"app_name": map[string]interface{}{
					"type":        "string",
					"description": "App name in the cluster",
				},
				"namespace": map[string]interface{}{
					"type":        "string",
					"description": "Kubernetes namespace",
					"default":     "default",
				},
				"replicas": map[string]interface{}{
					"type":        "integer",
					"description": "Number of replicas",
					"default":     1,
				},
				"container_port": map[string]interface{}{
					"type":        "integer",
					"description": "Container port",
					"default":     80,
				},
				"service_port": map[string]interface{}{
					"type":        "integer",
					"description": "Service port",
					"default":     80,
				},
				"service_type": map[string]interface{}{
					"type":        "string",
					"description": "Service type",
					"default":     "LoadBalancer",
				},
				"node_port": map[string]interface{}{
					"type":        "integer",
					"description": "Node port",
					"default":     30051,
				},
			},
			Required: []string{"image_url", "app_name"},
		},
	},
		handleServeContainerAgent,
	)

	s.MCPServer.AddTool(mcp.Tool{
		Name:        "deploy_workflow",
		Description: "Deploy a workflow to a target environment",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"agents": map[string]interface{}{
					"type":        "string",
					"description": "Agents yaml file contents",
				},
				"workflow": map[string]interface{}{
					"type":        "string",
					"description": "Workflow yaml file contents",
				},
				"target": map[string]interface{}{
					"type":        "string",
					"description": "Deploy target type (docker, kubernetes, or streamlit)",
					"default":     "streamlit",
				},
				"env": map[string]interface{}{
					"type":        "string",
					"description": "Environment variables set into container. Format: list of key=value string. Each key=value is separated by ','",
					"default":     "",
				},
			},
			Required: []string{"agents", "workflow"},
		},
	},
		handleDeployWorkflow,
	)
}

// Handler returns the HTTP handler for the MCP server
func (s *Server) Handler() interface{} {
	// In a real implementation, we would return the actual handler
	// For demonstration purposes, we'll return nil
	return nil
}

// Made with Bob
