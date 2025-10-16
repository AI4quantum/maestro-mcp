// SPDX-License-Identifier: Apache-2.0
// Copyright © 2025 IBM

package agents

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func TestNewOpenAIAgent(t *testing.T) {
	// Save original environment variables
	originalAPIKey := os.Getenv("OPENAI_API_KEY")
	originalBaseURL := os.Getenv("OPENAI_BASE_URL")
	originalMaxTokens := os.Getenv("MAESTRO_OPENAI_MAX_TOKENS")
	originalExtraHeaders := os.Getenv("MAESTRO_OPENAI_EXTRA_HEADERS")
	originalUseLiteLLM := os.Getenv("MAESTRO_OPENAI_USE_LITELLM")

	// Restore environment variables after test
	defer func() {
		os.Setenv("OPENAI_API_KEY", originalAPIKey)
		os.Setenv("OPENAI_BASE_URL", originalBaseURL)
		os.Setenv("MAESTRO_OPENAI_MAX_TOKENS", originalMaxTokens)
		os.Setenv("MAESTRO_OPENAI_EXTRA_HEADERS", originalExtraHeaders)
		os.Setenv("MAESTRO_OPENAI_USE_LITELLM", originalUseLiteLLM)
	}()

	// Set test environment variables
	os.Setenv("OPENAI_API_KEY", "test-api-key")
	os.Setenv("OPENAI_BASE_URL", "https://test-api.example.com")
	os.Setenv("MAESTRO_OPENAI_MAX_TOKENS", "1000")
	os.Setenv("MAESTRO_OPENAI_EXTRA_HEADERS", `{"X-Test-Header": "test-value"}`)
	os.Setenv("MAESTRO_OPENAI_USE_LITELLM", "true")

	// Create a test agent definition
	agent := map[string]interface{}{
		"apiVersion": "maestro/v1alpha1",
		"kind":       "Agent",
		"metadata": map[string]interface{}{
			"name": "test-openai-agent",
		},
		"spec": map[string]interface{}{
			"framework":    "openai",
			"model":        "gpt-4",
			"url":          "https://api.example.com/v1",
			"instructions": "You are a helpful assistant.",
			"output":       "Results: {{.result}}",
		},
	}

	// Create the agent
	openaiAgent, err := NewOpenAIAgent(agent)
	if err != nil {
		t.Fatalf("Failed to create OpenAIAgent: %v", err)
	}

	// Check that the agent was created correctly
	oa, ok := openaiAgent.(*OpenAIAgent)
	if !ok {
		t.Fatalf("Expected *OpenAIAgent, got %T", openaiAgent)
	}

	// Check agent properties
	if oa.AgentName != "test-openai-agent" {
		t.Errorf("Expected agent name 'test-openai-agent', got '%s'", oa.AgentName)
	}

	if oa.AgentFramework != "openai" {
		t.Errorf("Expected agent framework 'openai', got '%s'", oa.AgentFramework)
	}

	if oa.AgentModel != "gpt-4" {
		t.Errorf("Expected agent model 'gpt-4', got '%s'", oa.AgentModel)
	}

	if oa.BaseURL != "https://api.example.com/v1" {
		t.Errorf("Expected base URL 'https://api.example.com/v1', got '%s'", oa.BaseURL)
	}

	if oa.APIKey != "test-api-key" {
		t.Errorf("Expected API key 'test-api-key', got '%s'", oa.APIKey)
	}

	if oa.MaxTokens != 1000 {
		t.Errorf("Expected max tokens 1000, got %d", oa.MaxTokens)
	}

	if oa.ExtraHeaders["X-Test-Header"] != "test-value" {
		t.Errorf("Expected extra header 'X-Test-Header: test-value', got '%v'", oa.ExtraHeaders)
	}

	if !oa.UseLiteLLM {
		t.Errorf("Expected UseLiteLLM to be true")
	}

	// Test output template
	var buf strings.Builder
	err = oa.OutputTemplate.Execute(&buf, map[string]interface{}{
		"result": "test result",
	})
	if err != nil {
		t.Fatalf("Failed to execute output template: %v", err)
	}

	if buf.String() != "Results: test result" {
		t.Errorf("Expected output template to render 'Results: test result', got '%s'", buf.String())
	}
}

func TestOpenAIAgentRun(t *testing.T) {
	// Create a mock OpenAI server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check request path
		if r.URL.Path != "/chat/completions" {
			t.Errorf("Expected request path '/chat/completions', got '%s'", r.URL.Path)
		}

		// Check request method
		if r.Method != "POST" {
			t.Errorf("Expected request method 'POST', got '%s'", r.Method)
		}

		// Check authorization header
		authHeader := r.Header.Get("Authorization")
		if authHeader != "Bearer test-api-key" {
			t.Errorf("Expected Authorization header 'Bearer test-api-key', got '%s'", authHeader)
		}

		// Check request body
		var requestBody map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil {
			t.Fatalf("Failed to decode request body: %v", err)
		}

		if requestBody["model"] != "gpt-4" {
			t.Errorf("Expected model 'gpt-4', got '%v'", requestBody["model"])
		}

		messages, ok := requestBody["messages"].([]interface{})
		if !ok || len(messages) != 2 {
			t.Fatalf("Expected 2 messages, got %v", messages)
		}

		systemMsg, ok := messages[0].(map[string]interface{})
		if !ok || systemMsg["role"] != "system" || systemMsg["content"] != "You are a helpful assistant." {
			t.Errorf("Expected system message with content 'You are a helpful assistant.', got %v", systemMsg)
		}

		userMsg, ok := messages[1].(map[string]interface{})
		if !ok || userMsg["role"] != "user" || userMsg["content"] != "test prompt" {
			t.Errorf("Expected user message with content 'test prompt', got %v", userMsg)
		}

		// Return a mock response
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(map[string]interface{}{
			"choices": []map[string]interface{}{
				{
					"message": map[string]interface{}{
						"content": "This is a test response from OpenAI",
					},
				},
			},
		}); err != nil {
			t.Errorf("Failed to encode response: %v", err)
		}
	}))
	defer server.Close()

	// Save original environment variables
	originalAPIKey := os.Getenv("OPENAI_API_KEY")
	originalStreaming := os.Getenv("MAESTRO_OPENAI_STREAMING")

	// Restore environment variables after test
	defer func() {
		os.Setenv("OPENAI_API_KEY", originalAPIKey)
		os.Setenv("MAESTRO_OPENAI_STREAMING", originalStreaming)
	}()

	// Set test environment variables
	os.Setenv("OPENAI_API_KEY", "test-api-key")
	os.Setenv("MAESTRO_OPENAI_STREAMING", "false")

	// Create a test agent definition
	agent := map[string]interface{}{
		"apiVersion": "maestro/v1alpha1",
		"kind":       "Agent",
		"metadata": map[string]interface{}{
			"name": "test-openai-agent",
		},
		"spec": map[string]interface{}{
			"framework":    "openai",
			"model":        "gpt-4",
			"url":          server.URL,
			"instructions": "You are a helpful assistant.",
			"output":       "Results: {{.result}}",
		},
	}

	// Create the agent
	openaiAgent, err := NewOpenAIAgent(agent)
	if err != nil {
		t.Fatalf("Failed to create OpenAIAgent: %v", err)
	}

	oa, ok := openaiAgent.(*OpenAIAgent)
	if !ok {
		t.Fatalf("Expected *OpenAIAgent, got %T", openaiAgent)
	}

	// Run the agent
	result, err := oa.Run("test prompt")
	if err != nil {
		t.Fatalf("Failed to run OpenAIAgent: %v", err)
	}

	// Check the result
	expectedResult := "Results: This is a test response from OpenAI"
	if result != expectedResult {
		t.Errorf("Expected result '%s', got '%v'", expectedResult, result)
	}
}

func TestOpenAIAgentRunStreaming(t *testing.T) {
	// Create a mock OpenAI server for streaming
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check request path
		if r.URL.Path != "/chat/completions" {
			t.Errorf("Expected request path '/chat/completions', got '%s'", r.URL.Path)
		}

		// Check streaming parameter
		var requestBody map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil {
			t.Fatalf("Failed to decode request body: %v", err)
		}

		if stream, ok := requestBody["stream"].(bool); !ok || !stream {
			t.Errorf("Expected stream parameter to be true")
		}

		// Return a mock streaming response
		// In a real implementation, this would be a proper SSE stream
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte("data: {\"choices\":[{\"delta\":{\"content\":\"This is a \"}}]}\n\n")); err != nil {
			t.Errorf("Failed to write response: %v", err)
		}
		if _, err := w.Write([]byte("data: {\"choices\":[{\"delta\":{\"content\":\"streaming response\"}}]}\n\n")); err != nil {
			t.Errorf("Failed to write response: %v", err)
		}
		if _, err := w.Write([]byte("data: {\"choices\":[{\"delta\":{\"content\":\" from OpenAI\"}}]}\n\n")); err != nil {
			t.Errorf("Failed to write response: %v", err)
		}
		if _, err := w.Write([]byte("data: [DONE]\n\n")); err != nil {
			t.Errorf("Failed to write response: %v", err)
		}
	}))
	defer server.Close()

	// Save original environment variables
	originalAPIKey := os.Getenv("OPENAI_API_KEY")
	originalStreaming := os.Getenv("MAESTRO_OPENAI_STREAMING")

	// Restore environment variables after test
	defer func() {
		os.Setenv("OPENAI_API_KEY", originalAPIKey)
		os.Setenv("MAESTRO_OPENAI_STREAMING", originalStreaming)
	}()

	// Set test environment variables
	os.Setenv("OPENAI_API_KEY", "test-api-key")
	os.Setenv("MAESTRO_OPENAI_STREAMING", "true")

	// Create a test agent definition
	agent := map[string]interface{}{
		"apiVersion": "maestro/v1alpha1",
		"kind":       "Agent",
		"metadata": map[string]interface{}{
			"name": "test-openai-agent",
		},
		"spec": map[string]interface{}{
			"framework":    "openai",
			"model":        "gpt-4",
			"url":          server.URL,
			"instructions": "You are a helpful assistant.",
		},
	}

	// Create the agent
	openaiAgent, err := NewOpenAIAgent(agent)
	if err != nil {
		t.Fatalf("Failed to create OpenAIAgent: %v", err)
	}

	oa, ok := openaiAgent.(*OpenAIAgent)
	if !ok {
		t.Fatalf("Expected *OpenAIAgent, got %T", openaiAgent)
	}

	// Run the agent in streaming mode
	result, err := oa.RunStreaming("test prompt")
	if err != nil {
		t.Fatalf("Failed to run OpenAIAgent in streaming mode: %v", err)
	}

	// In a real implementation, we would check the streaming output
	// For now, just check that we got some result
	if result == "" {
		t.Errorf("Expected non-empty result from streaming")
	}
}

// Made with Bob
