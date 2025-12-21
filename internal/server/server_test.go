package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/sevir/mesnada/internal/orchestrator"
)

func setupTestServer(t *testing.T) (*Server, func()) {
	tmpDir, err := os.MkdirTemp("", "mesnada-server-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	orch, err := orchestrator.New(orchestrator.Config{
		StorePath:   filepath.Join(tmpDir, "tasks.json"),
		LogDir:      filepath.Join(tmpDir, "logs"),
		MaxParallel: 2,
	})
	if err != nil {
		os.RemoveAll(tmpDir)
		t.Fatalf("Failed to create orchestrator: %v", err)
	}

	srv := New(Config{
		Addr:         ":0",
		Orchestrator: orch,
	})

	cleanup := func() {
		orch.Shutdown()
		os.RemoveAll(tmpDir)
	}

	return srv, cleanup
}

func TestHealthEndpoint(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()

	srv.httpServer.Handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if response["status"] != "healthy" {
		t.Errorf("Expected status 'healthy', got '%v'", response["status"])
	}
}

func TestMCPInitialize(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	reqBody := JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "initialize",
		Params:  json.RawMessage(`{}`),
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", "/mcp", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	srv.httpServer.Handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response JSONRPCResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if response.Error != nil {
		t.Errorf("Expected no error, got: %+v", response.Error)
	}

	result := response.Result.(map[string]interface{})
	if result["protocolVersion"] != mcpVersion {
		t.Errorf("Expected protocol version %s, got %v", mcpVersion, result["protocolVersion"])
	}

	serverInfo := result["serverInfo"].(map[string]interface{})
	if serverInfo["name"] != "mesnada" {
		t.Errorf("Expected server name 'mesnada', got %v", serverInfo["name"])
	}

	// Check session header
	sessionID := w.Header().Get("Mcp-Session-Id")
	if sessionID == "" {
		t.Error("Expected Mcp-Session-Id header")
	}
}

func TestMCPToolsList(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	reqBody := JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "tools/list",
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", "/mcp", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	srv.httpServer.Handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response JSONRPCResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	result := response.Result.(map[string]interface{})
	tools := result["tools"].([]interface{})

	if len(tools) == 0 {
		t.Error("Expected at least one tool")
	}

	// Check for expected tools
	toolNames := make(map[string]bool)
	var listTasksTool map[string]interface{}
	for _, tool := range tools {
		t := tool.(map[string]interface{})
		name := t["name"].(string)
		toolNames[name] = true
		if name == "list_tasks" {
			listTasksTool = t
		}
	}

	expectedTools := []string{"spawn_agent", "get_task", "list_tasks", "wait_task", "cancel_task", "pause_task", "resume_task", "get_stats"}
	for _, name := range expectedTools {
		if !toolNames[name] {
			t.Errorf("Expected tool '%s' not found", name)
		}
	}

	// Regression check: VS Code validates tool schemas strictly.
	// Ensure list_tasks.status.items.enum is a JSON array (not a comma-separated string).
	if listTasksTool == nil {
		t.Fatal("Expected to find tool 'list_tasks'")
	}

	inputSchema, ok := listTasksTool["inputSchema"].(map[string]interface{})
	if !ok {
		t.Fatalf("Expected list_tasks.inputSchema to be an object, got %T", listTasksTool["inputSchema"])
	}
	properties, ok := inputSchema["properties"].(map[string]interface{})
	if !ok {
		t.Fatalf("Expected list_tasks.inputSchema.properties to be an object, got %T", inputSchema["properties"])
	}
	statusProp, ok := properties["status"].(map[string]interface{})
	if !ok {
		t.Fatalf("Expected list_tasks.inputSchema.properties.status to be an object, got %T", properties["status"])
	}
	items, ok := statusProp["items"].(map[string]interface{})
	if !ok {
		t.Fatalf("Expected list_tasks.status.items to be an object, got %T", statusProp["items"])
	}
	if items["type"] != "string" {
		t.Fatalf("Expected list_tasks.status.items.type to be 'string', got %v", items["type"])
	}

	enumVal, ok := items["enum"].([]interface{})
	if !ok {
		t.Fatalf("Expected list_tasks.status.items.enum to be an array, got %T (%v)", items["enum"], items["enum"])
	}

	expectedStatuses := map[string]bool{"pending": true, "running": true, "paused": true, "completed": true, "failed": true, "cancelled": true}
	if len(enumVal) != len(expectedStatuses) {
		t.Fatalf("Expected %d enum values, got %d", len(expectedStatuses), len(enumVal))
	}
	for _, v := range enumVal {
		status, ok := v.(string)
		if !ok {
			t.Fatalf("Expected enum values to be strings, got %T", v)
		}
		if !expectedStatuses[status] {
			t.Fatalf("Unexpected status enum value: %q", status)
		}
	}
}

func TestMCPToolsCall(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	// Call get_stats tool
	reqBody := JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "tools/call",
		Params:  json.RawMessage(`{"name": "get_stats", "arguments": {}}`),
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", "/mcp", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	srv.httpServer.Handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response JSONRPCResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if response.Error != nil {
		t.Errorf("Expected no error, got: %+v", response.Error)
	}

	result := response.Result.(map[string]interface{})
	content := result["content"].([]interface{})
	if len(content) == 0 {
		t.Error("Expected content in response")
	}
}

func TestMCPUnknownMethod(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	reqBody := JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "unknown/method",
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", "/mcp", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	srv.httpServer.Handler.ServeHTTP(w, req)

	var response JSONRPCResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if response.Error == nil {
		t.Error("Expected error for unknown method")
	}

	if response.Error.Code != -32601 {
		t.Errorf("Expected error code -32601, got %d", response.Error.Code)
	}
}

func TestMCPPing(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	reqBody := JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "ping",
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", "/mcp", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	srv.httpServer.Handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response JSONRPCResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if response.Error != nil {
		t.Errorf("Expected no error, got: %+v", response.Error)
	}
}

func TestCORSHeaders(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	req := httptest.NewRequest("OPTIONS", "/mcp", nil)
	w := httptest.NewRecorder()

	srv.httpServer.Handler.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("Expected status 204, got %d", w.Code)
	}

	if w.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Error("Expected CORS header")
	}
}

func TestSpawnAgentTool(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	reqBody := JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "tools/call",
		Params:  json.RawMessage(`{"name": "spawn_agent", "arguments": {"prompt": "echo hello", "work_dir": "/tmp", "background": true}}`),
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", "/mcp", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	srv.httpServer.Handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response JSONRPCResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if response.Error != nil {
		t.Errorf("Expected no error, got: %+v", response.Error)
	}

	// The result should contain a task_id
	result := response.Result.(map[string]interface{})
	content := result["content"].([]interface{})
	if len(content) == 0 {
		t.Error("Expected content in response")
	}

	// Parse the text content to check for task_id
	textContent := content[0].(map[string]interface{})
	text := textContent["text"].(string)
	if text == "" {
		t.Error("Expected text content")
	}
}
