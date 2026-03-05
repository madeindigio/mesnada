package footer

import (
	tuictx "github.com/sevir/mesnada/internal/tui/context"
	"github.com/sevir/mesnada/internal/tui/styles"
)

const defaultHints = "↑/k ↓/j h/l g/G nav  a/r/c/F filters  f cycle  / search  ctrl+r refresh  P/R/X/T/D actions  tab sidebar  ctrl+l logs  ? help  q quit"

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

func (m *Model) View(err error) string {
	width := 0
	if m.ctx != nil {
		width = m.ctx.ScreenWidth
	}

	if err != nil {
		return m.styles.Footer.Error.Width(width).Render("Error: " + err.Error())
	}

	return m.styles.Footer.Root.Width(width).Render(defaultHints)
}
