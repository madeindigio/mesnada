package logpanel

import (
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	tuictx "github.com/sevir/mesnada/internal/tui/context"
	"github.com/sevir/mesnada/internal/tui/logcapture"
	"github.com/sevir/mesnada/internal/tui/styles"
)

// Model is the log panel component that shows captured application logs.
// Hidden by default, toggled with a keybinding.
type Model struct {
	ctx      *tuictx.Context
	styles   *styles.Styles
	buf      *logcapture.Buffer
	viewport viewport.Model
	ready    bool
}

// New creates a new log panel component.
func New(ctx *tuictx.Context, uiStyles *styles.Styles, buf *logcapture.Buffer) *Model {
	return &Model{
		ctx:    ctx,
		styles: uiStyles,
		buf:    buf,
	}
}

// UpdateContext implements components.ContextAware.
func (m *Model) UpdateContext(ctx *tuictx.Context) {
	m.ctx = ctx
	if m.ready {
		m.viewport.Width = ctx.ScreenWidth
		m.viewport.Height = m.panelHeight()
	}
}

func (m *Model) panelHeight() int {
	if m.ctx == nil {
		return 0
	}
	return m.ctx.LogPanelHeight
}

// Update handles viewport scrolling messages.
func (m *Model) Update(msg tea.Msg) tea.Cmd {
	if !m.ready {
		return nil
	}
	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	return cmd
}

// Refresh updates the viewport content from the log buffer.
func (m *Model) Refresh() {
	if m.buf == nil {
		return
	}
	width := 0
	if m.ctx != nil {
		width = m.ctx.ScreenWidth - 2 // padding
	}
	height := m.panelHeight()

	if !m.ready {
		m.viewport = viewport.New(width, height)
		m.viewport.Style = lipgloss.NewStyle()
		m.ready = true
	} else {
		m.viewport.Width = width
		m.viewport.Height = height
	}

	content := m.buf.Content()
	m.viewport.SetContent(content)
	// Auto-scroll to bottom
	m.viewport.GotoBottom()
}

// View renders the log panel with a top border and title.
func (m *Model) View() string {
	if m.ctx == nil || !m.ctx.LogPanelOpen {
		return ""
	}

	width := m.ctx.ScreenWidth
	height := m.panelHeight()
	if width <= 0 || height <= 0 {
		return ""
	}

	m.Refresh()

	title := m.styles.Footer.KeyHint.Render(" LOGS ")
	lineLen := width - lipgloss.Width(title) - 1
	if lineLen < 0 {
		lineLen = 0
	}
	borderLine := lipgloss.NewStyle().
		Foreground(m.styles.Theme.BorderFaint).
		Render(strings.Repeat("─", lineLen))
	header := title + borderLine

	body := lipgloss.NewStyle().
		Width(width).
		Height(height).
		Padding(0, 1).
		Foreground(m.styles.Theme.TextSecondary).
		Render(m.viewport.View())

	return lipgloss.JoinVertical(lipgloss.Left, header, body)
}
