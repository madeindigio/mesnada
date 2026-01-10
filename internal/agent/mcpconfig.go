// Package agent handles spawning and managing CLI agent processes.
package agent

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// MesnadaMCPConfig represents the Mesnada MCP configuration format.
// This format is used in .github/mcp-config.json
type MesnadaMCPConfig struct {
	MCPServers map[string]MesnadaMCPServer `json:"mcpServers"`
}

// MesnadaMCPServer represents a server entry in Mesnada format.
type MesnadaMCPServer struct {
	Type    string   `json:"type"`              // "local" or "http"
	Command string   `json:"command,omitempty"` // for local
	Args    []string `json:"args,omitempty"`    // for local
	Cwd     string   `json:"cwd,omitempty"`     // for local
	URL     string   `json:"url,omitempty"`     // for http
	Tools   []string `json:"tools,omitempty"`
}

// ClaudeMCPConfig represents the Claude CLI MCP configuration format.
// This is the format expected by `claude --mcp-config`
type ClaudeMCPConfig struct {
	MCPServers map[string]ClaudeMCPServer `json:"mcpServers"`
}

// ClaudeMCPServer represents a server entry in Claude CLI format.
type ClaudeMCPServer struct {
	// For stdio transport (local commands)
	Command string            `json:"command,omitempty"`
	Args    []string          `json:"args,omitempty"`
	Cwd     string            `json:"cwd,omitempty"`
	Env     map[string]string `json:"env,omitempty"`

	// For SSE transport (HTTP)
	Type string `json:"type,omitempty"` // "sse" for HTTP
	URL  string `json:"url,omitempty"`
}

// ConvertMCPConfig converts a Mesnada MCP config file to Claude CLI format.
// It reads the source file and returns the path to a temporary file with the converted config.
func ConvertMCPConfig(mcpConfigPath, tempDir string) (string, error) {
	// Handle @ prefix (file reference)
	sourcePath := mcpConfigPath
	if strings.HasPrefix(sourcePath, "@") {
		sourcePath = sourcePath[1:]
	}

	// Read source file
	data, err := os.ReadFile(sourcePath)
	if err != nil {
		return "", fmt.Errorf("failed to read MCP config: %w", err)
	}

	// Parse Mesnada format
	var mesnadaConfig MesnadaMCPConfig
	if err := json.Unmarshal(data, &mesnadaConfig); err != nil {
		return "", fmt.Errorf("failed to parse MCP config: %w", err)
	}

	// Convert to Claude format
	claudeConfig := ClaudeMCPConfig{
		MCPServers: make(map[string]ClaudeMCPServer),
	}

	for name, server := range mesnadaConfig.MCPServers {
		claudeServer := ClaudeMCPServer{}

		switch server.Type {
		case "local":
			// stdio transport
			claudeServer.Command = server.Command
			claudeServer.Args = server.Args
			claudeServer.Cwd = server.Cwd
		case "http":
			// SSE transport
			claudeServer.Type = "sse"
			claudeServer.URL = server.URL
		default:
			// Assume local if type not specified
			claudeServer.Command = server.Command
			claudeServer.Args = server.Args
			claudeServer.Cwd = server.Cwd
		}

		claudeConfig.MCPServers[name] = claudeServer
	}

	// Ensure temp dir exists
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create temp dir: %w", err)
	}

	// Write to temp file
	outputPath := filepath.Join(tempDir, "claude-mcp-config.json")
	outputData, err := json.MarshalIndent(claudeConfig, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal Claude MCP config: %w", err)
	}

	if err := os.WriteFile(outputPath, outputData, 0644); err != nil {
		return "", fmt.Errorf("failed to write Claude MCP config: %w", err)
	}

	return outputPath, nil
}

// ConvertMCPConfigForTask converts MCP config for a specific task.
// Returns the path to use with --mcp-config flag.
func ConvertMCPConfigForTask(mcpConfigPath, taskID, baseDir string) (string, error) {
	if mcpConfigPath == "" {
		return "", nil
	}

	// Create task-specific temp directory
	tempDir := filepath.Join(baseDir, "claude-mcp", taskID)

	return ConvertMCPConfig(mcpConfigPath, tempDir)
}

// CleanupMCPConfig removes the temporary MCP config file for a task.
func CleanupMCPConfig(taskID, baseDir string) error {
	tempDir := filepath.Join(baseDir, "claude-mcp", taskID)
	return os.RemoveAll(tempDir)
}
