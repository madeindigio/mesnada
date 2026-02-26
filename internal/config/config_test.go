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

func TestValidateACP_ValidConfig(t *testing.T) {
	cfg := &Config{
		ACP: ACPConfig{
			Enabled:      true,
			DefaultAgent: "claude-code",
			DefaultMode:  "code",
			Agents: map[string]ACPAgentConfig{
				"claude-code": {
					Name:    "claude-code",
					Command: "claude-code-acp",
				},
			},
		},
	}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("expected valid config, got error: %v", err)
	}
}

func TestValidateACP_InvalidDefaultAgent(t *testing.T) {
	cfg := &Config{
		ACP: ACPConfig{
			Enabled:      true,
			DefaultAgent: "nonexistent",
			Agents: map[string]ACPAgentConfig{
				"claude-code": {
					Name:    "claude-code",
					Command: "claude-code-acp",
				},
			},
		},
	}
	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected error for nonexistent default_agent")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Fatalf("expected 'not found' error, got: %v", err)
	}
}

func TestValidateACP_InvalidMode(t *testing.T) {
	cfg := &Config{
		ACP: ACPConfig{
			Enabled:      true,
			DefaultAgent: "claude-code",
			DefaultMode:  "invalid-mode",
			Agents: map[string]ACPAgentConfig{
				"claude-code": {
					Name:    "claude-code",
					Command: "claude-code-acp",
				},
			},
		},
	}
	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected error for invalid default_mode")
	}
	if !strings.Contains(err.Error(), "invalid default_mode") {
		t.Fatalf("expected 'invalid default_mode' error, got: %v", err)
	}
}

func TestValidateACP_MissingCommand(t *testing.T) {
	cfg := &Config{
		ACP: ACPConfig{
			Enabled: true,
			Agents: map[string]ACPAgentConfig{
				"agent1": {
					Name: "agent1",
					// Command is missing
				},
			},
		},
	}
	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected error for missing command")
	}
	if !strings.Contains(err.Error(), "command is required") {
		t.Fatalf("expected 'command is required' error, got: %v", err)
	}
}

func TestValidateACP_DisabledNoValidation(t *testing.T) {
	cfg := &Config{
		ACP: ACPConfig{
			Enabled:      false,
			DefaultAgent: "nonexistent", // This should not trigger error when disabled
		},
	}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("expected no error when ACP is disabled, got: %v", err)
	}
}
