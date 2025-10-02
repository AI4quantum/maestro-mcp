package config

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/viper"
)

// Config represents the application configuration
type Config struct {
	Version  string         `mapstructure:"version"`
	Server   ServerConfig   `mapstructure:"server"`
	Database DatabaseConfig `mapstructure:"database"`
	Logging  LoggingConfig  `mapstructure:"logging"`
	MCP      MCPConfig      `mapstructure:"mcp"`
}

// ServerConfig contains server-related configuration
type ServerConfig struct {
	Host         string        `mapstructure:"host"`
	Port         int           `mapstructure:"port"`
	ReadTimeout  time.Duration `mapstructure:"read_timeout"`
	WriteTimeout time.Duration `mapstructure:"write_timeout"`
	IdleTimeout  time.Duration `mapstructure:"idle_timeout"`
}

// DatabaseConfig contains database-related configuration
type DatabaseConfig struct {
	Type         string `mapstructure:"type"`
	Host         string `mapstructure:"host"`
	Port         int    `mapstructure:"port"`
	Database     string `mapstructure:"database"`
	Username     string `mapstructure:"username"`
	Password     string `mapstructure:"password"`
	SSLMode      string `mapstructure:"ssl_mode"`
	MaxConns     int    `mapstructure:"max_connections"`
	MaxIdleConns int    `mapstructure:"max_idle_connections"`
}

// LoggingConfig contains logging-related configuration
type LoggingConfig struct {
	Level  string `mapstructure:"level"`
	Format string `mapstructure:"format"`
	Output string `mapstructure:"output"`
}

// MCPConfig contains MCP-specific configuration
type MCPConfig struct {
	ToolTimeout time.Duration            `mapstructure:"tool_timeout"`
	Timeouts    map[string]time.Duration `mapstructure:"timeouts"`
	Embedding   EmbeddingConfig          `mapstructure:"embedding"`
	VectorDB    VectorDBConfig           `mapstructure:"vector_db"`
}

// EmbeddingConfig contains embedding-related configuration
type EmbeddingConfig struct {
	Provider   string `mapstructure:"provider"`
	Model      string `mapstructure:"model"`
	APIKey     string `mapstructure:"api_key"`
	URL        string `mapstructure:"url"`
	VectorSize int    `mapstructure:"vector_size"`
}

// VectorDBConfig contains vector database configuration
type VectorDBConfig struct {
	Type     string         `mapstructure:"type"`
	Milvus   MilvusConfig   `mapstructure:"milvus"`
	Weaviate WeaviateConfig `mapstructure:"weaviate"`
}

// MilvusConfig contains Milvus-specific configuration
type MilvusConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	Username string `mapstructure:"username"`
	Password string `mapstructure:"password"`
	Database string `mapstructure:"database"`
}

// WeaviateConfig contains Weaviate-specific configuration
type WeaviateConfig struct {
	URL     string        `mapstructure:"url"`
	APIKey  string        `mapstructure:"api_key"`
	Timeout time.Duration `mapstructure:"timeout"`
}

// Load loads configuration from various sources
func Load() (*Config, error) {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	viper.AddConfigPath("./config")
	viper.AddConfigPath("/etc/maestro-mcp")

	// Set default values
	setDefaults()

	// Enable environment variable support
	viper.AutomaticEnv()
	viper.SetEnvPrefix("MAESTRO_MCP")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// Load .env file if it exists
	if err := loadEnvFile(); err != nil {
		return nil, fmt.Errorf("failed to load .env file: %w", err)
	}

	// Read config file (optional)
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}
		// Config file not found is OK, we'll use defaults and env vars
	}

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Validate configuration
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return &config, nil
}

// setDefaults sets default configuration values
func setDefaults() {
	viper.SetDefault("version", "0.1.0")

	// Server defaults
	viper.SetDefault("server.host", "localhost")
	viper.SetDefault("server.port", 8030)
	viper.SetDefault("server.read_timeout", "30s")
	viper.SetDefault("server.write_timeout", "30s")
	viper.SetDefault("server.idle_timeout", "120s")

	// Database defaults
	viper.SetDefault("database.type", "postgres")
	viper.SetDefault("database.host", "localhost")
	viper.SetDefault("database.port", 5432)
	viper.SetDefault("database.database", "maestro")
	viper.SetDefault("database.ssl_mode", "disable")
	viper.SetDefault("database.max_connections", 25)
	viper.SetDefault("database.max_idle_connections", 5)

	// Logging defaults
	viper.SetDefault("logging.level", "info")
	viper.SetDefault("logging.format", "json")
	viper.SetDefault("logging.output", "stdout")

	// MCP defaults
	viper.SetDefault("mcp.tool_timeout", "15s")
	viper.SetDefault("mcp.timeouts.health", "30s")
	viper.SetDefault("mcp.timeouts.query", "30s")
	viper.SetDefault("mcp.timeouts.write", "900s")
	viper.SetDefault("mcp.timeouts.delete", "60s")

	// Embedding defaults
	viper.SetDefault("mcp.embedding.provider", "openai")
	viper.SetDefault("mcp.embedding.model", "text-embedding-ada-002")
	viper.SetDefault("mcp.embedding.vector_size", 1536)

	// Vector DB defaults
	viper.SetDefault("mcp.vector_db.type", "milvus")
	viper.SetDefault("mcp.vector_db.milvus.host", "localhost")
	viper.SetDefault("mcp.vector_db.milvus.port", 19530)
	viper.SetDefault("mcp.vector_db.weaviate.timeout", "10s")
}

// loadEnvFile loads environment variables from .env file
func loadEnvFile() error {
	envFile := ".env"
	if _, err := os.Stat(envFile); os.IsNotExist(err) {
		return nil // .env file doesn't exist, that's OK
	}

	file, err := os.Open(envFile)
	if err != nil {
		return err
	}
	defer file.Close()

	// Simple .env file parser
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		// Remove quotes if present
		if len(value) >= 2 && ((value[0] == '"' && value[len(value)-1] == '"') ||
			(value[0] == '\'' && value[len(value)-1] == '\'')) {
			value = value[1 : len(value)-1]
		}

		os.Setenv(key, value)
	}

	return scanner.Err()
}

// Validate validates the configuration
func (c *Config) Validate() error {
	if c.Server.Port <= 0 || c.Server.Port > 65535 {
		return fmt.Errorf("invalid server port: %d", c.Server.Port)
	}

	if c.Database.Type == "" {
		return fmt.Errorf("database type is required")
	}

	if c.MCP.VectorDB.Type == "" {
		return fmt.Errorf("vector database type is required")
	}

	// Validate vector database specific configs
	switch c.MCP.VectorDB.Type {
	case "milvus":
		if c.MCP.VectorDB.Milvus.Host == "" {
			return fmt.Errorf("milvus host is required")
		}
		if c.MCP.VectorDB.Milvus.Port <= 0 || c.MCP.VectorDB.Milvus.Port > 65535 {
			return fmt.Errorf("invalid milvus port: %d", c.MCP.VectorDB.Milvus.Port)
		}
	case "weaviate":
		if c.MCP.VectorDB.Weaviate.URL == "" {
			return fmt.Errorf("weaviate URL is required")
		}
	default:
		return fmt.Errorf("unsupported vector database type: %s", c.MCP.VectorDB.Type)
	}

	return nil
}

// GetTimeout returns the timeout for a specific operation category
func (c *Config) GetTimeout(category string) time.Duration {
	if timeout, exists := c.MCP.Timeouts[category]; exists {
		return timeout
	}
	return c.MCP.ToolTimeout
}

// IsDevelopment returns true if running in development mode
func (c *Config) IsDevelopment() bool {
	return strings.ToLower(c.Logging.Level) == "debug"
}
