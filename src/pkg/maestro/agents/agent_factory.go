// SPDX-License-Identifier: Apache-2.0
// Copyright © 2025 IBM

package agents

import (
	"fmt"
	"log"
)

// AgentFramework represents the type of agent framework
type AgentFramework string

// Supported agent frameworks
const (
	BeeAI   AgentFramework = "beeai"
	CrewAI  AgentFramework = "crewai"
	Dspy    AgentFramework = "dspy"
	OpenAI  AgentFramework = "openai"
	Mock    AgentFramework = "mock"
	Remote  AgentFramework = "remote"
	Custom  AgentFramework = "custom"
	Code    AgentFramework = "code"
	Slack   AgentFramework = "slack"
	Scoring AgentFramework = "scoring"
	Query   AgentFramework = "query"
)

// AgentCreator is a function type that creates an agent
type AgentCreator func(agent map[string]interface{}) (interface{}, error)

// AgentFactory handles the creation of different types of agents
type AgentFactory struct {
	factories       map[AgentFramework]AgentCreator
	remoteFactories map[AgentFramework]AgentCreator
}

// NewAgentFactory creates a new agent factory
func NewAgentFactory() *AgentFactory {
	// In a real implementation, these would be actual agent implementations
	// For now, we'll use placeholder functions that return BaseAgent

	// Create a factory with placeholder implementations
	factory := &AgentFactory{
		factories:       make(map[AgentFramework]AgentCreator),
		remoteFactories: make(map[AgentFramework]AgentCreator),
	}

	// Register local agent factories
	factory.factories[BeeAI] = createBeeAIAgent
	factory.factories[CrewAI] = createCrewAIAgent
	factory.factories[Dspy] = createDspyAgent
	factory.factories[OpenAI] = createOpenAIAgent
	factory.factories[Code] = createCodeAgent
	factory.factories[Mock] = createMockAgent
	factory.factories[Slack] = createSlackAgent
	factory.factories[Scoring] = createScoringAgent
	factory.factories[Query] = createQueryAgent

	// Register remote agent factories
	factory.remoteFactories[Remote] = createRemoteAgent
	factory.remoteFactories[Mock] = createMockAgent

	return factory
}

// CreateAgent creates an agent of the specified framework and mode
func (f *AgentFactory) CreateAgent(framework AgentFramework, mode string) (AgentCreator, error) {
	// Handle custom agent separately
	if framework == Custom {
		return createCustomAgent, nil
	}

	// Check if the framework is supported
	_, localExists := f.factories[framework]
	_, remoteExists := f.remoteFactories[framework]

	if !localExists && !remoteExists {
		return nil, fmt.Errorf("unknown framework: %s", framework)
	}

	// Handle remote mode
	if mode == "remote" || framework == Remote {
		if framework == BeeAI {
			// BeeAI remote mode is no longer supported, fall back to local
			log.Printf("BeeAI remote mode is no longer supported, falling back to local mode")
			return f.factories[framework], nil
		}

		if creator, ok := f.remoteFactories[framework]; ok {
			return creator, nil
		}
	}

	// Default to local mode
	return f.factories[framework], nil
}

// GetFactory is a convenience method that calls CreateAgent
func (f *AgentFactory) GetFactory(framework string, mode string) (AgentCreator, error) {
	return f.CreateAgent(AgentFramework(framework), mode)
}

// Placeholder agent creator functions
// In a real implementation, these would create actual agent instances

func createBeeAIAgent(agent map[string]interface{}) (interface{}, error) {
	return NewBeeAIAgent(agent)
}

func createCrewAIAgent(agent map[string]interface{}) (interface{}, error) {
	return NewCrewAIAgent(agent)
}

func createDspyAgent(agent map[string]interface{}) (interface{}, error) {
	return NewDSPyAgent(agent)
}

func createOpenAIAgent(agent map[string]interface{}) (interface{}, error) {
	return NewOpenAIAgent(agent)
}

func createCodeAgent(agent map[string]interface{}) (interface{}, error) {
	return NewBaseAgent(agent)
}

func createMockAgent(agent map[string]interface{}) (interface{}, error) {
	return NewBaseAgent(agent)
}

func createRemoteAgent(agent map[string]interface{}) (interface{}, error) {
	return NewBaseAgent(agent)
}

func createCustomAgent(agent map[string]interface{}) (interface{}, error) {
	return NewBaseAgent(agent)
}

// Made with Bob
