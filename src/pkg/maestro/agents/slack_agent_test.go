// SPDX-License-Identifier: Apache-2.0
// Copyright © 2025 IBM

package agents

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestNewSlackAgent(t *testing.T) {
	// Create a test agent definition
	agent := map[string]interface{}{
		"apiVersion": "maestro/v1alpha1",
		"kind":       "Agent",
		"metadata": map[string]interface{}{
			"name": "test-slack-agent",
			"labels": map[string]interface{}{
				"custom_agent": "slack_agent",
			},
		},
		"spec": map[string]interface{}{
			"framework": "slack",
		},
	}

	// Create the agent
	slackAgent, err := NewSlackAgent(agent)
	if err != nil {
		t.Fatalf("Failed to create SlackAgent: %v", err)
	}

	// Check that the agent was created correctly
	sa, ok := slackAgent.(*SlackAgent)
	if !ok {
		t.Fatalf("Expected *SlackAgent, got %T", slackAgent)
	}

	// Check agent properties
	if sa.AgentName != "test-slack-agent" {
		t.Errorf("Expected agent name 'test-slack-agent', got '%s'", sa.AgentName)
	}

	if sa.AgentFramework != "slack" {
		t.Errorf("Expected agent framework 'slack', got '%s'", sa.AgentFramework)
	}
}

func TestSlackAgentRun(t *testing.T) {
	// Save original environment variables
	originalToken := os.Getenv("SLACK_BOT_TOKEN")
	originalChannel := os.Getenv("SLACK_TEAM_ID")

	// Set test environment variables
	os.Setenv("SLACK_BOT_TOKEN", "test-token")
	os.Setenv("SLACK_TEAM_ID", "test-channel")

	// Restore environment variables after test
	defer func() {
		os.Setenv("SLACK_BOT_TOKEN", originalToken)
		os.Setenv("SLACK_TEAM_ID", originalChannel)
	}()

	// Create a mock Slack API server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check request headers
		if r.Header.Get("Authorization") != "Bearer test-token" {
			t.Errorf("Expected Authorization header 'Bearer test-token', got '%s'", r.Header.Get("Authorization"))
		}

		// Return a successful response
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"ok": true, "ts": "1234567890.123456"}`))
	}))
	defer server.Close()

	// Create a test agent definition
	agent := map[string]interface{}{
		"apiVersion": "maestro/v1alpha1",
		"kind":       "Agent",
		"metadata": map[string]interface{}{
			"name": "test-slack-agent",
			"labels": map[string]interface{}{
				"custom_agent": "slack_agent",
			},
		},
		"spec": map[string]interface{}{
			"framework": "slack",
		},
	}

	// Create the agent
	slackAgent, err := NewSlackAgent(agent)
	if err != nil {
		t.Fatalf("Failed to create SlackAgent: %v", err)
	}

	sa, ok := slackAgent.(*SlackAgent)
	if !ok {
		t.Fatalf("Expected *SlackAgent, got %T", slackAgent)
	}

	// Override the postMessageFunc for testing
	originalPostMessageFunc := sa.postMessageFunc
	sa.postMessageFunc = func(channelID, message string) (interface{}, error) {
		if channelID != "test-channel" {
			t.Errorf("Expected channel ID 'test-channel', got '%s'", channelID)
		}
		if message != "test message" {
			t.Errorf("Expected message 'test message', got '%s'", message)
		}
		return "1234567890.123456", nil
	}
	defer func() {
		sa.postMessageFunc = originalPostMessageFunc
	}()

	// Run the agent
	result, err := sa.Run("test message")
	if err != nil {
		t.Fatalf("Failed to run SlackAgent: %v", err)
	}

	// Check the result
	if result != "1234567890.123456" {
		t.Errorf("Expected result '1234567890.123456', got '%v'", result)
	}
}

// Made with Bob
