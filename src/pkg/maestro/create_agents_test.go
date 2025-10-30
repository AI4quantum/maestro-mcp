// SPDX-License-Identifier: Apache-2.0
// Copyright © 2025 IBM

package maestro

import (
	"os"
	"testing"
)

func TestCreateAgents(t *testing.T) {
	// Save the original agents.db file if it exists
	originalDB, err := os.ReadFile("agents.db")
	hasOriginalDB := err == nil

	// Clean up after the test
	defer func() {
		// Remove the test agents.db file
		os.Remove("agents.db")

		// Restore the original agents.db file if it existed
		if hasOriginalDB {
			os.WriteFile("agents.db", originalDB, 0644)
		}
	}()

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
	err = CreateAgents(agentDefs)
	if err != nil {
		t.Fatalf("CreateAgents failed: %v", err)
	}

	// Verify agents.db file was created
	if _, err := os.Stat("agents.db"); os.IsNotExist(err) {
		t.Errorf("agents.db file not created")
		return
	}

	// Load the agent database to verify agents were saved
	db, err := LoadAgentDB()
	if err != nil {
		t.Fatalf("Failed to load agent database: %v", err)
	}

	// Check if agents exist in the database
	if _, ok := db.Agents["test-agent-1"]; !ok {
		t.Errorf("Agent 'test-agent-1' not found in database")
	}

	if _, ok := db.Agents["test-agent-2"]; !ok {
		t.Errorf("Agent 'test-agent-2' not found in database")
	}
}

// Made with Bob
