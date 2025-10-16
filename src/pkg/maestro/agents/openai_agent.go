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
	"text/template"
)

// Constants for OpenAI agent
const (
	OpenAIDefaultURL   = "https://api.openai.com/v1"
	OpenAIDefaultModel = "gpt-4o-mini"
)

// OpenAIAgent extends the BaseAgent to interact with OpenAI API
type OpenAIAgent struct {
	*BaseAgent
	Client         *http.Client
	BaseURL        string
	APIKey         string
	MaxTokens      int
	ExtraHeaders   map[string]string
	UseLiteLLM     bool
	OutputTemplate *template.Template
}

// NewOpenAIAgent creates a new OpenAIAgent
func NewOpenAIAgent(agent map[string]interface{}) (interface{}, error) {
	// Create the base agent
	baseAgent, err := NewBaseAgent(agent)
	if err != nil {
		return nil, err
	}

	// Extract spec
	spec, ok := agent["spec"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid agent definition: missing spec")
	}

	// Get model with default
	model := OpenAIDefaultModel
	if modelVal, ok := spec["model"].(string); ok && modelVal != "" {
		model = modelVal
	}
	baseAgent.AgentModel = model

	// Get base URL with default
	baseURL := os.Getenv("OPENAI_BASE_URL")
	if baseURL == "" {
		baseURL = OpenAIDefaultURL
	}
	if urlVal, ok := spec["url"].(string); ok && urlVal != "" {
		baseURL = urlVal
	}

	// Get API key from environment
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		apiKey = "dummy_key" // Default for testing
	}

	// Check if LiteLLM should be used
	useLiteLLM := strings.ToLower(os.Getenv("MAESTRO_OPENAI_USE_LITELLM")) == "true"

	// Get max tokens from environment
	maxTokens := 0
	if maxTokensStr := os.Getenv("MAESTRO_OPENAI_MAX_TOKENS"); maxTokensStr != "" {
		if _, err := fmt.Sscanf(maxTokensStr, "%d", &maxTokens); err != nil {
			baseAgent.Print(fmt.Sprintf("WARN: Failed to parse MAESTRO_OPENAI_MAX_TOKENS: %v", err))
		}
	}

	// Get extra headers from environment
	extraHeaders := make(map[string]string)
	if headersStr := os.Getenv("MAESTRO_OPENAI_EXTRA_HEADERS"); headersStr != "" {
		if err := json.Unmarshal([]byte(headersStr), &extraHeaders); err != nil {
			baseAgent.Print(fmt.Sprintf("WARN: Failed to parse MAESTRO_OPENAI_EXTRA_HEADERS: %v", err))
		}
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

	// Create HTTP client
	client := &http.Client{
		Timeout: 120 * 1000000000, // 120 seconds
	}

	return &OpenAIAgent{
		BaseAgent:      baseAgent,
		Client:         client,
		BaseURL:        baseURL,
		APIKey:         apiKey,
		MaxTokens:      maxTokens,
		ExtraHeaders:   extraHeaders,
		UseLiteLLM:     useLiteLLM,
		OutputTemplate: outputTemplate,
	}, nil
}

// Run implements the Agent interface Run method
func (o *OpenAIAgent) Run(args ...interface{}) (interface{}, error) {
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

	// Check if streaming is enabled
	streamingOverride := strings.ToLower(os.Getenv("MAESTRO_OPENAI_STREAMING"))
	useStreaming := streamingOverride == "true"

	o.Print(fmt.Sprintf("Running %s with prompt...", o.AgentName))

	var result string
	var err error

	if useStreaming {
		result, err = o.runStreaming(prompt, context, stepIndex)
	} else {
		result, err = o.runNonStreaming(prompt, context, stepIndex)
	}

	if err != nil {
		return nil, err
	}

	// Track token usage
	o.TrackTokens(prompt, result)

	// Render output template
	var buf bytes.Buffer
	err = o.OutputTemplate.Execute(&buf, map[string]interface{}{
		"result": result,
		"prompt": prompt,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to render output template: %w", err)
	}

	answer := buf.String()
	o.Print(fmt.Sprintf("Response from %s: %s\n", o.AgentName, answer))

	return answer, nil
}

// RunStreaming implements streaming for the OpenAIAgent
func (o *OpenAIAgent) RunStreaming(args ...interface{}) (interface{}, error) {
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

	// Check if streaming is disabled
	streamingOverride := strings.ToLower(os.Getenv("MAESTRO_OPENAI_STREAMING"))
	if streamingOverride == "false" {
		o.Print("MAESTRO_OPENAI_STREAMING=false, using non-streaming mode")
		return o.Run(args...)
	}

	o.Print(fmt.Sprintf("Running %s with prompt (streaming)...", o.AgentName))

	result, err := o.runStreaming(prompt, context, stepIndex)
	if err != nil {
		return nil, err
	}

	// Track token usage
	o.TrackTokens(prompt, result)

	// Render output template
	var buf bytes.Buffer
	err = o.OutputTemplate.Execute(&buf, map[string]interface{}{
		"result": result,
		"prompt": prompt,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to render output template: %w", err)
	}

	answer := buf.String()
	o.Print(fmt.Sprintf("Response from %s (streaming): %s\n", o.AgentName, answer))

	return answer, nil
}

// runNonStreaming runs the agent in non-streaming mode
func (o *OpenAIAgent) runNonStreaming(prompt string, context map[string]interface{}, stepIndex int) (string, error) {
	// Prepare request parameters
	params := map[string]interface{}{
		"model": o.AgentModel,
		"messages": []map[string]interface{}{
			{
				"role":    "system",
				"content": o.AgentInstr,
			},
			{
				"role":    "user",
				"content": prompt,
			},
		},
		"temperature": 0.7,
	}

	// Add max tokens if specified
	if o.MaxTokens > 0 {
		params["max_tokens"] = o.MaxTokens
	}

	// Add context if provided
	if context != nil {
		params["context"] = context
	}

	// Call the OpenAI API
	result, err := o.callOpenAIAPI("/chat/completions", params, false)
	if err != nil {
		return "", err
	}

	return result, nil
}

// runStreaming runs the agent in streaming mode
func (o *OpenAIAgent) runStreaming(prompt string, context map[string]interface{}, stepIndex int) (string, error) {
	// Prepare request parameters
	params := map[string]interface{}{
		"model": o.AgentModel,
		"messages": []map[string]interface{}{
			{
				"role":    "system",
				"content": o.AgentInstr,
			},
			{
				"role":    "user",
				"content": prompt,
			},
		},
		"temperature": 0.7,
		"stream":      true,
	}

	// Add max tokens if specified
	if o.MaxTokens > 0 {
		params["max_tokens"] = o.MaxTokens
	}

	// Add context if provided
	if context != nil {
		params["context"] = context
	}

	// Call the OpenAI API
	result, err := o.callOpenAIAPI("/chat/completions", params, true)
	if err != nil {
		return "", err
	}

	return result, nil
}

// callOpenAIAPI calls the OpenAI API with the given parameters
func (o *OpenAIAgent) callOpenAIAPI(endpoint string, params map[string]interface{}, streaming bool) (string, error) {
	// Prepare request URL
	url := o.BaseURL + endpoint

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
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", o.APIKey))

	// Add extra headers if specified
	for key, value := range o.ExtraHeaders {
		req.Header.Set(key, value)
	}

	// Send request
	resp, err := o.Client.Do(req)
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
	if streaming {
		// For streaming, we would need to parse SSE format
		// This is a simplified implementation
		return string(respBody), nil
	}

	var result struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	if len(result.Choices) == 0 {
		return "", fmt.Errorf("no choices in response")
	}

	return result.Choices[0].Message.Content, nil
}

// Made with Bob
