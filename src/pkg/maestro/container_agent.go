// SPDX-License-Identifier: Apache-2.0
// Copyright © 2025 IBM

package maestro

import (
	"context"
	"fmt"
	"os"

	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

// Variable to hold the deployment service function for testing
var mockDeploymentServiceFunc = CreateDeploymentService

// CreateContaineredAgent creates a containerized agent from an agent definition file
func CreateContaineredAgent(agentsFile string, agentName string, host string, port int, logger *zap.Logger) error {
	// Parse the agents YAML file
	agentsData, err := os.ReadFile(agentsFile)
	if err != nil {
		return fmt.Errorf("failed to read agents file: %w", err)
	}

	var agentsYAML []map[string]interface{}
	if err := yaml.Unmarshal(agentsData, &agentsYAML); err != nil {
		return fmt.Errorf("failed to parse agents YAML: %w", err)
	}

	// Find the specified agent or use the first one if no name is provided
	var agentDef map[string]interface{}
	var name string
	for _, agent := range agentsYAML {
		metadata, ok := agent["metadata"].(map[string]interface{})
		if !ok {
			continue
		}

		name, ok = metadata["name"].(string)
		if !ok {
			continue
		}

		if agentName == "" || agentName == name {
			agentDef = agent
			break
		}
	}

	if agentDef == nil {
		return fmt.Errorf("agent not found: %s", agentName)
	}

	// Extract image from agent definition
	spec, ok := agentDef["spec"].(map[string]interface{})
	if !ok {
		return fmt.Errorf("invalid agent definition: missing spec")
	}

	image, ok := spec["image"].(string)
	if !ok {
		return fmt.Errorf("invalid agent definition: missing image")
	}

	// Create deployment and service
	if err := mockDeploymentServiceFunc(image, name, "default", 1, int32(port), int32(port), "LoadBalancer", 30051, logger); err != nil {
		return fmt.Errorf("failed to create deployment and service: %w", err)
	}

	return nil
}

// CreateDeploymentService creates a Kubernetes Deployment and Service for a given container image
func CreateDeploymentService(
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
	// Load Kubernetes configuration
	config, err := clientcmd.BuildConfigFromFlags("", os.Getenv("KUBECONFIG"))
	if err != nil {
		return fmt.Errorf("failed to load kubeconfig: %w", err)
	}

	// Create Kubernetes clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return fmt.Errorf("failed to create Kubernetes client: %w", err)
	}

	// Define Deployment
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      appName,
			Namespace: namespace,
			Labels: map[string]string{
				"app": appName,
			},
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": appName,
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": appName,
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:            appName,
							Image:           imageURL,
							ImagePullPolicy: corev1.PullIfNotPresent,
							Ports: []corev1.ContainerPort{
								{
									ContainerPort: containerPort,
								},
							},
						},
					},
				},
			},
		},
	}

	// Create Deployment
	ctx := context.Background()
	_, err = clientset.AppsV1().Deployments(namespace).Create(ctx, deployment, metav1.CreateOptions{})
	if err != nil {
		if errors.IsAlreadyExists(err) {
			logger.Info("Deployment already exists", zap.String("name", appName), zap.String("namespace", namespace))
		} else {
			return fmt.Errorf("failed to create deployment: %w", err)
		}
	} else {
		logger.Info("Deployment created successfully", zap.String("name", appName), zap.String("namespace", namespace))
	}

	// Define Service
	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      appName,
			Namespace: namespace,
			Labels: map[string]string{
				"app": appName,
			},
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{
				"app": appName,
			},
			Ports: []corev1.ServicePort{
				{
					Port:       servicePort,
					TargetPort: intstr.FromInt(int(containerPort)),
					NodePort:   nodePort,
				},
			},
			Type: corev1.ServiceType(serviceType),
		},
	}

	// Create Service
	_, err = clientset.CoreV1().Services(namespace).Create(ctx, service, metav1.CreateOptions{})
	if err != nil {
		if errors.IsAlreadyExists(err) {
			logger.Info("Service already exists", zap.String("name", appName), zap.String("namespace", namespace))
		} else {
			return fmt.Errorf("failed to create service: %w", err)
		}
	} else {
		logger.Info("Service created successfully", zap.String("name", appName), zap.String("namespace", namespace))
	}

	return nil
}

// Made with Bob
