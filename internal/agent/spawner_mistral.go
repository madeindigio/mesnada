// Package agent handles spawning and managing CLI agent processes.
package agent

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/sevir/mesnada/pkg/models"
)

// MistralSpawner manages Mistral Vibe CLI process spawning.
type MistralSpawner struct {
	logDir     string
	processes  map[string]*MistralProcess
	mu         sync.RWMutex
	onComplete func(task *models.Task)
}

// MistralProcess represents a running Mistral Vibe CLI process.
type MistralProcess struct {
	cmd      *exec.Cmd
	task     *models.Task
	output   *strings.Builder
	logFile  *os.File
	cancel   context.CancelFunc
	done     chan struct{}
	vibeHome string // Temp dir used as VIBE_HOME
}

// NewMistralSpawner creates a new Mistral Vibe CLI agent spawner.
func NewMistralSpawner(logDir string, onComplete func(task *models.Task)) *MistralSpawner {
	if logDir == "" {
		home, _ := os.UserHomeDir()
		logDir = filepath.Join(home, defaultLogDir)
	}
	if abs, err := filepath.Abs(logDir); err == nil {
		logDir = abs
	}
	os.MkdirAll(logDir, 0755)

	return &MistralSpawner{
		logDir:     logDir,
		processes:  make(map[string]*MistralProcess),
		onComplete: onComplete,
	}
}

// createVibeTempHome creates a temporary VIBE_HOME directory with a config.toml
// containing MCP server configuration. Returns the path to the temp dir.
// Returns empty string (no error) if neither mcpConfigPath nor model are provided.
func createVibeTempHome(mcpConfigPath, taskID, baseDir, workDir, model string) (string, error) {
	tempDir := filepath.Join(baseDir, "vibe-home", taskID)
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create temp vibe home: %w", err)
	}

	var buf bytes.Buffer

	// Set model if specified
	if model != "" {
		fmt.Fprintf(&buf, "active_model = %q\n", model)
	}

	// Disable telemetry and auto-update for non-interactive use
	buf.WriteString("enable_telemetry = false\n")
	buf.WriteString("enable_auto_update = false\n")

	// Add MCP server entries if a config is provided
	if mcpConfigPath != "" {
		sourcePath := mcpConfigPath
		if strings.HasPrefix(sourcePath, "@") {
			sourcePath = sourcePath[1:]
		}

		absWorkDir := workDir
		if workDir != "" && !filepath.IsAbs(workDir) {
			abs, err := filepath.Abs(workDir)
			if err != nil {
				return "", fmt.Errorf("failed to resolve workDir: %w", err)
			}
			absWorkDir = abs
		}

		if !filepath.IsAbs(sourcePath) && absWorkDir != "" {
			sourcePath = filepath.Join(absWorkDir, sourcePath)
		}

		data, err := os.ReadFile(sourcePath)
		if err != nil {
			return "", fmt.Errorf("failed to read MCP config: %w", err)
		}

		var mesnadaConfig MesnadaMCPConfig
		if err := json.Unmarshal(data, &mesnadaConfig); err != nil {
			return "", fmt.Errorf("failed to parse MCP config: %w", err)
		}

		for name, server := range mesnadaConfig.MCPServers {
			buf.WriteString("\n[[mcp_servers]]\n")
			fmt.Fprintf(&buf, "name = %q\n", name)

			switch server.Type {
			case "http":
				fmt.Fprintf(&buf, "transport = %q\n", "http")
				fmt.Fprintf(&buf, "url = %q\n", server.URL)
			default:
				// "local" or unset → stdio
				fmt.Fprintf(&buf, "transport = %q\n", "stdio")
				fmt.Fprintf(&buf, "command = %q\n", server.Command)
				if len(server.Args) > 0 {
					buf.WriteString("args = [")
					for i, arg := range server.Args {
						if i > 0 {
							buf.WriteString(", ")
						}
						fmt.Fprintf(&buf, "%q", arg)
					}
					buf.WriteString("]\n")
				}
				// Resolve cwd relative to workDir and write as env VIBE_HOME cannot set cwd per server,
				// so skip cwd (vibe uses the global workdir flag instead).
			}
		}
	}

	configPath := filepath.Join(tempDir, "config.toml")
	if err := os.WriteFile(configPath, buf.Bytes(), 0644); err != nil {
		return "", fmt.Errorf("failed to write vibe config.toml: %w", err)
	}

	return tempDir, nil
}

// Spawn starts a new Mistral Vibe CLI agent.
func (s *MistralSpawner) Spawn(ctx context.Context, task *models.Task) error {
	vibeHome, err := createVibeTempHome(task.MCPConfig, task.ID, s.logDir, task.WorkDir, task.Model)
	if err != nil {
		log.Printf("Warning: failed to create Vibe temp home for task %s: %v", task.ID, err)
		vibeHome = ""
	}

	args := s.buildArgs(task)

	log.Printf("Executing: vibe %v", args)

	procCtx, cancel := context.WithCancel(ctx)
	if task.Timeout > 0 {
		procCtx, cancel = context.WithTimeout(ctx, time.Duration(task.Timeout))
	}

	cmd := exec.CommandContext(procCtx, "vibe", args...)
	if task.WorkDir != "" {
		cmd.Dir = task.WorkDir
	}

	env := append(os.Environ(), "NO_COLOR=1")
	if vibeHome != "" {
		env = append(env, "VIBE_HOME="+vibeHome)
	}
	cmd.Env = env

	logFile, err := openOrCreateLogFile(s.logDir, task)
	if err != nil {
		cancel()
		return err
	}

	output := &strings.Builder{}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		cancel()
		logFile.Close()
		return fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		cancel()
		logFile.Close()
		return fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		cancel()
		logFile.Close()
		return fmt.Errorf("failed to start vibe: %w", err)
	}

	task.PID = cmd.Process.Pid
	now := time.Now()
	task.StartedAt = &now
	task.Status = models.TaskStatusRunning

	log.Printf(
		"task_event=started task_id=%s status=%s pid=%d log_file=%q work_dir=%q model=%q engine=mistral",
		task.ID,
		task.Status,
		task.PID,
		task.LogFile,
		task.WorkDir,
		task.Model,
	)

	proc := &MistralProcess{
		cmd:      cmd,
		task:     task,
		output:   output,
		logFile:  logFile,
		cancel:   cancel,
		done:     make(chan struct{}),
		vibeHome: vibeHome,
	}

	s.mu.Lock()
	s.processes[task.ID] = proc
	s.mu.Unlock()

	go s.captureOutput(proc, stdout, stderr)
	go s.waitForCompletion(proc)

	return nil
}

func (s *MistralSpawner) buildArgs(task *models.Task) []string {
	promptWithTaskID := fmt.Sprintf("You are the task_id: %s\n\n%s", task.ID, task.Prompt)

	args := []string{
		"--agent", "auto-approve",
		"--output", "text",
	}

	if task.WorkDir != "" {
		args = append(args, "--workdir", task.WorkDir)
	}

	args = append(args, task.ExtraArgs...)

	args = append(args, "--prompt", promptWithTaskID)

	task.Prompt = promptWithTaskID

	return args
}

func (s *MistralSpawner) captureOutput(proc *MistralProcess, stdout, stderr io.ReadCloser) {
	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		scanner := bufio.NewScanner(stdout)
		buf := make([]byte, 0, 64*1024)
		scanner.Buffer(buf, 1024*1024)

		for scanner.Scan() {
			line := scanner.Text()
			fmt.Fprintf(proc.logFile, "%s\n", line)
			if proc.output.Len() < maxOutputCapture {
				proc.output.WriteString(line)
				proc.output.WriteString("\n")
			}
		}
	}()

	go func() {
		defer wg.Done()
		scanner := bufio.NewScanner(stderr)
		buf := make([]byte, 0, 64*1024)
		scanner.Buffer(buf, 1024*1024)

		for scanner.Scan() {
			line := scanner.Text()
			fmt.Fprintf(proc.logFile, "[stderr] %s\n", line)
			if proc.output.Len() < maxOutputCapture {
				proc.output.WriteString("[stderr] ")
				proc.output.WriteString(line)
				proc.output.WriteString("\n")
			}
		}
	}()

	wg.Wait()
}

func (s *MistralSpawner) waitForCompletion(proc *MistralProcess) {
	defer close(proc.done)
	defer proc.logFile.Close()

	if proc.vibeHome != "" {
		defer os.RemoveAll(proc.vibeHome)
	}

	err := proc.cmd.Wait()

	now := time.Now()
	proc.task.CompletedAt = &now
	proc.task.Output = proc.output.String()
	proc.task.OutputTail = s.getTail(proc.output.String(), outputTailLines)

	explicitStop := proc.task.Status == models.TaskStatusCancelled || proc.task.Status == models.TaskStatusPaused

	if err != nil {
		if explicitStop {
			if exitErr, ok := err.(*exec.ExitError); ok {
				code := exitErr.ExitCode()
				proc.task.ExitCode = &code
			}
		} else {
			proc.task.Status = models.TaskStatusFailed
			proc.task.Error = err.Error()
			if exitErr, ok := err.(*exec.ExitError); ok {
				code := exitErr.ExitCode()
				proc.task.ExitCode = &code
			}
		}
	} else {
		if !explicitStop {
			proc.task.Status = models.TaskStatusCompleted
		}
		code := 0
		proc.task.ExitCode = &code
	}

	s.mu.Lock()
	delete(s.processes, proc.task.ID)
	s.mu.Unlock()

	if s.onComplete != nil {
		s.onComplete(proc.task)
	}
}

func (s *MistralSpawner) getTail(output string, lines int) string {
	allLines := strings.Split(output, "\n")
	if len(allLines) <= lines {
		return output
	}
	return strings.Join(allLines[len(allLines)-lines:], "\n")
}

// Cancel stops a running agent.
func (s *MistralSpawner) Cancel(taskID string) error {
	s.mu.RLock()
	proc, exists := s.processes[taskID]
	s.mu.RUnlock()

	if !exists {
		return fmt.Errorf("process not found: %s", taskID)
	}

	proc.cancel()

	if proc.cmd.Process != nil {
		proc.cmd.Process.Signal(syscall.SIGTERM)
		select {
		case <-proc.done:
		case <-time.After(5 * time.Second):
			proc.cmd.Process.Kill()
		}
	}

	proc.task.Status = models.TaskStatusCancelled
	return nil
}

// Pause stops a running agent without marking it as cancelled.
func (s *MistralSpawner) Pause(taskID string) error {
	s.mu.RLock()
	proc, exists := s.processes[taskID]
	s.mu.RUnlock()

	if !exists {
		return fmt.Errorf("process not found: %s", taskID)
	}

	proc.cancel()

	if proc.cmd.Process != nil {
		proc.cmd.Process.Signal(syscall.SIGTERM)
		select {
		case <-proc.done:
		case <-time.After(5 * time.Second):
			proc.cmd.Process.Kill()
		}
	}

	proc.task.Status = models.TaskStatusPaused
	return nil
}

// IsRunning checks if a task is currently running.
func (s *MistralSpawner) IsRunning(taskID string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, exists := s.processes[taskID]
	return exists
}

// Wait blocks until a task completes or context is cancelled.
func (s *MistralSpawner) Wait(ctx context.Context, taskID string) error {
	s.mu.RLock()
	proc, exists := s.processes[taskID]
	s.mu.RUnlock()

	if !exists {
		return nil
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-proc.done:
		return nil
	}
}

// RunningCount returns the number of currently running processes.
func (s *MistralSpawner) RunningCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.processes)
}

// Shutdown cancels all running processes.
func (s *MistralSpawner) Shutdown() {
	s.mu.Lock()
	procs := make([]*MistralProcess, 0, len(s.processes))
	for _, p := range s.processes {
		procs = append(procs, p)
	}
	s.mu.Unlock()

	for _, proc := range procs {
		proc.cancel()
		if proc.cmd.Process != nil {
			proc.cmd.Process.Signal(syscall.SIGTERM)
		}
	}

	for _, proc := range procs {
		select {
		case <-proc.done:
		case <-time.After(10 * time.Second):
			if proc.cmd.Process != nil {
				proc.cmd.Process.Kill()
			}
		}
	}
}
