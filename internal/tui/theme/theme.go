package theme

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/sevir/mesnada/pkg/models"
)

// Theme defines semantic color/icon tokens for the TUI.
type Theme struct {
	Background    lipgloss.AdaptiveColor
	BackgroundAlt lipgloss.AdaptiveColor
	Surface       lipgloss.AdaptiveColor

	BorderBrand  lipgloss.AdaptiveColor
	BorderFaint  lipgloss.AdaptiveColor
	BorderActive lipgloss.AdaptiveColor

	TextPrimary   lipgloss.AdaptiveColor
	TextSecondary lipgloss.AdaptiveColor
	TextFaint     lipgloss.AdaptiveColor

	Brand   lipgloss.AdaptiveColor
	Success lipgloss.AdaptiveColor
	Warning lipgloss.AdaptiveColor
	Error   lipgloss.AdaptiveColor
	Info    lipgloss.AdaptiveColor

	EngineColors map[models.Engine]lipgloss.AdaptiveColor
	StatusColors map[models.TaskStatus]lipgloss.AdaptiveColor
	StatusIcons  map[models.TaskStatus]string
}

// Default returns the default Mesnada TUI theme tokens.
func Default() Theme {
	brand := lipgloss.AdaptiveColor{Light: "#6d4ccd", Dark: "#7c5cff"}
	success := lipgloss.AdaptiveColor{Light: "#16a34a", Dark: "#22c55e"}
	warning := lipgloss.AdaptiveColor{Light: "#d97706", Dark: "#f59e0b"}
	errorColor := lipgloss.AdaptiveColor{Light: "#dc2626", Dark: "#ef4444"}
	info := lipgloss.AdaptiveColor{Light: "#2563eb", Dark: "#38bdf8"}
	muted := lipgloss.AdaptiveColor{Light: "#6b7280", Dark: "#98a6bf"}
	faint := lipgloss.AdaptiveColor{Light: "#4b5563", Dark: "#4a5568"}

	return Theme{
		Background:    lipgloss.AdaptiveColor{Light: "#f8fafc", Dark: "#0b0f17"},
		BackgroundAlt: lipgloss.AdaptiveColor{Light: "#e5e7eb", Dark: "#0f1624"},
		Surface:       lipgloss.AdaptiveColor{Light: "#f3f4f6", Dark: "#1a2235"},

		BorderBrand:  brand,
		BorderFaint:  lipgloss.AdaptiveColor{Light: "#d1d5db", Dark: "#1f2a3d"},
		BorderActive: info,

		TextPrimary:   lipgloss.AdaptiveColor{Light: "#1f2937", Dark: "#e8eef9"},
		TextSecondary: muted,
		TextFaint:     faint,

		Brand:   brand,
		Success: success,
		Warning: warning,
		Error:   errorColor,
		Info:    info,

		EngineColors: map[models.Engine]lipgloss.AdaptiveColor{
			models.EngineCopilot:        brand,
			models.EngineClaude:         lipgloss.AdaptiveColor{Light: "#b45309", Dark: "#d97706"},
			models.EngineGemini:         success,
			models.EngineACP:            info,
			models.EngineACPClaudeCode:  info,
			models.EngineACPCodex:       info,
			models.EngineACPCustom:      info,
			models.EngineMistral:        lipgloss.AdaptiveColor{Light: "#ea580c", Dark: "#f97316"},
			models.EngineOpenCode:       lipgloss.AdaptiveColor{Light: "#7c3aed", Dark: "#8b5cf6"},
			models.EngineOllamaClaude:   lipgloss.AdaptiveColor{Light: "#4f46e5", Dark: "#6366f1"},
			models.EngineOllamaOpenCode: lipgloss.AdaptiveColor{Light: "#4f46e5", Dark: "#6366f1"},
		},
		StatusColors: map[models.TaskStatus]lipgloss.AdaptiveColor{
			models.TaskStatusRunning:   info,
			models.TaskStatusCompleted: success,
			models.TaskStatusFailed:    errorColor,
			models.TaskStatusPaused:    warning,
			models.TaskStatusPending:   muted,
			models.TaskStatusCancelled: lipgloss.AdaptiveColor{Light: "#6b7280", Dark: "#6b7280"},
		},
		StatusIcons: map[models.TaskStatus]string{
			models.TaskStatusRunning:   "●",
			models.TaskStatusCompleted: "✓",
			models.TaskStatusFailed:    "✗",
			models.TaskStatusPaused:    "◉",
			models.TaskStatusPending:   "○",
			models.TaskStatusCancelled: "⊘",
		},
	}
}

func (t Theme) StatusColor(status models.TaskStatus) lipgloss.AdaptiveColor {
	if c, ok := t.StatusColors[status]; ok {
		return c
	}
	return t.TextSecondary
}

func (t Theme) EngineColor(engine models.Engine) lipgloss.AdaptiveColor {
	if c, ok := t.EngineColors[engine]; ok {
		return c
	}
	return t.TextSecondary
}

func (t Theme) StatusIcon(status models.TaskStatus) string {
	if i, ok := t.StatusIcons[status]; ok {
		return i
	}
	return "?"
}
