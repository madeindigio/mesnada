package agent

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeSourceConfig(t *testing.T, workDir string) string {
	t.Helper()
	path := filepath.Join(workDir, "mcp-config.json")
	content := `{
  "mcpServers": {
    "localSrv": {
      "type": "local",
      "command": "node",
      "args": ["server.js"],
      "cwd": "tools/mcp"
    },
    "httpSrv": {
      "type": "http",
      "url": "https://example.test/sse"
    }
  }
}`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("write source config: %v", err)
	}
	return path
}

func writeVSCodeConfig(t *testing.T, workDir string) string {
	t.Helper()
	vscodeDir := filepath.Join(workDir, ".vscode")
	if err := os.MkdirAll(vscodeDir, 0755); err != nil {
		t.Fatalf("create .vscode dir: %v", err)
	}
	path := filepath.Join(vscodeDir, "mcp.json")
	content := `{
  "servers": {
    "localSrv": {
      "type": "stdio",
      "command": "node",
      "args": ["server.js", "${input:api-key}"],
      "env": {"API_KEY":"${input:api-key}"},
      "cwd": "tools/mcp"
    },
    "httpSrv": {
      "type": "http",
      "url": "https://example.test/sse"
    }
  },
  "inputs": [
    {"type":"promptString","id":"api-key","description":"API key","password":true}
  ]
}`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("write VS Code config: %v", err)
	}
	return path
}

func TestConvertMCPConfigForTask_Claude(t *testing.T) {
	baseDir := t.TempDir()
	workDir := t.TempDir()
	_ = writeSourceConfig(t, workDir)

	path, err := ConvertMCPConfigForTask("@mcp-config.json", "task1", baseDir, workDir)
	if err != nil {
		t.Fatalf("ConvertMCPConfigForTask: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read output: %v", err)
	}
	var out ClaudeMCPConfig
	if err := json.Unmarshal(data, &out); err != nil {
		t.Fatalf("unmarshal output: %v", err)
	}

	if got := out.MCPServers["httpSrv"].Command; got != "npx" {
		t.Fatalf("unexpected http conversion command: %q", got)
	}
	if got := out.MCPServers["localSrv"].Cwd; got != filepath.Join(workDir, "tools/mcp") {
		t.Fatalf("unexpected cwd conversion: %q", got)
	}
}

func TestCreateGeminiSettingsFile(t *testing.T) {
	baseDir := t.TempDir()
	workDir := t.TempDir()
	_ = writeSourceConfig(t, workDir)

	path, err := CreateGeminiSettingsFile("@mcp-config.json", "task2", baseDir, workDir)
	if err != nil {
		t.Fatalf("CreateGeminiSettingsFile: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read output: %v", err)
	}
	var out GeminiSettings
	if err := json.Unmarshal(data, &out); err != nil {
		t.Fatalf("unmarshal output: %v", err)
	}

	if !out.MCPServers["localSrv"].Trust {
		t.Fatal("expected trust=true for gemini")
	}
	if got := out.MCPServers["httpSrv"].Args; len(got) != 3 || got[2] != "https://example.test/sse" {
		t.Fatalf("unexpected gemini http conversion: %#v", got)
	}
}

func TestConvertMCPConfigForOpenCode(t *testing.T) {
	baseDir := t.TempDir()
	workDir := t.TempDir()
	_ = writeSourceConfig(t, workDir)

	path, err := ConvertMCPConfigForOpenCode("@mcp-config.json", "task3", baseDir, workDir)
	if err != nil {
		t.Fatalf("ConvertMCPConfigForOpenCode: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read output: %v", err)
	}
	var out OpenCodeMCPConfig
	if err := json.Unmarshal(data, &out); err != nil {
		t.Fatalf("unmarshal output: %v", err)
	}

	if out.MCP["localSrv"].Type != "local" || len(out.MCP["localSrv"].Command) < 1 {
		t.Fatalf("unexpected local opencode conversion: %#v", out.MCP["localSrv"])
	}
	if got := out.MCP["httpSrv"].Command; len(got) != 4 || got[3] != "https://example.test/sse" {
		t.Fatalf("unexpected http opencode conversion: %#v", got)
	}
}

func TestCreateVibeConfigToml(t *testing.T) {
	workDir := t.TempDir()
	_ = writeSourceConfig(t, workDir)

	toml, err := createVibeConfigToml("@mcp-config.json", workDir, "devstral-small")
	if err != nil {
		t.Fatalf("createVibeConfigToml: %v", err)
	}

	if !strings.Contains(toml, `active_model = "devstral-small"`) {
		t.Fatalf("missing model in toml: %s", toml)
	}
	if !strings.Contains(toml, `transport = "http"`) {
		t.Fatalf("expected http transport in toml: %s", toml)
	}
}

func TestConvertMCPConfigForTask_VSCodeInput(t *testing.T) {
	baseDir := t.TempDir()
	workDir := t.TempDir()
	_ = writeVSCodeConfig(t, workDir)

	path, err := ConvertMCPConfigForTask("@.vscode/mcp.json", "task-vscode", baseDir, workDir)
	if err != nil {
		t.Fatalf("ConvertMCPConfigForTask(vscode): %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read output: %v", err)
	}
	var out ClaudeMCPConfig
	if err := json.Unmarshal(data, &out); err != nil {
		t.Fatalf("unmarshal output: %v", err)
	}
	if got := out.MCPServers["httpSrv"].Command; got != "npx" {
		t.Fatalf("unexpected http conversion command: %q", got)
	}
	if got := out.MCPServers["localSrv"].Args[1]; got != "${input:api-key}" {
		t.Fatalf("expected unresolved placeholder, got %q", got)
	}
}

func TestConvertMCPConfigFormats_ToVSCode(t *testing.T) {
	workDir := t.TempDir()
	inputPath := writeSourceConfig(t, workDir)
	outputPath := filepath.Join(workDir, "converted-vscode.json")

	if err := ConvertMCPConfigFormats(inputPath, outputPath, workDir, MCPFormatMesnada, MCPFormatVSCode); err != nil {
		t.Fatalf("ConvertMCPConfigFormats: %v", err)
	}

	data, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("read output: %v", err)
	}
	var out VSCodeMCPConfig
	if err := json.Unmarshal(data, &out); err != nil {
		t.Fatalf("unmarshal VS Code output: %v", err)
	}
	if out.Servers["localSrv"].Type != "stdio" {
		t.Fatalf("expected stdio local type, got %q", out.Servers["localSrv"].Type)
	}
	if out.Servers["httpSrv"].Type != "http" {
		t.Fatalf("expected http type, got %q", out.Servers["httpSrv"].Type)
	}
}
