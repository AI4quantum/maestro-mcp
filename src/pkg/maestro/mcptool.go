// SPDX-License-Identifier: Apache-2.0
// Copyright © 2025 IBM

package maestro

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Constants for Kubernetes custom resources
const (
	// ToolHive CRD constants
	ToolHivePlural   = "mcpservers"
	ToolHiveSingular = "mcpserver"
	ToolHiveGroup    = "toolhive.stacklok.dev"
	ToolHiveVersion  = "v1alpha1"
	ToolHiveKind     = "MCPServer"

	// Remote MCP Server CRD constants
	RemotePlural   = "remotemcpservers"
	RemoteSingular = "remotemcpserver"
	RemoteGroup    = "maestro.ai4quantum.com"
	RemoteVersion  = "v1alpha1"
	RemoteKind     = "RemoteMCPServer"
)

// MCPServerJSON represents the JSON structure for MCP server configuration
type MCPServerJSON struct {
	Name        string `json:"name"`
	URL         string `json:"url"`
	Transport   string `json:"transport"`
	AccessToken string `json:"access_token,omitempty"`
}

// CreateMCPTools creates MCP tools from tool definitions
func CreateMCPTools(toolDefs []map[string]interface{}) error {
	// Check if Kubernetes is available
	kubeAvailable := checkKubernetesAvailable()

	// Store JSON data for tools
	jsonData := []MCPServerJSON{}

	// Process each tool definition
	for _, toolDef := range toolDefs {
		if kubeAvailable {
			// Try to create the tool in Kubernetes
			if err := createMCPTool(toolDef); err != nil {
				// If creation fails, disable Kubernetes for subsequent tools
				kubeAvailable = false
				fmt.Printf("Failed to create tool in Kubernetes: %v\n", err)
			}
		}

		// Create JSON entry regardless of Kubernetes availability
		if err := createJSON(toolDef, &jsonData); err != nil {
			fmt.Printf("Warning: Failed to create JSON for tool: %v\n", err)
		}
	}

	// If we have JSON data, save it to the configured file
	if len(jsonData) > 0 {
		if err := saveJSONData(jsonData); err != nil {
			return fmt.Errorf("failed to save JSON data: %w", err)
		}
	}

	return nil
}

// checkKubernetesAvailable checks if Kubernetes is available
func checkKubernetesAvailable() bool {
	// In a real implementation, this would check Kubernetes connectivity
	// For now, just return false as we don't have Kubernetes integration yet
	return false
}

// createMCPTool creates an MCP tool in Kubernetes
func createMCPTool(toolDef map[string]interface{}) error {
	// In a real implementation, this would use the Kubernetes client-go library
	// to create the custom resource
	// For now, just return nil as we don't have Kubernetes integration yet
	return nil
}

// createJSON creates a JSON entry for an MCP tool
func createJSON(toolDef map[string]interface{}, jsonData *[]MCPServerJSON) error {
	// Extract spec from tool definition
	spec, ok := toolDef["spec"].(map[string]interface{})
	if !ok {
		return fmt.Errorf("invalid tool definition: missing spec")
	}

	// Check if URL is present
	url, ok := spec["url"].(string)
	if !ok {
		// Skip tools without URL
		return nil
	}

	// Extract metadata from tool definition
	metadata, ok := toolDef["metadata"].(map[string]interface{})
	if !ok {
		return fmt.Errorf("invalid tool definition: missing metadata")
	}

	// Extract name from metadata
	name, ok := metadata["name"].(string)
	if !ok {
		return fmt.Errorf("invalid tool definition: missing name")
	}

	// Extract transport from spec
	transport, ok := spec["transport"].(string)
	if !ok {
		transport = "http" // Default transport
	}

	// Extract access token from metadata
	var accessToken string
	if token, ok := metadata["token"].(string); ok {
		accessToken = token
	}

	// Create JSON entry
	entry := MCPServerJSON{
		Name:        name,
		URL:         url,
		Transport:   transport,
		AccessToken: accessToken,
	}

	// Replace "/mcp" in URL if present
	entry.URL = replaceURLPath(entry.URL)

	// Add entry to JSON data
	*jsonData = append(*jsonData, entry)

	return nil
}

// replaceURLPath replaces "/mcp" in URL with empty string
func replaceURLPath(url string) string {
	// Simple string replacement for "/mcp"
	// In a real implementation, this would use proper URL parsing
	if len(url) >= 4 && url[len(url)-4:] == "/mcp" {
		return url[:len(url)-4]
	}
	return url
}

// saveJSONData saves JSON data to the configured file
func saveJSONData(jsonData []MCPServerJSON) error {
	// Get file path from environment variable
	filePath := os.Getenv("MCP_SERVER_LIST")
	if filePath == "" {
		// If environment variable is not set, use default path
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get user home directory: %w", err)
		}
		filePath = filepath.Join(homeDir, ".maestro", "mcp_servers.json")
	}

	// Create directory if it doesn't exist
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Check if file exists
	var existingData []MCPServerJSON
	if _, err := os.Stat(filePath); err == nil {
		// File exists, read existing data
		fileData, err := os.ReadFile(filePath)
		if err != nil {
			return fmt.Errorf("failed to read existing file: %w", err)
		}

		// Parse existing data
		if err := json.Unmarshal(fileData, &existingData); err != nil {
			return fmt.Errorf("failed to parse existing data: %w", err)
		}

		// Append new data to existing data
		jsonData = append(existingData, jsonData...)
	}

	// Marshal JSON data
	fileData, err := json.Marshal(jsonData)
	if err != nil {
		return fmt.Errorf("failed to marshal JSON data: %w", err)
	}

	// Write JSON data to file
	if err := os.WriteFile(filePath, fileData, 0644); err != nil {
		return fmt.Errorf("failed to write JSON data: %w", err)
	}

	return nil
}

// Made with Bob
