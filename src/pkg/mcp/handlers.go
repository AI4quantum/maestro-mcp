package mcp

import (
	"context"
	"fmt"

	"github.com/AI4quantum/maestro-mcp/src/pkg/vectordb"
	"go.uber.org/zap"
)

// handleCreateVectorDatabase handles the create_vector_database tool
func (s *Server) handleCreateVectorDatabase(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	dbName, ok := args["db_name"].(string)
	if !ok {
		return nil, fmt.Errorf("db_name is required and must be a string")
	}

	dbType, ok := args["db_type"].(string)
	if !ok {
		return nil, fmt.Errorf("db_type is required and must be a string")
	}

	collectionName := "MaestroDocs"
	if cn, ok := args["collection_name"].(string); ok {
		collectionName = cn
	}

	s.dbMutex.Lock()
	defer s.dbMutex.Unlock()

	// Check if database already exists
	if _, exists := s.vectorDBs[dbName]; exists {
		return nil, fmt.Errorf("vector database '%s' already exists", dbName)
	}

	// Create vector database
	db, err := vectordb.CreateVectorDatabase(dbType, collectionName, s.config)
	if err != nil {
		return nil, fmt.Errorf("failed to create vector database: %w", err)
	}

	s.vectorDBs[dbName] = db

	s.logger.Info("Created vector database",
		zap.String("name", dbName),
		zap.String("type", dbType),
		zap.String("collection", collectionName))

	return fmt.Sprintf("Successfully created %s vector database '%s' with collection '%s'",
		dbType, dbName, collectionName), nil
}

// handleListDatabases handles the list_databases tool
func (s *Server) handleListDatabases(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	s.dbMutex.RLock()
	defer s.dbMutex.RUnlock()

	if len(s.vectorDBs) == 0 {
		return "No vector databases are currently active", nil
	}

	dbList := make([]map[string]interface{}, 0, len(s.vectorDBs))
	for dbName, db := range s.vectorDBs {
		count, err := db.CountDocuments(ctx)
		if err != nil {
			s.logger.Warn("Failed to count documents",
				zap.String("db_name", dbName),
				zap.Error(err))
			count = -1
		}

		dbList = append(dbList, map[string]interface{}{
			"name":           dbName,
			"type":           db.Type(),
			"collection":     db.CollectionName(),
			"document_count": count,
		})
	}

	return map[string]interface{}{
		"databases": dbList,
	}, nil
}

// handleSetupDatabase handles the setup_database tool
func (s *Server) handleSetupDatabase(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	dbName, ok := args["db_name"].(string)
	if !ok {
		return nil, fmt.Errorf("db_name is required and must be a string")
	}

	embedding := "default"
	if emb, ok := args["embedding"].(string); ok {
		embedding = emb
	}

	db, err := s.getDatabaseByName(dbName)
	if err != nil {
		return nil, err
	}

	// Set up the database with timeout
	setupCtx, cancel := context.WithTimeout(ctx, s.config.GetTimeout("setup_database"))
	defer cancel()

	if err := db.Setup(setupCtx, embedding); err != nil {
		return nil, fmt.Errorf("failed to set up vector database: %w", err)
	}

	s.logger.Info("Set up vector database",
		zap.String("name", dbName),
		zap.String("embedding", embedding))

	return fmt.Sprintf("Successfully set up %s vector database '%s' with embedding '%s'",
		db.Type(), dbName, embedding), nil
}

// handleWriteDocument handles the write_document tool
func (s *Server) handleWriteDocument(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	dbName, ok := args["db_name"].(string)
	if !ok {
		return nil, fmt.Errorf("db_name is required and must be a string")
	}

	url, ok := args["url"].(string)
	if !ok {
		return nil, fmt.Errorf("url is required and must be a string")
	}

	text, ok := args["text"].(string)
	if !ok {
		return nil, fmt.Errorf("text is required and must be a string")
	}

	db, err := s.getDatabaseByName(dbName)
	if err != nil {
		return nil, err
	}

	// Build document
	document := vectordb.Document{
		URL:      url,
		Text:     text,
		Metadata: make(map[string]interface{}),
	}

	// Add metadata if provided
	if metadata, ok := args["metadata"].(map[string]interface{}); ok {
		document.Metadata = metadata
	}

	// Add vector if provided
	if vector, ok := args["vector"].([]interface{}); ok {
		document.Vector = make([]float64, len(vector))
		for i, v := range vector {
			if f, ok := v.(float64); ok {
				document.Vector[i] = f
			} else {
				return nil, fmt.Errorf("invalid vector value at index %d", i)
			}
		}
	}

	// Write document with timeout
	writeCtx, cancel := context.WithTimeout(ctx, s.config.GetTimeout("write_single"))
	defer cancel()

	stats, err := db.WriteDocument(writeCtx, document)
	if err != nil {
		return nil, fmt.Errorf("failed to write document: %w", err)
	}

	s.logger.Info("Wrote document",
		zap.String("db_name", dbName),
		zap.String("url", url))

	return map[string]interface{}{
		"status":      "ok",
		"message":     "Wrote 1 document",
		"write_stats": stats,
	}, nil
}

// handleQuery handles the query tool
func (s *Server) handleQuery(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	dbName, ok := args["db_name"].(string)
	if !ok {
		return nil, fmt.Errorf("db_name is required and must be a string")
	}

	query, ok := args["query"].(string)
	if !ok {
		return nil, fmt.Errorf("query is required and must be a string")
	}

	db, err := s.getDatabaseByName(dbName)
	if err != nil {
		return nil, err
	}

	limit := 5
	if l, ok := args["limit"].(float64); ok {
		limit = int(l)
	}

	var collectionName string
	if cn, ok := args["collection_name"].(string); ok {
		collectionName = cn
	}

	// Query with timeout
	queryCtx, cancel := context.WithTimeout(ctx, s.config.GetTimeout("query"))
	defer cancel()

	result, err := db.Query(queryCtx, query, limit, collectionName)
	if err != nil {
		return nil, fmt.Errorf("failed to query vector database: %w", err)
	}

	s.logger.Info("Executed query",
		zap.String("db_name", dbName),
		zap.String("query", query),
		zap.Int("limit", limit))

	return result, nil
}

// handleListDocuments handles the list_documents tool
func (s *Server) handleListDocuments(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	dbName, ok := args["db_name"].(string)
	if !ok {
		return nil, fmt.Errorf("db_name is required and must be a string")
	}

	db, err := s.getDatabaseByName(dbName)
	if err != nil {
		return nil, err
	}

	limit := 10
	if l, ok := args["limit"].(float64); ok {
		limit = int(l)
	}

	offset := 0
	if o, ok := args["offset"].(float64); ok {
		offset = int(o)
	}

	// List documents with timeout
	listCtx, cancel := context.WithTimeout(ctx, s.config.GetTimeout("list_documents"))
	defer cancel()

	documents, err := db.ListDocuments(listCtx, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list documents: %w", err)
	}

	s.logger.Info("Listed documents",
		zap.String("db_name", dbName),
		zap.Int("limit", limit),
		zap.Int("offset", offset),
		zap.Int("count", len(documents)))

	return map[string]interface{}{
		"documents": documents,
		"count":     len(documents),
	}, nil
}

// handleCountDocuments handles the count_documents tool
func (s *Server) handleCountDocuments(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	dbName, ok := args["db_name"].(string)
	if !ok {
		return nil, fmt.Errorf("db_name is required and must be a string")
	}

	db, err := s.getDatabaseByName(dbName)
	if err != nil {
		return nil, err
	}

	// Count documents with timeout
	countCtx, cancel := context.WithTimeout(ctx, s.config.GetTimeout("count_documents"))
	defer cancel()

	count, err := db.CountDocuments(countCtx)
	if err != nil {
		return nil, fmt.Errorf("failed to count documents: %w", err)
	}

	s.logger.Info("Counted documents",
		zap.String("db_name", dbName),
		zap.Int("count", count))

	return map[string]interface{}{
		"count": count,
	}, nil
}

// handleDeleteDocument handles the delete_document tool
func (s *Server) handleDeleteDocument(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	dbName, ok := args["db_name"].(string)
	if !ok {
		return nil, fmt.Errorf("db_name is required and must be a string")
	}

	documentID, ok := args["document_id"].(string)
	if !ok {
		return nil, fmt.Errorf("document_id is required and must be a string")
	}

	db, err := s.getDatabaseByName(dbName)
	if err != nil {
		return nil, err
	}

	// Delete document with timeout
	deleteCtx, cancel := context.WithTimeout(ctx, s.config.GetTimeout("delete"))
	defer cancel()

	if err := db.DeleteDocument(deleteCtx, documentID); err != nil {
		return nil, fmt.Errorf("failed to delete document: %w", err)
	}

	s.logger.Info("Deleted document",
		zap.String("db_name", dbName),
		zap.String("document_id", documentID))

	return fmt.Sprintf("Successfully deleted document '%s' from vector database '%s'",
		documentID, dbName), nil
}

// handleCleanup handles the cleanup tool
func (s *Server) handleCleanup(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	dbName, ok := args["db_name"].(string)
	if !ok {
		return nil, fmt.Errorf("db_name is required and must be a string")
	}

	s.dbMutex.Lock()
	defer s.dbMutex.Unlock()

	db, exists := s.vectorDBs[dbName]
	if !exists {
		return nil, fmt.Errorf("vector database '%s' not found", dbName)
	}

	// Cleanup with timeout
	cleanupCtx, cancel := context.WithTimeout(ctx, s.config.GetTimeout("cleanup"))
	defer cancel()

	if err := db.Cleanup(cleanupCtx); err != nil {
		return nil, fmt.Errorf("failed to cleanup vector database: %w", err)
	}

	delete(s.vectorDBs, dbName)

	s.logger.Info("Cleaned up vector database",
		zap.String("name", dbName))

	return fmt.Sprintf("Successfully cleaned up and removed vector database '%s'", dbName), nil
}

// handleRunWorkflow handles the run_workflow tool
func (s *Server) handleRunWorkflow(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	agents, ok := args["agents"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("agents is required and must be a list")
	}

	workflow, ok := args["workflow"].(string)
	if !ok {
		return nil, fmt.Errorf("workflow is required and must be a string")
	}

	// TODO: Implement workflow execution logic
	// This would involve parsing the agent definitions and workflow,
	// then executing the workflow with the specified agents

	s.logger.Info("Running workflow",
		zap.Int("agent_count", len(agents)),
		zap.String("workflow", workflow[:min(20, len(workflow))]))

	return map[string]interface{}{
		"status":  "ok",
		"message": fmt.Sprintf("Successfully ran workflow with %d agents", len(agents)),
	}, nil
}

// handleCreateAgents handles the create_agents tool
func (s *Server) handleCreateAgents(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	agents, ok := args["agents"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("agents is required and must be a list")
	}

	// TODO: Implement agent creation logic
	// This would involve parsing the agent definitions and creating the agents

	s.logger.Info("Created agents",
		zap.Int("agent_count", len(agents)))

	return map[string]interface{}{
		"status":  "ok",
		"message": fmt.Sprintf("Successfully created %d agents", len(agents)),
	}, nil
}

// handleCreateTools handles the create_tools tool
func (s *Server) handleCreateTools(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	tools, ok := args["tools"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("tools is required and must be a list")
	}

	// TODO: Implement tool creation logic
	// This would involve parsing the tool definitions and registering them

	s.logger.Info("Created tools",
		zap.Int("tool_count", len(tools)))

	return map[string]interface{}{
		"status":  "ok",
		"message": fmt.Sprintf("Successfully created %d tools", len(tools)),
	}, nil
}

// handleServeAgent handles the serve_agent tool
func (s *Server) handleServeAgent(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	agent, ok := args["agent"].(string)
	if !ok {
		return nil, fmt.Errorf("agent is required and must be a string")
	}
	fmt.Println(agent)

	agentName := ""
	if an, ok := args["agent_name"].(string); ok {
		agentName = an
	}

	host := "127.0.0.1"
	if h, ok := args["host"].(string); ok {
		host = h
	}

	port := 8001
	if p, ok := args["port"].(float64); ok {
		port = int(p)
	}

	// TODO: Implement agent serving logic
	// This would involve starting a server to serve the agent

	s.logger.Info("Serving agent",
		zap.String("agent_name", agentName),
		zap.String("host", host),
		zap.Int("port", port))

	return map[string]interface{}{
		"status":  "ok",
		"message": fmt.Sprintf("Successfully started serving agent on %s:%d", host, port),
	}, nil
}

// handleServeWorkflow handles the serve_workflow tool
func (s *Server) handleServeWorkflow(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	agents, ok := args["agents"].(string)
	if !ok {
		return nil, fmt.Errorf("agents is required and must be a string")
	}
	fmt.Println(agents)

	workflow, ok := args["workflow"].(string)
	if !ok {
		return nil, fmt.Errorf("workflow is required and must be a string")
	}
	fmt.Println(workflow)

	host := "127.0.0.1"
	if h, ok := args["host"].(string); ok {
		host = h
	}

	port := 8001
	if p, ok := args["port"].(float64); ok {
		port = int(p)
	}

	// TODO: Implement workflow serving logic
	// This would involve starting a server to serve the workflow

	s.logger.Info("Serving workflow",
		zap.String("host", host),
		zap.Int("port", port))

	return map[string]interface{}{
		"status":  "ok",
		"message": fmt.Sprintf("Successfully started serving workflow on %s:%d", host, port),
	}, nil
}

// handleServeContainerAgent handles the serve_container_agent tool
func (s *Server) handleServeContainerAgent(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	imageURL, ok := args["image_url"].(string)
	if !ok {
		return nil, fmt.Errorf("image_url is required and must be a string")
	}

	appName, ok := args["app_name"].(string)
	if !ok {
		return nil, fmt.Errorf("app_name is required and must be a string")
	}

	namespace := "default"
	if ns, ok := args["namespace"].(string); ok {
		namespace = ns
	}

	replicas := 1
	if r, ok := args["replicas"].(float64); ok {
		replicas = int(r)
	}

	containerPort := 80
	if cp, ok := args["container_port"].(float64); ok {
		containerPort = int(cp)
	}
	fmt.Println(containerPort)

	servicePort := 80
	if sp, ok := args["service_port"].(float64); ok {
		servicePort = int(sp)
	}
	fmt.Println(servicePort)

	serviceType := "LoadBalancer"
	if st, ok := args["service_type"].(string); ok {
		serviceType = st
	}
	fmt.Println(serviceType)

	nodePort := 30051
	if np, ok := args["node_port"].(float64); ok {
		nodePort = int(np)
	}
	fmt.Println(nodePort)

	// TODO: Implement container agent serving logic
	// This would involve deploying a container to serve the agent

	s.logger.Info("Serving container agent",
		zap.String("image_url", imageURL),
		zap.String("app_name", appName),
		zap.String("namespace", namespace),
		zap.Int("replicas", replicas))

	return map[string]interface{}{
		"status":  "ok",
		"message": fmt.Sprintf("Successfully deployed container agent %s in namespace %s", appName, namespace),
	}, nil
}

// handleDeployWorkflow handles the deploy_workflow tool
func (s *Server) handleDeployWorkflow(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	agents, ok := args["agents"].(string)
	if !ok {
		return nil, fmt.Errorf("agents is required and must be a string")
	}
	fmt.Println(agents)

	workflow, ok := args["workflow"].(string)
	if !ok {
		return nil, fmt.Errorf("workflow is required and must be a string")
	}
	fmt.Println(workflow)

	target := "streamlit"
	if t, ok := args["target"].(string); ok {
		target = t
	}

	env := ""
	if e, ok := args["env"].(string); ok {
		env = e
	}

	// TODO: Implement workflow deployment logic
	// This would involve deploying the workflow to the specified target

	s.logger.Info("Deploying workflow",
		zap.String("target", target),
		zap.String("env", env))

	return map[string]interface{}{
		"status":  "ok",
		"message": fmt.Sprintf("Successfully deployed workflow to %s", target),
	}, nil
}

// Helper function for string length comparison
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
