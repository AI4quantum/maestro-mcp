// SPDX-License-Identifier: Apache-2.0
// Copyright © 2025 IBM

package maestro

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
)

// EnvArrayDocker converts a string of environment variables into an array of arguments for Docker.
// Parameters:
//   - strEnvs: A string of environment variables separated by spaces.
//
// Returns:
//   - A list of arguments for Docker, where each environment variable is represented by two elements in the list: -e and the environment variable name and value.
func EnvArrayDocker(strEnvs string) []string {
	envArray := strings.Fields(strEnvs)
	envArgs := []string{}
	for _, env := range envArray {
		envArgs = append(envArgs, "-e")
		envArgs = append(envArgs, env)
	}
	return envArgs
}

// FlagArrayBuild builds an array of flags from a string of flags.
// Parameters:
//   - strFlags: A string of flags in the format "key1=value1 key2=value2".
//
// Returns:
//   - A list of flags in the format ["key1", "value1", "key2", "value2"].
func FlagArrayBuild(strFlags string) []string {
	flagArray := strings.Fields(strFlags)
	flags := []string{}
	for _, flag := range flagArray {
		parts := strings.SplitN(flag, "=", 2)
		if len(parts) == 2 {
			flags = append(flags, parts[0])
			flags = append(flags, parts[1])
		}
	}
	return flags
}

// CreateDockerArgs creates docker arguments for running a container.
// Parameters:
//   - cmd: The command to run.
//   - target: The target port.
//   - env: The environment variables.
//   - tmpDir: The temporary directory to mount to /app/src in the container.
//
// Returns:
//   - The docker arguments.
func CreateDockerArgs(cmd string, target string, env string, tmpDir string) []string {
	arg := []string{cmd, "run", "-d", "-p", fmt.Sprintf("%s:8080", "5050")}
	// Add volume mount for the temporary directory
	arg = append(arg, "-v", fmt.Sprintf("%s:/app/src", tmpDir))
	arg = append(arg, EnvArrayDocker(env)...)
	arg = append(arg, "maestro-api")
	return arg
}

// UpdateYAML updates the yaml file with the given environment variables.
// Parameters:
//   - yamlFile: The path to the yaml file.
//   - strEnvs: A string of environment variables in the format of "key1=value1 key2=value2".
//
// Returns:
//   - error if any
func UpdateYAML(yamlFile string, strEnvs string) error {
	// Read the YAML file
	data, err := os.ReadFile(yamlFile)
	if err != nil {
		return fmt.Errorf("failed to read YAML file: %w", err)
	}

	// Parse the YAML
	var yamlData map[string]interface{}
	if err := yaml.Unmarshal(data, &yamlData); err != nil {
		return fmt.Errorf("failed to parse YAML: %w", err)
	}

	// Get the container env array
	spec, ok := yamlData["spec"].(map[string]interface{})
	if !ok {
		return fmt.Errorf("invalid YAML: missing spec")
	}

	template, ok := spec["template"].(map[string]interface{})
	if !ok {
		return fmt.Errorf("invalid YAML: missing template")
	}

	templateSpec, ok := template["spec"].(map[string]interface{})
	if !ok {
		return fmt.Errorf("invalid YAML: missing template.spec")
	}

	containers, ok := templateSpec["containers"].([]interface{})
	if !ok || len(containers) == 0 {
		return fmt.Errorf("invalid YAML: missing or empty containers")
	}

	container, ok := containers[0].(map[string]interface{})
	if !ok {
		return fmt.Errorf("invalid YAML: invalid container")
	}

	// Get or create env array
	var env []interface{}
	if existingEnv, ok := container["env"].([]interface{}); ok {
		env = existingEnv
	} else {
		env = []interface{}{}
	}

	// Add environment variables
	pairs := strings.Fields(strEnvs)
	for _, pair := range pairs {
		parts := strings.SplitN(pair, "=", 2)
		if len(parts) == 2 {
			env = append(env, map[string]interface{}{
				"name":  parts[0],
				"value": parts[1],
			})
		}
	}

	// Update the env array
	container["env"] = env

	// Write the updated YAML back to the file
	updatedData, err := yaml.Marshal(yamlData)
	if err != nil {
		return fmt.Errorf("failed to marshal YAML: %w", err)
	}

	if err := os.WriteFile(yamlFile, updatedData, 0644); err != nil {
		return fmt.Errorf("failed to write YAML file: %w", err)
	}

	return nil
}

// Deploy struct for deploying agents and workflows to different environments.
type Deploy struct {
	Agent    string
	Workflow string
	Env      string
	Target   string
	Cmd      string
	Flags    string
	TmpDir   string
	Logger   *zap.Logger
}

// NewDeploy creates a new Deploy instance.
func NewDeploy(agentDefs string, workflowDefs string, env string, target string, logger *zap.Logger) *Deploy {
	if target == "" {
		target = "127.0.0.1:5000"
	}

	cmd := os.Getenv("CONTAINER_CMD")
	if cmd == "" {
		cmd = "docker"
	}

	return &Deploy{
		Agent:    agentDefs,
		Workflow: workflowDefs,
		Env:      env,
		Target:   target,
		Cmd:      cmd,
		Flags:    os.Getenv("BUILD_FLAGS"),
		Logger:   logger,
	}
}

// DeployToDocker deploys the agent to a Docker container.
func (d *Deploy) DeployToDocker() error {
	// Create temporary directory for deployment
	d.TmpDir = filepath.Join(os.TempDir(), "maestro")
	if err := os.MkdirAll(d.TmpDir, 0755); err != nil {
		return fmt.Errorf("failed to create temporary directory: %w", err)
	}

	// Create src directory in the temporary directory
	srcDir := filepath.Join(d.TmpDir, "src")
	if err := os.MkdirAll(srcDir, 0755); err != nil {
		return fmt.Errorf("failed to create src directory: %w", err)
	}

	// Write agent contents to file
	agentsFile := filepath.Join(srcDir, "agents.yaml")
	if err := os.WriteFile(agentsFile, []byte(d.Agent), 0644); err != nil {
		return fmt.Errorf("failed to write agents file: %w", err)
	}

	// Write workflow contents to file
	workflowFile := filepath.Join(srcDir, "workflow.yaml")
	if err := os.WriteFile(workflowFile, []byte(d.Workflow), 0644); err != nil {
		return fmt.Errorf("failed to write workflow file: %w", err)
	}

	// Run the container
	dockerArgs := CreateDockerArgs(d.Cmd, d.Target, d.Env, srcDir)
	cmd := exec.Command(dockerArgs[0], dockerArgs[1:]...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to run container: %w", err)
	}

	// Clean up temporary directory
	//if err := os.RemoveAll(d.TmpDir); err != nil {
	//	return fmt.Errorf("failed to clean up temporary directory: %w", err)
	//}

	return nil
}

// DeployToKubernetes deploys the trained model to Kubernetes.
func (d *Deploy) DeployToKubernetes() error {
	// Create temporary directory for deployment
	d.TmpDir = filepath.Join(os.TempDir(), "maestro")
	if err := os.MkdirAll(d.TmpDir, 0755); err != nil {
		return fmt.Errorf("failed to create temporary directory: %w", err)
	}

	// Update deployment YAML with environment variables
	if err := UpdateYAML(filepath.Join(d.TmpDir, "tmp/deployment.yaml"), d.Env); err != nil {
		return fmt.Errorf("failed to update deployment YAML: %w", err)
	}

	// Tag the image if IMAGE_TAG_CMD is set
	imageTagCmd := os.Getenv("IMAGE_TAG_CMD")
	if imageTagCmd != "" {
		cmd := exec.Command("sh", "-c", imageTagCmd)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to tag image: %w", err)
		}
	}

	// Push the image if IMAGE_PUSH_CMD is set
	imagePushCmd := os.Getenv("IMAGE_PUSH_CMD")
	if imagePushCmd != "" {
		cmd := exec.Command("sh", "-c", imagePushCmd)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to push image: %w", err)
		}
	}

	// Apply deployment
	deployCmd := exec.Command("kubectl", "apply", "-f", filepath.Join(d.TmpDir, "tmp/deployment.yaml"))
	deployCmd.Stdout = os.Stdout
	deployCmd.Stderr = os.Stderr

	if err := deployCmd.Run(); err != nil {
		return fmt.Errorf("failed to apply deployment: %w", err)
	}

	// Apply service
	serviceCmd := exec.Command("kubectl", "apply", "-f", filepath.Join(d.TmpDir, "tmp/service.yaml"))
	serviceCmd.Stdout = os.Stdout
	serviceCmd.Stderr = os.Stderr

	if err := serviceCmd.Run(); err != nil {
		return fmt.Errorf("failed to apply service: %w", err)
	}

	// Clean up temporary directory
	if err := os.RemoveAll(d.TmpDir); err != nil {
		return fmt.Errorf("failed to clean up temporary directory: %w", err)
	}

	return nil
}

// Made with Bob
