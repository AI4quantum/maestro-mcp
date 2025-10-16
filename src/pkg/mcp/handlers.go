package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/AI4quantum/maestro-mcp/src/pkg/maestro"
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
	agentsRaw, ok := args["agents"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("agents is required and must be a list")
	}

	workflow, ok := args["workflow"].(string)
	if !ok {
		return nil, fmt.Errorf("workflow is required and must be a string")
	}

	// Parse workflow definition
	var workflowDef map[string]interface{}
	if err := json.Unmarshal([]byte(workflow), &workflowDef); err != nil {
		return nil, fmt.Errorf("invalid workflow definition: %w", err)
	}

	// Parse agent definitions
	agentDefs := make([]map[string]interface{}, 0, len(agentsRaw))
	for i, agentRaw := range agentsRaw {
		agentStr, ok := agentRaw.(string)
		if !ok {
			return nil, fmt.Errorf("agent at index %d is not a string", i)
		}

		var agentDef map[string]interface{}
		if err := json.Unmarshal([]byte(agentStr), &agentDef); err != nil {
			return nil, fmt.Errorf("invalid agent definition at index %d: %w", i, err)
		}
		agentDefs = append(agentDefs, agentDef)
	}

	// Generate workflow ID
	workflowID := fmt.Sprintf("wf-%s", time.Now().Format("20060102-150405"))

	// Create workflow execution context with timeout
	execCtx, cancel := context.WithTimeout(ctx, s.config.GetTimeout("run_workflow"))
	defer cancel()

	// Create workflow instance
	workflowObj, err := maestro.NewWorkflow(
		agentDefs,
		workflowDef,
		workflowID,
		s.logger,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create workflow: %w", err)
	}
	defer workflowObj.Close()

	// Extract prompt from workflow definition if available
	var prompt string
	if template, ok := workflowDef["spec"].(map[string]interface{}); ok {
		if templateObj, ok := template["template"].(map[string]interface{}); ok {
			if p, ok := templateObj["prompt"].(string); ok {
				prompt = p
			}
		}
	}

	// Run the workflow
	result, err := workflowObj.Run(execCtx, prompt)
	if err != nil {
		return nil, fmt.Errorf("workflow execution failed: %w", err)
	}

	s.logger.Info("Running workflow",
		zap.Int("agent_count", len(agentDefs)),
		zap.String("workflow_id", workflowID),
		zap.String("workflow_preview", workflow[:min(20, len(workflow))]))

	return map[string]interface{}{
		"status":       "ok",
		"message":      fmt.Sprintf("Successfully ran workflow with %d agents", len(agentDefs)),
		"workflow_id":  workflowID,
		"final_prompt": result.FinalPrompt,
		"step_results": result.StepResults,
	}, nil
}

// handleCreateAgents handles the create_agents tool
func (s *Server) handleCreateAgents(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	agentsRaw, ok := args["agents"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("agents is required and must be a list")
	}

	// Parse agent definitions
	agentDefs := make([]map[string]interface{}, 0, len(agentsRaw))
	for i, agentRaw := range agentsRaw {
		agentStr, ok := agentRaw.(string)
		if !ok {
			return nil, fmt.Errorf("agent at index %d is not a string", i)
		}

		var agentDef map[string]interface{}
		if err := json.Unmarshal([]byte(agentStr), &agentDef); err != nil {
			return nil, fmt.Errorf("invalid agent definition at index %d: %w", i, err)
		}
		agentDefs = append(agentDefs, agentDef)
	}

	// TODO: Implement agent creation logic
	// This would involve creating the agents from the parsed definitions

	s.logger.Info("Created agents",
		zap.Int("agent_count", len(agentDefs)))

	return map[string]interface{}{
		"status":  "ok",
		"message": fmt.Sprintf("Successfully created %d agents", len(agentDefs)),
	}, nil
}

// handleCreateTools handles the create_tools tool
func (s *Server) handleCreateTools(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	toolsRaw, ok := args["tools"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("tools is required and must be a list")
	}

	// Parse tool definitions
	toolDefs := make([]map[string]interface{}, 0, len(toolsRaw))
	for i, toolRaw := range toolsRaw {
		toolStr, ok := toolRaw.(string)
		if !ok {
			return nil, fmt.Errorf("tool at index %d is not a string", i)
		}

		var toolDef map[string]interface{}
		if err := json.Unmarshal([]byte(toolStr), &toolDef); err != nil {
			return nil, fmt.Errorf("invalid tool definition at index %d: %w", i, err)
		}
		toolDefs = append(toolDefs, toolDef)
	}

	// TODO: Implement tool creation logic
	// This would involve registering the tools from the parsed definitions

	s.logger.Info("Created tools",
		zap.Int("tool_count", len(toolDefs)))

	return map[string]interface{}{
		"status":  "ok",
		"message": fmt.Sprintf("Successfully created %d tools", len(toolDefs)),
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

	// Parse agent definition if needed
	var agentDef map[string]interface{}
	if err := json.Unmarshal([]byte(agent), &agentDef); err != nil {
		return nil, fmt.Errorf("invalid agent definition: %w", err)
	}

	// Create a temporary file to store the agent definition
	tempAgentFile := fmt.Sprintf("agent_%s.yaml", time.Now().Format("20060102_150405"))

	// TODO: Convert agent definition to YAML and write to file

	// Serve the agent
	go func() {
		if err := maestro.ServeAgent(tempAgentFile, agentName, host, port); err != nil {
			s.logger.Error("Failed to serve agent",
				zap.String("agent_name", agentName),
				zap.String("host", host),
				zap.Int("port", port),
				zap.Error(err))
		}
	}()

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
	// Create temporary files to store the agent and workflow definitions
	tempAgentsFile := fmt.Sprintf("agents_%s.yaml", time.Now().Format("20060102_150405"))
	tempWorkflowFile := fmt.Sprintf("workflow_%s.yaml", time.Now().Format("20060102_150405"))

	// Write agents to file
	if err := os.WriteFile(tempAgentsFile, []byte(agents), 0644); err != nil {
		return nil, fmt.Errorf("failed to write agents to file: %w", err)
	}

	// Write workflow to file
	if err := os.WriteFile(tempWorkflowFile, []byte(workflow), 0644); err != nil {
		// Clean up agents file
		os.Remove(tempAgentsFile)
		return nil, fmt.Errorf("failed to write workflow to file: %w", err)
	}

	// Serve the workflow in a goroutine
	go func() {
		if err := maestro.ServeWorkflow(tempAgentsFile, tempWorkflowFile, host, port); err != nil {
			s.logger.Error("Failed to serve workflow",
				zap.String("host", host),
				zap.Int("port", port),
				zap.Error(err))

			// Clean up temporary files
			os.Remove(tempAgentsFile)
			os.Remove(tempWorkflowFile)
		}
	}()

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
	agent, ok := args["agent"].(string)
	if !ok {
		return nil, fmt.Errorf("agent is required and must be a string")
	}

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

	// Create a temporary file to store the agent definition
	tempAgentFile := fmt.Sprintf("agent_%s.yaml", time.Now().Format("20060102_150405"))

	// Write agent definition to file
	if err := os.WriteFile(tempAgentFile, []byte(agent), 0644); err != nil {
		return nil, fmt.Errorf("failed to write agent to file: %w", err)
	}

	// Create and deploy the containerized agent
	go func() {
		if err := maestro.CreateContaineredAgent(tempAgentFile, agentName, host, port, s.logger); err != nil {
			s.logger.Error("Failed to create containerized agent",
				zap.String("agent_name", agentName),
				zap.Error(err))

			// Clean up temporary file
			os.Remove(tempAgentFile)
		}
	}()

	s.logger.Info("Creating containerized agent",
		zap.String("agent_name", agentName),
		zap.String("host", host),
		zap.Int("port", port))

	return map[string]interface{}{
		"status":  "ok",
		"message": fmt.Sprintf("Successfully started creating containerized agent %s", agentName),
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

// Helper function to convert map[string]string to map[string]interface{}
// This function is used in handleRunWorkflow
// nolint:unused
func convertMapToStringMap(m map[string]string) map[string]interface{} {
	result := make(map[string]interface{})
	for k, v := range m {
		result[k] = v
	}
	return result
}
