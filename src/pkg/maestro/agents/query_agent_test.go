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

func TestNewQueryAgent(t *testing.T) {
	// Create a test agent definition
	agent := map[string]interface{}{
		"apiVersion": "maestro/v1alpha1",
		"kind":       "Agent",
		"metadata": map[string]interface{}{
			"name": "test-query-agent",
			"query_input": map[string]interface{}{
				"db_name":         "test-db",
				"collection_name": "test-collection",
				"limit":           5.0,
			},
			"labels": map[string]interface{}{
				"custom_agent": "query_agent",
			},
		},
		"spec": map[string]interface{}{
			"framework": "query",
			"output":    "Results: {{.result}}",
		},
	}

	// Create the agent
	queryAgent, err := NewQueryAgent(agent)
	if err != nil {
		t.Fatalf("Failed to create QueryAgent: %v", err)
	}

	// Check that the agent was created correctly
	qa, ok := queryAgent.(*QueryAgent)
	if !ok {
		t.Fatalf("Expected *QueryAgent, got %T", queryAgent)
	}

	// Check agent properties
	if qa.AgentName != "test-query-agent" {
		t.Errorf("Expected agent name 'test-query-agent', got '%s'", qa.AgentName)
	}

	if qa.AgentFramework != "query" {
		t.Errorf("Expected agent framework 'query', got '%s'", qa.AgentFramework)
	}

	if qa.DBName != "test-db" {
		t.Errorf("Expected DB name 'test-db', got '%s'", qa.DBName)
	}

	if qa.CollectionName != "test-collection" {
		t.Errorf("Expected collection name 'test-collection', got '%s'", qa.CollectionName)
	}

	if qa.Limit != 5 {
		t.Errorf("Expected limit 5, got %d", qa.Limit)
	}

	// Test output template
	var buf strings.Builder
	err = qa.OutputTemplate.Execute(&buf, map[string]interface{}{
		"result": "test result",
	})
	if err != nil {
		t.Fatalf("Failed to execute output template: %v", err)
	}

	if buf.String() != "Results: test result" {
		t.Errorf("Expected output template to render 'Results: test result', got '%s'", buf.String())
	}
}

func TestQueryAgentRun(t *testing.T) {
	// Create a mock MCP server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check request path
		if r.URL.Path != "/mcp/tool/search" {
			t.Errorf("Expected request path '/mcp/tool/search', got '%s'", r.URL.Path)
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

		input, ok := requestBody["input"].(map[string]interface{})
		if !ok {
			t.Fatalf("Expected input field in request body")
		}

		if input["db_name"] != "test-db" {
			t.Errorf("Expected db_name 'test-db', got '%v'", input["db_name"])
		}

		if input["query"] != "test query" {
			t.Errorf("Expected query 'test query', got '%v'", input["query"])
		}

		if input["collection_name"] != "test-collection" {
			t.Errorf("Expected collection_name 'test-collection', got '%v'", input["collection_name"])
		}

		if input["limit"] != float64(5) {
			t.Errorf("Expected limit 5, got %v", input["limit"])
		}

		// Return a mock response
		mockDocs := []map[string]interface{}{
			{
				"text": "Document 1 content",
				"id":   "doc1",
			},
			{
				"text": "Document 2 content",
				"id":   "doc2",
			},
		}

		mockDocsJSON, _ := json.Marshal(mockDocs)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(map[string]interface{}{
			"data": string(mockDocsJSON),
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
			"name": "test-query-agent",
			"query_input": map[string]interface{}{
				"db_name":         "test-db",
				"collection_name": "test-collection",
				"limit":           5.0,
			},
			"labels": map[string]interface{}{
				"custom_agent": "query_agent",
			},
		},
		"spec": map[string]interface{}{
			"framework": "query",
			"url":       server.URL + "/mcp/",
			"output":    "Results: {{.result}}",
		},
	}

	// Create the agent
	queryAgent, err := NewQueryAgent(agent)
	if err != nil {
		t.Fatalf("Failed to create QueryAgent: %v", err)
	}

	qa, ok := queryAgent.(*QueryAgent)
	if !ok {
		t.Fatalf("Expected *QueryAgent, got %T", queryAgent)
	}

	// Run the agent
	result, err := qa.Run("test query")
	if err != nil {
		t.Fatalf("Failed to run QueryAgent: %v", err)
	}

	// Check the result
	expectedResult := "Results: Document 1 content\n\nDocument 2 content"
	if result != expectedResult {
		t.Errorf("Expected result '%s', got '%v'", expectedResult, result)
	}
}

// Made with Bob
