# Sistema de Seguimiento de Progreso

Mesnada incluye un sistema de seguimiento de progreso que permite a los agentes reportar su estado de ejecución en tiempo real.

## Cómo funciona

### 1. Identificación del agente

Cuando se lanza un agente, automáticamente recibe su `task_id` al inicio del prompt:

```
You are the task_id: task-abc12345

[El resto del prompt del usuario]
```

Esto permite al agente conocer su propio identificador para reportar progreso.

### 2. Reportar progreso desde el agente

El agente puede usar la tool MCP `set_progress` para actualizar su estado:

```json
{
  "task_id": "task-abc12345",
  "percentage": 50,
  "description": "Analizando archivos: 50/100 completados"
}
```

### 3. Parámetros

- **task_id** (requerido): El identificador de la tarea
- **percentage** (requerido): Porcentaje de completitud (0-100)
- **description** (opcional): Descripción breve del estado actual

### 4. Sanitización automática

El sistema sanitiza automáticamente el campo `percentage`:

- Si es un número: se usa directamente
- Si es un string con símbolos: se extraen solo los dígitos
  - Ejemplo: `"50%"` → `50`
  - Ejemplo: `"75 percent"` → `75`
- El valor se limita automáticamente entre 0 y 100

## Consultar el progreso

### Desde una tool MCP

Usa `get_stats` para obtener el progreso de todas las tareas en ejecución:

```json
{
  "total": 10,
  "running": 3,
  "completed": 5,
  "failed": 1,
  "pending": 1,
  "running_progress": {
    "task-abc123": {
      "task_id": "task-abc123",
      "percentage": 45,
      "description": "Procesando archivos",
      "updated_at": "2024-12-19T10:30:00Z"
    },
    "task-def456": {
      "task_id": "task-def456",
      "percentage": 78,
      "description": "Ejecutando tests",
      "updated_at": "2024-12-19T10:32:15Z"
    }
  }
}
```

### Desde get_task

La información de progreso también está disponible en `get_task`:

```json
{
  "id": "task-abc123",
  "status": "running",
  "progress": {
    "percentage": 45,
    "description": "Procesando archivos",
    "updated_at": "2024-12-19T10:30:00Z"
  },
  ...
}
```

## Ejemplo completo

### Agente que reporta progreso

Cuando un agente ejecuta una tarea larga, puede reportar su progreso periódicamente:

```javascript
// El agente recibe: "You are the task_id: task-abc123"

// Al inicio
set_progress({
  task_id: "task-abc123",
  percentage: 0,
  description: "Iniciando procesamiento"
});

// Durante la ejecución
for (let i = 0; i < files.length; i++) {
  processFile(files[i]);
  
  if (i % 10 === 0) {
    const percent = Math.floor((i / files.length) * 100);
    set_progress({
      task_id: "task-abc123",
      percentage: percent,
      description: `Procesando archivo ${i}/${files.length}`
    });
  }
}

// Al finalizar (opcional, el sistema marca automáticamente como 100% al completar)
set_progress({
  task_id: "task-abc123",
  percentage: 100,
  description: "Procesamiento completado"
});
```

## Buenas prácticas

1. **Actualizar periódicamente**: Reporta progreso cada cierto tiempo o porcentaje significativo (cada 5-10%)
2. **Descripciones claras**: Usa descripciones concisas que indiquen qué está haciendo el agente
3. **No sobre-reportar**: Evita actualizar el progreso en cada iteración pequeña
4. **Usar task_id correcto**: Siempre usa el task_id proporcionado al inicio del prompt
5. **Progreso realista**: Reporta progreso basado en el trabajo real completado, no estimaciones arbitrarias

## Notas técnicas

- El progreso se almacena en el store junto con el resto de la información de la tarea
- Las actualizaciones de progreso incluyen timestamp automático
- El progreso no afecta el estado de la tarea (pending, running, completed, etc.)
- Si una tarea falla o se cancela, el último progreso reportado queda registrado
