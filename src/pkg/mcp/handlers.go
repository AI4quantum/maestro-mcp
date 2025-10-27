package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/AI4quantum/maestro-mcp/src/pkg/config"
	"github.com/AI4quantum/maestro-mcp/src/pkg/maestro"
	"github.com/AI4quantum/maestro-mcp/src/pkg/vectordb"
	"github.com/mark3labs/mcp-go/mcp"
	"go.uber.org/zap"
)

// ServerState holds the state needed by handler functions
type ServerState struct {
	Config    *config.Config
	Logger    *zap.Logger
	VectorDBs map[string]vectordb.VectorDatabase
	DBMutex   sync.RWMutex
}

// Global server state that will be used by all handler functions
var GlobalServerState *ServerState

// getDatabaseByName returns a vector database by name
func getDatabaseByName(dbName string) (vectordb.VectorDatabase, error) {
	GlobalServerState.DBMutex.RLock()
	defer GlobalServerState.DBMutex.RUnlock()

	db, exists := GlobalServerState.VectorDBs[dbName]
	if !exists {
		return nil, fmt.Errorf("vector database '%s' not found. Please create it first", dbName)
	}

	return db, nil
}

// handleCreateVectorDatabase handles the create_vector_database tool
func handleCreateVectorDatabase(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// Extract arguments from request
	args := request.Params.Arguments.(map[string]interface{})

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

	GlobalServerState.DBMutex.Lock()
	defer GlobalServerState.DBMutex.Unlock()

	// Check if database already exists
	if _, exists := GlobalServerState.VectorDBs[dbName]; exists {
		return nil, fmt.Errorf("vector database '%s' already exists", dbName)
	}

	// Create vector database
	db, err := vectordb.CreateVectorDatabase(dbType, collectionName, GlobalServerState.Config)
	if err != nil {
		return nil, fmt.Errorf("failed to create vector database: %w", err)
	}

	GlobalServerState.VectorDBs[dbName] = db

	GlobalServerState.Logger.Info("Created vector database",
		zap.String("name", dbName),
		zap.String("type", dbType),
		zap.String("collection", collectionName))

	return mcp.NewToolResultText(fmt.Sprintf("Successfully created %s vector database '%s' with collection '%s'",
		dbType, dbName, collectionName)), nil
}

// handleListDatabases handles the list_databases tool
func handleListDatabases(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	GlobalServerState.DBMutex.RLock()
	defer GlobalServerState.DBMutex.RUnlock()

	//if len(GlobalServerState.VectorDBs) == 0 {
	//	// For demonstration purposes, we'll return nil for now
	//	return nil, nil
	//}

	dbList := make([]map[string]interface{}, 0, len(GlobalServerState.VectorDBs))
	for dbName, db := range GlobalServerState.VectorDBs {
		GlobalServerState.Logger.Info(dbName)
		count, err := db.CountDocuments(ctx)
		if err != nil {
			GlobalServerState.Logger.Warn("Failed to count documents",
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

	response, err := mcp.NewToolResultJSON(map[string]interface{}{"databases": dbList})
	return response, err
}

// handleSetupDatabase handles the setup_database tool
func handleSetupDatabase(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// Extract arguments from request
	args := request.Params.Arguments.(map[string]interface{})

	dbName, ok := args["db_name"].(string)
	if !ok {
		return nil, fmt.Errorf("db_name is required and must be a string")
	}

	embedding := "default"
	if emb, ok := args["embedding"].(string); ok {
		embedding = emb
	}

	db, err := getDatabaseByName(dbName)
	if err != nil {
		return nil, err
	}

	// Set up the database with timeout
	setupCtx, cancel := context.WithTimeout(ctx, GlobalServerState.Config.GetTimeout("setup_database"))
	defer cancel()

	if err := db.Setup(setupCtx, embedding); err != nil {
		return nil, fmt.Errorf("failed to set up vector database: %w", err)
	}

	GlobalServerState.Logger.Info("Set up vector database",
		zap.String("name", dbName),
		zap.String("embedding", embedding))

	return mcp.NewToolResultText(fmt.Sprintf("Successfully set up %s vector database '%s' with embedding '%s'",
		db.Type(), dbName, embedding)), nil
}

// handleWriteDocument handles the write_document tool
func handleWriteDocument(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// Extract arguments from request
	args := request.Params.Arguments.(map[string]interface{})

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

	db, err := getDatabaseByName(dbName)
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
	writeCtx, cancel := context.WithTimeout(ctx, GlobalServerState.Config.GetTimeout("write_single"))
	defer cancel()

	stats, err := db.WriteDocument(writeCtx, document)
	if err != nil {
		return nil, fmt.Errorf("failed to write document: %w", err)
	}

	GlobalServerState.Logger.Info("Wrote document",
		zap.String("db_name", dbName),
		zap.String("url", url))

	response, err := mcp.NewToolResultJSON(map[string]interface{}{
		"status":      "ok",
		"message":     "Wrote 1 document",
		"write_stats": stats,
	})
	return response, err
}

// handleQuery handles the query tool
func handleQuery(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// Extract arguments from request
	args := request.Params.Arguments.(map[string]interface{})

	dbName, ok := args["db_name"].(string)
	if !ok {
		return nil, fmt.Errorf("db_name is required and must be a string")
	}

	query, ok := args["query"].(string)
	if !ok {
		return nil, fmt.Errorf("query is required and must be a string")
	}

	db, err := getDatabaseByName(dbName)
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
	queryCtx, cancel := context.WithTimeout(ctx, GlobalServerState.Config.GetTimeout("query"))
	defer cancel()

	// Use _ to ignore the result variable since we're not using it
	result, err := db.Query(queryCtx, query, limit, collectionName)
	if err != nil {
		return nil, fmt.Errorf("failed to query vector database: %w", err)
	}

	GlobalServerState.Logger.Info("Executed query",
		zap.String("db_name", dbName),
		zap.String("query", query),
		zap.Int("limit", limit))

	return mcp.NewToolResultText(result.(string)), nil
}

// handleListDocuments handles the list_documents tool
func handleListDocuments(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// Extract arguments from request
	args := request.Params.Arguments.(map[string]interface{})

	dbName, ok := args["db_name"].(string)
	if !ok {
		return nil, fmt.Errorf("db_name is required and must be a string")
	}

	db, err := getDatabaseByName(dbName)
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
	listCtx, cancel := context.WithTimeout(ctx, GlobalServerState.Config.GetTimeout("list_documents"))
	defer cancel()

	documents, err := db.ListDocuments(listCtx, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list documents: %w", err)
	}

	GlobalServerState.Logger.Info("Listed documents",
		zap.String("db_name", dbName),
		zap.Int("limit", limit),
		zap.Int("offset", offset),
		zap.Int("count", len(documents)))

	response, err := mcp.NewToolResultJSON(map[string]interface{}{
		"documents": documents,
		"count":     len(documents),
	})
	return response, err
}

// handleCountDocuments handles the count_documents tool
func handleCountDocuments(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// Extract arguments from request
	args := request.Params.Arguments.(map[string]interface{})

	dbName, ok := args["db_name"].(string)
	if !ok {
		return nil, fmt.Errorf("db_name is required and must be a string")
	}

	db, err := getDatabaseByName(dbName)
	if err != nil {
		return nil, err
	}

	// Count documents with timeout
	countCtx, cancel := context.WithTimeout(ctx, GlobalServerState.Config.GetTimeout("count_documents"))
	defer cancel()

	count, err := db.CountDocuments(countCtx)
	if err != nil {
		return nil, fmt.Errorf("failed to count documents: %w", err)
	}

	GlobalServerState.Logger.Info("Counted documents",
		zap.String("db_name", dbName),
		zap.Int("count", count))

	response, err := mcp.NewToolResultJSON(map[string]interface{}{
		"count": count,
	})
	return response, err
}

// handleDeleteDocument handles the delete_document tool
func handleDeleteDocument(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// Extract arguments from request
	args := request.Params.Arguments.(map[string]interface{})

	dbName, ok := args["db_name"].(string)
	if !ok {
		return nil, fmt.Errorf("db_name is required and must be a string")
	}

	documentID, ok := args["document_id"].(string)
	if !ok {
		return nil, fmt.Errorf("document_id is required and must be a string")
	}

	db, err := getDatabaseByName(dbName)
	if err != nil {
		return nil, err
	}

	// Delete document with timeout
	deleteCtx, cancel := context.WithTimeout(ctx, GlobalServerState.Config.GetTimeout("delete"))
	defer cancel()

	if err := db.DeleteDocument(deleteCtx, documentID); err != nil {
		return nil, fmt.Errorf("failed to delete document: %w", err)
	}

	GlobalServerState.Logger.Info("Deleted document",
		zap.String("db_name", dbName),
		zap.String("document_id", documentID))

	return mcp.NewToolResultText(fmt.Sprintf("Successfully deleted document '%s' from vector database '%s'",
		documentID, dbName)), nil
}

// handleCleanup handles the cleanup tool
func handleCleanup(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// Extract arguments from request
	args := request.Params.Arguments.(map[string]interface{})

	dbName, ok := args["db_name"].(string)
	if !ok {
		return nil, fmt.Errorf("db_name is required and must be a string")
	}

	GlobalServerState.DBMutex.Lock()
	defer GlobalServerState.DBMutex.Unlock()

	db, exists := GlobalServerState.VectorDBs[dbName]
	if !exists {
		return nil, fmt.Errorf("vector database '%s' not found", dbName)
	}

	// Cleanup with timeout
	cleanupCtx, cancel := context.WithTimeout(ctx, GlobalServerState.Config.GetTimeout("cleanup"))
	defer cancel()

	if err := db.Cleanup(cleanupCtx); err != nil {
		return nil, fmt.Errorf("failed to cleanup vector database: %w", err)
	}

	delete(GlobalServerState.VectorDBs, dbName)

	GlobalServerState.Logger.Info("Cleaned up vector database",
		zap.String("name", dbName))

	return mcp.NewToolResultText(fmt.Sprintf("Successfully cleaned up and removed vector database '%s'", dbName)), nil
}

// handleRunWorkflow handles the run_workflow tool
func handleRunWorkflow(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// Extract arguments from request
	args := request.Params.Arguments.(map[string]interface{})

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

	// Parse agent definitions and agent names
	agentDefs := make([]map[string]interface{}, 0)
	agentList := make([]string, 0)

	for i, agentRaw := range agentsRaw {
		switch agent := agentRaw.(type) {
		case string:
			// Try to unmarshal as JSON to see if it's an agent definition
			var agentDef map[string]interface{}
			if err := json.Unmarshal([]byte(agent), &agentDef); err == nil {
				// It's a valid JSON object, so it's an agent definition
				agentDefs = append(agentDefs, agentDef)
			} else {
				// It's not a valid JSON object, so it's an agent name
				agentList = append(agentList, agent)
			}
		case map[string]interface{}:
			// It's already a map, so it's an agent definition
			agentDefs = append(agentDefs, agent)
		default:
			return nil, fmt.Errorf("agent at index %d is not a string or map", i)
		}
	}

	// Generate workflow ID
	workflowID := fmt.Sprintf("wf-%s", time.Now().Format("20060102-150405"))

	// Create workflow execution context with timeout
	execCtx, cancel := context.WithTimeout(ctx, GlobalServerState.Config.GetTimeout("run_workflow"))
	defer cancel()

	// Create workflow instance
	workflowObj, err := maestro.NewWorkflow(
		agentDefs,
		agentList,
		workflowDef,
		workflowID,
		GlobalServerState.Logger,
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
	fmt.Println(result)

	GlobalServerState.Logger.Info("Running workflow",
		zap.Int("agent_count", len(agentDefs)),
		zap.String("workflow_id", workflowID),
		zap.String("workflow_preview", workflow[:min(20, len(workflow))]))

	response, err := mcp.NewToolResultJSON(
		map[string]interface{}{
			"status":       "ok",
			"message":      fmt.Sprintf("Successfully ran workflow with %d agents", len(agentDefs)),
			"workflow_id":  workflowID,
			"final_prompt": result.FinalPrompt,
			"step_results": result.StepResults,
		})

	return response, err
}

// handleCreateAgents handles the create_agents tool
func handleCreateAgents(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// Extract arguments from request
	args := request.Params.Arguments.(map[string]interface{})

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

	err := maestro.CreateAgents(agentDefs)
	if err != nil {
		return nil, fmt.Errorf("create agents failed: %w", err)
	}

	GlobalServerState.Logger.Info("Created agents",
		zap.Int("agent_count", len(agentDefs)))

	response, err := mcp.NewToolResultJSON(
		map[string]interface{}{
			"status":  "ok",
			"message": fmt.Sprintf("Successfully %d agents created", len(agentDefs)),
		})

	return response, err
}

// handleCreateTools handles the create_tools tool
func handleCreateTools(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// Extract arguments from request
	args := request.Params.Arguments.(map[string]interface{})

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

	err := maestro.CreateMCPTools(toolDefs)
	if err != nil {
		return nil, fmt.Errorf("create tools failed: %w", err)
	}

	GlobalServerState.Logger.Info("Created tools",
		zap.Int("tool_count", len(toolDefs)))

	response, err := mcp.NewToolResultJSON(
		map[string]interface{}{
			"status":  "ok",
			"message": fmt.Sprintf("Successfully %d toolss created", len(toolDefs)),
		})

	return response, err

}

// handleServeAgent handles the serve_agent tool
func handleServeAgent(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// Extract arguments from request
	args := request.Params.Arguments.(map[string]interface{})

	agents, ok := args["agent"].(string)
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
	tempAgentsFile := fmt.Sprintf("agent_%s.yaml", time.Now().Format("20060102_150405"))

	// Write agents to file
	if err := os.WriteFile(tempAgentsFile, []byte(agents), 0644); err != nil {
		return nil, fmt.Errorf("failed to write agents to file: %w", err)
	}

	// Serve the agent
	go func() {
		if err := maestro.ServeAgent(tempAgentsFile, agentName, host, port); err != nil {
			GlobalServerState.Logger.Error("Failed to serve agent",
				zap.String("agent_name", agentName),
				zap.String("host", host),
				zap.Int("port", port),
				zap.Error(err))
		}
	}()

	GlobalServerState.Logger.Info("Serving agent",
		zap.String("agent_name", agentName),
		zap.String("host", host),
		zap.Int("port", port))

	response, err := mcp.NewToolResultJSON(
		map[string]interface{}{
			"status":  "ok",
			"message": fmt.Sprintf("Successfully started serving agent on %s:%d", host, port),
		})

	return response, err
}

// handleServeWorkflow handles the serve_workflow tool
func handleServeWorkflow(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// Extract arguments from request
	args := request.Params.Arguments.(map[string]interface{})

	agents, ok := args["agents"].(string)
	if !ok {
		return nil, fmt.Errorf("agents is required and must be a string")
	}
	fmt.Println(agents)

	workflow, ok := args["workflow"].(string)
	if !ok {
		return nil, fmt.Errorf("workflow is required and must be a string")
	}

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
			GlobalServerState.Logger.Error("Failed to serve workflow",
				zap.String("host", host),
				zap.Int("port", port),
				zap.Error(err))

			// Clean up temporary files
			os.Remove(tempAgentsFile)
			os.Remove(tempWorkflowFile)
		}
	}()

	GlobalServerState.Logger.Info("Serving workflow",
		zap.String("host", host),
		zap.Int("port", port))

	response, err := mcp.NewToolResultJSON(
		map[string]interface{}{
			"status":  "ok",
			"message": fmt.Sprintf("Successfully started serving workflow on %s:%d", host, port),
		})

	return response, err
}

// handleServeContainerAgent handles the serve_container_agent tool
func handleServeContainerAgent(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// Extract arguments from request
	args := request.Params.Arguments.(map[string]interface{})

	imageURL, ok := args["image_url"].(string)
	if !ok {
		return nil, fmt.Errorf("image_url is required and must be a string")
	}

	agentName := ""
	if an, ok := args["app_name"].(string); ok {
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

	// Create and deploy the containerized agent
	go func() {
		if err := maestro.CreateContaineredAgent(imageURL, agentName, host, port, GlobalServerState.Logger); err != nil {
			GlobalServerState.Logger.Error("Failed to create containerized agent",
				zap.String("agent_name", agentName),
				zap.Error(err))
		}
	}()

	GlobalServerState.Logger.Info("Creating containerized agent",
		zap.String("agent_name", agentName),
		zap.String("host", host),
		zap.Int("port", port))

	response, err := mcp.NewToolResultJSON(
		map[string]interface{}{
			"status":  "ok",
			"message": fmt.Sprintf("Successfully started creating containerized agent %s", agentName),
		})

	return response, err
}

// handleDeployWorkflow handles the deploy_workflow tool
func handleDeployWorkflow(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// Extract arguments from request
	args := request.Params.Arguments.(map[string]interface{})

	agents, ok := args["agents"].(string)
	if !ok {
		return nil, fmt.Errorf("agents is required and must be a string")
	}

	workflow, ok := args["workflow"].(string)
	if !ok {
		return nil, fmt.Errorf("workflow is required and must be a string")
	}

	target := "streamlit"
	if t, ok := args["target"].(string); ok {
		target = t
	}

	env := ""
	if e, ok := args["env"].(string); ok {
		env = e
	}

	deploy := maestro.NewDeploy(agents, workflow, env, target, GlobalServerState.Logger)
	if target == "docker" {
		err := deploy.DeployToDocker()
		if err != nil {
			GlobalServerState.Logger.Error("Failed to deploy to docker",
				zap.Error(err))
		}
	} else if target == "kubernetes" {
		err := deploy.DeployToKubernetes()
		if err != nil {
			GlobalServerState.Logger.Error("Failed to deploy to kubernetes",
				zap.Error(err))

		}
	}
	// TODO: Implement workflow deployment logic
	// This would involve deploying the workflow to the specified target

	GlobalServerState.Logger.Info("Deploying workflow",
		zap.String("target", target),
		zap.String("env", env))

	response, err := mcp.NewToolResultJSON(
		map[string]interface{}{
			"status":  "ok",
			"message": fmt.Sprintf("Successfully deployed workflow to %s", target),
		})

	return response, err
}

// Helper function for string length comparison
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
