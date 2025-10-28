// SPDX-License-Identifier: Apache-2.0
// Copyright © 2025 IBM

package maestro

import (
	"context"
	"fmt"
	"time"
)

// Agent interface represents the common behavior of all agents
type Agent interface {
	Run(args ...interface{}) (interface{}, error)
	GetName() string
	GetModel() string
}

// AgentFramework represents the type of agent framework
type AgentFramework string

const (
	OpenAI AgentFramework = "openai"
	BeeAI  AgentFramework = "beeai"
	Custom AgentFramework = "custom"
)

// StepResult represents the result of running a step
type StepResult struct {
	Prompt   string                 // The output prompt
	Next     string                 // The next step to execute (if any)
	Metadata map[string]interface{} // Additional metadata
}

// WorkflowResult represents the result of running a workflow
type WorkflowResult struct {
	FinalPrompt string
	StepResults map[string]string
	Error       error
}

// StreamResult represents a streaming result from a workflow step
type StreamResult struct {
	StepName   string
	StepResult string
	StepIndex  int
	AgentName  string
	IsFinal    bool
	Error      error
}

// ExecutionMetrics represents workflow execution metrics
type ExecutionMetrics struct {
	WorkflowExecTimeSeconds float64
	AgentExecTimes          map[string]float64
	TotalAgentTimeSeconds   float64
	WorkflowStartTime       time.Time
	WorkflowEndTime         time.Time
	TimingStatus            string
}

// StepError represents an error that occurred in a step
type StepError struct {
	StepName string
	Err      error
}

func (e *StepError) Error() string {
	return fmt.Sprintf("error in step '%s': %v", e.StepName, e.Err)
}

// AgentError represents an error that occurred in an agent
type AgentError struct {
	AgentName string
	Err       error
}

func (e *AgentError) Error() string {
	return fmt.Sprintf("error in agent '%s': %v", e.AgentName, e.Err)
}

// StepRunner interface for running steps
type StepRunner interface {
	Run(ctx context.Context, input interface{}, stepIndex int) (*StepResult, error)
}

// WorkflowRunner interface for running workflows
type WorkflowRunner interface {
	Run(ctx context.Context, prompt string) (*WorkflowResult, error)
	RunStreaming(ctx context.Context, prompt string) (<-chan *StreamResult, error)
}

// MockAgent implements the Agent interface for dry runs
type MockAgent struct {
	Name  string
	Model string
}

func (m *MockAgent) Run(args ...interface{}) (interface{}, error) {
	// Mock implementation
	return "Mock response", nil
}

func (m *MockAgent) GetName() string {
	return m.Name
}

func (m *MockAgent) GetModel() string {
	return m.Model
}

// Constants for common keys
const (
	PromptKey    = "prompt"
	NextKey      = "next"
	AgentKey     = "agent"
	WorkflowKey  = "workflow"
	NameKey      = "name"
	FromKey      = "from"
	ConditionKey = "condition"
	ParallelKey  = "parallel"
	LoopKey      = "loop"
	IfKey        = "if"
	ThenKey      = "then"
	ElseKey      = "else"
	CaseKey      = "case"
	DoKey        = "do"
	UntilKey     = "until"
)

// Made with Bob
