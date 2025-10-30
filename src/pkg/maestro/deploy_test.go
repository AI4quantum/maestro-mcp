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
		tmpDir string
		want   []string
	}{
		{
			name:   "Basic docker command",
			cmd:    "docker",
			target: "8080",
			env:    "",
			tmpDir: "/tmp/test",
			want:   []string{"docker", "run", "-d", "-p", "8080:5000", "-v", "/tmp/test:/app/src", "maestro"},
		},
		{
			name:   "With environment variables",
			cmd:    "docker",
			target: "8080",
			env:    "KEY1=value1 KEY2=value2",
			tmpDir: "/tmp/test2",
			want:   []string{"docker", "run", "-d", "-p", "8080:5000", "-v", "/tmp/test2:/app/src", "-e", "KEY1=value1", "-e", "KEY2=value2", "maestro"},
		},
		{
			name:   "With podman",
			cmd:    "podman",
			target: "9000",
			env:    "DEBUG=true",
			tmpDir: "/tmp/test3",
			want:   []string{"podman", "run", "-d", "-p", "9000:5000", "-v", "/tmp/test3:/app/src", "-e", "DEBUG=true", "maestro"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CreateDockerArgs(tt.cmd, tt.target, tt.env, tt.tmpDir)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("CreateDockerArgs() = %v, want %v", got, tt.want)
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

// Note: We're not testing BuildImage, DeployToDocker, and DeployToKubernetes
// directly because they interact with external systems (Docker, Kubernetes).
// In a real-world scenario, these would be tested with mocks or in an integration test.

// TestCreateConfigMap tests the CreateConfigMap function.
// Note: This is a mock test that doesn't actually apply the ConfigMap to a Kubernetes cluster.
func TestCreateConfigMap(t *testing.T) {
	// Skip this test if we're not in a test environment with kubectl
	if os.Getenv("TEST_WITH_KUBECTL") != "true" {
		t.Skip("Skipping test that requires kubectl")
	}

	// Test data
	agentsYAML := `agents:
  - name: test-agent
    type: test`
	workflowYAML := `workflow:
  name: test-workflow
  steps:
    - name: test-step`

	// Call the function
	err := CreateConfigMap(agentsYAML, workflowYAML)
	if err != nil {
		t.Fatalf("CreateConfigMap failed: %v", err)
	}

	// Note: In a real test, we would verify that the ConfigMap was created correctly
	// by querying the Kubernetes API, but that's beyond the scope of this test.
}

// Made with Bob
