package server

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/sevir/mesnada/internal/orchestrator"
	"github.com/sevir/mesnada/pkg/models"
)

// Tool represents an MCP tool definition.
type Tool struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	InputSchema map[string]interface{} `json:"inputSchema"`
}

func (s *Server) registerTools() {
	s.tools["spawn_agent"] = s.toolSpawnAgent
	s.tools["get_task"] = s.toolGetTask
	s.tools["list_tasks"] = s.toolListTasks
	s.tools["wait_task"] = s.toolWaitTask
	s.tools["wait_multiple"] = s.toolWaitMultiple
	s.tools["cancel_task"] = s.toolCancelTask
	s.tools["pause_task"] = s.toolPauseTask
	s.tools["resume_task"] = s.toolResumeTask
	s.tools["delete_task"] = s.toolDeleteTask
	s.tools["get_stats"] = s.toolGetStats
	s.tools["get_task_output"] = s.toolGetTaskOutput
	s.tools["set_progress"] = s.toolSetProgress
}

func (s *Server) getToolDefinitions() []Tool {
	// Build model enum from configuration (all models from all engines + global)
	modelEnum := s.getAllModelIDs()

	return []Tool{
		{
			Name:        "spawn_agent",
			Description: "Spawn a new CLI agent to execute a task. Supports multiple engines: 'copilot' (GitHub Copilot CLI), 'claude' (Anthropic Claude CLI), 'gemini' (Google Gemini CLI), or 'opencode' (OpenCode.ai CLI). The agent runs in the specified working directory with full tool access. Use background=true for long-running tasks.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"prompt": map[string]interface{}{
						"type":        "string",
						"description": "The prompt/instruction for the agent to execute",
					},
					"work_dir": map[string]interface{}{
						"type":        "string",
						"description": "Working directory for the agent (absolute path)",
					},
					"engine": map[string]interface{}{
						"type":        "string",
						"description": "CLI engine to use: 'copilot' (GitHub Copilot CLI, default), 'claude' (Anthropic Claude CLI), 'gemini' (Google Gemini CLI), or 'opencode' (OpenCode.ai CLI)",
						"enum":        []string{"copilot", "claude", "gemini", "opencode"},
						"default":     "copilot",
					},
					"model": map[string]interface{}{
						"type":        "string",
						"description": "AI model to use (e.g., claude-sonnet-4, gpt-5.1-codex). Note: Model availability depends on the selected engine.",
						"enum":        modelEnum,
					},
					"background": map[string]interface{}{
						"type":        "boolean",
						"description": "Run in background (true) or wait for completion (false). Default: true",
						"default":     true,
					},
					"timeout": map[string]interface{}{
						"type":        "string",
						"description": "Timeout duration (e.g., '30m', '1h'). Empty for no timeout",
					},
					"dependencies": map[string]interface{}{
						"type":        "array",
						"items":       map[string]string{"type": "string"},
						"description": "List of task IDs that must complete before this task starts",
					},
					"tags": map[string]interface{}{
						"type":        "array",
						"items":       map[string]string{"type": "string"},
						"description": "Tags for organizing and filtering tasks",
					},
					"mcp_config": map[string]interface{}{
						"type":        "string",
						"description": "Additional MCP configuration JSON or file path (prefix with @). For Claude engine, this will be automatically converted to Claude CLI format.",
					},
					"extra_args": map[string]interface{}{
						"type":        "array",
						"items":       map[string]string{"type": "string"},
						"description": "Additional command-line arguments for the CLI engine",
					},
					"include_dependency_logs": map[string]interface{}{
						"type":        "boolean",
						"description": "Include logs from dependency tasks in the prompt. When true, the last N lines of logs from all dependency tasks will be added to the prompt with the header '===LAST TASK RESULTS==='",
						"default":     false,
					},
					"dependency_log_lines": map[string]interface{}{
						"type":        "integer",
						"description": "Number of lines to include from each dependency task log (default: 100)",
						"default":     100,
					},
				},
				"required": []string{"prompt"},
			},
		},
		{
			Name:        "get_task",
			Description: "Get detailed information about a specific task including status, output, and timing",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"task_id": map[string]interface{}{
						"type":        "string",
						"description": "The task ID to retrieve",
					},
				},
				"required": []string{"task_id"},
			},
		},
		{
			Name:        "list_tasks",
			Description: "List tasks with optional filtering by status and tags",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"status": map[string]interface{}{
						"type": "array",
						"items": map[string]interface{}{
							"type": "string",
							"enum": []string{"pending", "running", "paused", "completed", "failed", "cancelled"},
						},
						"description": "Filter by task status",
					},
					"tags": map[string]interface{}{
						"type":        "array",
						"items":       map[string]string{"type": "string"},
						"description": "Filter by tags (tasks must have all specified tags)",
					},
					"limit": map[string]interface{}{
						"type":        "integer",
						"description": "Maximum number of tasks to return",
						"default":     20,
					},
					"offset": map[string]interface{}{
						"type":        "integer",
						"description": "Number of tasks to skip",
						"default":     0,
					},
				},
			},
		},
		{
			Name:        "wait_task",
			Description: "Wait for a specific task to complete. Returns the task when it reaches a terminal state (completed, failed, or cancelled)",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"task_id": map[string]interface{}{
						"type":        "string",
						"description": "The task ID to wait for",
					},
					"timeout": map[string]interface{}{
						"type":        "string",
						"description": "Maximum time to wait (e.g., '5m', '1h'). Empty for no timeout",
					},
				},
				"required": []string{"task_id"},
			},
		},
		{
			Name:        "wait_multiple",
			Description: "Wait for multiple tasks to complete. Can wait for all tasks or return when any task completes",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"task_ids": map[string]interface{}{
						"type":        "array",
						"items":       map[string]string{"type": "string"},
						"description": "List of task IDs to wait for",
					},
					"wait_all": map[string]interface{}{
						"type":        "boolean",
						"description": "Wait for all tasks (true) or return when first completes (false)",
						"default":     true,
					},
					"timeout": map[string]interface{}{
						"type":        "string",
						"description": "Maximum time to wait (e.g., '10m', '1h')",
					},
				},
				"required": []string{"task_ids"},
			},
		},
		{
			Name:        "cancel_task",
			Description: "Cancel a running or pending task",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"task_id": map[string]interface{}{
						"type":        "string",
						"description": "The task ID to cancel",
					},
				},
				"required": []string{"task_id"},
			},
		},
		{
			Name:        "pause_task",
			Description: "Pause a running or pending task without marking it as cancelled",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"task_id": map[string]interface{}{
						"type":        "string",
						"description": "The task ID to pause",
					},
				},
				"required": []string{"task_id"},
			},
		},
		{
			Name:        "resume_task",
			Description: "Resume a paused task by spawning a new agent task that continues work",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"task_id": map[string]interface{}{
						"type":        "string",
						"description": "The paused task ID to resume",
					},
					"prompt": map[string]interface{}{
						"type":        "string",
						"description": "Additional resume prompt/instructions",
					},
					"model": map[string]interface{}{
						"type":        "string",
						"description": "AI model to use (optional; defaults to previous task model)",
						"enum":        modelEnum,
					},
					"background": map[string]interface{}{
						"type":        "boolean",
						"description": "Run in background (true) or wait for completion (false). Default: true",
						"default":     true,
					},
					"timeout": map[string]interface{}{
						"type":        "string",
						"description": "Timeout duration (e.g., '30m', '1h'). Empty for no timeout",
					},
					"tags": map[string]interface{}{
						"type":        "array",
						"items":       map[string]string{"type": "string"},
						"description": "Tags for organizing and filtering tasks (optional; defaults to previous task tags)",
					},
				},
				"required": []string{"task_id", "prompt"},
			},
		},
		{
			Name:        "delete_task",
			Description: "Delete a completed, failed, or cancelled task from the store",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"task_id": map[string]interface{}{
						"type":        "string",
						"description": "The task ID to delete",
					},
				},
				"required": []string{"task_id"},
			},
		},
		{
			Name:        "get_stats",
			Description: "Get orchestrator statistics including task counts by status",
			InputSchema: map[string]interface{}{
				"type":       "object",
				"properties": map[string]interface{}{},
			},
		},
		{
			Name:        "get_task_output",
			Description: "Get the output (stdout/stderr) of a task. For running tasks, returns current output. For completed tasks, returns full or tail output",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"task_id": map[string]interface{}{
						"type":        "string",
						"description": "The task ID",
					},
					"tail": map[string]interface{}{
						"type":        "boolean",
						"description": "Return only the last 50 lines (default: false for completed, true for running)",
					},
				},
				"required": []string{"task_id"},
			},
		},
		{
			Name:        "set_progress",
			Description: "Update the progress of a running task. This tool should be called by the agent task itself to report its progress. The percentage will be sanitized to be between 0 and 100.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"task_id": map[string]interface{}{
						"type":        "string",
						"description": "The task ID to update progress for",
					},
					"percentage": map[string]interface{}{
						"type":        "integer",
						"description": "Progress percentage (0-100). Any non-numeric characters will be stripped.",
					},
					"description": map[string]interface{}{
						"type":        "string",
						"description": "Brief description of current progress or what the task is currently doing",
					},
				},
				"required": []string{"task_id", "percentage"},
			},
		},
	}
}

func (s *Server) toolSpawnAgent(ctx context.Context, params json.RawMessage) (interface{}, error) {
	var req struct {
		Prompt                string   `json:"prompt"`
		WorkDir               string   `json:"work_dir"`
		Engine                string   `json:"engine"`
		Model                 string   `json:"model"`
		Background            *bool    `json:"background"`
		Timeout               string   `json:"timeout"`
		Dependencies          []string `json:"dependencies"`
		Tags                  []string `json:"tags"`
		MCPConfig             string   `json:"mcp_config"`
		ExtraArgs             []string `json:"extra_args"`
		IncludeDependencyLogs *bool    `json:"include_dependency_logs"`
		DependencyLogLines    *int     `json:"dependency_log_lines"`
	}

	if err := json.Unmarshal(params, &req); err != nil {
		return nil, fmt.Errorf("invalid parameters: %w", err)
	}

	if req.Prompt == "" {
		return nil, fmt.Errorf("prompt is required")
	}

	// Validate engine if provided
	engine := models.Engine(req.Engine)
	if req.Engine != "" && !models.ValidEngine(engine) {
		return nil, fmt.Errorf("invalid engine: %s (valid: copilot, claude, gemini, opencode)", req.Engine)
	}

	// Use default engine if not specified
	if engine == "" {
		engine = models.Engine(s.getDefaultEngine())
	}

	// Validate model for the specified engine
	if req.Model != "" && s.config != nil {
		if !s.config.ValidateModelForEngine(string(engine), req.Model) {
			availableModels := s.config.GetModelIDsForEngine(string(engine))
			return nil, fmt.Errorf("invalid model '%s' for engine '%s'. Available models: %v", req.Model, engine, availableModels)
		}
	}

	// Default to background execution
	background := true
	if req.Background != nil {
		background = *req.Background
	}

	// Default values for dependency logs
	includeDependencyLogs := false
	if req.IncludeDependencyLogs != nil {
		includeDependencyLogs = *req.IncludeDependencyLogs
	}

	dependencyLogLines := 100
	if req.DependencyLogLines != nil {
		dependencyLogLines = *req.DependencyLogLines
	}

	task, err := s.orchestrator.Spawn(ctx, models.SpawnRequest{
		Prompt:                req.Prompt,
		WorkDir:               req.WorkDir,
		Engine:                engine,
		Model:                 req.Model,
		Background:            background,
		Timeout:               req.Timeout,
		Dependencies:          req.Dependencies,
		Tags:                  req.Tags,
		MCPConfig:             req.MCPConfig,
		ExtraArgs:             req.ExtraArgs,
		IncludeDependencyLogs: includeDependencyLogs,
		DependencyLogLines:    dependencyLogLines,
	})

	if err != nil {
		return nil, err
	}

	result := map[string]interface{}{
		"task_id":    task.ID,
		"status":     task.Status,
		"engine":     task.Engine,
		"work_dir":   task.WorkDir,
		"created_at": task.CreatedAt,
	}

	if !background && task.IsTerminal() {
		result["output_tail"] = task.OutputTail
		result["exit_code"] = task.ExitCode
		if task.Error != "" {
			result["error"] = task.Error
		}
	}

	return result, nil
}

func (s *Server) toolGetTask(ctx context.Context, params json.RawMessage) (interface{}, error) {
	var req struct {
		TaskID string `json:"task_id"`
	}

	if err := json.Unmarshal(params, &req); err != nil {
		return nil, fmt.Errorf("invalid parameters: %w", err)
	}

	task, err := s.orchestrator.GetTask(req.TaskID)
	if err != nil {
		return nil, err
	}

	return task, nil
}

func (s *Server) toolListTasks(ctx context.Context, params json.RawMessage) (interface{}, error) {
	var req struct {
		Status []string `json:"status"`
		Tags   []string `json:"tags"`
		Limit  int      `json:"limit"`
		Offset int      `json:"offset"`
	}

	if err := json.Unmarshal(params, &req); err != nil {
		return nil, fmt.Errorf("invalid parameters: %w", err)
	}

	// Convert status strings to TaskStatus
	var statuses []models.TaskStatus
	for _, s := range req.Status {
		statuses = append(statuses, models.TaskStatus(s))
	}

	if req.Limit == 0 {
		req.Limit = 20
	}

	tasks, err := s.orchestrator.ListTasks(models.ListRequest{
		Status: statuses,
		Tags:   req.Tags,
		Limit:  req.Limit,
		Offset: req.Offset,
	})

	if err != nil {
		return nil, err
	}

	// Return summaries instead of full tasks
	summaries := make([]models.TaskSummary, len(tasks))
	for i, t := range tasks {
		summaries[i] = t.ToSummary()
	}

	return map[string]interface{}{
		"tasks": summaries,
		"total": len(summaries),
	}, nil
}

func (s *Server) toolWaitTask(ctx context.Context, params json.RawMessage) (interface{}, error) {
	var req struct {
		TaskID  string `json:"task_id"`
		Timeout string `json:"timeout"`
	}

	if err := json.Unmarshal(params, &req); err != nil {
		return nil, fmt.Errorf("invalid parameters: %w", err)
	}

	var timeout time.Duration
	if req.Timeout != "" {
		var err error
		timeout, err = time.ParseDuration(req.Timeout)
		if err != nil {
			return nil, fmt.Errorf("invalid timeout: %w", err)
		}
	}

	task, err := s.orchestrator.Wait(ctx, req.TaskID, timeout)
	if err != nil {
		// Still return task state even on timeout
		if task != nil {
			return map[string]interface{}{
				"task":    task,
				"error":   err.Error(),
				"timeout": true,
			}, nil
		}
		return nil, err
	}

	return map[string]interface{}{
		"task":        task,
		"output_tail": task.OutputTail,
	}, nil
}

func (s *Server) toolWaitMultiple(ctx context.Context, params json.RawMessage) (interface{}, error) {
	var req struct {
		TaskIDs []string `json:"task_ids"`
		WaitAll bool     `json:"wait_all"`
		Timeout string   `json:"timeout"`
	}

	if err := json.Unmarshal(params, &req); err != nil {
		return nil, fmt.Errorf("invalid parameters: %w", err)
	}

	var timeout time.Duration
	if req.Timeout != "" {
		var err error
		timeout, err = time.ParseDuration(req.Timeout)
		if err != nil {
			return nil, fmt.Errorf("invalid timeout: %w", err)
		}
	}

	results, err := s.orchestrator.WaitMultiple(ctx, req.TaskIDs, req.WaitAll, timeout)

	// Convert to response format
	taskResults := make(map[string]interface{})
	for id, task := range results {
		taskResults[id] = map[string]interface{}{
			"status":      task.Status,
			"output_tail": task.OutputTail,
			"exit_code":   task.ExitCode,
			"error":       task.Error,
		}
	}

	response := map[string]interface{}{
		"tasks":     taskResults,
		"completed": len(results),
		"requested": len(req.TaskIDs),
	}

	if err != nil {
		response["error"] = err.Error()
	}

	return response, nil
}

func (s *Server) toolCancelTask(ctx context.Context, params json.RawMessage) (interface{}, error) {
	var req struct {
		TaskID string `json:"task_id"`
	}

	if err := json.Unmarshal(params, &req); err != nil {
		return nil, fmt.Errorf("invalid parameters: %w", err)
	}

	if err := s.orchestrator.Cancel(req.TaskID); err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"task_id":   req.TaskID,
		"cancelled": true,
	}, nil
}

func (s *Server) toolPauseTask(ctx context.Context, params json.RawMessage) (interface{}, error) {
	var req struct {
		TaskID string `json:"task_id"`
	}

	if err := json.Unmarshal(params, &req); err != nil {
		return nil, fmt.Errorf("invalid parameters: %w", err)
	}

	task, err := s.orchestrator.Pause(req.TaskID)
	if err != nil {
		return nil, err
	}

	return task, nil
}

func (s *Server) toolResumeTask(ctx context.Context, params json.RawMessage) (interface{}, error) {
	var req struct {
		TaskID     string    `json:"task_id"`
		Prompt     string    `json:"prompt"`
		Model      string    `json:"model"`
		Background *bool     `json:"background"`
		Timeout    string    `json:"timeout"`
		Tags       *[]string `json:"tags"`
	}

	if err := json.Unmarshal(params, &req); err != nil {
		return nil, fmt.Errorf("invalid parameters: %w", err)
	}

	background := true
	if req.Background != nil {
		background = *req.Background
	}

	task, err := s.orchestrator.Resume(ctx, req.TaskID, orchestrator.ResumeOptions{
		Prompt:     req.Prompt,
		Model:      req.Model,
		Background: background,
		Timeout:    req.Timeout,
		Tags:       req.Tags,
	})
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"task_id": task.ID,
		"task":    task,
	}, nil
}

func (s *Server) toolDeleteTask(ctx context.Context, params json.RawMessage) (interface{}, error) {
	var req struct {
		TaskID string `json:"task_id"`
	}

	if err := json.Unmarshal(params, &req); err != nil {
		return nil, fmt.Errorf("invalid parameters: %w", err)
	}

	if err := s.orchestrator.Delete(req.TaskID); err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"task_id": req.TaskID,
		"deleted": true,
	}, nil
}

func (s *Server) toolGetStats(ctx context.Context, params json.RawMessage) (interface{}, error) {
	stats := s.orchestrator.GetStats()
	return stats, nil
}

func (s *Server) toolGetTaskOutput(ctx context.Context, params json.RawMessage) (interface{}, error) {
	var req struct {
		TaskID string `json:"task_id"`
		Tail   *bool  `json:"tail"`
	}

	if err := json.Unmarshal(params, &req); err != nil {
		return nil, fmt.Errorf("invalid parameters: %w", err)
	}

	task, err := s.orchestrator.GetTask(req.TaskID)
	if err != nil {
		return nil, err
	}

	// Determine whether to return tail or full output
	useTail := task.IsRunning() // Default to tail for running tasks
	if req.Tail != nil {
		useTail = *req.Tail
	}

	output := task.Output
	if useTail {
		output = task.OutputTail
	}

	return map[string]interface{}{
		"task_id":  task.ID,
		"status":   task.Status,
		"output":   output,
		"log_file": task.LogFile,
		"is_tail":  useTail,
	}, nil
}

func (s *Server) toolSetProgress(ctx context.Context, params json.RawMessage) (interface{}, error) {
	var req struct {
		TaskID      string      `json:"task_id"`
		Percentage  interface{} `json:"percentage"`
		Description string      `json:"description"`
	}

	if err := json.Unmarshal(params, &req); err != nil {
		return nil, fmt.Errorf("invalid parameters: %w", err)
	}

	// Sanitize percentage - extract numeric value
	percentage := 0
	switch v := req.Percentage.(type) {
	case float64:
		percentage = int(v)
	case int:
		percentage = v
	case string:
		// Strip any non-numeric characters except minus sign
		sanitized := ""
		for _, ch := range v {
			if ch >= '0' && ch <= '9' || (ch == '-' && len(sanitized) == 0) {
				sanitized += string(ch)
			}
		}
		if sanitized != "" {
			fmt.Sscanf(sanitized, "%d", &percentage)
		}
	default:
		return nil, fmt.Errorf("invalid percentage type: %T", v)
	}

	if err := s.orchestrator.SetProgress(req.TaskID, percentage, req.Description); err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"task_id":     req.TaskID,
		"percentage":  percentage,
		"description": req.Description,
		"updated":     true,
	}, nil
}

// getAllModelIDs returns a deduplicated list of all model IDs from config.
// This includes models from all engines plus the global models list.
func (s *Server) getAllModelIDs() []string {
	if s.config == nil {
		// Fallback to hardcoded list if no config
		return []string{
			"claude-sonnet-4.5", "claude-haiku-4.5", "claude-opus-4.5", "claude-sonnet-4",
			"gpt-5.1-codex-max", "gpt-5.1-codex", "gpt-5.2", "gpt-5.1", "gpt-5",
			"gpt-5.1-codex-mini", "gpt-5-mini", "gpt-4.1", "gemini-3-pro-preview",
		}
	}

	seen := make(map[string]bool)
	var modelIDs []string

	// Add global models
	for _, m := range s.config.Models {
		if !seen[m.ID] {
			seen[m.ID] = true
			modelIDs = append(modelIDs, m.ID)
		}
	}

	// Add engine-specific models
	if s.config.Engines != nil {
		for _, engineCfg := range s.config.Engines {
			for _, m := range engineCfg.Models {
				if !seen[m.ID] {
					seen[m.ID] = true
					modelIDs = append(modelIDs, m.ID)
				}
			}
		}
	}

	return modelIDs
}

// getDefaultEngine returns the default engine from config or fallback.
func (s *Server) getDefaultEngine() string {
	if s.config != nil && s.config.Orchestrator.DefaultEngine != "" {
		return s.config.Orchestrator.DefaultEngine
	}
	return "copilot"
}
