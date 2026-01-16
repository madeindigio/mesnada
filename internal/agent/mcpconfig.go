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

// GeminiMCPServer represents a server entry in Gemini CLI format.
// Gemini CLI expects mcpServers in the settings.json file.
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

// GeminiSettings represents the Gemini CLI settings format.
// This is the format expected by GEMINI_CLI_SYSTEM_SETTINGS_PATH
type GeminiSettings struct {
	MCPServers map[string]GeminiMCPServer `json:"mcpServers,omitempty"`
}

// CreateGeminiSettingsFile creates a temporary settings.json file with MCP configuration.
// The file path should be passed to Gemini CLI via GEMINI_CLI_SYSTEM_SETTINGS_PATH env var.
// Returns the path to the settings file (for cleanup), or empty if no config.
func CreateGeminiSettingsFile(mcpConfigPath, taskID, baseDir, workDir string) (string, error) {
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
	mcpServers := make(map[string]GeminiMCPServer)

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

		mcpServers[name] = geminiServer
	}

	// Create task-specific temp directory
	tempDir := filepath.Join(baseDir, "gemini-settings", taskID)
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create temp dir: %w", err)
	}

	// Create settings with MCP config
	settings := GeminiSettings{
		MCPServers: mcpServers,
	}

	// Write to temp settings.json file
	outputPath := filepath.Join(tempDir, "settings.json")
	outputData, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal Gemini settings: %w", err)
	}

	if err := os.WriteFile(outputPath, outputData, 0644); err != nil {
		return "", fmt.Errorf("failed to write Gemini settings: %w", err)
	}

	return outputPath, nil
}

// CleanupGeminiSettingsFile removes the temporary settings file created for a task.
func CleanupGeminiSettingsFile(settingsPath string) error {
	if settingsPath == "" {
		return nil
	}
	// Remove the file and the parent directory
	dir := filepath.Dir(settingsPath)
	return os.RemoveAll(dir)
}

// OpenCodeMCPConfig represents the OpenCode.ai MCP configuration format.
type OpenCodeMCPConfig struct {
	MCP map[string]OpenCodeMCPServer `json:"mcp"`
}

// OpenCodeMCPServer represents a server entry in OpenCode.ai format.
type OpenCodeMCPServer struct {
	Type        string            `json:"type"`              // "local" or "remote"
	Command     []string          `json:"command,omitempty"` // Array: [command, ...args]
	Environment map[string]string `json:"environment,omitempty"`
	URL         string            `json:"url,omitempty"`
	Enabled     bool              `json:"enabled,omitempty"`
	Timeout     int               `json:"timeout,omitempty"`
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
			Timeout: 5000, // Default timeout in ms
		}

		switch server.Type {
		case "local":
			opencodeServer.Type = "local"
			// Combine command and args into a single array
			opencodeServer.Command = append([]string{server.Command}, server.Args...)
		case "http":
			// Convert HTTP to stdio using mcp-remote
			// OpenCode CLI doesn't support HTTP MCP servers natively
			opencodeServer.Type = "local"
			opencodeServer.Command = []string{"npx", "-y", "mcp-remote", server.URL}
		default:
			// Assume local if type not specified
			opencodeServer.Type = "local"
			// Combine command and args into a single array
			opencodeServer.Command = append([]string{server.Command}, server.Args...)
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
