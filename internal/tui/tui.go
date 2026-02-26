package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/sevir/mesnada/internal/config"
	"github.com/sevir/mesnada/internal/orchestrator"
)

// Run starts the TUI application in alt-screen mode.
// It blocks until the user quits (q / Ctrl+C).
func Run(orch *orchestrator.Orchestrator, cfg *config.Config) error {
	m := NewModel(orch, cfg)
	p := tea.NewProgram(
		m,
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)
	_, err := p.Run()
	return err
}
