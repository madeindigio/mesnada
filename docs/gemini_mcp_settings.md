Sí, hay varias formas de hacer esto con Gemini CLI. Aquí están las opciones principales:

## 1. Archivo de configuración JSON

Puedes crear un archivo de configuración JSON y pasárselo directamente:

```bash
# Crear archivo de configuración
cat > gemini-config.json <<EOF
{
  "mcpServers": {
    "remembrances": {
      "command": "node",
      "args": ["/path/to/remembrances-mcp/build/index.js"],
      "env": {
        "DATABASE_URL": "postgresql://..."
      }
    },
    "filesystem": {
      "command": "npx",
      "args": ["-y", "@modelcontextprotocol/server-filesystem", "/allowed/path"]
    }
  }
}
EOF

# Usar con --config
gemini-cli --config gemini-config.json --prompt "Tu prompt inicial"
```

## 2. Variable de entorno

Puedes configurar los servidores MCP mediante variable de entorno:

```bash
export GEMINI_MCP_CONFIG='{"mcpServers": {...}}'
gemini-cli --prompt "Tu prompt"
```

## 3. Argumentos de línea de comandos

Si estás lanzando procesos en segundo plano, puedes usar algo así:

```bash
# Lanzar con toda la configuración de una vez
gemini-cli \
  --model gemini-2.0-flash-exp \
  --prompt "Tu prompt inicial" \
  --config ./gemini-config.json \
  --mcp-server "remembrances:node:/path/to/build/index.js" \
  --mcp-server "filesystem:npx:-y,@modelcontextprotocol/server-filesystem,/path" \
  > output.log 2>&1 &

# Guardar el PID
echo $! > gemini.pid
```

## 4. Script wrapper para procesos en background

Para gestionar múltiples procesos, te recomiendo crear un script:

```bash
#!/bin/bash
# launch-gemini-background.sh

CONFIG_FILE="${1:-./gemini-config.json}"
PROMPT="${2:-Sistema listo para consultas}"
LOG_FILE="${3:-./gemini-$(date +%s).log}"

# Verificar que existe la config
if [ ! -f "$CONFIG_FILE" ]; then
    echo "Config file not found: $CONFIG_FILE"
    exit 1
fi

# Lanzar en background con nohup
nohup gemini-cli \
    --config "$CONFIG_FILE" \
    --prompt "$PROMPT" \
    --interactive=false \
    >> "$LOG_FILE" 2>&1 &

PID=$!
echo "Gemini CLI launched with PID: $PID"
echo "$PID" > "gemini-${PID}.pid"

echo "Log file: $LOG_FILE"
```

Uso:
```bash
chmod +x launch-gemini-background.sh
./launch-gemini-background.sh ./my-config.json "Analiza el código" ./logs/analysis.log
```

## 5. Formato completo del config JSON

```json
{
  "model": "gemini-2.0-flash-exp",
  "temperature": 0.7,
  "topP": 0.9,
  "topK": 40,
  "maxOutputTokens": 8192,
  "mcpServers": {
    "remembrances": {
      "command": "node",
      "args": ["/absolute/path/to/remembrances-mcp/build/index.js"],
      "env": {
        "DATABASE_URL": "postgresql://user:pass@localhost/db",
        "DEBUG": "mcp:*"
      }
    },
    "github": {
      "command": "npx",
      "args": ["-y", "@modelcontextprotocol/server-github"],
      "env": {
        "GITHUB_TOKEN": "ghp_xxx"
      }
    }
  },
  "systemInstruction": "Tu prompt de sistema aquí"
}
```

## 6. Para múltiples instancias paralelas

Si necesitas lanzar varias instancias con diferentes configuraciones:

```bash
# Crear configs específicas
for i in {1..3}; do
    cat > config-worker-$i.json <<EOF
{
  "mcpServers": {
    "remembrances": {
      "command": "node",
      "args": ["./remembrances-mcp/build/index.js"],
      "env": {
        "DATABASE_URL": "postgresql://localhost/worker_$i",
        "WORKER_ID": "$i"
      }
    }
  }
}
EOF
    
    gemini-cli --config config-worker-$i.json \
               --prompt "Worker $i ready" \
               > worker-$i.log 2>&1 &
    echo $! > worker-$i.pid
done
```
