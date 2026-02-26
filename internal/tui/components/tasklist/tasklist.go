package tasklist

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	tuictx "github.com/sevir/mesnada/internal/tui/context"
	"github.com/sevir/mesnada/internal/tui/styles"
	"github.com/sevir/mesnada/pkg/models"
)

type FilterStatus string

const (
	FilterAll       FilterStatus = "all"
	FilterPending   FilterStatus = FilterStatus(models.TaskStatusPending)
	FilterRunning   FilterStatus = FilterStatus(models.TaskStatusRunning)
	FilterCompleted FilterStatus = FilterStatus(models.TaskStatusCompleted)
	FilterFailed    FilterStatus = FilterStatus(models.TaskStatusFailed)
	FilterCancelled FilterStatus = FilterStatus(models.TaskStatusCancelled)
	FilterPaused    FilterStatus = FilterStatus(models.TaskStatusPaused)
)

var filterOrder = []FilterStatus{
	FilterAll, FilterPending, FilterRunning, FilterCompleted, FilterFailed, FilterCancelled, FilterPaused,
}

type Model struct {
	ctx            *tuictx.Context
	styles         *styles.Styles
	tasks          []*models.Task
	filtered       []*models.Task
	filter         FilterStatus
	search         string
	selected       int
	offset         int
	runningSpinner string
}

func New(ctx *tuictx.Context, uiStyles *styles.Styles) *Model {
	return &Model{ctx: ctx, styles: uiStyles, filter: FilterAll}
}

func (m *Model) UpdateContext(ctx *tuictx.Context) {
	m.ctx = ctx
}

func (m *Model) SetTasks(tasks []*models.Task) bool {
	prev := m.SelectedTaskID()
	m.tasks = tasks
	m.applyFilter()
	return prev != m.SelectedTaskID()
}

func (m *Model) CycleFilter() bool {
	prev := m.SelectedTaskID()
	for i, value := range filterOrder {
		if m.filter == value {
			m.filter = filterOrder[(i+1)%len(filterOrder)]
			m.applyFilter()
			return prev != m.SelectedTaskID()
		}
	}
	m.filter = FilterAll
	m.applyFilter()
	return prev != m.SelectedTaskID()
}

func (m *Model) SetFilter(filter FilterStatus) bool {
	prev := m.SelectedTaskID()
	m.filter = filter
	m.applyFilter()
	return prev != m.SelectedTaskID()
}

func (m *Model) SetSearchQuery(query string) bool {
	prev := m.SelectedTaskID()
	m.search = strings.TrimSpace(query)
	m.applyFilter()
	return prev != m.SelectedTaskID()
}

func (m *Model) MoveUp() bool {
	if len(m.filtered) == 0 || m.selected == 0 {
		return false
	}
	m.selected--
	return true
}

func (m *Model) MoveDown() bool {
	if len(m.filtered) == 0 || m.selected >= len(m.filtered)-1 {
		return false
	}
	m.selected++
	return true
}

func (m *Model) MoveTop() bool {
	if len(m.filtered) == 0 || m.selected == 0 {
		return false
	}
	m.selected = 0
	return true
}

func (m *Model) MoveBottom() bool {
	if len(m.filtered) == 0 || m.selected == len(m.filtered)-1 {
		return false
	}
	m.selected = len(m.filtered) - 1
	return true
}

func (m *Model) SetRunningSpinner(frame string) {
	m.runningSpinner = strings.TrimSpace(frame)
}

func (m *Model) SelectedTask() *models.Task {
	if len(m.filtered) == 0 || m.selected < 0 || m.selected >= len(m.filtered) {
		return nil
	}
	return m.filtered[m.selected]
}

func (m *Model) SelectedTaskID() string {
	if task := m.SelectedTask(); task != nil {
		return task.ID
	}
	return ""
}

func (m *Model) View(width, height int) string {
	if width <= 0 || height <= 0 {
		return ""
	}

	header := m.renderHeader(width)
	if len(m.filtered) == 0 {
		empty := m.styles.Table.Cell.Render("  No tasks for filter: " + string(m.filter))
		return lipgloss.NewStyle().Width(width).Height(height).Render(header + "\n" + empty)
	}

	visibleRows := height - 2
	if visibleRows < 1 {
		visibleRows = 1
	}
	m.syncOffset(visibleRows)

	idW, engineW, statusW, progressW, tagsW, promptW := calcColumnWidths(width)
	var lines []string
	lines = append(lines, header)
	lines = append(lines, lipgloss.NewStyle().Foreground(m.styles.Theme.BorderFaint).Render(strings.Repeat("─", max(width-1, 1))))

	end := min(len(m.filtered), m.offset+visibleRows)
	for i := m.offset; i < end; i++ {
		task := m.filtered[i]
		progress := "-"
		if task.Progress != nil {
			progress = fmt.Sprintf("%d%%", task.Progress.Percentage)
		}
		tags := "-"
		if len(task.Tags) > 0 {
			tags = strings.Join(task.Tags, ",")
		}

		engineRaw := padRight(clean(string(task.Engine)), engineW)
		statusText := clean(string(task.Status))
		if task.Status == models.TaskStatusRunning && m.runningSpinner != "" {
			statusText = m.runningSpinner + " " + statusText
		}
		statusRaw := padRight(statusText, statusW)
		line := fmt.Sprintf("%s %-*s %s %s %-*s %-*s %s",
			selectionMarker(i == m.selected),
			idW, truncate(task.ID, idW),
			m.styles.EngineText(task.Engine, engineRaw),
			m.styles.StatusText(task.Status, statusRaw),
			progressW, truncate(progress, progressW),
			tagsW, truncate(tags, tagsW),
			truncate(clean(task.Prompt), promptW),
		)
		if i == m.selected {
			line = m.styles.Table.SelectedCell.Render(line)
		}
		lines = append(lines, line)
	}

	return lipgloss.NewStyle().Width(width).Height(height).Render(strings.Join(lines, "\n"))
}

func (m *Model) applyFilter() {
	m.filtered = m.filtered[:0]
	query := strings.ToLower(m.search)
	for _, task := range m.tasks {
		if m.filter != FilterAll && string(task.Status) != string(m.filter) {
			continue
		}
		if query != "" {
			haystack := strings.ToLower(task.ID + " " + task.Prompt + " " + strings.Join(task.Tags, " "))
			if !strings.Contains(haystack, query) {
				continue
			}
		}
		m.filtered = append(m.filtered, task)
	}

	if len(m.filtered) == 0 {
		m.selected = 0
		m.offset = 0
		return
	}
	if m.selected >= len(m.filtered) {
		m.selected = len(m.filtered) - 1
	}
	if m.selected < 0 {
		m.selected = 0
	}
	if m.offset > m.selected {
		m.offset = m.selected
	}
}

func (m *Model) renderHeader(width int) string {
	idW, engineW, statusW, progressW, tagsW, promptW := calcColumnWidths(width)
	label := m.styles.Table.FilterActive.Render("filter:" + string(m.filter))
	line := fmt.Sprintf("  %-*s %-*s %-*s %-*s %-*s %s  %s",
		idW, "ID",
		engineW, "ENGINE",
		statusW, "STATUS",
		progressW, "PROGRESS",
		tagsW, "TAGS",
		padRight("PROMPT", promptW),
		label,
	)
	return m.styles.Table.HeaderRow.Render(truncate(line, width))
}

func (m *Model) syncOffset(visibleRows int) {
	if m.selected < m.offset {
		m.offset = m.selected
	}
	if m.selected >= m.offset+visibleRows {
		m.offset = m.selected - visibleRows + 1
	}
	if m.offset < 0 {
		m.offset = 0
	}
}

func calcColumnWidths(totalWidth int) (idW, engineW, statusW, progressW, tagsW, promptW int) {
	idW, engineW, statusW, progressW, tagsW = 14, 12, 10, 8, 14
	fixed := idW + engineW + statusW + progressW + tagsW + 11
	promptW = totalWidth - fixed
	if promptW < 12 {
		promptW = 12
	}
	return
}

func selectionMarker(selected bool) string {
	if selected {
		return ">"
	}
	return " "
}

func padRight(s string, width int) string {
	if len(s) >= width {
		return s[:width]
	}
	return s + strings.Repeat(" ", width-len(s))
}

func truncate(s string, max int) string {
	if max <= 0 {
		return ""
	}
	if len(s) <= max {
		return s
	}
	if max == 1 {
		return "…"
	}
	return s[:max-1] + "…"
}

func clean(s string) string {
	return strings.ReplaceAll(strings.ReplaceAll(s, "\n", " "), "\r", " ")
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
