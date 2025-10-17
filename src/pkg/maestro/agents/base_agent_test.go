// SPDX-License-Identifier: Apache-2.0
// Copyright © 2025 IBM

package agents

import (
	"testing"
)

func TestBaseAgent(t *testing.T) {
	// Create a test agent definition
	agentDef := map[string]interface{}{
		"metadata": map[string]interface{}{
			"name": "test-base-agent",
		},
		"spec": map[string]interface{}{
			"framework":    "openai",
			"model":        "gpt-4",
			"description":  "Test base agent",
			"instructions": "This is a test base agent",
		},
	}

	// Create a new base agent
	baseAgent, err := NewBaseAgent(agentDef)
	if err != nil {
		t.Fatalf("Failed to create base agent: %v", err)
	}

	// Check agent properties
	if baseAgent.GetName() != "test-base-agent" {
		t.Errorf("Expected agent name to be 'test-base-agent', got '%s'", baseAgent.GetName())
	}
	if baseAgent.GetModel() != "gpt-4" {
		t.Errorf("Expected agent model to be 'gpt-4', got '%s'", baseAgent.GetModel())
	}

	// Test Run method
	response, err := baseAgent.Run("Test prompt")
	if err != nil {
		t.Fatalf("Failed to run agent: %v", err)
	}

	// Check that response is a string
	_, ok := response.(string)
	if !ok {
		t.Fatalf("Expected response to be a string, got %T", response)
	}

	// Check that token usage was tracked
	if baseAgent.PromptTokens == 0 {
		t.Error("Expected prompt tokens to be non-zero")
	}
	if baseAgent.ResponseTokens == 0 {
		t.Error("Expected response tokens to be non-zero")
	}
	if baseAgent.TotalTokens == 0 {
		t.Error("Expected total tokens to be non-zero")
	}

	// Test Run method with invalid arguments
	_, err = baseAgent.Run()
	if err == nil {
		t.Error("Expected error when running with no arguments")
	}

	_, err = baseAgent.Run(123)
	if err == nil {
		t.Error("Expected error when running with non-string argument")
	}
}

// Made with Bob
