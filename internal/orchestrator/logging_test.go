package orchestrator

import (
	bytes2 "bytes"
	context2 "context"
	"log"
	"strings"
	"testing"
	"time"

	"github.com/sevir/mesnada/pkg/models"
)

func captureStdLogger(t *testing.T) (*bytes2.Buffer, func()) {
	t.Helper()

	buf := &bytes2.Buffer{}
	prevOut := log.Writer()
	prevFlags := log.Flags()
	prevPrefix := log.Prefix()

	log.SetOutput(buf)
	log.SetFlags(0)
	log.SetPrefix("")

	return buf, func() {
		log.SetOutput(prevOut)
		log.SetFlags(prevFlags)
		log.SetPrefix(prevPrefix)
	}
}

func TestTaskLifecycleLogging_ReceivedAndStartable(t *testing.T) {
	orch, cleanup := setupTestOrchestrator(t)
	defer cleanup()

	buf, restore := captureStdLogger(t)
	defer restore()

	ctx := context2.Background()
	_, err := orch.Spawn(ctx, models.SpawnRequest{
		Prompt:     "echo hello",
		WorkDir:    "/tmp",
		Background: true,
		Tags:       []string{"test"},
	})
	if err != nil {
		t.Fatalf("Failed to spawn task: %v", err)
	}

	// Give the async start path a moment; the log assertions below only require
	// synchronous logs (received/startable) but this avoids flakiness with buffering.
	time.Sleep(20 * time.Millisecond)

	out := buf.String()
	if !strings.Contains(out, "task_event=received") {
		t.Fatalf("Expected received log entry, got:\n%s", out)
	}
	if !strings.Contains(out, "task_id=task-") {
		t.Fatalf("Expected task_id in logs, got:\n%s", out)
	}
	if !strings.Contains(out, "status=pending") {
		t.Fatalf("Expected pending status in logs, got:\n%s", out)
	}
	if !strings.Contains(out, "task_event=startable") {
		t.Fatalf("Expected startable log entry, got:\n%s", out)
	}
}

func TestTaskLifecycleLogging_StartableWhenDependenciesSatisfied(t *testing.T) {
	orch, cleanup := setupTestOrchestrator(t)
	defer cleanup()

	// Create a completed dependency in the store.
	dep := &models.Task{
		ID:        "task-dep-1",
		Status:    models.TaskStatusCompleted,
		CreatedAt: time.Now(),
	}
	if err := orch.store.Save(dep); err != nil {
		t.Fatalf("Failed to save dependency task: %v", err)
	}

	buf, restore := captureStdLogger(t)
	defer restore()

	ctx := context2.Background()
	_, err := orch.Spawn(ctx, models.SpawnRequest{
		Prompt:       "echo dependent",
		WorkDir:      "/tmp",
		Background:   true,
		Dependencies: []string{dep.ID},
	})
	if err != nil {
		t.Fatalf("Failed to spawn task: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "task_event=startable") {
		t.Fatalf("Expected startable log entry, got:\n%s", out)
	}
	if !strings.Contains(out, "dependencies=[task-dep-1]") {
		t.Fatalf("Expected dependencies to be logged, got:\n%s", out)
	}
}
