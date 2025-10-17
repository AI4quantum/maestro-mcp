// SPDX-License-Identifier: Apache-2.0
// Copyright © 2025 IBM

package maestro

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
)

// Step represents a step in a workflow
type Step struct {
	Name      string                   // Name of the step
	Agent     Agent                    // Agent to run for this step
	Workflow  string                   // URL of workflow to run
	Input     map[string]interface{}   // Input configuration
	Condition []map[string]interface{} // Conditional branches
	Parallel  []Agent                  // Agents to run in parallel
	Loop      map[string]interface{}   // Loop configuration
}

// NewStep creates a new Step instance
func NewStep(stepDef map[string]interface{}) (*Step, error) {
	name, ok := stepDef["name"].(string)
	if !ok {
		return nil, fmt.Errorf("step definition missing required 'name' field")
	}

	step := &Step{
		Name: name,
	}

	// Set agent if present
	if agent, ok := stepDef["agent"]; ok {
		if agentObj, ok := agent.(Agent); ok {
			step.Agent = agentObj
		}
	}

	// Set workflow if present
	if workflow, ok := stepDef["workflow"].(string); ok {
		step.Workflow = workflow
	}

	// Set input if present
	if input, ok := stepDef["input"].(map[string]interface{}); ok {
		step.Input = input
	}

	// Set condition if present
	if condition, ok := stepDef["condition"].([]map[string]interface{}); ok {
		step.Condition = condition
	} else if conditionList, ok := stepDef["condition"].([]interface{}); ok {
		// Convert []interface{} to []map[string]interface{}
		step.Condition = make([]map[string]interface{}, len(conditionList))
		for i, c := range conditionList {
			if cMap, ok := c.(map[string]interface{}); ok {
				step.Condition[i] = cMap
			}
		}
	}

	// Set parallel if present
	if parallel, ok := stepDef["parallel"].([]Agent); ok {
		step.Parallel = parallel
	}

	// Set loop if present
	if loop, ok := stepDef["loop"].(map[string]interface{}); ok {
		step.Loop = loop
	}

	return step, nil
}

// Run executes the step with the given input
func (s *Step) Run(ctx context.Context, input interface{}, stepIndex int) (*StepResult, error) {
	var result *StepResult
	var err error

	// Convert input to string if it's not already
	inputStr := ""
	if str, ok := input.(string); ok {
		inputStr = str
	} else {
		// Try to convert to string
		inputStr = fmt.Sprintf("%v", input)
	}

	// Run the appropriate action based on step type
	if s.Agent != nil {
		// Run agent
		agentResult, err := s.Agent.Run(inputStr)
		if err != nil {
			return nil, fmt.Errorf("agent execution failed: %w", err)
		}

		// Process agent result
		result, err = s.processResult(agentResult)
		if err != nil {
			return nil, err
		}
	} else if s.Workflow != "" {
		// Run workflow
		result, err = s.runWorkflow(ctx, inputStr)
		if err != nil {
			return nil, err
		}
	} else {
		// No agent or workflow, just pass through the input
		result = &StepResult{
			Prompt: inputStr,
		}
	}

	// Apply input template if present
	if s.Input != nil {
		prompt, err := s.applyInput(result.Prompt)
		if err != nil {
			return nil, err
		}
		result.Prompt = prompt
	}

	// Evaluate condition if present
	if s.Condition != nil {
		next, err := s.evaluateCondition(result.Prompt)
		if err != nil {
			return nil, err
		}
		result.Next = next
	}

	// Run parallel agents if present
	if s.Parallel != nil {
		parallelResult, err := s.runParallel(ctx, result.Prompt, stepIndex)
		if err != nil {
			return nil, err
		}
		result.Prompt = parallelResult
	}

	// Run loop if present
	if s.Loop != nil {
		loopResult, err := s.runLoop(ctx, result.Prompt, stepIndex)
		if err != nil {
			return nil, err
		}
		result.Prompt = loopResult
	}

	// Strip think tags from the result
	result.Prompt = StripThinkTags(result.Prompt)

	return result, nil
}

// processResult processes the result from an agent or workflow
func (s *Step) processResult(result interface{}) (*StepResult, error) {
	// If result is already a StepResult, return it
	if sr, ok := result.(*StepResult); ok {
		return sr, nil
	}

	// If result is a map, extract prompt and next
	if resultMap, ok := result.(map[string]interface{}); ok {
		prompt := ""
		if p, ok := resultMap["prompt"]; ok {
			prompt = fmt.Sprintf("%v", p)
		}

		next := ""
		if n, ok := resultMap["next"]; ok {
			next = fmt.Sprintf("%v", n)
		}

		return &StepResult{
			Prompt:   prompt,
			Next:     next,
			Metadata: resultMap,
		}, nil
	}

	// Default: treat result as the prompt
	return &StepResult{
		Prompt: fmt.Sprintf("%v", result),
	}, nil
}

// runWorkflow runs an external workflow via HTTP
func (s *Step) runWorkflow(ctx context.Context, input string) (*StepResult, error) {
	// Create request body
	reqBody := map[string]interface{}{
		"prompt": input,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request body: %w", err)
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "POST", s.Workflow+"/chat", strings.NewReader(string(jsonData)))
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	// Send request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("workflow request failed with status %d", resp.StatusCode)
	}

	// Parse response
	var responseData map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&responseData); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Extract message from response
	message, ok := responseData["response"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid response format")
	}

	return &StepResult{
		Prompt: message,
	}, nil
}

// evaluateCondition evaluates the condition and returns the next step
func (s *Step) evaluateCondition(prompt string) (string, error) {
	if len(s.Condition) == 0 {
		return "", nil
	}

	// Check if this is an if/then/else condition
	if ifExpr, ok := s.Condition[0]["if"]; ok {
		return s.processIfCondition(ifExpr.(string), prompt)
	}

	// Otherwise, process as case/do conditions
	return s.processCaseCondition(prompt)
}

// processIfCondition processes an if/then/else condition
func (s *Step) processIfCondition(expr string, prompt string) (string, error) {
	result, err := EvalExpression(expr, prompt)
	if err != nil {
		return "", fmt.Errorf("failed to evaluate condition: %w", err)
	}

	if result {
		if then, ok := s.Condition[0]["then"].(string); ok {
			return then, nil
		}
	} else {
		if elseStep, ok := s.Condition[0]["else"].(string); ok {
			return elseStep, nil
		}
	}

	return "", nil
}

// processCaseCondition processes case/do conditions
func (s *Step) processCaseCondition(prompt string) (string, error) {
	defaultStep := ""

	for _, cond := range s.Condition {
		if expr, ok := cond["case"].(string); ok {
			result, err := EvalExpression(expr, prompt)
			if err != nil {
				return "", fmt.Errorf("failed to evaluate case condition: %w", err)
			}

			if result {
				if doStep, ok := cond["do"].(string); ok {
					return doStep, nil
				}
			}
		}

		// Store default step if present
		if doStep, ok := cond["do"].(string); ok {
			defaultStep = doStep
		}
	}

	return defaultStep, nil
}

// applyInput applies the input template to the prompt
func (s *Step) applyInput(prompt string) (string, error) {
	if s.Input == nil {
		return prompt, nil
	}

	// Get template and user prompt
	template, ok := s.Input["template"].(string)
	if !ok {
		return "", fmt.Errorf("input template not found")
	}

	userPrompt, ok := s.Input["prompt"].(string)
	if !ok {
		return "", fmt.Errorf("input prompt not found")
	}

	// Special connector handling
	if strings.Contains(template, "{CONNECTOR}") {
		return prompt, nil
	}

	// Replace {prompt} in user prompt
	userPrompt = strings.ReplaceAll(userPrompt, "{prompt}", prompt)

	// Get user input
	var response string
	fmt.Print(userPrompt)
	if _, err := fmt.Scanln(&response); err != nil {
		// If there's an error reading input, use an empty response
		response = ""
	}

	// Apply template
	result := strings.ReplaceAll(template, "{prompt}", prompt)
	result = strings.ReplaceAll(result, "{response}", response)

	return result, nil
}

// runParallel runs multiple agents in parallel
func (s *Step) runParallel(ctx context.Context, prompt string, stepIndex int) (string, error) {
	if len(s.Parallel) == 0 {
		return prompt, nil
	}

	// Check if prompt is a list
	var inputs []interface{}
	if strings.HasPrefix(prompt, "[") {
		inputs = ConvertToList(prompt)
	} else {
		// Use the same prompt for all agents
		inputs = make([]interface{}, len(s.Parallel))
		for i := range s.Parallel {
			inputs[i] = prompt
		}
	}

	// Create a wait group to synchronize goroutines
	var wg sync.WaitGroup
	results := make([]interface{}, len(s.Parallel))
	errors := make([]error, len(s.Parallel))

	// Run each agent in a goroutine
	for i, agent := range s.Parallel {
		wg.Add(1)
		go func(idx int, a Agent, input interface{}) {
			defer wg.Done()
			result, err := a.Run(input)
			results[idx] = result
			errors[idx] = err
		}(i, agent, inputs[i%len(inputs)])
	}

	// Wait for all goroutines to complete
	wg.Wait()

	// Check for errors
	for _, err := range errors {
		if err != nil {
			return "", fmt.Errorf("parallel execution failed: %w", err)
		}
	}

	// Convert results to string
	resultStr, err := json.Marshal(results)
	if err != nil {
		return "", fmt.Errorf("failed to marshal parallel results: %w", err)
	}

	return string(resultStr), nil
}

// runLoop runs a loop until the condition is met
func (s *Step) runLoop(ctx context.Context, prompt string, stepIndex int) (string, error) {
	if s.Loop == nil {
		return prompt, nil
	}

	// Get loop agent
	agent, ok := s.Loop["agent"].(Agent)
	if !ok {
		return "", fmt.Errorf("loop agent not found")
	}

	// Get until expression
	until, ok := s.Loop["until"].(string)
	if !ok {
		return "", fmt.Errorf("loop until condition not found")
	}

	// Check if prompt is a list
	if strings.HasPrefix(prompt, "[") {
		inputs := ConvertToList(prompt)
		results := make([]interface{}, len(inputs))

		for i, input := range inputs {
			result, err := agent.Run(input)
			if err != nil {
				return "", fmt.Errorf("loop execution failed: %w", err)
			}
			results[i] = result
		}

		resultStr, err := json.Marshal(results)
		if err != nil {
			return "", fmt.Errorf("failed to marshal loop results: %w", err)
		}

		return string(resultStr), nil
	}

	// Run loop until condition is met
	currentPrompt := prompt
	for {
		result, err := agent.Run(currentPrompt)
		if err != nil {
			return "", fmt.Errorf("loop execution failed: %w", err)
		}

		// Convert result to string
		resultStr := ""
		if str, ok := result.(string); ok {
			resultStr = str
		} else {
			resultStr = fmt.Sprintf("%v", result)
		}

		currentPrompt = resultStr

		// Check if condition is met
		conditionMet, err := EvalExpression(until, currentPrompt)
		if err != nil {
			return "", fmt.Errorf("failed to evaluate loop condition: %w", err)
		}

		if conditionMet {
			break
		}
	}

	return currentPrompt, nil
}

// Made with Bob
