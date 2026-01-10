// Package config handles application configuration.
package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v2"
)

// ModelConfig defines a model with its description.
type ModelConfig struct {
	ID          string `json:"id" yaml:"id"`
	Description string `json:"description" yaml:"description"`
}

// Config holds the application configuration.
type Config struct {
	DefaultModel string             `json:"default_model" yaml:"default_model"`
	Models       []ModelConfig      `json:"models" yaml:"models"`
	Server       ServerConfig       `json:"server" yaml:"server"`
	Orchestrator OrchestratorConfig `json:"orchestrator" yaml:"orchestrator"`
}

// ServerConfig holds HTTP server configuration.
type ServerConfig struct {
	Host string `json:"host" yaml:"host"`
	Port int    `json:"port" yaml:"port"`
}

// OrchestratorConfig holds orchestrator configuration.
type OrchestratorConfig struct {
	StorePath        string `json:"store_path" yaml:"store_path"`
	LogDir           string `json:"log_dir" yaml:"log_dir"`
	MaxParallel      int    `json:"max_parallel" yaml:"max_parallel"`
	DefaultMCPConfig string `json:"default_mcp_config" yaml:"default_mcp_config"`
	DefaultEngine    string `json:"default_engine" yaml:"default_engine"`
}

// DefaultConfig returns the default configuration.
func DefaultConfig() *Config {
	home, _ := os.UserHomeDir()
	mesnadaDir := filepath.Join(home, ".mesnada")

	return &Config{
		DefaultModel: "claude-sonnet-4.5",
		Models: []ModelConfig{
			{ID: "claude-sonnet-4.5", Description: "Balanced performance and speed for general tasks"},
			{ID: "claude-opus-4.5", Description: "Highest capability for complex reasoning and analysis"},
			{ID: "claude-haiku-4.5", Description: "Fast responses for simple tasks and quick iterations"},
			{ID: "gpt-5.1-codex-max", Description: "Advanced coding capabilities with extended context"},
			{ID: "gpt-5.1-codex", Description: "Optimized for code generation and refactoring"},
			{ID: "gpt-5.2", Description: "Latest GPT model with improved reasoning"},
			{ID: "gpt-5.1", Description: "Stable GPT model for production use"},
			{ID: "gpt-5", Description: "Base GPT-5 model"},
			{ID: "gpt-5.1-codex-mini", Description: "Lightweight coding model for quick tasks"},
			{ID: "gpt-5-mini", Description: "Fast and efficient for simple queries"},
			{ID: "gpt-4.1", Description: "Reliable GPT-4 variant"},
			{ID: "gemini-3-pro-preview", Description: "Google's latest multimodal model"},
		},
		Server: ServerConfig{
			Host: "127.0.0.1",
			Port: 8765,
		},
		Orchestrator: OrchestratorConfig{
			StorePath:   filepath.Join(mesnadaDir, "tasks.json"),
			LogDir:      filepath.Join(mesnadaDir, "logs"),
			MaxParallel: 5,
		},
	}
}

// Load loads configuration from a file (supports JSON and YAML).
func Load(path string) (*Config, error) {
	cfg := DefaultConfig()
	baseDir := ""

	if path == "" {
		home, _ := os.UserHomeDir()
		// Try YAML first, then JSON
		yamlPath := filepath.Join(home, ".mesnada", "config.yaml")
		jsonPath := filepath.Join(home, ".mesnada", "config.json")

		if _, err := os.Stat(yamlPath); err == nil {
			path = yamlPath
			baseDir = filepath.Dir(path)
		} else if _, err := os.Stat(jsonPath); err == nil {
			path = jsonPath
			baseDir = filepath.Dir(path)
		} else {
			// No config file found, return defaults
			return cfg, nil
		}
	} else {
		baseDir = filepath.Dir(path)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return nil, fmt.Errorf("failed to read config: %w", err)
	}

	// Detect format by extension
	isYAML := strings.HasSuffix(strings.ToLower(path), ".yaml") || strings.HasSuffix(strings.ToLower(path), ".yml")

	if isYAML {
		if err := yaml.Unmarshal(data, cfg); err != nil {
			return nil, fmt.Errorf("failed to parse YAML config: %w", err)
		}
	} else {
		if err := json.Unmarshal(data, cfg); err != nil {
			return nil, fmt.Errorf("failed to parse JSON config: %w", err)
		}
	}

	// Expand/resolve paths from config file
	// - StorePath/LogDir: expand ~ and resolve relative paths relative to the config file directory
	// - DefaultMCPConfig: expand ~ (supports both "~/..." and "@~/...") but keep relative paths as-is
	cfg.Orchestrator.StorePath = resolvePath(cfg.Orchestrator.StorePath, baseDir)
	cfg.Orchestrator.LogDir = resolvePath(cfg.Orchestrator.LogDir, baseDir)
	cfg.Orchestrator.DefaultMCPConfig = expandMCPConfig(cfg.Orchestrator.DefaultMCPConfig)

	return cfg, nil
}

// Save saves configuration to a file.
func (c *Config) Save(path string) error {
	if path == "" {
		home, _ := os.UserHomeDir()
		path = filepath.Join(home, ".mesnada", "config.json")
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	return nil
}

// Address returns the server address.
func (c *Config) Address() string {
	return fmt.Sprintf("%s:%d", c.Server.Host, c.Server.Port)
}

// GetModelByID returns a model configuration by ID.
func (c *Config) GetModelByID(id string) *ModelConfig {
	for _, m := range c.Models {
		if m.ID == id {
			return &m
		}
	}
	return nil
}

// ValidateModel checks if a model ID is valid.
func (c *Config) ValidateModel(id string) bool {
	return c.GetModelByID(id) != nil
}

// expandHome expands ~ to home directory in paths.
func expandHome(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return path
	}
	if path == "~" {
		home, _ := os.UserHomeDir()
		return home
	}
	// Support "~/..." (and Windows separators just in case)
	if strings.HasPrefix(path, "~/") || strings.HasPrefix(path, "~\\") {
		home, _ := os.UserHomeDir()
		rest := path[2:]
		return filepath.Join(home, rest)
	}
	// We intentionally don't expand "~user/..." forms.
	return path
}

// resolvePath expands ~ and resolves relative paths against baseDir.
// If baseDir is empty, relative paths are returned unchanged.
func resolvePath(value, baseDir string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return value
	}
	p := expandHome(value)
	if filepath.IsAbs(p) {
		return p
	}
	if baseDir == "" {
		return p
	}
	return filepath.Clean(filepath.Join(baseDir, p))
}

// expandMCPConfig expands ~ in MCP config values.
// It supports both "~/..." and "@~/..." forms.
func expandMCPConfig(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return value
	}
	if strings.HasPrefix(value, "@") {
		return "@" + expandHome(value[1:])
	}
	return expandHome(value)
}
