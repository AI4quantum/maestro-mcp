// SPDX-License-Identifier: Apache-2.0
// Copyright © 2025 IBM

package main

import (
	"bufio"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/AI4quantum/maestro-mcp/src/pkg/maestro"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

// APIServer represents the API server for Maestro
type APIServer struct {
	Router       *gin.Engine
	Workflow     *maestro.Workflow
	WorkflowName string
	AgentsFile   string
	WorkflowFile string
}

// NewAPIServer creates a new API server
func NewAPIServer() (*APIServer, error) {
	// Set default file paths
	agentsFile := "src/agents.yaml"
	workflowFile := "src/workflow.yaml"

	// Create server instance
	server := &APIServer{
		AgentsFile:   agentsFile,
		WorkflowFile: workflowFile,
	}

	// Initialize router
	router := gin.Default()

	// No need for custom template functions

	// Configure CORS
	corsAllowOrigins := os.Getenv("CORS_ALLOW_ORIGINS")
	var allowOrigins []string
	if corsAllowOrigins != "" {
		allowOrigins = []string{corsAllowOrigins}
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

// LoadWorkflow loads the workflow from YAML files
func (s *APIServer) LoadWorkflow() error {
	// Parse YAML files
	agentsYAML, err := maestro.ParseYAML(s.AgentsFile)
	if err != nil {
		return fmt.Errorf("failed to read agents file: %w", err)
	}

	workflowsYAML, err := maestro.ParseYAML(s.WorkflowFile)
	if err != nil {
		return fmt.Errorf("failed to read workflow file: %w", err)
	}

	// Create workflow
	workflowID := fmt.Sprintf("workflow-%d", time.Now().Unix())
	workflow, err := maestro.NewWorkflow(agentsYAML, []string{}, workflowsYAML[0], workflowID, nil)
	if err != nil {
		return fmt.Errorf("failed to create workflow: %w", err)
	}

	s.Workflow = workflow

	// Get workflow name
	metadata, ok := workflowsYAML[0]["metadata"].(map[string]interface{})
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
func (s *APIServer) SetupRoutes() {
	// Serve static files
	s.Router.StaticFile("src/workflow.yaml", s.WorkflowFile)
	s.Router.StaticFile("src/agents.yaml", s.AgentsFile)
	// Main page with HTML rendering
	// Use a relative path that works regardless of the working directory
	// First try the current directory
	if _, err := os.Stat("maestro.html"); err == nil {
		s.Router.LoadHTMLFiles("maestro.html")
	} else if _, err := os.Stat("deployments/maestro.html"); err == nil {
		// Then try the deployments directory
		s.Router.LoadHTMLFiles("deployments/maestro.html")
	} else {
		// Finally try to find it relative to the executable
		exePath, err := os.Executable()
		if err == nil {
			dir := filepath.Dir(exePath)
			// Try different possible locations
			possiblePaths := []string{
				filepath.Join(dir, "maestro.html"),
				filepath.Join(dir, "deployments", "maestro.html"),
				filepath.Join(dir, "..", "deployments", "maestro.html"),
			}

			for _, path := range possiblePaths {
				if _, err := os.Stat(path); err == nil {
					s.Router.LoadHTMLFiles(path)
					break
				}
			}
		}
	}
	s.Router.GET("/", func(c *gin.Context) {
		// Get prompt from query parameters
		prompt := c.Query("Prompt")
		autoRun := os.Getenv("AUTO_RUN") == "true"
		shouldRun := autoRun || prompt != ""

		// Update prompt in workflow if provided
		if prompt != "" {
			if spec, ok := s.Workflow.WorkflowDef["spec"].(map[string]interface{}); ok {
				if template, ok := spec["template"].(map[string]interface{}); ok {
					template["prompt"] = prompt
				}
			}
		}

		// Generate diagram
		diagram, err := s.Workflow.ToMermaid("sequenceDiagram", "TD")
		if err != nil {
			c.HTML(http.StatusInternalServerError, "maestro.html", gin.H{
				"error": fmt.Sprintf("Error generating diagram: %v", err),
				"title": s.WorkflowName,
			})
			return
		}

		// Run workflow if needed
		if shouldRun {
			// Run workflow in a goroutine
			go func() {
				_, err := s.Workflow.Run(c.Request.Context(), prompt)
				if err != nil {
					log.Printf("Error running workflow: %v", err)
				}
			}()
		}

		// Render HTML template
		c.HTML(http.StatusOK, "maestro.html", gin.H{
			"result":  "",
			"title":   s.WorkflowName,
			"diagram": diagram,
		})
	})

	// Stream endpoint for SSE
	s.Router.GET("/stream", func(c *gin.Context) {
		c.Header("Content-Type", "text/event-stream")
		c.Header("Cache-Control", "no-cache")
		c.Header("Connection", "keep-alive")
		c.Header("Access-Control-Allow-Origin", "*")

		// Create a channel for client disconnect
		clientGone := c.Writer.CloseNotify()

		// Create a buffer to store output
		var outputBuffer []string

		// Create a channel to signal when to stop
		done := make(chan bool)

		// Start a goroutine to capture workflow output
		go func() {
			// Redirect stdout to capture output
			oldStdout := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			// Create a channel for reading from the pipe
			outputChan := make(chan string)

			// Start a goroutine to read from the pipe
			go func() {
				scanner := bufio.NewScanner(r)
				for scanner.Scan() {
					outputChan <- scanner.Text()
				}
			}()

			// Wait for output or client disconnect
			for {
				select {
				case <-clientGone:
					w.Close()
					os.Stdout = oldStdout
					done <- true
					return
				case output := <-outputChan:
					// Send the output as an SSE event
					c.SSEvent("message", output)
					c.Writer.Flush()
					outputBuffer = append(outputBuffer, output)
				}
			}
		}()

		// Wait for done signal
		<-done
	})

	// Chat endpoint
	s.Router.POST("/chat", func(c *gin.Context) {
		var req struct {
			Prompt string `json:"prompt"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Run workflow
		result, err := s.Workflow.Run(c.Request.Context(), req.Prompt)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		// Return response
		c.JSON(http.StatusOK, gin.H{
			"response":      result.FinalPrompt,
			"workflow_name": s.WorkflowName,
			"timestamp":     time.Now().UTC().Format(time.RFC3339),
		})
	})

	// Streaming chat endpoint
	s.Router.POST("/chat/stream", func(c *gin.Context) {
		var req struct {
			Prompt string `json:"prompt"`
		}
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
		resultChan, err := s.Workflow.RunStreaming(c.Request.Context(), req.Prompt)
		if err != nil {
			event := gin.H{
				"error": err.Error(),
			}
			c.SSEvent("", event)
			c.Writer.Flush()
			return
		}

		// Stream results
		for result := range resultChan {
			if result.Error != nil {
				event := gin.H{
					"error": result.Error.Error(),
				}
				c.SSEvent("", event)
				c.Writer.Flush()
				continue
			}

			if result.IsFinal {
				event := gin.H{
					"response":          result.StepResult,
					"workflow_name":     s.WorkflowName,
					"workflow_complete": true,
				}
				c.SSEvent("", event)
				c.Writer.Flush()
				continue
			}

			event := gin.H{
				"step_name":     result.StepName,
				"step_result":   result.StepResult,
				"agent_name":    result.AgentName,
				"step_complete": true,
			}
			c.SSEvent("", event)
			c.Writer.Flush()
		}
	})

	// Health endpoint
	s.Router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":        "healthy",
			"workflow_name": s.WorkflowName,
			"timestamp":     time.Now().UTC().Format(time.RFC3339),
		})
	})

	// Diagram endpoint
	s.Router.GET("/diagram", func(c *gin.Context) {
		diagram, err := s.Workflow.ToMermaid("sequenceDiagram", "TD")
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"diagram":       diagram,
			"workflow_name": s.WorkflowName,
		})
	})
}

// Run starts the server
func (s *APIServer) Run() error {
	// Get host and port from environment or use defaults
	host := os.Getenv("HOST")
	if host == "" {
		host = "0.0.0.0"
	}

	portStr := os.Getenv("PORT")
	if portStr == "" {
		portStr = "8080"
	}

	port, err := strconv.Atoi(portStr)
	if err != nil {
		return fmt.Errorf("invalid port: %w", err)
	}

	// Start server
	addr := fmt.Sprintf("%s:%d", host, port)
	log.Printf("Starting Maestro API server on %s", addr)
	return s.Router.Run(addr)
}

func main() {
	// Set working directory to the project root
	exePath, err := os.Executable()
	if err == nil {
		dir := filepath.Dir(exePath)
		err = os.Chdir(dir)
		if err != nil {
			log.Printf("Warning: Failed to change working directory: %v", err)
		}
	}

	// Create and run server
	server, err := NewAPIServer()
	if err != nil {
		log.Fatalf("Failed to create API server: %v", err)
	}

	if err := server.Run(); err != nil {
		log.Fatalf("Failed to run API server: %v", err)
	}
}

// Made with Bob
