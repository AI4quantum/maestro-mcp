// SPDX-License-Identifier: Apache-2.0
// Copyright © 2025 IBM

package agents

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewRemoteAgent(t *testing.T) {
	// Create a test agent definition
	agentDef := map[string]interface{}{
		"metadata": map[string]interface{}{
			"name": "test-remote-agent",
		},
		"spec": map[string]interface{}{
			"framework":         "remote",
			"description":       "Test remote agent",
			"instructions":      "This is a test remote agent",
			"url":               "https://example.com/api",
			"request_template":  `{"message": "${prompt}"}`,
			"response_template": `${response.answer}`,
		},
	}

	// Create a new remote agent
	agent, err := NewRemoteAgent(agentDef)
	if err != nil {
		t.Fatalf("Failed to create remote agent: %v", err)
	}

	// Check that the agent is a RemoteAgent
	remoteAgent, ok := agent.(*RemoteAgent)
	if !ok {
		t.Fatalf("Expected agent to be a RemoteAgent, got %T", agent)
	}

	// Check agent properties
	if remoteAgent.AgentName != "test-remote-agent" {
		t.Errorf("Expected agent name to be 'test-remote-agent', got '%s'", remoteAgent.AgentName)
	}
	if remoteAgent.AgentFramework != "remote" {
		t.Errorf("Expected agent framework to be 'remote', got '%s'", remoteAgent.AgentFramework)
	}
	if remoteAgent.URL != "https://example.com/api" {
		t.Errorf("Expected URL to be 'https://example.com/api', got '%s'", remoteAgent.URL)
	}
	if remoteAgent.RequestTemplate != `{"message": "${prompt}"}` {
		t.Errorf("Expected request template to be '{\"message\": \"${prompt}\"}', got '%s'", remoteAgent.RequestTemplate)
	}
	if remoteAgent.ResponseTemplate != `${response.answer}` {
		t.Errorf("Expected response template to be '${response.answer}', got '%s'", remoteAgent.ResponseTemplate)
	}
}

func TestRemoteAgentRun(t *testing.T) {
	// Create a test HTTP server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check request method
		if r.Method != http.MethodPost {
			t.Errorf("Expected POST request, got %s", r.Method)
		}

		// Check content type
		contentType := r.Header.Get("Content-Type")
		if contentType != "application/json" {
			t.Errorf("Expected Content-Type: application/json, got %s", contentType)
		}

		// Parse request body
		var requestData map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&requestData); err != nil {
			t.Errorf("Failed to parse request body: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		// Check that the prompt is included
		prompt, ok := requestData["prompt"].(string)
		if !ok {
			t.Errorf("Expected prompt in request data, got %v", requestData)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		// Send response
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		response := map[string]interface{}{
			"answer": "Response to: " + prompt,
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	// Create a test agent definition
	agentDef := map[string]interface{}{
		"metadata": map[string]interface{}{
			"name": "test-remote-agent",
		},
		"spec": map[string]interface{}{
			"framework": "remote",
			"url":       server.URL,
		},
	}

	// Create a new remote agent
	agent, err := NewRemoteAgent(agentDef)
	if err != nil {
		t.Fatalf("Failed to create remote agent: %v", err)
	}

	remoteAgent := agent.(*RemoteAgent)

	// Run the agent
	result, err := remoteAgent.Run("Test prompt")
	if err != nil {
		t.Fatalf("Failed to run remote agent: %v", err)
	}

	// Check the result
	resultMap, ok := result.(map[string]interface{})
	if !ok {
		t.Fatalf("Expected result to be a map, got %T", result)
	}

	answer, ok := resultMap["answer"].(string)
	if !ok {
		t.Fatalf("Expected result to contain 'answer' key with string value, got %v", resultMap)
	}

	if answer != "Response to: Test prompt" {
		t.Errorf("Expected answer to be 'Response to: Test prompt', got '%s'", answer)
	}
}

func TestRemoteAgentWithTemplates(t *testing.T) {
	// Create a test HTTP server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Parse request body
		var requestData map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&requestData); err != nil {
			t.Errorf("Failed to parse request body: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		// Check that the message is included
		message, ok := requestData["message"].(string)
		if !ok {
			t.Errorf("Expected message in request data, got %v", requestData)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		// Send response
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		response := map[string]interface{}{
			"data": map[string]interface{}{
				"text": "Response to: " + message,
			},
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	// Create a test agent definition with templates
	agentDef := map[string]interface{}{
		"metadata": map[string]interface{}{
			"name": "test-remote-agent-templates",
		},
		"spec": map[string]interface{}{
			"framework":         "remote",
			"url":               server.URL,
			"request_template":  `{"message": "{{.prompt}}"}`,
			"response_template": `{{.response.data.text}}`,
		},
	}

	// Create a new remote agent
	agent, err := NewRemoteAgent(agentDef)
	if err != nil {
		t.Fatalf("Failed to create remote agent: %v", err)
	}

	remoteAgent := agent.(*RemoteAgent)

	// Run the agent
	result, err := remoteAgent.Run("Test prompt")
	if err != nil {
		t.Fatalf("Failed to run remote agent: %v", err)
	}

	// Check the result
	resultStr, ok := result.(string)
	if !ok {
		t.Fatalf("Expected result to be a string, got %T", result)
	}

	if resultStr != "Response to: Test prompt" {
		t.Errorf("Expected result to be 'Response to: Test prompt', got '%s'", resultStr)
	}
}

func TestRemoteAgentWithInvalidDefinition(t *testing.T) {
	testCases := []struct {
		name     string
		agentDef map[string]interface{}
	}{
		{
			name: "Missing metadata",
			agentDef: map[string]interface{}{
				"spec": map[string]interface{}{
					"framework": "remote",
					"url":       "https://example.com/api",
				},
			},
		},
		{
			name: "Missing spec",
			agentDef: map[string]interface{}{
				"metadata": map[string]interface{}{
					"name": "test-remote-agent",
				},
			},
		},
		{
			name: "Missing URL",
			agentDef: map[string]interface{}{
				"metadata": map[string]interface{}{
					"name": "test-remote-agent",
				},
				"spec": map[string]interface{}{
					"framework": "remote",
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := NewRemoteAgent(tc.agentDef)
			if err == nil {
				t.Errorf("Expected error for invalid agent definition, got nil")
			}
		})
	}
}

// Made with Bob
