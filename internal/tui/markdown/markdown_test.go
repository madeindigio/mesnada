package markdown

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
)

func TestTruncateANSISafe(t *testing.T) {
	input := "\x1b[31mhello world\x1b[0m\nline2\nline3"
	out := TruncateANSISafe(input, 2, 6)
	lines := strings.Split(out, "\n")
	if len(lines) != 3 {
		t.Fatalf("expected 3 lines (2 + ellipsis), got %d", len(lines))
	}
	if ansi.StringWidth(lines[0]) > 6 {
		t.Fatalf("first line width should be <= 6, got %d", ansi.StringWidth(lines[0]))
	}
	if lines[2] != "…" {
		t.Fatalf("expected final ellipsis line, got %q", lines[2])
	}
}

func TestRender(t *testing.T) {
	out, err := Render("# Title\n\n- a", 50)
	if err != nil {
		t.Fatalf("Render returned error: %v", err)
	}
	if !strings.Contains(out, "Title") {
		t.Fatalf("expected rendered output to contain heading text, got %q", out)
	}
}
