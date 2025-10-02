package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/AI4quantum/maestro-mcp/src/pkg/config"
	"github.com/AI4quantum/maestro-mcp/src/pkg/vectordb"
	"go.uber.org/zap"
)

// Server represents the MCP server implementation
type Server struct {
	config    *config.Config
	logger    *zap.Logger
	vectorDBs map[string]vectordb.VectorDatabase
	dbMutex   sync.RWMutex
	Tools     map[string]Tool
}

// Tool represents an MCP tool
type Tool struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	InputSchema map[string]interface{} `json:"inputSchema"`
	Handler     func(ctx context.Context, args map[string]interface{}) (interface{}, error)
}

// NewServer creates a new MCP server
func NewServer(cfg *config.Config, logger *zap.Logger) (*Server, error) {
	server := &Server{
		config:    cfg,
		logger:    logger,
		vectorDBs: make(map[string]vectordb.VectorDatabase),
		Tools:     make(map[string]Tool),
	}

	// Register tools
	server.registerTools()

	return server, nil
}

// Handler returns the HTTP handler for the MCP server
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()

	// Health check endpoint
	mux.HandleFunc("/health", s.handleHealth)

	// MCP endpoints
	mux.HandleFunc("/mcp/tools/list", s.handleToolsList)
	mux.HandleFunc("/mcp/tools/call", s.handleToolCall)

	return mux
}

// registerTools registers all available MCP tools
func (s *Server) registerTools() {
	// Database management tools
	s.registerTool(Tool{
		Name:        "create_vector_database",
		Description: "Create a new vector database instance",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
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
			"required": []string{"db_name", "db_type"},
		},
		Handler: s.handleCreateVectorDatabase,
	})

	s.registerTool(Tool{
		Name:        "list_databases",
		Description: "List all available vector database instances",
		InputSchema: map[string]interface{}{
			"type":       "object",
			"properties": map[string]interface{}{},
		},
		Handler: s.handleListDatabases,
	})

	s.registerTool(Tool{
		Name:        "setup_database",
		Description: "Set up a vector database and create collections",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
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
			"required": []string{"db_name"},
		},
		Handler: s.handleSetupDatabase,
	})

	// Document operations
	s.registerTool(Tool{
		Name:        "write_document",
		Description: "Write a single document to a vector database",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
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
			"required": []string{"db_name", "url", "text"},
		},
		Handler: s.handleWriteDocument,
	})

	s.registerTool(Tool{
		Name:        "query",
		Description: "Query a vector database using natural language",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
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
			"required": []string{"db_name", "query"},
		},
		Handler: s.handleQuery,
	})

	s.registerTool(Tool{
		Name:        "list_documents",
		Description: "List documents from a vector database",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
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
			"required": []string{"db_name"},
		},
		Handler: s.handleListDocuments,
	})

	s.registerTool(Tool{
		Name:        "count_documents",
		Description: "Get the current count of documents in a collection",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"db_name": map[string]interface{}{
					"type":        "string",
					"description": "Name of the vector database instance",
				},
			},
			"required": []string{"db_name"},
		},
		Handler: s.handleCountDocuments,
	})

	s.registerTool(Tool{
		Name:        "delete_document",
		Description: "Delete a single document from a vector database",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"db_name": map[string]interface{}{
					"type":        "string",
					"description": "Name of the vector database instance",
				},
				"document_id": map[string]interface{}{
					"type":        "string",
					"description": "Document ID to delete",
				},
			},
			"required": []string{"db_name", "document_id"},
		},
		Handler: s.handleDeleteDocument,
	})

	s.registerTool(Tool{
		Name:        "cleanup",
		Description: "Clean up resources and close connections for a vector database",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"db_name": map[string]interface{}{
					"type":        "string",
					"description": "Name of the vector database instance to clean up",
				},
			},
			"required": []string{"db_name"},
		},
		Handler: s.handleCleanup,
	})
}

// registerTool registers a tool with the server
func (s *Server) registerTool(tool Tool) {
	s.Tools[tool.Name] = tool
	s.logger.Debug("Registered tool", zap.String("name", tool.Name))
}

// handleHealth handles health check requests
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	s.dbMutex.RLock()
	dbCount := len(s.vectorDBs)
	s.dbMutex.RUnlock()

	response := map[string]interface{}{
		"status":           "healthy",
		"timestamp":        time.Now().UTC(),
		"vector_databases": dbCount,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		s.logger.Error("Failed to encode health response", zap.Error(err))
	}
}

// handleToolsList handles tool listing requests
func (s *Server) handleToolsList(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	tools := make([]map[string]interface{}, 0, len(s.Tools))
	for _, tool := range s.Tools {
		tools = append(tools, map[string]interface{}{
			"name":        tool.Name,
			"description": tool.Description,
			"inputSchema": tool.InputSchema,
		})
	}

	response := map[string]interface{}{
		"tools": tools,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		s.logger.Error("Failed to encode tools list response", zap.Error(err))
	}
}

// handleToolCall handles tool execution requests
func (s *Server) handleToolCall(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var request struct {
		Name      string                 `json:"name"`
		Arguments map[string]interface{} `json:"arguments"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	tool, exists := s.Tools[request.Name]
	if !exists {
		http.Error(w, fmt.Sprintf("Tool '%s' not found", request.Name), http.StatusNotFound)
		return
	}

	// Execute tool with timeout
	ctx, cancel := context.WithTimeout(r.Context(), s.config.GetTimeout("tool_call"))
	defer cancel()

	result, err := tool.Handler(ctx, request.Arguments)
	if err != nil {
		s.logger.Error("Tool execution failed",
			zap.String("tool", request.Name),
			zap.Error(err))

		response := map[string]interface{}{
			"error": err.Error(),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		if encodeErr := json.NewEncoder(w).Encode(response); encodeErr != nil {
			s.logger.Error("Failed to encode error response", zap.Error(encodeErr))
		}
		return
	}

	response := map[string]interface{}{
		"result": result,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		s.logger.Error("Failed to encode tool call response", zap.Error(err))
	}
}

// getDatabaseByName returns a vector database by name
func (s *Server) getDatabaseByName(dbName string) (vectordb.VectorDatabase, error) {
	s.dbMutex.RLock()
	defer s.dbMutex.RUnlock()

	db, exists := s.vectorDBs[dbName]
	if !exists {
		return nil, fmt.Errorf("vector database '%s' not found. Please create it first", dbName)
	}

	return db, nil
}
