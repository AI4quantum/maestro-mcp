// SPDX-License-Identifier: Apache-2.0
// Copyright © 2025 IBM

package maestro

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCreateAgents(t *testing.T) {
	// Create a temporary directory for agent files
	tempDir := filepath.Join(os.TempDir(), "maestro_test")
	defer os.RemoveAll(tempDir)

	// Create test agent definitions
	agentDefs := []map[string]interface{}{
		{
			"metadata": map[string]interface{}{
				"name": "test-agent-1",
			},
			"spec": map[string]interface{}{
				"framework": "openai",
				"mode":      "local",
			},
		},
		{
			"metadata": map[string]interface{}{
				"name": "test-agent-2",
			},
			"spec": map[string]interface{}{
				// No framework specified, should default to "beeai"
			},
		},
	}

	// Call CreateAgents
	err := CreateAgents(agentDefs)
	if err != nil {
		t.Fatalf("createAgents failed: %v", err)
	}

	// Verify agent files were created
	agentsDir := filepath.Join(os.TempDir(), "maestro", "agents")

	// Check if agent files exist
	agent1Path := filepath.Join(agentsDir, "test-agent-1.json")
	if _, err := os.Stat(agent1Path); os.IsNotExist(err) {
		t.Errorf("Agent file not created: %s", agent1Path)
	}

	agent2Path := filepath.Join(agentsDir, "test-agent-2.json")
	if _, err := os.Stat(agent2Path); os.IsNotExist(err) {
		t.Errorf("Agent file not created: %s", agent2Path)
	}
}

// Made with Bob
