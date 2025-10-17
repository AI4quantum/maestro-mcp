// SPDX-License-Identifier: Apache-2.0
// Copyright © 2025 IBM

package agents

import (
	"fmt"
)

// CustomAgentCreator is a function type that creates a custom agent
type CustomAgentCreator func(agent map[string]interface{}) (interface{}, error)

// CustomAgentRegistry maps custom agent names to their creator functions
var CustomAgentRegistry = map[string]CustomAgentCreator{
	// These would be implemented separately
	"slack_agent":   createSlackAgent,
	"scoring_agent": createScoringAgent,
	"prompt_agent":  createPromptAgent,
	"query_agent":   createQueryAgent,
}

// CustomAgent is a proxy that dispatches to the configured custom agent
type CustomAgent struct {
	*BaseAgent
	agent interface{} // The actual agent implementation
}

// NewCustomAgent creates a new CustomAgent
func NewCustomAgent(agentDef map[string]interface{}) (interface{}, error) {
	// Create the base agent
	baseAgent, err := NewBaseAgent(agentDef)
	if err != nil {
		return nil, err
	}

	// Get the custom agent type from metadata.labels.custom_agent
	metadata, ok := agentDef["metadata"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid agent definition: missing metadata")
	}

	labels, ok := metadata["labels"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid agent definition: missing metadata.labels")
	}

	customAgentType, ok := labels["custom_agent"].(string)
	if !ok || customAgentType == "" {
		return nil, fmt.Errorf("invalid agent definition: missing or empty metadata.labels.custom_agent")
	}

	// Check if the custom agent type is registered
	creator, ok := CustomAgentRegistry[customAgentType]
	if !ok {
		return nil, fmt.Errorf("unknown custom_agent '%s'", customAgentType)
	}

	// Create the actual agent
	agent, err := creator(agentDef)
	if err != nil {
		return nil, fmt.Errorf("failed to create custom agent '%s': %w", customAgentType, err)
	}

	return &CustomAgent{
		BaseAgent: baseAgent,
		agent:     agent,
	}, nil
}

// Run implements the Agent interface Run method
func (c *CustomAgent) Run(args ...interface{}) (interface{}, error) {
	// Forward the call to the underlying agent
	if runner, ok := c.agent.(interface {
		Run(args ...interface{}) (interface{}, error)
	}); ok {
		return runner.Run(args...)
	}
	return nil, fmt.Errorf("underlying agent does not implement Run method")
}

// Placeholder implementations for custom agent creators
// These would be implemented in separate files in a real implementation

func createSlackAgent(agent map[string]interface{}) (interface{}, error) {
	// Use the actual SlackAgent implementation
	return NewSlackAgent(agent)
}

func createScoringAgent(agent map[string]interface{}) (interface{}, error) {
	// Use the actual ScoringAgent implementation
	return NewScoringAgent(agent)
}

func createPromptAgent(agent map[string]interface{}) (interface{}, error) {
	// This is a placeholder - in a real implementation, this would create a PromptAgent
	return NewBaseAgent(agent)
}

func createQueryAgent(agent map[string]interface{}) (interface{}, error) {
	// Use the actual QueryAgent implementation
	return NewQueryAgent(agent)
}

// Made with Bob
