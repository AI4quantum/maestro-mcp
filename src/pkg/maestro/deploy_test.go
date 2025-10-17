// SPDX-License-Identifier: Apache-2.0
// Copyright © 2025 IBM

package maestro

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
)

func TestEnvArrayDocker(t *testing.T) {
	tests := []struct {
		name    string
		strEnvs string
		want    []string
	}{
		{
			name:    "Empty string",
			strEnvs: "",
			want:    []string{},
		},
		{
			name:    "Single environment variable",
			strEnvs: "KEY=value",
			want:    []string{"-e", "KEY=value"},
		},
		{
			name:    "Multiple environment variables",
			strEnvs: "KEY1=value1 KEY2=value2 KEY3=value3",
			want:    []string{"-e", "KEY1=value1", "-e", "KEY2=value2", "-e", "KEY3=value3"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := EnvArrayDocker(tt.strEnvs)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("EnvArrayDocker() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFlagArrayBuild(t *testing.T) {
	tests := []struct {
		name     string
		strFlags string
		want     []string
	}{
		{
			name:     "Empty string",
			strFlags: "",
			want:     []string{},
		},
		{
			name:     "Single flag",
			strFlags: "key=value",
			want:     []string{"key", "value"},
		},
		{
			name:     "Multiple flags",
			strFlags: "key1=value1 key2=value2 key3=value3",
			want:     []string{"key1", "value1", "key2", "value2", "key3", "value3"},
		},
		{
			name:     "Flag without value",
			strFlags: "key",
			want:     []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FlagArrayBuild(tt.strFlags)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("FlagArrayBuild() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCreateDockerArgs(t *testing.T) {
	tests := []struct {
		name   string
		cmd    string
		target string
		env    string
		want   []string
	}{
		{
			name:   "Basic docker command",
			cmd:    "docker",
			target: "8080",
			env:    "",
			want:   []string{"docker", "run", "-d", "-p", "8080:5000", "maestro"},
		},
		{
			name:   "With environment variables",
			cmd:    "docker",
			target: "8080",
			env:    "KEY1=value1 KEY2=value2",
			want:   []string{"docker", "run", "-d", "-p", "8080:5000", "-e", "KEY1=value1", "-e", "KEY2=value2", "maestro"},
		},
		{
			name:   "With podman",
			cmd:    "podman",
			target: "9000",
			env:    "DEBUG=true",
			want:   []string{"podman", "run", "-d", "-p", "9000:5000", "-e", "DEBUG=true", "maestro"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CreateDockerArgs(tt.cmd, tt.target, tt.env)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("CreateDockerArgs() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCreateBuildArgs(t *testing.T) {
	tests := []struct {
		name  string
		cmd   string
		flags string
		want  []string
	}{
		{
			name:  "Basic build command",
			cmd:   "docker",
			flags: "",
			want:  []string{"docker", "build", "-t", "maestro", "-f", "Dockerfile", ".."},
		},
		{
			name:  "With build flags",
			cmd:   "docker",
			flags: "no-cache=true pull=true",
			want:  []string{"docker", "build", "no-cache", "true", "pull", "true", "-t", "maestro", "-f", "Dockerfile", ".."},
		},
		{
			name:  "With podman",
			cmd:   "podman",
			flags: "force-rm=true",
			want:  []string{"podman", "build", "force-rm", "true", "-t", "maestro", "-f", "Dockerfile", ".."},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CreateBuildArgs(tt.cmd, tt.flags)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("CreateBuildArgs() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestUpdateYAML(t *testing.T) {
	// Create a temporary directory for the test
	tempDir, err := os.MkdirTemp("", "deploy_test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a test YAML file
	yamlContent := `
spec:
  template:
    spec:
      containers:
      - name: test-container
        image: test-image:latest
        env:
        - name: EXISTING_VAR
          value: existing_value
`
	yamlFile := filepath.Join(tempDir, "deployment.yaml")
	if err := os.WriteFile(yamlFile, []byte(yamlContent), 0644); err != nil {
		t.Fatalf("Failed to write YAML file: %v", err)
	}

	// Test updating the YAML
	err = UpdateYAML(yamlFile, "NEW_VAR1=new_value1 NEW_VAR2=new_value2")
	if err != nil {
		t.Fatalf("UpdateYAML failed: %v", err)
	}

	// Read the updated YAML
	data, err := os.ReadFile(yamlFile)
	if err != nil {
		t.Fatalf("Failed to read updated YAML: %v", err)
	}

	// Parse the YAML
	var yamlData map[string]interface{}
	if err := yaml.Unmarshal(data, &yamlData); err != nil {
		t.Fatalf("Failed to parse YAML: %v", err)
	}

	// Verify the environment variables
	spec := yamlData["spec"].(map[string]interface{})
	template := spec["template"].(map[string]interface{})
	templateSpec := template["spec"].(map[string]interface{})
	containers := templateSpec["containers"].([]interface{})
	container := containers[0].(map[string]interface{})
	env := container["env"].([]interface{})

	// Should have 3 environment variables (1 existing + 2 new)
	if len(env) != 3 {
		t.Errorf("Expected 3 environment variables, got %d", len(env))
	}

	// Check if the new variables were added
	foundNew1 := false
	foundNew2 := false
	foundExisting := false

	for _, e := range env {
		envVar := e.(map[string]interface{})
		name := envVar["name"].(string)
		value := envVar["value"].(string)

		switch name {
		case "EXISTING_VAR":
			foundExisting = true
			if value != "existing_value" {
				t.Errorf("Expected EXISTING_VAR=existing_value, got %s", value)
			}
		case "NEW_VAR1":
			foundNew1 = true
			if value != "new_value1" {
				t.Errorf("Expected NEW_VAR1=new_value1, got %s", value)
			}
		case "NEW_VAR2":
			foundNew2 = true
			if value != "new_value2" {
				t.Errorf("Expected NEW_VAR2=new_value2, got %s", value)
			}
		}
	}

	if !foundExisting {
		t.Error("Existing environment variable not found")
	}
	if !foundNew1 {
		t.Error("NEW_VAR1 not found")
	}
	if !foundNew2 {
		t.Error("NEW_VAR2 not found")
	}
}

func TestNewDeploy(t *testing.T) {
	// Setup logger
	logger, _ := zap.NewDevelopment()

	// Test with default values
	deploy := NewDeploy("agent.yaml", "workflow.yaml", "", "", logger)
	if deploy.Agent != "agent.yaml" {
		t.Errorf("Expected Agent to be 'agent.yaml', got '%s'", deploy.Agent)
	}
	if deploy.Workflow != "workflow.yaml" {
		t.Errorf("Expected Workflow to be 'workflow.yaml', got '%s'", deploy.Workflow)
	}
	if deploy.Target != "127.0.0.1:5000" {
		t.Errorf("Expected Target to be '127.0.0.1:5000', got '%s'", deploy.Target)
	}
	if deploy.Cmd != "docker" {
		t.Errorf("Expected Cmd to be 'docker', got '%s'", deploy.Cmd)
	}

	// Test with custom values
	deploy = NewDeploy("custom-agent.yaml", "custom-workflow.yaml", "ENV=value", "8080", logger)
	if deploy.Agent != "custom-agent.yaml" {
		t.Errorf("Expected Agent to be 'custom-agent.yaml', got '%s'", deploy.Agent)
	}
	if deploy.Workflow != "custom-workflow.yaml" {
		t.Errorf("Expected Workflow to be 'custom-workflow.yaml', got '%s'", deploy.Workflow)
	}
	if deploy.Env != "ENV=value" {
		t.Errorf("Expected Env to be 'ENV=value', got '%s'", deploy.Env)
	}
	if deploy.Target != "8080" {
		t.Errorf("Expected Target to be '8080', got '%s'", deploy.Target)
	}

	// Test with environment variable
	os.Setenv("CONTAINER_CMD", "podman")
	defer os.Unsetenv("CONTAINER_CMD")
	deploy = NewDeploy("agent.yaml", "workflow.yaml", "", "", logger)
	if deploy.Cmd != "podman" {
		t.Errorf("Expected Cmd to be 'podman', got '%s'", deploy.Cmd)
	}
}

func TestCopyFile(t *testing.T) {
	// Create a temporary directory for the test
	tempDir, err := os.MkdirTemp("", "deploy_test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a source file
	srcContent := "test content"
	srcFile := filepath.Join(tempDir, "source.txt")
	if err := os.WriteFile(srcFile, []byte(srcContent), 0644); err != nil {
		t.Fatalf("Failed to write source file: %v", err)
	}

	// Copy the file
	dstFile := filepath.Join(tempDir, "destination.txt")
	if err := copyFile(srcFile, dstFile); err != nil {
		t.Fatalf("copyFile failed: %v", err)
	}

	// Verify the destination file
	dstContent, err := os.ReadFile(dstFile)
	if err != nil {
		t.Fatalf("Failed to read destination file: %v", err)
	}

	if string(dstContent) != srcContent {
		t.Errorf("Expected content '%s', got '%s'", srcContent, string(dstContent))
	}
}

func TestCopyDir(t *testing.T) {
	// Create a temporary directory for the test
	tempDir, err := os.MkdirTemp("", "deploy_test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a source directory structure
	srcDir := filepath.Join(tempDir, "src")
	if err := os.MkdirAll(srcDir, 0755); err != nil {
		t.Fatalf("Failed to create source directory: %v", err)
	}

	// Create a subdirectory
	subDir := filepath.Join(srcDir, "subdir")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatalf("Failed to create subdirectory: %v", err)
	}

	// Create files in the source directory
	if err := os.WriteFile(filepath.Join(srcDir, "file1.txt"), []byte("file1 content"), 0644); err != nil {
		t.Fatalf("Failed to write file1: %v", err)
	}
	if err := os.WriteFile(filepath.Join(subDir, "file2.txt"), []byte("file2 content"), 0644); err != nil {
		t.Fatalf("Failed to write file2: %v", err)
	}

	// Copy the directory
	dstDir := filepath.Join(tempDir, "dst")
	if err := copyDir(srcDir, dstDir); err != nil {
		t.Fatalf("copyDir failed: %v", err)
	}

	// Verify the destination directory structure
	if _, err := os.Stat(dstDir); os.IsNotExist(err) {
		t.Errorf("Destination directory not created")
	}
	if _, err := os.Stat(filepath.Join(dstDir, "file1.txt")); os.IsNotExist(err) {
		t.Errorf("file1.txt not copied")
	}
	if _, err := os.Stat(filepath.Join(dstDir, "subdir")); os.IsNotExist(err) {
		t.Errorf("subdir not copied")
	}
	if _, err := os.Stat(filepath.Join(dstDir, "subdir", "file2.txt")); os.IsNotExist(err) {
		t.Errorf("file2.txt not copied")
	}

	// Verify file contents
	content1, err := os.ReadFile(filepath.Join(dstDir, "file1.txt"))
	if err != nil {
		t.Fatalf("Failed to read file1.txt: %v", err)
	}
	if string(content1) != "file1 content" {
		t.Errorf("Expected file1.txt content 'file1 content', got '%s'", string(content1))
	}

	content2, err := os.ReadFile(filepath.Join(dstDir, "subdir", "file2.txt"))
	if err != nil {
		t.Fatalf("Failed to read file2.txt: %v", err)
	}
	if string(content2) != "file2 content" {
		t.Errorf("Expected file2.txt content 'file2 content', got '%s'", string(content2))
	}
}

// Note: We're not testing BuildImage, DeployToDocker, and DeployToKubernetes
// directly because they interact with external systems (Docker, Kubernetes).
// In a real-world scenario, these would be tested with mocks or in an integration test.

// Made with Bob
