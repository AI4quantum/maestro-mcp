// SPDX-License-Identifier: Apache-2.0
// Copyright © 2025 IBM

package agents

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"text/template"
)

// RemoteAgent extends the BaseAgent to run agents via HTTP requests
type RemoteAgent struct {
	*BaseAgent
	URL              string
	RequestTemplate  string
	ResponseTemplate string
}

// NewRemoteAgent creates a new RemoteAgent
func NewRemoteAgent(agent map[string]interface{}) (interface{}, error) {
	// Create the base agent
	baseAgent, err := NewBaseAgent(agent)
	if err != nil {
		return nil, err
	}

	// Extract URL and templates from spec
	spec, ok := agent["spec"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid agent definition: missing spec")
	}

	url, ok := spec["url"].(string)
	if !ok || url == "" {
		return nil, fmt.Errorf("invalid agent definition: missing or empty URL")
	}

	requestTemplate, _ := spec["request_template"].(string)
	responseTemplate, _ := spec["response_template"].(string)

	return &RemoteAgent{
		BaseAgent:        baseAgent,
		URL:              url,
		RequestTemplate:  requestTemplate,
		ResponseTemplate: responseTemplate,
	}, nil
}

// Run implements the Agent interface Run method
func (r *RemoteAgent) Run(args ...interface{}) (interface{}, error) {
	if len(args) == 0 {
		return nil, fmt.Errorf("no prompt provided")
	}

	prompt, ok := args[0].(string)
	if !ok {
		return nil, fmt.Errorf("prompt must be a string")
	}

	r.Print(fmt.Sprintf("Running %s...\n", r.AgentName))

	// Prepare request data
	var requestData map[string]interface{}
	if r.RequestTemplate != "" {
		// Parse the template
		tmpl, err := template.New("request").Parse(r.RequestTemplate)
		if err != nil {
			return nil, fmt.Errorf("failed to parse request template: %w", err)
		}

		// Execute the template
		var buf bytes.Buffer
		err = tmpl.Execute(&buf, map[string]interface{}{
			"prompt": prompt,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to execute request template: %w", err)
		}

		// Parse the JSON
		if err := json.Unmarshal(buf.Bytes(), &requestData); err != nil {
			return nil, fmt.Errorf("failed to parse request JSON: %w", err)
		}
	} else {
		// Default request data
		requestData = map[string]interface{}{
			"prompt": prompt,
		}
	}

	// Print the prompt
	r.Print(fmt.Sprintf("❓ %s", prompt))

	// Send the request
	requestBody, err := json.Marshal(requestData)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request data: %w", err)
	}

	resp, err := http.Post(r.URL, "application/json", bytes.NewBuffer(requestBody))
	if err != nil {
		r.Print(fmt.Sprintf("An error occurred: %v", err))
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("request failed with status code %d", resp.StatusCode)
	}

	// Read response body
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Parse response JSON
	var responseData interface{}
	if err := json.Unmarshal(responseBody, &responseData); err != nil {
		return nil, fmt.Errorf("failed to parse response JSON: %w", err)
	}

	// Process response using template if provided
	var answer interface{}
	if r.ResponseTemplate != "" {
		// Parse the template
		tmpl, err := template.New("response").Parse(r.ResponseTemplate)
		if err != nil {
			return nil, fmt.Errorf("failed to parse response template: %w", err)
		}

		// Execute the template
		var buf bytes.Buffer
		err = tmpl.Execute(&buf, map[string]interface{}{
			"response": responseData,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to execute response template: %w", err)
		}

		// The result is the template output
		answer = strings.TrimSpace(buf.String())
	} else {
		// Default to the raw response data
		answer = responseData
	}

	// Print the answer
	r.Print(fmt.Sprintf("🤖 %v", answer))

	return answer, nil
}

// Made with Bob
