// SPDX-License-Identifier: Apache-2.0
// Copyright © 2025 IBM

package agents

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// CodeAgent extends the BaseAgent to execute arbitrary code specified in the agent definition
type CodeAgent struct {
	*BaseAgent
	venvPath string // Path to virtual environment
}

// NewCodeAgent creates a new CodeAgent
func NewCodeAgent(agent map[string]interface{}) (interface{}, error) {
	baseAgent, err := NewBaseAgent(agent)
	if err != nil {
		return nil, err
	}

	return &CodeAgent{
		BaseAgent: baseAgent,
		venvPath:  "",
	}, nil
}

// createVirtualEnv creates a virtual environment for installing dependencies
func (c *CodeAgent) createVirtualEnv() error {
	// Create a virtual environment in a temporary directory
	tempDir := os.TempDir()
	c.venvPath = filepath.Join(tempDir, fmt.Sprintf("venv-%s-%d", c.AgentName, os.Getpid()))
	c.Print(fmt.Sprintf("Creating virtual environment at %s", c.venvPath))

	// Use the Python venv module to create a virtual environment
	cmd := exec.Command(pythonExecutable(), "-m", "venv", c.venvPath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		errorMsg := fmt.Sprintf("Error creating virtual environment: %s", string(output))
		c.Print(errorMsg)
		c.venvPath = ""
		return fmt.Errorf("%s: %w", errorMsg, err)
	}

	c.Print("Virtual environment created successfully.")
	return nil
}

// removeVirtualEnv removes the virtual environment if it exists
func (c *CodeAgent) removeVirtualEnv() {
	if c.venvPath != "" && dirExists(c.venvPath) {
		c.Print(fmt.Sprintf("Removing virtual environment at %s", c.venvPath))
		err := os.RemoveAll(c.venvPath)
		if err != nil {
			c.Print(fmt.Sprintf("Warning: Failed to remove virtual environment %s: %v", c.venvPath, err))
		} else {
			c.Print("Virtual environment removed successfully.")
		}
		c.venvPath = ""
	}
}

// installDependencies checks if the agent has dependencies in its metadata and installs them
func (c *CodeAgent) installDependencies(agentDef map[string]interface{}) error {
	// Create virtual environment
	if err := c.createVirtualEnv(); err != nil {
		return err
	}

	// Check for dependencies in metadata
	metadata, ok := agentDef["metadata"].(map[string]interface{})
	if !ok {
		c.Print("No metadata found")
		return nil
	}

	// Print the metadata for debugging
	c.Print(fmt.Sprintf("Metadata: %v", metadata))

	dependencies, ok := metadata["dependencies"].(string)
	if !ok || strings.TrimSpace(dependencies) == "" {
		c.Print("No dependencies found")
		return nil
	}

	c.Print(fmt.Sprintf("Dependencies: %s", dependencies))

	c.Print(fmt.Sprintf("Installing dependencies for %s...", c.AgentName))

	// Create a temporary requirements.txt file
	tempFile, err := os.CreateTemp("", "requirements-*.txt")
	if err != nil {
		return fmt.Errorf("failed to create temporary file: %w", err)
	}
	tempFilePath := tempFile.Name()
	defer os.Remove(tempFilePath)

	// Write dependencies to the temporary file
	sourceFile, _ := agentDef["source_file"].(string)
	content := getContent(dependencies, sourceFile)
	if _, err := tempFile.WriteString(content); err != nil {
		return fmt.Errorf("failed to write to temporary file: %w", err)
	}
	tempFile.Close()

	// Determine pip path in the virtual environment
	var pipPath string
	if runtime.GOOS == "windows" {
		pipPath = filepath.Join(c.venvPath, "Scripts", "pip.exe")
	} else {
		pipPath = filepath.Join(c.venvPath, "bin", "pip")
	}

	// Install dependencies using pip
	c.Print(fmt.Sprintf("Running pip install with requirements file: %s", tempFilePath))
	cmd := exec.Command(pipPath, "install", "-r", tempFilePath, "--verbose")
	output, err := cmd.CombinedOutput()
	if err != nil {
		errorMsg := fmt.Sprintf("Error installing dependencies: %s", string(output))
		c.Print(errorMsg)

		// Provide more helpful error messages for common issues
		outputStr := string(output)
		if strings.Contains(outputStr, "No matching distribution found") {
			c.Print("Suggestion: Check if the package names and versions are correct.")
		} else if strings.Contains(outputStr, "FileNotFoundError") {
			c.Print("Error: pip command not found. Please ensure pip is installed and in your PATH.")
		} else if strings.Contains(outputStr, "Could not find a version that satisfies the requirement") {
			c.Print("Suggestion: The specified package version might not be available. Try using a different version.")
		} else if strings.Contains(outputStr, "HTTP error") || strings.Contains(outputStr, "Connection error") {
			c.Print("Suggestion: Check your internet connection or try again later.")
		}

		return fmt.Errorf("failed to install dependencies: %w", err)
	}

	c.Print("Dependencies installed successfully in virtual environment.")
	c.Print(fmt.Sprintf("Installation output: %s", string(output)))
	return nil
}

// Run implements the Agent interface Run method
func (c *CodeAgent) Run(args ...interface{}) (interface{}, error) {
	if len(args) == 0 {
		return nil, fmt.Errorf("no arguments provided")
	}

	c.Print(fmt.Sprintf("Running %s with %v...\n", c.AgentName, args))

	// Get the agent definition from the BaseAgent
	agentDef := map[string]interface{}{
		"metadata": map[string]interface{}{
			"name": c.AgentName,
		},
		"spec": map[string]interface{}{
			"framework": c.AgentFramework,
			"model":     c.AgentModel,
			"code":      c.AgentCode,
		},
	}

	// Install dependencies
	if err := c.installDependencies(agentDef); err != nil {
		return nil, err
	}

	// Ensure cleanup on exit
	defer c.removeVirtualEnv()

	// Determine Python interpreter path in the virtual environment
	var pythonPath string
	if runtime.GOOS == "windows" {
		pythonPath = filepath.Join(c.venvPath, "Scripts", "python.exe")
	} else {
		pythonPath = filepath.Join(c.venvPath, "bin", "python")
	}

	// Escape the agent code for safe inclusion in a string
	escapedCode := strings.ReplaceAll(c.AgentCode, "\\", "\\\\")
	escapedCode = strings.ReplaceAll(escapedCode, "\"", "\\\"")
	escapedCode = strings.ReplaceAll(escapedCode, "'", "\\'")

	// Create the Python command
	argsJSON, err := json.Marshal(args)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal arguments: %w", err)
	}

	pythonCommand := fmt.Sprintf(`
import json, sys
input = %s
output = {}
exec('''%s''')
print(json.dumps(output))
`, string(argsJSON), escapedCode)

	// Execute the command using the Python interpreter from the virtual environment
	c.Print(fmt.Sprintf("Executing agent code in virtual environment at %s", c.venvPath))
	cmd := exec.Command(pythonPath, "-c", pythonCommand)
	output, err := cmd.CombinedOutput()
	if err != nil {
		c.Print(fmt.Sprintf("Exception executing code in virtual environment: %v\n", err))
		c.Print(fmt.Sprintf("Process output: %s", string(output)))

		// Check if the error is related to missing modules/imports
		outputStr := string(output)
		if strings.Contains(outputStr, "ModuleNotFoundError") ||
			strings.Contains(outputStr, "ImportError") ||
			strings.Contains(outputStr, "No module named") {
			return nil, fmt.Errorf("failed to execute agent code in virtual environment: %s", outputStr)
		}

		return nil, fmt.Errorf("failed to execute agent code in virtual environment: %w", err)
	}

	// Parse the output from stdout
	var outputData interface{}
	outputStr := strings.TrimSpace(string(output))
	if err := json.Unmarshal([]byte(outputStr), &outputData); err != nil {
		c.Print(fmt.Sprintf("JSON decode error: %v. Raw output: %s", err, outputStr))
		outputData = outputStr
	}

	answer := fmt.Sprintf("%v", outputData)
	c.Print(fmt.Sprintf("Response from %s: %s\n", c.AgentName, answer))
	return outputData, nil
}

// Helper function to get the Python executable path
func pythonExecutable() string {
	// Try to use the PYTHON_EXECUTABLE environment variable if set
	if pythonPath := os.Getenv("PYTHON_EXECUTABLE"); pythonPath != "" {
		return pythonPath
	}

	// Default to "python" or "python3" depending on the platform
	if runtime.GOOS == "windows" {
		return "python"
	}

	// Check if python3 exists
	if _, err := exec.LookPath("python3"); err == nil {
		return "python3"
	}

	// Fall back to python
	return "python"
}

// Helper function to check if a directory exists
func dirExists(path string) bool {
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		return false
	}
	return info.IsDir()
}

// Made with Bob
