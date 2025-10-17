// SPDX-License-Identifier: Apache-2.0
// Copyright © 2025 IBM

package agents

import (
	"fmt"
	"os"
	"strings"
)

// ScoringAgent extends the BaseAgent to score responses using relevance and hallucination metrics
type ScoringAgent struct {
	*BaseAgent
	Name         string
	LitellmModel string
	// Function pointers for metrics calculation to allow mocking in tests
	calculateRelevance     func(prompt, response string, context []string) (float64, string, error)
	calculateHallucination func(prompt, response string, context []string) (float64, string, error)
}

// NewScoringAgent creates a new ScoringAgent
func NewScoringAgent(agent map[string]interface{}) (interface{}, error) {
	// Create the base agent
	baseAgent, err := NewBaseAgent(agent)
	if err != nil {
		return nil, err
	}

	// Extract name from metadata
	metadata, ok := agent["metadata"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid agent definition: missing metadata")
	}

	name, ok := metadata["name"].(string)
	if !ok {
		name = "scoring-agent"
	}

	// Extract model from spec
	spec, ok := agent["spec"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid agent definition: missing spec")
	}

	rawModel, ok := spec["model"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid agent definition: missing model")
	}

	// Format the model name for litellm
	litellmModel := rawModel
	if !strings.HasPrefix(rawModel, "ollama/") && !strings.HasPrefix(rawModel, "openai/") {
		litellmModel = fmt.Sprintf("ollama/%s", rawModel)
	}

	// Create the agent
	scoringAgent := &ScoringAgent{
		BaseAgent:    baseAgent,
		Name:         name,
		LitellmModel: litellmModel,
	}

	// Set the metric calculation functions
	scoringAgent.calculateRelevance = scoringAgent.defaultCalculateRelevance
	scoringAgent.calculateHallucination = scoringAgent.defaultCalculateHallucination

	return scoringAgent, nil
}

// Run implements the Agent interface Run method
func (s *ScoringAgent) Run(args ...interface{}) (interface{}, error) {
	// Check that we have at least prompt and response
	if len(args) < 2 {
		return nil, fmt.Errorf("scoring agent requires at least prompt and response arguments")
	}

	// Extract prompt
	prompt, ok := args[0].(string)
	if !ok {
		return nil, fmt.Errorf("prompt must be a string")
	}

	// Extract response
	response, ok := args[1].(string)
	if !ok {
		return nil, fmt.Errorf("response must be a string")
	}

	// Extract context (optional)
	var context []string
	if len(args) > 2 {
		if contextArg, ok := args[2].([]string); ok {
			context = contextArg
		} else if contextArg, ok := args[2].([]interface{}); ok {
			// Convert []interface{} to []string
			for _, c := range contextArg {
				if str, ok := c.(string); ok {
					context = append(context, str)
				}
			}
		}
	}

	// If no context provided, use prompt as context
	if len(context) == 0 {
		context = []string{prompt}
	}

	// Calculate metrics
	metrics, err := s.calculateMetrics(prompt, response, context)
	if err != nil {
		s.Print(fmt.Sprintf("[ScoringAgent] Warning: could not calculate metrics: %v", err))
		return map[string]interface{}{
			"prompt":          response,
			"scoring_metrics": nil,
		}, nil
	}

	// Log metrics
	s.logMetricsToTrace(metrics)

	// Print metrics
	s.printMetrics(response, metrics)

	// Format and return response
	return s.formatResponse(response, metrics), nil
}

// calculateMetrics calculates relevance and hallucination metrics for the response
func (s *ScoringAgent) calculateMetrics(prompt, response string, context []string) (map[string]interface{}, error) {
	// Set environment variable to disable tracking
	os.Setenv("OPIK_TRACK_DISABLE", "true")
	defer os.Unsetenv("OPIK_TRACK_DISABLE")

	// Calculate relevance
	relevanceScore, relevanceReason, err := s.calculateRelevance(prompt, response, context)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate relevance: %w", err)
	}

	// Calculate hallucination
	hallucinationScore, hallucinationReason, err := s.calculateHallucination(prompt, response, context)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate hallucination: %w", err)
	}

	// Return metrics
	return map[string]interface{}{
		"relevance":            relevanceScore,
		"hallucination":        hallucinationScore,
		"relevance_reason":     s.normalizeReason(relevanceReason),
		"hallucination_reason": s.normalizeReason(hallucinationReason),
	}, nil
}

// defaultCalculateRelevance is the default implementation of the relevance metric
func (s *ScoringAgent) defaultCalculateRelevance(prompt, response string, context []string) (float64, string, error) {
	// This is a placeholder implementation
	// In a real implementation, this would call the Opik library
	s.Print("[ScoringAgent] Using placeholder implementation for relevance metric")
	return 0.75, "Response appears to be relevant to the prompt", nil
}

// defaultCalculateHallucination is the default implementation of the hallucination metric
func (s *ScoringAgent) defaultCalculateHallucination(prompt, response string, context []string) (float64, string, error) {
	// This is a placeholder implementation
	// In a real implementation, this would call the Opik library
	s.Print("[ScoringAgent] Using placeholder implementation for hallucination metric")
	return 0.25, "Response appears to be grounded in the context", nil
}

// normalizeReason normalizes the reason field from metrics into a string
func (s *ScoringAgent) normalizeReason(reason interface{}) string {
	switch r := reason.(type) {
	case []string:
		return strings.Join(r, ", ")
	case string:
		return r
	default:
		return ""
	}
}

// logMetricsToTrace logs scoring metrics to the current trace
func (s *ScoringAgent) logMetricsToTrace(metrics map[string]interface{}) {
	// This is a placeholder implementation
	// In a real implementation, this would call the Opik library
	s.Print("[ScoringAgent] Logging metrics to trace (placeholder)")
}

// printMetrics prints the scoring metrics to stdout
func (s *ScoringAgent) printMetrics(response string, metrics map[string]interface{}) {
	relevance, _ := metrics["relevance"].(float64)
	hallucination, _ := metrics["hallucination"].(float64)
	metricsLine := fmt.Sprintf("relevance: %.2f, hallucination: %.2f", relevance, hallucination)
	s.Print(fmt.Sprintf("%s\n[%s]", response, metricsLine))
}

// formatResponse formats the final response with scoring metrics
func (s *ScoringAgent) formatResponse(response string, metrics map[string]interface{}) map[string]interface{} {
	return map[string]interface{}{
		"prompt": response,
		"scoring_metrics": map[string]interface{}{
			"relevance":            metrics["relevance"],
			"hallucination":        metrics["hallucination"],
			"relevance_reason":     metrics["relevance_reason"],
			"hallucination_reason": metrics["hallucination_reason"],
			"model":                s.LitellmModel,
			"agent":                s.Name,
			"provider":             "ollama",
		},
	}
}

// Made with Bob
