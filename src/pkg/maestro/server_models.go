// SPDX-License-Identifier: Apache-2.0
// Copyright © 2025 IBM

package maestro

import (
	"time"
)

// ChatRequest represents a request to chat with an agent
type ChatRequest struct {
	Prompt string `json:"prompt"`
	Stream bool   `json:"stream,omitempty"`
}

// ChatResponse represents a response from an agent
type ChatResponse struct {
	Response  string    `json:"response"`
	AgentName string    `json:"agent_name"`
	Timestamp time.Time `json:"timestamp"`
}

// HealthResponse represents a health check response
type HealthResponse struct {
	Status    string    `json:"status"`
	AgentName string    `json:"agent_name,omitempty"`
	Timestamp time.Time `json:"timestamp"`
}

// AgentListResponse represents a response listing available agents
type AgentListResponse struct {
	Agents       []string `json:"agents"`
	CurrentAgent string   `json:"current_agent,omitempty"`
}

// WorkflowChatRequest represents a request to chat with a workflow
type WorkflowChatRequest struct {
	Prompt string `json:"prompt"`
}

// WorkflowChatResponse represents a response from a workflow
type WorkflowChatResponse struct {
	Response     string    `json:"response"`
	WorkflowName string    `json:"workflow_name"`
	Timestamp    time.Time `json:"timestamp"`
}

// WorkflowHealthResponse represents a health check response for a workflow
type WorkflowHealthResponse struct {
	Status       string    `json:"status"`
	WorkflowName string    `json:"workflow_name"`
	Timestamp    time.Time `json:"timestamp"`
}

// DiagramResponse represents a response containing a workflow diagram
type DiagramResponse struct {
	Diagram      string `json:"diagram"`
	WorkflowName string `json:"workflow_name"`
}

// StreamEvent represents an event in a streaming response
type StreamEvent struct {
	Response         string `json:"response,omitempty"`
	AgentName        string `json:"agent_name,omitempty"`
	StepName         string `json:"step_name,omitempty"`
	StepResult       string `json:"step_result,omitempty"`
	StepComplete     bool   `json:"step_complete,omitempty"`
	WorkflowName     string `json:"workflow_name,omitempty"`
	WorkflowComplete bool   `json:"workflow_complete,omitempty"`
	Error            string `json:"error,omitempty"`
}

// Made with Bob
