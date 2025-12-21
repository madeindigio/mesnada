package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/sevir/mesnada/pkg/models"
)

type listTasksResp struct {
	Tasks []struct {
		ID            string            `json:"id"`
		Status        models.TaskStatus `json:"status"`
		PromptExcerpt string            `json:"prompt_excerpt"`
		LogFile       string            `json:"log_file"`
		CreatedAt     string            `json:"created_at"`
	} `json:"tasks"`
}

type logResp struct {
	Content    string `json:"content"`
	NextOffset int64  `json:"next_offset"`
	Truncated  bool   `json:"truncated"`
}

func TestAPITasksList_FilterAndOrder(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	// Create a couple tasks via Spawn (store save path).
	t1, err := srv.orchestrator.Spawn(httptest.NewRequest("GET", "/", nil).Context(), models.SpawnRequest{Prompt: "p1", WorkDir: "/tmp", Background: false})
	if err != nil {
		t.Fatal(err)
	}
	t2, err := srv.orchestrator.Spawn(httptest.NewRequest("GET", "/", nil).Context(), models.SpawnRequest{Prompt: "p2", WorkDir: "/tmp", Background: false})
	if err != nil {
		t.Fatal(err)
	}

	// Deterministic ordering.
	now := time.Now()
	t1.CreatedAt = now.Add(-2 * time.Minute)
	t2.CreatedAt = now.Add(-1 * time.Minute)
	t1.Status = models.TaskStatusRunning
	t2.Status = models.TaskStatusFailed

	req := httptest.NewRequest("GET", "/api/tasks?status=running&status=failed", nil)
	w := httptest.NewRecorder()
	srv.httpServer.Handler.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 got %d", w.Code)
	}

	var resp listTasksResp
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if len(resp.Tasks) < 2 {
		t.Fatalf("expected at least 2 tasks")
	}
	for _, it := range resp.Tasks {
		if it.Status != models.TaskStatusRunning && it.Status != models.TaskStatusFailed {
			t.Fatalf("unexpected status %s", it.Status)
		}
		if it.PromptExcerpt == "" {
			t.Fatalf("missing prompt_excerpt")
		}
	}
	if resp.Tasks[0].ID != t2.ID {
		t.Fatalf("expected newest task first")
	}
}

func TestAPITaskLog_TailAndOffset(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	// create task with log file
	logPath := filepath.Join(os.TempDir(), "mesnada-api-log-test.log")
	_ = os.MkdirAll(filepath.Dir(logPath), 0755)
	_ = os.Remove(logPath)
	defer os.Remove(logPath)

	content1 := "hello\n"
	if err := os.WriteFile(logPath, []byte(content1), 0644); err != nil {
		t.Fatal(err)
	}
	_ = srv.orchestrator.Delete("task-log")
	// create task via spawn and point it to our temp log file
	task, err := srv.orchestrator.Spawn(httptest.NewRequest("GET", "/", nil).Context(), models.SpawnRequest{Prompt: "p", WorkDir: "/tmp", Background: false})
	if err != nil {
		t.Fatal(err)
	}
	// mutate in-memory entry in store
	task.LogFile = logPath

	// tail without offset
	req := httptest.NewRequest("GET", "/api/tasks/"+task.ID+"/log", nil)
	w := httptest.NewRecorder()
	srv.httpServer.Handler.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 got %d", w.Code)
	}
	var lr logResp
	if err := json.Unmarshal(w.Body.Bytes(), &lr); err != nil {
		t.Fatal(err)
	}
	if lr.Content != content1 {
		t.Fatalf("expected %q got %q", content1, lr.Content)
	}
	if lr.NextOffset != int64(len(content1)) {
		t.Fatalf("expected next_offset %d got %d", len(content1), lr.NextOffset)
	}

	// append and read from offset
	content2 := "world\n"
	f, _ := os.OpenFile(logPath, os.O_APPEND|os.O_WRONLY, 0644)
	f.Write([]byte(content2))
	f.Close()

	req2 := httptest.NewRequest("GET", "/api/tasks/"+task.ID+"/log?offset="+jsonNumber(lr.NextOffset), nil)
	w2 := httptest.NewRecorder()
	srv.httpServer.Handler.ServeHTTP(w2, req2)
	if w2.Code != http.StatusOK {
		t.Fatalf("expected 200 got %d", w2.Code)
	}
	var lr2 logResp
	if err := json.Unmarshal(w2.Body.Bytes(), &lr2); err != nil {
		t.Fatal(err)
	}
	if lr2.Content != content2 {
		t.Fatalf("expected %q got %q", content2, lr2.Content)
	}
}

func TestAPIPauseAndResumeTask(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	ctx := httptest.NewRequest("GET", "/", nil).Context()
	// Create a task that stays pending.
	task, err := srv.orchestrator.Spawn(ctx, models.SpawnRequest{Prompt: "p", WorkDir: "/tmp", Background: true, Dependencies: []string{"missing"}})
	if err != nil {
		t.Fatal(err)
	}
	// Ensure previous log path is referenced on resume.
	task.LogFile = "/tmp/mesnada-prev.log"

	// Pause
	req := httptest.NewRequest("POST", "/api/tasks/"+task.ID+"/pause", nil)
	w := httptest.NewRecorder()
	srv.httpServer.Handler.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 got %d", w.Code)
	}
	var pauseResp struct {
		Task models.Task `json:"task"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &pauseResp); err != nil {
		t.Fatal(err)
	}
	if pauseResp.Task.Status != models.TaskStatusPaused {
		t.Fatalf("expected paused got %s", pauseResp.Task.Status)
	}

	// Resume
	body := []byte(`{"prompt":"continue","background":true}`)
	req2 := httptest.NewRequest("POST", "/api/tasks/"+task.ID+"/resume", bytes.NewReader(body))
	req2.Header.Set("Content-Type", "application/json")
	w2 := httptest.NewRecorder()
	srv.httpServer.Handler.ServeHTTP(w2, req2)
	if w2.Code != http.StatusOK {
		t.Fatalf("expected 200 got %d", w2.Code)
	}
	var resumeResp struct {
		Task models.Task `json:"task"`
	}
	if err := json.Unmarshal(w2.Body.Bytes(), &resumeResp); err != nil {
		t.Fatal(err)
	}
	if resumeResp.Task.ID == task.ID {
		t.Fatalf("expected new task id")
	}
	if resumeResp.Task.Prompt == "" || !bytes.Contains([]byte(resumeResp.Task.Prompt), []byte(task.LogFile)) {
		t.Fatalf("expected resumed task prompt to reference previous log file")
	}
}

func TestAPIPurgeTask_TerminalAndMissingLogIdempotent(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	logPath := filepath.Join(os.TempDir(), "mesnada-api-purge-test.log")
	_ = os.Remove(logPath)
	if err := os.WriteFile(logPath, []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}

	task, err := srv.orchestrator.Spawn(httptest.NewRequest("GET", "/", nil).Context(), models.SpawnRequest{Prompt: "p", WorkDir: "/tmp", Background: false})
	if err != nil {
		t.Fatal(err)
	}
	id := task.ID
	task.Status = models.TaskStatusCompleted
	task.LogFile = logPath

	req := httptest.NewRequest("DELETE", "/api/tasks/"+id+"/purge", nil)
	w := httptest.NewRecorder()
	srv.httpServer.Handler.ServeHTTP(w, req)
	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204 got %d", w.Code)
	}
	if _, err := os.Stat(logPath); !os.IsNotExist(err) {
		t.Fatalf("expected log removed")
	}

	// second purge should still succeed even if task/log missing
	req2 := httptest.NewRequest("DELETE", "/api/tasks/"+id+"/purge", nil)
	w2 := httptest.NewRecorder()
	srv.httpServer.Handler.ServeHTTP(w2, req2)
	if w2.Code != http.StatusNoContent {
		t.Fatalf("expected 204 got %d", w2.Code)
	}
}

func jsonNumber(n int64) string {
	return strconv.FormatInt(n, 10)
}
