# ACP Agent Configuration Guide

This guide explains how to configure different ACP-compatible agents with mesnada.

## Supported Agents

### Claude Code (Anthropic)

Claude Code is Anthropic's official ACP-compatible coding agent.

**Installation:**
```bash
# Install Claude Code CLI
npm install -g @anthropic-ai/claude-code

# Or use npx (no install needed)
npx @anthropic-ai/claude-code --version
```

**Configuration:**
```yaml
acp:
  default_agent: "claude-code"
  auto_permission: false

  agents:
    claude-code:
      name: "claude-code"
      title: "Claude Code (Sonnet 4.5)"
      command: "claude-code"
      args: ["--acp", "--model", "claude-sonnet-4.5"]
      mode: "code"
      env:
        NO_COLOR: "1"
        ANTHROPIC_API_KEY: "${ANTHROPIC_API_KEY}"

      capabilities:
        terminals: true
        file_access: true
        permissions: true

      mcp_servers:
        - name: "filesystem"
          command: "npx"
          args: ["-y", "@modelcontextprotocol/server-filesystem"]
```

**Usage:**
```bash
# Spawn Claude Code agent
curl -X POST http://localhost:8765/api/spawn \
  -H "Content-Type: application/json" \
  -d '{
    "prompt": "Add user authentication",
    "engine": "acp-claude",
    "work_dir": "/path/to/project",
    "background": true
  }'
```

### OpenAI Codex (via ACP)

OpenAI Codex can be used through an ACP adapter.

**Installation:**
```bash
# Install ACP adapter for Codex
npm install -g @openai/acp-codex-adapter
```

**Configuration:**
```yaml
acp:
  agents:
    codex:
      name: "codex"
      title: "OpenAI Codex"
      command: "acp-codex-adapter"
      args: ["--model", "gpt-5.1-codex"]
      mode: "code"
      env:
        NO_COLOR: "1"
        OPENAI_API_KEY: "${OPENAI_API_KEY}"

      capabilities:
        terminals: true
        file_access: true
        permissions: true

      mcp_servers:
        - name: "github"
          command: "npx"
          args: ["-y", "@modelcontextprotocol/server-github"]
          env:
            GITHUB_TOKEN: "${GITHUB_TOKEN}"
```

**Usage:**
```bash
curl -X POST http://localhost:8765/api/spawn \
  -H "Content-Type: application/json" \
  -d '{
    "prompt": "Implement OAuth2 flow",
    "engine": "acp-codex",
    "work_dir": "/path/to/project",
    "background": true
  }'
```

### Custom ACP Agent

You can implement your own ACP-compatible agent using the ACP Go SDK or any other language.

**Example: Go-based Custom Agent**

See `examples/acp-custom-agent/` for a complete implementation.

**Configuration:**
```yaml
acp:
  agents:
    my-agent:
      name: "my-agent"
      title: "My Custom Agent"
      command: "/path/to/my-acp-agent"
      args: []
      mode: "code"
      env:
        MY_API_KEY: "${MY_API_KEY}"

      capabilities:
        terminals: false    # Disable shell access
        file_access: true
        permissions: true

      mcp_servers:
        - name: "database"
          command: "/path/to/mcp-server-database"
```

## Operation Modes

ACP agents support three operation modes:

### Code Mode (`mode: "code"`)
- **Purpose**: Write, edit, and refactor code
- **Behavior**: Proactive, makes changes directly
- **Best for**: Implementation tasks, bug fixes, refactoring

**Example prompt:**
```
Add unit tests for the authentication module
```

### Ask Mode (`mode: "ask"`)
- **Purpose**: Answer questions, provide explanations
- **Behavior**: Read-only, doesn't modify code
- **Best for**: Code review, understanding, documentation

**Example prompt:**
```
Explain how the authentication flow works
```

### Architect Mode (`mode: "architect"`)
- **Purpose**: Plan and design system architecture
- **Behavior**: Creates plans, suggests approaches
- **Best for**: Design decisions, refactoring strategies

**Example prompt:**
```
Design a microservices architecture for this monolith
```

## MCP Server Configuration

ACP agents can use MCP servers to extend their capabilities.

### Common MCP Servers

#### Filesystem
```yaml
mcp_servers:
  - name: "filesystem"
    command: "npx"
    args: ["-y", "@modelcontextprotocol/server-filesystem"]
    env: {}
```

#### GitHub
```yaml
mcp_servers:
  - name: "github"
    command: "npx"
    args: ["-y", "@modelcontextprotocol/server-github"]
    env:
      GITHUB_TOKEN: "${GITHUB_TOKEN}"
```

#### Brave Search
```yaml
mcp_servers:
  - name: "brave-search"
    command: "npx"
    args: ["-y", "@modelcontextprotocol/server-brave-search"]
    env:
      BRAVE_API_KEY: "${BRAVE_API_KEY}"
```

#### PostgreSQL
```yaml
mcp_servers:
  - name: "postgres"
    command: "npx"
    args: ["-y", "@modelcontextprotocol/server-postgres"]
    env:
      DATABASE_URL: "${DATABASE_URL}"
```

#### Puppeteer (Browser Automation)
```yaml
mcp_servers:
  - name: "puppeteer"
    command: "npx"
    args: ["-y", "@modelcontextprotocol/server-puppeteer"]
    env: {}
```

### Custom MCP Server
```yaml
mcp_servers:
  - name: "my-service"
    command: "/path/to/my-mcp-server"
    args: ["--port", "8080"]
    env:
      MY_API_KEY: "${MY_API_KEY}"
```

## Capability Restrictions

### Limiting Agent Access

For security-sensitive environments, restrict agent capabilities:

```yaml
acp:
  agents:
    restricted-agent:
      capabilities:
        terminals: false      # No shell commands
        file_access: false    # No direct file access (use MCP)
        permissions: true     # Still ask for permission
```

### Read-Only Agent

Create an agent that can only read, not modify:

```yaml
acp:
  agents:
    read-only:
      command: "my-agent"
      mode: "ask"  # Ask mode is inherently read-only
      capabilities:
        terminals: false
        file_access: true  # Can read files
        permissions: false
```

## Environment Variables

### Using Shell Variables

Reference environment variables in config:

```yaml
agents:
  claude-code:
    env:
      ANTHROPIC_API_KEY: "${ANTHROPIC_API_KEY}"
      CUSTOM_VAR: "${MY_CUSTOM_VAR}"
```

Mesnada will substitute `${VAR}` with the value from the shell environment.

### Setting Static Values

```yaml
agents:
  claude-code:
    env:
      NO_COLOR: "1"
      LOG_LEVEL: "debug"
```

## Advanced Configuration

### Multiple Agent Variants

Configure multiple variants of the same agent:

```yaml
acp:
  agents:
    claude-sonnet:
      command: "claude-code"
      args: ["--acp", "--model", "claude-sonnet-4.5"]
      mode: "code"

    claude-opus:
      command: "claude-code"
      args: ["--acp", "--model", "claude-opus-4.5"]
      mode: "architect"

    claude-haiku:
      command: "claude-code"
      args: ["--acp", "--model", "claude-haiku-4.5"]
      mode: "ask"
```

Use via engine selection:
```json
{
  "engine": "acp",
  "model": "claude-opus",
  "prompt": "Design a scalable architecture"
}
```

### Agent-Specific MCP Servers

Different agents can have different MCP servers:

```yaml
acp:
  agents:
    backend-agent:
      mcp_servers:
        - name: "postgres"
          command: "mcp-server-postgres"
        - name: "redis"
          command: "mcp-server-redis"

    frontend-agent:
      mcp_servers:
        - name: "puppeteer"
          command: "mcp-server-puppeteer"
        - name: "figma"
          command: "mcp-server-figma"
```

## Testing Agent Configuration

### Verify Agent is Available

```bash
# Test agent binary
which claude-code

# Test with --version
claude-code --version

# Test ACP handshake manually (advanced)
echo '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":1}}' | claude-code --acp
```

### Test via Mesnada

```bash
# Start mesnada
./mesnada

# In another terminal, spawn a test task
curl -X POST http://localhost:8765/api/spawn \
  -H "Content-Type: application/json" \
  -d '{
    "prompt": "echo \"Hello, ACP!\"",
    "engine": "acp-claude",
    "background": true
  }'

# Check logs
curl http://localhost:8765/api/tasks/<task-id>/log
```

## Troubleshooting

### Agent Not Found

**Error**: `exec: "claude-code": executable file not found in $PATH`

**Solution**: Install the agent or use full path:
```yaml
agents:
  claude-code:
    command: "/usr/local/bin/claude-code"
```

### API Key Issues

**Error**: Authentication failures

**Solution**: Set environment variable before starting mesnada:
```bash
export ANTHROPIC_API_KEY="your-key-here"
./mesnada
```

Or in config:
```yaml
agents:
  claude-code:
    env:
      ANTHROPIC_API_KEY: "${ANTHROPIC_API_KEY}"
```

### MCP Server Fails

**Error**: MCP server not starting

**Solution**: Test MCP server independently:
```bash
# Test stdio MCP server
npx -y @modelcontextprotocol/server-filesystem

# Send initialize message
echo '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}' | npx -y @modelcontextprotocol/server-filesystem
```

## Examples

See complete examples in:
- `examples/acp-custom-agent/` - Custom ACP agent implementation
- `examples/acp-config.yaml` - Sample configuration file
- `config.example.yaml` - Full mesnada configuration with ACP section

## Further Reading

- [ACP_SUPPORT.md](./ACP_SUPPORT.md) - General ACP support in mesnada
- [ACP Specification](https://github.com/coder/acp-spec)
- [ACP Go SDK](https://github.com/coder/acp-go-sdk)
- [Model Context Protocol](https://modelcontextprotocol.io/)
