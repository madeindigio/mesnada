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
	for _, tool := range tools {
		t := tool.(map[string]interface{})
		toolNames[t["name"].(string)] = true
	}

	expectedTools := []string{"spawn_agent", "get_task", "list_tasks", "wait_task", "cancel_task", "get_stats"}
	for _, name := range expectedTools {
		if !toolNames[name] {
			t.Errorf("Expected tool '%s' not found", name)
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
