// SPDX-License-Identifier: Apache-2.0
// Copyright © 2025 IBM

package maestro

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"go.uber.org/zap"
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
func CreateContaineredAgent(imageURL string, agentName string, host string, port int, logger *zap.Logger) error {
	// Create deployment and service
	if err := CreateDeploymentService(imageURL, agentName, "default", 1, int32(port), int32(port), "LoadBalancer", 30051, logger); err != nil {
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
	kubeconfig := filepath.Join(os.Getenv("HOME"), ".kube", "config")
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		return fmt.Errorf("Error building kubeconfig: %v", err)
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
