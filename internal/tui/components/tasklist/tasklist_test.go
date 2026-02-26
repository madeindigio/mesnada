package tasklist

import (
	"testing"

	"github.com/sevir/mesnada/internal/config"
	tuictx "github.com/sevir/mesnada/internal/tui/context"
	"github.com/sevir/mesnada/internal/tui/styles"
	"github.com/sevir/mesnada/internal/tui/theme"
	"github.com/sevir/mesnada/pkg/models"
)

func TestSetTasksAndFiltering(t *testing.T) {
	ctx := tuictx.New(&config.Config{})
	uiStyles := styles.InitStyles(theme.Default())
	m := New(ctx, &uiStyles)

	tasks := []*models.Task{
		{ID: "t-pending", Status: models.TaskStatusPending},
		{ID: "t-running", Status: models.TaskStatusRunning},
		{ID: "t-completed", Status: models.TaskStatusCompleted},
	}

	if changed := m.SetTasks(tasks); !changed {
		t.Fatalf("expected initial selection change")
	}
	if got := m.SelectedTaskID(); got != "t-pending" {
		t.Fatalf("expected selected task t-pending, got %q", got)
	}

	m.CycleFilter() // pending
	if got := m.SelectedTaskID(); got != "t-pending" {
		t.Fatalf("expected pending task selected, got %q", got)
	}

	m.CycleFilter() // running
	if got := m.SelectedTaskID(); got != "t-running" {
		t.Fatalf("expected running task selected, got %q", got)
	}

	if changed := m.SetFilter(FilterCompleted); !changed {
		t.Fatalf("expected selection change when setting completed filter")
	}
	if got := m.SelectedTaskID(); got != "t-completed" {
		t.Fatalf("expected completed task selected, got %q", got)
	}

	if changed := m.SetSearchQuery("pending"); !changed {
		t.Fatalf("expected selection change for empty completed search result")
	}
	if got := m.SelectedTaskID(); got != "" {
		t.Fatalf("expected no selected task for unmatched search, got %q", got)
	}

	m.SetFilter(FilterAll)
	if changed := m.SetSearchQuery("running"); !changed {
		t.Fatalf("expected selection change when search matches running")
	}
	if got := m.SelectedTaskID(); got != "t-running" {
		t.Fatalf("expected running task selected by search, got %q", got)
	}
}

func TestNavigation(t *testing.T) {
	ctx := tuictx.New(&config.Config{})
	uiStyles := styles.InitStyles(theme.Default())
	m := New(ctx, &uiStyles)
	_ = m.SetTasks([]*models.Task{
		{ID: "t1", Status: models.TaskStatusPending},
		{ID: "t2", Status: models.TaskStatusPending},
		{ID: "t3", Status: models.TaskStatusPending},
	})

	if !m.MoveDown() || m.SelectedTaskID() != "t2" {
		t.Fatalf("expected move down to t2, got %q", m.SelectedTaskID())
	}
	if !m.MoveBottom() || m.SelectedTaskID() != "t3" {
		t.Fatalf("expected move bottom to t3, got %q", m.SelectedTaskID())
	}
	if m.MoveDown() {
		t.Fatalf("expected move down at bottom to be false")
	}
	if !m.MoveTop() || m.SelectedTaskID() != "t1" {
		t.Fatalf("expected move top to t1, got %q", m.SelectedTaskID())
	}
}
