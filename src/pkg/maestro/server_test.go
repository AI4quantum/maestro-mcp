// SPDX-License-Identifier: Apache-2.0
// Copyright © 2025 IBM

package maestro

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestAgentServer(t *testing.T) {
	// Create temporary agent file
	tempDir, err := os.MkdirTemp("", "agent_server_test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create agent YAML file
	agentYAML := `
- apiVersion: maestro/v1alpha1
  kind: Agent
  metadata:
    name: test-agent
  spec:
    framework: beeai
    mode: local
    model: test-model
`
	agentFile := filepath.Join(tempDir, "agent.yaml")
	if err := os.WriteFile(agentFile, []byte(agentYAML), 0644); err != nil {
		t.Fatalf("Failed to write agent file: %v", err)
	}

	// Set up test server
	gin.SetMode(gin.TestMode)
	server, err := NewAgentServer(agentFile, "")
	if err != nil {
		t.Fatalf("Failed to create agent server: %v", err)
	}

	// Test health endpoint
	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()
	server.Router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, w.Code)
	}

	var healthResp HealthResponse
	if err := json.Unmarshal(w.Body.Bytes(), &healthResp); err != nil {
		t.Errorf("Failed to parse response: %v", err)
	}

	if healthResp.Status != "healthy" {
		t.Errorf("Expected status 'healthy', got '%s'", healthResp.Status)
	}

	if healthResp.AgentName != "test-agent" {
		t.Errorf("Expected agent name 'test-agent', got '%s'", healthResp.AgentName)
	}

	// Test agents endpoint
	req = httptest.NewRequest("GET", "/agents", nil)
	w = httptest.NewRecorder()
	server.Router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, w.Code)
	}

	var agentsResp AgentListResponse
	if err := json.Unmarshal(w.Body.Bytes(), &agentsResp); err != nil {
		t.Errorf("Failed to parse response: %v", err)
	}

	if len(agentsResp.Agents) != 1 || agentsResp.Agents[0] != "test-agent" {
		t.Errorf("Expected agents ['test-agent'], got %v", agentsResp.Agents)
	}

	// Test chat endpoint
	chatReq := ChatRequest{
		Prompt: "Hello, world!",
	}
	reqBody, _ := json.Marshal(chatReq)
	req = httptest.NewRequest("POST", "/chat", bytes.NewBuffer(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	server.Router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
	}

	var chatResp ChatResponse
	if err := json.Unmarshal(w.Body.Bytes(), &chatResp); err != nil {
		t.Errorf("Failed to parse response: %v", err)
	}

	if chatResp.AgentName != "DefaultAgent" {
		t.Errorf("Expected agent name 'DefaultAgent', got '%s'", chatResp.AgentName)
	}
}

func TestWorkflowServer(t *testing.T) {
	// Create temporary files
	tempDir, err := os.MkdirTemp("", "workflow_server_test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create agent YAML file
	agentYAML := `
- apiVersion: maestro/v1alpha1
  kind: Agent
  metadata:
    name: test-agent
  spec:
    framework: beeai
    mode: local
    model: test-model
`
	agentFile := filepath.Join(tempDir, "agent.yaml")
	if err := os.WriteFile(agentFile, []byte(agentYAML), 0644); err != nil {
		t.Fatalf("Failed to write agent file: %v", err)
	}

	// Create workflow YAML file
	workflowYAML := `
- apiVersion: maestro/v1
  kind: Workflow
  metadata:
    name: test-workflow
  spec:
    template:
      agents: [test-agent]
      prompt: "Test prompt"
      steps:
      - name: step1
        agent: test-agent
`
	workflowFile := filepath.Join(tempDir, "workflow.yaml")
	if err := os.WriteFile(workflowFile, []byte(workflowYAML), 0644); err != nil {
		t.Fatalf("Failed to write workflow file: %v", err)
	}

	// Set up test server
	gin.SetMode(gin.TestMode)
	server, err := NewWorkflowServer(agentFile, workflowFile)
	if err != nil {
		t.Fatalf("Failed to create workflow server: %v", err)
	}

	// Test health endpoint
	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()
	server.Router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, w.Code)
	}

	var healthResp WorkflowHealthResponse
	if err := json.Unmarshal(w.Body.Bytes(), &healthResp); err != nil {
		t.Errorf("Failed to parse response: %v", err)
	}

	if healthResp.Status != "healthy" {
		t.Errorf("Expected status 'healthy', got '%s'", healthResp.Status)
	}

	if healthResp.WorkflowName != "test-workflow" {
		t.Errorf("Expected workflow name 'test-workflow', got '%s'", healthResp.WorkflowName)
	}

	// Test diagram endpoint
	req = httptest.NewRequest("GET", "/diagram", nil)
	w = httptest.NewRecorder()
	server.Router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, w.Code)
	}

	var diagramResp DiagramResponse
	if err := json.Unmarshal(w.Body.Bytes(), &diagramResp); err != nil {
		t.Errorf("Failed to parse response: %v", err)
	}

	if diagramResp.WorkflowName != "test-workflow" {
		t.Errorf("Expected workflow name 'test-workflow', got '%s'", diagramResp.WorkflowName)
	}

	if diagramResp.Diagram == "" {
		t.Errorf("Expected non-empty diagram")
	}

	// Test chat endpoint
	chatReq := WorkflowChatRequest{
		Prompt: "Hello, workflow!",
	}
	reqBody, _ := json.Marshal(chatReq)
	req = httptest.NewRequest("POST", "/chat", bytes.NewBuffer(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	server.Router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
	}

	var chatResp WorkflowChatResponse
	if err := json.Unmarshal(w.Body.Bytes(), &chatResp); err != nil {
		t.Errorf("Failed to parse response: %v", err)
	}

	if chatResp.WorkflowName != "test-workflow" {
		t.Errorf("Expected workflow name 'test-workflow', got '%s'", chatResp.WorkflowName)
	}
}

// Made with Bob
