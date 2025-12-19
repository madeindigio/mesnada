# Mesnada

Orquestador MCP para coordinar múltiples instancias de GitHub Copilot CLI en paralelo y secuencialmente.

## Características

- **Servidor MCP HTTP Streamable**: Un único servidor que coordina todas las instancias de Copilot
- **Ejecución en segundo plano**: Lanza agentes y espera a que terminen sin bloquear
- **Dependencias entre tareas**: Define qué tareas deben completarse antes de iniciar otras
- **Persistencia**: Estado de tareas guardado en disco para recuperación
- **Logs completos**: Cada tarea genera un archivo de log con toda la salida

## Instalación

```bash
cd mesnada
go mod tidy
go build -o mesnada ./cmd/mesnada
```

## Configuración

Mesnada soporta archivos de configuración en formato YAML o JSON. Por defecto busca:
1. `~/.mesnada/config.yaml`
2. `~/.mesnada/config.json`

### Ejemplo de configuración YAML

```yaml
# Modelo por defecto cuando no se especifica
default_model: "claude-sonnet-4.5"

# Lista de modelos disponibles con descripciones
models:
  - id: "claude-sonnet-4.5"
    description: "Balanced performance and speed for general tasks"
  - id: "claude-opus-4.5"
    description: "Highest capability for complex reasoning"
  - id: "gpt-5.1-codex"
    description: "Optimized for code generation"

# Configuración del servidor
server:
  host: "127.0.0.1"
  port: 8765

# Configuración del orquestador
orchestrator:
  store_path: "~/.mesnada/tasks.json"
  log_dir: "~/.mesnada/logs"
  max_parallel: 5
```

Para crear una configuración inicial:

```bash
cp config.example.yaml ~/.mesnada/config.yaml
# Edita el archivo según tus necesidades
```

## Uso

### Iniciar el servidor

```bash
# Con configuración por defecto (puerto 8765)
./mesnada

# Con puerto personalizado
./mesnada --port 9000

# Con configuración personalizada
./mesnada --config ~/.mesnada/config.json
```

### Opciones de línea de comandos

```
--config       Ruta al archivo de configuración
--host         Host del servidor (default: 127.0.0.1)
--port         Puerto del servidor (default: 8765)
--store        Ruta al archivo de tareas
--log-dir      Directorio para logs de agentes
--max-parallel Máximo de agentes en paralelo
--version      Mostrar versión
--init         Inicializar configuración por defecto
```

## Configuración MCP para VSCode Copilot

Añade la siguiente configuración en tu `~/.copilot/mcp-config.json`:

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

O para usarlo directamente con Copilot CLI:

```bash
copilot --additional-mcp-config '{"mcpServers":{"mesnada":{"type":"http","url":"http://127.0.0.1:8765/mcp"}}}'
```

## Herramientas MCP disponibles

### spawn_agent
Lanza un nuevo agente Copilot CLI para ejecutar una tarea.

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
Obtiene información detallada de una tarea.

```json
{
  "task_id": "task-abc123"
}
```

### list_tasks
Lista tareas con filtros opcionales.

```json
{
  "status": ["running", "pending"],
  "tags": ["urgent"],
  "limit": 10
}
```

### wait_task
Espera a que una tarea termine.

```json
{
  "task_id": "task-abc123",
  "timeout": "5m"
}
```

### wait_multiple
Espera a múltiples tareas.

```json
{
  "task_ids": ["task-1", "task-2", "task-3"],
  "wait_all": true,
  "timeout": "10m"
}
```

### cancel_task
Cancela una tarea en ejecución.

```json
{
  "task_id": "task-abc123"
}
```

### get_task_output
Obtiene la salida de una tarea.

```json
{
  "task_id": "task-abc123",
  "tail": true
}
```

### set_progress
Actualiza el progreso de una tarea en ejecución. Esta tool debe ser llamada por el propio agente.

```json
{
  "task_id": "task-abc123",
  "percentage": 45,
  "description": "Processing files 45/100"
}
```

**Nota**: El campo `percentage` acepta valores numéricos o strings. Cualquier carácter no numérico será eliminado automáticamente (ej: "45%" → 45).

### get_stats
Obtiene estadísticas del orquestador, incluyendo el progreso de las tareas en ejecución.

**Respuesta incluye**:
- Contadores por estado (pending, running, completed, failed, cancelled)
- `running_progress`: Mapa con el progreso de cada tarea activa

## Ejemplos de uso desde Copilot

### Ejecutar tareas en paralelo

```
Usa mesnada para ejecutar estas 3 tareas en paralelo:
1. En /project/frontend: "Añade validación al formulario de login"
2. En /project/backend: "Implementa el endpoint /api/users"
3. En /project/docs: "Actualiza la documentación de la API"

Espera a que todas terminen y muéstrame un resumen.
```

### Ejecutar tareas con dependencias

```
Usa mesnada para:
1. Primero ejecuta "npm install" en /project
2. Cuando termine, ejecuta en paralelo:
   - "npm run lint" 
   - "npm run test"
3. Si ambos pasan, ejecuta "npm run build"
```

## Estructura del proyecto

```
mesnada/
├── cmd/mesnada/          # Punto de entrada
├── internal/
│   ├── agent/            # Spawner de procesos Copilot
│   ├── config/           # Configuración
│   ├── orchestrator/     # Coordinador principal
│   ├── server/           # Servidor MCP HTTP
│   └── store/            # Persistencia de tareas
└── pkg/models/           # Modelos de dominio
```

## Desarrollo

```bash
# Ejecutar tests
go test ./... -v

# Build
go build -o mesnada ./cmd/mesnada

# Build con información de versión
go build -ldflags "-X main.version=1.0.0 -X main.commit=$(git rev-parse --short HEAD)" -o mesnada ./cmd/mesnada
```
## Tasks

### build

Compila el binario de mesnada.

```bash
# Get version from last git tag
VERSION=$(git describe --tags --abbrev=0)
go build -ldflags "-X main.version=$VERSION -X main.commit=$(git rev-parse --short HEAD)" -o mesnada ./cmd/mesnada
```
## Licencia

MIT
