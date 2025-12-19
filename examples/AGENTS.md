# Instrucciones para Copilot - Mesnada Orchestrator

## Contexto
Tienes acceso al servidor MCP "mesnada" que permite orquestar múltiples instancias de Copilot CLI en paralelo.

## Herramientas disponibles

### spawn_agent
Lanza un agente Copilot en segundo plano. Parámetros:
- `prompt` (requerido): La instrucción para el agente
- `work_dir`: Directorio de trabajo (ruta absoluta)
- `model`: Modelo a usar (claude-sonnet-4, gpt-5.1-codex, etc.)
- `background`: true para ejecutar en segundo plano (default: true)
- `timeout`: Tiempo máximo (ej: "30m", "1h")
- `dependencies`: Lista de task_ids que deben completarse antes
- `tags`: Etiquetas para organizar tareas

### wait_task / wait_multiple
Espera a que las tareas terminen antes de continuar.

### get_task_output
Obtiene la salida de una tarea completada.

## Patrones de uso

### Tareas en paralelo
```
1. spawn_agent para cada tarea con background=true
2. wait_multiple con wait_all=true para esperar todas
3. get_task_output para revisar resultados
```

### Tareas secuenciales
```
1. spawn_agent primera tarea con background=false
2. Si éxito, spawn_agent siguiente tarea
```

### Tareas con dependencias
```
1. spawn_agent tarea A
2. spawn_agent tarea B con dependencies=[A.task_id]
3. spawn_agent tarea C con dependencies=[A.task_id]
4. spawn_agent tarea D con dependencies=[B.task_id, C.task_id]
```

## Ejemplo: Pipeline de desarrollo

Para ejecutar un pipeline completo:
1. Instalar dependencias
2. En paralelo: lint + tests
3. Si pasan, build
4. Si build OK, deploy

```
# Paso 1
task1 = spawn_agent(prompt="npm install", work_dir="/project", background=false)

# Paso 2 (paralelo)
task2 = spawn_agent(prompt="npm run lint", work_dir="/project", dependencies=[task1.id])
task3 = spawn_agent(prompt="npm run test", work_dir="/project", dependencies=[task1.id])

# Paso 3
task4 = spawn_agent(prompt="npm run build", work_dir="/project", dependencies=[task2.id, task3.id])

# Esperar resultado final
wait_task(task4.id)
```
