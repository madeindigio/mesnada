---
created: 2026-01-11T22:32:32 (UTC +01:00)
tags: []
source: https://claude.ai/chat/dfdfa531-691a-4756-a6b4-f8dc2d55ec5a
author: 
---

# Modo no interactivo en Claude Code - Claude

> ## Excerpt
> Talk with Claude, an AI assistant from Anthropic

---
Perfecto, ahora tengo información completa de los 4 asistentes. Voy a crear una comparativa completa:

## Comparativa completa de asistentes AI de terminal

### Tabla comparativa general

| Característica | Claude Code | OpenAI Codex | OpenCode.ai | Gemini CLI |
| --- | --- | --- | --- | --- |
| **Empresa** | Anthropic | OpenAI | Open Source (Go) | Google |
| **Licencia** | Propietaria | Propietaria | Open Source | Apache 2.0 |
| **Lenguaje** | \- | \- | Go | TypeScript/Node.js |

### Modo no interactivo

| CLI | Comando | Sintaxis | Ejemplo |
| --- | --- | --- | --- |
| **Claude Code** | `-p` o `--print` | `claude -p "prompt"` | `claude -p "analiza este código"` |
| **Codex** | `exec` o `e` | `codex exec "prompt"` | `codex exec "implementa API REST"` |
| **OpenCode** | `-p` o `--prompt` | `opencode -p "prompt"` | `opencode -p "explica este código"` |
| **Gemini CLI** | `-p` o posicional | `gemini -p "prompt"` o `gemini "prompt"` | `gemini -p "genera tests"` |

### Especificar modelo

| CLI | Método | Ejemplo | Notas |
| --- | --- | --- | --- |
| **Claude Code** | `--model` | `claude -p "task" --model opus` | Aliases: sonnet, opus, haiku |
| **Codex** | `--model` | `codex exec --model gpt-5 "task"` | Soporte OSS con `--oss` |
| **OpenCode** | `--model` o `-m` | `opencode -p "task" -m provider/model` | Formato: `provider/model-id` |
| **Gemini CLI** | `--model` | `gemini -p "task" --model gemini-2.5-pro` | También en config o env vars |

### Gestión de MCP Servers

#### Claude Code:

bash

```
<span><span class="token" style="color: rgb(92, 99, 112); font-style: italic;"># No tiene gestión MCP CLI nativa</span><span>
</span></span><span><span></span><span class="token" style="color: rgb(92, 99, 112); font-style: italic;"># Se configura en ~/.claude/settings.json manualmente</span></span>
```

#### Codex:

bash

```
<span><span class="token" style="color: rgb(92, 99, 112); font-style: italic;"># Gestión completa por CLI</span><span>
</span></span><span>codex mcp list
</span><span><span>codex mcp </span><span class="token" style="color: rgb(97, 175, 239);">add</span><span> context7 -- npx -y @upstash/context7-mcp
</span></span><span><span>codex mcp </span><span class="token" style="color: rgb(97, 175, 239);">add</span><span> my-server --url https://api.example.com/mcp
</span></span><span>codex mcp remove context7
</span><span>codex mcp get context7</span>
```

#### OpenCode:

bash

```
<span><span class="token" style="color: rgb(92, 99, 112); font-style: italic;"># Gestión por CLI</span><span>
</span></span><span><span>opencode mcp </span><span class="token" style="color: rgb(97, 175, 239);">add</span><span>
</span></span><span>opencode mcp list
</span><span><span>opencode mcp </span><span class="token" style="color: rgb(97, 175, 239);">ls</span><span>  </span><span class="token" style="color: rgb(92, 99, 112); font-style: italic;"># alias corto</span><span>
</span></span><span><span>opencode mcp login </span><span class="token" style="color: rgb(97, 175, 239);">&lt;</span><span>server</span><span class="token" style="color: rgb(97, 175, 239);">&gt;</span><span>
</span></span><span><span>opencode mcp </span><span class="token" style="color: rgb(209, 154, 102);">logout</span><span> </span><span class="token" style="color: rgb(97, 175, 239);">&lt;</span><span>server</span><span class="token" style="color: rgb(97, 175, 239);">&gt;</span></span>
```

#### Gemini CLI:

bash

```
<span><span class="token" style="color: rgb(92, 99, 112); font-style: italic;"># No tiene gestión MCP CLI directa</span><span>
</span></span><span><span></span><span class="token" style="color: rgb(92, 99, 112); font-style: italic;"># Se configura en settings.json o archivos TOML</span><span>
</span></span><span><span></span><span class="token" style="color: rgb(92, 99, 112); font-style: italic;"># Soporte MCP mediante configuración manual</span></span>
```

### Configuración de archivos

| CLI | Ubicación config | Formato | MCP Config |
| --- | --- | --- | --- |
| **Claude Code** | `~/.claude/settings.json` | JSON | Sí, en mismo archivo |
| **Codex** | `~/.codex/config.toml` | TOML | Sí, tabla `[mcp_servers]` |
| **OpenCode** | `~/.config/opencode/opencode.json` | JSON | Sí, objeto `mcpServers` |
| **Gemini CLI** | `~/.gemini/settings.json` | JSON | Sí, en configuración |

### Ejemplo de configuración MCP

#### Claude Code (`~/.claude/settings.json`):

json

```
<span><span class="token" style="color: rgb(171, 178, 191);">{</span><span>
</span></span><span><span>  </span><span class="token" style="color: rgb(224, 108, 117);">"mcpServers"</span><span class="token" style="color: rgb(97, 175, 239);">:</span><span> </span><span class="token" style="color: rgb(171, 178, 191);">{</span><span>
</span></span><span><span>    </span><span class="token" style="color: rgb(224, 108, 117);">"context7"</span><span class="token" style="color: rgb(97, 175, 239);">:</span><span> </span><span class="token" style="color: rgb(171, 178, 191);">{</span><span>
</span></span><span><span>      </span><span class="token" style="color: rgb(224, 108, 117);">"command"</span><span class="token" style="color: rgb(97, 175, 239);">:</span><span> </span><span class="token" style="color: rgb(152, 195, 121);">"npx"</span><span class="token" style="color: rgb(171, 178, 191);">,</span><span>
</span></span><span><span>      </span><span class="token" style="color: rgb(224, 108, 117);">"args"</span><span class="token" style="color: rgb(97, 175, 239);">:</span><span> </span><span class="token" style="color: rgb(171, 178, 191);">[</span><span class="token" style="color: rgb(152, 195, 121);">"-y"</span><span class="token" style="color: rgb(171, 178, 191);">,</span><span> </span><span class="token" style="color: rgb(152, 195, 121);">"@upstash/context7-mcp"</span><span class="token" style="color: rgb(171, 178, 191);">]</span><span class="token" style="color: rgb(171, 178, 191);">,</span><span>
</span></span><span><span>      </span><span class="token" style="color: rgb(224, 108, 117);">"env"</span><span class="token" style="color: rgb(97, 175, 239);">:</span><span> </span><span class="token" style="color: rgb(171, 178, 191);">{</span><span>
</span></span><span><span>        </span><span class="token" style="color: rgb(224, 108, 117);">"API_KEY"</span><span class="token" style="color: rgb(97, 175, 239);">:</span><span> </span><span class="token" style="color: rgb(152, 195, 121);">"xxx"</span><span>
</span></span><span><span>      </span><span class="token" style="color: rgb(171, 178, 191);">}</span><span>
</span></span><span><span>    </span><span class="token" style="color: rgb(171, 178, 191);">}</span><span>
</span></span><span><span>  </span><span class="token" style="color: rgb(171, 178, 191);">}</span><span>
</span></span><span><span></span><span class="token" style="color: rgb(171, 178, 191);">}</span></span>
```

#### Codex (`~/.codex/config.toml`):

toml

```
<span><span class="token" style="color: rgb(171, 178, 191);">[</span><span class="token table" style="color: rgb(209, 154, 102);">mcp_servers.context7</span><span class="token" style="color: rgb(171, 178, 191);">]</span><span>
</span></span><span><span></span><span class="token key" style="color: rgb(224, 108, 117);">command</span><span> </span><span class="token" style="color: rgb(171, 178, 191);">=</span><span> </span><span class="token" style="color: rgb(152, 195, 121);">"npx"</span><span>
</span></span><span><span></span><span class="token key" style="color: rgb(224, 108, 117);">args</span><span> </span><span class="token" style="color: rgb(171, 178, 191);">=</span><span> </span><span class="token" style="color: rgb(171, 178, 191);">[</span><span class="token" style="color: rgb(152, 195, 121);">"-y"</span><span class="token" style="color: rgb(171, 178, 191);">,</span><span> </span><span class="token" style="color: rgb(152, 195, 121);">"@upstash/context7-mcp"</span><span class="token" style="color: rgb(171, 178, 191);">]</span><span>
</span></span><span><span></span><span class="token key" style="color: rgb(224, 108, 117);">startup_timeout_sec</span><span> </span><span class="token" style="color: rgb(171, 178, 191);">=</span><span> </span><span class="token" style="color: rgb(209, 154, 102);">10</span><span>
</span></span><span><span></span><span class="token key" style="color: rgb(224, 108, 117);">tool_timeout_sec</span><span> </span><span class="token" style="color: rgb(171, 178, 191);">=</span><span> </span><span class="token" style="color: rgb(209, 154, 102);">60</span><span>
</span></span><span>
</span><span><span></span><span class="token" style="color: rgb(171, 178, 191);">[</span><span class="token table" style="color: rgb(209, 154, 102);">mcp_servers.context7.env</span><span class="token" style="color: rgb(171, 178, 191);">]</span><span>
</span></span><span><span></span><span class="token key" style="color: rgb(224, 108, 117);">API_KEY</span><span> </span><span class="token" style="color: rgb(171, 178, 191);">=</span><span> </span><span class="token" style="color: rgb(152, 195, 121);">"xxx"</span></span>
```

#### OpenCode (`~/.config/opencode/opencode.json`):

json

```
<span><span class="token" style="color: rgb(171, 178, 191);">{</span><span>
</span></span><span><span>  </span><span class="token" style="color: rgb(224, 108, 117);">"mcpServers"</span><span class="token" style="color: rgb(97, 175, 239);">:</span><span> </span><span class="token" style="color: rgb(171, 178, 191);">{</span><span>
</span></span><span><span>    </span><span class="token" style="color: rgb(224, 108, 117);">"context7"</span><span class="token" style="color: rgb(97, 175, 239);">:</span><span> </span><span class="token" style="color: rgb(171, 178, 191);">{</span><span>
</span></span><span><span>      </span><span class="token" style="color: rgb(224, 108, 117);">"type"</span><span class="token" style="color: rgb(97, 175, 239);">:</span><span> </span><span class="token" style="color: rgb(152, 195, 121);">"stdio"</span><span class="token" style="color: rgb(171, 178, 191);">,</span><span>
</span></span><span><span>      </span><span class="token" style="color: rgb(224, 108, 117);">"command"</span><span class="token" style="color: rgb(97, 175, 239);">:</span><span> </span><span class="token" style="color: rgb(152, 195, 121);">"npx"</span><span class="token" style="color: rgb(171, 178, 191);">,</span><span>
</span></span><span><span>      </span><span class="token" style="color: rgb(224, 108, 117);">"args"</span><span class="token" style="color: rgb(97, 175, 239);">:</span><span> </span><span class="token" style="color: rgb(171, 178, 191);">[</span><span class="token" style="color: rgb(152, 195, 121);">"-y"</span><span class="token" style="color: rgb(171, 178, 191);">,</span><span> </span><span class="token" style="color: rgb(152, 195, 121);">"@upstash/context7-mcp"</span><span class="token" style="color: rgb(171, 178, 191);">]</span><span class="token" style="color: rgb(171, 178, 191);">,</span><span>
</span></span><span><span>      </span><span class="token" style="color: rgb(224, 108, 117);">"env"</span><span class="token" style="color: rgb(97, 175, 239);">:</span><span> </span><span class="token" style="color: rgb(171, 178, 191);">[</span><span class="token" style="color: rgb(152, 195, 121);">"API_KEY=xxx"</span><span class="token" style="color: rgb(171, 178, 191);">]</span><span>
</span></span><span><span>    </span><span class="token" style="color: rgb(171, 178, 191);">}</span><span>
</span></span><span><span>  </span><span class="token" style="color: rgb(171, 178, 191);">}</span><span>
</span></span><span><span></span><span class="token" style="color: rgb(171, 178, 191);">}</span></span>
```

#### Gemini CLI (`~/.gemini/settings.json`):

json

```
<span><span class="token" style="color: rgb(171, 178, 191);">{</span><span>
</span></span><span><span>  </span><span class="token" style="color: rgb(224, 108, 117);">"mcpServers"</span><span class="token" style="color: rgb(97, 175, 239);">:</span><span> </span><span class="token" style="color: rgb(171, 178, 191);">{</span><span>
</span></span><span><span>    </span><span class="token" style="color: rgb(224, 108, 117);">"context7"</span><span class="token" style="color: rgb(97, 175, 239);">:</span><span> </span><span class="token" style="color: rgb(171, 178, 191);">{</span><span>
</span></span><span><span>      </span><span class="token" style="color: rgb(224, 108, 117);">"command"</span><span class="token" style="color: rgb(97, 175, 239);">:</span><span> </span><span class="token" style="color: rgb(152, 195, 121);">"npx"</span><span class="token" style="color: rgb(171, 178, 191);">,</span><span>
</span></span><span><span>      </span><span class="token" style="color: rgb(224, 108, 117);">"args"</span><span class="token" style="color: rgb(97, 175, 239);">:</span><span> </span><span class="token" style="color: rgb(171, 178, 191);">[</span><span class="token" style="color: rgb(152, 195, 121);">"-y"</span><span class="token" style="color: rgb(171, 178, 191);">,</span><span> </span><span class="token" style="color: rgb(152, 195, 121);">"@upstash/context7-mcp"</span><span class="token" style="color: rgb(171, 178, 191);">]</span><span class="token" style="color: rgb(171, 178, 191);">,</span><span>
</span></span><span><span>      </span><span class="token" style="color: rgb(224, 108, 117);">"env"</span><span class="token" style="color: rgb(97, 175, 239);">:</span><span> </span><span class="token" style="color: rgb(171, 178, 191);">{</span><span>
</span></span><span><span>        </span><span class="token" style="color: rgb(224, 108, 117);">"API_KEY"</span><span class="token" style="color: rgb(97, 175, 239);">:</span><span> </span><span class="token" style="color: rgb(152, 195, 121);">"xxx"</span><span>
</span></span><span><span>      </span><span class="token" style="color: rgb(171, 178, 191);">}</span><span>
</span></span><span><span>    </span><span class="token" style="color: rgb(171, 178, 191);">}</span><span>
</span></span><span><span>  </span><span class="token" style="color: rgb(171, 178, 191);">}</span><span>
</span></span><span><span></span><span class="token" style="color: rgb(171, 178, 191);">}</span></span>
```

### Perfiles de configuración

| CLI | Soporte Perfiles | Sintaxis | Ejemplo |
| --- | --- | --- | --- |
| **Claude Code** | ❌ No | \- | \- |
| **Codex** | ✅ Sí | `--profile name` | `codex exec --profile prod "task"` |
| **OpenCode** | ❌ No | \- | \- |
| **Gemini CLI** | ❌ No | \- | \- |

### Formatos de salida no interactiva

| CLI | Formatos | Flags | Ejemplo |
| --- | --- | --- | --- |
| **Claude Code** | text, json, stream-json | `--output-format` | `claude -p "task" --output-format json` |
| **Codex** | text, json | `--json` | `codex exec --json "task"` |
| **OpenCode** | text, json, markdown | `-f` o `--format` | `opencode -p "task" -f json` |
| **Gemini CLI** | text, json, stream-json | `--output-format` | `gemini -p "task" --output-format json` |

### Control de permisos en modo no interactivo

| CLI | Control permisos | Ejemplo |
| --- | --- | --- |
| **Claude Code** | `--allowedTools` | `claude -p "task" --allowedTools "Bash,Edit"` |
| **Codex** | `--full-auto`, `--yolo` | `codex exec --full-auto "task"` |
| **OpenCode** | Auto-aprobado en `-p` | Todas las herramientas auto-aprobadas |
| **Gemini CLI** | `--approval-mode`, `--yolo` | `gemini -p "task" --approval-mode yolo` |

### Modelos disponibles

#### Claude Code:

-   Claude Sonnet 4.5, 4
-   Claude Opus 4.5, 4.1
-   Claude Haiku 4.5
-   Alias: `sonnet`, `opus`, `haiku`, `opusplan`

#### Codex:

-   GPT-4.1 (mini, nano, full)
-   GPT-5
-   Modelos OSS via Ollama

#### OpenCode:

-   OpenAI (GPT-4.1, GPT-5)
-   Anthropic Claude
-   Google Gemini
-   AWS Bedrock
-   Groq
-   Azure OpenAI
-   OpenRouter
-   Ollama (local)

#### Gemini CLI:

-   Gemini 2.5 Pro
-   Gemini 2.5 Flash
-   Gemini 1.5 Pro/Flash
-   Auto mode (selección automática)

### Comandos personalizados

| CLI | Soporte | Ubicación | Formato |
| --- | --- | --- | --- |
| **Claude Code** | ✅ Sí | `.claude/commands/` | Markdown |
| **Codex** | ✅ Sí | `~/.codex/commands/` | Markdown |
| **OpenCode** | ✅ Sí | `~/.config/opencode/commands/` | Markdown |
| **Gemini CLI** | ✅ Sí | `.gemini/commands/` | TOML |

### Características especiales

| CLI | Características únicas |
| --- | --- |
| **Claude Code** | \- Integración con claude.ai  
\- Memory system  
\- Deep Research mode  
\- Artifacts support |
| **Codex** | \- Sistema de perfiles robusto  
\- Sandbox avanzado (Seatbelt/Landlock)  
\- Gestión MCP más completa  
\- GitHub Action |
| **OpenCode** | \- Open Source  
\- Plan mode (review sin cambios)  
\- LSP integration  
\- Multi-proveedor más flexible  
\- Servidor headless (`opencode serve`) |
| **Gemini CLI** | \- Open Source (Apache 2.0)  
\- Checkpointing  
\- GEMINI.md context files  
\- Token caching  
\- VS Code integration  
\- Sandboxing con Docker |

### Pricing y autenticación

| CLI | Autenticación | Modelos gratuitos |
| --- | --- | --- |
| **Claude Code** | Claude Pro/Max subscription | No (requiere suscripción) |
| **Codex** | OpenAI API key o ChatGPT OAuth | No (pago por uso API) |
| **OpenCode** | Configuración por proveedor | Sí (Ollama local) |
| **Gemini CLI** | Google Account, API Key, Vertex AI | Sí (Gemini 2.5 Pro gratis tier: 60 req/min) |

### Ejemplo completo de uso no interactivo

#### Claude Code:

bash

```
<span><span>claude -p </span><span class="token" style="color: rgb(152, 195, 121);">"implementa tests unitarios"</span><span> </span><span class="token" style="color: rgb(171, 178, 191);">\</span><span>
</span></span><span><span>  --model opus </span><span class="token" style="color: rgb(171, 178, 191);">\</span><span>
</span></span><span><span>  --allowedTools </span><span class="token" style="color: rgb(152, 195, 121);">"Bash,Read,Edit,Write"</span><span> </span><span class="token" style="color: rgb(171, 178, 191);">\</span><span>
</span></span><span><span>  --output-format json </span><span class="token" style="color: rgb(171, 178, 191);">\</span><span>
</span></span><span><span>  --max-budget-usd </span><span class="token" style="color: rgb(209, 154, 102);">1.0</span></span>
```

#### Codex:

bash

```
<span><span>codex </span><span class="token" style="color: rgb(209, 154, 102);">exec</span><span> </span><span class="token" style="color: rgb(152, 195, 121);">"implementa tests unitarios"</span><span> </span><span class="token" style="color: rgb(171, 178, 191);">\</span><span>
</span></span><span><span>  --model gpt-5 </span><span class="token" style="color: rgb(171, 178, 191);">\</span><span>
</span></span><span><span>  --profile production </span><span class="token" style="color: rgb(171, 178, 191);">\</span><span>
</span></span><span><span>  --full-auto </span><span class="token" style="color: rgb(171, 178, 191);">\</span><span>
</span></span><span>  --json</span>
```

#### OpenCode:

bash

```
<span><span>opencode -p </span><span class="token" style="color: rgb(152, 195, 121);">"implementa tests unitarios"</span><span> </span><span class="token" style="color: rgb(171, 178, 191);">\</span><span>
</span></span><span><span>  -m anthropic/claude-sonnet-4 </span><span class="token" style="color: rgb(171, 178, 191);">\</span><span>
</span></span><span><span>  -f json </span><span class="token" style="color: rgb(171, 178, 191);">\</span><span>
</span></span><span><span>  -q  </span><span class="token" style="color: rgb(92, 99, 112); font-style: italic;"># quiet mode, sin spinner</span></span>
```

#### Gemini CLI:

bash

```
<span><span>gemini -p </span><span class="token" style="color: rgb(152, 195, 121);">"implementa tests unitarios"</span><span> </span><span class="token" style="color: rgb(171, 178, 191);">\</span><span>
</span></span><span><span>  --model gemini-2.5-pro </span><span class="token" style="color: rgb(171, 178, 191);">\</span><span>
</span></span><span><span>  --output-format json </span><span class="token" style="color: rgb(171, 178, 191);">\</span><span>
</span></span><span>  --approval-mode yolo</span>
```

### Integración con pipes

Todos soportan pipes de forma similar:

bash

```
<span><span class="token" style="color: rgb(92, 99, 112); font-style: italic;"># Claude Code</span><span>
</span></span><span><span></span><span class="token" style="color: rgb(97, 175, 239);">cat</span><span> file.py </span><span class="token" style="color: rgb(97, 175, 239);">|</span><span> claude -p </span><span class="token" style="color: rgb(152, 195, 121);">"analiza este código"</span><span>
</span></span><span>
</span><span><span></span><span class="token" style="color: rgb(92, 99, 112); font-style: italic;"># Codex</span><span>
</span></span><span><span></span><span class="token" style="color: rgb(97, 175, 239);">cat</span><span> file.py </span><span class="token" style="color: rgb(97, 175, 239);">|</span><span> codex </span><span class="token" style="color: rgb(209, 154, 102);">exec</span><span> -
</span></span><span>
</span><span><span></span><span class="token" style="color: rgb(92, 99, 112); font-style: italic;"># OpenCode</span><span>
</span></span><span><span></span><span class="token" style="color: rgb(97, 175, 239);">cat</span><span> file.py </span><span class="token" style="color: rgb(97, 175, 239);">|</span><span> opencode -p </span><span class="token" style="color: rgb(152, 195, 121);">"analiza este código"</span><span>
</span></span><span>
</span><span><span></span><span class="token" style="color: rgb(92, 99, 112); font-style: italic;"># Gemini CLI</span><span>
</span></span><span><span></span><span class="token" style="color: rgb(97, 175, 239);">cat</span><span> file.py </span><span class="token" style="color: rgb(97, 175, 239);">|</span><span> gemini -p </span><span class="token" style="color: rgb(152, 195, 121);">"analiza este código"</span></span>
```

## Recomendaciones de uso

### Claude Code

**Mejor para:**

-   Usuarios de Claude Pro/Max
-   Integración con ecosistema Anthropic
-   Necesitas Deep Research o Artifacts

### Codex

**Mejor para:**

-   Workflows complejos con perfiles
-   Necesitas control granular de sandbox
-   Gestión MCP intensiva
-   Integración con GitHub Actions

### OpenCode

**Mejor para:**

-   Open source y customización
-   Multi-proveedor (incluyendo local con Ollama)
-   Necesitas modo servidor headless
-   Quieres contribuir al proyecto

### Gemini CLI

**Mejor para:**

-   Usuarios de Google Cloud/Workspace
-   Tier gratuito generoso (60 req/min)
-   Open source (Apache 2.0)
-   Integración con ecosistema Google
-   Context files (GEMINI.md) y checkpointing
