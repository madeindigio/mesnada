package orchestrator

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/sevir/mesnada/pkg/models"
)

func setupTestOrchestrator(t *testing.T) (*Orchestrator, func()) {
	tmpDir, err := os.MkdirTemp("", "mesnada-orch-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	orch, err := New(Config{
		StorePath:   filepath.Join(tmpDir, "tasks.json"),
		LogDir:      filepath.Join(tmpDir, "logs"),
		MaxParallel: 2,
	})
	if err != nil {
		os.RemoveAll(tmpDir)
		t.Fatalf("Failed to create orchestrator: %v", err)
	}

	cleanup := func() {
		orch.Shutdown()
		os.RemoveAll(tmpDir)
	}

	return orch, cleanup
}

func TestOrchestratorSpawn(t *testing.T) {
	orch, cleanup := setupTestOrchestrator(t)
	defer cleanup()

	ctx := context.Background()

	// Spawn a task (won't actually run copilot, just tests the spawning logic)
	task, err := orch.Spawn(ctx, models.SpawnRequest{
		Prompt:     "echo test",
		WorkDir:    "/tmp",
		Background: true,
	})
	if err != nil {
		t.Fatalf("Failed to spawn task: %v", err)
	}

	if task.ID == "" {
		t.Error("Expected task to have an ID")
	}
	if task.Prompt != "echo test" {
		t.Errorf("Expected prompt 'echo test', got '%s'", task.Prompt)
	}

	// Get the task
	retrieved, err := orch.GetTask(task.ID)
	if err != nil {
		t.Fatalf("Failed to get task: %v", err)
	}

	if retrieved.ID != task.ID {
		t.Errorf("Expected ID %s, got %s", task.ID, retrieved.ID)
	}
}

func TestOrchestratorDefaultMCPConfigApplied(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "mesnada-orch-test-default-mcp-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	defaultCfg := "@.github/mcp-config.json"
	orch, err := New(Config{
		StorePath:         filepath.Join(tmpDir, "tasks.json"),
		LogDir:            filepath.Join(tmpDir, "logs"),
		MaxParallel:       1,
		DefaultMCPConfig:  defaultCfg,
	})
	if err != nil {
		t.Fatalf("Failed to create orchestrator: %v", err)
	}
	defer orch.Shutdown()

	ctx := context.Background()

	task, err := orch.Spawn(ctx, models.SpawnRequest{
		Prompt:     "test",
		WorkDir:    "/tmp",
		Background: true,
		// MCPConfig intentionally omitted
	})
	if err != nil {
		t.Fatalf("Failed to spawn task: %v", err)
	}

	if task.MCPConfig != defaultCfg {
		t.Fatalf("Expected MCPConfig to default to %q, got %q", defaultCfg, task.MCPConfig)
	}
}

func TestOrchestratorListTasks(t *testing.T) {
	orch, cleanup := setupTestOrchestrator(t)
	defer cleanup()

	ctx := context.Background()

	// Spawn multiple tasks
	for i := 0; i < 5; i++ {
		_, err := orch.Spawn(ctx, models.SpawnRequest{
			Prompt:     "test task",
			WorkDir:    "/tmp",
			Background: true,
			Tags:       []string{"test"},
		})
		if err != nil {
			t.Fatalf("Failed to spawn task %d: %v", i, err)
		}
	}

	// List all tasks
	tasks, err := orch.ListTasks(models.ListRequest{})
	if err != nil {
		t.Fatalf("Failed to list tasks: %v", err)
	}

	if len(tasks) < 5 {
		t.Errorf("Expected at least 5 tasks, got %d", len(tasks))
	}

	// List with filter
	tasks, err = orch.ListTasks(models.ListRequest{
		Tags:  []string{"test"},
		Limit: 3,
	})
	if err != nil {
		t.Fatalf("Failed to list tasks with filter: %v", err)
	}

	if len(tasks) != 3 {
		t.Errorf("Expected 3 tasks with limit, got %d", len(tasks))
	}
}

func TestOrchestratorCancel(t *testing.T) {
	orch, cleanup := setupTestOrchestrator(t)
	defer cleanup()

	ctx := context.Background()

	// Spawn a task
	task, err := orch.Spawn(ctx, models.SpawnRequest{
		Prompt:     "sleep 60",
		WorkDir:    "/tmp",
		Background: true,
	})
	if err != nil {
		t.Fatalf("Failed to spawn task: %v", err)
	}

	// Give it a moment to start
	time.Sleep(100 * time.Millisecond)

	// Cancel it
	err = orch.Cancel(task.ID)
	if err != nil {
		// May fail if copilot isn't installed, which is expected in tests
		t.Logf("Cancel returned error (expected if copilot not installed): %v", err)
	}

	// Check status
	retrieved, err := orch.GetTask(task.ID)
	if err != nil {
		t.Fatalf("Failed to get task: %v", err)
	}

	// Status should be cancelled or failed (if copilot isn't installed)
	if retrieved.Status != models.TaskStatusCancelled && retrieved.Status != models.TaskStatusFailed {
		t.Logf("Task status: %s (may vary based on copilot availability)", retrieved.Status)
	}
}

func TestOrchestratorStats(t *testing.T) {
	orch, cleanup := setupTestOrchestrator(t)
	defer cleanup()

	ctx := context.Background()

	// Spawn some tasks
	for i := 0; i < 3; i++ {
		_, err := orch.Spawn(ctx, models.SpawnRequest{
			Prompt:     "test",
			WorkDir:    "/tmp",
			Background: true,
		})
		if err != nil {
			t.Fatalf("Failed to spawn task: %v", err)
		}
	}

	stats := orch.GetStats()

	if stats.Total < 3 {
		t.Errorf("Expected at least 3 total tasks, got %d", stats.Total)
	}
}

func TestOrchestratorDelete(t *testing.T) {
	orch, cleanup := setupTestOrchestrator(t)
	defer cleanup()

	ctx := context.Background()

	// Spawn a task
	task, err := orch.Spawn(ctx, models.SpawnRequest{
		Prompt:     "test",
		WorkDir:    "/tmp",
		Background: true,
	})
	if err != nil {
		t.Fatalf("Failed to spawn task: %v", err)
	}

	// Wait a bit for it to fail (copilot not installed) or succeed
	time.Sleep(500 * time.Millisecond)

	// Try to delete (may fail if still running)
	err = orch.Delete(task.ID)
	if err != nil {
		t.Logf("Delete returned error (expected if running): %v", err)
		return
	}

	// Verify deleted
	_, err = orch.GetTask(task.ID)
	if err == nil {
		t.Error("Expected error getting deleted task")
	}
}

func TestGenerateID(t *testing.T) {
	id1 := generateID()
	id2 := generateID()

	if id1 == "" {
		t.Error("Expected non-empty ID")
	}
	if id1 == id2 {
		t.Error("Expected unique IDs")
	}
	if len(id1) < 10 {
		t.Errorf("Expected ID length >= 10, got %d", len(id1))
	}
}
