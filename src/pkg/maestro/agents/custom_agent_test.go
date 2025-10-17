// SPDX-License-Identifier: Apache-2.0
// Copyright © 2025 IBM

package agents

import (
	"testing"
)

// Mock custom agent for testing
type mockCustomAgent struct {
	*BaseAgent
	runCalled bool
	response  string
}

func (m *mockCustomAgent) Run(args ...interface{}) (interface{}, error) {
	m.runCalled = true
	return m.response, nil
}

// Register a mock custom agent creator for testing
func init() {
	CustomAgentRegistry["mock_custom_agent"] = func(agent map[string]interface{}) (interface{}, error) {
		baseAgent, err := NewBaseAgent(agent)
		if err != nil {
			return nil, err
		}
		return &mockCustomAgent{
			BaseAgent: baseAgent,
			response:  "Mock custom agent response",
		}, nil
	}
}

func TestNewCustomAgent(t *testing.T) {
	// Create a test agent definition
	agentDef := map[string]interface{}{
		"metadata": map[string]interface{}{
			"name": "test-custom-agent",
			"labels": map[string]interface{}{
				"custom_agent": "mock_custom_agent",
			},
		},
		"spec": map[string]interface{}{
			"framework":    "custom",
			"description":  "Test custom agent",
			"instructions": "This is a test custom agent",
		},
	}

	// Create a new custom agent
	agent, err := NewCustomAgent(agentDef)
	if err != nil {
		t.Fatalf("Failed to create custom agent: %v", err)
	}

	// Check that the agent is a CustomAgent
	customAgent, ok := agent.(*CustomAgent)
	if !ok {
		t.Fatalf("Expected agent to be a CustomAgent, got %T", agent)
	}

	// Check agent properties
	if customAgent.AgentName != "test-custom-agent" {
		t.Errorf("Expected agent name to be 'test-custom-agent', got '%s'", customAgent.AgentName)
	}
	if customAgent.AgentFramework != "custom" {
		t.Errorf("Expected agent framework to be 'custom', got '%s'", customAgent.AgentFramework)
	}

	// Check that the underlying agent is a mockCustomAgent
	_, ok = customAgent.agent.(*mockCustomAgent)
	if !ok {
		t.Errorf("Expected underlying agent to be a mockCustomAgent, got %T", customAgent.agent)
	}
}

func TestCustomAgentRun(t *testing.T) {
	// Create a test agent definition
	agentDef := map[string]interface{}{
		"metadata": map[string]interface{}{
			"name": "test-custom-agent",
			"labels": map[string]interface{}{
				"custom_agent": "mock_custom_agent",
			},
		},
		"spec": map[string]interface{}{
			"framework":    "custom",
			"description":  "Test custom agent",
			"instructions": "This is a test custom agent",
		},
	}

	// Create a new custom agent
	agent, err := NewCustomAgent(agentDef)
	if err != nil {
		t.Fatalf("Failed to create custom agent: %v", err)
	}

	customAgent := agent.(*CustomAgent)
	mockAgent := customAgent.agent.(*mockCustomAgent)

	// Run the agent
	result, err := customAgent.Run("test prompt")
	if err != nil {
		t.Fatalf("Failed to run custom agent: %v", err)
	}

	// Check that the underlying agent's Run method was called
	if !mockAgent.runCalled {
		t.Error("Expected underlying agent's Run method to be called")
	}

	// Check the result
	if result != "Mock custom agent response" {
		t.Errorf("Expected result to be 'Mock custom agent response', got '%v'", result)
	}
}

func TestCustomAgentWithInvalidDefinition(t *testing.T) {
	testCases := []struct {
		name     string
		agentDef map[string]interface{}
	}{
		{
			name: "Missing metadata",
			agentDef: map[string]interface{}{
				"spec": map[string]interface{}{
					"framework": "custom",
				},
			},
		},
		{
			name: "Missing labels",
			agentDef: map[string]interface{}{
				"metadata": map[string]interface{}{
					"name": "test-custom-agent",
				},
				"spec": map[string]interface{}{
					"framework": "custom",
				},
			},
		},
		{
			name: "Missing custom_agent",
			agentDef: map[string]interface{}{
				"metadata": map[string]interface{}{
					"name":   "test-custom-agent",
					"labels": map[string]interface{}{},
				},
				"spec": map[string]interface{}{
					"framework": "custom",
				},
			},
		},
		{
			name: "Unknown custom_agent",
			agentDef: map[string]interface{}{
				"metadata": map[string]interface{}{
					"name": "test-custom-agent",
					"labels": map[string]interface{}{
						"custom_agent": "unknown_agent",
					},
				},
				"spec": map[string]interface{}{
					"framework": "custom",
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := NewCustomAgent(tc.agentDef)
			if err == nil {
				t.Errorf("Expected error for invalid agent definition, got nil")
			}
		})
	}
}

// Made with Bob
