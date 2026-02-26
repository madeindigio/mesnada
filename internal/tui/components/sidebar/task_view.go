package sidebar

import (
	"fmt"
	"strings"
	"time"

	"github.com/sevir/mesnada/internal/tui/markdown"
	"github.com/sevir/mesnada/internal/tui/styles"
	"github.com/sevir/mesnada/pkg/models"
)

const (
	maxLogLines = 80
)

func RenderTaskView(task *models.Task, width int, uiStyles *styles.Styles) string {
	if task == nil {
		return ""
	}

	if width < 20 {
		width = 20
	}

	var sb strings.Builder
	sb.WriteString(sectionHeader("Task", uiStyles))
	sb.WriteString(fmt.Sprintf("%s\n\n", task.ID))

	sb.WriteString(sectionHeader("Metadata", uiStyles))
	sb.WriteString(renderMetadata(task))
	sb.WriteString("\n\n")

	sb.WriteString(sectionHeader("Progress", uiStyles))
	sb.WriteString(renderProgress(task, width))
	sb.WriteString("\n\n")

	sb.WriteString(sectionHeader("Prompt", uiStyles))
	sb.WriteString(renderPrompt(task.Prompt, width))
	sb.WriteString("\n\n")

	if acp := renderACP(task); acp != "" {
		sb.WriteString(sectionHeader("ACP", uiStyles))
		sb.WriteString(acp)
		sb.WriteString("\n\n")
	}

	sb.WriteString(sectionHeader("Actions", uiStyles))
	sb.WriteString("P pause | R resume | X cancel | T retry | D purge")
	sb.WriteString("\n\n")

	sb.WriteString(sectionHeader("Output", uiStyles))
	sb.WriteString(renderOutput(task, width))

	return sb.String()
}

func sectionHeader(title string, uiStyles *styles.Styles) string {
	if uiStyles == nil {
		return title
	}
	return uiStyles.Sidebar.SectionTitle.Render(title)
}

func renderMetadata(task *models.Task) string {
	rows := []string{
		fmt.Sprintf("Status: %s", task.Status),
		fmt.Sprintf("Engine: %s", task.Engine),
		fmt.Sprintf("Model: %s", emptyFallback(task.Model, "-")),
		fmt.Sprintf("PID: %d", task.PID),
		fmt.Sprintf("Created: %s", task.CreatedAt.Format(time.RFC3339)),
		fmt.Sprintf("Started: %s", formatTime(task.StartedAt)),
		fmt.Sprintf("Completed: %s", formatTime(task.CompletedAt)),
		fmt.Sprintf("Workdir: %s", emptyFallback(task.WorkDir, "-")),
		fmt.Sprintf("Tags: %s", strings.Join(task.Tags, ", ")),
	}
	return strings.Join(rows, "\n")
}

func renderProgress(task *models.Task, width int) string {
	if task.Progress == nil {
		return "No progress reported"
	}
	p := task.Progress.Percentage
	if p < 0 {
		p = 0
	}
	if p > 100 {
		p = 100
	}
	barWidth := width - 10
	if barWidth < 10 {
		barWidth = 10
	}
	filled := barWidth * p / 100
	bar := strings.Repeat("█", filled) + strings.Repeat("░", barWidth-filled)
	return fmt.Sprintf("[%s] %3d%%\n%s", bar, p, emptyFallback(task.Progress.Description, "No description"))
}

func renderPrompt(prompt string, width int) string {
	rendered, err := markdown.Render(prompt, width)
	if err != nil {
		return prompt
	}
	return rendered
}

func renderACP(task *models.Task) string {
	if task.ACPStatus == nil && task.ACPSessionID == "" && task.ACPMode == "" {
		return ""
	}
	rows := []string{
		fmt.Sprintf("Session: %s", emptyFallback(task.ACPSessionID, "-")),
		fmt.Sprintf("Mode: %s", emptyFallback(task.ACPMode, "-")),
	}
	if task.ACPStatus != nil {
		rows = append(rows,
			fmt.Sprintf("Agent: %s", emptyFallback(task.ACPStatus.AgentName, "-")),
			fmt.Sprintf("Connected: %t", task.ACPStatus.IsConnected),
			fmt.Sprintf("Tool Calls: %d", task.ACPStatus.ToolCalls),
			fmt.Sprintf("Last Activity: %s", emptyFallback(task.ACPStatus.LastActivity, "-")),
		)
	}
	return strings.Join(rows, "\n")
}

func renderOutput(task *models.Task, width int) string {
	source := task.OutputTail
	if source == "" {
		source = task.Output
	}
	if source == "" && task.Error != "" {
		source = "Error:\n" + task.Error
	}
	if source == "" {
		return "(no output yet)"
	}
	return markdown.TruncateANSISafe(source, maxLogLines, width)
}

func formatTime(t *time.Time) string {
	if t == nil {
		return "-"
	}
	return t.Format(time.RFC3339)
}

func emptyFallback(s, fallback string) string {
	if strings.TrimSpace(s) == "" {
		return fallback
	}
	return s
}
