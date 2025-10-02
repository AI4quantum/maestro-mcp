package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/maximilien/maestro-mcp/src/pkg/config"
	"github.com/maximilien/maestro-mcp/src/pkg/server"
	"go.uber.org/zap"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Initialize logger
	logger, err := zap.NewProduction()
	if err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}
	defer func() {
		if err := logger.Sync(); err != nil {
			log.Printf("Failed to sync logger: %v", err)
		}
	}()

	logger.Info("Starting Maestro MCP Server",
		zap.String("version", cfg.Version),
		zap.String("host", cfg.Server.Host),
		zap.Int("port", cfg.Server.Port))

	// Create server
	srv, err := server.New(cfg, logger)
	if err != nil {
		logger.Fatal("Failed to create server", zap.Error(err))
	}

	// Setup graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle shutdown signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		logger.Info("Received shutdown signal")
		cancel()
	}()

	// Start server
	if err := srv.Start(ctx); err != nil {
		logger.Fatal("Server failed to start", zap.Error(err))
	}

	logger.Info("Server shutdown complete")
}
