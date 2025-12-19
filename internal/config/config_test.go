package config

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestExpandHome_TildeOnly(t *testing.T) {
	home := expandHome("~")
	if home == "" {
		t.Fatalf("expected non-empty home")
	}
}

func TestExpandHome_TildeSlash(t *testing.T) {
	got := expandHome("~/.mesnada/tasks.json")
	if got == "~/.mesnada/tasks.json" {
		t.Fatalf("expected ~ to be expanded, got %q", got)
	}
	if strings.Contains(got, "~") {
		t.Fatalf("expected no ~ after expansion, got %q", got)
	}
	if !filepath.IsAbs(got) {
		t.Fatalf("expected absolute path after expansion, got %q", got)
	}
}

func TestExpandMCPConfig_AtTilde(t *testing.T) {
	got := expandMCPConfig("@~/.copilot/mcp-config.json")
	if !strings.HasPrefix(got, "@") {
		t.Fatalf("expected leading @, got %q", got)
	}
	inner := strings.TrimPrefix(got, "@")
	if strings.Contains(inner, "~") {
		t.Fatalf("expected ~ to be expanded, got %q", got)
	}
	if !filepath.IsAbs(inner) {
		t.Fatalf("expected absolute inner path after expansion, got %q", got)
	}
}

func TestResolvePath_RelativeAgainstBaseDir(t *testing.T) {
	base := "/tmp/mesnada-config-dir"
	got := resolvePath("tasks.json", base)
	want := filepath.Clean(filepath.Join(base, "tasks.json"))
	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

func TestResolvePath_AbsoluteUnchanged(t *testing.T) {
	abs := "/var/lib/mesnada/tasks.json"
	got := resolvePath(abs, "/tmp/whatever")
	if got != abs {
		t.Fatalf("expected %q, got %q", abs, got)
	}
}
