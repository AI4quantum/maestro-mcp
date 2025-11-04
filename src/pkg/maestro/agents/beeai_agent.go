// SPDX-License-Identifier: Apache-2.0
// Copyright © 2025 IBM

package agents

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"text/template"
)

// BeeAIAgent extends the BaseAgent to interact with BeeAI framework
type BeeAIAgent struct {
	*BaseAgent
	OutputTemplate *template.Template
	ModelParams    map[string]interface{}
}

// NewBeeAIAgent creates a new BeeAIAgent
func NewBeeAIAgent(agent map[string]interface{}) (interface{}, error) {
	// Check if BeeAI framework is installed
	if err := checkBeeAIInstalled(); err != nil {
		return nil, fmt.Errorf("cannot initialize BeeAIAgent: %w", err)
	}

	// Create the base agent
	baseAgent, err := NewBaseAgent(agent)
	if err != nil {
		return nil, err
	}

	// Create output template
	outputTemplateStr := "{{.result}}"
	if baseAgent.AgentOutput != "" {
		outputTemplateStr = baseAgent.AgentOutput
	}

	outputTemplate, err := template.New("output").Parse(outputTemplateStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse output template: %w", err)
	}

	// Extract spec for model parameters
	spec, ok := agent["spec"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid agent definition: missing spec")
	}

	// Initialize model parameters
	modelParams := initializeModelParameters(spec, baseAgent)
	print(modelParams)

	return &BeeAIAgent{
		BaseAgent:      baseAgent,
		OutputTemplate: outputTemplate,
		ModelParams:    modelParams,
	}, nil
}

// Run implements the Agent interface Run method
func (b *BeeAIAgent) Run(args ...interface{}) (interface{}, error) {
	if len(args) == 0 {
		return nil, fmt.Errorf("no prompt provided")
	}

	prompt, ok := args[0].(string)
	if !ok {
		return nil, fmt.Errorf("prompt must be a string")
	}

	b.Print(fmt.Sprintf("Running %s with prompt...", b.AgentName))

	// Run the BeeAI agent using Python
	result, err := b.runBeeAIAgent(prompt)
	if err != nil {
		return nil, err
	}

	// Track token usage
	b.TrackTokens(prompt, result)

	// Render output template
	var buf bytes.Buffer
	err = b.OutputTemplate.Execute(&buf, map[string]interface{}{
		"result": result,
		"prompt": prompt,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to render output template: %w", err)
	}

	answer := buf.String()
	b.Print(fmt.Sprintf("Response from %s: %s\n", b.AgentName, answer))

	return answer, nil
}

// RunStreaming implements streaming for the BeeAIAgent
func (b *BeeAIAgent) RunStreaming(args ...interface{}) (interface{}, error) {
	// BeeAI streaming is handled the same way as regular run
	// The Python implementation handles streaming internally
	return b.Run(args...)
}

// runBeeAIAgent runs the BeeAI agent using Python
func (b *BeeAIAgent) runBeeAIAgent(prompt string) (string, error) {
	// Escape special characters in strings for Python
	escapedPrompt := strings.ReplaceAll(prompt, `"`, `\"`)
	escapedPrompt = strings.ReplaceAll(escapedPrompt, "\n", "\\n")

	escapedInstr := strings.ReplaceAll(b.AgentInstr, `"`, `\"`)
	escapedInstr = strings.ReplaceAll(escapedInstr, "\n", "\\n")

	escapedModel := b.AgentModel
	escapedURL := b.AgentURL
	if escapedURL == "" {
		escapedURL = "http://localhost:11434" // Default Ollama URL
	}

	escapedCode := ""
	if b.AgentCode != "" {
		escapedCode = strings.ReplaceAll(b.AgentCode, `"`, `\"`)
		escapedCode = strings.ReplaceAll(escapedCode, "\n", "\\n")
	}

	// Build tools list
	toolsList := "[]"
	if len(b.AgentTools) > 0 {
		toolsJSON, _ := json.Marshal(b.AgentTools)
		toolsList = string(toolsJSON)
	}

	// Build model parameters
	modelParamsJSON, _ := json.Marshal(b.ModelParams)
	modelParamsStr := string(modelParamsJSON)

	// Create Python script that uses BeeAI framework
	pythonScript := fmt.Sprintf(`
import sys
import json
import os
import asyncio
import tempfile
from contextlib import AsyncExitStack
from typing import Any, Callable
from pydantic import BaseModel

try:
    from beeai_framework.adapters.ollama import OllamaChatModel
    from beeai_framework.agents.tool_calling import ToolCallingAgent
    from beeai_framework.backend import ChatModel, ChatModelParameters
    from beeai_framework.backend.utils import find_provider_def
    from beeai_framework.tools.code import PythonTool, LocalPythonStorage, SandboxTool
    from beeai_framework.agents import AgentExecutionConfig, AgentMeta
    from beeai_framework.memory import UnconstrainedMemory
    from beeai_framework.template import PromptTemplateInput
    from beeai_framework.tools import AnyTool
    from beeai_framework.tools.mcp import MCPTool
    from beeai_framework.tools.weather import OpenMeteoTool
    from beeai_framework.utils import AbortSignal
    from beeai_framework.emitter import Emitter, EmitterOptions, EventMeta
    from beeai_framework.errors import FrameworkError
except ImportError as e:
    print(json.dumps({"error": "ImportError", "message": str(e)}))
    sys.exit(1)

def user_customizer(config: PromptTemplateInput[Any]) -> PromptTemplateInput[Any]:
    """user_customizer"""
    class UserSchema(BaseModel):
        """user schema"""
        input: str
    
    new_config = config.model_copy()
    new_config.input_schema = UserSchema
    new_config.template = """User: {{input}}"""
    return new_config

def no_result_customizer(config: PromptTemplateInput[Any]) -> PromptTemplateInput[Any]:
    """no_result_customizer"""
    new_config = config.model_copy()
    config.template += """\nPlease reformat your input."""
    return new_config

def not_found_customizer(config: PromptTemplateInput[Any]) -> PromptTemplateInput[Any]:
    """not_found_customizer"""
    class ToolSchema(BaseModel):
        """Tool Schema"""
        name: str
    
    class NotFoundSchema(BaseModel):
        """Not found schema"""
        tools: list[ToolSchema]
    
    new_config = config.model_copy()
    new_config.input_schema = NotFoundSchema
    new_config.template = """Tool does not exist!
{{#tools.length}}
Use one of the following tools: {{#trim}}{{#tools}}{{name}},{{/tools}}{{/trim}}
{{/tools.length}}"""
    return new_config

def user_template_func(template: PromptTemplateInput[Any]) -> PromptTemplateInput[Any]:
    return template.fork(customizer=user_customizer)

def get_system_template_func(instructions: str) -> Callable[[PromptTemplateInput], PromptTemplateInput[Any]]:
    def system_template_func(template: PromptTemplateInput[Any]) -> PromptTemplateInput[Any]:
        return template.update(
            defaults={
                "instructions": instructions or "You are a helpful assistant that uses tools to answer questions."
            }
        )
    return system_template_func

def tool_no_result_error_template_func(template: PromptTemplateInput[Any]) -> PromptTemplateInput[Any]:
    return template.fork(customizer=no_result_customizer)

def tool_not_found_error_template_func(template: PromptTemplateInput[Any]) -> PromptTemplateInput[Any]:
    return template.fork(customizer=not_found_customizer)

def process_agent_events(data, event: EventMeta) -> None:
    """Process agent events and log appropriately"""
    if event.name == "error":
        print(f"Agent 🤖 : {FrameworkError.ensure(data.error).explain()}", file=sys.stderr)
    elif event.name == "retry":
        print("Agent 🤖 : retrying the action...", file=sys.stderr)
    elif event.name == "update":
        print(f"Agent({data.update.key}) 🤖 : {data.update.parsed_value}", file=sys.stderr)
    elif event.name == "start":
        print("Agent 🤖 : starting new iteration", file=sys.stderr)
    elif event.name == "success":
        print("Agent 🤖 : success", file=sys.stderr)

def observer(emitter: Emitter) -> None:
    """Observer for agent events"""
    emitter.on("*", process_agent_events, EmitterOptions(match_nested=False))

async def run_agent():
    try:
        # Parse model parameters
        model_params_dict = json.loads('%s')
        model_params = ChatModelParameters(**model_params_dict)
        
        # Initialize LLM
        model_name = "%s"
        base_url = "%s"
        
        if find_provider_def(model_name.split(":")[0]) is not None:
            llm = ChatModel.from_name(model_name, base_url=base_url, parameters=model_params)
        else:
            llm = OllamaChatModel(model_name, base_url=base_url, parameters=model_params)
        
        # Initialize tools
        tools = []
        tools_list = json.loads('%s')
        embedded_tools = []
        
        # Weather tools
        weather_tools = ["weather", "openmeteo", "openmeteotool"]
        if any(tool.lower() in weather_tools for tool in tools_list):
            tools.append(OpenMeteoTool())
        embedded_tools.extend(weather_tools)
                
        # Code interpreter tools
        code_tools = ["code_interpreter", "code", "pythontool"]
        if any(tool.lower() in code_tools for tool in tools_list):
            tools.append(
                PythonTool(
                    os.getenv("CODE_INTERPRETER_URL", "http://localhost:50081"),
                    LocalPythonStorage(
                        local_working_dir=tempfile.mkdtemp(prefix="code_interpreter_source"),
                        interpreter_working_dir=os.getenv(
                            "CODE_INTERPRETER_TMPDIR", "./tmp/code_interpreter_target"
                        ),
                    ),
                )
            )
        embedded_tools.extend(code_tools)
        
        # Sandbox tool from code
        code = "%s"
        if code:
            sandbox_tool = await SandboxTool.from_source_code(
                url=os.getenv("CODE_INTERPRETER_URL", "http://localhost:50081"),
                source_code=code,
            )
            tools.append(sandbox_tool)
        
        # MCP tools (not yet implemented in Go version)
        # for tool in tools_list:
        #     if tool.lower() not in embedded_tools:
        #         # MCP tool loading would go here
        #         pass
        
        # Create templates
        instructions = "%s"
        templates = {
            "user": user_template_func,
            "system": get_system_template_func(instructions),
            "tool_no_result_error": tool_no_result_error_template_func,
            "tool_not_found_error": tool_not_found_error_template_func,
        }
        
        # Create agent
        agent = ToolCallingAgent(
            llm=llm,
            templates=templates,
            tools=tools,
            memory=UnconstrainedMemory(),
            meta=AgentMeta(
                name="%s",
                description=instructions or "A helpful assistant",
                tools=tools
            ),
        )
        
        # Run agent with observer
        user_prompt = "%s"
        response = await agent.run(agent,
            prompt=user_prompt,
            execution=AgentExecutionConfig(
                max_retries_per_step=3,
                total_max_retries=10,
                max_iterations=20
            ),
            signal=AbortSignal.timeout(2 * 60 * 1000),
        ).observe(observer)
        
        # Return result
        if response.last_message:
            print(response.last_message.text)
        else:
            print("None")

        
    except Exception as e:
        print(json.dumps({"error": str(type(e).__name__), "message": str(e)}))
        sys.exit(1)

# Run the async function
asyncio.run(run_agent())
`, modelParamsStr, escapedModel, escapedURL, toolsList, escapedCode, escapedInstr, b.AgentName, escapedPrompt)

	// Execute the Python script
	cmd := exec.Command("python", "-c", pythonScript)

	// Set environment variables
	cmd.Env = os.Environ()

	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to execute BeeAI agent: %w, output: %s", err, string(output))
	}

	result := strings.TrimSpace(string(output))

	// Check if the output is an error JSON
	var errorResult struct {
		Error   string `json:"error"`
		Message string `json:"message"`
	}
	if json.Unmarshal([]byte(result), &errorResult) == nil && errorResult.Error != "" {
		return "", fmt.Errorf("BeeAI agent error: %s - %s", errorResult.Error, errorResult.Message)
	}

	return result, nil
}

// initializeModelParameters initializes model parameters from agent spec
func initializeModelParameters(spec map[string]interface{}, baseAgent *BaseAgent) map[string]interface{} {
	params := make(map[string]interface{})

	specParams, ok := spec["model_parameters"].(map[string]interface{})
	if !ok {
		return params
	}

	// Temperature
	if temp, ok := specParams["temperature"].(float64); ok {
		if temp >= 0.0 && temp <= 2.0 {
			params["temperature"] = temp
			baseAgent.Print(fmt.Sprintf("INFO [BeeAIAgent %s]: Using temperature: %f", baseAgent.AgentName, temp))
		}
	}

	// Max tokens
	if maxTokens, ok := specParams["max_tokens"].(float64); ok {
		if maxTokens > 0 {
			params["max_tokens"] = int(maxTokens)
			baseAgent.Print(fmt.Sprintf("INFO [BeeAIAgent %s]: Using max_tokens: %d", baseAgent.AgentName, int(maxTokens)))
		}
	}

	// Top P
	if topP, ok := specParams["top_p"].(float64); ok {
		if topP >= 0.0 && topP <= 1.0 {
			params["top_p"] = topP
			baseAgent.Print(fmt.Sprintf("INFO [BeeAIAgent %s]: Using top_p: %f", baseAgent.AgentName, topP))
		}
	}

	// Top K
	if topK, ok := specParams["top_k"].(float64); ok {
		if topK > 0 {
			params["top_k"] = int(topK)
			baseAgent.Print(fmt.Sprintf("INFO [BeeAIAgent %s]: Using top_k: %d", baseAgent.AgentName, int(topK)))
		}
	}

	// Frequency penalty
	if freqPenalty, ok := specParams["frequency_penalty"].(float64); ok {
		if freqPenalty >= -2.0 && freqPenalty <= 2.0 {
			params["frequency_penalty"] = freqPenalty
			baseAgent.Print(fmt.Sprintf("INFO [BeeAIAgent %s]: Using frequency_penalty: %f", baseAgent.AgentName, freqPenalty))
		}
	}

	// Presence penalty
	if presPenalty, ok := specParams["presence_penalty"].(float64); ok {
		if presPenalty >= -2.0 && presPenalty <= 2.0 {
			params["presence_penalty"] = presPenalty
			baseAgent.Print(fmt.Sprintf("INFO [BeeAIAgent %s]: Using presence_penalty: %f", baseAgent.AgentName, presPenalty))
		}
	}

	// Stop sequences
	if stopSeqs, ok := specParams["stop_sequences"].([]interface{}); ok {
		stopStrings := make([]string, 0, len(stopSeqs))
		for _, seq := range stopSeqs {
			if str, ok := seq.(string); ok {
				stopStrings = append(stopStrings, str)
			}
		}
		if len(stopStrings) > 0 {
			params["stop"] = stopStrings
			baseAgent.Print(fmt.Sprintf("INFO [BeeAIAgent %s]: Using stop_sequences: %v", baseAgent.AgentName, stopStrings))
		}
	}

	return params
}

// checkBeeAIInstalled checks if the BeeAI framework is installed
func checkBeeAIInstalled() error {
	cmd := exec.Command("python", "-c", "import beeai_framework")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("BeeAI support is disabled because the 'beeai_framework' library could not be imported. To enable, run `pip install beeai-framework`")
	}
	return nil
}

// Made with Bob
