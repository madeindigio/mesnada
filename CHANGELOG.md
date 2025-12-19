# Changelog

## [Unreleased]

### Added

- **Configuración YAML de modelos**: Soporte para archivos de configuración YAML (`config.yaml`) con listado de modelos válidos y sus descripciones
- **Modelo por defecto**: Configuración de modelo por defecto cuando no se especifica uno en el spawn
- **Prefijo de task_id en prompts**: Cada agente recibe automáticamente su task_id al inicio del prompt con el formato "You are the task_id: XXX"
- **Sistema de progreso**: Nueva tool `set_progress` para que los agentes reporten su progreso
- **Entrada por stdin**: Los prompts se envían a copilot-cli por stdin en lugar de argumentos de línea de comandos
- **Flags actualizados de copilot-cli**: Uso de `--allow-all-tools`, `--no-color` y `--no-custom-instructions`
- **Sanitización de progreso**: Limpieza automática de valores de porcentaje para extraer solo números
- **Progreso en estadísticas**: `get_stats` ahora incluye información de progreso de las tareas en ejecución

### Changed

- Configuración ahora soporta tanto JSON como YAML (prioridad a YAML)
- El spawner ahora envía el prompt completo (incluyendo task_id) por stdin
- La estructura `Task` ahora incluye campo `Progress` opcional
- La estructura `Stats` ahora incluye `RunningProgress` con detalles de progreso por tarea

### Technical Details

- Añadida dependencia `gopkg.in/yaml.v3` para parsing de YAML
- Nuevo método `SetProgress` en el orchestrator
- Función `expandHome` para expandir `~` en rutas de configuración
- Validación de modelos con `ValidateModel` y `GetModelByID`

## [1.0.0] - 2024-12-19

### Initial Release

- Servidor MCP HTTP para orquestación de agentes
- Ejecución de múltiples instancias de GitHub Copilot CLI
- Sistema de dependencias entre tareas
- Ejecución en background y foreground
- Persistencia de estado en disco
- Logs completos por tarea
