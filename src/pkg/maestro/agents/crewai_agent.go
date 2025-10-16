// SPDX-License-Identifier: Apache-2.0
// Copyright © 2025 IBM

package agents

import (
	"fmt"
	"os/exec"
	"strings"
)

// CrewAIAgent extends the BaseAgent to interact with CrewAI framework
type CrewAIAgent struct {
	*BaseAgent
	ModuleName         string
	ClassName          string
	FactoryName        string
	ProviderURL        string
	CrewRole           string
	CrewGoal           string
	CrewBackstory      string
	CrewDescription    string
	CrewExpectedOutput string
}

// NewCrewAIAgent creates a new CrewAIAgent
func NewCrewAIAgent(agent map[string]interface{}) (interface{}, error) {
	// Check if CrewAI is installed
	if err := checkCrewAIInstalled(); err != nil {
		return nil, fmt.Errorf("cannot initialize CrewAIAgent: %w", err)
	}

	// Create the base agent
	baseAgent, err := NewBaseAgent(agent)
	if err != nil {
		return nil, err
	}

	// Extract metadata
	metadata, ok := agent["metadata"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid agent definition: missing metadata")
	}

	// Extract labels
	labels, ok := metadata["labels"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid agent definition: missing labels in metadata")
	}

	// Extract spec
	spec, ok := agent["spec"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid agent definition: missing spec")
	}

	// Create CrewAI agent
	crewAgent := &CrewAIAgent{
		BaseAgent: baseAgent,
	}

	// Check if using module or direct configuration
	if moduleName, ok := labels["module"].(string); ok && moduleName != "" {
		// Using module configuration
		crewAgent.ModuleName = moduleName

		className, ok := labels["class"].(string)
		if !ok {
			return nil, fmt.Errorf("invalid agent definition: missing class in labels")
		}
		crewAgent.ClassName = className

		factoryName, ok := labels["factory"].(string)
		if !ok {
			return nil, fmt.Errorf("invalid agent definition: missing factory in labels")
		}
		crewAgent.FactoryName = factoryName
	} else {
		// Using direct configuration
		if url, ok := spec["url"].(string); ok {
			crewAgent.ProviderURL = url
		}
		if role, ok := labels["crew_role"].(string); ok {
			crewAgent.CrewRole = role
		}
		if goal, ok := labels["crew_goal"].(string); ok {
			crewAgent.CrewGoal = goal
		}
		if backstory, ok := labels["crew_backstory"].(string); ok {
			crewAgent.CrewBackstory = backstory
		}
		if description, ok := labels["crew_description"].(string); ok {
			crewAgent.CrewDescription = description
		}
		if expectedOutput, ok := labels["crew_expected_output"].(string); ok {
			crewAgent.CrewExpectedOutput = expectedOutput
		}

		// Validate required fields
		if crewAgent.ProviderURL == "" ||
			crewAgent.AgentModel == "" ||
			crewAgent.CrewRole == "" ||
			crewAgent.CrewGoal == "" ||
			crewAgent.CrewDescription == "" ||
			crewAgent.CrewExpectedOutput == "" {
			return nil, fmt.Errorf("missing required configuration for direct CrewAI agent definition")
		}
	}

	return crewAgent, nil
}

// Run implements the Agent interface Run method
func (c *CrewAIAgent) Run(args ...interface{}) (interface{}, error) {
	if len(args) == 0 {
		return nil, fmt.Errorf("no prompt provided")
	}

	prompt, ok := args[0].(string)
	if !ok {
		return nil, fmt.Errorf("prompt must be a string")
	}

	c.Print(fmt.Sprintf("Running CrewAI agent: %s with prompt: %s", c.AgentName, prompt))

	var result string
	var err error

	if c.ModuleName != "" {
		// Using module configuration
		result, err = c.runWithModule(prompt)
	} else {
		// Using direct configuration
		result, err = c.runWithDirectConfig(prompt)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to run CrewAI agent: %w", err)
	}

	c.Print(fmt.Sprintf("Response from %s: %s", c.AgentName, result))
	return result, nil
}

// RunStreaming implements streaming for the CrewAIAgent
func (c *CrewAIAgent) RunStreaming(args ...interface{}) (interface{}, error) {
	// CrewAI doesn't support streaming yet
	return nil, fmt.Errorf("streaming execution for CrewAI agent '%s' is not implemented yet", c.AgentName)
}

// runWithModule runs the agent using the specified Python module
func (c *CrewAIAgent) runWithModule(prompt string) (string, error) {
	// Create a Python script that imports the module and calls the factory method
	pythonScript := fmt.Sprintf(`
import sys
import json
try:
    import %s
    instance = %s.%s()
    factory = getattr(instance, "%s")
    result = factory().kickoff({"prompt": %q})
    # Handle different result types
    if hasattr(result, "raw"):
        print(result.raw)
    else:
        print(str(result))
except ImportError as e:
    print(json.dumps({"error": "ImportError", "message": str(e)}))
    sys.exit(1)
except Exception as e:
    print(json.dumps({"error": str(type(e).__name__), "message": str(e)}))
    sys.exit(1)
`, c.ModuleName, c.ModuleName, c.ClassName, c.FactoryName, prompt)

	// Execute the Python script
	cmd := exec.Command("python", "-c", pythonScript)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to execute Python module: %w, output: %s", err, string(output))
	}

	return strings.TrimSpace(string(output)), nil
}

// runWithDirectConfig runs the agent using direct configuration
func (c *CrewAIAgent) runWithDirectConfig(prompt string) (string, error) {
	// Create a Python script that creates a CrewAI agent and runs it
	pythonScript := fmt.Sprintf(`
import sys
import json
try:
    from crewai import Agent as CrewAI_Agent, Crew, Task, Process
    from crewai import LLM

    # Create LLM
    llm = LLM(
        model="%s",
        base_url="%s",
    )

    # Create agent
    agent = CrewAI_Agent(
        role="%s",
        goal="%s",
        backstory="%s",
        llm=llm,
        verbose=False,
        allow_delegation=False,
    )

    # Create task
    task = Task(
        description="%s",
        expected_output="%s",
        agent=agent,
    )

    # Create crew
    crew = Crew(
        agents=[agent],
        tasks=[task],
        process=Process.sequential,
        verbose=False,
    )

    # Run crew
    result = crew.kickoff({"prompt": %q})
    
    # Handle different result types
    if hasattr(result, "raw"):
        print(result.raw)
    else:
        print(str(result))
except ImportError as e:
    print(json.dumps({"error": "ImportError", "message": str(e)}))
    sys.exit(1)
except Exception as e:
    print(json.dumps({"error": str(type(e).__name__), "message": str(e)}))
    sys.exit(1)
`, c.AgentModel, c.ProviderURL, c.CrewRole, c.CrewGoal, c.CrewBackstory,
		c.CrewDescription, c.CrewExpectedOutput, prompt)

	// Execute the Python script
	cmd := exec.Command("python", "-c", pythonScript)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to execute CrewAI: %w, output: %s", err, string(output))
	}

	return strings.TrimSpace(string(output)), nil
}

// checkCrewAIInstalled checks if the CrewAI library is installed
func checkCrewAIInstalled() error {
	cmd := exec.Command("python", "-c", "import crewai")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("CrewAI support is disabled because the 'crewai' library could not be imported. To enable, run `pip install crewai`")
	}
	return nil
}

// Made with Bob
