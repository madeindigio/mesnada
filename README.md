# Mesnada

MCP orchestrator to coordinate multiple instances of GitHub Copilot CLI in parallel and sequentially.

## Features

- **Streamable MCP HTTP Server**: A single server that coordinates all Copilot instances
- **Background execution**: Launches agents and waits for them to finish without blocking
- **Task dependencies**: Define which tasks must be completed before starting others
- **Persistence**: Task state saved to disk for recovery
- **Full logs**: Each task generates a log file with all output

![Mesnada Web UI](https://github.com/madeindigio/mesnada/blob/main/examples/image.png?raw=true)

## Installation

```bash
cd mesnada
go mod tidy
go build -o mesnada ./cmd/mesnada
```

## Configuration

Mesnada supports configuration files in YAML or JSON format. By default, it looks for:
1. `~/.mesnada/config.yaml`

### Example YAML configuration

```yaml
# Default model when not specified
default_model: "claude-sonnet-4.5"

# List of available models with descriptions
models:
  - id: "claude-sonnet-4.5"
    description: "Balanced performance and speed for general tasks"
  - id: "claude-opus-4.5"
    description: "Highest capability for complex reasoning"
  - id: "gpt-5.1-codex"
    description: "Optimized for code generation"

# Server configuration
server:
  host: "127.0.0.1"
  port: 8765

# Orchestrator configuration
orchestrator:
  store_path: "~/.mesnada/tasks.json"
  log_dir: "~/.mesnada/logs"
  max_parallel: 5

  # (Optional) Additional MCP config that will be passed to *all* Copilot CLI instances.
  # Translates to: copilot --additional-mcp-config <value>
  # If pointing to a file, use the prefix @ (e.g. @.github/mcp-config.json)
  default_mcp_config: "@.github/mcp-config.json"
```

To create an initial configuration:

```bash
cp config.example.yaml ~/.mesnada/config.yaml
# Edit the file as needed
```

## Usage

### Start the server

```bash
# With default configuration (HTTP mode on port 8765)
./mesnada

# With custom port
./mesnada --port 9000

# With custom configuration
./mesnada --config ~/.mesnada/config.json

# In stdio mode (for MCP clients that use stdio transport)
./mesnada --stdio
```

### Command line options

```
--config       Path to the configuration file
--host         Server host (default: 127.0.0.1)
--port         Server port (default: 8765)
--store        Path to the tasks file
--log-dir      Directory for agent logs
--max-parallel Maximum parallel agents
--stdio        Use stdio transport instead of HTTP
--version      Show version
--init         Initialize default configuration
```

## MCP Configuration

### HTTP Transport (Default)

Add the following configuration to your `~/.copilot/mcp-config.json`:

```json
{
  "mcpServers": {
    "mesnada": {
      "type": "http",
      "url": "http://127.0.0.1:8765/mcp"
    }
  }
}
```

Or to use it directly with Copilot CLI:

```bash
copilot --additional-mcp-config '{"mcpServers":{"mesnada":{"type":"http","url":"http://127.0.0.1:8765/mcp"}}}'
```

### Stdio Transport

For MCP clients that support stdio transport (like Claude Desktop), add this to your MCP settings:

```json
{
  "mcpServers": {
    "mesnada": {
      "command": "/path/to/mesnada",
      "args": ["--stdio"]
    }
  }
}
```

Example for Claude Desktop (`~/Library/Application Support/Claude/claude_desktop_config.json` on macOS):

```json
{
  "mcpServers": {
    "mesnada": {
      "command": "/usr/local/bin/mesnada",
      "args": ["--stdio", "--config", "~/.mesnada/config.yaml"]
    }
  }
}
```

## Available MCP tools

### spawn_agent
Launches a new Copilot CLI agent to execute a task.

```json
{
  "prompt": "Fix the bug in main.js",
  "work_dir": "/path/to/project",
  "model": "claude-sonnet-4",
  "background": true,
  "timeout": "30m",
  "dependencies": ["task-abc123"],
  "tags": ["bugfix", "urgent"]
}
```

### get_task
Gets detailed information about a task.

```json
{
  "task_id": "task-abc123"
}
```

### list_tasks
Lists tasks with optional filters.

```json
{
  "status": ["running", "pending"],
  "tags": ["urgent"],
  "limit": 10
}
```

### wait_task
Waits for a task to finish.

```json
{
  "task_id": "task-abc123",
  "timeout": "5m"
}
```

### wait_multiple
Waits for multiple tasks.

```json
{
  "task_ids": ["task-1", "task-2", "task-3"],
  "wait_all": true,
  "timeout": "10m"
}
```

### cancel_task
Cancels a running task.

```json
{
  "task_id": "task-abc123"
}
```

### get_task_output
Gets the output of a task.

```json
{
  "task_id": "task-abc123",
  "tail": true
}
```

### set_progress
Updates the progress of a running task. This tool should be called by the agent itself.

```json
{
  "task_id": "task-abc123",
  "percentage": 45,
  "description": "Processing files 45/100"
}
```

**Note**: The `percentage` field accepts numeric values or strings. Any non-numeric character will be automatically removed (e.g., "45%" → 45).

### get_stats
Gets orchestrator statistics, including the progress of running tasks.

**Response includes**:
- Counters by status (pending, running, completed, failed, cancelled)
- `running_progress`: Map with the progress of each active task

## Usage examples from Copilot

### Run tasks in parallel

```
Use mesnada to run these 3 tasks in parallel:
1. In /project/frontend: "Add validation to the login form"
2. In /project/backend: "Implement the /api/users endpoint"
3. In /project/docs: "Update the API documentation"

Wait for all to finish and show me a summary.
```

### Run tasks with dependencies

```
Use mesnada to:
1. First run "npm install" in /project
2. When finished, run in parallel:
   - "npm run lint"
   - "npm run test"
3. If both pass, run "npm run build"
```

## Project structure

```
mesnada/
├── cmd/mesnada/          # Entry point
├── internal/
│   ├── agent/            # Copilot process spawner
│   ├── config/           # Configuration
│   ├── orchestrator/     # Main coordinator
│   ├── server/           # MCP HTTP server
│   └── store/            # Task persistence
└── pkg/models/           # Domain models
```

## Development

```bash
# Run tests
go test ./... -v

# Build
go build -o mesnada ./cmd/mesnada

# Build with version info
go build -ldflags "-X main.version=1.0.0 -X main.commit=$(git rev-parse --short HEAD)" -o mesnada ./cmd/mesnada
```
## Tasks

### build

Compiles the mesnada binary.

```bash
# Get version from last git tag
VERSION=$(git describe --tags --abbrev=0)
go build -ldflags "-X main.version=$VERSION -X main.commit=$(git rev-parse --short HEAD)" -o mesnada ./cmd/mesnada
```

### build-and-copy

Compiles the mesnada binary. Copies the binary to the specified path folder.

```bash
# Get version from last git tag
VERSION=$(git describe --tags --abbrev=0)
go build -ldflags "-X main.version=$VERSION -X main.commit=$(git rev-parse --short HEAD)" -o mesnada ./cmd/mesnada
rm -f ~/bin/mesnada
cp mesnada ~/bin/mesnada
```

### release

Compiles binaries for multiple platforms (Linux x64, Windows x64, macOS aarch64) and generates compressed releases in the `dist` folder.

```bash
# Create dist folder
mkdir -p dist

# Get version from last git tag
VERSION=$(git describe --tags --abbrev=0)

# Function to build and zip
build_and_zip() {
    local os=$1
    local arch=$2
    local suffix=$3
    local ext=$4
    
    echo "Building for $os-$arch..."
    GOOS=$os GOARCH=$arch go build -ldflags "-X main.version=$VERSION -X main.commit=$(git rev-parse --short HEAD)" -o dist/mesnada-$suffix$ext ./cmd/mesnada
    
    echo "Zipping mesnada-$suffix$ext..."
    cd dist
    zip mesnada-$suffix.zip mesnada-$suffix$ext
    rm mesnada-$suffix$ext
    cd ..
}

# Linux x64
build_and_zip linux amd64 linux-x64 ""

# Windows x64
build_and_zip windows amd64 windows-x64 ".exe"

# macOS aarch64
build_and_zip darwin arm64 darwin-arm64 ""

echo "Release builds completed in dist/"
```
## License

MIT
