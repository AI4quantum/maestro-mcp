// SPDX-License-Identifier: Apache-2.0
// Copyright © 2025 IBM

package maestro

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/AI4quantum/maestro-mcp/src/pkg/maestro/agents"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

// AgentServer represents a server for serving Maestro agents
type AgentServer struct {
	AgentsFile string
	AgentName  string
	Agents     map[string]Agent
	Router     *gin.Engine
}

// NewAgentServer creates a new agent server
func NewAgentServer(agentsFile string, agentName string) (*AgentServer, error) {
	server := &AgentServer{
		AgentsFile: agentsFile,
		AgentName:  agentName,
		Agents:     make(map[string]Agent),
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

	// Load agents
	if err := server.LoadAgents(); err != nil {
		return nil, fmt.Errorf("failed to load agents: %w", err)
	}

	// Set up routes
	server.SetupRoutes()

	return server, nil
}

// LoadAgents loads agents from the agents file
func (s *AgentServer) LoadAgents() error {
	agentsYAML, err := ParseYAML(s.AgentsFile)
	if err != nil {
		return fmt.Errorf("failed to read agents file: %w", err)
	}

	// Create agents
	if err := CreateAgents(agentsYAML); err != nil {
		return fmt.Errorf("failed to create agents: %w", err)
	}

	// Load agents into memory
	for _, agentDef := range agentsYAML {
		metadata, ok := agentDef["metadata"].(map[string]interface{})
		if !ok {
			return fmt.Errorf("invalid agent definition: missing metadata")
		}

		agentName, ok := metadata["name"].(string)
		if !ok {
			return fmt.Errorf("invalid agent definition: missing name")
		}

		// Skip if specific agent name is provided and doesn't match
		if s.AgentName != "" && agentName != s.AgentName {
			continue
		}

		// Get agent class
		spec, ok := agentDef["spec"].(map[string]interface{})
		if !ok {
			return fmt.Errorf("invalid agent definition: missing spec")
		}

		framework, _ := spec["framework"].(string)
		if framework == "" {
			framework = "beeai" // Default framework
		}

		mode, _ := spec["mode"].(string)
		agentClass, err := getAgentClass(agents.AgentFramework(framework), mode)
		if err != nil {
			return fmt.Errorf("failed to get agent class: %w", err)
		}

		// Create agent instance
		agentInstance, err := agentClass(agentDef)
		if err != nil {
			return fmt.Errorf("failed to create agent: %w", err)
		}

		s.Agents[agentName] = agentInstance.(Agent)
	}

	if len(s.Agents) == 0 {
		return fmt.Errorf("no agents found in %s", s.AgentsFile)
	}

	log.Printf("Loaded %d agent(s): %v", len(s.Agents), getMapKeys(s.Agents))
	return nil
}

// SetupRoutes sets up the HTTP routes
func (s *AgentServer) SetupRoutes() {
	// Chat endpoint
	s.Router.POST("/chat", func(c *gin.Context) {
		var req ChatRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Get agent
		var agent Agent
		if s.AgentName != "" && s.Agents[s.AgentName] != nil {
			agent = s.Agents[s.AgentName]
		} else if len(s.Agents) == 1 {
			for _, a := range s.Agents {
				agent = a
				break
			}
		} else {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": fmt.Sprintf("Agent '%s' not found. Available agents: %v", s.AgentName, getMapKeys(s.Agents)),
			})
			return
		}

		// Handle streaming request
		if req.Stream {
			c.Header("Content-Type", "text/event-stream")
			c.Header("Cache-Control", "no-cache")
			c.Header("Connection", "keep-alive")

			// Flush headers
			c.Writer.Flush()

			// Run agent
			response, err := agent.Run(req.Prompt)
			if err != nil {
				event := StreamEvent{
					Error: err.Error(),
				}
				data, _ := json.Marshal(event)
				if _, err := c.Writer.Write([]byte(fmt.Sprintf("data: %s\n\n", data))); err != nil {
					log.Printf("Error writing response: %v", err)
				}
				c.Writer.Flush()
				return
			}

			// Send response
			event := StreamEvent{
				Response:  response.(string),
				AgentName: agent.GetName(),
			}
			data, _ := json.Marshal(event)
			if _, err := c.Writer.Write([]byte(fmt.Sprintf("data: %s\n\n", data))); err != nil {
				log.Printf("Error writing response: %v", err)
			}
			c.Writer.Flush()
			return
		}

		// Handle regular request
		response, err := agent.Run(req.Prompt)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, ChatResponse{
			Response:  response.(string),
			AgentName: agent.GetName(),
			Timestamp: time.Now().UTC(),
		})
	})

	// Health endpoint
	s.Router.GET("/health", func(c *gin.Context) {
		var agentName string
		if s.AgentName != "" {
			agentName = s.AgentName
		} else if len(s.Agents) > 0 {
			for name := range s.Agents {
				agentName = name
				break
			}
		}

		c.JSON(http.StatusOK, HealthResponse{
			Status:    "healthy",
			AgentName: agentName,
			Timestamp: time.Now().UTC(),
		})
	})

	// Agents endpoint
	s.Router.GET("/agents", func(c *gin.Context) {
		var currentAgent string
		if s.AgentName != "" {
			currentAgent = s.AgentName
		} else if len(s.Agents) > 0 {
			for name := range s.Agents {
				currentAgent = name
				break
			}
		}

		c.JSON(http.StatusOK, AgentListResponse{
			Agents:       getMapKeys(s.Agents),
			CurrentAgent: currentAgent,
		})
	})
}

// Run starts the server
func (s *AgentServer) Run(host string, port int) error {
	addr := fmt.Sprintf("%s:%d", host, port)
	log.Printf("Starting Maestro agent server on %s", addr)
	log.Printf("API documentation available at: http://%s/docs", addr)
	log.Printf("Health check available at: http://%s/health", addr)

	return s.Router.Run(addr)
}

// ServeAgent serves an agent via HTTP
func ServeAgent(agentsFile string, agentName string, host string, port int) error {
	server, err := NewAgentServer(agentsFile, agentName)
	if err != nil {
		return err
	}
	return server.Run(host, port)
}

// Helper function to get map keys as a slice
func getMapKeys(m map[string]Agent) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

// Made with Bob
