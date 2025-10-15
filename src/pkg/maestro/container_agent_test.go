// SPDX-License-Identifier: Apache-2.0
// Copyright © 2025 IBM

package maestro

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

// TestCreateContaineredAgent tests the CreateContaineredAgent function
func TestCreateContaineredAgent(t *testing.T) {
	// Save the original function
	originalFunc := CreateDeploymentService

	// Create a temporary directory for the test
	tempDir, err := os.MkdirTemp("", "container-agent-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a test agent YAML file
	agentYAML := `
- apiVersion: maestro/v1alpha1
  kind: Agent
  metadata:
    name: test-agent
    labels:
      app: test-example
  spec:
    model: gpt-4
    framework: openai
    mode: local
    description: Test agent
    image: test-image:latest
    instructions: This is a test agent
`
	agentFile := filepath.Join(tempDir, "agents.yaml")
	err = os.WriteFile(agentFile, []byte(agentYAML), 0644)
	require.NoError(t, err)

	// Create a logger for testing
	logger, _ := zap.NewDevelopment()

	// Create variables to capture function call parameters
	var capturedImage, capturedName, capturedNamespace string
	var capturedReplicas, capturedContainerPort, capturedServicePort, capturedNodePort int32
	var capturedServiceType string

	// Mock the CreateDeploymentService function by replacing it with a test double
	// We'll use a package-level variable to hold our mock function
	mockDeploymentServiceFunc = func(
		imageURL string,
		appName string,
		namespace string,
		replicas int32,
		containerPort int32,
		servicePort int32,
		serviceType string,
		nodePort int32,
		logger *zap.Logger,
	) error {
		capturedImage = imageURL
		capturedName = appName
		capturedNamespace = namespace
		capturedReplicas = replicas
		capturedContainerPort = containerPort
		capturedServicePort = servicePort
		capturedServiceType = serviceType
		capturedNodePort = nodePort
		return nil
	}

	// Restore the original function after the test
	defer func() {
		mockDeploymentServiceFunc = originalFunc
	}()

	// Test with valid agent name
	err = CreateContaineredAgent(agentFile, "test-agent", "localhost", 8080, logger)
	assert.NoError(t, err)
	assert.Equal(t, "test-image:latest", capturedImage)
	assert.Equal(t, "test-agent", capturedName)
	assert.Equal(t, "default", capturedNamespace)
	assert.Equal(t, int32(1), capturedReplicas)
	assert.Equal(t, int32(8080), capturedContainerPort)
	assert.Equal(t, int32(8080), capturedServicePort)
	assert.Equal(t, "LoadBalancer", capturedServiceType)
	assert.Equal(t, int32(30051), capturedNodePort)

	// Test with empty agent name (should use the first agent)
	err = CreateContaineredAgent(agentFile, "", "localhost", 9090, logger)
	assert.NoError(t, err)
	assert.Equal(t, "test-image:latest", capturedImage)
	assert.Equal(t, "test-agent", capturedName)
	assert.Equal(t, int32(9090), capturedContainerPort)
	assert.Equal(t, int32(9090), capturedServicePort)

	// Test with non-existent agent name
	err = CreateContaineredAgent(agentFile, "non-existent", "localhost", 8080, logger)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "agent not found")

	// Test with invalid YAML file
	invalidFile := filepath.Join(tempDir, "invalid.yaml")
	err = os.WriteFile(invalidFile, []byte("invalid yaml"), 0644)
	require.NoError(t, err)
	err = CreateContaineredAgent(invalidFile, "test-agent", "localhost", 8080, logger)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse agents YAML")

	// Test with missing image in agent definition
	noImageYAML := `
- apiVersion: maestro/v1alpha1
  kind: Agent
  metadata:
    name: no-image-agent
    labels:
      app: test-example
  spec:
    model: gpt-4
    framework: openai
    mode: local
    description: Test agent without image
    instructions: This is a test agent
`
	noImageFile := filepath.Join(tempDir, "no-image.yaml")
	err = os.WriteFile(noImageFile, []byte(noImageYAML), 0644)
	require.NoError(t, err)
	err = CreateContaineredAgent(noImageFile, "no-image-agent", "localhost", 8080, logger)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "missing image")
}

// TestCreateDeploymentService tests the CreateDeploymentService function
func TestCreateDeploymentService(t *testing.T) {
	// Skip if running in CI or without Kubernetes config
	if os.Getenv("CI") == "true" || os.Getenv("KUBECONFIG") == "" {
		t.Skip("Skipping Kubernetes test in CI environment or without KUBECONFIG")
	}

	// Create a logger for testing
	logger, _ := zap.NewDevelopment()

	// Create a fake Kubernetes clientset
	clientset := fake.NewSimpleClientset()

	// Mock the Kubernetes client creation
	// We'll use a package-level variable to hold our mock function
	mockClientsetFunc = func() (*fake.Clientset, error) {
		return clientset, nil
	}

	// Test creating a deployment and service
	err := CreateDeploymentService(
		"test-image:latest",
		"test-app",
		"default",
		1,
		8080,
		8080,
		"LoadBalancer",
		30051,
		logger,
	)
	assert.NoError(t, err)

	// Verify the deployment was created
	deployment, err := clientset.AppsV1().Deployments("default").Get(context.Background(), "test-app", metav1.GetOptions{})
	assert.NoError(t, err)
	assert.Equal(t, "test-app", deployment.Name)
	assert.Equal(t, int32(1), *deployment.Spec.Replicas)
	assert.Equal(t, "test-image:latest", deployment.Spec.Template.Spec.Containers[0].Image)
	assert.Equal(t, int32(8080), deployment.Spec.Template.Spec.Containers[0].Ports[0].ContainerPort)

	// Verify the service was created
	service, err := clientset.CoreV1().Services("default").Get(context.Background(), "test-app", metav1.GetOptions{})
	assert.NoError(t, err)
	assert.Equal(t, "test-app", service.Name)
	assert.Equal(t, int32(8080), service.Spec.Ports[0].Port)
	assert.Equal(t, int32(30051), service.Spec.Ports[0].NodePort)
	assert.Equal(t, corev1.ServiceTypeLoadBalancer, service.Spec.Type)
}

// Variables to hold mock functions
var mockClientsetFunc = func() (*fake.Clientset, error) {
	return nil, nil
}

// Made with Bob
