package acp

import (
	"testing"
)

func TestACPAgentConfig(t *testing.T) {
	config := ACPAgentConfig{
		Name:    "test-agent",
		Title:   "Test Agent",
		Command: "claude-code",
		Args:    []string{"--mode", "code"},
		Env: map[string]string{
			"API_KEY": "test-key",
		},
		WorkDir: "/tmp/test",
		Mode:    "code",
		McpServers: []McpServerConfig{
			{
				Name:    "test-mcp",
				Command: "mcp-server",
				Args:    []string{"--port", "8080"},
				Type:    "stdio",
			},
		},
	}

	if config.Name != "test-agent" {
		t.Errorf("Expected Name to be 'test-agent', got '%s'", config.Name)
	}

	if config.Title != "Test Agent" {
		t.Errorf("Expected Title to be 'Test Agent', got '%s'", config.Title)
	}

	if config.Command != "claude-code" {
		t.Errorf("Expected Command to be 'claude-code', got '%s'", config.Command)
	}

	if len(config.Args) != 2 {
		t.Errorf("Expected 2 Args, got %d", len(config.Args))
	}

	if len(config.Env) != 1 {
		t.Errorf("Expected 1 Env variable, got %d", len(config.Env))
	}

	if config.WorkDir != "/tmp/test" {
		t.Errorf("Expected WorkDir to be '/tmp/test', got '%s'", config.WorkDir)
	}

	if config.Mode != "code" {
		t.Errorf("Expected Mode to be 'code', got '%s'", config.Mode)
	}

	if len(config.McpServers) != 1 {
		t.Errorf("Expected 1 McpServer, got %d", len(config.McpServers))
	}
}

func TestMcpServerConfig(t *testing.T) {
	tests := []struct {
		name   string
		config McpServerConfig
	}{
		{
			name: "stdio transport",
			config: McpServerConfig{
				Name:    "stdio-server",
				Command: "mcp-server",
				Args:    []string{"--stdio"},
				Type:    "stdio",
			},
		},
		{
			name: "http transport",
			config: McpServerConfig{
				Name: "http-server",
				Type: "http",
				URL:  "http://localhost:8080",
				Headers: map[string]string{
					"Authorization": "Bearer token",
				},
			},
		},
		{
			name: "sse transport",
			config: McpServerConfig{
				Name: "sse-server",
				Type: "sse",
				URL:  "http://localhost:9000/events",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.config.Name == "" {
				t.Error("Expected Name to be set")
			}

			switch tt.config.Type {
			case "stdio":
				if tt.config.Command == "" {
					t.Error("Expected Command to be set for stdio transport")
				}
			case "http", "sse":
				if tt.config.URL == "" {
					t.Error("Expected URL to be set for http/sse transport")
				}
			}
		})
	}
}

func TestSessionUpdateInfo(t *testing.T) {
	update := SessionUpdateInfo{
		TaskID:      "task-123",
		MessageText: "Processing request...",
		ToolCall: &ToolCallInfo{
			Name: "read_file",
			Arguments: map[string]interface{}{
				"path": "/tmp/test.txt",
			},
			Status: "started",
		},
		Plan:       "Step 1: Read file",
		StopReason: "",
	}

	if update.TaskID != "task-123" {
		t.Errorf("Expected TaskID to be 'task-123', got '%s'", update.TaskID)
	}

	if update.MessageText != "Processing request..." {
		t.Errorf("Expected MessageText to be 'Processing request...', got '%s'", update.MessageText)
	}

	if update.ToolCall == nil {
		t.Error("Expected ToolCall to be set")
	} else {
		if update.ToolCall.Name != "read_file" {
			t.Errorf("Expected ToolCall.Name to be 'read_file', got '%s'", update.ToolCall.Name)
		}

		if update.ToolCall.Status != "started" {
			t.Errorf("Expected ToolCall.Status to be 'started', got '%s'", update.ToolCall.Status)
		}

		if len(update.ToolCall.Arguments) != 1 {
			t.Errorf("Expected 1 argument, got %d", len(update.ToolCall.Arguments))
		}
	}

	if update.Plan != "Step 1: Read file" {
		t.Errorf("Expected Plan to be 'Step 1: Read file', got '%s'", update.Plan)
	}
}

func TestToolCallInfo(t *testing.T) {
	toolCall := ToolCallInfo{
		Name: "execute_command",
		Arguments: map[string]interface{}{
			"command": "ls -la",
			"cwd":     "/tmp",
		},
		Status: "completed",
		Result: "file1.txt\nfile2.txt",
	}

	if toolCall.Name != "execute_command" {
		t.Errorf("Expected Name to be 'execute_command', got '%s'", toolCall.Name)
	}

	if toolCall.Status != "completed" {
		t.Errorf("Expected Status to be 'completed', got '%s'", toolCall.Status)
	}

	if len(toolCall.Arguments) != 2 {
		t.Errorf("Expected 2 arguments, got %d", len(toolCall.Arguments))
	}

	if toolCall.Result == "" {
		t.Error("Expected Result to be set")
	}
}
