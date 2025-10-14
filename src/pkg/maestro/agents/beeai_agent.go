// SPDX-License-Identifier: Apache-2.0
// Copyright © 2025 IBM

package agents

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"text/template"
)

// BeeAIAgent extends the BaseAgent to interact with BeeAI framework
type BeeAIAgent struct {
	*BaseAgent
	MCPStack       *sync.WaitGroup
	Agent          interface{}
	OutputTemplate *template.Template
}

// NewBeeAIAgent creates a new BeeAIAgent
func NewBeeAIAgent(agent map[string]interface{}) (interface{}, error) {
	// Create the base agent
	baseAgent, err := NewBaseAgent(agent)
	if err != nil {
		return nil, err
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

	return &BeeAIAgent{
		BaseAgent:      baseAgent,
		MCPStack:       &sync.WaitGroup{},
		OutputTemplate: outputTemplate,
	}, nil
}

// Run implements the Agent interface Run method
func (b *BeeAIAgent) Run(args ...interface{}) (interface{}, error) {
	if len(args) == 0 {
		return nil, fmt.Errorf("no prompt provided")
	}

	prompt, ok := args[0].(string)
	if !ok {
		return nil, fmt.Errorf("prompt must be a string")
	}

	// Extract context if provided
	var context map[string]interface{}
	if len(args) > 1 {
		if ctx, ok := args[1].(map[string]interface{}); ok {
			context = ctx
		}
	}

	// Extract step index if provided
	var stepIndex int
	if len(args) > 2 {
		if idx, ok := args[2].(int); ok {
			stepIndex = idx
		}
	}

	b.Print(fmt.Sprintf("Running %s with prompt...", b.AgentName))

	// Determine BeeAI URL
	beeaiURL := b.AgentURL
	if beeaiURL == "" {
		beeaiURL = "http://localhost:8080"
	}

	// Ensure URL ends with /
	if !strings.HasSuffix(beeaiURL, "/") {
		beeaiURL += "/"
	}

	// Prepare request parameters
	params := map[string]interface{}{
		"prompt":       prompt,
		"model":        b.AgentModel,
		"instructions": b.AgentInstr,
		"tools":        b.AgentTools,
		"code":         b.AgentCode,
	}

	// Add context and step index if available
	if context != nil {
		params["context"] = context
	}
	if stepIndex > 0 {
		params["step_index"] = stepIndex
	}

	// Call the BeeAI API
	result, err := b.callBeeAIAPI(beeaiURL, params)
	if err != nil {
		return nil, err
	}

	// Track token usage
	b.TrackTokens(prompt, result)

	// Render output template
	var buf bytes.Buffer
	err = b.OutputTemplate.Execute(&buf, map[string]interface{}{
		"result": result,
		"prompt": prompt,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to render output template: %w", err)
	}

	answer := buf.String()
	b.Print(fmt.Sprintf("Response from %s: %s\n", b.AgentName, answer))

	return answer, nil
}

// RunStreaming implements streaming for the BeeAIAgent
func (b *BeeAIAgent) RunStreaming(args ...interface{}) (interface{}, error) {
	// For now, streaming is the same as regular Run
	// In a real implementation, we would use a streaming API
	return b.Run(args...)
}

// callBeeAIAPI calls the BeeAI API with the given parameters
func (b *BeeAIAgent) callBeeAIAPI(beeaiURL string, params map[string]interface{}) (string, error) {
	// Prepare request URL
	url := fmt.Sprintf("%srun", beeaiURL)

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

	// Add authorization if available
	if token := os.Getenv("BEEAI_API_KEY"); token != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	}

	// Send request
	client := &http.Client{
		Timeout: 120 * 1000000000, // 120 seconds
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
		Result string `json:"result"`
		Text   string `json:"text"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		// If we can't parse the response as JSON, return it as-is
		return string(respBody), nil
	}

	// Return the result or text field, whichever is available
	if result.Result != "" {
		return result.Result, nil
	}
	return result.Text, nil
}

// getMCPTools gets tools from MCP
func (b *BeeAIAgent) getMCPTools(toolName string) ([]interface{}, error) {
	// This is a simplified implementation
	// In a real implementation, we would call the MCP API to get tools
	b.Print(fmt.Sprintf("Getting MCP tools for %s...", toolName))
	return []interface{}{}, nil
}

// Made with Bob
