// Package main implements a minimal ACP-compatible agent using the ACP Go SDK.
// This example demonstrates how to create a custom agent that works with mesnada.
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	acpsdk "github.com/coder/acp-go-sdk"
)

func main() {
	// Create our custom agent
	agent := NewCustomAgent()

	// Create an agent-side connection
	// stdin = where we read from the client (mesnada)
	// stdout = where we write to the client
	conn := acpsdk.NewAgentSideConnection(agent, os.Stdin, os.Stdout)

	// Listen for requests from the client
	log.Println("[CustomAgent] Starting...")
	if err := conn.Listen(context.Background()); err != nil {
		log.Fatalf("[CustomAgent] Error: %v", err)
	}
}

// CustomAgent implements the acpsdk.Agent interface.
type CustomAgent struct {
	sessions map[acpsdk.SessionId]*SessionState
}

// SessionState tracks the state of an active session.
type SessionState struct {
	id           acpsdk.SessionId
	cwd          string
	mcpServers   []acpsdk.McpServer
	messageCount int
}

// NewCustomAgent creates a new custom ACP agent.
func NewCustomAgent() *CustomAgent {
	return &CustomAgent{
		sessions: make(map[acpsdk.SessionId]*SessionState),
	}
}

// Initialize handles the initialize request - capability negotiation.
func (a *CustomAgent) Initialize(ctx context.Context, req acpsdk.InitializeRequest) (acpsdk.InitializeResponse, error) {
	log.Printf("[CustomAgent] Initialize called (protocol version %d)", req.ProtocolVersion)

	return acpsdk.InitializeResponse{
		ProtocolVersion: acpsdk.ProtocolVersionNumber,
		ServerInfo: acpsdk.Implementation{
			Name:    "custom-acp-agent",
			Version: "1.0.0",
		},
		Capabilities: acpsdk.ServerCapabilities{
			// We support basic prompting
			Prompting: true,
		},
	}, nil
}

// NewSession handles the session/new request - creates a new session.
func (a *CustomAgent) NewSession(ctx context.Context, req acpsdk.NewSessionRequest) (acpsdk.NewSessionResponse, error) {
	sessionID := acpsdk.SessionId(fmt.Sprintf("custom-session-%d", len(a.sessions)+1))

	log.Printf("[CustomAgent] NewSession called: session_id=%s, cwd=%s, mcp_servers=%d",
		sessionID, req.Cwd, len(req.McpServers))

	// Create session state
	state := &SessionState{
		id:         sessionID,
		cwd:        req.Cwd,
		mcpServers: req.McpServers,
	}

	a.sessions[sessionID] = state

	return acpsdk.NewSessionResponse{
		SessionId: sessionID,
	}, nil
}

// Prompt handles the session/prompt request - processes a user prompt.
func (a *CustomAgent) Prompt(ctx context.Context, req acpsdk.PromptRequest) (acpsdk.PromptResponse, error) {
	session, exists := a.sessions[req.SessionId]
	if !exists {
		return acpsdk.PromptResponse{}, fmt.Errorf("session not found: %s", req.SessionId)
	}

	log.Printf("[CustomAgent] Prompt called: session_id=%s", req.SessionId)

	// Extract text from prompt
	promptText := ""
	for _, block := range req.Prompt {
		if block.Text != nil {
			promptText += block.Text.Text
		}
	}

	log.Printf("[CustomAgent] Prompt text: %s", promptText)

	// Send a message update to the client
	if req.SessionUpdates != nil {
		// Send initial message
		req.SessionUpdates(acpsdk.SessionUpdate{
			AgentMessageChunk: &acpsdk.SessionUpdateAgentMessageChunk{
				Content: acpsdk.TextBlock(fmt.Sprintf("Custom agent received your prompt!\n\nYou said: %s\n\n", promptText)),
			},
		})

		// Simulate some work
		req.SessionUpdates(acpsdk.SessionUpdate{
			AgentMessageChunk: &acpsdk.SessionUpdateAgentMessageChunk{
				Content: acpsdk.TextBlock(fmt.Sprintf("Processing in directory: %s\n\n", session.cwd)),
			},
		})

		// Count MCP servers
		req.SessionUpdates(acpsdk.SessionUpdate{
			AgentMessageChunk: &acpsdk.SessionUpdateAgentMessageChunk{
				Content: acpsdk.TextBlock(fmt.Sprintf("Available MCP servers: %d\n\n", len(session.mcpServers))),
			},
		})

		// Send completion message
		session.messageCount++
		req.SessionUpdates(acpsdk.SessionUpdate{
			AgentMessageChunk: &acpsdk.SessionUpdateAgentMessageChunk{
				Content: acpsdk.TextBlock(fmt.Sprintf("✓ Task completed! (message #%d)\n", session.messageCount)),
			},
		})
	}

	return acpsdk.PromptResponse{
		StopReason: acpsdk.StopReasonEndTurn,
	}, nil
}

// Cancel handles the session/cancel notification - cancels a session.
func (a *CustomAgent) Cancel(ctx context.Context, notification acpsdk.CancelNotification) error {
	log.Printf("[CustomAgent] Cancel called: session_id=%s", notification.SessionId)

	// Clean up session
	delete(a.sessions, notification.SessionId)

	return nil
}
