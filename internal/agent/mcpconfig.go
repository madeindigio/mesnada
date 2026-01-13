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
// workDir is the working directory to resolve relative paths (should be task.WorkDir).
func ConvertMCPConfig(mcpConfigPath, tempDir, workDir string) (string, error) {
	// Handle @ prefix (file reference)
	sourcePath := mcpConfigPath
	if strings.HasPrefix(sourcePath, "@") {
		sourcePath = sourcePath[1:]
	}

	// Convert workDir to absolute path if it's relative
	absWorkDir := workDir
	if workDir != "" && !filepath.IsAbs(workDir) {
		var err error
		absWorkDir, err = filepath.Abs(workDir)
		if err != nil {
			return "", fmt.Errorf("failed to resolve workDir to absolute path: %w", err)
		}
	}

	// Resolve relative paths from workDir
	if !filepath.IsAbs(sourcePath) && absWorkDir != "" {
		sourcePath = filepath.Join(absWorkDir, sourcePath)
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
			// Convert relative cwd to absolute path
			if server.Cwd != "" {
				if filepath.IsAbs(server.Cwd) {
					claudeServer.Cwd = server.Cwd
				} else if absWorkDir != "" {
					claudeServer.Cwd = filepath.Join(absWorkDir, server.Cwd)
				} else {
					claudeServer.Cwd = server.Cwd
				}
			}
		case "http":
			// Convert HTTP to stdio using mcp-remote
			// Claude CLI doesn't support HTTP MCP servers natively
			claudeServer.Command = "npx"
			claudeServer.Args = []string{"-y", "mcp-remote", server.URL}
		default:
			// Assume local if type not specified
			claudeServer.Command = server.Command
			claudeServer.Args = server.Args
			// Convert relative cwd to absolute path
			if server.Cwd != "" {
				if filepath.IsAbs(server.Cwd) {
					claudeServer.Cwd = server.Cwd
				} else if absWorkDir != "" {
					claudeServer.Cwd = filepath.Join(absWorkDir, server.Cwd)
				} else {
					claudeServer.Cwd = server.Cwd
				}
			}
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
func ConvertMCPConfigForTask(mcpConfigPath, taskID, baseDir, workDir string) (string, error) {
	if mcpConfigPath == "" {
		return "", nil
	}

	// Create task-specific temp directory
	tempDir := filepath.Join(baseDir, "claude-mcp", taskID)

	return ConvertMCPConfig(mcpConfigPath, tempDir, workDir)
}

// CleanupMCPConfig removes the temporary MCP config file for a task.
func CleanupMCPConfig(taskID, baseDir string) error {
	tempDir := filepath.Join(baseDir, "claude-mcp", taskID)
	return os.RemoveAll(tempDir)
}

// GeminiMCPConfig represents the Gemini CLI MCP configuration format.
// This is the format expected by `gemini --mcp-config`
type GeminiMCPConfig struct {
	MCPServers map[string]GeminiMCPServer `json:"mcpServers"`
}

// GeminiMCPServer represents a server entry in Gemini CLI format.
type GeminiMCPServer struct {
	// For stdio transport (local commands)
	Command string            `json:"command,omitempty"`
	Args    []string          `json:"args,omitempty"`
	Env     map[string]string `json:"env,omitempty"`
	Cwd     string            `json:"cwd,omitempty"`

	// For HTTP transport
	URL     string `json:"url,omitempty"`
	HttpURL string `json:"httpUrl,omitempty"`
	Trust   bool   `json:"trust,omitempty"`
}

// ConvertMCPConfigForGemini converts Mesnada MCP config to Gemini CLI format.
func ConvertMCPConfigForGemini(mcpConfigPath, taskID, baseDir, workDir string) (string, error) {
	if mcpConfigPath == "" {
		return "", nil
	}

	// Handle @ prefix (file reference)
	sourcePath := mcpConfigPath
	if strings.HasPrefix(sourcePath, "@") {
		sourcePath = sourcePath[1:]
	}

	// Convert workDir to absolute path if it's relative
	absWorkDir := workDir
	if workDir != "" && !filepath.IsAbs(workDir) {
		var err error
		absWorkDir, err = filepath.Abs(workDir)
		if err != nil {
			return "", fmt.Errorf("failed to resolve workDir to absolute path: %w", err)
		}
	}

	// Resolve relative paths from workDir
	if !filepath.IsAbs(sourcePath) && absWorkDir != "" {
		sourcePath = filepath.Join(absWorkDir, sourcePath)
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

	// Convert to Gemini format
	geminiConfig := GeminiMCPConfig{
		MCPServers: make(map[string]GeminiMCPServer),
	}

	for name, server := range mesnadaConfig.MCPServers {
		geminiServer := GeminiMCPServer{
			Trust: true, // Auto-trust for automation
		}

		switch server.Type {
		case "local":
			// stdio transport
			geminiServer.Command = server.Command
			geminiServer.Args = server.Args
			// Convert relative cwd to absolute path
			if server.Cwd != "" {
				if filepath.IsAbs(server.Cwd) {
					geminiServer.Cwd = server.Cwd
				} else if absWorkDir != "" {
					geminiServer.Cwd = filepath.Join(absWorkDir, server.Cwd)
				} else {
					geminiServer.Cwd = server.Cwd
				}
			}
		case "http":
			// Convert HTTP to stdio using mcp-remote
			// Gemini CLI doesn't support HTTP MCP servers natively
			geminiServer.Command = "npx"
			geminiServer.Args = []string{"-y", "mcp-remote", server.URL}
		default:
			// Assume local if type not specified
			geminiServer.Command = server.Command
			geminiServer.Args = server.Args
			// Convert relative cwd to absolute path
			if server.Cwd != "" {
				if filepath.IsAbs(server.Cwd) {
					geminiServer.Cwd = server.Cwd
				} else if absWorkDir != "" {
					geminiServer.Cwd = filepath.Join(absWorkDir, server.Cwd)
				} else {
					geminiServer.Cwd = server.Cwd
				}
			}
		}

		geminiConfig.MCPServers[name] = geminiServer
	}

	// Create task-specific temp directory
	tempDir := filepath.Join(baseDir, "gemini-mcp", taskID)
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create temp dir: %w", err)
	}

	// Write to temp file
	outputPath := filepath.Join(tempDir, "gemini-mcp-config.json")
	outputData, err := json.MarshalIndent(geminiConfig, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal Gemini MCP config: %w", err)
	}

	if err := os.WriteFile(outputPath, outputData, 0644); err != nil {
		return "", fmt.Errorf("failed to write Gemini MCP config: %w", err)
	}

	return outputPath, nil
}

// OpenCodeMCPConfig represents the OpenCode.ai MCP configuration format.
type OpenCodeMCPConfig struct {
	MCP map[string]OpenCodeMCPServer `json:"mcp"`
}

// OpenCodeMCPServer represents a server entry in OpenCode.ai format.
type OpenCodeMCPServer struct {
	Type    string   `json:"type"` // "local" or "remote"
	Command string   `json:"command,omitempty"`
	Args    []string `json:"args,omitempty"`
	Env     []string `json:"env,omitempty"` // Array of "KEY=value" strings
	URL     string   `json:"url,omitempty"`
	Enabled bool     `json:"enabled,omitempty"`
}

// ConvertMCPConfigForOpenCode converts Mesnada MCP config to OpenCode.ai format.
func ConvertMCPConfigForOpenCode(mcpConfigPath, taskID, baseDir, workDir string) (string, error) {
	if mcpConfigPath == "" {
		return "", nil
	}

	// Handle @ prefix (file reference)
	sourcePath := mcpConfigPath
	if strings.HasPrefix(sourcePath, "@") {
		sourcePath = sourcePath[1:]
	}

	// Convert workDir to absolute path if it's relative
	absWorkDir := workDir
	if workDir != "" && !filepath.IsAbs(workDir) {
		var err error
		absWorkDir, err = filepath.Abs(workDir)
		if err != nil {
			return "", fmt.Errorf("failed to resolve workDir to absolute path: %w", err)
		}
	}

	// Resolve relative paths from workDir
	if !filepath.IsAbs(sourcePath) && absWorkDir != "" {
		sourcePath = filepath.Join(absWorkDir, sourcePath)
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

	// Convert to OpenCode format
	opencodeConfig := OpenCodeMCPConfig{
		MCP: make(map[string]OpenCodeMCPServer),
	}

	for name, server := range mesnadaConfig.MCPServers {
		opencodeServer := OpenCodeMCPServer{
			Enabled: true,
		}

		switch server.Type {
		case "local":
			opencodeServer.Type = "local"
			opencodeServer.Command = server.Command
			opencodeServer.Args = server.Args
		case "http":
			// Convert HTTP to stdio using mcp-remote
			// OpenCode CLI doesn't support HTTP MCP servers natively
			opencodeServer.Type = "local"
			opencodeServer.Command = "npx"
			opencodeServer.Args = []string{"-y", "mcp-remote", server.URL}
		default:
			// Assume local if type not specified
			opencodeServer.Type = "local"
			opencodeServer.Command = server.Command
			opencodeServer.Args = server.Args
		}

		opencodeConfig.MCP[name] = opencodeServer
	}

	// Create task-specific temp directory
	tempDir := filepath.Join(baseDir, "opencode-mcp", taskID)
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create temp dir: %w", err)
	}

	// Write to temp file
	outputPath := filepath.Join(tempDir, "opencode.json")
	outputData, err := json.MarshalIndent(opencodeConfig, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal OpenCode MCP config: %w", err)
	}

	if err := os.WriteFile(outputPath, outputData, 0644); err != nil {
		return "", fmt.Errorf("failed to write OpenCode MCP config: %w", err)
	}

	return outputPath, nil
}
