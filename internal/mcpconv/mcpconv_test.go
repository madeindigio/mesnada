package mcpconv

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func boolPtr(v bool) *bool { return &v }

func TestParseZedSettings(t *testing.T) {
	input := []byte(`{
"theme": "One",
"context_servers": {
"local-a": {
"command": "node",
"args": ["server.js"],
"env": {"TOKEN": "abc"},
"enabled": false,
"timeout": 9
},
"remote-a": {
"url": "https://example.com/mcp",
"headers": {"Authorization": "Bearer t"},
"enabled": true,
"timeout": 15
}
}
}`)

	cfg, err := ParseZedSettings(input)
	if err != nil {
		t.Fatalf("ParseZedSettings returned error: %v", err)
	}

	local := cfg.MCPServers["local-a"]
	if local.Type != "local" || local.Command != "node" {
		t.Fatalf("unexpected local mapping: %+v", local)
	}
	if local.Env["TOKEN"] != "abc" || local.Enabled == nil || *local.Enabled {
		t.Fatalf("expected env/enabled mapping on local server, got %+v", local)
	}
	if local.Timeout != 9 {
		t.Fatalf("expected timeout=9 for local, got %d", local.Timeout)
	}

	remote := cfg.MCPServers["remote-a"]
	if remote.Type != "http" || remote.URL != "https://example.com/mcp" {
		t.Fatalf("unexpected remote mapping: %+v", remote)
	}
	if remote.Headers["Authorization"] == "" || remote.Enabled == nil || !*remote.Enabled {
		t.Fatalf("expected headers/enabled mapping on remote server, got %+v", remote)
	}
	if remote.Timeout != 15 {
		t.Fatalf("expected timeout=15 for remote, got %d", remote.Timeout)
	}
}

func TestRenderZedSettings(t *testing.T) {
	cfg := CanonicalConfig{MCPServers: map[string]CanonicalServer{
		"local-a": {
			Type:    "local",
			Command: "python",
			Args:    []string{"app.py"},
			Env:     map[string]string{"X": "1"},
			Enabled: boolPtr(true),
			Timeout: 11,
		},
		"remote-a": {
			Type:    "http",
			URL:     "https://remote.example/mcp",
			Headers: map[string]string{"Authorization": "Bearer x"},
			Enabled: boolPtr(false),
			Timeout: 20,
		},
	}}

	rendered := RenderZed(cfg)
	data, err := json.Marshal(rendered)
	if err != nil {
		t.Fatalf("marshal rendered zed settings: %v", err)
	}

	var roundTrip ZedSettings
	if err := json.Unmarshal(data, &roundTrip); err != nil {
		t.Fatalf("unmarshal rendered zed settings: %v", err)
	}

	if roundTrip.ContextServers["local-a"].Command != "python" || roundTrip.ContextServers["local-a"].Env["X"] != "1" {
		t.Fatalf("unexpected local zed render: %+v", roundTrip.ContextServers["local-a"])
	}
	if roundTrip.ContextServers["remote-a"].URL != "https://remote.example/mcp" || roundTrip.ContextServers["remote-a"].Headers["Authorization"] == "" {
		t.Fatalf("unexpected remote zed render: %+v", roundTrip.ContextServers["remote-a"])
	}
}

func TestRenderFormatsRemainCompatible(t *testing.T) {
	cfg := CanonicalConfig{MCPServers: map[string]CanonicalServer{
		"local-default": {Type: "local", Command: "node", Args: []string{"a.js"}, Env: map[string]string{"A": "1"}},
		"local-custom":  {Type: "local", Command: "bash", Args: []string{"-lc", "echo ok"}, Enabled: boolPtr(false), Timeout: 1200},
		"remote":        {Type: "http", URL: "https://remote"},
	}}

	claude := RenderClaude(cfg, "")
	if claude.MCPServers["local-default"].Command != "node" || claude.MCPServers["local-default"].Env["A"] != "1" {
		t.Fatalf("unexpected Claude local render: %+v", claude.MCPServers["local-default"])
	}
	if claude.MCPServers["remote"].Command != "npx" {
		t.Fatalf("expected Claude remote conversion via npx mcp-remote, got %+v", claude.MCPServers["remote"])
	}

	gemini := RenderGemini(cfg, "")
	if gemini.MCPServers["local-default"].Env["A"] != "1" || !gemini.MCPServers["local-default"].Trust {
		t.Fatalf("unexpected Gemini local render: %+v", gemini.MCPServers["local-default"])
	}

	opencode := RenderOpenCode(cfg)
	if !opencode.MCP["local-default"].Enabled || opencode.MCP["local-default"].Timeout != 5000 {
		t.Fatalf("expected default OpenCode enabled/timeout compatibility, got %+v", opencode.MCP["local-default"])
	}
	if opencode.MCP["local-custom"].Enabled || opencode.MCP["local-custom"].Timeout != 1200 {
		t.Fatalf("expected custom OpenCode enabled/timeout mapping, got %+v", opencode.MCP["local-custom"])
	}
}

func TestZedProjectSettingsPath(t *testing.T) {
	if ZedProjectSettingsPath != ".zed/settings.json" {
		t.Fatalf("unexpected Zed project path: %s", ZedProjectSettingsPath)
	}
}

func TestParseCanonicalFile_AntigravityCompat(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "mcp_config.json")
	data := []byte(`{
  "mcpServers": {
    "local": {"command":"node","args":["a.js"],"env":{"TOKEN":"abc"}},
    "remote": {"serverUrl":"https://example.test/mcp"}
  }
}`)
	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatalf("write temp config: %v", err)
	}

	cfg, err := ParseCanonicalFile(path)
	if err != nil {
		t.Fatalf("ParseCanonicalFile error: %v", err)
	}
	if cfg.MCPServers["local"].Env["TOKEN"] != "abc" {
		t.Fatalf("expected local env to parse, got %+v", cfg.MCPServers["local"])
	}
	if cfg.MCPServers["remote"].ServerURL != "https://example.test/mcp" {
		t.Fatalf("expected remote serverUrl to parse, got %+v", cfg.MCPServers["remote"])
	}
}

func TestRenderAntigravity_PrefersServerURL(t *testing.T) {
	t.Parallel()

	cfg := CanonicalConfig{MCPServers: map[string]CanonicalServer{
		"remote": {Type: "http", URL: "https://legacy.test/mcp", ServerURL: "https://preferred.test/mcp"},
	}}
	out := RenderAntigravity(cfg)
	remote := out.MCPServers["remote"]
	if remote.ServerURL != "https://preferred.test/mcp" {
		t.Fatalf("expected preferred serverUrl, got %+v", remote)
	}
	if remote.URL != "" {
		t.Fatalf("expected url omitted when serverUrl is used, got %+v", remote)
	}
}

func TestParseCanonicalBytesWithFormat_VSCodeIgnoresInputs(t *testing.T) {
	t.Parallel()

	data := []byte(`{
  "servers": {
    "localSrv": {
      "type": "stdio",
      "command": "node",
      "args": ["server.js", "${input:api-key}"],
      "env": {"API_KEY": "${input:api-key}"}
    },
    "httpSrv": {
      "type": "http",
      "url": "https://example.test/mcp"
    }
  },
  "inputs": [
    {"type":"promptString","id":"api-key","description":"API key","password":true}
  ]
}`)

	cfg, err := ParseCanonicalBytesWithFormat(data, FormatVSCode)
	if err != nil {
		t.Fatalf("ParseCanonicalBytesWithFormat(vscode) error: %v", err)
	}
	if cfg.MCPServers["localSrv"].Type != "local" {
		t.Fatalf("expected local type for stdio server, got %q", cfg.MCPServers["localSrv"].Type)
	}
	if got := cfg.MCPServers["localSrv"].Env["API_KEY"]; got != "${input:api-key}" {
		t.Fatalf("expected placeholder unchanged, got %q", got)
	}
	if cfg.MCPServers["httpSrv"].Type != "http" {
		t.Fatalf("expected http type for remote server, got %q", cfg.MCPServers["httpSrv"].Type)
	}
}

func TestRenderByFormat_VSCode(t *testing.T) {
	t.Parallel()

	cfg := CanonicalConfig{
		MCPServers: map[string]CanonicalServer{
			"localSrv": {
				Type:    "local",
				Command: "node",
				Args:    []string{"server.js"},
				Env:     map[string]string{"A": "B"},
				Cwd:     "./tools",
			},
			"httpSrv": {
				Type:    "http",
				URL:     "https://example.test/mcp",
				Headers: map[string]string{"Authorization": "Bearer token"},
			},
		},
	}

	payload, err := RenderByFormat(cfg, FormatVSCode, "")
	if err != nil {
		t.Fatalf("RenderByFormat(vscode) error: %v", err)
	}
	out, ok := payload.(VSCodeConfig)
	if !ok {
		t.Fatalf("expected VSCodeConfig payload, got %T", payload)
	}
	if out.Servers["localSrv"].Type != "stdio" {
		t.Fatalf("expected stdio type for local server, got %q", out.Servers["localSrv"].Type)
	}
	if out.Servers["httpSrv"].Type != "http" {
		t.Fatalf("expected http type for remote server, got %q", out.Servers["httpSrv"].Type)
	}
	if got := out.Servers["httpSrv"].Headers["Authorization"]; got != "Bearer token" {
		t.Fatalf("expected headers preserved, got %q", got)
	}
}

func TestRenderByFormat_AllFormats(t *testing.T) {
	t.Parallel()

	cfg := CanonicalConfig{MCPServers: map[string]CanonicalServer{
		"local": {Type: "local", Command: "node", Args: []string{"a.js"}, Env: map[string]string{"X": "1"}},
		"http":  {Type: "http", URL: "https://example.test/mcp"},
	}}

	for _, f := range []string{FormatMesnada, FormatVSCode, FormatClaude, FormatGemini, FormatOpenCode, FormatVibe, FormatZed, FormatAntigravity} {
		payload, err := RenderByFormat(cfg, f, "")
		if err != nil {
			t.Errorf("RenderByFormat(%s): %v", f, err)
			continue
		}
		if f == FormatVibe {
			s, ok := payload.(string)
			if !ok || s == "" {
				t.Errorf("expected non-empty string for vibe, got %T: %v", payload, payload)
			}
			continue
		}
		data, err := json.Marshal(payload)
		if err != nil {
			t.Errorf("json.Marshal for format %s: %v", f, err)
		}
		if len(data) == 0 {
			t.Errorf("empty JSON for format %s", f)
		}
	}
}

func TestRenderByFormat_NonCopilotAbsolutizesRelativePaths(t *testing.T) {
	t.Parallel()

	workDir := "/tmp/workspace"
	cfg := CanonicalConfig{MCPServers: map[string]CanonicalServer{
		"local": {
			Type:    "local",
			Command: "./bin/mcp-server",
			Args:    []string{"--config=./configs/mcp.json", "./scripts/server.js", "-y"},
		},
	}}

	payload, err := RenderByFormat(cfg, FormatOpenCode, workDir)
	if err != nil {
		t.Fatalf("RenderByFormat(opencode): %v", err)
	}

	out, ok := payload.(OpenCodeConfig)
	if !ok {
		t.Fatalf("expected OpenCodeConfig payload, got %T", payload)
	}
	cmd := out.MCP["local"].Command
	if len(cmd) < 3 {
		t.Fatalf("unexpected command rendering: %#v", cmd)
	}
	if cmd[0] != filepath.Join(workDir, "bin", "mcp-server") {
		t.Fatalf("expected absolute command path, got %q", cmd[0])
	}
	if cmd[1] != "--config="+filepath.Join(workDir, "configs", "mcp.json") {
		t.Fatalf("expected absolute --config path, got %q", cmd[1])
	}
	if cmd[2] != filepath.Join(workDir, "scripts", "server.js") {
		t.Fatalf("expected absolute script path, got %q", cmd[2])
	}
}

func TestRenderByFormat_CopilotKeepsRelativePaths(t *testing.T) {
	t.Parallel()

	cfg := CanonicalConfig{MCPServers: map[string]CanonicalServer{
		"local": {
			Type:    "local",
			Command: "./bin/mcp-server",
			Args:    []string{"--config=./configs/mcp.json", "./scripts/server.js"},
		},
	}}

	payload, err := RenderByFormat(cfg, FormatMesnada, "/tmp/workspace")
	if err != nil {
		t.Fatalf("RenderByFormat(mesnada): %v", err)
	}

	out, ok := payload.(CanonicalConfig)
	if !ok {
		t.Fatalf("expected CanonicalConfig payload, got %T", payload)
	}
	srv := out.MCPServers["local"]
	if srv.Command != "./bin/mcp-server" {
		t.Fatalf("expected relative command preserved, got %q", srv.Command)
	}
	if len(srv.Args) != 2 || srv.Args[0] != "--config=./configs/mcp.json" || srv.Args[1] != "./scripts/server.js" {
		t.Fatalf("expected relative args preserved, got %#v", srv.Args)
	}
}

func TestParseCanonicalBytesWithFormat_ZedInput(t *testing.T) {
	t.Parallel()

	data := []byte(`{
		"context_servers": {
			"myServer": {"command": "go", "args": ["run", "."], "env": {"PORT": "9000"}}
		}
	}`)

	cfg, err := ParseCanonicalBytesWithFormat(data, FormatZed)
	if err != nil {
		t.Fatalf("ParseCanonicalBytesWithFormat(zed): %v", err)
	}
	srv := cfg.MCPServers["myServer"]
	if srv.Command != "go" || len(srv.Args) != 2 {
		t.Fatalf("unexpected zed parse: %+v", srv)
	}
}

func TestParseCanonicalBytesWithFormat_AntigravityInput(t *testing.T) {
	t.Parallel()

	data := []byte(`{
		"mcpServers": {
			"remote": {"serverUrl": "https://example.test/mcp", "type": "http"}
		}
	}`)

	cfg, err := ParseCanonicalBytesWithFormat(data, FormatAntigravity)
	if err != nil {
		t.Fatalf("ParseCanonicalBytesWithFormat(antigravity): %v", err)
	}
	srv := cfg.MCPServers["remote"]
	if srv.ServerURL != "https://example.test/mcp" {
		t.Fatalf("unexpected antigravity parse: %+v", srv)
	}
}

func TestWriteAllFormats(t *testing.T) {
	t.Parallel()

	cfg := CanonicalConfig{MCPServers: map[string]CanonicalServer{
		"local": {Type: "local", Command: "node", Args: []string{"a.js"}},
	}}

	projectDir := t.TempDir()
	written, err := WriteAllFormats(cfg, projectDir, projectDir)
	if err != nil {
		t.Fatalf("WriteAllFormats: %v", err)
	}
	if len(written) != len(AllFormats) {
		t.Fatalf("expected %d files, got %d: %v", len(AllFormats), len(written), written)
	}
	for _, path := range written {
		if _, err := os.Stat(path); err != nil {
			t.Errorf("expected file at %s: %v", path, err)
		}
	}
}

func TestWriteAllFormatsSkippingSource_DoesNotOverwriteSource(t *testing.T) {
	t.Parallel()

	cfg := CanonicalConfig{MCPServers: map[string]CanonicalServer{
		"local": {Type: "local", Command: "node", Args: []string{"a.js"}},
	}}

	projectDir := t.TempDir()
	sourcePath := filepath.Join(projectDir, ".vscode", "mcp.json")
	if err := os.MkdirAll(filepath.Dir(sourcePath), 0755); err != nil {
		t.Fatalf("mkdir source dir: %v", err)
	}
	original := []byte("SENTINEL")
	if err := os.WriteFile(sourcePath, original, 0644); err != nil {
		t.Fatalf("write source file: %v", err)
	}

	written, err := WriteAllFormatsSkippingSource(cfg, projectDir, projectDir, sourcePath)
	if err != nil {
		t.Fatalf("WriteAllFormatsSkippingSource: %v", err)
	}

	for _, p := range written {
		if filepath.Clean(p) == filepath.Clean(sourcePath) {
			t.Fatalf("source file was unexpectedly written: %s", p)
		}
	}

	data, err := os.ReadFile(sourcePath)
	if err != nil {
		t.Fatalf("read source file: %v", err)
	}
	if string(data) != string(original) {
		t.Fatalf("source file was overwritten; got %q want %q", string(data), string(original))
	}
}

func TestParseCanonicalBytesWithFormat_VSCode_JSONC(t *testing.T) {
	t.Parallel()

	data := []byte(`{
  // line comment
  "servers": {
    "srv": {
      "type": "stdio",
      "command": "node",
      "args": ["server.js",],
    },
  },
}`)

	cfg, err := ParseCanonicalBytesWithFormat(data, FormatVSCode)
	if err != nil {
		t.Fatalf("ParseCanonicalBytesWithFormat(vscode jsonc) error: %v", err)
	}

	srv, ok := cfg.MCPServers["srv"]
	if !ok {
		t.Fatalf("expected server 'srv' to be parsed")
	}
	if srv.Command != "node" || len(srv.Args) != 1 || srv.Args[0] != "server.js" {
		t.Fatalf("unexpected parsed server: %+v", srv)
	}
}
