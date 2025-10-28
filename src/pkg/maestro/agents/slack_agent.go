// SPDX-License-Identifier: Apache-2.0
// Copyright © 2025 IBM

package agents

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
)

// SlackAgent extends the BaseAgent to post messages to a Slack channel
type SlackAgent struct {
	*BaseAgent
	Channel         string
	postMessageFunc func(channelID, message string) (interface{}, error)
}

// NewSlackAgent creates a new SlackAgent
func NewSlackAgent(agent map[string]interface{}) (interface{}, error) {
	// Create the base agent
	baseAgent, err := NewBaseAgent(agent)
	if err != nil {
		return nil, err
	}

	// Get the channel from environment variable
	channel := os.Getenv("SLACK_TEAM_ID")

	// Create the agent
	slackAgent := &SlackAgent{
		BaseAgent: baseAgent,
		Channel:   channel,
	}

	// Set the postMessageFunc to use the postMessageToSlack method
	slackAgent.postMessageFunc = slackAgent.postMessageToSlack

	return slackAgent, nil
}

// Run implements the Agent interface Run method
func (s *SlackAgent) Run(args ...interface{}) (interface{}, error) {
	if len(args) == 0 {
		return nil, fmt.Errorf("no prompt provided")
	}

	prompt, ok := args[0].(string)
	if !ok {
		return nil, fmt.Errorf("prompt must be a string")
	}

	s.Print(fmt.Sprintf("Running %s...\n", s.AgentName))

	// Post message to Slack using the function pointer
	answer, err := s.postMessageFunc(s.Channel, prompt)
	if err != nil {
		return nil, err
	}

	s.Print(fmt.Sprintf("Response from %s: %v\n", s.AgentName, answer))
	return answer, nil
}

// RunStreaming implements streaming for the SlackAgent
func (s *SlackAgent) RunStreaming(args ...interface{}) (interface{}, error) {
	// For SlackAgent, streaming is the same as regular Run
	return s.Run(args...)
}

// postMessageToSlack posts a message to a Slack channel
func (s *SlackAgent) postMessageToSlack(channelID, message string) (interface{}, error) {
	// Add deprecation notice
	s.Print("⚠️ This agent is deprecated. The posting slack message is supported by slack MCP tool now. " +
		"To use slack mcp tool, refer to mcp/examples/slack")

	// Get token from environment
	slackToken := os.Getenv("SLACK_BOT_TOKEN")
	if slackToken == "" {
		s.Print("Error: SLACK_BOT_TOKEN environment variable not set.")
		return nil, fmt.Errorf("SLACK_BOT_TOKEN environment variable not set")
	}

	// Prepare request payload
	payload := map[string]string{
		"channel": channelID,
		"text":    message,
	}

	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal payload: %w", err)
	}

	// Create request
	req, err := http.NewRequest("POST", "https://slack.com/api/chat.postMessage", bytes.NewBuffer(jsonPayload))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+slackToken)

	// Send request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Parse response
	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// Check if the request was successful
	if ok, exists := result["ok"].(bool); !exists || !ok {
		errorMsg := "unknown error"
		if err, exists := result["error"].(string); exists {
			errorMsg = err
		}
		return nil, fmt.Errorf("slack API error: %s", errorMsg)
	}

	// Return timestamp of the message
	if ts, exists := result["ts"].(string); exists {
		s.Print(fmt.Sprintf("Message posted to channel %s: %s", channelID, ts))
		return ts, nil
	}

	return "Message sent", nil
}

// Made with Bob
