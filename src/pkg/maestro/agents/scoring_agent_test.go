// SPDX-License-Identifier: Apache-2.0
// Copyright © 2025 IBM

package agents

import (
	"testing"
)

func TestNewScoringAgent(t *testing.T) {
	// Create a test agent definition
	agent := map[string]interface{}{
		"apiVersion": "maestro/v1alpha1",
		"kind":       "Agent",
		"metadata": map[string]interface{}{
			"name": "test-scoring-agent",
			"labels": map[string]interface{}{
				"custom_agent": "scoring_agent",
			},
		},
		"spec": map[string]interface{}{
			"framework": "scoring",
			"model":     "llama3",
		},
	}

	// Create the agent
	scoringAgent, err := NewScoringAgent(agent)
	if err != nil {
		t.Fatalf("Failed to create ScoringAgent: %v", err)
	}

	// Check that the agent was created correctly
	sa, ok := scoringAgent.(*ScoringAgent)
	if !ok {
		t.Fatalf("Expected *ScoringAgent, got %T", scoringAgent)
	}

	// Check agent properties
	if sa.AgentName != "test-scoring-agent" {
		t.Errorf("Expected agent name 'test-scoring-agent', got '%s'", sa.AgentName)
	}

	if sa.AgentFramework != "scoring" {
		t.Errorf("Expected agent framework 'scoring', got '%s'", sa.AgentFramework)
	}

	if sa.LitellmModel != "ollama/llama3" {
		t.Errorf("Expected litellm model 'ollama/llama3', got '%s'", sa.LitellmModel)
	}
}

func TestScoringAgentRun(t *testing.T) {
	// Create a test agent definition
	agent := map[string]interface{}{
		"apiVersion": "maestro/v1alpha1",
		"kind":       "Agent",
		"metadata": map[string]interface{}{
			"name": "test-scoring-agent",
			"labels": map[string]interface{}{
				"custom_agent": "scoring_agent",
			},
		},
		"spec": map[string]interface{}{
			"framework": "scoring",
			"model":     "llama3",
		},
	}

	// Create the agent
	scoringAgent, err := NewScoringAgent(agent)
	if err != nil {
		t.Fatalf("Failed to create ScoringAgent: %v", err)
	}

	sa, ok := scoringAgent.(*ScoringAgent)
	if !ok {
		t.Fatalf("Expected *ScoringAgent, got %T", scoringAgent)
	}

	// Override the metric calculation functions for testing
	sa.calculateRelevance = func(prompt, response string, context []string) (float64, string, error) {
		if prompt != "test prompt" {
			t.Errorf("Expected prompt 'test prompt', got '%s'", prompt)
		}
		if response != "test response" {
			t.Errorf("Expected response 'test response', got '%s'", response)
		}
		if len(context) != 1 || context[0] != "test prompt" {
			t.Errorf("Expected context ['test prompt'], got %v", context)
		}
		return 0.9, "Response is highly relevant", nil
	}

	sa.calculateHallucination = func(prompt, response string, context []string) (float64, string, error) {
		if prompt != "test prompt" {
			t.Errorf("Expected prompt 'test prompt', got '%s'", prompt)
		}
		if response != "test response" {
			t.Errorf("Expected response 'test response', got '%s'", response)
		}
		if len(context) != 1 || context[0] != "test prompt" {
			t.Errorf("Expected context ['test prompt'], got %v", context)
		}
		return 0.1, "Response is well-grounded", nil
	}

	// Run the agent
	result, err := sa.Run("test prompt", "test response")
	if err != nil {
		t.Fatalf("Failed to run ScoringAgent: %v", err)
	}

	// Check the result
	resultMap, ok := result.(map[string]interface{})
	if !ok {
		t.Fatalf("Expected map[string]interface{}, got %T", result)
	}

	// Check that the prompt is returned
	prompt, ok := resultMap["prompt"].(string)
	if !ok || prompt != "test response" {
		t.Errorf("Expected prompt 'test response', got '%v'", resultMap["prompt"])
	}

	// Check that the scoring metrics are returned
	metrics, ok := resultMap["scoring_metrics"].(map[string]interface{})
	if !ok {
		t.Fatalf("Expected scoring_metrics to be map[string]interface{}, got %T", resultMap["scoring_metrics"])
	}

	// Check relevance score
	relevance, ok := metrics["relevance"].(float64)
	if !ok || relevance != 0.9 {
		t.Errorf("Expected relevance 0.9, got %v", metrics["relevance"])
	}

	// Check hallucination score
	hallucination, ok := metrics["hallucination"].(float64)
	if !ok || hallucination != 0.1 {
		t.Errorf("Expected hallucination 0.1, got %v", metrics["hallucination"])
	}

	// Check model
	model, ok := metrics["model"].(string)
	if !ok || model != "ollama/llama3" {
		t.Errorf("Expected model 'ollama/llama3', got %v", metrics["model"])
	}
}

func TestScoringAgentRunWithContext(t *testing.T) {
	// Create a test agent definition
	agent := map[string]interface{}{
		"apiVersion": "maestro/v1alpha1",
		"kind":       "Agent",
		"metadata": map[string]interface{}{
			"name": "test-scoring-agent",
			"labels": map[string]interface{}{
				"custom_agent": "scoring_agent",
			},
		},
		"spec": map[string]interface{}{
			"framework": "scoring",
			"model":     "llama3",
		},
	}

	// Create the agent
	scoringAgent, err := NewScoringAgent(agent)
	if err != nil {
		t.Fatalf("Failed to create ScoringAgent: %v", err)
	}

	sa, ok := scoringAgent.(*ScoringAgent)
	if !ok {
		t.Fatalf("Expected *ScoringAgent, got %T", scoringAgent)
	}

	// Override the metric calculation functions for testing
	sa.calculateRelevance = func(prompt, response string, context []string) (float64, string, error) {
		if prompt != "test prompt" {
			t.Errorf("Expected prompt 'test prompt', got '%s'", prompt)
		}
		if response != "test response" {
			t.Errorf("Expected response 'test response', got '%s'", response)
		}
		if len(context) != 2 || context[0] != "context1" || context[1] != "context2" {
			t.Errorf("Expected context ['context1', 'context2'], got %v", context)
		}
		return 0.8, "Response is relevant", nil
	}

	sa.calculateHallucination = func(prompt, response string, context []string) (float64, string, error) {
		if prompt != "test prompt" {
			t.Errorf("Expected prompt 'test prompt', got '%s'", prompt)
		}
		if response != "test response" {
			t.Errorf("Expected response 'test response', got '%s'", response)
		}
		if len(context) != 2 || context[0] != "context1" || context[1] != "context2" {
			t.Errorf("Expected context ['context1', 'context2'], got %v", context)
		}
		return 0.2, "Response has some hallucination", nil
	}

	// Run the agent with context
	result, err := sa.Run("test prompt", "test response", []string{"context1", "context2"})
	if err != nil {
		t.Fatalf("Failed to run ScoringAgent: %v", err)
	}

	// Check the result
	resultMap, ok := result.(map[string]interface{})
	if !ok {
		t.Fatalf("Expected map[string]interface{}, got %T", result)
	}

	// Check that the scoring metrics are returned
	metrics, ok := resultMap["scoring_metrics"].(map[string]interface{})
	if !ok {
		t.Fatalf("Expected scoring_metrics to be map[string]interface{}, got %T", resultMap["scoring_metrics"])
	}

	// Check relevance score
	relevance, ok := metrics["relevance"].(float64)
	if !ok || relevance != 0.8 {
		t.Errorf("Expected relevance 0.8, got %v", metrics["relevance"])
	}

	// Check hallucination score
	hallucination, ok := metrics["hallucination"].(float64)
	if !ok || hallucination != 0.2 {
		t.Errorf("Expected hallucination 0.2, got %v", metrics["hallucination"])
	}
}

// Made with Bob
