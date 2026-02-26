package styles

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/sevir/mesnada/internal/tui/theme"
	"github.com/sevir/mesnada/pkg/models"
)

type HeaderStyles struct {
	Root        lipgloss.Style
	Title       lipgloss.Style
	EndpointDot lipgloss.Style
	StatsText   lipgloss.Style
}

type TableStyles struct {
	HeaderRow      lipgloss.Style
	Cell           lipgloss.Style
	SelectedCell   lipgloss.Style
	FilterActive   lipgloss.Style
	FilterInactive lipgloss.Style
}

type SidebarStyles struct {
	Root         lipgloss.Style
	SectionTitle lipgloss.Style
	MetaKey      lipgloss.Style
	MetaValue    lipgloss.Style
	ActionKey    lipgloss.Style
}

type FooterStyles struct {
	Root     lipgloss.Style
	KeyHint  lipgloss.Style
	HintText lipgloss.Style
	Error    lipgloss.Style
}

// Styles defines concrete lipgloss styles derived from a Theme.
type Styles struct {
	Theme theme.Theme

	Header  HeaderStyles
	Table   TableStyles
	Sidebar SidebarStyles
	Footer  FooterStyles

	Status map[models.TaskStatus]lipgloss.Style
	Engine map[models.Engine]lipgloss.Style
}

// InitStyles builds all concrete style instances from semantic theme tokens.
func InitStyles(th theme.Theme) Styles {
	s := Styles{
		Theme: th,
		Header: HeaderStyles{
			Root: lipgloss.NewStyle().
				Background(th.BackgroundAlt).
				Foreground(th.TextPrimary).
				BorderBottom(true).
				BorderForeground(th.BorderFaint).
				Padding(0, 1),
			Title: lipgloss.NewStyle().
				Bold(true).
				Foreground(th.Brand),
			EndpointDot: lipgloss.NewStyle().
				Foreground(th.Success),
			StatsText: lipgloss.NewStyle().
				Foreground(th.TextSecondary),
		},
		Table: TableStyles{
			HeaderRow: lipgloss.NewStyle().
				Bold(true).
				Foreground(th.TextSecondary),
			Cell: lipgloss.NewStyle().
				Foreground(th.TextPrimary),
			SelectedCell: lipgloss.NewStyle().
				Background(th.Surface).
				Foreground(th.TextPrimary),
			FilterActive: lipgloss.NewStyle().
				Bold(true).
				Foreground(th.Brand),
			FilterInactive: lipgloss.NewStyle().
				Foreground(th.TextFaint),
		},
		Sidebar: SidebarStyles{
			Root: lipgloss.NewStyle().
				Background(th.Surface).
				BorderLeft(true).
				BorderForeground(th.BorderBrand).
				Padding(0, 1),
			SectionTitle: lipgloss.NewStyle().
				Bold(true).
				Foreground(th.Brand),
			MetaKey: lipgloss.NewStyle().
				Foreground(th.TextSecondary),
			MetaValue: lipgloss.NewStyle().
				Foreground(th.TextPrimary),
			ActionKey: lipgloss.NewStyle().
				Bold(true).
				Foreground(th.Info),
		},
		Footer: FooterStyles{
			Root: lipgloss.NewStyle().
				Background(th.BackgroundAlt).
				Foreground(th.TextSecondary).
				BorderTop(true).
				BorderForeground(th.BorderFaint).
				Padding(0, 1),
			KeyHint: lipgloss.NewStyle().
				Bold(true).
				Foreground(th.Brand),
			HintText: lipgloss.NewStyle().
				Foreground(th.TextSecondary),
			Error: lipgloss.NewStyle().
				Bold(true).
				Background(th.Error).
				Foreground(lipgloss.AdaptiveColor{Light: "#ffffff", Dark: "#0b0f17"}).
				Padding(0, 1),
		},
		Status: map[models.TaskStatus]lipgloss.Style{},
		Engine: map[models.Engine]lipgloss.Style{},
	}

	for status, color := range th.StatusColors {
		s.Status[status] = lipgloss.NewStyle().Foreground(color)
	}
	for engine, color := range th.EngineColors {
		s.Engine[engine] = lipgloss.NewStyle().Foreground(color)
	}

	return s
}

func (s Styles) StatusIcon(status models.TaskStatus) string {
	return s.Theme.StatusIcon(status)
}

func (s Styles) StatusText(status models.TaskStatus, text string) string {
	if style, ok := s.Status[status]; ok {
		return style.Render(text)
	}
	return text
}

func (s Styles) EngineBadge(engine models.Engine) string {
	label := strings.ToUpper(string(engine))
	if label == "" {
		label = "DEFAULT"
	}
	if style, ok := s.Engine[engine]; ok {
		return style.Copy().Padding(0, 1).Render(label)
	}
	return label
}

func (s Styles) EngineText(engine models.Engine, text string) string {
	if style, ok := s.Engine[engine]; ok {
		return style.Render(text)
	}
	return text
}
