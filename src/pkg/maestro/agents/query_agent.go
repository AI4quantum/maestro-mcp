// SPDX-License-Identifier: Apache-2.0
// Copyright © 2025 IBM

package agents

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"text/template"
)

// QueryAgent extends the BaseAgent to query vector databases
type QueryAgent struct {
	*BaseAgent
	DBName         string
	CollectionName string
	Limit          int
	OutputTemplate *template.Template
}

// NewQueryAgent creates a new QueryAgent
func NewQueryAgent(agent map[string]interface{}) (interface{}, error) {
	// Create the base agent
	baseAgent, err := NewBaseAgent(agent)
	if err != nil {
		return nil, err
	}

	// Extract metadata
	metadata, ok := agent["metadata"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid agent definition: missing metadata")
	}

	// Extract query_input
	queryInput, ok := metadata["query_input"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid agent definition: missing query_input in metadata")
	}

	// Extract DB name
	dbName, ok := queryInput["db_name"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid agent definition: missing db_name in query_input")
	}

	// Extract collection name with default
	collectionName := "MaestroDocs"
	if cn, ok := queryInput["collection_name"].(string); ok {
		collectionName = cn
	}

	// Extract limit with default
	limit := 10
	if l, ok := queryInput["limit"].(float64); ok {
		limit = int(l)
	}

	// Create output template
	outputTemplateStr := "{{.result}}"
	if baseAgent.AgentOutput != "" {
		outputTemplateStr = baseAgent.AgentOutput
	}

	outputTemplate, err := template.New("output").Parse(outputTemplateStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse output template: %w", err)
	}

	return &QueryAgent{
		BaseAgent:      baseAgent,
		DBName:         dbName,
		CollectionName: collectionName,
		Limit:          limit,
		OutputTemplate: outputTemplate,
	}, nil
}

// Run implements the Agent interface Run method
func (q *QueryAgent) Run(args ...interface{}) (interface{}, error) {
	if len(args) == 0 {
		return nil, fmt.Errorf("no prompt provided")
	}

	prompt, ok := args[0].(string)
	if !ok {
		return nil, fmt.Errorf("prompt must be a string")
	}

	q.Print(fmt.Sprintf("Running %s with prompt...", q.AgentName))

	// Determine MCP URL
	mcpURL := q.AgentURL
	if mcpURL == "" {
		mcpURL = "http://localhost:8030/mcp/"
	}

	// Ensure URL ends with /
	if !strings.HasSuffix(mcpURL, "/") {
		mcpURL += "/"
	}

	q.Print(fmt.Sprintf("Querying vector database '%s'...", q.DBName))

	// Prepare request parameters
	params := map[string]interface{}{
		"input": map[string]interface{}{
			"db_name":         q.DBName,
			"query":           prompt,
			"limit":           q.Limit,
			"collection_name": q.CollectionName,
		},
	}

	// Call the search tool
	result, err := q.callMCPTool(mcpURL, "search", params)
	if err != nil {
		return nil, err
	}

	// Parse the result
	var docs []map[string]interface{}
	if err := json.Unmarshal([]byte(result), &docs); err != nil {
		q.Print(fmt.Sprintf("ERROR [QueryAgent %s]: %s", q.AgentName, result))
		return result, nil
	}

	// Extract text from documents
	var texts []string
	for _, doc := range docs {
		if text, ok := doc["text"].(string); ok {
			texts = append(texts, text)
		}
	}

	// Join texts
	output := strings.Join(texts, "\n\n")

	// Render output template
	var buf bytes.Buffer
	err = q.OutputTemplate.Execute(&buf, map[string]interface{}{
		"result": output,
		"prompt": prompt,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to render output template: %w", err)
	}

	answer := buf.String()
	q.Print(fmt.Sprintf("Response from %s: %s\n", q.AgentName, answer))

	return answer, nil
}

// RunStreaming implements streaming for the QueryAgent
func (q *QueryAgent) RunStreaming(args ...interface{}) (interface{}, error) {
	// For QueryAgent, streaming is the same as regular Run
	return q.Run(args...)
}

// callMCPTool calls an MCP tool with the given parameters
func (q *QueryAgent) callMCPTool(mcpURL, toolName string, params map[string]interface{}) (string, error) {
	// Prepare request URL
	url := fmt.Sprintf("%stool/%s", mcpURL, toolName)

	// Prepare request body
	body, err := json.Marshal(params)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request body: %w", err)
	}

	// Create request
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(body))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")

	// Send request
	client := &http.Client{
		Timeout: 30 * 1000000000, // 30 seconds
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Read response
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	// Check response status
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("request failed with status code %d: %s", resp.StatusCode, string(respBody))
	}

	// Parse response
	var result struct {
		Data string `json:"data"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	return result.Data, nil
}

// Made with Bob
