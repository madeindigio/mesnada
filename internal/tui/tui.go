package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/sevir/mesnada/internal/config"
	"github.com/sevir/mesnada/internal/orchestrator"
	"github.com/sevir/mesnada/internal/tui/logcapture"
)

// Run starts the TUI application in alt-screen mode.
// It blocks until the user quits (q / Ctrl+C).
// All standard log output is captured and displayed in the log panel.
func Run(orch *orchestrator.Orchestrator, cfg *config.Config) error {
	logBuf := logcapture.New(500)
	logBuf.Install()
	defer logBuf.Restore()

	m := NewModel(orch, cfg, logBuf)
	p := tea.NewProgram(
		m,
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)
	_, err := p.Run()
	return err
}
