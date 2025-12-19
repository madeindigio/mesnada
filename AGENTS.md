# mesnada is a MCP server for subagent orchestration

mesnada is a MCP server implementation written in Go, designed to facilitate the orchestration of subagents in a distributed system using Github Copilot CLI `copilot` for command line. It provides a robust framework for managing communication between the main server and its subagents, ensuring efficient data exchange and coordination.

## Cómo desarrollar

1. Comprueba que el proyecto está indexado con las tools de code de Remembrances
Si no lo está, indexa el código de este proyecto
2. Activa la monitorización de código de Remembrances
3. Usa la búsqueda híbrida y la búsqueda de código para localizar información relevante para la tarea
4. Usa las tools de context7 para obtener contexto de cómo funciona una librería que necesites utilizar
5. Usa las tools de búsqueda en internet con Google, Brave y Perplexity para obtener información adicional si es necesario

## Características principales

### Gestión de modelos

- **Configuración centralizada**: Define modelos disponibles y sus propósitos en un archivo YAML
- **Modelo por defecto**: Configura qué modelo usar cuando no se especifica uno
- **Validación automática**: Verifica que los modelos solicitados estén en la lista de modelos válidos

### Identificación de agentes

Cada agente lanzado recibe automáticamente su `task_id` al inicio del prompt:

```
You are the task_id: task-abc12345

[Prompt original del usuario]
```

Esto permite a los agentes:
- Conocer su propia identidad
- Reportar progreso usando su task_id
- Coordinar con otros agentes si es necesario

### Sistema de progreso

Los agentes pueden reportar su progreso en tiempo real usando la tool `set_progress`:

```json
{
  "task_id": "task-abc12345",
  "percentage": 45,
  "description": "Procesando archivos 45/100"
}
```

El sistema:
- Sanitiza automáticamente los valores de porcentaje (elimina símbolos como "%")
- Limita valores entre 0 y 100
- Almacena el progreso con timestamp
- Expone el progreso en `get_stats` y `get_task`

### Comunicación con Copilot CLI

Los agentes se lanzan con los siguientes parámetros:
- `--allow-all-tools`: Acceso completo a todas las tools MCP
- `--no-color`: Salida sin códigos de color para mejor parsing
- `--no-custom-instructions`: Sin instrucciones personalizadas del usuario

El prompt se envía por **stdin** en lugar de como argumento de línea de comandos, permitiendo prompts más largos y complejos.

## Arquitectura

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
