package server

import (
	"fmt"
	"html/template"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/sevir/mesnada/pkg/models"
)

type uiTaskRow struct {
	ID            string
	Status        models.TaskStatus
	StatusClass   string
	ProgressText  string
	WhenText      string
	WhenTitle     string
	Tags          []string
	PromptExcerpt string
}

type uiTasksVM struct {
	Tasks []uiTaskRow
}

type uiPanelVM struct {
	Task         *models.Task
	ProgressText string
	WhenText     string
	WhenTitle    string
	FinishedText  string
	FinishedTitle string
	DurationText  string
	TagsText     string
	Prompt       string
}

type uiLogVM struct {
	Log string
}

func (s *Server) getUITemplates() (*template.Template, error) {
	s.uiOnce.Do(func() {
		s.uiTpl, s.uiTplErr = template.ParseFiles(
			"ui/partials/tasks.html",
			"ui/partials/panel.html",
			"ui/partials/log.html",
		)
	})
	return s.uiTpl, s.uiTplErr
}

func (s *Server) handleUITasks(w http.ResponseWriter, r *http.Request) {
	_ = r.ParseForm()
	status := strings.TrimSpace(r.FormValue("status"))

	var statuses []models.TaskStatus
	if status != "" && status != "all" {
		statuses = []models.TaskStatus{models.TaskStatus(status)}
	}

	tasks, err := s.orchestrator.ListTasks(models.ListRequest{Status: statuses})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	vm := uiTasksVM{Tasks: make([]uiTaskRow, 0, len(tasks))}
	for _, t := range tasks {
		when := t.CreatedAt
		if t.StartedAt != nil {
			when = *t.StartedAt
		}

		progressText := "—"
		if t.Progress != nil {
			progressText = fmt.Sprintf("%d%%", t.Progress.Percentage)
		}

		vm.Tasks = append(vm.Tasks, uiTaskRow{
			ID:            t.ID,
			Status:        t.Status,
			StatusClass:   statusClass(t.Status),
			ProgressText:  progressText,
			WhenText:      when.Format("2006-01-02 15:04:05"),
			WhenTitle:     when.Format(time.RFC3339),
			Tags:          t.Tags,
			PromptExcerpt: truncate(stripTaskIDPrefix(t.Prompt), 100),
		})
	}

	tpl, err := s.getUITemplates()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_ = tpl.ExecuteTemplate(w, "tasks.html", vm)
}

func (s *Server) handleUIPanel(w http.ResponseWriter, r *http.Request) {
	_ = r.ParseForm()
	taskID := strings.TrimSpace(r.FormValue("task_id"))
	if taskID == "" {
		http.Error(w, "missing task_id", http.StatusBadRequest)
		return
	}

	task, err := s.orchestrator.GetTask(taskID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	when := task.CreatedAt
	if task.StartedAt != nil {
		when = *task.StartedAt
	}

	finishedText := "—"
	finishedTitle := ""
	durationText := "—"
	if task.CompletedAt != nil {
		finished := *task.CompletedAt
		finishedText = finished.Format("2006-01-02 15:04:05")
		finishedTitle = finished.Format(time.RFC3339)

		startForDuration := task.CreatedAt
		if task.StartedAt != nil {
			startForDuration = *task.StartedAt
		}
		d := finished.Sub(startForDuration).Round(time.Second)
		if d < 0 {
			d = 0
		}
		durationText = d.String()
	}

	progressText := "—"
	if task.Progress != nil {
		progressText = fmt.Sprintf("%d%%", task.Progress.Percentage)
	}

	tagsText := "—"
	if len(task.Tags) > 0 {
		tagsText = strings.Join(task.Tags, ", ")
	}

	vm := uiPanelVM{
		Task:         task,
		ProgressText: progressText,
		WhenText:     when.Format("2006-01-02 15:04:05"),
		WhenTitle:    when.Format(time.RFC3339),
		FinishedText:  finishedText,
		FinishedTitle: finishedTitle,
		DurationText:  durationText,
		TagsText:     tagsText,
		Prompt:       stripTaskIDPrefix(task.Prompt),
	}

	tpl, err := s.getUITemplates()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_ = tpl.ExecuteTemplate(w, "panel.html", vm)
}

func (s *Server) handleUILog(w http.ResponseWriter, r *http.Request) {
	_ = r.ParseForm()
	taskID := strings.TrimSpace(r.FormValue("task_id"))
	if taskID == "" {
		http.Error(w, "missing task_id", http.StatusBadRequest)
		return
	}

	task, err := s.orchestrator.GetTask(taskID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	logText := ""
	if task.LogFile != "" {
		logText = readLastBytes(task.LogFile, 1024*1024) // 1MB tail
	}
	if logText == "" {
		logText = task.Output
	}

	tpl, err := s.getUITemplates()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_ = tpl.ExecuteTemplate(w, "log.html", uiLogVM{Log: logText})
}

func (s *Server) handleUIPurge(w http.ResponseWriter, r *http.Request) {
	_ = r.ParseForm()
	taskID := strings.TrimSpace(r.FormValue("task_id"))
	if taskID == "" {
		http.Error(w, "missing task_id", http.StatusBadRequest)
		return
	}

	if err := s.orchestrator.Purge(taskID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Return refreshed list fragment.
	s.handleUITasks(w, r)
}

func statusClass(st models.TaskStatus) string {
	switch st {
	case models.TaskStatusPending:
		return "st-pending"
	case models.TaskStatusRunning:
		return "st-running"
	case models.TaskStatusCompleted:
		return "st-completed"
	case models.TaskStatusFailed:
		return "st-failed"
	case models.TaskStatusCancelled:
		return "st-cancelled"
	case models.TaskStatusPaused:
		return "st-paused"
	default:
		return ""
	}
}

func stripTaskIDPrefix(prompt string) string {
	p := strings.TrimSpace(prompt)
	if strings.HasPrefix(p, "You are the task_id:") {
		// Drop the first line and any subsequent blank lines.
		if idx := strings.Index(p, "\n"); idx >= 0 {
			p = strings.TrimSpace(p[idx+1:])
		}
	}
	return p
}

func truncate(s string, max int) string {
	if max <= 0 {
		return ""
	}
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	if max <= 1 {
		return string(r[:max])
	}
	return string(r[:max-1]) + "…"
}

func readLastBytes(path string, max int64) string {
	f, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer f.Close()

	st, err := f.Stat()
	if err != nil {
		return ""
	}

	size := st.Size()
	start := int64(0)
	if size > max {
		start = size - max
	}

	if _, err := f.Seek(start, io.SeekStart); err != nil {
		return ""
	}
	b, err := io.ReadAll(f)
	if err != nil {
		return ""
	}
	return string(b)
}
