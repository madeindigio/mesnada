package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/sevir/mesnada/internal/config"
	"github.com/sevir/mesnada/pkg/models"
)

func TestKeyboardNavigationVimLike(t *testing.T) {
	m := testModelWithTasks()

	m = sendKey(m, runeKey('j'))
	if got := m.taskList.SelectedTaskID(); got != "t2" {
		t.Fatalf("expected selected t2 after j, got %q", got)
	}

	m = sendKey(m, runeKey('k'))
	if got := m.taskList.SelectedTaskID(); got != "t1" {
		t.Fatalf("expected selected t1 after k, got %q", got)
	}

	m = sendKey(m, runeKey('G'))
	if got := m.taskList.SelectedTaskID(); got != "t3" {
		t.Fatalf("expected selected t3 after G, got %q", got)
	}

	m = sendKey(m, runeKey('g'))
	if got := m.taskList.SelectedTaskID(); got != "t1" {
		t.Fatalf("expected selected t1 after g, got %q", got)
	}
}

func TestKeyRoutingHelpAndSearchPriority(t *testing.T) {
	m := testModelWithTasks()

	m = sendKey(m, runeKey('?'))
	if !m.showHelp {
		t.Fatalf("expected help overlay enabled")
	}

	m = sendKey(m, runeKey('j'))
	if got := m.taskList.SelectedTaskID(); got != "t1" {
		t.Fatalf("expected help mode to capture keys, got selected %q", got)
	}

	m = sendKey(m, runeKey('?'))
	if m.showHelp {
		t.Fatalf("expected help overlay disabled")
	}

	m = sendKey(m, runeKey('/'))
	if !m.searchMode {
		t.Fatalf("expected search mode enabled")
	}

	m = sendKey(m, runeKey('r'))
	if got := m.searchQuery; got != "r" {
		t.Fatalf("expected search query r, got %q", got)
	}

	m = sendKey(m, tea.KeyMsg{Type: tea.KeyEsc})
	if m.searchMode {
		t.Fatalf("expected search mode disabled")
	}
}

func TestConfirmYNForDestructiveActions(t *testing.T) {
	m := testModelWithTasks()

	m = sendKey(m, runeKey('X'))
	if m.confirmPending == nil || m.confirmPending.kind != "cancel" {
		t.Fatalf("expected cancel confirmation pending")
	}

	m = sendKey(m, runeKey('?'))
	if m.showHelp {
		t.Fatalf("expected confirm mode priority over help")
	}

	m = sendKey(m, runeKey('n'))
	if m.confirmPending != nil {
		t.Fatalf("expected confirmation dismissed with n")
	}

	m = sendKey(m, runeKey('D'))
	if m.confirmPending == nil || m.confirmPending.kind != "purge" {
		t.Fatalf("expected purge confirmation pending")
	}

	m = sendKey(m, runeKey('y'))
	if m.confirmPending != nil {
		t.Fatalf("expected confirmation cleared with y")
	}
}

func TestQuitRequiresConfirmationWhenRunningTasks(t *testing.T) {
	m := testModelWithTasks()

	var quit bool
	m, quit = sendKeyWithQuit(m, runeKey('q'))
	if quit {
		t.Fatalf("expected quit to require confirmation when running tasks exist")
	}
	if m.confirmPending == nil || m.confirmPending.kind != "quit" {
		t.Fatalf("expected quit confirmation pending")
	}

	m, quit = sendKeyWithQuit(m, runeKey('y'))
	if !quit {
		t.Fatalf("expected quit after confirming y")
	}
	if m.confirmPending != nil {
		t.Fatalf("expected quit confirmation cleared")
	}
}

func TestQuitWithoutRunningTasksExitsImmediately(t *testing.T) {
	m := testModelNoRunningTasks()

	_, quit := sendKeyWithQuit(m, runeKey('q'))
	if !quit {
		t.Fatalf("expected immediate quit when no running tasks exist")
	}
}

func testModelWithTasks() Model {
	m := NewModel(nil, config.DefaultConfig())
	mAny, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	m = mAny.(Model)
	mAny, _ = m.Update(TaskListUpdatedMsg{Tasks: []*models.Task{
		{ID: "t1", Status: models.TaskStatusPending, Prompt: "first"},
		{ID: "t2", Status: models.TaskStatusRunning, Prompt: "second"},
		{ID: "t3", Status: models.TaskStatusFailed, Prompt: "third"},
	}})
	return mAny.(Model)
}

func sendKey(m Model, msg tea.KeyMsg) Model {
	mAny, cmd := m.Update(msg)
	m = mAny.(Model)
	if cmd != nil {
		if follow := cmd(); follow != nil {
			mAny, _ = m.Update(follow)
			m = mAny.(Model)
		}
	}
	return m
}

func sendKeyWithQuit(m Model, msg tea.KeyMsg) (Model, bool) {
	mAny, cmd := m.Update(msg)
	m = mAny.(Model)
	if cmd == nil {
		return m, false
	}

	follow := cmd()
	if _, ok := follow.(tea.QuitMsg); ok {
		return m, true
	}
	if follow == nil {
		return m, false
	}

	mAny, cmd = m.Update(follow)
	m = mAny.(Model)
	if cmd == nil {
		return m, false
	}
	if _, ok := cmd().(tea.QuitMsg); ok {
		return m, true
	}
	return m, false
}

func testModelNoRunningTasks() Model {
	m := NewModel(nil, config.DefaultConfig())
	mAny, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	m = mAny.(Model)
	mAny, _ = m.Update(TaskListUpdatedMsg{Tasks: []*models.Task{
		{ID: "t1", Status: models.TaskStatusPending, Prompt: "first"},
		{ID: "t2", Status: models.TaskStatusCompleted, Prompt: "second"},
		{ID: "t3", Status: models.TaskStatusFailed, Prompt: "third"},
	}})
	return mAny.(Model)
}

func runeKey(r rune) tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}}
}
