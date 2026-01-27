# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- **Engine parameter in spawn_agent**: New `engine` parameter in the MCP `spawn_agent` tool with disambiguated enum: 'copilot', 'claude-code' (executes `claude`), 'gemini-cli' (executes `gemini`), 'opencode'
- **Auto-detection of engine**: If engine is not specified but model is, the system automatically detects which engine to use based on model configuration and verifies the binary is installed
- **Dynamic model description**: The `model` parameter description in `spawn_agent` is now dynamically generated from YAML configuration, showing available models for each engine
- **Model suggestions on errors**: When an agent fails, the response includes `available_models` and `suggestion` with available models for that engine, facilitating retries with alternative models
- **Per-engine model configuration**: Support for defining available models specific to each CLI engine (copilot, claude, gemini, opencode) in the YAML configuration file
- **Per-engine model validation**: The system now validates that the specified model is compatible with the selected engine
- **Improved error messages**: When trying to use an invalid model, the error shows the list of available models for that engine
- **YAML model configuration**: Support for YAML configuration files (`config.yaml`) with valid models and their descriptions
- **Per-engine default model**: Each engine can have its own default model in the configuration
- **Dynamic MCP schema**: The model enum in the MCP API is dynamically generated from the configuration
- **Task ID prefix in prompts**: Each agent automatically receives its task_id at the beginning of the prompt with the format "You are the task_id: XXX"
- **Progress system**: New `set_progress` tool for agents to report their progress
- **Stdin input**: Prompts are sent to copilot-cli via stdin instead of command line arguments
- **Updated copilot-cli flags**: Use of `--allow-all-tools`, `--no-color` and `--no-custom-instructions`
- **Progress sanitization**: Automatic cleaning of percentage values to extract only numbers
- **Progress in statistics**: `get_stats` now includes progress information for running tasks
- **Configuration documentation**: Complete new guide in `docs/MODEL_CONFIGURATION.md`

### Changed

- **Full backward compatibility**: Configurations without `engines` section continue to work with the global models list
- Configuration now supports both JSON and YAML (YAML has priority)
- The spawner now sends the complete prompt (including task_id) via stdin
- `Task` structure now includes optional `Progress` field
- `Stats` structure now includes `RunningProgress` with progress details per task
- `Config` structure now includes the `Engines` map for per-engine configuration

### Technical Details

- New `EngineConfig` structure in `internal/config/config.go`
- New methods: `GetModelsForEngine`, `ValidateModelForEngine`, `GetDefaultModelForEngine`, `GetModelIDsForEngine`
- `getAllModelIDs()` method in server to generate dynamic model enum
- Automatic model validation in `toolSpawnAgent` before creating the task
- Added `gopkg.in/yaml.v3` dependency for YAML parsing
- New `SetProgress` method in the orchestrator
- `expandHome` function to expand `~` in configuration paths
- Model validation with `ValidateModel` and `GetModelByID`

## [3.5.2] - 2024-01-27

### Added

- Favicon and logo in web UI

## [3.5.1] - 2024-01-27

### Added

- Support for "persona" or specific instructions for subagents
- Persona system allows applying custom instructions/roles to agents

### Changed

- Removed `@` prefix from default_mcp_config

## [3.4.0] - 2024-01-27

### Added

- Config template generation
- Ability to generate configuration in custom paths

## [3.3.4] - 2024-01-27

### Fixed

- Broken web UI from previous changes

## [3.3.3] - 2024-01-27

### Added

- MCP support for Gemini engine

## [3.2.9] - 2024-01-27

### Fixed

- Enabled MCP config for OpenCode engine

## [3.2.8] - 2024-01-27

### Fixed

- Working Gemini and OpenCode subagents without stderr in logs

## [3.2.2] - 2024-01-27

### Added

- Stdio mode support
- Updated tool descriptions

## [3.0.1] - 2024-01-27

### Fixed

- Removed autoscroll from subagent list

## [3.0.0] - 2024-01-27

### Added

- Support for OpenCode and Gemini CLI engines
- Dependencies task context

## [2.0.1] - 2024-01-27

### Changed

- Moved autoscroll to header in the web UI

## [2.0.0] - 2024-01-27

### Added

- Initial support for Claude CLI
- Display engine information in web UI
- Improved details in web UI

## [1.0.1] - 2024-01-27

### Fixed

- Web UI code was not embedded

## [1.0.0] - 2024-01-27

### Added

- Web UI for managing subagents
- Real-time task monitoring
- Task log viewer

## [0.2.4] - 2024-01-27

### Added

- Task information logging
- Improved log output

### Fixed

- Minor bug corrections

## [0.2.1] - 2024-01-27

### Added

- MCP configuration file support

### Changed

- Use relative paths in configuration

### Fixed

- Minor bugfixes

## [0.1.0] - 2024-01-27

### Added

- Initial working version
- MCP HTTP server for agent orchestration
- Multiple GitHub Copilot CLI instances execution
- Task dependency system
- Background and foreground execution modes
- Disk state persistence
- Complete per-task logging
