// Package agent handles spawning and managing CLI agent processes.
package agent

import (
	"context"
	"fmt"
	"sync"

	"github.com/sevir/mesnada/pkg/models"
)

// Manager coordinates multiple engine spawners.
type Manager struct {
	copilotSpawner  *CopilotSpawner
	claudeSpawner   *ClaudeSpawner
	geminiSpawner   *GeminiSpawner
	opencodeSpawner *OpenCodeSpawner
	taskEngines     map[string]models.Engine // Maps task ID to engine
	mu              sync.RWMutex
}

// NewManager creates a new agent manager.
func NewManager(logDir string, onComplete func(task *models.Task)) *Manager {
	return &Manager{
		copilotSpawner:  NewCopilotSpawner(logDir, onComplete),
		claudeSpawner:   NewClaudeSpawner(logDir, onComplete),
		geminiSpawner:   NewGeminiSpawner(logDir, onComplete),
		opencodeSpawner: NewOpenCodeSpawner(logDir, onComplete),
		taskEngines:     make(map[string]models.Engine),
	}
}

// Spawn starts a new agent using the appropriate engine.
func (m *Manager) Spawn(ctx context.Context, task *models.Task) error {
	engine := task.Engine
	if engine == "" {
		engine = models.DefaultEngine()
	}

	// Track which engine is handling this task
	m.mu.Lock()
	m.taskEngines[task.ID] = engine
	m.mu.Unlock()

	switch engine {
	case models.EngineClaude:
		return m.claudeSpawner.Spawn(ctx, task)
	case models.EngineGemini:
		return m.geminiSpawner.Spawn(ctx, task)
	case models.EngineOpenCode:
		return m.opencodeSpawner.Spawn(ctx, task)
	case models.EngineCopilot:
		return m.copilotSpawner.Spawn(ctx, task)
	default:
		return m.copilotSpawner.Spawn(ctx, task)
	}
}

// Cancel stops a running agent.
func (m *Manager) Cancel(taskID string) error {
	engine := m.getTaskEngine(taskID)

	switch engine {
	case models.EngineClaude:
		return m.claudeSpawner.Cancel(taskID)
	case models.EngineGemini:
		return m.geminiSpawner.Cancel(taskID)
	case models.EngineOpenCode:
		return m.opencodeSpawner.Cancel(taskID)
	default:
		return m.copilotSpawner.Cancel(taskID)
	}
}

// Pause stops a running agent without marking it as cancelled.
func (m *Manager) Pause(taskID string) error {
	engine := m.getTaskEngine(taskID)

	switch engine {
	case models.EngineClaude:
		return m.claudeSpawner.Pause(taskID)
	case models.EngineGemini:
		return m.geminiSpawner.Pause(taskID)
	case models.EngineOpenCode:
		return m.opencodeSpawner.Pause(taskID)
	default:
		return m.copilotSpawner.Pause(taskID)
	}
}

// Wait blocks until a task completes or context is cancelled.
func (m *Manager) Wait(ctx context.Context, taskID string) error {
	engine := m.getTaskEngine(taskID)

	switch engine {
	case models.EngineClaude:
		return m.claudeSpawner.Wait(ctx, taskID)
	case models.EngineGemini:
		return m.geminiSpawner.Wait(ctx, taskID)
	case models.EngineOpenCode:
		return m.opencodeSpawner.Wait(ctx, taskID)
	default:
		return m.copilotSpawner.Wait(ctx, taskID)
	}
}

// IsRunning checks if a task is currently running.
func (m *Manager) IsRunning(taskID string) bool {
	engine := m.getTaskEngine(taskID)

	switch engine {
	case models.EngineClaude:
		return m.claudeSpawner.IsRunning(taskID)
	case models.EngineGemini:
		return m.geminiSpawner.IsRunning(taskID)
	case models.EngineOpenCode:
		return m.opencodeSpawner.IsRunning(taskID)
	default:
		return m.copilotSpawner.IsRunning(taskID)
	}
}

// RunningCount returns the total number of currently running processes.
func (m *Manager) RunningCount() int {
	return m.copilotSpawner.RunningCount() +
		m.claudeSpawner.RunningCount() +
		m.geminiSpawner.RunningCount() +
		m.opencodeSpawner.RunningCount()
}

// Shutdown cancels all running processes.
func (m *Manager) Shutdown() {
	m.copilotSpawner.Shutdown()
	m.claudeSpawner.Shutdown()
	m.geminiSpawner.Shutdown()
	m.opencodeSpawner.Shutdown()
}

// getTaskEngine returns the engine used for a task.
func (m *Manager) getTaskEngine(taskID string) models.Engine {
	m.mu.RLock()
	defer m.mu.RUnlock()

	engine, exists := m.taskEngines[taskID]
	if !exists {
		return models.DefaultEngine()
	}
	return engine
}

// CleanupTask removes the engine tracking for a completed task.
func (m *Manager) CleanupTask(taskID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.taskEngines, taskID)
}

// GetProcess returns information about a running process (legacy support).
func (m *Manager) GetProcess(taskID string) (*Process, bool) {
	return m.copilotSpawner.GetProcess(taskID)
}

// ValidateEngine checks if an engine string is valid.
func ValidateEngine(engine string) error {
	e := models.Engine(engine)
	if e != "" && !models.ValidEngine(e) {
		return fmt.Errorf("invalid engine: %s (valid: copilot, claude, gemini, opencode)", engine)
	}
	return nil
}
