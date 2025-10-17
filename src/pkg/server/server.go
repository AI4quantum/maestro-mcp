package server

import (
	"context"
	"fmt"
	"time"

	"github.com/AI4quantum/maestro-mcp/src/pkg/config"
	localmcp "github.com/AI4quantum/maestro-mcp/src/pkg/mcp"
	"github.com/mark3labs/mcp-go/server"
	"go.uber.org/zap"
)

// Server represents the MCP server
type Server struct {
	config     *config.Config
	logger     *zap.Logger
	mcpServer  *localmcp.Server
	httpServer *server.StreamableHTTPServer
}

func ServerFromContext(ctx context.Context) (*localmcp.Server, error) {
	server, ok := ctx.Value(localmcp.Server{}).(*localmcp.Server)
	if !ok {
		return nil, fmt.Errorf("failed to get server from context")
	}
	return server, nil
}

// New creates a new server instance
func New(cfg *config.Config, logger *zap.Logger) (*Server, error) {
	// Create MCP server
	mcpServer, err := localmcp.NewServer(cfg, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create MCP server: %w", err)
	}
	httpServer := server.NewStreamableHTTPServer(mcpServer.MCPServer)

	return &Server{
		config:     cfg,
		logger:     logger,
		mcpServer:  mcpServer,
		httpServer: httpServer,
	}, nil
}

// Start starts the server
func (s *Server) Start(ctx context.Context) error {
	//s.logger.Info("Starting MCP server",
	//	zap.String("address", s.httpServer.Addr))

	// Start HTTP server in a goroutine
	serverErr := make(chan error, 1)
	go func() {
		if err := s.httpServer.Start(":8030"); err != nil {
			serverErr <- err
		}
	}()

	// Wait for context cancellation or server error
	select {
	case <-ctx.Done():
		s.logger.Info("Shutting down server...")

		// Create shutdown context with timeout
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		// Shutdown HTTP server
		if err := s.httpServer.Shutdown(shutdownCtx); err != nil {
			s.logger.Error("Server shutdown error", zap.Error(err))
			return fmt.Errorf("server shutdown error: %w", err)
		}

		s.logger.Info("Server shutdown complete")
		return nil

	case err := <-serverErr:
		s.logger.Error("Server error", zap.Error(err))
		return fmt.Errorf("server error: %w", err)
	}
}

// Stop gracefully stops the server
func (s *Server) Stop() error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	return s.httpServer.Shutdown(ctx)
}

// Made with Bob
