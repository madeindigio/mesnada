# Custom ACP Agent Example

This directory contains a minimal example of a custom ACP-compatible agent written in Go using the [ACP Go SDK](https://github.com/coder/acp-go-sdk).

## What This Agent Does

This is a simple "echo" agent that:
- Accepts ACP connections via stdin/stdout
- Creates sessions with working directory and MCP servers
- Receives prompts and echoes them back with additional context
- Demonstrates basic session management and streaming updates

## Building

```bash
cd examples/acp-custom-agent
go mod tidy
go build -o custom-acp-agent .
```

## Configuration

Add this agent to your mesnada config (`~/.mesnada/config.yaml`):

```yaml
acp:
  agents:
    custom:
      name: "custom"
      title: "Custom ACP Agent"
      command: "/path/to/custom-acp-agent"
      args: []
      mode: "code"
      env:
        LOG_LEVEL: "info"

      capabilities:
        terminals: false      # This example doesn't implement terminal support
        file_access: false    # This example doesn't implement file operations
        permissions: false

      mcp_servers:
        - name: "example"
          command: "echo"
          args: ["MCP server simulation"]
```

## Usage

### Via MCP Tool (from another agent)

```json
{
  "tool": "spawn_agent",
  "arguments": {
    "prompt": "Hello, custom agent!",
    "engine": "acp-custom",
    "work_dir": "/path/to/workspace",
    "background": true
  }
}
```

### Via HTTP API

```bash
curl -X POST http://localhost:8765/api/spawn \
  -H "Content-Type: application/json" \
  -d '{
    "prompt": "Test the custom agent",
    "engine": "acp-custom",
    "work_dir": "/tmp/test",
    "background": true
  }'
```

### Via CLI (direct testing)

```bash
# Test the agent directly with ACP protocol
echo '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":1,"clientInfo":{"name":"test"}}}' | ./custom-acp-agent
```

## Code Structure

### main.go

The main file implements:

- **CustomAgent**: The main agent structure implementing `acpsdk.Agent`
- **SessionState**: Tracks per-session state (cwd, MCP servers, etc.)
- **Initialize()**: Capability negotiation with the client
- **NewSession()**: Creates a new agent session
- **Prompt()**: Processes user prompts and sends streaming updates
- **Cancel()**: Handles session cancellation

## Extending This Example

### Adding File Operations

Implement the `acpsdk.Client` interface callbacks:

```go
func (a *CustomAgent) ReadTextFile(ctx context.Context, req acpsdk.ReadTextFileRequest) (acpsdk.ReadTextFileResponse, error) {
    // Read file from session.cwd
    content, err := os.ReadFile(filepath.Join(session.cwd, req.Path))
    return acpsdk.ReadTextFileResponse{Content: string(content)}, err
}

func (a *CustomAgent) WriteTextFile(ctx context.Context, req acpsdk.WriteTextFileRequest) (acpsdk.WriteTextFileResponse, error) {
    // Write file to session.cwd
    err := os.WriteFile(filepath.Join(session.cwd, req.Path), []byte(req.Content), 0644)
    return acpsdk.WriteTextFileResponse{}, err
}
```

Then update capabilities:

```yaml
capabilities:
  file_access: true
```

### Adding Terminal Support

Implement terminal callbacks:

```go
func (a *CustomAgent) CreateTerminal(ctx context.Context, req acpsdk.CreateTerminalRequest) (acpsdk.CreateTerminalResponse, error) {
    // Execute command and track terminal
    termID := fmt.Sprintf("term-%d", len(a.terminals))
    // ... start process
    return acpsdk.CreateTerminalResponse{TerminalId: termID}, nil
}

// Also implement: TerminalOutput, WaitForTerminalExit, ReleaseTerminal, KillTerminalCommand
```

Then update capabilities:

```yaml
capabilities:
  terminals: true
```

### Adding Permission Requests

Use the `RequestPermission` callback to ask the user for approval:

```go
func (a *CustomAgent) Prompt(ctx context.Context, req acpsdk.PromptRequest) (acpsdk.PromptResponse, error) {
    // Before doing something sensitive, ask permission
    resp, err := req.RequestPermission(acpsdk.RequestPermissionRequest{
        SessionId: req.SessionId,
        ToolCall: acpsdk.ToolCallInfo{
            Title: ptrString("Delete file config.json"),
        },
        Options: []acpsdk.PermissionOption{
            {OptionId: "allow", Label: "Allow"},
            {OptionId: "deny", Label: "Deny"},
        },
    })

    if err != nil || resp.Outcome.Cancelled != nil {
        // User denied or cancelled
        return acpsdk.PromptResponse{StopReason: acpsdk.StopReasonCancelled}, nil
    }

    // User approved, proceed with operation
    // ...
}
```

### Using MCP Servers

Access MCP servers provided by the client:

```go
func (a *CustomAgent) Prompt(ctx context.Context, req acpsdk.PromptRequest) (acpsdk.PromptResponse, error) {
    session := a.sessions[req.SessionId]

    // MCP servers are available in session.mcpServers
    for _, mcp := range session.mcpServers {
        if mcp.Stdio != nil {
            log.Printf("MCP server available: %s (command: %s)", mcp.Stdio.Name, mcp.Stdio.Command)
        }
    }

    // To actually use them, you'd need to implement MCP client logic
    // See: https://modelcontextprotocol.io/
}
```

## Testing

### Unit Tests

```go
// custom_agent_test.go
func TestCustomAgent_Initialize(t *testing.T) {
    agent := NewCustomAgent()
    resp, err := agent.Initialize(context.Background(), acpsdk.InitializeRequest{
        ProtocolVersion: 1,
    })

    if err != nil {
        t.Fatalf("Initialize failed: %v", err)
    }

    if resp.ServerInfo.Name != "custom-acp-agent" {
        t.Errorf("Expected name 'custom-acp-agent', got %s", resp.ServerInfo.Name)
    }
}
```

### Integration Tests

```bash
# Start mesnada
./mesnada &

# Spawn a test task
curl -X POST http://localhost:8765/api/spawn \
  -H "Content-Type: application/json" \
  -d '{
    "prompt": "Test prompt",
    "engine": "acp-custom",
    "background": true
  }' | jq -r '.task.id'

# Get task output
TASK_ID="<task-id-from-above>"
curl "http://localhost:8765/api/tasks/${TASK_ID}/log"
```

## Further Reading

- [ACP Specification](https://github.com/coder/acp-spec)
- [ACP Go SDK](https://github.com/coder/acp-go-sdk)
- [docs/ACP_SUPPORT.md](../../docs/ACP_SUPPORT.md) - Full ACP support guide
- [docs/ACP_AGENTS.md](../../docs/ACP_AGENTS.md) - Agent configuration guide

## License

This example is provided as-is for educational purposes.
