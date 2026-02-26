package markdown

import (
	"strings"

	"github.com/charmbracelet/glamour"
	glamouransi "github.com/charmbracelet/glamour/ansi"
)

func Render(content string, width int) (string, error) {
	if strings.TrimSpace(content) == "" {
		return "(empty prompt)", nil
	}
	if width < 20 {
		width = 20
	}

	r, err := glamour.NewTermRenderer(
		glamour.WithWordWrap(width),
		glamour.WithStyles(darkStyleConfig()),
	)
	if err != nil {
		return "", err
	}
	return r.Render(content)
}

func darkStyleConfig() glamouransi.StyleConfig {
	return glamouransi.StyleConfig{
		Document: glamouransi.StyleBlock{
			StylePrimitive: glamouransi.StylePrimitive{
				Color: stringPtr("252"),
			},
		},
		Heading: glamouransi.StyleBlock{
			StylePrimitive: glamouransi.StylePrimitive{
				Color: stringPtr("111"),
				Bold:  boolPtr(true),
			},
		},
		CodeBlock: glamouransi.StyleCodeBlock{
			StyleBlock: glamouransi.StyleBlock{
				StylePrimitive: glamouransi.StylePrimitive{
					Color:           stringPtr("223"),
					BackgroundColor: stringPtr("235"),
				},
			},
		},
		Link: glamouransi.StylePrimitive{
			Color:     stringPtr("117"),
			Underline: boolPtr(true),
		},
		Paragraph: glamouransi.StyleBlock{
			StylePrimitive: glamouransi.StylePrimitive{
				Color: stringPtr("252"),
			},
		},
	}
}

func stringPtr(v string) *string { return &v }
func boolPtr(v bool) *bool       { return &v }
