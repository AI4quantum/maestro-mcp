// SPDX-License-Identifier: Apache-2.0
// Copyright © 2025 IBM

package agents

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Agent is the base struct for all agent implementations
type Agent struct {
	AgentName      string
	AgentFramework string
	AgentModel     string
	AgentURL       string
	AgentTools     []interface{}
	AgentDesc      string
	AgentInstr     string
	AgentInput     string
	AgentOutput    string
	AgentCode      string
	Instructions   string

	// Token counters for LLM-style agents
	PromptTokens   int
	ResponseTokens int
	TotalTokens    int
}

// Emojis maps agent frameworks to their emoji representations
var Emojis = map[string]string{
	"beeai":   "🐝",
	"crewai":  "👥",
	"dspy":    "💭",
	"openai":  "🔓",
	"mock":    "🤖",
	"remote":  "💸",
	"slack":   "💬",
	"scoring": "📊",
	"query":   "🔍",
}

// NewAgent creates a new agent from an agent definition
func NewAgent(agent map[string]interface{}) (*Agent, error) {
	// Extract metadata
	metadata, ok := agent["metadata"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid agent definition: missing metadata")
	}

	name, ok := metadata["name"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid agent definition: missing name")
	}

	// Extract spec
	spec, ok := agent["spec"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid agent definition: missing spec")
	}

	// Extract fields with defaults
	framework, _ := spec["framework"].(string)
	model, _ := spec["model"].(string)
	url, _ := spec["url"].(string)

	var tools []interface{}
	if toolsVal, ok := spec["tools"]; ok {
		if toolsSlice, ok := toolsVal.([]interface{}); ok {
			tools = toolsSlice
		}
	}

	description, _ := spec["description"].(string)
	instructions, _ := spec["instructions"].(string)
	input, _ := spec["input"].(string)
	output, _ := spec["output"].(string)
	code, _ := spec["code"].(string)

	// Get content from file if source_file is provided
	sourceFile, _ := agent["source_file"].(string)
	if sourceFile != "" {
		if instructions == "" {
			instructions = getContent(spec["instructions"], sourceFile)
		}
		if code == "" {
			code = getContent(spec["code"], sourceFile)
		}
	}

	// Build full instructions
	fullInstructions := instructions
	if input != "" {
		fullInstructions = fmt.Sprintf("%s Input is expected in format: %s", fullInstructions, input)
	}
	if output != "" {
		fullInstructions = fmt.Sprintf("%s Output must be in format: %s", fullInstructions, output)
	}

	return &Agent{
		AgentName:      name,
		AgentFramework: framework,
		AgentModel:     model,
		AgentURL:       url,
		AgentTools:     tools,
		AgentDesc:      description,
		AgentInstr:     instructions,
		AgentInput:     input,
		AgentOutput:    output,
		AgentCode:      code,
		Instructions:   fullInstructions,
		PromptTokens:   0,
		ResponseTokens: 0,
		TotalTokens:    0,
	}, nil
}

// Emoji returns the emoji for the agent's framework
func (a *Agent) Emoji() string {
	emoji, ok := Emojis[a.AgentFramework]
	if !ok {
		return "⚙️" // Default emoji
	}
	return emoji
}

// Print prints a message with timestamp and agent emoji
func (a *Agent) Print(message string) {
	now := time.Now()
	formattedTime := now.Format("01-02-2006 15:04:05")
	fmt.Printf("%s %s: %s\n", a.Emoji(), formattedTime, message)
}

// GetTokenUsage returns token usage statistics for the agent
func (a *Agent) GetTokenUsage() map[string]interface{} {
	if a.AgentFramework == "custom" {
		if a.AgentName != "" && strings.Contains(strings.ToLower(a.AgentName), "scoring") {
			return map[string]interface{}{
				"agent_type":  "scoring_agent",
				"description": "Uses Opik evaluation metrics (relevance, hallucination)",
			}
		}
		return map[string]interface{}{
			"agent_type":  "custom_agent",
			"description": "Custom agent - no traditional token usage",
		}
	}

	return map[string]interface{}{
		"prompt_tokens":   a.PromptTokens,
		"response_tokens": a.ResponseTokens,
		"total_tokens":    a.TotalTokens,
	}
}

// ResetTokenUsage resets token usage counters to zero
func (a *Agent) ResetTokenUsage() {
	a.PromptTokens = 0
	a.ResponseTokens = 0
	a.TotalTokens = 0
}

// CountTokens counts tokens for text using a shared utility
func (a *Agent) CountTokens(text string) int {
	agentLabel := fmt.Sprintf("%T %s", a, a.AgentName)
	// This is a simplified implementation
	// In a real implementation, you would use a tokenizer library
	tokenCount := len(text) / 4 // Rough approximation
	a.Print(fmt.Sprintf("Counted %d tokens for %s", tokenCount, agentLabel))
	return tokenCount
}

// TrackTokens computes and stores token usage for a prompt/response pair
func (a *Agent) TrackTokens(prompt string, response string) map[string]int {
	agentLabel := fmt.Sprintf("%T %s", a, a.AgentName)

	promptTokens := a.CountTokens(prompt)
	responseTokens := a.CountTokens(response)
	totalTokens := promptTokens + responseTokens

	a.PromptTokens = promptTokens
	a.ResponseTokens = responseTokens
	a.TotalTokens = totalTokens

	a.Print(fmt.Sprintf("Token usage for %s: %d prompt, %d response, %d total",
		agentLabel, promptTokens, responseTokens, totalTokens))

	return map[string]int{
		"prompt_tokens":   promptTokens,
		"response_tokens": responseTokens,
		"total_tokens":    totalTokens,
	}
}

// ExtractAndSetTokenUsageFromResult extracts token usage from a provider-specific result object
func (a *Agent) ExtractAndSetTokenUsageFromResult(result interface{}) map[string]int {
	agentLabel := fmt.Sprintf("%T %s", a, a.AgentName)

	// This is a simplified implementation
	// In a real implementation, you would extract token usage from the provider's response

	// For now, just log that we're extracting tokens
	a.Print(fmt.Sprintf("Extracting token usage from result for %s", agentLabel))

	// Return default values
	return map[string]int{
		"prompt_tokens":   a.PromptTokens,
		"response_tokens": a.ResponseTokens,
		"total_tokens":    a.TotalTokens,
	}
}

// Helper function to get content from a file or return the default value
func getContent(value interface{}, sourceFile string) string {
	if value == nil {
		return ""
	}

	if strValue, ok := value.(string); ok {
		// If it's a file path, read the file
		if strings.HasPrefix(strValue, "file://") {
			filePath := strings.TrimPrefix(strValue, "file://")
			// If the path is relative, make it relative to the source file directory
			if !filepath.IsAbs(filePath) && sourceFile != "" {
				filePath = filepath.Join(filepath.Dir(sourceFile), filePath)
			}

			content, err := os.ReadFile(filePath)
			if err == nil {
				return string(content)
			}
		}
		return strValue
	}

	return ""
}

// AgentDB represents the agent database
type AgentDB struct {
	Agents map[string][]byte
}

// LoadAgentDB loads agents from database file
func LoadAgentDB() (*AgentDB, error) {
	db := &AgentDB{
		Agents: make(map[string][]byte),
	}

	// Check if agents.db exists
	if _, err := os.Stat("agents.db"); os.IsNotExist(err) {
		return db, nil
	}

	// Read the file
	data, err := os.ReadFile("agents.db")
	if err != nil {
		return nil, fmt.Errorf("failed to read agents.db: %w", err)
	}

	// Unmarshal the data
	if err := json.Unmarshal(data, &db.Agents); err != nil {
		return nil, fmt.Errorf("failed to unmarshal agents.db: %w", err)
	}

	return db, nil
}

// SaveAgentDB saves the agent database to a file
func SaveAgentDB(db *AgentDB) error {
	data, err := json.Marshal(db.Agents)
	if err != nil {
		return fmt.Errorf("failed to marshal agents: %w", err)
	}

	return os.WriteFile("agents.db", data, 0644)
}

// SaveAgent saves an agent to the database
func SaveAgent(agent interface{}, agentDef map[string]interface{}) error {
	db, err := LoadAgentDB()
	if err != nil {
		return err
	}

	// Get agent name
	metadata, ok := agentDef["metadata"].(map[string]interface{})
	if !ok {
		return fmt.Errorf("invalid agent definition: missing metadata")
	}

	name, ok := metadata["name"].(string)
	if !ok {
		return fmt.Errorf("invalid agent definition: missing name")
	}

	// Serialize the agent
	var agentData []byte
	var serializeErr error

	// Try to serialize the agent object
	agentData, serializeErr = json.Marshal(agent)
	if serializeErr != nil {
		// If that fails, serialize the agent definition
		agentData, serializeErr = json.Marshal(agentDef)
		if serializeErr != nil {
			return fmt.Errorf("failed to serialize agent: %w", serializeErr)
		}
	}

	// Save to database
	db.Agents[name] = agentData
	return SaveAgentDB(db)
}

// RestoreAgent restores an agent from the database
func RestoreAgent(agentName string) (interface{}, bool, error) {
	db, err := LoadAgentDB()
	if err != nil {
		return nil, false, err
	}

	agentData, ok := db.Agents[agentName]
	if !ok {
		return agentName, false, nil
	}

	// Try to determine if this is an agent definition or a serialized agent
	var agentDef map[string]interface{}
	if err := json.Unmarshal(agentData, &agentDef); err != nil {
		// If it's not a JSON object, it's probably a serialized agent
		var agent Agent
		if err := json.Unmarshal(agentData, &agent); err != nil {
			return nil, false, fmt.Errorf("failed to unmarshal agent data: %w", err)
		}
		return &agent, true, nil
	}

	// Check if it's an agent definition
	if _, ok := agentDef["metadata"]; ok {
		if apiVersion, ok := agentDef["apiVersion"].(string); ok && strings.Contains(apiVersion, "maestro/v1alpha1") {
			return agentDef, false, nil
		}

		// Create a new agent from the definition
		agent, err := NewAgent(agentDef)
		if err != nil {
			return nil, false, fmt.Errorf("failed to create agent from definition: %w", err)
		}
		return agent, true, nil
	}

	// Default to treating it as a serialized agent
	var agent Agent
	if err := json.Unmarshal(agentData, &agent); err != nil {
		return nil, false, fmt.Errorf("failed to unmarshal agent data: %w", err)
	}
	return &agent, true, nil
}

// RemoveAgent removes an agent from the database
func RemoveAgent(agentName string) error {
	db, err := LoadAgentDB()
	if err != nil {
		return err
	}

	delete(db.Agents, agentName)
	return SaveAgentDB(db)
}

// Made with Bob
