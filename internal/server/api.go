package server

import (
	"errors"
	"io"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/sevir/mesnada/pkg/models"
)

const defaultLogTailBytes = 64 * 1024

func (s *Server) registerAPI(mux *http.ServeMux) {
	r := gin.New()
	r.Use(gin.Recovery())

	api := r.Group("/api")
	api.GET("/tasks", s.apiListTasks)
	api.GET("/tasks/:id/log", s.apiTaskLog)
	api.DELETE("/tasks/:id/purge", s.apiPurgeTask)

	mux.Handle("/api/", r)
}

type apiTaskListItem struct {
	ID            string               `json:"id"`
	Status        models.TaskStatus    `json:"status"`
	Tags          []string             `json:"tags,omitempty"`
	Progress      *models.TaskProgress `json:"progress,omitempty"`
	CreatedAt     string               `json:"created_at"`
	StartedAt     *string              `json:"started_at,omitempty"`
	PromptExcerpt string               `json:"prompt_excerpt"`
	LogFile       string               `json:"log_file,omitempty"`
}

func (s *Server) apiListTasks(c *gin.Context) {
	statuses := c.QueryArray("status")
	var filter []models.TaskStatus
	for _, st := range statuses {
		if st == "" {
			continue
		}
		filter = append(filter, models.TaskStatus(st))
	}

	tasks, err := s.orchestrator.ListTasks(models.ListRequest{Status: filter})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Ensure newest-first by created_at for UI.
	sort.Slice(tasks, func(i, j int) bool { return tasks[i].CreatedAt.After(tasks[j].CreatedAt) })

	items := make([]apiTaskListItem, 0, len(tasks))
	for _, t := range tasks {
		created := t.CreatedAt.Format("2006-01-02T15:04:05Z07:00")
		var started *string
		if t.StartedAt != nil {
			s := t.StartedAt.Format("2006-01-02T15:04:05Z07:00")
			started = &s
		}
		items = append(items, apiTaskListItem{
			ID:            t.ID,
			Status:        t.Status,
			Tags:          t.Tags,
			Progress:      t.Progress,
			CreatedAt:     created,
			StartedAt:     started,
			PromptExcerpt: sanitizeExcerpt(truncateString(t.Prompt, 100)),
			LogFile:       t.LogFile,
		})
	}

	c.JSON(http.StatusOK, gin.H{"tasks": items})
}

type apiLogResponse struct {
	Content    string `json:"content"`
	NextOffset int64  `json:"next_offset"`
	Truncated  bool   `json:"truncated"`
}

func (s *Server) apiTaskLog(c *gin.Context) {
	id := c.Param("id")
	task, err := s.orchestrator.GetTask(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "task not found"})
		return
	}
	if task.LogFile == "" {
		c.JSON(http.StatusOK, apiLogResponse{Content: "", NextOffset: 0, Truncated: false})
		return
	}

	limit := int64(defaultLogTailBytes)
	if v := c.Query("limit"); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil && n > 0 {
			limit = n
		}
	}

	var (
		offset    *int64
		offsetVal int64
	)
	if v := c.Query("offset"); v != "" {
		n, err := strconv.ParseInt(v, 10, 64)
		if err != nil || n < 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid offset"})
			return
		}
		offsetVal = n
		offset = &offsetVal
	}

	content, next, truncated, err := readGrowingFile(task.LogFile, offset, limit)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			c.JSON(http.StatusOK, apiLogResponse{Content: "", NextOffset: 0, Truncated: false})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, apiLogResponse{Content: content, NextOffset: next, Truncated: truncated})
}

func readGrowingFile(path string, offset *int64, limit int64) (string, int64, bool, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", 0, false, err
	}
	defer f.Close()

	st, err := f.Stat()
	if err != nil {
		return "", 0, false, err
	}
	size := st.Size()

	var start int64
	truncated := false
	if offset == nil {
		if size > limit {
			start = size - limit
			truncated = true
		}
	} else {
		start = *offset
		if start > size {
			start = size
		}
	}

	if _, err := f.Seek(start, io.SeekStart); err != nil {
		return "", 0, false, err
	}

	buf := make([]byte, int(min64(limit, size-start)))
	n, err := io.ReadFull(f, buf)
	if err != nil {
		if !errors.Is(err, io.ErrUnexpectedEOF) && !errors.Is(err, io.EOF) {
			return "", 0, false, err
		}
	}

	content := string(buf[:n])
	return content, start + int64(n), truncated, nil
}

func min64(a, b int64) int64 {
	if a < b {
		return a
	}
	return b
}

func truncateString(s string, maxLen int) string {
	if maxLen <= 0 {
		return ""
	}
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen]
}

func (s *Server) apiPurgeTask(c *gin.Context) {
	id := c.Param("id")
	err := s.orchestrator.Purge(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}

// helper for tests/UI: removes aggressive whitespace from prompt excerpt.
func sanitizeExcerpt(s string) string {
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.ReplaceAll(s, "\r", " ")
	return strings.Join(strings.Fields(s), " ")
}
