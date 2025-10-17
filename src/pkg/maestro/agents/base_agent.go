// SPDX-License-Identifier: Apache-2.0
// Copyright © 2025 IBM

package agents

import (
	"fmt"
)

// BaseAgent implements the Agent interface from the maestro package
type BaseAgent struct {
	*Agent
}

// NewBaseAgent creates a new BaseAgent
func NewBaseAgent(agent map[string]interface{}) (*BaseAgent, error) {
	baseAgent, err := NewAgent(agent)
	if err != nil {
		return nil, err
	}

	return &BaseAgent{
		Agent: baseAgent,
	}, nil
}

// Run implements the Agent interface Run method
func (b *BaseAgent) Run(args ...interface{}) (interface{}, error) {
	if len(args) == 0 {
		return nil, fmt.Errorf("no prompt provided")
	}

	prompt, ok := args[0].(string)
	if !ok {
		return nil, fmt.Errorf("prompt must be a string")
	}

	// This is a base implementation that should be overridden by specific agent types
	b.Print(fmt.Sprintf("Running with prompt: %s", prompt))

	// Track token usage
	b.TrackTokens(prompt, "Base agent response")

	return "This is a base agent implementation. Override this method in specific agent types.", nil
}

// GetName implements the Agent interface GetName method
func (b *BaseAgent) GetName() string {
	return b.AgentName
}

// GetModel implements the Agent interface GetModel method
func (b *BaseAgent) GetModel() string {
	return b.AgentModel
}

// Made with Bob
