# ACP Support in Mesnada

## Overview

Mesnada supports the **Agent Client Protocol (ACP)**, an open protocol for communication between AI coding agents and their clients (editors, orchestrators). ACP provides a standardized way to:

- Execute agent sessions with consistent lifecycle management
- Control file system access and terminal operations securely
- Handle permission requests for sensitive operations
- Stream session updates in real-time
- Manage multi-turn conversations

## What is ACP?

The Agent Client Protocol (ACP) is a JSON-RPC 2.0 based protocol that defines how clients communicate with AI coding agents. Unlike traditional stdio-based CLI agents, ACP provides:

- **Bidirectional streaming**: Agents can send updates during execution
- **Structured capabilities**: Explicit declaration of what agents can do
- **Permission system**: Agents can request approval for sensitive operations
- **MCP Integration**: Native support for Model Context Protocol servers
- **Session management**: Persistent sessions for multi-turn interactions

## Configuration

### Basic Configuration

Configure ACP agents in your `~/.mesnada/config.yaml`:

```yaml
acp:
  default_agent: "claude-code"
  auto_permission: false  # Require manual approval for sensitive operations

  agents:
    claude-code:
      name: "claude-code"
      title: "Claude Code (ACP)"
      command: "claude-code"
      args: ["--acp"]
      mode: "code"
      work_dir: ""
      env:
        NO_COLOR: "1"

      capabilities:
        terminals: true
        file_access: true
        permissions: true

      mcp_servers:
        - name: "filesystem"
          command: "mcp-server-filesystem"
          args: []
          env: {}
```

### Configuration Options

#### Global ACP Settings

- **`default_agent`**: The default ACP agent to use (must match an agent name)
- **`auto_permission`**: If `true`, auto-approve all permission requests (useful for CI/batch mode)

#### Agent Configuration

Each agent under `agents:` can have:

- **`name`**: Unique identifier for the agent
- **`title`**: Human-readable name (displayed in UI)
- **`command`**: Binary to execute (e.g., `claude-code`, `/path/to/custom-agent`)
- **`args`**: Command-line arguments
- **`mode`**: Default operation mode - `"code"`, `"ask"`, or `"architect"`
- **`work_dir`**: Default working directory (empty = use task work_dir)
- **`env`**: Environment variables to set

#### Capabilities

Control what the agent can do:

- **`terminals`**: Allow terminal/shell command execution
- **`file_access`**: Allow reading/writing files in workspace
- **`permissions`**: Enable the permission request system

#### MCP Servers

List of MCP servers to provide to the agent:

```yaml
mcp_servers:
  - name: "brave-search"
    command: "npx"
    args: ["-y", "@modelcontextprotocol/server-brave-search"]
    env:
      BRAVE_API_KEY: "your-api-key"
```

## Usage

### Spawning ACP Agents

#### Via MCP Tool (from another agent)

```json
{
  "tool": "spawn_agent",
  "arguments": {
    "prompt": "Implement user authentication",
    "engine": "acp-claude",
    "work_dir": "/path/to/project",
    "background": true
  }
}
```

#### Via HTTP API

```bash
curl -X POST http://localhost:8765/api/spawn \
  -H "Content-Type: application/json" \
  -d '{
    "prompt": "Add dark mode support",
    "engine": "acp-claude",
    "work_dir": "/path/to/project",
    "background": true,
    "acp_mode": "code"
  }'
```

#### Via Go Client

```go
import "github.com/sevir/mesnada/pkg/client"

mcpClient := client.NewMCPClient("http://localhost:8765")

task, err := mcpClient.SpawnAgent(ctx, client.SpawnRequest{
    Prompt: "Refactor authentication module",
    Engine: "acp-claude",
    WorkDir: "/path/to/project",
    Background: true,
    ACPMode: "code",
})
```

### Engine Types

Mesnada supports several ACP engine types:

- **`acp`**: Generic ACP agent (uses `default_agent` from config)
- **`acp-claude`**: Claude Code via ACP
- **`acp-codex`**: OpenAI Codex via ACP
- **`acp-custom`**: Custom ACP agent

Specify the engine when spawning:

```json
{
  "engine": "acp-claude",
  "model": "claude-sonnet-4.5"
}
```

### Session Control

ACP sessions support advanced control operations:

#### Get Session Status

```bash
curl http://localhost:8765/api/tasks/{task_id}/acp/status
```

Response:
```json
{
  "task_id": "task-abc123",
  "session_id": "session-xyz789",
  "connected": true,
  "mode": "code",
  "acp_status": {
    "session_id": "session-xyz789",
    "mode": "code",
    "agent_name": "claude-code",
    "is_connected": true,
    "tool_calls": 42
  }
}
```

#### Send Follow-up Prompt (Phase 6+)

```bash
curl -X POST http://localhost:8765/api/tasks/{task_id}/acp/prompt \
  -H "Content-Type: application/json" \
  -d '{
    "message": "Also add unit tests for the new feature"
  }'
```

#### Change Operation Mode (Phase 6+)

```bash
curl -X POST http://localhost:8765/api/tasks/{task_id}/acp/mode \
  -H "Content-Type: application/json" \
  -d '{
    "mode": "ask"
  }'
```

Valid modes: `code`, `ask`, `architect`

## Permission System

When `auto_permission: false`, agents must request permission for sensitive operations:

### List Pending Permissions

```bash
curl http://localhost:8765/api/tasks/{task_id}/acp/permissions
```

Response:
```json
{
  "task_id": "task-abc123",
  "permissions": [
    {
      "request_id": "perm-1",
      "tool_call": "Delete file: config.json",
      "options": [
        {"option_id": "allow", "label": "Allow"},
        {"option_id": "deny", "label": "Deny"}
      ],
      "created_at": "2024-02-26T10:30:00Z"
    }
  ]
}
```

### Approve/Deny Permission

```bash
curl -X POST http://localhost:8765/api/tasks/{task_id}/acp/permission \
  -H "Content-Type: application/json" \
  -d '{
    "request_id": "perm-1",
    "action": "approve",
    "option_id": "allow"
  }'
```

Actions: `approve`, `deny`

## Security

### File Access Sandboxing

ACP agents can only access files within their workspace (`work_dir`):

- ✅ Allowed: `src/main.go` (relative path within workspace)
- ❌ Blocked: `/etc/passwd` (absolute path)
- ❌ Blocked: `../../../etc/passwd` (path traversal)

### Terminal Commands

Terminal commands run in the agent's workspace and inherit limited environment variables.

### Capability Control

Disable risky operations per agent:

```yaml
agents:
  restricted-agent:
    capabilities:
      terminals: false      # No shell access
      file_access: false    # Read-only (via MCP tools)
      permissions: true
```

## UI Integration

The mesnada web UI displays ACP-specific information:

### Task List
- **ACP Badge**: Tasks show an "A" icon and "ACP" badge
- **Mode Indicator**: Current operation mode (code/ask/architect)

### Task Details Panel
- **Session ID**: Truncated session identifier
- **Agent Name**: Name of the ACP agent
- **Tool Call Count**: Number of tools called during session
- **Mode**: Current operation mode

### Log View
- Color-coded ACP events:
  - Blue: Agent messages
  - Yellow: Tool calls
  - Green: Tool results
  - Gray: Thinking blocks
  - Red: Errors and denied permissions

## Troubleshooting

### Agent Not Starting

**Problem**: Task fails immediately with "no ACP agent configuration found"

**Solution**: Check `config.yaml` for matching agent name:
```yaml
acp:
  agents:
    claude-code:  # This name must match the engine or be set as default_agent
      command: "claude-code"
```

### Permission Timeout

**Problem**: Task hangs waiting for permission approval

**Solution**: Either:
1. Approve via API: `POST /api/tasks/{id}/acp/permission`
2. Enable auto-permission: `auto_permission: true` in config
3. Cancel task: `POST /api/tasks/{id}/cancel`

### File Access Denied

**Problem**: Agent can't read/write files

**Solution**:
1. Check `work_dir` is correct
2. Ensure `file_access: true` in capabilities
3. Verify file paths are relative (not absolute or traversing)

### Terminal Commands Fail

**Problem**: `CreateTerminal` returns "not allowed"

**Solution**: Enable terminal capability:
```yaml
capabilities:
  terminals: true
```

## Examples

See `examples/acp-custom-agent/` for a complete custom ACP agent implementation.

## Further Reading

- [ACP Specification](https://github.com/coder/acp-spec)
- [ACP Go SDK](https://github.com/coder/acp-go-sdk)
- [ACP_AGENTS.md](./ACP_AGENTS.md) - Agent-specific configuration
- [AGENTS.md](../AGENTS.md) - General agent usage
