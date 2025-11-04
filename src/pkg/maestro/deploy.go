// SPDX-License-Identifier: Apache-2.0
// Copyright © 2025 IBM

package maestro

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

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
	// Create ConfigMap with agents and workflow YAML
	if err := CreateConfigMap(d.Agent, d.Workflow); err != nil {
		return fmt.Errorf("failed to create ConfigMap: %w", err)
	}

	// Create temporary directory for deployment
	d.TmpDir = filepath.Join(os.TempDir(), "maestro")
	if err := os.MkdirAll(d.TmpDir, 0755); err != nil {
		return fmt.Errorf("failed to create temporary directory: %w", err)
	}

	// Deployment template - using raw string to avoid any hidden characters
	deploymentTemplate := "apiVersion: apps/v1\n" +
		"kind: Deployment\n" +
		"metadata:\n" +
		"  name: maestro\n" +
		"spec:\n" +
		"  replicas: 1\n" +
		"  selector:\n" +
		"    matchLabels:\n" +
		"      app: maestro\n" +
		"  template:\n" +
		"    metadata:\n" +
		"      labels:\n" +
		"        app: maestro\n" +
		"    spec:\n" +
		"      containers:\n" +
		"      - name: maestro\n" +
		"        image: maestro-api:latest\n" +
		"        imagePullPolicy: Never\n" +
		"        ports:\n" +
		"        - containerPort: 8080\n" +
		"        env:\n" +
		"        - name: DUMMY\n" +
		"          value: dummyvalue\n" +
		"        volumeMounts:\n" +
		"        - name: maestro-config\n" +
		"          mountPath: /app/src\n" +
		"      volumes:\n" +
		"      - name: maestro-config\n" +
		"        configMap:\n" +
		"          name: maestrodata"

	// Parse the deployment template
	var deploymentData map[string]interface{}
	if err := yaml.Unmarshal([]byte(deploymentTemplate), &deploymentData); err != nil {
		return fmt.Errorf("failed to parse deployment template: %w", err)
	}

	// Add environment variables
	if d.Env != "" {
		spec := deploymentData["spec"].(map[string]interface{})
		template := spec["template"].(map[string]interface{})
		templateSpec := template["spec"].(map[string]interface{})
		containers := templateSpec["containers"].([]interface{})
		container := containers[0].(map[string]interface{})

		// Get or create env array
		var env []interface{}
		if existingEnv, ok := container["env"].([]interface{}); ok {
			env = existingEnv
		} else {
			env = []interface{}{}
		}

		// Add environment variables
		pairs := strings.Fields(d.Env)
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
	}

	// Marshal the updated deployment YAML
	updatedDeploymentYAML, err := yaml.Marshal(deploymentData)
	if err != nil {
		return fmt.Errorf("failed to marshal deployment YAML: %w", err)
	}

	// Create a temporary file for the deployment YAML
	deploymentFile, err := os.CreateTemp(d.TmpDir, "deployment-*.yaml")
	if err != nil {
		return fmt.Errorf("failed to create temporary deployment file: %w", err)
	}
	defer os.Remove(deploymentFile.Name())

	// Write the deployment YAML to the temporary file
	if _, err := deploymentFile.Write(updatedDeploymentYAML); err != nil {
		return fmt.Errorf("failed to write deployment YAML: %w", err)
	}
	if err := deploymentFile.Close(); err != nil {
		return fmt.Errorf("failed to close deployment file: %w", err)
	}

	// Apply deployment
	deployCmd := exec.Command("kubectl", "apply", "-f", deploymentFile.Name())
	deployCmd.Stdout = os.Stdout
	deployCmd.Stderr = os.Stderr

	if err := deployCmd.Run(); err != nil {
		return fmt.Errorf("failed to apply deployment: %w", err)
	}

	// Service template - using raw string to avoid any hidden characters
	serviceTemplate := "apiVersion: v1\n" +
		"kind: Service\n" +
		"metadata:\n" +
		"  name: maestro\n" +
		"spec:\n" +
		"  selector:\n" +
		"    app: maestro\n" +
		"  ports:\n" +
		"    - protocol: TCP\n" +
		"      port: 80\n" +
		"      targetPort: 8080\n" +
		"      nodePort: 30051\n" +
		"  type: NodePort"

	// Create a temporary file for the service YAML
	serviceFile, err := os.CreateTemp(d.TmpDir, "service-*.yaml")
	if err != nil {
		return fmt.Errorf("failed to create temporary service file: %w", err)
	}
	defer os.Remove(serviceFile.Name())

	// Write the service YAML to the temporary file
	if _, err := serviceFile.WriteString(serviceTemplate); err != nil {
		return fmt.Errorf("failed to write service YAML: %w", err)
	}
	if err := serviceFile.Close(); err != nil {
		return fmt.Errorf("failed to close service file: %w", err)
	}

	// Apply service
	serviceCmd := exec.Command("kubectl", "apply", "-f", serviceFile.Name())
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

// CreateConfigMap creates a Kubernetes ConfigMap with the given agents and workflow YAML content
// and applies it to the Kubernetes cluster.
// Parameters:
//   - agentsYAML: The content of the agents.yaml file.
//   - workflowYAML: The content of the workflow.yaml file.
//
// Returns:
//   - error if any
func CreateConfigMap(agentsYAML string, workflowYAML string) error {
	// Create a temporary directory for the ConfigMap YAML
	tmpDir := filepath.Join(os.TempDir(), "maestro-configmap")
	if err := os.MkdirAll(tmpDir, 0755); err != nil {
		return fmt.Errorf("failed to create temporary directory: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create the ConfigMap structure
	configMap := map[string]interface{}{
		"apiVersion": "v1",
		"kind":       "ConfigMap",
		"metadata": map[string]interface{}{
			"name": "maestrodata",
		},
		"data": map[string]interface{}{
			"agents.yaml":   agentsYAML,
			"workflow.yaml": workflowYAML,
		},
	}

	// Marshal the ConfigMap to YAML
	configMapYAML, err := yaml.Marshal(configMap)
	if err != nil {
		return fmt.Errorf("failed to marshal ConfigMap to YAML: %w", err)
	}

	// Write the ConfigMap YAML to a temporary file
	configMapFile := filepath.Join(tmpDir, "configmap.yaml")
	if err := os.WriteFile(configMapFile, configMapYAML, 0644); err != nil {
		return fmt.Errorf("failed to write ConfigMap YAML file: %w", err)
	}

	// Apply the ConfigMap using kubectl
	cmd := exec.Command("kubectl", "apply", "-f", configMapFile)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to apply ConfigMap: %w", err)
	}

	return nil
}

// DeployDefault deploys the workflow with API server and UI (default deployment method).
// This is similar to the Python __deploy_agents_workflow_node implementation.
func (d *Deploy) DeployDefault() error {
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

	// Get API host and port from environment or use defaults
	apiHost := os.Getenv("MAESTRO_API_HOST")
	if apiHost == "" {
		apiHost = "127.0.0.1"
	}

	apiPortStr := os.Getenv("MAESTRO_API_PORT")
	if apiPortStr == "" {
		apiPortStr = "8000"
	}

	uiPort := os.Getenv("MAESTRO_UI_PORT")
	if uiPort == "" {
		uiPort = "5173"
	}

	// Convert port string to int
	apiPort := 8000
	if port, err := fmt.Sscanf(apiPortStr, "%d", &apiPort); err != nil || port != 1 {
		d.Logger.Warn("Invalid API port, using default 8000",
			zap.String("provided_port", apiPortStr))
		apiPort = 8000
	}

	// Set CORS environment variable
	corsOrigins := os.Getenv("CORS_ALLOW_ORIGINS")
	if corsOrigins == "" {
		corsOrigins = fmt.Sprintf("http://%s:%s", apiHost, uiPort)
		os.Setenv("CORS_ALLOW_ORIGINS", corsOrigins)
	}

	d.Logger.Info("Starting default deployment",
		zap.String("api_host", apiHost),
		zap.Int("api_port", apiPort),
		zap.String("ui_port", uiPort),
		zap.String("agents_file", agentsFile),
		zap.String("workflow_file", workflowFile))

	// Start the API server using ServeWorkflow in a goroutine
	apiErrChan := make(chan error, 1)
	go func() {
		if err := ServeWorkflow(agentsFile, workflowFile, apiHost, apiPort); err != nil {
			d.Logger.Error("Failed to start API server",
				zap.Error(err))
			apiErrChan <- err
		}
	}()

	// Wait for API server to be healthy
	if err := waitForAPIHealth(apiHost, apiPort, 60, 1, apiErrChan, d.Logger); err != nil {
		d.Logger.Error("Failed to start API server - health check failed",
			zap.Error(err))
		return fmt.Errorf("API server failed to become healthy: %w", err)
	}

	// Start the UI server
	if err := startUIServer(uiPort, d.Logger); err != nil {
		d.Logger.Error("Failed to start UI server",
			zap.Error(err))
		return fmt.Errorf("failed to start UI server: %w", err)
	}

	d.Logger.Info("Default deployment started successfully",
		zap.String("api_url", fmt.Sprintf("http://%s:%d", apiHost, apiPort)),
		zap.String("ui_url", fmt.Sprintf("http://localhost:%s", uiPort)))

	return nil
}

// waitForAPIHealth waits for the API server to be healthy by polling the /health endpoint.
// Parameters:
//   - host: API server host
//   - port: API server port
//   - timeout: Maximum time to wait in seconds
//   - checkInterval: Time between health checks in seconds
//   - apiErrChan: Channel to receive errors from the API server goroutine
//   - logger: Logger instance
//
// Returns:
//   - error: nil if API is healthy, error if startup failed or timeout reached
func waitForAPIHealth(host string, port int, timeout int, checkInterval int, apiErrChan <-chan error, logger *zap.Logger) error {
	url := fmt.Sprintf("http://%s:%d/health", host, port)
	startTime := time.Now()
	timeoutDuration := time.Duration(timeout) * time.Second
	ticker := time.NewTicker(time.Duration(checkInterval) * time.Second)
	defer ticker.Stop()

	logger.Info("Waiting for API server to be ready", zap.String("url", url))

	for {
		select {
		case err := <-apiErrChan:
			// API server failed to start
			logger.Error("API server startup failed", zap.Error(err))
			return fmt.Errorf("API server startup failed: %w", err)

		case <-ticker.C:
			// Check if timeout has been reached
			if time.Since(startTime) >= timeoutDuration {
				logger.Error("API server failed to become ready", zap.Int("timeout_seconds", timeout))
				return fmt.Errorf("API server health check timeout after %d seconds", timeout)
			}

			// Try to check health endpoint
			resp, err := http.Get(url)
			if err == nil {
				defer resp.Body.Close()
				if resp.StatusCode == 200 {
					var healthData map[string]interface{}
					if err := json.NewDecoder(resp.Body).Decode(&healthData); err == nil {
						if status, ok := healthData["status"].(string); ok && status == "healthy" {
							logger.Info("API server is ready!")
							return nil
						}
					}
				}
			}
		}
	}
}

// startUIServer starts the UI server using npm.
// Parameters:
//   - uiPort: Port for the UI server
//   - logger: Logger instance
//
// Returns:
//   - error if any
func startUIServer(uiPort string, logger *zap.Logger) error {
	// Get the project root directory
	// Assuming the current working directory structure
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current working directory: %w", err)
	}

	// Try to find the UI directory
	// Look for src/maestro/ui or ui directory
	uiPaths := []string{
		filepath.Join(cwd, "src", "maestro", "ui"),
		filepath.Join(cwd, "ui"),
		filepath.Join(cwd, "..", "maestro", "src", "maestro", "ui"),
	}

	var uiCwd string
	for _, path := range uiPaths {
		if _, err := os.Stat(path); err == nil {
			uiCwd = path
			break
		}
	}

	if uiCwd == "" {
		logger.Warn("UI directory not found, skipping UI server startup")
		return nil
	}

	// Set up environment for UI server
	uiEnv := os.Environ()
	uiEnv = append(uiEnv, fmt.Sprintf("PORT=%s", uiPort))

	// Start the UI server
	npmCmd := exec.Command("npm", "run", "dev")
	npmCmd.Dir = uiCwd
	npmCmd.Env = uiEnv
	// Suppress stderr to avoid cluttering logs
	npmCmd.Stderr = nil

	if err := npmCmd.Start(); err != nil {
		return fmt.Errorf("failed to start UI server: %w", err)
	}

	logger.Info("UI server started",
		zap.String("ui_cwd", uiCwd),
		zap.String("port", uiPort))

	return nil
}

// Made with Bob
