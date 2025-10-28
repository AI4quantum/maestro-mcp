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
//
// Returns:
//   - The docker arguments.
func CreateDockerArgs(cmd string, target string, env string) []string {
	arg := []string{cmd, "run", "-d", "-p", fmt.Sprintf("%s:5000", target)}
	arg = append(arg, EnvArrayDocker(env)...)
	arg = append(arg, "maestro")
	return arg
}

// CreateBuildArgs creates the build arguments for the given command and flags.
// Parameters:
//   - cmd: The command to be executed.
//   - flags: A string of flags to be included in the build arguments.
//
// Returns:
//   - A list of build arguments.
func CreateBuildArgs(cmd string, flags string) []string {
	arg := []string{cmd, "build"}
	if flags != "" {
		arg = append(arg, FlagArrayBuild(flags)...)
	}
	arg = append(arg, "-t", "maestro", "-f", "Dockerfile", "..")
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

// BuildImage builds an image for the Maestro application.
func (d *Deploy) BuildImage(agent string, workflow string) error {
	// Get the module directory
	moduleDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	// Create temporary directory
	d.TmpDir = filepath.Join(os.TempDir(), "maestro")
	if err := os.MkdirAll(d.TmpDir, 0755); err != nil {
		return fmt.Errorf("failed to create temporary directory: %w", err)
	}

	// Copy source files
	srcDir := filepath.Join(moduleDir, "..")
	if err := copyDir(srcDir, d.TmpDir); err != nil {
		return fmt.Errorf("failed to copy source files: %w", err)
	}

	// Copy deployment files
	deploymentsDir := filepath.Join(moduleDir, "deployments")
	tmpDeployDir := filepath.Join(d.TmpDir, "tmp")
	if err := os.MkdirAll(tmpDeployDir, 0755); err != nil {
		return fmt.Errorf("failed to create tmp directory: %w", err)
	}

	if err := copyDir(deploymentsDir, tmpDeployDir); err != nil {
		return fmt.Errorf("failed to copy deployment files: %w", err)
	}

	// Write agent contents to file
	if err := os.WriteFile(filepath.Join(tmpDeployDir, "agents.yaml"), []byte(agent), 0644); err != nil {
		return fmt.Errorf("failed to write agent file: %w", err)
	}

	// Write workflow contents to file
	if err := os.WriteFile(filepath.Join(tmpDeployDir, "workflow.yaml"), []byte(workflow), 0644); err != nil {
		return fmt.Errorf("failed to write workflow file: %w", err)
	}

	// Change to tmp directory and build the image
	currentDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	if err := os.Chdir(tmpDeployDir); err != nil {
		return fmt.Errorf("failed to change directory: %w", err)
	}

	// Build the image
	buildArgs := CreateBuildArgs(d.Cmd, d.Flags)
	cmd := exec.Command(buildArgs[0], buildArgs[1:]...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		// Change back to original directory before returning error
		_ = os.Chdir(currentDir)
		return fmt.Errorf("failed to build image: %w", err)
	}

	// Change back to original directory
	if err := os.Chdir(currentDir); err != nil {
		return fmt.Errorf("failed to change back to original directory: %w", err)
	}

	return nil
}

// DeployToDocker deploys the agent to a Docker container.
func (d *Deploy) DeployToDocker() error {
	// Build the image
	if err := d.BuildImage(d.Agent, d.Workflow); err != nil {
		return err
	}

	// Run the container
	dockerArgs := CreateDockerArgs(d.Cmd, d.Target, d.Env)
	cmd := exec.Command(dockerArgs[0], dockerArgs[1:]...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to run container: %w", err)
	}

	// Clean up temporary directory
	if err := os.RemoveAll(d.TmpDir); err != nil {
		return fmt.Errorf("failed to clean up temporary directory: %w", err)
	}

	return nil
}

// DeployToKubernetes deploys the trained model to Kubernetes.
func (d *Deploy) DeployToKubernetes() error {
	// Build the image
	if err := d.BuildImage(d.Agent, d.Workflow); err != nil {
		return err
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

// Helper functions

// copyFile copies a file from src to dst
func copyFile(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return fmt.Errorf("failed to read source file: %w", err)
	}

	if err := os.WriteFile(dst, data, 0644); err != nil {
		return fmt.Errorf("failed to write destination file: %w", err)
	}

	return nil
}

// copyDir recursively copies a directory from src to dst
func copyDir(src, dst string) error {
	// Get file info
	info, err := os.Stat(src)
	if err != nil {
		return fmt.Errorf("failed to get source directory info: %w", err)
	}

	// Check if it's a directory
	if !info.IsDir() {
		return fmt.Errorf("source is not a directory")
	}

	// Create destination directory
	if err := os.MkdirAll(dst, info.Mode()); err != nil {
		return fmt.Errorf("failed to create destination directory: %w", err)
	}

	// Read directory entries
	entries, err := os.ReadDir(src)
	if err != nil {
		return fmt.Errorf("failed to read source directory: %w", err)
	}

	// Copy each entry
	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())

		if entry.IsDir() {
			// Recursively copy subdirectory
			if err := copyDir(srcPath, dstPath); err != nil {
				return err
			}
		} else {
			// Copy file
			if err := copyFile(srcPath, dstPath); err != nil {
				return err
			}
		}
	}

	return nil
}

// Made with Bob
