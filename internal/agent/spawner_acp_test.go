package agent

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/sevir/mesnada/internal/config"
	"github.com/sevir/mesnada/pkg/models"
)

func TestNewACPSpawner(t *testing.T) {
	tmpDir := t.TempDir()

	acpConfig := &config.ACPConfig{
		Enabled:      true,
		DefaultAgent: "test-agent",
		Agents: map[string]config.ACPAgentConfig{
			"test-agent": {
				Name:    "test-agent",
				Title:   "Test Agent",
				Command: "echo",
				Args:    []string{"test"},
			},
		},
	}

	spawner := NewACPSpawner(acpConfig, tmpDir, nil)

	if spawner == nil {
		t.Fatal("NewACPSpawner returned nil")
	}

	if spawner.config != acpConfig {
		t.Errorf("expected config to be set")
	}

	if spawner.logDir == "" {
		t.Errorf("expected logDir to be set")
	}

	if spawner.processes == nil {
		t.Errorf("expected processes map to be initialized")
	}
}

func TestACPSpawner_ResolveAgentConfig(t *testing.T) {
	acpConfig := &config.ACPConfig{
		Enabled:      true,
		DefaultAgent: "default-agent",
		Agents: map[string]config.ACPAgentConfig{
			"default-agent": {
				Name:    "default-agent",
				Command: "/usr/bin/default-agent",
			},
			"claude-code": {
				Name:    "claude-code",
				Command: "/usr/bin/claude-code-acp",
			},
			"codex": {
				Name:    "codex",
				Command: "/usr/bin/codex-acp",
			},
		},
	}

	spawner := NewACPSpawner(acpConfig, t.TempDir(), nil)

	tests := []struct {
		name          string
		task          *models.Task
		expectedAgent string
		expectNil     bool
	}{
		{
			name: "engine acp-claude",
			task: &models.Task{
				Engine: models.EngineACPClaudeCode,
			},
			expectedAgent: "claude-code",
		},
		{
			name: "engine acp-codex",
			task: &models.Task{
				Engine: models.EngineACPCodex,
			},
			expectedAgent: "codex",
		},
		{
			name: "engine acp with default",
			task: &models.Task{
				Engine: models.EngineACP,
			},
			expectedAgent: "default-agent",
		},
		{
			name: "unknown engine falls back to default",
			task: &models.Task{
				Engine: "unknown-engine",
			},
			expectedAgent: "default-agent", // Falls back to default agent
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := spawner.resolveAgentConfig(tt.task)
			if tt.expectNil {
				if config != nil {
					t.Errorf("expected nil config, got %+v", config)
				}
			} else {
				if config == nil {
					t.Fatal("expected non-nil config")
				}
				if config.Name != tt.expectedAgent {
					t.Errorf("expected agent %s, got %s", tt.expectedAgent, config.Name)
				}
			}
		})
	}
}

func TestACPSpawner_IsRunning(t *testing.T) {
	spawner := NewACPSpawner(&config.ACPConfig{}, t.TempDir(), nil)

	// Should return false for non-existent task
	if spawner.IsRunning("non-existent") {
		t.Error("expected IsRunning to return false for non-existent task")
	}
}

func TestACPSpawner_RunningCount(t *testing.T) {
	spawner := NewACPSpawner(&config.ACPConfig{}, t.TempDir(), nil)

	// Should return 0 initially
	if count := spawner.RunningCount(); count != 0 {
		t.Errorf("expected RunningCount to return 0, got %d", count)
	}
}

func TestACPSpawner_Cancel_NonExistent(t *testing.T) {
	spawner := NewACPSpawner(&config.ACPConfig{}, t.TempDir(), nil)

	// Should return error for non-existent task
	err := spawner.Cancel("non-existent")
	if err == nil {
		t.Error("expected error when cancelling non-existent task")
	}
}

func TestACPSpawner_Pause_NonExistent(t *testing.T) {
	spawner := NewACPSpawner(&config.ACPConfig{}, t.TempDir(), nil)

	// Should return error for non-existent task
	err := spawner.Pause("non-existent")
	if err == nil {
		t.Error("expected error when pausing non-existent task")
	}
}

func TestACPSpawner_Wait_NonExistent(t *testing.T) {
	spawner := NewACPSpawner(&config.ACPConfig{}, t.TempDir(), nil)

	// Should return immediately for non-existent task
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	err := spawner.Wait(ctx, "non-existent")
	if err != nil {
		t.Errorf("expected nil error when waiting for non-existent task, got %v", err)
	}
}

func TestACPSpawner_Spawn_NoAgent(t *testing.T) {
	// Create spawner with no agents configured
	spawner := NewACPSpawner(&config.ACPConfig{
		Enabled: true,
		Agents:  map[string]config.ACPAgentConfig{},
	}, t.TempDir(), nil)

	task := &models.Task{
		ID:      "test-task",
		Prompt:  "test prompt",
		WorkDir: t.TempDir(),
		Engine:  models.EngineACP,
	}

	// Should return error when no agent is configured
	err := spawner.Spawn(context.Background(), task)
	if err == nil {
		t.Error("expected error when spawning with no agent configured")
	}
}

func TestACPSpawner_Shutdown(t *testing.T) {
	spawner := NewACPSpawner(&config.ACPConfig{}, t.TempDir(), nil)

	// Should not panic when shutting down with no processes
	spawner.Shutdown()
}

func TestACPSpawner_GetTail(t *testing.T) {
	spawner := NewACPSpawner(&config.ACPConfig{}, t.TempDir(), nil)

	tests := []struct {
		name     string
		output   string
		lines    int
		expected string
	}{
		{
			name:     "empty output",
			output:   "",
			lines:    10,
			expected: "",
		},
		{
			name:     "fewer lines than requested",
			output:   "line1\nline2\nline3",
			lines:    10,
			expected: "line1\nline2\nline3",
		},
		{
			name:     "more lines than requested",
			output:   "line1\nline2\nline3\nline4\nline5",
			lines:    3,
			expected: "line3\nline4\nline5",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := spawner.getTail(tt.output, tt.lines)
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestACPSpawner_LogDir(t *testing.T) {
	// Test with empty logDir (should use default)
	spawner := NewACPSpawner(&config.ACPConfig{}, "", nil)
	if spawner.logDir == "" {
		t.Error("expected logDir to be set to default when empty string provided")
	}

	// Test with explicit logDir
	tmpDir := t.TempDir()
	spawner = NewACPSpawner(&config.ACPConfig{}, tmpDir, nil)

	// Check if logDir is absolute
	if !filepath.IsAbs(spawner.logDir) {
		t.Errorf("expected absolute path for logDir, got %s", spawner.logDir)
	}

	// Check if directory was created
	if _, err := os.Stat(spawner.logDir); os.IsNotExist(err) {
		t.Errorf("expected logDir to be created, but it doesn't exist: %s", spawner.logDir)
	}
}

func TestACPSpawner_OnComplete(t *testing.T) {
	callbackCalled := false
	onComplete := func(task *models.Task) {
		callbackCalled = true
	}

	spawner := NewACPSpawner(&config.ACPConfig{}, t.TempDir(), onComplete)

	if spawner.onComplete == nil {
		t.Error("expected onComplete callback to be set")
	}

	// Verify callback can be called
	spawner.onComplete(&models.Task{})
	if !callbackCalled {
		t.Error("expected onComplete callback to be called")
	}
}
