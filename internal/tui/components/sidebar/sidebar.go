package sidebar

import (
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	tuictx "github.com/sevir/mesnada/internal/tui/context"
	"github.com/sevir/mesnada/internal/tui/styles"
	"github.com/sevir/mesnada/pkg/models"
)

type Model struct {
	ctx        *tuictx.Context
	styles     *styles.Styles
	viewport   viewport.Model
	content    string
	task       *models.Task
	emptyMsg   string
	customView bool
}

func New(ctx *tuictx.Context, uiStyles *styles.Styles) *Model {
	vp := viewport.New(0, 0)
	m := &Model{
		ctx:      ctx,
		styles:   uiStyles,
		viewport: vp,
		emptyMsg: "\n  Select a task to view details\n  (Phase 4: sidebar implementation)",
	}
	m.resize()
	return m
}

func (m *Model) UpdateContext(ctx *tuictx.Context) {
	m.ctx = ctx
	m.resize()
	m.refreshContent()
}

func (m *Model) SetContent(content string) {
	m.content = content
	m.customView = content != ""
	m.refreshContent()
}

func (m *Model) SetTask(task *models.Task) {
	m.task = task
	m.customView = false
	m.refreshContent()
}

func (m *Model) Update(msg tea.Msg) tea.Cmd {
	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	return cmd
}

func (m *Model) View() string {
	width, height := 0, 0
	if m.ctx != nil {
		width = m.ctx.SidebarWidth
		height = m.ctx.MainContentHeight
	}

	if m.viewport.Width != max(0, width-2) || m.viewport.Height != height {
		m.resize()
		m.refreshContent()
	}

	return m.styles.Sidebar.Root.
		Width(width).
		Height(height).
		Render(m.viewport.View())
}

func (m *Model) resize() {
	width, height := 0, 0
	if m.ctx != nil {
		width = m.ctx.SidebarWidth - 2
		height = m.ctx.MainContentHeight
	}
	if width < 0 {
		width = 0
	}
	if height < 0 {
		height = 0
	}
	m.viewport.Width = width
	m.viewport.Height = height
}

func (m *Model) refreshContent() {
	if m.customView {
		m.viewport.SetContent(m.content)
		return
	}

	if m.task == nil {
		m.viewport.SetContent(m.styles.Sidebar.MetaKey.Render(m.emptyMsg))
		return
	}

	m.viewport.SetContent(RenderTaskView(m.task, m.viewport.Width, m.styles))
}
