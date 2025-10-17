// SPDX-License-Identifier: Apache-2.0
// Copyright © 2025 IBM

package maestro

import (
	"context"

	"go.uber.org/zap"
)

// TestWorkflow is a test-specific implementation for workflow testing
type TestWorkflow struct {
	Agents         map[string]Agent
	WorkflowModels map[string]string
	WorkflowDef    map[string]interface{}
	Logger         *zap.Logger
}

// NewTestWorkflow creates a new TestWorkflow instance for testing
func NewTestWorkflow(
	agentDefs []map[string]interface{},
	workflowDef map[string]interface{},
	workflowID string,
	logger *zap.Logger,
) (*TestWorkflow, error) {
	return &TestWorkflow{
		Agents:         make(map[string]Agent),
		WorkflowModels: make(map[string]string),
		WorkflowDef:    workflowDef,
		Logger:         logger,
	}, nil
}

// Run overrides the Run method for testing
func (tw *TestWorkflow) Run(ctx context.Context, prompt string) (*WorkflowResult, error) {
	// For testing, we'll use the first agent's response as the final prompt
	for _, agent := range tw.Agents {
		response, err := agent.Run(prompt)
		if err != nil {
			return nil, err
		}

		// Convert response to string if needed
		responseStr := ""
		if str, ok := response.(string); ok {
			responseStr = str
		} else {
			responseStr = "Mock response"
		}

		return &WorkflowResult{
			FinalPrompt: responseStr,
			StepResults: map[string]string{
				"step1": responseStr,
			},
		}, nil
	}

	// If no agents, just return the initial prompt
	return &WorkflowResult{
		FinalPrompt: prompt,
		StepResults: map[string]string{},
	}, nil
}

// RunStreaming overrides the RunStreaming method for testing
func (tw *TestWorkflow) RunStreaming(ctx context.Context, prompt string) (<-chan *StreamResult, error) {
	resultChan := make(chan *StreamResult, 2)

	go func() {
		defer close(resultChan)

		// For testing, we'll use the first agent's response
		for name, agent := range tw.Agents {
			response, err := agent.Run(prompt)
			if err != nil {
				resultChan <- &StreamResult{
					Error: err,
				}
				return
			}

			// Convert response to string if needed
			responseStr := ""
			if str, ok := response.(string); ok {
				responseStr = str
			} else {
				responseStr = "Mock streaming response"
			}

			// Send a stream result
			resultChan <- &StreamResult{
				StepName:   "step1",
				StepResult: responseStr,
				StepIndex:  0,
				AgentName:  name,
				IsFinal:    false,
			}

			// Send final result
			resultChan <- &StreamResult{
				StepName:   "step1",
				StepResult: responseStr,
				StepIndex:  0,
				AgentName:  name,
				IsFinal:    true,
			}

			return
		}

		// If no agents, just send a default response
		resultChan <- &StreamResult{
			StepName:   "step1",
			StepResult: "Default streaming response",
			StepIndex:  0,
			AgentName:  "default",
			IsFinal:    true,
		}
	}()

	return resultChan, nil
}

// Made with Bob
