// SPDX-License-Identifier: Apache-2.0
// Copyright © 2025 IBM

package agents

import (
	"os"
	"testing"
)

func TestNewAgent(t *testing.T) {
	// Create a test agent definition
	agentDef := map[string]interface{}{
		"metadata": map[string]interface{}{
			"name": "test-agent",
		},
		"spec": map[string]interface{}{
			"framework":    "openai",
			"model":        "gpt-4",
			"description":  "Test agent",
			"instructions": "This is a test agent",
			"input":        "JSON",
			"output":       "Markdown",
		},
	}

	// Create a new agent
	agent, err := NewAgent(agentDef)
	if err != nil {
		t.Fatalf("Failed to create agent: %v", err)
	}

	// Check agent properties
	if agent.AgentName != "test-agent" {
		t.Errorf("Expected agent name to be 'test-agent', got '%s'", agent.AgentName)
	}
	if agent.AgentFramework != "openai" {
		t.Errorf("Expected agent framework to be 'openai', got '%s'", agent.AgentFramework)
	}
	if agent.AgentModel != "gpt-4" {
		t.Errorf("Expected agent model to be 'gpt-4', got '%s'", agent.AgentModel)
	}
	if agent.AgentDesc != "Test agent" {
		t.Errorf("Expected agent description to be 'Test agent', got '%s'", agent.AgentDesc)
	}
	if agent.AgentInstr != "This is a test agent" {
		t.Errorf("Expected agent instructions to be 'This is a test agent', got '%s'", agent.AgentInstr)
	}
	if agent.AgentInput != "JSON" {
		t.Errorf("Expected agent input to be 'JSON', got '%s'", agent.AgentInput)
	}
	if agent.AgentOutput != "Markdown" {
		t.Errorf("Expected agent output to be 'Markdown', got '%s'", agent.AgentOutput)
	}

	// Check that instructions were combined correctly
	expectedInstructions := "This is a test agent Input is expected in format: JSON Output must be in format: Markdown"
	if agent.Instructions != expectedInstructions {
		t.Errorf("Expected instructions to be '%s', got '%s'", expectedInstructions, agent.Instructions)
	}
}

func TestEmoji(t *testing.T) {
	testCases := []struct {
		framework string
		expected  string
	}{
		{"openai", "🔓"},
		{"beeai", "🐝"},
		{"crewai", "👥"},
		{"dspy", "💭"},
		{"mock", "🤖"},
		{"remote", "💸"},
		{"unknown", "⚙️"},
	}

	for _, tc := range testCases {
		agent := &Agent{AgentFramework: tc.framework}
		emoji := agent.Emoji()
		if emoji != tc.expected {
			t.Errorf("Expected emoji for '%s' to be '%s', got '%s'", tc.framework, tc.expected, emoji)
		}
	}
}

func TestGetTokenUsage(t *testing.T) {
	// Test regular agent
	agent := &Agent{
		AgentName:      "test-agent",
		AgentFramework: "openai",
		PromptTokens:   100,
		ResponseTokens: 50,
		TotalTokens:    150,
	}

	usage := agent.GetTokenUsage()
	if usage["prompt_tokens"] != 100 {
		t.Errorf("Expected prompt_tokens to be 100, got %v", usage["prompt_tokens"])
	}
	if usage["response_tokens"] != 50 {
		t.Errorf("Expected response_tokens to be 50, got %v", usage["response_tokens"])
	}
	if usage["total_tokens"] != 150 {
		t.Errorf("Expected total_tokens to be 150, got %v", usage["total_tokens"])
	}

	// Test custom agent
	customAgent := &Agent{
		AgentName:      "custom-agent",
		AgentFramework: "custom",
	}

	customUsage := customAgent.GetTokenUsage()
	if customUsage["agent_type"] != "custom_agent" {
		t.Errorf("Expected agent_type to be 'custom_agent', got %v", customUsage["agent_type"])
	}

	// Test scoring agent
	scoringAgent := &Agent{
		AgentName:      "scoring-agent",
		AgentFramework: "custom",
	}

	scoringUsage := scoringAgent.GetTokenUsage()
	if scoringUsage["agent_type"] != "scoring_agent" {
		t.Errorf("Expected agent_type to be 'scoring_agent', got %v", scoringUsage["agent_type"])
	}
}

func TestTrackTokens(t *testing.T) {
	agent := &Agent{
		AgentName:      "test-agent",
		AgentFramework: "openai",
	}

	// Track tokens for a prompt and response
	usage := agent.TrackTokens("This is a test prompt", "This is a test response")

	// Check that token counts were updated
	if agent.PromptTokens == 0 {
		t.Error("Expected prompt tokens to be non-zero")
	}
	if agent.ResponseTokens == 0 {
		t.Error("Expected response tokens to be non-zero")
	}
	if agent.TotalTokens == 0 {
		t.Error("Expected total tokens to be non-zero")
	}

	// Check that usage map was returned correctly
	if usage["prompt_tokens"] != agent.PromptTokens {
		t.Errorf("Expected usage prompt_tokens to be %d, got %d", agent.PromptTokens, usage["prompt_tokens"])
	}
	if usage["response_tokens"] != agent.ResponseTokens {
		t.Errorf("Expected usage response_tokens to be %d, got %d", agent.ResponseTokens, usage["response_tokens"])
	}
	if usage["total_tokens"] != agent.TotalTokens {
		t.Errorf("Expected usage total_tokens to be %d, got %d", agent.TotalTokens, usage["total_tokens"])
	}
}

func TestResetTokenUsage(t *testing.T) {
	agent := &Agent{
		AgentName:      "test-agent",
		AgentFramework: "openai",
		PromptTokens:   100,
		ResponseTokens: 50,
		TotalTokens:    150,
	}

	agent.ResetTokenUsage()

	if agent.PromptTokens != 0 {
		t.Errorf("Expected prompt tokens to be reset to 0, got %d", agent.PromptTokens)
	}
	if agent.ResponseTokens != 0 {
		t.Errorf("Expected response tokens to be reset to 0, got %d", agent.ResponseTokens)
	}
	if agent.TotalTokens != 0 {
		t.Errorf("Expected total tokens to be reset to 0, got %d", agent.TotalTokens)
	}
}

func TestAgentPersistence(t *testing.T) {
	// Create a temporary directory for the test
	tempDir, err := os.MkdirTemp("", "agent-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Change to the temp directory for the test
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}
	defer func() {
		if err := os.Chdir(originalDir); err != nil {
			t.Fatalf("Failed to change back to original directory: %v", err)
		}
	}()

	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("Failed to change to temp directory: %v", err)
	}

	// Create a test agent
	agent := &Agent{
		AgentName:      "test-agent",
		AgentFramework: "openai",
		AgentModel:     "gpt-4",
		AgentDesc:      "Test agent",
		AgentInstr:     "This is a test agent",
	}

	// Create agent definition
	agentDef := map[string]interface{}{
		"metadata": map[string]interface{}{
			"name": "test-agent",
		},
		"spec": map[string]interface{}{
			"framework":    "openai",
			"model":        "gpt-4",
			"description":  "Test agent",
			"instructions": "This is a test agent",
		},
	}

	// Save the agent
	err = SaveAgent(agent, agentDef)
	if err != nil {
		t.Fatalf("Failed to save agent: %v", err)
	}

	// Check that agents.db was created
	if _, err := os.Stat("agents.db"); os.IsNotExist(err) {
		t.Error("agents.db was not created")
	}

	// Restore the agent
	restored, isAgent, err := RestoreAgent("test-agent")
	if err != nil {
		t.Fatalf("Failed to restore agent: %v", err)
	}

	// Check that the agent was restored correctly
	if !isAgent {
		t.Error("Expected restored object to be an agent")
	}

	restoredAgent, ok := restored.(*Agent)
	if !ok {
		t.Fatalf("Restored object is not an Agent")
	}

	if restoredAgent.AgentName != "test-agent" {
		t.Errorf("Expected restored agent name to be 'test-agent', got '%s'", restoredAgent.AgentName)
	}

	// Remove the agent
	err = RemoveAgent("test-agent")
	if err != nil {
		t.Fatalf("Failed to remove agent: %v", err)
	}

	// Try to restore the agent again
	_, isAgent, restoreErr := RestoreAgent("test-agent")
	if isAgent {
		t.Error("Expected agent to be removed")
	}
	if restoreErr != nil {
		t.Logf("Restore error after removal: %v", restoreErr)
	}
}

// Made with Bob
