package acp

import (
	"context"
	"testing"

	acpsdk "github.com/coder/acp-go-sdk"
)

func TestNewMesnadaACPClient(t *testing.T) {
	client := NewMesnadaACPClient("task-123", "/tmp/workdir", nil, nil)

	if client == nil {
		t.Fatal("Expected client to be created")
	}

	if client.taskID != "task-123" {
		t.Errorf("Expected taskID to be 'task-123', got '%s'", client.taskID)
	}

	if client.workDir != "/tmp/workdir" {
		t.Errorf("Expected workDir to be '/tmp/workdir', got '%s'", client.workDir)
	}

	if client.output == nil {
		t.Error("Expected output builder to be initialized")
	}

	if client.terminals == nil {
		t.Error("Expected terminals map to be initialized")
	}
}

func TestMesnadaACPClient_SessionUpdate(t *testing.T) {
	updateCalled := false
	var receivedUpdate SessionUpdateInfo

	onUpdate := func(update SessionUpdateInfo) {
		updateCalled = true
		receivedUpdate = update
	}

	client := NewMesnadaACPClient("task-456", "/tmp/workdir", nil, onUpdate)
	ctx := context.Background()

	// Create a notification with actual content to trigger callback
	title := "test tool"
	notification := acpsdk.SessionNotification{
		SessionId: acpsdk.SessionId("session-123"),
		Update: acpsdk.SessionUpdate{
			AgentMessageChunk: &acpsdk.SessionUpdateAgentMessageChunk{
				Content: acpsdk.TextBlock("test message"),
			},
		},
	}

	err := client.SessionUpdate(ctx, notification)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if !updateCalled {
		t.Error("Expected onUpdate callback to be called")
	}

	if receivedUpdate.TaskID != "task-456" {
		t.Errorf("Expected TaskID to be 'task-456', got '%s'", receivedUpdate.TaskID)
	}

	_ = title // Avoid unused variable warning
}

func TestMesnadaACPClient_RequestPermission(t *testing.T) {
	client := NewMesnadaACPClient("task-789", "/tmp/workdir", nil, nil)
	// Enable auto-permission for this test
	client.SetAutoPermission(true)
	ctx := context.Background()

	request := acpsdk.RequestPermissionRequest{
		SessionId: acpsdk.SessionId("session-456"),
		Options: []acpsdk.PermissionOption{
			{
				OptionId: acpsdk.PermissionOptionId("option-1"),
				Name:     "Allow",
			},
		},
	}

	response, err := client.RequestPermission(ctx, request)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if response.Outcome.Selected == nil {
		t.Error("Expected Selected outcome to be set")
	}
}

func TestMesnadaACPClient_CreateTerminal(t *testing.T) {
	client := NewMesnadaACPClient("task-abc", "/tmp/workdir", nil, nil)
	ctx := context.Background()

	cwd := "subdir" // Use relative path within workspace
	request := acpsdk.CreateTerminalRequest{
		SessionId: acpsdk.SessionId("session-789"),
		Command:   "ls",
		Args:      []string{"-la"},
		Cwd:       &cwd,
	}

	response, err := client.CreateTerminal(ctx, request)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if response.TerminalId == "" {
		t.Error("Expected TerminalId to be set")
	}

	// Verify terminal was added to the map
	client.terminalsMu.RLock()
	defer client.terminalsMu.RUnlock()

	if len(client.terminals) != 1 {
		t.Errorf("Expected 1 terminal, got %d", len(client.terminals))
	}

	term, ok := client.terminals[response.TerminalId]
	if !ok {
		t.Error("Expected terminal to be in map")
	}

	if term.command != "ls" {
		t.Errorf("Expected command to be 'ls', got '%s'", term.command)
	}
}

func TestMesnadaACPClient_ReleaseTerminal(t *testing.T) {
	client := NewMesnadaACPClient("task-def", "/tmp/workdir", nil, nil)
	ctx := context.Background()

	// Create a terminal first
	createReq := acpsdk.CreateTerminalRequest{
		SessionId: acpsdk.SessionId("session-abc"),
		Command:   "echo",
		Args:      []string{"hello"},
	}

	createResp, err := client.CreateTerminal(ctx, createReq)
	if err != nil {
		t.Fatalf("Failed to create terminal: %v", err)
	}

	// Release it
	releaseReq := acpsdk.ReleaseTerminalRequest{
		SessionId:  acpsdk.SessionId("session-abc"),
		TerminalId: createResp.TerminalId,
	}

	_, err = client.ReleaseTerminal(ctx, releaseReq)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	// Verify terminal was marked as released (but not removed from map)
	client.terminalsMu.RLock()
	defer client.terminalsMu.RUnlock()

	if len(client.terminals) != 1 {
		t.Errorf("Expected 1 terminal (marked released), got %d", len(client.terminals))
	}

	term, ok := client.terminals[createResp.TerminalId]
	if !ok {
		t.Error("Expected terminal to still be in map")
	}

	if !term.isReleased {
		t.Error("Expected terminal to be marked as released")
	}
}

func TestMesnadaACPClient_GetOutput(t *testing.T) {
	client := NewMesnadaACPClient("task-ghi", "/tmp/workdir", nil, nil)

	// Initially empty
	output := client.GetOutput()
	if output != "" {
		t.Errorf("Expected empty output, got '%s'", output)
	}

	// Write some data
	client.mu.Lock()
	client.output.WriteString("test output\n")
	client.mu.Unlock()

	output = client.GetOutput()
	if output != "test output\n" {
		t.Errorf("Expected 'test output\\n', got '%s'", output)
	}
}

// TestSessionUpdate_AgentMessageChunk tests processing of agent message chunks
func TestSessionUpdate_AgentMessageChunk(t *testing.T) {
	var updates []SessionUpdateInfo
	onUpdate := func(info SessionUpdateInfo) {
		updates = append(updates, info)
	}

	client := NewMesnadaACPClient("test-task", "/tmp", nil, onUpdate)

	err := client.SessionUpdate(context.Background(), acpsdk.SessionNotification{
		SessionId: "test-session",
		Update: acpsdk.SessionUpdate{
			AgentMessageChunk: &acpsdk.SessionUpdateAgentMessageChunk{
				Content: acpsdk.TextBlock("Hello, world!"),
			},
		},
	})

	if err != nil {
		t.Errorf("SessionUpdate failed: %v", err)
	}

	// Check that message was accumulated
	output := client.GetOutput()
	if output != "Hello, world!" {
		t.Errorf("Expected output 'Hello, world!', got: %s", output)
	}

	// Check callback was invoked
	if len(updates) != 1 {
		t.Fatalf("Expected 1 update, got %d", len(updates))
	}
	if updates[0].MessageText != "Hello, world!" {
		t.Errorf("Expected message text 'Hello, world!', got: %s", updates[0].MessageText)
	}
}

// TestSessionUpdate_AgentThoughtChunk tests that thinking is logged but not added to output
func TestSessionUpdate_AgentThoughtChunk(t *testing.T) {
	client := NewMesnadaACPClient("test-task", "/tmp", nil, nil)
	outputBefore := client.GetOutput()

	err := client.SessionUpdate(context.Background(), acpsdk.SessionNotification{
		SessionId: "test-session",
		Update: acpsdk.SessionUpdate{
			AgentThoughtChunk: &acpsdk.SessionUpdateAgentThoughtChunk{
				Content: acpsdk.TextBlock("Thinking..."),
			},
		},
	})

	if err != nil {
		t.Errorf("SessionUpdate failed: %v", err)
	}

	// Thinking should NOT be added to output
	outputAfter := client.GetOutput()
	if outputAfter != outputBefore {
		t.Errorf("Expected thinking not to be added to output")
	}
}

// TestSessionUpdate_ToolCall tests tool call tracking
func TestSessionUpdate_ToolCall(t *testing.T) {
	var updates []SessionUpdateInfo
	onUpdate := func(info SessionUpdateInfo) {
		updates = append(updates, info)
	}

	client := NewMesnadaACPClient("test-task", "/tmp", nil, onUpdate)
	toolCallsBefore := client.GetToolCallCount()

	err := client.SessionUpdate(context.Background(), acpsdk.SessionNotification{
		SessionId: "test-session",
		Update: acpsdk.SessionUpdate{
			ToolCall: &acpsdk.SessionUpdateToolCall{
				ToolCallId: "tool-1",
				Title:      "read_file",
				Status:     acpsdk.ToolCallStatusInProgress,
				RawInput:   map[string]interface{}{"path": "/tmp/test.txt"},
			},
		},
	})

	if err != nil {
		t.Errorf("SessionUpdate failed: %v", err)
	}

	// Check tool call was counted
	toolCallsAfter := client.GetToolCallCount()
	if toolCallsAfter != toolCallsBefore+1 {
		t.Errorf("Expected tool call count to increment, got %d -> %d", toolCallsBefore, toolCallsAfter)
	}

	// Check callback
	if len(updates) != 1 {
		t.Fatalf("Expected 1 update, got %d", len(updates))
	}
	if updates[0].ToolCall == nil {
		t.Fatal("Expected ToolCall in update")
	}
	if updates[0].ToolCall.Name != "read_file" {
		t.Errorf("Expected tool name 'read_file', got: %s", updates[0].ToolCall.Name)
	}
}

// TestSessionUpdate_Plan tests plan processing and progress calculation
func TestSessionUpdate_Plan(t *testing.T) {
	client := NewMesnadaACPClient("test-task", "/tmp", nil, nil)

	err := client.SessionUpdate(context.Background(), acpsdk.SessionNotification{
		SessionId: "test-session",
		Update: acpsdk.SessionUpdate{
			Plan: &acpsdk.SessionUpdatePlan{
				Entries: []acpsdk.PlanEntry{
					{Content: "Step 1", Status: acpsdk.PlanEntryStatusCompleted, Priority: acpsdk.PlanEntryPriorityMedium},
					{Content: "Step 2", Status: acpsdk.PlanEntryStatusInProgress, Priority: acpsdk.PlanEntryPriorityMedium},
					{Content: "Step 3", Status: acpsdk.PlanEntryStatusPending, Priority: acpsdk.PlanEntryPriorityMedium},
					{Content: "Step 4", Status: acpsdk.PlanEntryStatusPending, Priority: acpsdk.PlanEntryPriorityMedium},
				},
			},
		},
	})

	if err != nil {
		t.Errorf("SessionUpdate failed: %v", err)
	}

	// Check progress was calculated (1/4 = 25%)
	progress, description := client.GetProgress()
	if progress != 25 {
		t.Errorf("Expected progress 25%%, got %d%%", progress)
	}
	if description != "Plan: 1/4 steps completed" {
		t.Errorf("Expected description 'Plan: 1/4 steps completed', got: %s", description)
	}
}

// TestSessionUpdate_PlanProgressUpdate tests updating plan progress
func TestSessionUpdate_PlanProgressUpdate(t *testing.T) {
	client := NewMesnadaACPClient("test-task", "/tmp", nil, nil)

	// Initial plan with 1/4 completed
	client.SessionUpdate(context.Background(), acpsdk.SessionNotification{
		SessionId: "test-session",
		Update: acpsdk.SessionUpdate{
			Plan: &acpsdk.SessionUpdatePlan{
				Entries: []acpsdk.PlanEntry{
					{Content: "Step 1", Status: acpsdk.PlanEntryStatusCompleted, Priority: acpsdk.PlanEntryPriorityMedium},
					{Content: "Step 2", Status: acpsdk.PlanEntryStatusInProgress, Priority: acpsdk.PlanEntryPriorityMedium},
					{Content: "Step 3", Status: acpsdk.PlanEntryStatusPending, Priority: acpsdk.PlanEntryPriorityMedium},
					{Content: "Step 4", Status: acpsdk.PlanEntryStatusPending, Priority: acpsdk.PlanEntryPriorityMedium},
				},
			},
		},
	})

	// Update to 3/4 completed
	err := client.SessionUpdate(context.Background(), acpsdk.SessionNotification{
		SessionId: "test-session",
		Update: acpsdk.SessionUpdate{
			Plan: &acpsdk.SessionUpdatePlan{
				Entries: []acpsdk.PlanEntry{
					{Content: "Step 1", Status: acpsdk.PlanEntryStatusCompleted, Priority: acpsdk.PlanEntryPriorityMedium},
					{Content: "Step 2", Status: acpsdk.PlanEntryStatusCompleted, Priority: acpsdk.PlanEntryPriorityMedium},
					{Content: "Step 3", Status: acpsdk.PlanEntryStatusCompleted, Priority: acpsdk.PlanEntryPriorityMedium},
					{Content: "Step 4", Status: acpsdk.PlanEntryStatusInProgress, Priority: acpsdk.PlanEntryPriorityMedium},
				},
			},
		},
	})

	if err != nil {
		t.Errorf("SessionUpdate failed: %v", err)
	}

	// Check progress updated to 75%
	progress, _ := client.GetProgress()
	if progress != 75 {
		t.Errorf("Expected progress 75%%, got %d%%", progress)
	}
}

// TestSessionUpdate_ToolCallUpdate tests tool call status updates
func TestSessionUpdate_ToolCallUpdate(t *testing.T) {
	var updates []SessionUpdateInfo
	onUpdate := func(info SessionUpdateInfo) {
		updates = append(updates, info)
	}

	client := NewMesnadaACPClient("test-task", "/tmp", nil, onUpdate)

	// Tool call update with completion
	completedStatus := acpsdk.ToolCallStatusCompleted
	err := client.SessionUpdate(context.Background(), acpsdk.SessionNotification{
		SessionId: "test-session",
		Update: acpsdk.SessionUpdate{
			ToolCallUpdate: &acpsdk.SessionToolCallUpdate{
				ToolCallId: "tool-1",
				Status:     &completedStatus,
				RawOutput:  "Success!",
			},
		},
	})

	if err != nil {
		t.Errorf("SessionUpdate failed: %v", err)
	}

	// Check callback was invoked for completed tool call
	if len(updates) != 1 {
		t.Fatalf("Expected 1 update, got %d", len(updates))
	}
	if updates[0].ToolCall == nil {
		t.Fatal("Expected ToolCall in update")
	}
	if updates[0].ToolCall.Status != string(acpsdk.ToolCallStatusCompleted) {
		t.Errorf("Expected status 'completed', got: %s", updates[0].ToolCall.Status)
	}
	if updates[0].ToolCall.Result != "Success!" {
		t.Errorf("Expected result 'Success!', got: %s", updates[0].ToolCall.Result)
	}
}
