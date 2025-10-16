// SPDX-License-Identifier: Apache-2.0
// Copyright © 2025 IBM

package agents

import (
	"testing"
)

func TestNewAgentFactory(t *testing.T) {
	factory := NewAgentFactory()

	if factory == nil {
		t.Fatal("Expected non-nil factory")
	}

	// Check that factories are initialized
	if len(factory.factories) == 0 {
		t.Error("Expected non-empty factories map")
	}

	if len(factory.remoteFactories) == 0 {
		t.Error("Expected non-empty remote factories map")
	}
}

func TestCreateAgent(t *testing.T) {
	factory := NewAgentFactory()

	testCases := []struct {
		name      string
		framework AgentFramework
		mode      string
		expectErr bool
	}{
		{"BeeAI Local", BeeAI, "local", false},
		{"BeeAI Remote", BeeAI, "remote", false}, // Should fall back to local
		// {"CrewAI Local", CrewAI, "local", false},
		{"Dspy Local", Dspy, "local", false},
		{"OpenAI Local", OpenAI, "local", false},
		{"Mock Local", Mock, "local", false},
		{"Mock Remote", Mock, "remote", false},
		{"Remote", Remote, "local", false}, // Remote framework always uses remote mode
		{"Custom", Custom, "local", false},
		{"Code Local", Code, "local", false},
		{"Unknown", AgentFramework("unknown"), "local", true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			creator, err := factory.CreateAgent(tc.framework, tc.mode)

			if tc.expectErr {
				if err == nil {
					t.Errorf("Expected error for framework %s, mode %s", tc.framework, tc.mode)
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error for framework %s, mode %s: %v", tc.framework, tc.mode, err)
				return
			}

			if creator == nil {
				t.Errorf("Expected non-nil creator for framework %s, mode %s", tc.framework, tc.mode)
				return
			}

			// Test that the creator can create an agent
			agentDef := map[string]interface{}{
				"metadata": map[string]interface{}{
					"name": "test-agent",
				},
				"spec": map[string]interface{}{
					"framework": string(tc.framework),
				},
			}

			agent, err := creator(agentDef)
			if err != nil {
				t.Errorf("Failed to create agent: %v", err)
				return
			}

			if agent == nil {
				t.Error("Expected non-nil agent")
			}
		})
	}
}

func TestGetFactory(t *testing.T) {
	factory := NewAgentFactory()

	// Test with string framework
	creator, err := factory.GetFactory("beeai", "local")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if creator == nil {
		t.Error("Expected non-nil creator")
	}

	// Test with invalid framework
	_, err = factory.GetFactory("invalid", "local")
	if err == nil {
		t.Error("Expected error for invalid framework")
	}
}

// Made with Bob
