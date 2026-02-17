# mesnada is a MCP server for subagent orchestration

mesnada is a MCP server implementation written in Go, designed to facilitate the orchestration of subagents in a distributed system using Github Copilot CLI `copilot` for command line. It provides a robust framework for managing communication between the main server and its subagents, ensuring efficient data exchange and coordination.

## How to develop

1. Check that the project is indexed with Remembrances code tools.
If it is not, index the code of this project.
2. Activate code monitoring with Remembrances.
3. Use hybrid search and code search to locate relevant information for the task.
4. Use context7 tools to get context on how a library you need to use works.
5. Use internet search tools with Google, Brave, and Perplexity to get additional information if necessary.

## Main Features

### Model Management

- **Centralized configuration**: Define available models and their purposes in a YAML file
- **Default model**: Configure which model to use when one is not specified
- **Automatic validation**: Verifies that requested models are in the list of valid models

### Agent Identification

Each launched agent automatically receives its `task_id` at the beginning of the prompt:

```
You are the task_id: task-abc12345

[Original user prompt]
```

This allows agents to:
- Know their own identity
- Report progress using their task_id
- Coordinate with other agents if necessary

### Progress System

Agents can report their progress in real time using the `set_progress` tool:

```json
{
  "task_id": "task-abc12345",
  "percentage": 45,
  "description": "Processing files 45/100"
}
```

The system:
- Automatically sanitizes percentage values (removes symbols like "%")
- Limits values between 0 and 100
- Stores progress with timestamp
- Exposes progress in `get_stats` and `get_task`

### Communication with Copilot CLI

Agents are launched with the following parameters:
- `--allow-all-tools`: Full access to all MCP tools
- `--no-color`: Output without color codes for better parsing
- `--no-custom-instructions`: No custom user instructions

The prompt is sent via **stdin** instead of as a command line argument, allowing for longer and more complex prompts.

## Architecture

```
┌─────────────────────┐
│   MCP Client        │
│  (Main Copilot)     │
└──────────┬──────────┘
           │ HTTP/MCP
           ▼
┌─────────────────────┐
│  Mesnada Server     │
│  ┌───────────────┐  │
│  │ Orchestrator  │  │
│  ├───────────────┤  │
│  │ Config (YAML) │  │
│  ├───────────────┤  │
│  │ Task Store    │  │
│  └───────────────┘  │
└──────────┬──────────┘
           │ stdin/stdout
           ▼
    ┌─────────────┐
    │  Copilot    │
    │  Agent 1    │◄── You are the task_id: XXX
    └─────────────┘
    ┌─────────────┐
    │  Copilot    │
    │  Agent 2    │◄── You are the task_id: YYY
    └─────────────┘
```
