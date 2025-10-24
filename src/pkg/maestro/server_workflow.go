// SPDX-License-Identifier: Apache-2.0
// Copyright © 2025 IBM

package maestro

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"gopkg.in/yaml.v3"
)

// WorkflowServer represents a server for serving Maestro workflows
type WorkflowServer struct {
	AgentsFile   string
	WorkflowFile string
	Workflow     *Workflow
	WorkflowName string
	Router       *gin.Engine
}

// NewWorkflowServer creates a new workflow server
func NewWorkflowServer(agentsFile string, workflowFile string) (*WorkflowServer, error) {
	server := &WorkflowServer{
		AgentsFile:   agentsFile,
		WorkflowFile: workflowFile,
	}

	// Initialize router
	router := gin.Default()

	// Configure CORS
	corsAllowOrigins := os.Getenv("CORS_ALLOW_ORIGINS")
	var allowOrigins []string
	if corsAllowOrigins != "" {
		allowOrigins = strings.Split(corsAllowOrigins, ",")
		for i := range allowOrigins {
			allowOrigins[i] = strings.TrimSpace(allowOrigins[i])
		}
	} else {
		allowOrigins = []string{"*"}
	}

	router.Use(cors.New(cors.Config{
		AllowOrigins: allowOrigins,
		AllowMethods: []string{"GET", "POST"},
		AllowHeaders: []string{"Origin", "Content-Type"},
	}))

	server.Router = router

	// Load workflow
	if err := server.LoadWorkflow(); err != nil {
		return nil, fmt.Errorf("failed to load workflow: %w", err)
	}

	// Set up routes
	server.SetupRoutes()

	return server, nil
}

// LoadWorkflow loads the workflow from the workflow file
func (s *WorkflowServer) LoadWorkflow() error {
	// Read agents file
	agentsData, err := os.ReadFile(s.AgentsFile)
	if err != nil {
		return fmt.Errorf("failed to read agents file: %w", err)
	}

	// Parse agents YAML
	var agentsYAML []map[string]interface{}
	if err := yaml.Unmarshal(agentsData, &agentsYAML); err != nil {
		return fmt.Errorf("failed to parse agents YAML: %w", err)
	}

	// Read workflow file
	workflowData, err := os.ReadFile(s.WorkflowFile)
	if err != nil {
		return fmt.Errorf("failed to read workflow file: %w", err)
	}

	// Parse workflow YAML
	var workflowYAML []map[string]interface{}
	if err := yaml.Unmarshal(workflowData, &workflowYAML); err != nil {
		return fmt.Errorf("failed to parse workflow YAML: %w", err)
	}

	if len(workflowYAML) == 0 {
		return fmt.Errorf("no workflow found in %s", s.WorkflowFile)
	}

	// Create workflow
	workflow, err := NewWorkflow(agentsYAML, []string{}, workflowYAML[0], "", nil)
	if err != nil {
		return fmt.Errorf("failed to create workflow: %w", err)
	}

	s.Workflow = workflow

	// Get workflow name
	metadata, ok := workflowYAML[0]["metadata"].(map[string]interface{})
	if !ok {
		return fmt.Errorf("invalid workflow definition: missing metadata")
	}

	name, ok := metadata["name"].(string)
	if !ok {
		return fmt.Errorf("invalid workflow definition: missing name")
	}

	s.WorkflowName = name

	log.Printf("Workflow loaded: %s", s.WorkflowName)
	return nil
}

// SetupRoutes sets up the HTTP routes
func (s *WorkflowServer) SetupRoutes() {
	// Chat endpoint
	s.Router.POST("/chat", func(c *gin.Context) {
		var req WorkflowChatRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Run workflow
		result, err := s.Workflow.Run(context.Background(), req.Prompt)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		// Convert result to string
		var responseStr string
		if result.FinalPrompt != "" {
			responseStr = result.FinalPrompt
		} else {
			responseBytes, err := json.Marshal(result)
			if err != nil {
				responseStr = fmt.Sprintf("%+v", result)
			} else {
				responseStr = string(responseBytes)
			}
		}

		c.JSON(http.StatusOK, WorkflowChatResponse{
			Response:     responseStr,
			WorkflowName: s.WorkflowName,
			Timestamp:    time.Now().UTC(),
		})
	})

	// Streaming chat endpoint
	s.Router.POST("/chat/stream", func(c *gin.Context) {
		var req WorkflowChatRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		c.Header("Content-Type", "text/event-stream")
		c.Header("Cache-Control", "no-cache")
		c.Header("Connection", "keep-alive")

		// Flush headers
		c.Writer.Flush()

		// Run workflow with streaming
		resultChan, err := s.Workflow.RunStreaming(context.Background(), req.Prompt)
		if err != nil {
			event := StreamEvent{
				Error: err.Error(),
			}
			data, err := json.Marshal(event)
			if err != nil {
				log.Printf("Error marshaling event: %v", err)
			} else {
				if _, err := c.Writer.Write([]byte(fmt.Sprintf("data: %s\n\n", data))); err != nil {
					log.Printf("Error writing response: %v", err)
				}
				c.Writer.Flush()
			}
			return
		}

		// Stream results
		for result := range resultChan {
			if result.Error != nil {
				event := StreamEvent{
					Error: result.Error.Error(),
				}
				data, err := json.Marshal(event)
				if err != nil {
					log.Printf("Error marshaling event: %v", err)
					continue
				}
				if _, err := c.Writer.Write([]byte(fmt.Sprintf("data: %s\n\n", data))); err != nil {
					log.Printf("Error writing response: %v", err)
				}
				c.Writer.Flush()
				continue
			}

			if result.IsFinal {
				event := StreamEvent{
					Response:         result.StepResult,
					WorkflowName:     s.WorkflowName,
					WorkflowComplete: true,
				}
				data, err := json.Marshal(event)
				if err != nil {
					log.Printf("Error marshaling event: %v", err)
					continue
				}
				if _, err := c.Writer.Write([]byte(fmt.Sprintf("data: %s\n\n", data))); err != nil {
					log.Printf("Error writing response: %v", err)
				}
				c.Writer.Flush()
				continue
			}

			event := StreamEvent{
				StepName:     result.StepName,
				StepResult:   result.StepResult,
				AgentName:    result.AgentName,
				StepComplete: true,
			}
			data, err := json.Marshal(event)
			if err != nil {
				log.Printf("Error marshaling event: %v", err)
				continue
			}
			if _, err := c.Writer.Write([]byte(fmt.Sprintf("data: %s\n\n", data))); err != nil {
				log.Printf("Error writing response: %v", err)
			}
			c.Writer.Flush()
		}
	})

	// Health endpoint
	s.Router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, WorkflowHealthResponse{
			Status:       "healthy",
			WorkflowName: s.WorkflowName,
			Timestamp:    time.Now().UTC(),
		})
	})

	// Diagram endpoint
	s.Router.GET("/diagram", func(c *gin.Context) {
		diagram, err := s.Workflow.ToMermaid("sequenceDiagram", "")
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, DiagramResponse{
			Diagram:      diagram,
			WorkflowName: s.WorkflowName,
		})
	})
}

// Run starts the server
func (s *WorkflowServer) Run(host string, port int) error {
	addr := fmt.Sprintf("%s:%d", host, port)
	log.Printf("Starting Maestro workflow server on %s", addr)
	log.Printf("API documentation available at: http://%s/docs", addr)
	log.Printf("Health check available at: http://%s/health", addr)

	return s.Router.Run(addr)
}

// ServeWorkflow serves a workflow via HTTP
func ServeWorkflow(agentsFile string, workflowFile string, host string, port int) error {
	server, err := NewWorkflowServer(agentsFile, workflowFile)
	if err != nil {
		return err
	}
	return server.Run(host, port)
}

// Made with Bob
