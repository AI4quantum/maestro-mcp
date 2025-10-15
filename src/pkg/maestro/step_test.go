// SPDX-License-Identifier: Apache-2.0
// Copyright © 2025 IBM

package maestro

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestStepAgent is a mock implementation of the Agent interface for testing steps
type TestStepAgent struct {
	Name         string
	Model        string
	MockResponse string
	MockError    error
}

// Run implements the Agent interface
func (m *TestStepAgent) Run(args ...interface{}) (interface{}, error) {
	if m.MockError != nil {
		return nil, m.MockError
	}
	return m.MockResponse, nil
}

// GetName implements the Agent interface
func (m *TestStepAgent) GetName() string {
	return m.Name
}

// GetModel implements the Agent interface
func (m *TestStepAgent) GetModel() string {
	return m.Model
}

// TestNewStep tests the NewStep function
func TestNewStep(t *testing.T) {
	tests := []struct {
		name        string
		stepDef     map[string]interface{}
		expectError bool
	}{
		{
			name: "Valid step definition with agent",
			stepDef: map[string]interface{}{
				"name":  "test-step",
				"agent": &TestStepAgent{Name: "test-agent", Model: "test-model"},
			},
			expectError: false,
		},
		{
			name: "Valid step definition with workflow",
			stepDef: map[string]interface{}{
				"name":     "test-step",
				"workflow": "http://example.com/workflow",
			},
			expectError: false,
		},
		{
			name: "Valid step definition with input",
			stepDef: map[string]interface{}{
				"name": "test-step",
				"input": map[string]interface{}{
					"template": "Template: {prompt}",
					"prompt":   "User prompt: {prompt}",
				},
			},
			expectError: false,
		},
		{
			name: "Valid step definition with condition",
			stepDef: map[string]interface{}{
				"name": "test-step",
				"condition": []map[string]interface{}{
					{
						"if":   "prompt.contains('test')",
						"then": "next-step",
						"else": "error-step",
					},
				},
			},
			expectError: false,
		},
		{
			name:        "Missing name",
			stepDef:     map[string]interface{}{},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			step, err := NewStep(tt.stepDef)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, step)
				assert.Equal(t, tt.stepDef["name"], step.Name)
			}
		})
	}
}

// TestStepRun tests the Run method of the Step struct
func TestStepRun(t *testing.T) {
	// Create a test server for workflow tests
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"response": "Workflow response"}`))
	}))
	defer server.Close()

	tests := []struct {
		name           string
		step           *Step
		input          interface{}
		stepIndex      int
		expectedPrompt string
		expectError    bool
	}{
		{
			name: "Step with agent",
			step: &Step{
				Name: "agent-step",
				Agent: &TestStepAgent{
					Name:         "test-agent",
					Model:        "test-model",
					MockResponse: "Agent response",
				},
			},
			input:          "Test input",
			stepIndex:      0,
			expectedPrompt: "Agent response",
			expectError:    false,
		},
		{
			name: "Step with agent error",
			step: &Step{
				Name: "error-step",
				Agent: &TestStepAgent{
					Name:      "error-agent",
					Model:     "test-model",
					MockError: errors.New("agent error"),
				},
			},
			input:       "Test input",
			stepIndex:   0,
			expectError: true,
		},
		{
			name: "Step with workflow",
			step: &Step{
				Name:     "workflow-step",
				Workflow: server.URL + "/chat",
			},
			input:          "Test input",
			stepIndex:      0,
			expectedPrompt: "Workflow response",
			expectError:    false,
		},
		{
			name: "Step with no agent or workflow",
			step: &Step{
				Name: "passthrough-step",
			},
			input:          "Test input",
			stepIndex:      0,
			expectedPrompt: "Test input",
			expectError:    false,
		},
		{
			name: "Step with condition",
			step: &Step{
				Name: "condition-step",
				Agent: &TestStepAgent{
					Name:         "test-agent",
					Model:        "test-model",
					MockResponse: "Contains test keyword",
				},
				Condition: []map[string]interface{}{
					{
						"if":   "prompt.contains('test')",
						"then": "next-step",
						"else": "error-step",
					},
				},
			},
			input:          "Test input",
			stepIndex:      0,
			expectedPrompt: "Contains test keyword",
			expectError:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := tt.step.Run(context.Background(), tt.input, tt.stepIndex)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, result)
				assert.Equal(t, tt.expectedPrompt, result.Prompt)
			}
		})
	}
}

// TestProcessResult tests the processResult method of the Step struct
func TestProcessResult(t *testing.T) {
	step := &Step{
		Name: "test-step",
	}

	tests := []struct {
		name           string
		result         interface{}
		expectedPrompt string
		expectedNext   string
	}{
		{
			name:           "String result",
			result:         "Test result",
			expectedPrompt: "Test result",
			expectedNext:   "",
		},
		{
			name: "Map result with prompt and next",
			result: map[string]interface{}{
				"prompt": "Test prompt",
				"next":   "next-step",
			},
			expectedPrompt: "Test prompt",
			expectedNext:   "next-step",
		},
		{
			name: "StepResult",
			result: &StepResult{
				Prompt: "Test prompt",
				Next:   "next-step",
			},
			expectedPrompt: "Test prompt",
			expectedNext:   "next-step",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := step.processResult(tt.result)
			require.NoError(t, err)
			assert.Equal(t, tt.expectedPrompt, result.Prompt)
			assert.Equal(t, tt.expectedNext, result.Next)
		})
	}
}

// TestEvaluateCondition tests the evaluateCondition method of the Step struct
func TestEvaluateCondition(t *testing.T) {
	tests := []struct {
		name         string
		step         *Step
		prompt       string
		expectedNext string
		expectError  bool
	}{
		{
			name: "Simple if condition",
			step: &Step{
				Name: "if-condition-step",
				Condition: []map[string]interface{}{
					{
						"if":   "true",
						"then": "then-step",
						"else": "else-step",
					},
				},
			},
			prompt:       "This is a test prompt",
			expectedNext: "then-step",
			expectError:  false,
		},
		{
			name: "Simple if condition false",
			step: &Step{
				Name: "if-condition-step",
				Condition: []map[string]interface{}{
					{
						"if":   "false",
						"then": "then-step",
						"else": "else-step",
					},
				},
			},
			prompt:       "This is a test prompt",
			expectedNext: "else-step",
			expectError:  false,
		},
		{
			name: "Simple case condition",
			step: &Step{
				Name: "case-condition-step",
				Condition: []map[string]interface{}{
					{
						"case": "true",
						"do":   "test-step",
					},
					{
						"case": "false",
						"do":   "other-step",
					},
				},
			},
			prompt:       "This is a test prompt",
			expectedNext: "test-step",
			expectError:  false,
		},
		{
			name: "Default case",
			step: &Step{
				Name: "case-condition-step",
				Condition: []map[string]interface{}{
					{
						"case": "false",
						"do":   "missing-step",
					},
					{
						"do": "default-step",
					},
				},
			},
			prompt:       "This is a test prompt",
			expectedNext: "default-step",
			expectError:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			next, err := tt.step.evaluateCondition(tt.prompt)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expectedNext, next)
			}
		})
	}
}

// TestRunParallel tests the runParallel method of the Step struct
func TestRunParallel(t *testing.T) {
	tests := []struct {
		name        string
		step        *Step
		prompt      string
		stepIndex   int
		expectError bool
	}{
		{
			name: "Run multiple agents in parallel",
			step: &Step{
				Name: "parallel-step",
				Parallel: []Agent{
					&TestStepAgent{
						Name:         "agent1",
						Model:        "test-model",
						MockResponse: "Response from agent1",
					},
					&TestStepAgent{
						Name:         "agent2",
						Model:        "test-model",
						MockResponse: "Response from agent2",
					},
				},
			},
			prompt:      "Test input",
			stepIndex:   0,
			expectError: false,
		},
		{
			name: "Run with list input",
			step: &Step{
				Name: "parallel-list-step",
				Parallel: []Agent{
					&TestStepAgent{
						Name:         "agent1",
						Model:        "test-model",
						MockResponse: "Response from agent1",
					},
					&TestStepAgent{
						Name:         "agent2",
						Model:        "test-model",
						MockResponse: "Response from agent2",
					},
				},
			},
			prompt:      "[\"Input 1\", \"Input 2\"]",
			stepIndex:   0,
			expectError: false,
		},
		{
			name: "Error in parallel execution",
			step: &Step{
				Name: "parallel-error-step",
				Parallel: []Agent{
					&TestStepAgent{
						Name:      "error-agent",
						Model:     "test-model",
						MockError: errors.New("agent error"),
					},
				},
			},
			prompt:      "Test input",
			stepIndex:   0,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := tt.step.runParallel(context.Background(), tt.prompt, tt.stepIndex)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.NotEmpty(t, result)
			}
		})
	}
}

// TestRunLoop tests the runLoop method of the Step struct
func TestRunLoop(t *testing.T) {
	tests := []struct {
		name        string
		step        *Step
		prompt      string
		stepIndex   int
		expectError bool
	}{
		{
			name: "Simple loop with until condition",
			step: &Step{
				Name: "loop-step",
				Loop: map[string]interface{}{
					"agent": &TestStepAgent{
						Name:         "loop-agent",
						Model:        "test-model",
						MockResponse: "Final response",
					},
					"until": "prompt == 'Final response'",
				},
			},
			prompt:      "Initial input",
			stepIndex:   0,
			expectError: false,
		},
		{
			name: "Loop with list input",
			step: &Step{
				Name: "loop-list-step",
				Loop: map[string]interface{}{
					"agent": &TestStepAgent{
						Name:         "loop-agent",
						Model:        "test-model",
						MockResponse: "Processed item",
					},
					"until": "true", // Not used for list input
				},
			},
			prompt:      "[\"Item 1\", \"Item 2\"]",
			stepIndex:   0,
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := tt.step.runLoop(context.Background(), tt.prompt, tt.stepIndex)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.NotEmpty(t, result)
			}
		})
	}
}

// Made with Bob
