package models

import (
	"encoding/json"
	"testing"
	"time"
)

func TestTaskStatus(t *testing.T) {
	task := &Task{
		ID:     "test-1",
		Status: TaskStatusPending,
	}

	if !task.IsPending() {
		t.Error("Expected task to be pending")
	}
	if task.IsRunning() {
		t.Error("Expected task to not be running")
	}
	if task.IsTerminal() {
		t.Error("Expected task to not be terminal")
	}

	task.Status = TaskStatusRunning
	if task.IsPending() {
		t.Error("Expected task to not be pending")
	}
	if !task.IsRunning() {
		t.Error("Expected task to be running")
	}
	if task.IsTerminal() {
		t.Error("Expected task to not be terminal")
	}

	task.Status = TaskStatusCompleted
	if !task.IsTerminal() {
		t.Error("Expected task to be terminal")
	}

	task.Status = TaskStatusFailed
	if !task.IsTerminal() {
		t.Error("Expected task to be terminal")
	}

	task.Status = TaskStatusCancelled
	if !task.IsTerminal() {
		t.Error("Expected task to be terminal")
	}
}

func TestTaskToSummary(t *testing.T) {
	now := time.Now()
	later := now.Add(5 * time.Minute)

	task := &Task{
		ID:          "test-1",
		Prompt:      "Test prompt",
		WorkDir:     "/test/dir",
		Status:      TaskStatusCompleted,
		CreatedAt:   now,
		StartedAt:   &now,
		CompletedAt: &later,
	}

	summary := task.ToSummary()

	if summary.ID != task.ID {
		t.Errorf("Expected ID %s, got %s", task.ID, summary.ID)
	}
	if summary.Prompt != task.Prompt {
		t.Errorf("Expected Prompt %s, got %s", task.Prompt, summary.Prompt)
	}
	if summary.Duration != "5m0s" {
		t.Errorf("Expected Duration 5m0s, got %s", summary.Duration)
	}
}

func TestTaskToSummaryTruncatesLongPrompt(t *testing.T) {
	longPrompt := ""
	for i := 0; i < 150; i++ {
		longPrompt += "a"
	}

	task := &Task{
		ID:        "test-1",
		Prompt:    longPrompt,
		Status:    TaskStatusPending,
		CreatedAt: time.Now(),
	}

	summary := task.ToSummary()

	if len(summary.Prompt) > 100 {
		t.Errorf("Expected prompt to be truncated to 100 chars, got %d", len(summary.Prompt))
	}
	if summary.Prompt[len(summary.Prompt)-3:] != "..." {
		t.Error("Expected truncated prompt to end with ...")
	}
}

func TestDurationMarshalJSON(t *testing.T) {
	d := Duration(5 * time.Minute)

	data, err := json.Marshal(d)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	expected := `"5m0s"`
	if string(data) != expected {
		t.Errorf("Expected %s, got %s", expected, string(data))
	}
}

func TestDurationUnmarshalJSON(t *testing.T) {
	tests := []struct {
		input    string
		expected Duration
	}{
		{`"5m"`, Duration(5 * time.Minute)},
		{`"1h30m"`, Duration(90 * time.Minute)},
		{`"30s"`, Duration(30 * time.Second)},
		{`""`, Duration(0)},
	}

	for _, tt := range tests {
		var d Duration
		if err := json.Unmarshal([]byte(tt.input), &d); err != nil {
			t.Errorf("Failed to unmarshal %s: %v", tt.input, err)
			continue
		}
		if d != tt.expected {
			t.Errorf("For %s: expected %v, got %v", tt.input, tt.expected, d)
		}
	}
}

func TestSpawnRequest(t *testing.T) {
	req := SpawnRequest{
		Prompt:     "Fix the bug",
		WorkDir:    "/project",
		Model:      "claude-sonnet-4",
		Background: true,
		Timeout:    "30m",
		Tags:       []string{"bugfix", "urgent"},
	}

	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var decoded SpawnRequest
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if decoded.Prompt != req.Prompt {
		t.Errorf("Expected Prompt %s, got %s", req.Prompt, decoded.Prompt)
	}
	if decoded.Model != req.Model {
		t.Errorf("Expected Model %s, got %s", req.Model, decoded.Model)
	}
	if len(decoded.Tags) != len(req.Tags) {
		t.Errorf("Expected %d tags, got %d", len(req.Tags), len(decoded.Tags))
	}
}
