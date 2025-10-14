// SPDX-License-Identifier: Apache-2.0
// Copyright © 2025 IBM

package maestro

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"sync"
)

// StripThinkTags removes <think>...</think> tags from text
func StripThinkTags(text string) string {
	if text == "" {
		return text
	}

	re := regexp.MustCompile(`(?s)<think>.*?</think>`)
	return strings.TrimSpace(re.ReplaceAllString(text, ""))
}

// EvalExpression evaluates a simple expression against a context
// This is a simplified version of Python's eval_expression
func EvalExpression(expr string, context interface{}) (bool, error) {
	// Handle simple string comparison expressions
	if strings.Contains(expr, "==") {
		parts := strings.Split(expr, "==")
		if len(parts) != 2 {
			return false, fmt.Errorf("invalid expression: %s", expr)
		}

		left := strings.TrimSpace(parts[0])
		right := strings.TrimSpace(parts[1])

		// Extract values from context if needed
		leftVal, err := extractValueFromContext(left, context)
		if err != nil {
			return false, err
		}

		rightVal, err := extractValueFromContext(right, context)
		if err != nil {
			return false, err
		}

		return leftVal == rightVal, nil
	}

	// Handle contains expression
	if strings.Contains(expr, "in") {
		parts := strings.Split(expr, "in")
		if len(parts) != 2 {
			return false, fmt.Errorf("invalid expression: %s", expr)
		}

		item := strings.TrimSpace(parts[0])
		collection := strings.TrimSpace(parts[1])

		// Extract values from context
		itemVal, err := extractValueFromContext(item, context)
		if err != nil {
			return false, err
		}

		collectionVal, err := extractValueFromContext(collection, context)
		if err != nil {
			return false, err
		}

		// Check if item is in collection
		switch c := collectionVal.(type) {
		case string:
			return strings.Contains(c, fmt.Sprintf("%v", itemVal)), nil
		case []interface{}:
			for _, v := range c {
				if v == itemVal {
					return true, nil
				}
			}
			return false, nil
		default:
			return false, fmt.Errorf("unsupported collection type: %T", collectionVal)
		}
	}

	// Handle simple boolean expressions
	switch strings.ToLower(expr) {
	case "true":
		return true, nil
	case "false":
		return false, nil
	}

	// For more complex expressions, we'd need a proper expression evaluator
	// This is a simplified implementation
	return false, fmt.Errorf("unsupported expression: %s", expr)
}

// extractValueFromContext extracts a value from the context
func extractValueFromContext(key string, context interface{}) (interface{}, error) {
	// If key is a literal string in quotes, return it without quotes
	if (strings.HasPrefix(key, "'") && strings.HasSuffix(key, "'")) ||
		(strings.HasPrefix(key, "\"") && strings.HasSuffix(key, "\"")) {
		return key[1 : len(key)-1], nil
	}

	// If context is a string and key is "prompt", return the context
	if contextStr, ok := context.(string); ok && key == "prompt" {
		return contextStr, nil
	}

	// If context is a map, try to get the value by key
	if contextMap, ok := context.(map[string]interface{}); ok {
		if val, exists := contextMap[key]; exists {
			return val, nil
		}
	}

	// If key is a number or boolean literal, return it
	switch key {
	case "true":
		return true, nil
	case "false":
		return false, nil
	}

	// Default: return the key itself
	return key, nil
}

// ConvertToList converts a string representation of a list to a slice
func ConvertToList(input interface{}) []interface{} {
	// If input is already a slice, return it
	if slice, ok := input.([]interface{}); ok {
		return slice
	}

	// If input is a string, try to parse it as JSON
	if str, ok := input.(string); ok {
		// Check if it looks like a JSON array
		if strings.HasPrefix(str, "[") && strings.HasSuffix(str, "]") {
			var result []interface{}
			if err := json.Unmarshal([]byte(str), &result); err == nil {
				return result
			}
		}

		// If not JSON, split by commas as a fallback
		parts := strings.Split(str, ",")
		result := make([]interface{}, len(parts))
		for i, part := range parts {
			result[i] = strings.TrimSpace(part)
		}
		return result
	}

	// Default: wrap in a slice
	return []interface{}{input}
}

// AggregateTokenUsageFromAgents aggregates token usage from all agents
func AggregateTokenUsageFromAgents(agents map[string]Agent) map[string]interface{} {
	totalPromptTokens := 0
	totalCompletionTokens := 0
	totalTokens := 0

	// In a real implementation, we would iterate through agents and collect token usage
	// This is a placeholder implementation

	return map[string]interface{}{
		"prompt_tokens":     totalPromptTokens,
		"completion_tokens": totalCompletionTokens,
		"total_tokens":      totalTokens,
	}
}

// Helper functions for common operations
func getStringFromMap(m map[string]interface{}, key string) (string, bool) {
	if val, ok := m[key]; ok {
		if str, ok := val.(string); ok {
			return str, true
		}
	}
	return "", false
}

func getMapFromMap(m map[string]interface{}, key string) (map[string]interface{}, bool) {
	if val, ok := m[key]; ok {
		if mapVal, ok := val.(map[string]interface{}); ok {
			return mapVal, true
		}
	}
	return nil, false
}

func getSliceFromMap(m map[string]interface{}, key string) ([]interface{}, bool) {
	if val, ok := m[key]; ok {
		if slice, ok := val.([]interface{}); ok {
			return slice, true
		}
	}
	return nil, false
}

// convertMapToStringMap converts a map[string]interface{} to map[string]string
func convertMapToStringMap(m map[string]interface{}) map[string]string {
	result := make(map[string]string)
	for k, v := range m {
		result[k] = fmt.Sprintf("%v", v)
	}
	return result
}

// Performance optimizations
var stepResultPool = sync.Pool{
	New: func() interface{} {
		return &StepResult{
			Metadata: make(map[string]interface{}),
		}
	},
}

// GetStepResult gets a StepResult from the pool
func GetStepResult() *StepResult {
	return stepResultPool.Get().(*StepResult)
}

// PutStepResult puts a StepResult back in the pool
func PutStepResult(sr *StepResult) {
	sr.Prompt = ""
	sr.Next = ""
	for k := range sr.Metadata {
		delete(sr.Metadata, k)
	}
	stepResultPool.Put(sr)
}

// JoinContextInputs joins multiple inputs with newlines
func JoinContextInputs(inputs []string) string {
	if len(inputs) == 0 {
		return ""
	}
	if len(inputs) == 1 {
		return inputs[0]
	}

	var sb strings.Builder
	for i, input := range inputs {
		if i > 0 {
			sb.WriteString("\n\n")
		}
		sb.WriteString(input)
	}
	return sb.String()
}

// Made with Bob
