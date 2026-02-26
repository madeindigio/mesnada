package header

import (
	"fmt"

	"github.com/sevir/mesnada/internal/orchestrator"
	tuictx "github.com/sevir/mesnada/internal/tui/context"
	"github.com/sevir/mesnada/internal/tui/styles"
)

type Model struct {
	ctx    *tuictx.Context
	styles *styles.Styles
}

func New(ctx *tuictx.Context, uiStyles *styles.Styles) *Model {
	return &Model{ctx: ctx, styles: uiStyles}
}

func (m *Model) UpdateContext(ctx *tuictx.Context) {
	m.ctx = ctx
}

func (m *Model) View(stats orchestrator.Stats) string {
	addr := ""
	if m.ctx != nil && m.ctx.Config != nil {
		addr = m.ctx.Config.Address()
	}

	title := m.styles.Header.Title.Render("MESNADA")
	endpoint := m.styles.Header.EndpointDot.Render("● HTTP " + addr)

	taskCount := fmt.Sprintf("Tasks: %d running / %d total", stats.Running, stats.Total)
	taskInfo := m.styles.Header.StatsText.Render(taskCount)

	width := 0
	if m.ctx != nil {
		width = m.ctx.ScreenWidth
	}

	return m.styles.Header.Root.
		Width(width).
		Render(title + "  " + endpoint + "  " + taskInfo)
}
