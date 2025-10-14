// SPDX-License-Identifier: Apache-2.0
// Copyright © 2025 IBM

package agents

import (
	"fmt"
	"os/exec"
	"strings"
	"sync"
)

// DSPyAgent extends the BaseAgent to interact with DSPy framework
type DSPyAgent struct {
	*BaseAgent
	ProviderURL string
	ToolNames   []string
	MCPStack    *sync.WaitGroup
}

// NewDSPyAgent creates a new DSPyAgent
func NewDSPyAgent(agent map[string]interface{}) (interface{}, error) {
	// Check if DSPy is installed
	if err := checkDSPyInstalled(); err != nil {
		return nil, fmt.Errorf("cannot initialize DSPyAgent: %w", err)
	}

	// Create the base agent
	baseAgent, err := NewBaseAgent(agent)
	if err != nil {
		return nil, err
	}

	// Extract spec
	spec, ok := agent["spec"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid agent definition: missing spec")
	}

	// Get provider URL
	providerURL, _ := spec["url"].(string)

	// Get tools
	var toolNames []string
	if toolsVal, ok := spec["tools"]; ok {
		if toolsSlice, ok := toolsVal.([]interface{}); ok {
			for _, tool := range toolsSlice {
				if toolStr, ok := tool.(string); ok {
					toolNames = append(toolNames, toolStr)
				}
			}
		}
	}

	return &DSPyAgent{
		BaseAgent:   baseAgent,
		ProviderURL: providerURL,
		ToolNames:   toolNames,
		MCPStack:    &sync.WaitGroup{},
	}, nil
}

// Run implements the Agent interface Run method
func (d *DSPyAgent) Run(args ...interface{}) (interface{}, error) {
	if len(args) == 0 {
		return nil, fmt.Errorf("no prompt provided")
	}

	prompt, ok := args[0].(string)
	if !ok {
		return nil, fmt.Errorf("prompt must be a string")
	}

	d.Print(fmt.Sprintf("Running DSPy agent: %s with prompt: %s", d.AgentName, prompt))

	// Create a Python script that uses DSPy
	pythonScript := fmt.Sprintf(`
import sys
import json
import asyncio
from contextlib import AsyncExitStack

try:
    import dspy
    from maestro.tool_utils import get_mcp_tools

    # Configure DSPy
    dspy.configure(lm=dspy.LM("%s", api_base="%s"))

    # Define signature
    class BaseDSPySignature(dspy.Signature):
        """You are a good agent that helps user answer questions and carries out tasks.

        You are given a list of tools to handle user request, and you should decide the right tool to use in order to
        fullfil users' request."""

        user_request: str = dspy.InputField()
        process_result: str = dspy.OutputField(
            desc=(
                "Message that summarizes the process result, and the final answer to the user questions and requests."
            )
        )

    # Add instructions
    signature = BaseDSPySignature.with_instructions(
        "You are %s\\nYou are expected to do %s"
    )

    async def run_agent():
        mcp_stack = AsyncExitStack()
        try:
            dspy_tools = []
            tool_names = %s
            if tool_names and len(tool_names):
                for tool_name in tool_names:
                    dspy_tools.extend(
                        await get_mcp_tools(
                            tool_name, dspy.Tool.from_mcp_tool, mcp_stack
                        )
                    )
            
            dspy_agent = dspy.ReAct(signature, dspy_tools)
            result = await dspy_agent.acall(user_request="%s")
            
            await mcp_stack.aclose()
            if result and result.process_result:
                print(result.process_result)
                return
            
            print("No response from Agent")
            sys.exit(1)
        except Exception as e:
            print(f"Failed to execute dspy agent: {e}")
            sys.exit(1)

    asyncio.run(run_agent())
except ImportError as e:
    print(json.dumps({"error": "ImportError", "message": str(e)}))
    sys.exit(1)
except Exception as e:
    print(json.dumps({"error": str(type(e).__name__), "message": str(e)}))
    sys.exit(1)
`, d.AgentModel, d.ProviderURL, d.AgentDesc, d.AgentInstr, d.ToolNames, prompt)

	// Execute the Python script
	cmd := exec.Command("python", "-c", pythonScript)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("failed to execute DSPy agent: %w, output: %s", err, string(output))
	}

	result := strings.TrimSpace(string(output))
	d.Print(fmt.Sprintf("Response from %s: %s", d.AgentName, result))
	return result, nil
}

// RunStreaming implements streaming for the DSPyAgent
func (d *DSPyAgent) RunStreaming(args ...interface{}) (interface{}, error) {
	// DSPy doesn't support streaming yet
	return nil, fmt.Errorf("streaming execution for DSPy agent '%s' is not implemented yet", d.AgentName)
}

// checkDSPyInstalled checks if the DSPy library is installed
func checkDSPyInstalled() error {
	cmd := exec.Command("python", "-c", "import dspy")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("DSPy support is disabled because the 'dspy' library could not be imported. To enable, run `pip install dspy`")
	}
	return nil
}

// Made with Bob
