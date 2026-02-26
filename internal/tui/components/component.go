package components

import tuictx "github.com/sevir/mesnada/internal/tui/context"

// ContextAware defines components that receive shared TUI context updates.
type ContextAware interface {
	UpdateContext(ctx *tuictx.Context)
}
