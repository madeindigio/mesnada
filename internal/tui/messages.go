package tui

import (
	"github.com/sevir/mesnada/internal/orchestrator"
	"github.com/sevir/mesnada/pkg/models"
)

// TaskListUpdatedMsg is sent when the task list has been refreshed from the orchestrator.
type TaskListUpdatedMsg struct {
	Tasks []*models.Task
}

// TaskDetailMsg is sent when a specific task's detail is loaded.
type TaskDetailMsg struct {
	Task *models.Task
}

// TaskSelectedMsg is emitted when task list selection changes.
type TaskSelectedMsg struct {
	TaskID string
	Task   *models.Task
}

// TaskListTickMsg triggers task list refresh polling.
type TaskListTickMsg struct{}

// TaskDetailTickMsg triggers selected task detail refresh polling.
type TaskDetailTickMsg struct{}

// TaskLogTickMsg triggers selected task log refresh polling.
type TaskLogTickMsg struct{}

// TaskStatsTickMsg triggers stats refresh polling.
type TaskStatsTickMsg struct{}

// TaskLogUpdatedMsg is sent when selected task log is refreshed.
type TaskLogUpdatedMsg struct {
	TaskID     string
	Content    string
	NextOffset int64
}

// TaskStatsMsg carries current orchestrator stats.
type TaskStatsMsg struct {
	Stats orchestrator.Stats
}

type TaskActionType string

const (
	TaskActionPause  TaskActionType = "pause"
	TaskActionResume TaskActionType = "resume"
	TaskActionCancel TaskActionType = "cancel"
	TaskActionRetry  TaskActionType = "retry"
	TaskActionPurge  TaskActionType = "purge"
)

// TaskActionResultMsg is sent when a task action completes.
type TaskActionResultMsg struct {
	Action TaskActionType
	Task   *models.Task
}

// ErrorMsg is sent when an error occurs during an async operation.
type ErrorMsg struct {
	Err error
}

// ActionResultMsg reports completion for keyboard-triggered task actions.
type ActionResultMsg struct {
	Err error
}

// ExitRequestedMsg asks the root model to quit the TUI.
type ExitRequestedMsg struct{}
