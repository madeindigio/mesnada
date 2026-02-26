package acp

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"testing"

	acpsdk "github.com/coder/acp-go-sdk"
)

// MockACPAgent implements a simple ACP agent for testing purposes.
// It responds to JSON-RPC requests and simulates agent behavior.
type MockACPAgent struct {
	stdin  io.Reader
	stdout io.Writer
	t      *testing.T

	// Configuration
	agentName      string
	agentVersion   string
	simulateError  bool
	simulateCancel bool
	toolCallCount  int

	mu sync.Mutex
}

// NewMockACPAgent creates a new mock ACP agent for testing.
func NewMockACPAgent(t *testing.T, stdin io.Reader, stdout io.Writer) *MockACPAgent {
	return &MockACPAgent{
		stdin:        stdin,
		stdout:       stdout,
		t:            t,
		agentName:    "mock-acp-agent",
		agentVersion: "1.0.0-test",
	}
}

// SetSimulateError configures the mock to simulate an error during execution.
func (m *MockACPAgent) SetSimulateError(simulate bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.simulateError = simulate
}

// SetSimulateCancel configures the mock to simulate cancellation.
func (m *MockACPAgent) SetSimulateCancel(simulate bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.simulateCancel = simulate
}

// Run starts the mock agent and processes JSON-RPC requests.
func (m *MockACPAgent) Run(ctx context.Context) error {
	scanner := bufio.NewScanner(m.stdin)
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)

	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		// Parse JSON-RPC request
		var req struct {
			JSONRPC string          `json:"jsonrpc"`
			ID      json.RawMessage `json:"id"`
			Method  string          `json:"method"`
			Params  json.RawMessage `json:"params"`
		}

		if err := json.Unmarshal(line, &req); err != nil {
			m.t.Logf("[MockAgent] Failed to parse request: %v", err)
			continue
		}

		m.t.Logf("[MockAgent] Received: %s", req.Method)

		// Handle request based on method
		var response interface{}
		var err error

		switch req.Method {
		case "initialize":
			response, err = m.handleInitialize(req.Params)
		case "session/new":
			response, err = m.handleSessionNew(req.Params)
		case "session/prompt":
			response, err = m.handleSessionPrompt(ctx, req.Params)
		case "session/cancel":
			response, err = m.handleSessionCancel(req.Params)
		default:
			err = fmt.Errorf("unknown method: %s", req.Method)
		}

		// Send response
		if err != nil {
			m.sendError(req.ID, -32603, err.Error())
		} else {
			m.sendResponse(req.ID, response)
		}

		// Check context cancellation
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
	}

	return scanner.Err()
}

// handleInitialize handles the initialize request.
func (m *MockACPAgent) handleInitialize(params json.RawMessage) (interface{}, error) {
	return map[string]interface{}{
		"protocolVersion": acpsdk.ProtocolVersionNumber,
		"serverInfo": map[string]string{
			"name":    m.agentName,
			"version": m.agentVersion,
		},
		"capabilities": map[string]interface{}{
			"fs": map[string]bool{
				"readTextFile":  true,
				"writeTextFile": true,
			},
			"terminal": true,
		},
	}, nil
}

// handleSessionNew handles the session/new request.
func (m *MockACPAgent) handleSessionNew(params json.RawMessage) (interface{}, error) {
	return map[string]interface{}{
		"sessionId": "test-session-mock-123",
	}, nil
}

// handleSessionPrompt handles the session/prompt request.
func (m *MockACPAgent) handleSessionPrompt(ctx context.Context, params json.RawMessage) (interface{}, error) {
	var req struct {
		SessionID string `json:"sessionId"`
		Prompt    []struct {
			Type string `json:"type"`
			Text string `json:"text,omitempty"`
		} `json:"prompt"`
	}

	if err := json.Unmarshal(params, &req); err != nil {
		return nil, fmt.Errorf("failed to parse prompt params: %w", err)
	}

	m.mu.Lock()
	simulateError := m.simulateError
	simulateCancel := m.simulateCancel
	m.mu.Unlock()

	// Send session updates to simulate agent behavior
	if err := m.sendSessionUpdate(req.SessionID, "agentMessageChunk", map[string]interface{}{
		"content": map[string]interface{}{
			"type": "text",
			"text": "Mock agent starting execution...\n",
		},
	}); err != nil {
		return nil, err
	}

	// Simulate some tool calls
	for i := 0; i < 3; i++ {
		m.mu.Lock()
		m.toolCallCount++
		m.mu.Unlock()

		if err := m.sendSessionUpdate(req.SessionID, "toolCall", map[string]interface{}{
			"toolCallId": fmt.Sprintf("tool-call-%d", i+1),
			"title":      fmt.Sprintf("Tool %d", i+1),
			"status":     "started",
			"rawInput": map[string]interface{}{
				"arg1": "value1",
			},
		}); err != nil {
			return nil, err
		}

		// Simulate tool completion
		if err := m.sendSessionUpdate(req.SessionID, "toolCallUpdate", map[string]interface{}{
			"toolCallId": fmt.Sprintf("tool-call-%d", i+1),
			"status":     "completed",
			"rawOutput": map[string]interface{}{
				"result": "success",
			},
		}); err != nil {
			return nil, err
		}
	}

	// Check for cancellation
	if simulateCancel {
		return map[string]interface{}{
			"stopReason": "cancelled",
		}, nil
	}

	// Check for error simulation
	if simulateError {
		return nil, fmt.Errorf("simulated agent error")
	}

	// Send final message
	if err := m.sendSessionUpdate(req.SessionID, "agentMessageChunk", map[string]interface{}{
		"content": map[string]interface{}{
			"type": "text",
			"text": "Mock agent completed execution successfully!\n",
		},
	}); err != nil {
		return nil, err
	}

	// Return success
	return map[string]interface{}{
		"stopReason": "end_turn",
	}, nil
}

// handleSessionCancel handles the session/cancel notification.
func (m *MockACPAgent) handleSessionCancel(params json.RawMessage) (interface{}, error) {
	m.t.Logf("[MockAgent] Received cancel notification")
	m.mu.Lock()
	m.simulateCancel = true
	m.mu.Unlock()
	return nil, nil
}

// sendSessionUpdate sends a session update notification to the client.
func (m *MockACPAgent) sendSessionUpdate(sessionID string, updateType string, update interface{}) error {
	notification := map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  "session/update",
		"params": map[string]interface{}{
			"sessionId": sessionID,
			"update": map[string]interface{}{
				updateType: update,
			},
		},
	}

	data, err := json.Marshal(notification)
	if err != nil {
		return fmt.Errorf("failed to marshal notification: %w", err)
	}

	data = append(data, '\n')
	_, err = m.stdout.Write(data)
	return err
}

// sendResponse sends a JSON-RPC response.
func (m *MockACPAgent) sendResponse(id json.RawMessage, result interface{}) {
	response := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      id,
		"result":  result,
	}

	data, err := json.Marshal(response)
	if err != nil {
		m.t.Logf("[MockAgent] Failed to marshal response: %v", err)
		return
	}

	data = append(data, '\n')
	if _, err := m.stdout.Write(data); err != nil {
		m.t.Logf("[MockAgent] Failed to write response: %v", err)
	}
}

// sendError sends a JSON-RPC error response.
func (m *MockACPAgent) sendError(id json.RawMessage, code int, message string) {
	response := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      id,
		"error": map[string]interface{}{
			"code":    code,
			"message": message,
		},
	}

	data, err := json.Marshal(response)
	if err != nil {
		m.t.Logf("[MockAgent] Failed to marshal error: %v", err)
		return
	}

	data = append(data, '\n')
	if _, err := m.stdout.Write(data); err != nil {
		m.t.Logf("[MockAgent] Failed to write error: %v", err)
	}
}

// MockAgentBinary is a helper function that creates a test binary that acts as a mock ACP agent.
// This is useful for integration tests that need to spawn actual processes.
func MockAgentBinary(t *testing.T) string {
	// Check if we're running as the mock agent subprocess
	if os.Getenv("MOCK_ACP_AGENT") == "1" {
		// Run as mock agent
		agent := NewMockACPAgent(t, os.Stdin, os.Stdout)

		// Check for error simulation
		if os.Getenv("MOCK_ACP_SIMULATE_ERROR") == "1" {
			agent.SetSimulateError(true)
		}

		// Check for cancel simulation
		if os.Getenv("MOCK_ACP_SIMULATE_CANCEL") == "1" {
			agent.SetSimulateCancel(true)
		}

		ctx := context.Background()
		if err := agent.Run(ctx); err != nil {
			fmt.Fprintf(os.Stderr, "Mock agent error: %v\n", err)
			os.Exit(1)
		}
		os.Exit(0)
		return "" // Never reached
	}

	// Return the path to the current test binary
	// When spawned with MOCK_ACP_AGENT=1, it will act as a mock agent
	return os.Args[0]
}

// TestMockACPAgentBasic verifies that the mock agent responds to basic requests.
func TestMockACPAgentBasic(t *testing.T) {
	// Create pipes for communication
	clientR, agentW := io.Pipe()
	agentR, clientW := io.Pipe()

	agent := NewMockACPAgent(t, agentR, agentW)

	// Start agent in background
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		if err := agent.Run(ctx); err != nil && err != io.EOF && !strings.Contains(err.Error(), "closed pipe") {
			t.Logf("Agent error: %v", err)
		}
	}()

	// Send initialize request
	initReq := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "initialize",
		"params": map[string]interface{}{
			"protocolVersion": acpsdk.ProtocolVersionNumber,
			"clientInfo": map[string]string{
				"name":    "test-client",
				"version": "1.0",
			},
		},
	}

	data, _ := json.Marshal(initReq)
	data = append(data, '\n')
	clientW.Write(data)

	// Read response
	scanner := bufio.NewScanner(clientR)
	if !scanner.Scan() {
		t.Fatal("Failed to read initialize response")
	}

	var initResp struct {
		JSONRPC string `json:"jsonrpc"`
		ID      int    `json:"id"`
		Result  struct {
			ProtocolVersion int `json:"protocolVersion"`
			ServerInfo      struct {
				Name    string `json:"name"`
				Version string `json:"version"`
			} `json:"serverInfo"`
		} `json:"result"`
	}

	if err := json.Unmarshal(scanner.Bytes(), &initResp); err != nil {
		t.Fatalf("Failed to parse initialize response: %v", err)
	}

	if initResp.Result.ServerInfo.Name != "mock-acp-agent" {
		t.Errorf("Expected agent name 'mock-acp-agent', got %s", initResp.Result.ServerInfo.Name)
	}

	// Cleanup
	clientW.Close()
	clientR.Close()
}
