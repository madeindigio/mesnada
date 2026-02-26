// Package agent handles spawning and managing CLI agent processes.
package agent

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/sevir/mesnada/internal/config"
	"github.com/sevir/mesnada/pkg/models"
)

// TestACPRoutingWithConfig tests that ACP engines route correctly when ACP is enabled.
func TestACPRoutingWithConfig(t *testing.T) {
	cfg := &config.Config{
		ACP: config.ACPConfig{
			Enabled:      true,
			DefaultAgent: "test-agent",
			Agents: map[string]config.ACPAgentConfig{
				"test-agent": {
					Name:    "test-agent",
					Command: "/bin/false", // Command that will fail immediately
					Args:    []string{},
				},
				"claude-code": {
					Name:    "claude-code",
					Command: "/bin/false",
					Args:    []string{},
				},
			},
		},
	}

	manager := NewManager(cfg, t.TempDir(), nil)

	// Verify ACP spawner was initialized
	if manager.acpSpawner == nil {
		t.Fatal("Expected ACP spawner to be initialized when ACP is enabled")
	}

	// Test that ACP engine types route to ACP spawner
	testCases := []struct {
		engine      models.Engine
		description string
	}{
		{models.EngineACP, "generic ACP engine"},
		{models.EngineACPClaudeCode, "ACP Claude Code engine"},
		{models.EngineACPCodex, "ACP Codex engine"},
		{models.EngineACPCustom, "ACP Custom engine"},
		{"acp-myagent", "dynamic ACP engine"},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			task := &models.Task{
				ID:      "test-" + string(tc.engine),
				Prompt:  "test prompt",
				WorkDir: t.TempDir(),
				Engine:  tc.engine,
				Status:  models.TaskStatusPending,
			}

			ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
			defer cancel()

			err := manager.Spawn(ctx, task)

			// Spawn itself should succeed (spawner starts the process)
			// The process will fail, but spawn() returns nil initially
			if err != nil {
				// If there's an error, it should NOT be copilot-related
				if strings.Contains(strings.ToLower(err.Error()), "copilot") {
					t.Errorf("Task was routed to copilot instead of ACP spawner: %v", err)
				}
			}

			// Verify the task was tracked with correct engine
			manager.mu.RLock()
			trackedEngine := manager.taskEngines[task.ID]
			manager.mu.RUnlock()

			if trackedEngine != tc.engine {
				t.Errorf("Expected engine %s to be tracked, got %s", tc.engine, trackedEngine)
			}

			// Verify IsRunning uses the correct spawner
			// (acpSpawner tracks it, not copilotSpawner)
			isRunning := manager.IsRunning(task.ID)
			// The process may have already exited, but the point is it doesn't panic
			_ = isRunning
		})
	}
}

// TestACPRoutingWithoutConfig tests that ACP engines fail gracefully when ACP is disabled.
func TestACPRoutingWithoutConfig(t *testing.T) {
	cfg := &config.Config{
		ACP: config.ACPConfig{
			Enabled: false,
		},
	}

	manager := NewManager(cfg, t.TempDir(), nil)

	// Verify ACP spawner was NOT initialized
	if manager.acpSpawner != nil {
		t.Fatal("Expected ACP spawner to be nil when ACP is disabled")
	}

	testCases := []struct {
		engine      models.Engine
		description string
	}{
		{models.EngineACP, "generic ACP engine"},
		{models.EngineACPClaudeCode, "ACP Claude Code engine"},
		{"acp-custom", "dynamic ACP engine"},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			task := &models.Task{
				ID:      "test-" + string(tc.engine),
				Prompt:  "test prompt",
				WorkDir: t.TempDir(),
				Engine:  tc.engine,
				Status:  models.TaskStatusPending,
			}

			ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
			defer cancel()

			err := manager.Spawn(ctx, task)

			// Should get clear error that ACP is not enabled
			if err == nil {
				t.Errorf("Expected error when ACP is disabled, got nil")
			}

			if err != nil && !strings.Contains(strings.ToLower(err.Error()), "acp") {
				t.Errorf("Expected error message to mention ACP, got: %v", err)
			}

			if err != nil && !strings.Contains(strings.ToLower(err.Error()), "not enabled") {
				t.Errorf("Expected error message to mention 'not enabled', got: %v", err)
			}
		})
	}
}

// TestACPEngineValidation tests that dynamic ACP engines are validated correctly.
func TestACPEngineValidation(t *testing.T) {
	testCases := []struct {
		engine      string
		shouldPass  bool
		description string
	}{
		{"acp", true, "base ACP engine"},
		{"acp-claude", true, "ACP Claude engine"},
		{"acp-codex", true, "ACP Codex engine"},
		{"acp-custom", true, "ACP Custom engine"},
		{"acp-myagent", true, "dynamic ACP agent"},
		{"acp-test-123", true, "dynamic ACP with numbers"},
		{"copilot", true, "standard copilot engine"},
		{"claude", true, "standard claude engine"},
		{"invalid", false, "invalid engine"},
		{"acp_invalid", false, "ACP with underscore (should use dash)"},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			err := ValidateEngine(tc.engine)

			if tc.shouldPass && err != nil {
				t.Errorf("Expected engine '%s' to be valid, got error: %v", tc.engine, err)
			}

			if !tc.shouldPass && err == nil {
				t.Errorf("Expected engine '%s' to be invalid, got no error", tc.engine)
			}
		})
	}
}

// TestACPRunningCount tests that ACP processes are counted correctly.
func TestACPRunningCount(t *testing.T) {
	cfg := &config.Config{
		ACP: config.ACPConfig{
			Enabled:      true,
			DefaultAgent: "test-agent",
			Agents: map[string]config.ACPAgentConfig{
				"test-agent": {
					Name:    "test-agent",
					Command: "/bin/sleep",
					Args:    []string{"10"},
				},
			},
		},
	}

	manager := NewManager(cfg, t.TempDir(), nil)

	// Initial count should be 0
	if count := manager.RunningCount(); count != 0 {
		t.Errorf("Expected RunningCount to be 0, got %d", count)
	}

	// Note: We can't easily test actual running count without real ACP agents
	// This test mainly verifies that RunningCount doesn't panic when acpSpawner is present
}

// TestACPShutdown tests that shutdown includes ACP spawner.
func TestACPShutdown(t *testing.T) {
	cfg := &config.Config{
		ACP: config.ACPConfig{
			Enabled:      true,
			DefaultAgent: "test-agent",
			Agents: map[string]config.ACPAgentConfig{
				"test-agent": {
					Name:    "test-agent",
					Command: "/bin/echo",
					Args:    []string{"test"},
				},
			},
		},
	}

	manager := NewManager(cfg, t.TempDir(), nil)

	// Shutdown should not panic even with ACP spawner
	manager.Shutdown()
}

// TestNonACPEnginesUnaffected tests that non-ACP engines still work correctly.
func TestNonACPEnginesUnaffected(t *testing.T) {
	cfg := &config.Config{
		ACP: config.ACPConfig{
			Enabled:      true,
			DefaultAgent: "test-agent",
			Agents: map[string]config.ACPAgentConfig{
				"test-agent": {
					Name:    "test-agent",
					Command: "/bin/echo",
					Args:    []string{"test"},
				},
			},
		},
	}

	manager := NewManager(cfg, t.TempDir(), nil)

	// Test that standard engines still route to their original spawners
	standardEngines := []models.Engine{
		models.EngineClaude,
		models.EngineGemini,
		models.EngineOpenCode,
		models.EngineMistral,
		models.EngineCopilot,
	}

	for _, engine := range standardEngines {
		t.Run(string(engine), func(t *testing.T) {
			task := &models.Task{
				ID:      "test-" + string(engine),
				Prompt:  "test",
				WorkDir: t.TempDir(),
				Engine:  engine,
				Status:  models.TaskStatusPending,
			}

			ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
			defer cancel()

			err := manager.Spawn(ctx, task)

			// We expect errors (no real CLI tools), but should not be ACP-related
			if err != nil && strings.Contains(strings.ToLower(err.Error()), "acp") {
				t.Errorf("Non-ACP engine %s should not route to ACP spawner", engine)
			}
		})
	}
}
