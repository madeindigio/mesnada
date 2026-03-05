package tuictx

import "github.com/sevir/mesnada/internal/config"

const (
	HeaderHeight        = 1
	FooterHeight        = 1
	SidebarPercent      = 65
	SidebarDividerWidth = 1
	LogPanelPercent     = 30 // percentage of screen height when open
	LogPanelBorderH     = 1  // top border/title line
)

// Context holds shared state propagated to all TUI components.
type Context struct {
	ScreenWidth       int
	ScreenHeight      int
	MainContentWidth  int
	MainContentHeight int
	SidebarOpen       bool
	SidebarWidth      int
	LogPanelOpen      bool
	LogPanelHeight    int
	Config            *config.Config
	Error             error
	SelectedTaskID    string
}

func New(cfg *config.Config) *Context {
	return &Context{
		Config:      cfg,
		SidebarOpen: true,
	}
}

func (c *Context) SetWindowSize(width, height int) {
	c.ScreenWidth = width
	c.ScreenHeight = height
	c.SidebarWidth = width * SidebarPercent / 100
	c.RecalcDimensions()
}

func (c *Context) SetSidebarOpen(open bool) {
	c.SidebarOpen = open
	c.RecalcDimensions()
}

func (c *Context) ToggleSidebar() {
	c.SetSidebarOpen(!c.SidebarOpen)
}

func (c *Context) ToggleLogPanel() {
	c.LogPanelOpen = !c.LogPanelOpen
	c.RecalcDimensions()
}

func (c *Context) RecalcDimensions() {
	available := c.ScreenHeight - HeaderHeight - FooterHeight
	if c.LogPanelOpen {
		c.LogPanelHeight = c.ScreenHeight * LogPanelPercent / 100
		if c.LogPanelHeight < 3 {
			c.LogPanelHeight = 3
		}
		available -= c.LogPanelHeight + LogPanelBorderH
	} else {
		c.LogPanelHeight = 0
	}
	c.MainContentHeight = available
	if c.MainContentHeight < 0 {
		c.MainContentHeight = 0
	}
	c.SyncMainContentWidth()
}

// SyncMainContentWidth keeps the left panel width in sync with sidebar visibility.
func (c *Context) SyncMainContentWidth() {
	if c.SidebarOpen && c.SidebarWidth > 0 {
		c.MainContentWidth = c.ScreenWidth - c.SidebarWidth - SidebarDividerWidth
	} else {
		c.MainContentWidth = c.ScreenWidth
	}
	if c.MainContentWidth < 0 {
		c.MainContentWidth = 0
	}
}
