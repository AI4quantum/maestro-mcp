// SPDX-License-Identifier: Apache-2.0
// Copyright © 2025 IBM

package agents

import (
	"os"
	"os/exec"
	"testing"
)

// TestNewCrewAIAgent tests the creation of a new CrewAIAgent
func TestNewCrewAIAgent(t *testing.T) {
	// Skip test if CrewAI is not installed
	if err := exec.Command("python", "-c", "import crewai").Run(); err != nil {
		t.Skip("Skipping test because CrewAI is not installed")
	}

	// Test with module configuration
	t.Run("ModuleConfiguration", func(t *testing.T) {
		agent := map[string]interface{}{
			"apiVersion": "maestro/v1alpha1",
			"kind":       "Agent",
			"metadata": map[string]interface{}{
				"name": "test-crewai-module-agent",
				"labels": map[string]interface{}{
					"module":  "test_module",
					"class":   "TestClass",
					"factory": "test_factory",
				},
			},
			"spec": map[string]interface{}{
				"framework": "crewai",
				"model":     "ollama/llama3.1",
				"url":       "http://localhost:11434",
			},
		}

		crewaiAgent, err := NewCrewAIAgent(agent)
		if err != nil {
			// This is expected if the module doesn't exist
			if _, ok := err.(*exec.ExitError); !ok {
				t.Fatalf("Expected exec.ExitError for non-existent module, got %T: %v", err, err)
			}
			return
		}

		ca, ok := crewaiAgent.(*CrewAIAgent)
		if !ok {
			t.Fatalf("Expected *CrewAIAgent, got %T", crewaiAgent)
		}

		if ca.ModuleName != "test_module" {
			t.Errorf("Expected module name 'test_module', got '%s'", ca.ModuleName)
		}

		if ca.ClassName != "TestClass" {
			t.Errorf("Expected class name 'TestClass', got '%s'", ca.ClassName)
		}

		if ca.FactoryName != "test_factory" {
			t.Errorf("Expected factory name 'test_factory', got '%s'", ca.FactoryName)
		}
	})

	// Test with direct configuration
	t.Run("DirectConfiguration", func(t *testing.T) {
		agent := map[string]interface{}{
			"apiVersion": "maestro/v1alpha1",
			"kind":       "Agent",
			"metadata": map[string]interface{}{
				"name": "test-crewai-direct-agent",
				"labels": map[string]interface{}{
					"crew_role":            "researcher",
					"crew_goal":            "find information",
					"crew_backstory":       "expert researcher",
					"crew_description":     "research the topic",
					"crew_expected_output": "detailed report",
				},
			},
			"spec": map[string]interface{}{
				"framework": "crewai",
				"model":     "gpt-4",
				"url":       "https://api.example.com/v1",
			},
		}

		crewaiAgent, err := NewCrewAIAgent(agent)
		if err != nil {
			t.Fatalf("Failed to create CrewAIAgent: %v", err)
		}

		ca, ok := crewaiAgent.(*CrewAIAgent)
		if !ok {
			t.Fatalf("Expected *CrewAIAgent, got %T", crewaiAgent)
		}

		if ca.AgentName != "test-crewai-direct-agent" {
			t.Errorf("Expected agent name 'test-crewai-direct-agent', got '%s'", ca.AgentName)
		}

		if ca.AgentFramework != "crewai" {
			t.Errorf("Expected agent framework 'crewai', got '%s'", ca.AgentFramework)
		}

		if ca.AgentModel != "gpt-4" {
			t.Errorf("Expected agent model 'gpt-4', got '%s'", ca.AgentModel)
		}

		if ca.ProviderURL != "https://api.example.com/v1" {
			t.Errorf("Expected provider URL 'https://api.example.com/v1', got '%s'", ca.ProviderURL)
		}

		if ca.CrewRole != "researcher" {
			t.Errorf("Expected crew role 'researcher', got '%s'", ca.CrewRole)
		}

		if ca.CrewGoal != "find information" {
			t.Errorf("Expected crew goal 'find information', got '%s'", ca.CrewGoal)
		}

		if ca.CrewBackstory != "expert researcher" {
			t.Errorf("Expected crew backstory 'expert researcher', got '%s'", ca.CrewBackstory)
		}

		if ca.CrewDescription != "research the topic" {
			t.Errorf("Expected crew description 'research the topic', got '%s'", ca.CrewDescription)
		}

		if ca.CrewExpectedOutput != "detailed report" {
			t.Errorf("Expected crew expected output 'detailed report', got '%s'", ca.CrewExpectedOutput)
		}
	})

	// Test with missing required fields
	t.Run("MissingRequiredFields", func(t *testing.T) {
		agent := map[string]interface{}{
			"apiVersion": "maestro/v1alpha1",
			"kind":       "Agent",
			"metadata": map[string]interface{}{
				"name":   "test-crewai-missing-fields",
				"labels": map[string]interface{}{
					// Missing required fields
				},
			},
			"spec": map[string]interface{}{
				"framework": "crewai",
			},
		}

		_, err := NewCrewAIAgent(agent)
		if err == nil {
			t.Fatalf("Expected error for missing required fields, got nil")
		}
	})
}

// TestCrewAIAgentRun tests the Run method of CrewAIAgent
func TestCrewAIAgentRun(t *testing.T) {
	// Skip test if CrewAI is not installed
	if err := exec.Command("python", "-c", "import crewai").Run(); err != nil {
		t.Skip("Skipping test because CrewAI is not installed")
	}

	// Create a mock Python module for testing
	mockPythonModule := `
class TestClass:
    def test_factory(self):
        return MockCrew()

class MockCrew:
    def kickoff(self, args):
        return "Mock response: " + args["prompt"]
`

	// Write the mock module to a temporary file
	tempDir, err := os.MkdirTemp("", "crewai-test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	modulePath := tempDir + "/mock_module.py"
	if err := os.WriteFile(modulePath, []byte(mockPythonModule), 0644); err != nil {
		t.Fatalf("Failed to write mock module: %v", err)
	}

	// Add the temp directory to PYTHONPATH
	originalPythonPath := os.Getenv("PYTHONPATH")
	os.Setenv("PYTHONPATH", tempDir+":"+originalPythonPath)
	defer os.Setenv("PYTHONPATH", originalPythonPath)

	// Create a test agent with the mock module
	agent := map[string]interface{}{
		"apiVersion": "maestro/v1alpha1",
		"kind":       "Agent",
		"metadata": map[string]interface{}{
			"name": "test-crewai-module-agent",
			"labels": map[string]interface{}{
				"module":  "mock_module",
				"class":   "TestClass",
				"factory": "test_factory",
			},
		},
		"spec": map[string]interface{}{
			"framework": "crewai",
		},
	}

	crewaiAgent, err := NewCrewAIAgent(agent)
	if err != nil {
		t.Fatalf("Failed to create CrewAIAgent: %v", err)
	}

	ca, ok := crewaiAgent.(*CrewAIAgent)
	if !ok {
		t.Fatalf("Expected *CrewAIAgent, got %T", crewaiAgent)
	}

	// Run the agent
	result, err := ca.Run("test prompt")
	if err != nil {
		t.Fatalf("Failed to run CrewAIAgent: %v", err)
	}

	// Check the result
	expectedResult := "Mock response: test prompt"
	if result != expectedResult {
		t.Errorf("Expected result '%s', got '%v'", expectedResult, result)
	}
}

// TestCrewAIAgentRunStreaming tests the RunStreaming method of CrewAIAgent
func TestCrewAIAgentRunStreaming(t *testing.T) {
	// Skip test if CrewAI is not installed
	if err := exec.Command("python", "-c", "import crewai").Run(); err != nil {
		t.Skip("Skipping test because CrewAI is not installed")
	}

	// Create a test agent
	agent := map[string]interface{}{
		"apiVersion": "maestro/v1alpha1",
		"kind":       "Agent",
		"metadata": map[string]interface{}{
			"name": "test-crewai-streaming-agent",
			"labels": map[string]interface{}{
				"crew_role":            "researcher",
				"crew_goal":            "find information",
				"crew_backstory":       "expert researcher",
				"crew_description":     "research the topic",
				"crew_expected_output": "detailed report",
			},
		},
		"spec": map[string]interface{}{
			"framework": "crewai",
			"model":     "gpt-4",
			"url":       "https://api.example.com/v1",
		},
	}

	crewaiAgent, err := NewCrewAIAgent(agent)
	if err != nil {
		t.Fatalf("Failed to create CrewAIAgent: %v", err)
	}

	ca, ok := crewaiAgent.(*CrewAIAgent)
	if !ok {
		t.Fatalf("Expected *CrewAIAgent, got %T", crewaiAgent)
	}

	// Run the agent in streaming mode
	_, err = ca.RunStreaming("test prompt")
	if err == nil {
		t.Fatalf("Expected error for streaming not implemented, got nil")
	}

	// Check the error message
	expectedError := "streaming execution for CrewAI agent 'test-crewai-streaming-agent' is not implemented yet"
	if err.Error() != expectedError {
		t.Errorf("Expected error message '%s', got '%s'", expectedError, err.Error())
	}
}

// Made with Bob
