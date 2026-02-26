package tui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/sevir/mesnada/internal/config"
	"github.com/sevir/mesnada/internal/orchestrator"
	"github.com/sevir/mesnada/internal/tui/components"
	"github.com/sevir/mesnada/internal/tui/components/footer"
	"github.com/sevir/mesnada/internal/tui/components/header"
	"github.com/sevir/mesnada/internal/tui/components/sidebar"
	"github.com/sevir/mesnada/internal/tui/components/tasklist"
	tuictx "github.com/sevir/mesnada/internal/tui/context"
	"github.com/sevir/mesnada/internal/tui/styles"
	"github.com/sevir/mesnada/internal/tui/theme"
	"github.com/sevir/mesnada/pkg/models"
)

const refreshInterval = 2 * time.Second

// Model is the root BubbleTea model for the mesnada TUI.
// Implements tea.Model (Init/Update/View) following the Elm architecture.
type Model struct {
	ctx            *TUIContext
	keys           *KeyMap
	showHelp       bool
	searchMode     bool
	searchQuery    string
	sidebarFocused bool
	confirmPending *pendingAction
	spinner        spinner.Model
	orch           *orchestrator.Orchestrator
	tasks          []*models.Task
	header         *header.Model
	footer         *footer.Model
	sidebar        *sidebar.Model
	taskList       *tasklist.Model
	contextAware   []components.ContextAware
	ready          bool
	err            error
	theme          theme.Theme
	styles         styles.Styles
}

type pendingAction struct {
	kind   string
	taskID string
}

// NewModel creates a new root TUI Model.
func NewModel(orch *orchestrator.Orchestrator, cfg *config.Config) Model {
	uiTheme := theme.Default()
	uiStyles := styles.InitStyles(uiTheme)

	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(uiTheme.Brand)

	ctx := newContext(cfg)
	headerComponent := header.New(ctx, &uiStyles)
	footerComponent := footer.New(ctx, &uiStyles)
	sidebarComponent := sidebar.New(ctx, &uiStyles)
	taskListComponent := tasklist.New(ctx, &uiStyles)

	return Model{
		ctx:      ctx,
		keys:     DefaultKeyMap(),
		spinner:  s,
		orch:     orch,
		header:   headerComponent,
		footer:   footerComponent,
		sidebar:  sidebarComponent,
		taskList: taskListComponent,
		theme:    uiTheme,
		styles:   uiStyles,
		contextAware: []components.ContextAware{
			headerComponent,
			footerComponent,
			sidebarComponent,
			taskListComponent,
		},
	}
}

// Init implements tea.Model.
func (m Model) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Tick,
		loadTasksCmd(m.orch),
		tickCmd(),
	)
}

func loadTasksCmd(orch *orchestrator.Orchestrator) tea.Cmd {
	return func() tea.Msg {
		if orch == nil {
			return TaskListUpdatedMsg{}
		}
		tasks, err := orch.ListTasks(models.ListRequest{})
		if err != nil {
			return ErrorMsg{Err: err}
		}
		return TaskListUpdatedMsg{Tasks: tasks}
	}
}

func tickCmd() tea.Cmd {
	return tea.Tick(refreshInterval, func(time.Time) tea.Msg {
		return TaskListTickMsg{}
	})
}

// Update implements tea.Model.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.ctx.SetWindowSize(msg.Width, msg.Height)
		m.propagateContext()
		m.ready = true

	case tea.KeyMsg:
		cmd, handled, quit := m.handleKeyMsg(msg)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
		if quit {
			return m, tea.Quit
		}
		if !handled && m.ctx.SidebarOpen {
			if cmd := m.sidebar.Update(msg); cmd != nil {
				cmds = append(cmds, cmd)
			}
		}

	case TaskListUpdatedMsg:
		m.tasks = msg.Tasks
		m.err = nil
		if m.taskList.SetTasks(msg.Tasks) {
			cmds = append(cmds, m.selectedTaskCmd())
		}
		if m.searchQuery != "" && m.taskList.SetSearchQuery(m.searchQuery) {
			cmds = append(cmds, m.selectedTaskCmd())
		}

	case TaskSelectedMsg:
		m.ctx.SelectedTaskID = msg.TaskID
		if msg.Task != nil {
			m.sidebar.SetTask(msg.Task)
		} else {
			m.sidebar.SetTask(m.taskList.SelectedTask())
		}

	case TaskListTickMsg:
		cmds = append(cmds, loadTasksCmd(m.orch), tickCmd())

	case ActionResultMsg:
		m.err = msg.Err
		if msg.Err == nil {
			cmds = append(cmds, loadTasksCmd(m.orch))
		}

	case ErrorMsg:
		m.err = msg.Err

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		cmds = append(cmds, cmd)

	case ExitRequestedMsg:
		return m, tea.Quit
	}

	if m.ctx.SidebarOpen {
		if _, isKey := msg.(tea.KeyMsg); !isKey {
			if cmd := m.sidebar.Update(msg); cmd != nil {
				cmds = append(cmds, cmd)
			}
		}
	}

	return m, tea.Batch(cmds...)
}

func (m *Model) handleKeyMsg(msg tea.KeyMsg) (tea.Cmd, bool, bool) {
	if cmd, handled := m.handleConfirmMode(msg); handled {
		return cmd, true, false
	}
	if handled := m.handleHelpMode(msg); handled {
		return nil, true, false
	}
	if cmd, handled := m.handleSearchMode(msg); handled {
		return cmd, true, false
	}
	if cmd, handled := m.handleSidebarMode(msg); handled {
		return cmd, true, false
	}
	return m.handleGlobalKeys(msg)
}

func (m *Model) handleConfirmMode(msg tea.KeyMsg) (tea.Cmd, bool) {
	if m.confirmPending == nil {
		return nil, false
	}
	switch {
	case key.Matches(msg, m.keys.ConfirmYes):
		pending := m.confirmPending
		m.confirmPending = nil
		if pending.kind == "quit" {
			return quitCmd(), true
		}
		return runActionCmd(m.orch, pending.kind, pending.taskID), true
	case key.Matches(msg, m.keys.ConfirmNo):
		m.confirmPending = nil
		return nil, true
	default:
		return nil, true
	}
}

func (m *Model) handleHelpMode(msg tea.KeyMsg) bool {
	if !m.showHelp {
		return false
	}
	if key.Matches(msg, m.keys.Help) || msg.Type == tea.KeyEsc || msg.Type == tea.KeyEnter {
		m.showHelp = false
		return true
	}
	return true
}

func (m *Model) handleSearchMode(msg tea.KeyMsg) (tea.Cmd, bool) {
	if !m.searchMode {
		return nil, false
	}
	changed := false
	switch msg.Type {
	case tea.KeyEsc, tea.KeyEnter:
		m.searchMode = false
		return nil, true
	case tea.KeyCtrlU:
		m.searchQuery = ""
		changed = true
	case tea.KeyBackspace:
		if len(m.searchQuery) > 0 {
			m.searchQuery = m.searchQuery[:len(m.searchQuery)-1]
			changed = true
		}
	case tea.KeyRunes:
		m.searchQuery += string(msg.Runes)
		changed = true
	}
	if changed && m.taskList.SetSearchQuery(m.searchQuery) {
		return m.selectedTaskCmd(), true
	}
	if changed {
		m.taskList.SetSearchQuery(m.searchQuery)
	}
	return nil, true
}

func (m *Model) handleSidebarMode(msg tea.KeyMsg) (tea.Cmd, bool) {
	if !m.ctx.SidebarOpen || !m.sidebarFocused {
		return nil, false
	}
	if key.Matches(msg, m.keys.Left) {
		m.sidebarFocused = false
		return nil, true
	}
	switch msg.String() {
	case "up", "down", "j", "k", "pgup", "pgdown", "home", "end", "ctrl+u", "ctrl+d":
		return m.sidebar.Update(msg), true
	}
	return nil, false
}

func (m *Model) handleGlobalKeys(msg tea.KeyMsg) (tea.Cmd, bool, bool) {
	if !m.sidebarFocused {
		switch msg.String() {
		case "H":
			if m.taskList.ScrollLeft() {
				return nil, true, false
			}
			return nil, true, false
		case "L":
			if m.taskList.ScrollRight(m.currentListWidth()) {
				return nil, true, false
			}
			return nil, true, false
		}
	}

	switch {
	case key.Matches(msg, m.keys.Quit):
		if m.shouldConfirmQuit() {
			m.confirmPending = &pendingAction{kind: "quit"}
			return nil, true, false
		}
		return nil, true, true
	case key.Matches(msg, m.keys.Help):
		m.showHelp = !m.showHelp
		return nil, true, false
	case key.Matches(msg, m.keys.Search):
		m.searchMode = true
		return nil, true, false
	case key.Matches(msg, m.keys.ToggleSidebar):
		m.ctx.ToggleSidebar()
		m.propagateContext()
		if !m.ctx.SidebarOpen {
			m.sidebarFocused = false
		}
		return nil, true, false
	case key.Matches(msg, m.keys.Left):
		m.sidebarFocused = false
		return nil, true, false
	case key.Matches(msg, m.keys.Right):
		if !m.ctx.SidebarOpen {
			m.ctx.SetSidebarOpen(true)
			m.propagateContext()
		}
		m.sidebarFocused = true
		return nil, true, false
	case key.Matches(msg, m.keys.Refresh):
		return loadTasksCmd(m.orch), true, false
	case key.Matches(msg, m.keys.Pause):
		if task := m.taskList.SelectedTask(); task != nil {
			return runActionCmd(m.orch, "pause", task.ID), true, false
		}
		return nil, true, false
	case key.Matches(msg, m.keys.Resume):
		if task := m.taskList.SelectedTask(); task != nil {
			return runActionCmd(m.orch, "resume", task.ID), true, false
		}
		return nil, true, false
	case key.Matches(msg, m.keys.Cancel):
		if task := m.taskList.SelectedTask(); task != nil {
			m.confirmPending = &pendingAction{kind: "cancel", taskID: task.ID}
		}
		return nil, true, false
	case key.Matches(msg, m.keys.Retry):
		if task := m.taskList.SelectedTask(); task != nil {
			return runActionCmd(m.orch, "retry", task.ID), true, false
		}
		return nil, true, false
	case key.Matches(msg, m.keys.Purge):
		if task := m.taskList.SelectedTask(); task != nil {
			m.confirmPending = &pendingAction{kind: "purge", taskID: task.ID}
		}
		return nil, true, false
	}

	if m.sidebarFocused {
		return nil, false, false
	}

	switch {
	case key.Matches(msg, m.keys.Up):
		if m.taskList.MoveUp() {
			return m.selectedTaskCmd(), true, false
		}
		return nil, true, false
	case key.Matches(msg, m.keys.Down):
		if m.taskList.MoveDown() {
			return m.selectedTaskCmd(), true, false
		}
		return nil, true, false
	case key.Matches(msg, m.keys.Top):
		if m.taskList.MoveTop() {
			return m.selectedTaskCmd(), true, false
		}
		return nil, true, false
	case key.Matches(msg, m.keys.Bottom):
		if m.taskList.MoveBottom() {
			return m.selectedTaskCmd(), true, false
		}
		return nil, true, false
	case key.Matches(msg, m.keys.FilterAll):
		if m.taskList.SetFilter(tasklist.FilterAll) {
			return m.selectedTaskCmd(), true, false
		}
		return nil, true, false
	case key.Matches(msg, m.keys.FilterRunning):
		if m.taskList.SetFilter(tasklist.FilterRunning) {
			return m.selectedTaskCmd(), true, false
		}
		return nil, true, false
	case key.Matches(msg, m.keys.FilterComplete):
		if m.taskList.SetFilter(tasklist.FilterCompleted) {
			return m.selectedTaskCmd(), true, false
		}
		return nil, true, false
	case key.Matches(msg, m.keys.FilterFailed):
		if m.taskList.SetFilter(tasklist.FilterFailed) {
			return m.selectedTaskCmd(), true, false
		}
		return nil, true, false
	case key.Matches(msg, m.keys.FilterStatus):
		if m.taskList.CycleFilter() {
			return m.selectedTaskCmd(), true, false
		}
		return nil, true, false
	}

	return nil, false, false
}

func (m *Model) currentListWidth() int {
	if m.ctx == nil {
		return 0
	}
	if m.ctx.SidebarOpen {
		return m.ctx.MainContentWidth
	}
	return m.ctx.ScreenWidth
}

func runActionCmd(orch *orchestrator.Orchestrator, kind, taskID string) tea.Cmd {
	return func() tea.Msg {
		if orch == nil {
			return ActionResultMsg{Err: nil}
		}
		var err error
		switch kind {
		case "pause":
			_, err = orch.Pause(taskID)
		case "resume":
			_, err = orch.Resume(context.Background(), taskID, orchestrator.ResumeOptions{
				Prompt:     "Resume from TUI keyboard shortcut",
				Background: true,
			})
		case "cancel":
			err = orch.Cancel(taskID)
		case "retry":
			_, err = orch.Retry(context.Background(), taskID, orchestrator.RetryOptions{Background: true})
		case "purge":
			err = orch.Purge(taskID)
		}
		return ActionResultMsg{Err: err}
	}
}

func quitCmd() tea.Cmd {
	return func() tea.Msg {
		return ExitRequestedMsg{}
	}
}

func (m *Model) shouldConfirmQuit() bool {
	if m.ctx == nil || m.ctx.Config == nil || !m.ctx.Config.TUI.ConfirmQuitWithRunningTasks {
		return false
	}
	for _, task := range m.tasks {
		if task != nil && task.Status == models.TaskStatusRunning {
			return true
		}
	}
	return false
}

// View implements tea.Model.
// Renders: header / content / footer using lipgloss.JoinVertical.
func (m Model) View() string {
	if !m.ready {
		return "\n  " + m.spinner.View() + " Initializing..."
	}

	head := m.header.View(m.orch.GetStats())
	content := m.renderContent()
	if m.showHelp {
		content = m.renderHelpOverlay()
	}
	foot := m.footer.View(m.renderStatusError())

	return lipgloss.JoinVertical(lipgloss.Left, head, content, foot)
}

func (m Model) renderStatusError() error {
	if m.confirmPending != nil {
		return &statusError{msg: "Confirm " + strings.ToUpper(m.confirmPending.kind) + " with y/n"}
	}
	if m.searchMode {
		return &statusError{msg: "Search: " + m.searchQuery}
	}
	return m.err
}

func (m Model) renderHelpOverlay() string {
	width := m.ctx.ScreenWidth
	height := m.ctx.MainContentHeight
	if width <= 0 || height <= 0 {
		return ""
	}

	boxWidth := width - 8
	if boxWidth > 100 {
		boxWidth = 100
	}
	if boxWidth < 50 {
		boxWidth = width - 2
	}

	helpText := renderHelpText(m.keys) + "\n\nPress ? or esc to close"
	box := lipgloss.NewStyle().
		Width(boxWidth).
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(m.theme.BorderBrand).
		Background(m.theme.Surface).
		Padding(1, 2).
		Render(helpText)

	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, box)
}

// renderContent renders the main content area (task list + sidebar placeholder).
func (m Model) renderContent() string {
	h := m.ctx.MainContentHeight
	if h <= 0 {
		return ""
	}

	listWidth := m.ctx.ScreenWidth
	if m.ctx.SidebarOpen {
		listWidth = m.ctx.MainContentWidth
	}

	taskListView := m.taskList.View(listWidth, h)

	if m.ctx.SidebarOpen {
		divider := lipgloss.NewStyle().
			Width(tuictx.SidebarDividerWidth).
			Height(h).
			Background(m.theme.BackgroundAlt).
			Render("")

		m.sidebar.SetTask(m.taskList.SelectedTask())
		return lipgloss.JoinHorizontal(lipgloss.Top, taskListView, divider, m.renderSidebarStub())
	}

	return taskListView
}

// renderSidebarStub renders a placeholder sidebar. Replaced by Phase 4.
func (m Model) renderSidebarStub() string {
	return m.sidebar.View()
}

func (m *Model) propagateContext() {
	for _, component := range m.contextAware {
		component.UpdateContext(m.ctx)
	}
}

func (m Model) selectedTaskCmd() tea.Cmd {
	task := m.taskList.SelectedTask()
	return func() tea.Msg {
		if task == nil {
			return TaskSelectedMsg{}
		}
		return TaskSelectedMsg{TaskID: task.ID, Task: task}
	}
}

type statusError struct {
	msg string
}

func (e *statusError) Error() string {
	return e.msg
}

func renderHelpText(keys *KeyMap) string {
	if keys == nil {
		return ""
	}
	var lines []string
	for _, group := range keys.FullHelp() {
		var cols []string
		for _, binding := range group {
			h := binding.Help()
			cols = append(cols, fmt.Sprintf("%s: %s", h.Key, h.Desc))
		}
		lines = append(lines, strings.Join(cols, "  •  "))
	}
	return strings.Join(lines, "\n")
}
