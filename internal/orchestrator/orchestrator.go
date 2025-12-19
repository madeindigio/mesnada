// Package orchestrator coordinates agent tasks and dependencies.
package orchestrator

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/sevir/mesnada/internal/agent"
	"github.com/sevir/mesnada/internal/store"
	"github.com/sevir/mesnada/pkg/models"
)

// Orchestrator coordinates the execution of Copilot CLI agents.
type Orchestrator struct {
	store       store.Store
	spawner     *agent.Spawner
	subscribers map[string][]chan *models.Task
	subMu       sync.RWMutex
	maxParallel int
	wg          sync.WaitGroup
	ctx         context.Context
	cancel      context.CancelFunc
}

// Config holds orchestrator configuration.
type Config struct {
	StorePath   string
	LogDir      string
	MaxParallel int
}

// New creates a new Orchestrator.
func New(cfg Config) (*Orchestrator, error) {
	if cfg.MaxParallel <= 0 {
		cfg.MaxParallel = 5
	}

	fileStore, err := store.NewFileStore(cfg.StorePath)
	if err != nil {
		return nil, fmt.Errorf("failed to create store: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	o := &Orchestrator{
		store:       fileStore,
		subscribers: make(map[string][]chan *models.Task),
		maxParallel: cfg.MaxParallel,
		ctx:         ctx,
		cancel:      cancel,
	}

	o.spawner = agent.NewSpawner(cfg.LogDir, o.onTaskComplete)

	return o, nil
}

func (o *Orchestrator) onTaskComplete(task *models.Task) {
	// Save final state
	o.store.Save(task)

	// Notify subscribers
	o.subMu.RLock()
	subs := o.subscribers[task.ID]
	o.subMu.RUnlock()

	for _, ch := range subs {
		select {
		case ch <- task:
		default:
		}
	}

	// Clean up subscribers
	o.subMu.Lock()
	delete(o.subscribers, task.ID)
	o.subMu.Unlock()

	// Check for dependent tasks
	o.processDependentTasks(task)
}

func (o *Orchestrator) processDependentTasks(completed *models.Task) {
	if completed.Status != models.TaskStatusCompleted {
		return
	}

	// Find tasks waiting on this one
	tasks, _ := o.store.List(store.ListFilter{
		Status: []models.TaskStatus{models.TaskStatusPending},
	})

	for _, task := range tasks {
		if o.canStart(task) {
			go o.startTask(task)
		}
	}
}

func (o *Orchestrator) canStart(task *models.Task) bool {
	if len(task.Dependencies) == 0 {
		return true
	}

	for _, depID := range task.Dependencies {
		dep, err := o.store.Get(depID)
		if err != nil {
			return false
		}
		if dep.Status != models.TaskStatusCompleted {
			return false
		}
	}

	return true
}

func (o *Orchestrator) startTask(task *models.Task) {
	if err := o.spawner.Spawn(o.ctx, task); err != nil {
		task.Status = models.TaskStatusFailed
		task.Error = err.Error()
		now := time.Now()
		task.CompletedAt = &now
	}
	o.store.Save(task)
}

// Spawn creates and optionally starts a new agent task.
func (o *Orchestrator) Spawn(ctx context.Context, req models.SpawnRequest) (*models.Task, error) {
	// Validate work directory
	workDir := req.WorkDir
	if workDir == "" {
		workDir = "."
	}

	// Parse timeout
	var timeout models.Duration
	if req.Timeout != "" {
		dur, err := time.ParseDuration(req.Timeout)
		if err != nil {
			return nil, fmt.Errorf("invalid timeout: %w", err)
		}
		timeout = models.Duration(dur)
	}

	task := &models.Task{
		ID:           generateID(),
		Prompt:       req.Prompt,
		WorkDir:      workDir,
		Status:       models.TaskStatusPending,
		Model:        req.Model,
		Dependencies: req.Dependencies,
		Tags:         req.Tags,
		Priority:     req.Priority,
		Timeout:      timeout,
		MCPConfig:    req.MCPConfig,
		ExtraArgs:    req.ExtraArgs,
		CreatedAt:    time.Now(),
	}

	// Save task
	if err := o.store.Save(task); err != nil {
		return nil, fmt.Errorf("failed to save task: %w", err)
	}

	// Check if can start immediately
	if o.canStart(task) {
		if req.Background {
			go o.startTask(task)
		} else {
			o.startTask(task)
		}
	}

	return task, nil
}

// GetTask retrieves a task by ID.
func (o *Orchestrator) GetTask(taskID string) (*models.Task, error) {
	return o.store.Get(taskID)
}

// ListTasks lists tasks matching the filter.
func (o *Orchestrator) ListTasks(req models.ListRequest) ([]*models.Task, error) {
	return o.store.List(store.ListFilter{
		Status: req.Status,
		Tags:   req.Tags,
		Limit:  req.Limit,
		Offset: req.Offset,
	})
}

// Wait waits for a task to complete.
func (o *Orchestrator) Wait(ctx context.Context, taskID string, timeout time.Duration) (*models.Task, error) {
	// Check if already complete
	task, err := o.store.Get(taskID)
	if err != nil {
		return nil, err
	}

	if task.IsTerminal() {
		return task, nil
	}

	// Set up timeout context
	waitCtx := ctx
	if timeout > 0 {
		var cancel context.CancelFunc
		waitCtx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}

	// Subscribe to completion
	ch := make(chan *models.Task, 1)
	o.subMu.Lock()
	o.subscribers[taskID] = append(o.subscribers[taskID], ch)
	o.subMu.Unlock()

	defer func() {
		o.subMu.Lock()
		subs := o.subscribers[taskID]
		for i, sub := range subs {
			if sub == ch {
				o.subscribers[taskID] = append(subs[:i], subs[i+1:]...)
				break
			}
		}
		o.subMu.Unlock()
	}()

	// Also wait on spawner in case task completes between check and subscribe
	go func() {
		o.spawner.Wait(waitCtx, taskID)
		task, _ := o.store.Get(taskID)
		if task != nil && task.IsTerminal() {
			select {
			case ch <- task:
			default:
			}
		}
	}()

	select {
	case <-waitCtx.Done():
		// Return current state even on timeout
		task, _ = o.store.Get(taskID)
		return task, fmt.Errorf("timeout waiting for task %s: %w", taskID, waitCtx.Err())
	case task := <-ch:
		return task, nil
	}
}

// WaitMultiple waits for multiple tasks.
func (o *Orchestrator) WaitMultiple(ctx context.Context, taskIDs []string, waitAll bool, timeout time.Duration) (map[string]*models.Task, error) {
	results := make(map[string]*models.Task)
	var mu sync.Mutex
	var wg sync.WaitGroup

	waitCtx := ctx
	if timeout > 0 {
		var cancel context.CancelFunc
		waitCtx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}

	done := make(chan struct{})

	for _, id := range taskIDs {
		wg.Add(1)
		go func(taskID string) {
			defer wg.Done()

			task, err := o.Wait(waitCtx, taskID, 0)
			if task != nil {
				mu.Lock()
				results[taskID] = task
				mu.Unlock()
			}

			if !waitAll && err == nil && task != nil && task.IsTerminal() {
				select {
				case done <- struct{}{}:
				default:
				}
			}
		}(id)
	}

	if waitAll {
		wg.Wait()
	} else {
		select {
		case <-waitCtx.Done():
		case <-done:
		}
	}

	return results, nil
}

// Cancel cancels a running task.
func (o *Orchestrator) Cancel(taskID string) error {
	task, err := o.store.Get(taskID)
	if err != nil {
		return err
	}

	if task.IsTerminal() {
		return fmt.Errorf("task %s is already in terminal state: %s", taskID, task.Status)
	}

	if task.Status == models.TaskStatusRunning {
		if err := o.spawner.Cancel(taskID); err != nil {
			return err
		}
	}

	task.Status = models.TaskStatusCancelled
	now := time.Now()
	task.CompletedAt = &now

	return o.store.Save(task)
}

// Delete removes a task from the store.
func (o *Orchestrator) Delete(taskID string) error {
	task, err := o.store.Get(taskID)
	if err != nil {
		return err
	}

	if task.Status == models.TaskStatusRunning {
		return fmt.Errorf("cannot delete running task %s", taskID)
	}

	return o.store.Delete(taskID)
}

// SetProgress updates the progress of a running task.
func (o *Orchestrator) SetProgress(taskID string, percentage int, description string) error {
	task, err := o.store.Get(taskID)
	if err != nil {
		return err
	}

	// Sanitize percentage to be between 0 and 100
	if percentage < 0 {
		percentage = 0
	}
	if percentage > 100 {
		percentage = 100
	}

	task.Progress = &models.TaskProgress{
		Percentage:  percentage,
		Description: description,
		UpdatedAt:   time.Now(),
	}

	return o.store.Save(task)
}

// GetStats returns orchestrator statistics.
func (o *Orchestrator) GetStats() Stats {
	tasks, _ := o.store.List(store.ListFilter{})

	stats := Stats{
		Running:         o.spawner.RunningCount(),
		RunningProgress: make(map[string]TaskProgressInfo),
	}

	for _, task := range tasks {
		stats.Total++
		switch task.Status {
		case models.TaskStatusPending:
			stats.Pending++
		case models.TaskStatusRunning:
			// Add progress information for running tasks
			if task.Progress != nil {
				stats.RunningProgress[task.ID] = TaskProgressInfo{
					TaskID:      task.ID,
					Percentage:  task.Progress.Percentage,
					Description: task.Progress.Description,
					UpdatedAt:   task.Progress.UpdatedAt,
				}
			}
		case models.TaskStatusCompleted:
			stats.Completed++
		case models.TaskStatusFailed:
			stats.Failed++
		case models.TaskStatusCancelled:
			stats.Cancelled++
		}
	}

	return stats
}

// TaskProgressInfo holds progress information for a task.
type TaskProgressInfo struct {
	TaskID      string    `json:"task_id"`
	Percentage  int       `json:"percentage"`
	Description string    `json:"description"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// Stats holds orchestrator statistics.
type Stats struct {
	Total           int                         `json:"total"`
	Pending         int                         `json:"pending"`
	Running         int                         `json:"running"`
	Completed       int                         `json:"completed"`
	Failed          int                         `json:"failed"`
	Cancelled       int                         `json:"cancelled"`
	RunningProgress map[string]TaskProgressInfo `json:"running_progress,omitempty"`
}

// Shutdown gracefully shuts down the orchestrator.
func (o *Orchestrator) Shutdown() error {
	o.cancel()
	o.spawner.Shutdown()
	return o.store.Close()
}

func generateID() string {
	return fmt.Sprintf("task-%s", uuid.New().String()[:8])
}
