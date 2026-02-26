package markdown

import (
	"strings"

	"github.com/charmbracelet/x/ansi"
)

func TruncateANSISafe(content string, maxLines, maxWidth int) string {
	if maxLines <= 0 || maxWidth <= 0 || content == "" {
		return ""
	}

	lines := strings.Split(strings.ReplaceAll(content, "\r\n", "\n"), "\n")
	if len(lines) == 0 {
		return ""
	}

	truncated := lines
	hasMore := false
	if len(lines) > maxLines {
		truncated = lines[:maxLines]
		hasMore = true
	}

	for i := range truncated {
		truncated[i] = ansi.Truncate(truncated[i], maxWidth, "…")
	}

	out := strings.Join(truncated, "\n")
	if hasMore {
		out += "\n…"
	}
	return out
}
