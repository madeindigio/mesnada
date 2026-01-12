# Model Configuration by Engine

## Overview

Mesnada supports configuring available models on a per-engine basis. This allows you to define which models are valid for each CLI engine (copilot, claude, gemini, opencode) while maintaining backward compatibility with legacy configurations.

## Configuration Structure

### Engine-Specific Models (Recommended)

Define models for each engine separately:

```yaml
# Default model when none is specified
default_model: "claude-sonnet-4.5"

# Global/shared models (used when engines section is not defined)
models:
  - id: "claude-sonnet-4.5"
    description: "Balanced performance and speed"
  # ... more models

# Engine-specific configurations
engines:
  copilot:
    default_model: "gpt-5.1-codex"
    models:
      - id: "gpt-5.1-codex"
        description: "Optimized for code generation"
      - id: "gpt-5.2"
        description: "Latest GPT model"
      # ... more GPT models
  
  claude:
    default_model: "claude-sonnet-4.5"
    models:
      - id: "claude-sonnet-4.5"
        description: "Balanced performance"
      - id: "claude-opus-4.5"
        description: "Highest capability"
      # ... more Claude models
  
  gemini:
    default_model: "gemini-3-pro-preview"
    models:
      - id: "gemini-3-pro-preview"
        description: "Google's latest model"
      # ... more Gemini models
  
  opencode:
    default_model: "claude-sonnet-4.5"
    models:
      - id: "claude-sonnet-4.5"
        description: "Via OpenCode"
      # ... more models supported by OpenCode
```

### Legacy Configuration (Backward Compatible)

If you don't define the `engines` section, all engines will use the global `models` list:

```yaml
default_model: "claude-sonnet-4.5"

models:
  - id: "claude-sonnet-4.5"
    description: "Balanced performance"
  - id: "gpt-5.1-codex"
    description: "Coding model"
  # All engines can use any model from this list
```

## Validation Behavior

### With Engine-Specific Configuration

When you spawn an agent with a specific engine, the model is validated against that engine's model list:

```bash
# ✓ Valid: gpt-5.1-codex is in copilot's model list
mesnada spawn --engine copilot --model gpt-5.1-codex "Task description"

# ✗ Invalid: claude-opus-4.5 is NOT in copilot's model list
mesnada spawn --engine copilot --model claude-opus-4.5 "Task description"
# Error: invalid model 'claude-opus-4.5' for engine 'copilot'. 
#        Available models: [gpt-5.1-codex, gpt-5.2, gpt-5.1, ...]
```

### With Legacy Configuration

All models are available for all engines (no validation):

```bash
# All these are valid with legacy config
mesnada spawn --engine copilot --model claude-opus-4.5 "Task"
mesnada spawn --engine claude --model gpt-5.1-codex "Task"
```

## Error Messages

When you use an invalid model, you'll get a helpful error message listing the available models:

```
Error: invalid model 'gpt-5.1-codex' for engine 'claude'. 
Available models: [claude-sonnet-4.5, claude-opus-4.5, claude-haiku-4.5, 
                   claude-sonnet-4, sonnet, opus, haiku]
```

## MCP Tool Schema

The `spawn_agent` MCP tool dynamically generates its model enum from your configuration:

- **With engine-specific config**: Includes all models from all engines plus global models
- **With legacy config**: Includes only the global models list

This ensures IDEs and MCP clients always show all possible models in autocomplete.

## Default Models

Each engine can have its own default model:

```yaml
engines:
  copilot:
    default_model: "gpt-5.1-codex"  # Used when model is not specified
  claude:
    default_model: "claude-sonnet-4.5"
```

If an engine-specific default is not configured, falls back to the global `default_model`.

## Migration Guide

### From Legacy to Engine-Specific

1. Keep your existing `models` section for backward compatibility
2. Add the `engines` section with per-engine models
3. Test your configuration:
   ```bash
   mesnada --config your-config.yaml --init
   ```

### Example Migration

**Before (legacy):**
```yaml
models:
  - id: "claude-sonnet-4.5"
  - id: "gpt-5.1-codex"
  - id: "gemini-3-pro-preview"
```

**After (engine-specific):**
```yaml
# Keep for backward compatibility
models:
  - id: "claude-sonnet-4.5"
  - id: "gpt-5.1-codex"
  - id: "gemini-3-pro-preview"

# Add engine-specific configuration
engines:
  copilot:
    models:
      - id: "gpt-5.1-codex"
  claude:
    models:
      - id: "claude-sonnet-4.5"
  gemini:
    models:
      - id: "gemini-3-pro-preview"
```

## Benefits

1. **Type Safety**: Prevents using incompatible models with engines
2. **Better Errors**: Clear messages showing available models
3. **Flexible Configuration**: Each engine can have its own model list
4. **Backward Compatible**: Existing configs work without changes
5. **Self-Documenting**: Engine sections clearly show what models work where
