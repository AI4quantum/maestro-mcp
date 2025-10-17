// SPDX-License-Identifier: Apache-2.0
// Copyright © 2025 IBM

package maestro

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestCreateMCPTools(t *testing.T) {
	// Create a temporary directory for the test
	tempDir, err := os.MkdirTemp("", "mcptools_test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Set environment variable for MCP server list
	tempFile := filepath.Join(tempDir, "mcp_servers.json")
	os.Setenv("MCP_SERVER_LIST", tempFile)
	defer os.Unsetenv("MCP_SERVER_LIST")

	// Create test tool definitions
	toolDefs := []map[string]interface{}{
		{
			"metadata": map[string]interface{}{
				"name":  "test-tool-1",
				"token": "test-token-1",
			},
			"spec": map[string]interface{}{
				"url":       "https://example.com/mcp",
				"transport": "http",
			},
		},
		{
			"metadata": map[string]interface{}{
				"name": "test-tool-2",
			},
			"spec": map[string]interface{}{
				"url": "https://example.org/mcp",
				// No transport specified, should default to "http"
			},
		},
		{
			"metadata": map[string]interface{}{
				"name": "test-tool-3",
			},
			"spec": map[string]interface{}{
				// No URL specified, should be skipped
				"transport": "http",
			},
		},
	}

	// Call CreateMCPTools
	err = CreateMCPTools(toolDefs)
	if err != nil {
		t.Fatalf("CreateMCPTools failed: %v", err)
	}

	// Verify JSON file was created
	if _, err := os.Stat(tempFile); os.IsNotExist(err) {
		t.Errorf("JSON file not created: %s", tempFile)
	}

	// Read JSON file
	data, err := os.ReadFile(tempFile)
	if err != nil {
		t.Fatalf("Failed to read JSON file: %v", err)
	}

	// Parse JSON data
	var jsonData []MCPServerJSON
	if err := json.Unmarshal(data, &jsonData); err != nil {
		t.Fatalf("Failed to parse JSON data: %v", err)
	}

	// Verify JSON data
	if len(jsonData) != 2 {
		t.Errorf("Expected 2 JSON entries, got %d", len(jsonData))
	}

	// Verify first entry
	if len(jsonData) > 0 {
		entry := jsonData[0]
		if entry.Name != "test-tool-1" {
			t.Errorf("Expected name 'test-tool-1', got '%s'", entry.Name)
		}
		if entry.URL != "https://example.com" {
			t.Errorf("Expected URL 'https://example.com', got '%s'", entry.URL)
		}
		if entry.Transport != "http" {
			t.Errorf("Expected transport 'http', got '%s'", entry.Transport)
		}
		if entry.AccessToken != "test-token-1" {
			t.Errorf("Expected access token 'test-token-1', got '%s'", entry.AccessToken)
		}
	}

	// Verify second entry
	if len(jsonData) > 1 {
		entry := jsonData[1]
		if entry.Name != "test-tool-2" {
			t.Errorf("Expected name 'test-tool-2', got '%s'", entry.Name)
		}
		if entry.URL != "https://example.org" {
			t.Errorf("Expected URL 'https://example.org', got '%s'", entry.URL)
		}
		if entry.Transport != "http" {
			t.Errorf("Expected transport 'http', got '%s'", entry.Transport)
		}
		if entry.AccessToken != "" {
			t.Errorf("Expected empty access token, got '%s'", entry.AccessToken)
		}
	}
}

// Made with Bob
