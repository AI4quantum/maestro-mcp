// SPDX-License-Identifier: Apache-2.0
// Copyright © 2025 IBM

package maestro

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"go.uber.org/zap"
)

// SaveAgent saves an agent instance to the filesystem
func SaveAgent(agent Agent, agentDef map[string]interface{}) error {
	// Get agent name from metadata
	metadata, ok := agentDef["metadata"].(map[string]interface{})
	if !ok {
		return fmt.Errorf("invalid agent definition: missing metadata")
	}

	name, ok := metadata["name"].(string)
	if !ok {
		return fmt.Errorf("invalid agent definition: missing name")
	}

	// Create agents directory if it doesn't exist
	agentsDir := filepath.Join(os.TempDir(), "maestro", "agents")
	if err := os.MkdirAll(agentsDir, 0755); err != nil {
		return fmt.Errorf("failed to create agents directory: %w", err)
	}

	// Save agent definition to file
	agentPath := filepath.Join(agentsDir, name+".json")
	agentJSON, err := json.Marshal(agentDef)
	if err != nil {
		return fmt.Errorf("failed to marshal agent definition: %w", err)
	}

	if err := os.WriteFile(agentPath, agentJSON, 0644); err != nil {
		return fmt.Errorf("failed to write agent file: %w", err)
	}

	return nil
}

// Workflow represents a workflow execution environment
type Workflow struct {
	Agents            map[string]Agent
	Steps             map[string]*Step
	AgentDefs         []map[string]interface{}
	WorkflowDef       map[string]interface{}
	WorkflowID        string
	Logger            *zap.Logger
	Opik              interface{} // Placeholder for Opik equivalent
	ScoringMetrics    map[string]interface{}
	WorkflowModels    map[string]string
	WorkflowStartTime time.Time
	WorkflowEndTime   time.Time
	AgentExecTimes    map[string]float64
	TimingStarted     bool
	Context           map[string]interface{} // For storing context between steps

	// Mutex for thread safety
	mu sync.RWMutex
}

// NewWorkflow creates a new Workflow instance
func NewWorkflow(
	agentDefs []map[string]interface{},
	workflowDef map[string]interface{},
	workflowID string,
	logger *zap.Logger,
) (*Workflow, error) {
	workflow := &Workflow{
		Agents:         make(map[string]Agent),
		Steps:          make(map[string]*Step),
		AgentDefs:      agentDefs,
		WorkflowDef:    workflowDef,
		WorkflowID:     workflowID,
		Logger:         logger,
		ScoringMetrics: nil,
		WorkflowModels: make(map[string]string),
		AgentExecTimes: make(map[string]float64),
		TimingStarted:  false,
		Context:        make(map[string]interface{}),
	}

	return workflow, nil
}

// Close ensures timing is ended when workflow is destroyed
func (w *Workflow) Close() {
	if w.TimingStarted {
		w.endWorkflowTiming()
	}
}

// ToMermaid converts the workflow to a mermaid diagram
func (w *Workflow) ToMermaid(kind string, orientation string) (string, error) {
	if kind == "" {
		kind = "sequenceDiagram"
	}
	if orientation == "" {
		orientation = "TD"
	}

	mermaid := NewMermaid(w.WorkflowDef, kind, orientation)
	return mermaid.ToMarkdown()
}

// Run executes the workflow with the given prompt
func (w *Workflow) Run(ctx context.Context, prompt string) (*WorkflowResult, error) {
	// Set prompt if provided
	if prompt != "" {
		template, ok := w.WorkflowDef["spec"].(map[string]interface{})["template"].(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("invalid workflow definition: missing template")
		}
		template["prompt"] = prompt
	}

	// Create or restore agents
	if err := w.createOrRestoreAgents(); err != nil {
		return nil, fmt.Errorf("failed to create or restore agents: %w", err)
	}

	// Get template
	template, ok := w.WorkflowDef["spec"].(map[string]interface{})["template"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid workflow definition: missing template")
	}

	initialPrompt, ok := template["prompt"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid workflow definition: missing prompt")
	}

	// Start timing
	w.startWorkflowTiming()

	var result map[string]interface{}
	var err error

	// Check if this is an event-based workflow
	if _, hasEvent := template["event"]; hasEvent {
		result, err = w.runCondition(ctx, initialPrompt)
		w.endWorkflowTiming()
		if err != nil {
			return nil, err
		}

		// Process event
		eventResult, err := w.processEvent(ctx, result)
		if err != nil {
			return nil, err
		}

		return &WorkflowResult{
			FinalPrompt: eventResult["final_prompt"].(string),
			StepResults: convertMapToStringMap(eventResult),
		}, nil
	} else {
		// Regular workflow
		result, err = w.runCondition(ctx, initialPrompt)
		w.endWorkflowTiming()
		if err != nil {
			// Handle exception if defined
			if excDef, ok := template["exception"].(map[string]interface{}); ok {
				agentName, _ := excDef["agent"].(string)
				if agent, ok := w.Agents[agentName]; ok {
					_, _ = agent.Run(err.Error())
					return nil, err
				}
			}
			return nil, err
		}

		// Create workflow trace
		w.createWorkflowTrace(initialPrompt, result["final_prompt"].(string), convertMapToStringMap(result))

		return &WorkflowResult{
			FinalPrompt: result["final_prompt"].(string),
			StepResults: convertMapToStringMap(result),
		}, nil
	}
}

// RunStreaming executes the workflow with streaming results
func (w *Workflow) RunStreaming(ctx context.Context, prompt string) (<-chan *StreamResult, error) {
	// Set prompt if provided
	if prompt != "" {
		template, ok := w.WorkflowDef["spec"].(map[string]interface{})["template"].(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("invalid workflow definition: missing template")
		}
		template["prompt"] = prompt
	}

	// Create or restore agents
	if err := w.createOrRestoreAgents(); err != nil {
		return nil, fmt.Errorf("failed to create or restore agents: %w", err)
	}

	// Get template
	template, ok := w.WorkflowDef["spec"].(map[string]interface{})["template"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid workflow definition: missing template")
	}

	// Start timing
	w.startWorkflowTiming()

	// Create channel for streaming results
	resultChan := make(chan *StreamResult)

	// Run workflow in a goroutine
	go func() {
		defer close(resultChan)
		defer w.endWorkflowTiming()

		// Check if this is an event-based workflow
		if _, hasEvent := template["event"]; hasEvent {
			// Run condition with streaming
			stepResults, err := w.runConditionStreaming(ctx, resultChan)
			if err != nil {
				resultChan <- &StreamResult{
					Error: err,
				}
				return
			}

			// Process event
			result, err := w.processEvent(ctx, stepResults)
			if err != nil {
				resultChan <- &StreamResult{
					Error: err,
				}
				return
			}

			// Send final result
			resultChan <- &StreamResult{
				IsFinal:    true,
				StepResult: result["final_prompt"].(string),
			}
		} else {
			// Regular workflow with streaming
			_, err := w.runConditionStreaming(ctx, resultChan)
			if err != nil {
				// Handle exception if defined
				if excDef, ok := template["exception"].(map[string]interface{}); ok {
					agentName, _ := excDef["agent"].(string)
					if agent, ok := w.Agents[agentName]; ok {
						_, _ = agent.Run(err.Error())
					}
				}

				resultChan <- &StreamResult{
					Error: err,
				}
			}
		}
	}()

	return resultChan, nil
}

// GetContextState returns the current context state
func (w *Workflow) GetContextState() map[string]interface{} {
	w.mu.RLock()
	defer w.mu.RUnlock()

	return w.Context
}

// Helper methods (implementation details)

// getAgentClass returns the appropriate agent class based on framework and mode
func getAgentClass(framework string, mode string) (Agent, error) {
	// Check for dry run environment variable
	if os.Getenv("DRY_RUN") != "" {
		return &MockAgent{
			Name:  "MockAgent",
			Model: "mock",
		}, nil
	}

	// In a real implementation, this would use the agent factory
	// For now, return a mock agent
	return &MockAgent{
		Name:  "DefaultAgent",
		Model: fmt.Sprintf("%s:%s", framework, mode),
	}, nil
}

// createOrRestoreAgents creates or restores agents for the workflow
func (w *Workflow) createOrRestoreAgents() error {
	if len(w.AgentDefs) > 0 {
		for _, agentDef := range w.AgentDefs {
			// Check if agent definition is a string (agent name)
			if agentName, ok := agentDef["name"].(string); ok {
				// In a real implementation, we would restore the agent
				// For now, just create a mock agent
				w.Agents[agentName] = &MockAgent{
					Name:  agentName,
					Model: fmt.Sprintf("code:%s", agentName),
				}
				continue
			}

			// Get or set framework
			spec, ok := agentDef["spec"].(map[string]interface{})
			if !ok {
				return fmt.Errorf("invalid agent definition: missing spec")
			}

			framework, _ := spec["framework"].(string)
			if framework == "" {
				framework = "beeai" // Default framework
				spec["framework"] = framework
			}

			// Get agent class
			mode, _ := spec["mode"].(string)
			agentClass, err := getAgentClass(framework, mode)
			if err != nil {
				return fmt.Errorf("failed to get agent class: %w", err)
			}

			// Set agent properties
			metadata, ok := agentDef["metadata"].(map[string]interface{})
			if !ok {
				return fmt.Errorf("invalid agent definition: missing metadata")
			}

			agentName, ok := metadata["name"].(string)
			if !ok {
				return fmt.Errorf("invalid agent definition: missing name")
			}

			agentModel, _ := spec["model"].(string)
			if agentModel == "" {
				agentModel = fmt.Sprintf("code:%s", agentName)
			}

			// In a real implementation, we would set more properties
			// For now, just store the agent in the map
			w.Agents[agentName] = agentClass

			// Store model if not a scoring agent
			if !w.isScoringAgent(agentDef) {
				w.WorkflowModels[agentName] = agentModel
			}
		}
	} else {
		// Get agents from template
		template, ok := w.WorkflowDef["spec"].(map[string]interface{})["template"].(map[string]interface{})
		if !ok {
			return fmt.Errorf("invalid workflow definition: missing template")
		}

		agents, ok := template["agents"].([]interface{})
		if !ok {
			return nil // No agents defined
		}

		for _, agent := range agents {
			agentName, ok := agent.(string)
			if !ok {
				continue
			}

			// In a real implementation, we would restore the agent
			// For now, just create a mock agent
			w.Agents[agentName] = &MockAgent{
				Name:  agentName,
				Model: fmt.Sprintf("code:%s", agentName),
			}
		}
	}

	// Initialize Opik if there's a scoring agent
	if w.hasScoringAgent() {
		w.initializeOpik()
	}

	return nil
}

// findIndex finds the index of a step by name
func (w *Workflow) findIndex(steps []map[string]interface{}, name string) (int, error) {
	for i, step := range steps {
		if stepName, ok := step["name"].(string); ok && stepName == name {
			return i, nil
		}
	}
	return -1, fmt.Errorf("step not found: %s", name)
}

// runCondition runs the workflow steps based on conditions
func (w *Workflow) runCondition(ctx context.Context, initialPrompt string) (map[string]interface{}, error) {
	// Implementation omitted for brevity
	// This would contain the core workflow execution logic
	return map[string]interface{}{
		"final_prompt": initialPrompt,
	}, nil
}

// runConditionStreaming runs the workflow steps with streaming results
func (w *Workflow) runConditionStreaming(ctx context.Context, resultChan chan<- *StreamResult) (map[string]interface{}, error) {
	// Implementation omitted for brevity
	// This would contain the streaming workflow execution logic
	return map[string]interface{}{
		"final_prompt": "Streaming result",
	}, nil
}

// processEvent processes an event-based workflow
func (w *Workflow) processEvent(ctx context.Context, result map[string]interface{}) (map[string]interface{}, error) {
	// This is a simplified implementation of the event processing
	return result, nil
}

// startWorkflowTiming starts timing the workflow execution
func (w *Workflow) startWorkflowTiming() {
	w.WorkflowStartTime = time.Now()
	w.TimingStarted = true
}

// endWorkflowTiming ends timing the workflow execution
func (w *Workflow) endWorkflowTiming() {
	if w.TimingStarted && w.WorkflowEndTime.IsZero() {
		w.WorkflowEndTime = time.Now()
		w.TimingStarted = false
	}
}

// hasScoringAgent checks if there's a scoring agent in the workflow
func (w *Workflow) hasScoringAgent() bool {
	for _, agentDef := range w.AgentDefs {
		if w.isScoringAgent(agentDef) {
			return true
		}
	}
	return false
}

// isScoringAgent checks if an agent definition is for a scoring agent
func (w *Workflow) isScoringAgent(agentDef map[string]interface{}) bool {
	// Check if agent has scoring capability
	spec, ok := agentDef["spec"].(map[string]interface{})
	if !ok {
		return false
	}

	capabilities, ok := spec["capabilities"].([]interface{})
	if !ok {
		return false
	}

	for _, cap := range capabilities {
		if capStr, ok := cap.(string); ok && capStr == "scoring" {
			return true
		}
	}

	return false
}

// initializeOpik initializes the Opik integration
func (w *Workflow) initializeOpik() {
	// This is a placeholder for Opik initialization
	w.Opik = nil
}

// createWorkflowTrace creates a trace of the workflow execution
func (w *Workflow) createWorkflowTrace(initialPrompt string, finalPrompt string, stepResults map[string]string) {
	if w.Logger == nil {
		return
	}

	// Get workflow name
	workflowName := "unknown"
	if metadata, ok := w.WorkflowDef["metadata"].(map[string]interface{}); ok {
		if name, ok := metadata["name"].(string); ok {
			workflowName = name
		}
	}

	// Calculate duration
	var durationMS int64
	if !w.WorkflowStartTime.IsZero() && !w.WorkflowEndTime.IsZero() {
		durationMS = w.WorkflowEndTime.Sub(w.WorkflowStartTime).Milliseconds()
	}

	// Get models used
	modelsUsed := make([]string, 0, len(w.WorkflowModels))
	for _, model := range w.WorkflowModels {
		modelsUsed = append(modelsUsed, model)
	}

	// Log workflow run using zap logger
	w.Logger.Info("Workflow run completed",
		zap.String("workflow_id", w.WorkflowID),
		zap.String("workflow_name", workflowName),
		zap.String("prompt", initialPrompt),
		zap.String("output", finalPrompt),
		zap.Strings("models_used", modelsUsed),
		zap.String("status", "completed"),
		zap.Time("start_time", w.WorkflowStartTime),
		zap.Time("end_time", w.WorkflowEndTime),
		zap.Int64("duration_ms", durationMS),
	)
}

// CreateAgents creates agent instances from agent definitions
func CreateAgents(agentDefs []map[string]interface{}) error {
	for _, agentDef := range agentDefs {
		// Set default framework if not provided
		spec, ok := agentDef["spec"].(map[string]interface{})
		if !ok {
			return fmt.Errorf("invalid agent definition: missing spec")
		}

		framework, _ := spec["framework"].(string)
		if framework == "" {
			framework = "beeai" // Default framework
			spec["framework"] = framework
		}

		// Get agent class based on framework and mode
		mode, _ := spec["mode"].(string)
		agentClass, err := getAgentClass(framework, mode)
		if err != nil {
			return fmt.Errorf("failed to get agent class: %w", err)
		}

		// Save agent
		if err := SaveAgent(agentClass, agentDef); err != nil {
			return fmt.Errorf("failed to save agent: %w", err)
		}
	}

	return nil
}

// Made with Bob
