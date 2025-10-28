// SPDX-License-Identifier: Apache-2.0
// Copyright © 2025 IBM

package agents

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestNewBeeAIAgent(t *testing.T) {
	// Create a test agent definition
	agent := map[string]interface{}{
		"apiVersion": "maestro/v1alpha1",
		"kind":       "Agent",
		"metadata": map[string]interface{}{
			"name": "test-beeai-agent",
		},
		"spec": map[string]interface{}{
			"framework":    "beeai",
			"model":        "llama3:8b",
			"url":          "http://localhost:8080",
			"instructions": "You are a helpful assistant.",
			"tools":        []interface{}{"weather", "search"},
			"output":       "Results: {{.result}}",
		},
	}

	// Create the agent
	beeaiAgent, err := NewBeeAIAgent(agent)
	if err != nil {
		t.Fatalf("Failed to create BeeAIAgent: %v", err)
	}

	// Check that the agent was created correctly
	ba, ok := beeaiAgent.(*BeeAIAgent)
	if !ok {
		t.Fatalf("Expected *BeeAIAgent, got %T", beeaiAgent)
	}

	// Check agent properties
	if ba.AgentName != "test-beeai-agent" {
		t.Errorf("Expected agent name 'test-beeai-agent', got '%s'", ba.AgentName)
	}

	if ba.AgentFramework != "beeai" {
		t.Errorf("Expected agent framework 'beeai', got '%s'", ba.AgentFramework)
	}

	if ba.AgentModel != "llama3:8b" {
		t.Errorf("Expected agent model 'llama3:8b', got '%s'", ba.AgentModel)
	}

	if ba.AgentURL != "http://localhost:8080" {
		t.Errorf("Expected agent URL 'http://localhost:8080', got '%s'", ba.AgentURL)
	}

	if ba.AgentInstr != "You are a helpful assistant." {
		t.Errorf("Expected agent instructions 'You are a helpful assistant.', got '%s'", ba.AgentInstr)
	}

	// Check tools
	if len(ba.AgentTools) != 2 {
		t.Errorf("Expected 2 tools, got %d", len(ba.AgentTools))
	}

	// Test output template
	var buf strings.Builder
	err = ba.OutputTemplate.Execute(&buf, map[string]interface{}{
		"result": "test result",
	})
	if err != nil {
		t.Fatalf("Failed to execute output template: %v", err)
	}

	if buf.String() != "Results: test result" {
		t.Errorf("Expected output template to render 'Results: test result', got '%s'", buf.String())
	}
}

func TestBeeAIAgentRun(t *testing.T) {
	// Create a mock BeeAI server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check request path
		if r.URL.Path != "/run" {
			t.Errorf("Expected request path '/run', got '%s'", r.URL.Path)
		}

		// Check request method
		if r.Method != "POST" {
			t.Errorf("Expected request method 'POST', got '%s'", r.Method)
		}

		// Check request body
		var requestBody map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil {
			t.Fatalf("Failed to decode request body: %v", err)
		}

		if requestBody["prompt"] != "test prompt" {
			t.Errorf("Expected prompt 'test prompt', got '%v'", requestBody["prompt"])
		}

		if requestBody["model"] != "llama3:8b" {
			t.Errorf("Expected model 'llama3:8b', got '%v'", requestBody["model"])
		}

		if requestBody["instructions"] != "You are a helpful assistant." {
			t.Errorf("Expected instructions 'You are a helpful assistant.', got '%v'", requestBody["instructions"])
		}

		// Return a mock response
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(map[string]interface{}{
			"result": "This is a test response from BeeAI",
		}); err != nil {
			t.Errorf("Failed to encode response: %v", err)
		}
	}))
	defer server.Close()

	// Create a test agent definition
	agent := map[string]interface{}{
		"apiVersion": "maestro/v1alpha1",
		"kind":       "Agent",
		"metadata": map[string]interface{}{
			"name": "test-beeai-agent",
		},
		"spec": map[string]interface{}{
			"framework":    "beeai",
			"model":        "llama3:8b",
			"url":          server.URL,
			"instructions": "You are a helpful assistant.",
			"tools":        []interface{}{"weather", "search"},
			"output":       "Results: {{.result}}",
		},
	}

	// Create the agent
	beeaiAgent, err := NewBeeAIAgent(agent)
	if err != nil {
		t.Fatalf("Failed to create BeeAIAgent: %v", err)
	}

	ba, ok := beeaiAgent.(*BeeAIAgent)
	if !ok {
		t.Fatalf("Expected *BeeAIAgent, got %T", beeaiAgent)
	}

	// Run the agent
	result, err := ba.Run("test prompt")
	if err != nil {
		t.Fatalf("Failed to run BeeAIAgent: %v", err)
	}

	// Check the result
	expectedResult := "Results: This is a test response from BeeAI"
	if result != expectedResult {
		t.Errorf("Expected result '%s', got '%v'", expectedResult, result)
	}
}

func TestBeeAIAgentRunWithContext(t *testing.T) {
	// Create a mock BeeAI server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check request body
		var requestBody map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil {
			t.Fatalf("Failed to decode request body: %v", err)
		}

		// Check context
		context, ok := requestBody["context"].(map[string]interface{})
		if !ok {
			t.Fatalf("Expected context in request body")
		}

		if context["previous_step"] != "step1" {
			t.Errorf("Expected previous_step 'step1', got '%v'", context["previous_step"])
		}

		// Return a mock response
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(map[string]interface{}{
			"result": "Response with context",
		}); err != nil {
			t.Errorf("Failed to encode response: %v", err)
		}
	}))
	defer server.Close()

	// Create a test agent definition
	agent := map[string]interface{}{
		"apiVersion": "maestro/v1alpha1",
		"kind":       "Agent",
		"metadata": map[string]interface{}{
			"name": "test-beeai-agent",
		},
		"spec": map[string]interface{}{
			"framework": "beeai",
			"model":     "llama3:8b",
			"url":       server.URL,
		},
	}

	// Create the agent
	beeaiAgent, err := NewBeeAIAgent(agent)
	if err != nil {
		t.Fatalf("Failed to create BeeAIAgent: %v", err)
	}

	ba, ok := beeaiAgent.(*BeeAIAgent)
	if !ok {
		t.Fatalf("Expected *BeeAIAgent, got %T", beeaiAgent)
	}

	// Create context
	context := map[string]interface{}{
		"previous_step": "step1",
	}

	// Run the agent with context
	result, err := ba.Run("test prompt", context)
	if err != nil {
		t.Fatalf("Failed to run BeeAIAgent: %v", err)
	}

	// Check the result
	if result != "Response with context" {
		t.Errorf("Expected result 'Response with context', got '%v'", result)
	}
}

// Made with Bob
