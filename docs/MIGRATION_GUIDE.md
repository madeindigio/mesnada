# Guía de Migración - Nuevas Funcionalidades

Esta guía describe las nuevas funcionalidades añadidas a mesnada y cómo utilizarlas.

## Cambios principales

### 1. Configuración YAML de modelos

**Antes**: No había configuración centralizada de modelos disponibles.

**Ahora**: Puedes definir modelos en un archivo YAML:

```yaml
default_model: "claude-sonnet-4.5"

models:
  - id: "claude-sonnet-4.5"
    description: "Balanced performance and speed"
  - id: "claude-opus-4.5"
    description: "Highest capability for complex tasks"
```

**Ubicación**: `~/.mesnada/config.yaml` o `~/.mesnada/config.json`

**Ejemplo**: Ver `config.example.yaml` en la raíz del proyecto

### 2. Task ID en el prompt

**Antes**: Los agentes no conocían su propio identificador.

**Ahora**: Cada agente recibe automáticamente su task_id:

```
You are the task_id: task-abc12345

[Resto del prompt]
```

**Beneficios**:
- Los agentes pueden reportar su progreso
- Mejor trazabilidad en logs
- Posibilidad de coordinación entre agentes

### 3. Sistema de progreso

**Nueva tool**: `set_progress`

**Uso desde un agente**:

```json
{
  "task_id": "task-abc12345",
  "percentage": 50,
  "description": "Procesando archivos: 50/100"
}
```

**Características**:
- Sanitización automática de porcentajes (elimina símbolos)
- Límite automático entre 0-100
- Timestamp automático
- Visible en `get_stats` y `get_task`

### 4. Flags de Copilot CLI actualizados

**Antes**:
```bash
copilot -p "prompt" --allow-all-tools --allow-all-paths --no-color
```

**Ahora**:
```bash
copilot --allow-all-tools --no-color --no-custom-instructions
# (El prompt se envía por stdin)
```

**Cambios**:
- Eliminado `--allow-all-paths` (ya no necesario)
- Añadido `--no-custom-instructions` (evita instrucciones de usuario)
- Prompt enviado por **stdin** (soporta prompts más largos)

### 5. Información de progreso en estadísticas

**Antes** (`get_stats`):
```json
{
  "total": 10,
  "running": 3,
  "completed": 5
}
```

**Ahora** (`get_stats`):
```json
{
  "total": 10,
  "running": 3,
  "completed": 5,
  "running_progress": {
    "task-abc123": {
      "task_id": "task-abc123",
      "percentage": 45,
      "description": "Procesando archivos",
      "updated_at": "2024-12-19T10:30:00Z"
    }
  }
}
```

## Compatibilidad

### Configuración existente

Si tienes un `config.json` existente, seguirá funcionando. El sistema busca en este orden:

1. `~/.mesnada/config.yaml` (nuevo, preferido)
2. `~/.mesnada/config.json` (existente, compatible)
3. Configuración por defecto

### Tareas existentes

Las tareas creadas con versiones anteriores seguirán funcionando. El campo `progress` es opcional y aparecerá como `null` si no se reporta progreso.

### Tools MCP

Todas las tools existentes son compatibles. La nueva tool `set_progress` es opcional.

## Migración paso a paso

### Paso 1: Actualizar el binario

```bash
cd /www/MCP/mesnada
go mod tidy
go build -o mesnada ./cmd/mesnada
```

### Paso 2: (Opcional) Crear configuración YAML

```bash
cp config.example.yaml ~/.mesnada/config.yaml
```

Edita el archivo para personalizar modelos disponibles.

### Paso 3: Reiniciar el servidor

```bash
# Detener el servidor actual (Ctrl+C)
./mesnada
```

### Paso 4: Verificar

```bash
# En otra terminal
curl http://localhost:8765/mcp
```

Deberías ver la nueva tool `set_progress` en la lista de tools disponibles.

## Uso recomendado para agentes

### Patrón de uso de progreso

```javascript
// 1. Extraer task_id del prompt inicial
// El agente recibe: "You are the task_id: task-abc123"

// 2. Reportar inicio
set_progress({
  task_id: "task-abc123",
  percentage: 0,
  description: "Iniciando tarea"
});

// 3. Reportar progreso durante ejecución
// (cada 10-20% o en hitos importantes)
set_progress({
  task_id: "task-abc123",
  percentage: 25,
  description: "Análisis completado, iniciando procesamiento"
});

set_progress({
  task_id: "task-abc123",
  percentage: 50,
  description: "Procesamiento a mitad"
});

// 4. El sistema marca automáticamente 100% al completar la tarea
```

## Troubleshooting

### Error: "invalid percentage type"

**Causa**: El campo percentage no es un número válido.

**Solución**: El sistema sanitiza automáticamente, pero asegúrate de enviar un número o string con dígitos:
- ✅ `45`
- ✅ `"45"`
- ✅ `"45%"` (se convierte a 45)
- ❌ `"forty-five"`

### Error: "failed to parse YAML config"

**Causa**: El archivo YAML tiene errores de sintaxis.

**Solución**: Verifica la indentación y sintaxis YAML. Usa `config.example.yaml` como referencia.

### El agente no puede usar set_progress

**Causa**: El agente no tiene acceso a la tool MCP.

**Solución**: Verifica que mesnada esté configurado como servidor MCP adicional y que el agente tenga acceso a todas las tools.

## Más información

- Ver `PROGRESS_TRACKING.md` para detalles sobre el sistema de progreso
- Ver `README.md` para documentación completa
- Ver `CHANGELOG.md` para lista completa de cambios
