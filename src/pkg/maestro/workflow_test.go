// SPDX-License-Identifier: Apache-2.0
// Copyright © 2025 IBM

package maestro

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// TestWorkflowAgent is a mock implementation of the Agent interface for testing
type TestWorkflowAgent struct {
	AgentName    string
	AgentModel   string
	MockResponse string
	MockError    error
}

// Run implements the Agent interface
func (m *TestWorkflowAgent) Run(args ...interface{}) (interface{}, error) {
	if m.MockError != nil {
		return nil, m.MockError
	}
	return m.MockResponse, nil
}

// GetName implements the Agent interface
func (m *TestWorkflowAgent) GetName() string {
	return m.AgentName
}

// GetModel implements the Agent interface
func (m *TestWorkflowAgent) GetModel() string {
	return m.AgentModel
}

// TestWorkflowRun tests the Run method of the Workflow struct
func TestWorkflowRun(t *testing.T) {
	// Create a logger for testing
	logger, _ := zap.NewDevelopment()

	// Test cases
	tests := []struct {
		name           string
		agentDefs      []map[string]interface{}
		workflowDef    map[string]interface{}
		prompt         string
		mockAgents     map[string]Agent
		expectedResult string
		expectError    bool
	}{
		{
			name: "Simple workflow with one step",
			agentDefs: []map[string]interface{}{
				{
					"metadata": map[string]interface{}{
						"name": "agent1",
					},
					"spec": map[string]interface{}{
						"framework": "mock",
						"model":     "mock-model",
					},
				},
			},
			workflowDef: map[string]interface{}{
				"metadata": map[string]interface{}{
					"name": "test-workflow",
				},
				"spec": map[string]interface{}{
					"template": map[string]interface{}{
						"agents": []string{"agent1"},
						"prompt": "Initial prompt",
						"steps": []map[string]interface{}{
							{
								"name":  "step1",
								"agent": "agent1",
							},
						},
					},
				},
			},
			prompt: "Test prompt",
			mockAgents: map[string]Agent{
				"agent1": &TestWorkflowAgent{
					AgentName:    "agent1",
					AgentModel:   "mock-model",
					MockResponse: "Response from agent1",
				},
			},
			expectedResult: "Response from agent1",
			expectError:    false,
		},
		{
			name: "Workflow with error",
			agentDefs: []map[string]interface{}{
				{
					"metadata": map[string]interface{}{
						"name": "agent1",
					},
					"spec": map[string]interface{}{
						"framework": "mock",
						"model":     "mock-model",
					},
				},
			},
			workflowDef: map[string]interface{}{
				"metadata": map[string]interface{}{
					"name": "test-workflow",
				},
				"spec": map[string]interface{}{
					"template": map[string]interface{}{
						"agents": []string{"agent1"},
						"prompt": "Initial prompt",
						"steps": []map[string]interface{}{
							{
								"name":  "step1",
								"agent": "agent1",
							},
						},
					},
				},
			},
			prompt: "Test prompt",
			mockAgents: map[string]Agent{
				"agent1": &TestWorkflowAgent{
					AgentName:  "agent1",
					AgentModel: "mock-model",
					MockError:  errors.New("test error"),
				},
			},
			expectedResult: "",
			expectError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a test workflow
			testWorkflow, err := NewTestWorkflow(tt.agentDefs, tt.workflowDef, "test-workflow-id", logger)
			require.NoError(t, err)

			// Add mock agents to the workflow
			for name, agent := range tt.mockAgents {
				testWorkflow.Agents[name] = agent
				testWorkflow.WorkflowModels[name] = agent.GetModel()
			}

			// Run the workflow
			result, err := testWorkflow.Run(context.Background(), tt.prompt)

			// Check results
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedResult, result.FinalPrompt)
			}
		})
	}
}

// TestWorkflowRunStreaming tests the RunStreaming method of the Workflow struct
func TestWorkflowRunStreaming(t *testing.T) {
	// Create a logger for testing
	logger, _ := zap.NewDevelopment()

	// Create a simple workflow definition
	agentDefs := []map[string]interface{}{
		{
			"metadata": map[string]interface{}{
				"name": "agent1",
			},
			"spec": map[string]interface{}{
				"framework": "mock",
				"model":     "mock-model",
			},
		},
	}

	workflowDef := map[string]interface{}{
		"metadata": map[string]interface{}{
			"name": "test-workflow",
		},
		"spec": map[string]interface{}{
			"template": map[string]interface{}{
				"agents": []string{"agent1"},
				"prompt": "Initial prompt",
				"steps": []map[string]interface{}{
					{
						"name":  "step1",
						"agent": "agent1",
					},
				},
			},
		},
	}

	// Create a test workflow
	testWorkflow, err := NewTestWorkflow(agentDefs, workflowDef, "test-workflow-id", logger)
	require.NoError(t, err)

	// Create a mock agent
	mockAgent := &TestWorkflowAgent{
		AgentName:    "agent1",
		AgentModel:   "mock-model",
		MockResponse: "Response from agent1",
	}

	// Add mock agent to the workflow
	testWorkflow.Agents["agent1"] = mockAgent
	testWorkflow.WorkflowModels["agent1"] = mockAgent.AgentModel

	// Run the streaming workflow
	resultChan, err := testWorkflow.RunStreaming(context.Background(), "Test prompt")
	require.NoError(t, err)

	// Collect results
	var results []*StreamResult
	for result := range resultChan {
		results = append(results, result)
	}

	// Verify we got at least one result
	assert.NotEmpty(t, results)
}

// Made with Bob
