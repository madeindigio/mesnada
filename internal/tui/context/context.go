package tuictx

import "github.com/sevir/mesnada/internal/config"

const (
	HeaderHeight        = 1
	FooterHeight        = 1
	SidebarPercent      = 65
	SidebarDividerWidth = 1
)

// Context holds shared state propagated to all TUI components.
type Context struct {
	ScreenWidth       int
	ScreenHeight      int
	MainContentWidth  int
	MainContentHeight int
	SidebarOpen       bool
	SidebarWidth      int
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

func (c *Context) RecalcDimensions() {
	c.MainContentHeight = c.ScreenHeight - HeaderHeight - FooterHeight
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
