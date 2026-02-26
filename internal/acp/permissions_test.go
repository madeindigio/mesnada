package acp

import (
	"context"
	"testing"
	"time"

	acpsdk "github.com/coder/acp-go-sdk"
)

func TestPermissionQueue_QueueAndResolve(t *testing.T) {
	queue := NewPermissionQueue()

	// Create a mock permission request
	sessionID := acpsdk.SessionId("test-session")
	req := acpsdk.RequestPermissionRequest{
		SessionId: sessionID,
		Options: []acpsdk.PermissionOption{
			{
				OptionId: "option1",
				Name:     "Allow",
				Kind:     "allow",
			},
		},
		ToolCall: acpsdk.RequestPermissionToolCall{
			ToolCallId: "tool1",
		},
	}

	// Queue the permission
	requestID := queue.QueuePermission("task-123", sessionID, req)
	if requestID == "" {
		t.Fatal("Expected non-empty request ID")
	}

	// Check it's pending
	pending := queue.GetPending("task-123")
	if len(pending) != 1 {
		t.Fatalf("Expected 1 pending permission, got %d", len(pending))
	}

	if pending[0].RequestID != requestID {
		t.Errorf("Expected request ID %s, got %s", requestID, pending[0].RequestID)
	}

	// Resolve it
	outcome := acpsdk.NewRequestPermissionOutcomeSelected("option1")
	err := queue.ResolvePermission(requestID, outcome)
	if err != nil {
		t.Fatalf("Failed to resolve permission: %v", err)
	}

	// Check it's no longer pending
	pending = queue.GetPending("task-123")
	if len(pending) != 0 {
		t.Errorf("Expected 0 pending permissions, got %d", len(pending))
	}

	// Check we can get the resolved permission
	perm, exists := queue.GetPermission(requestID)
	if !exists {
		t.Fatal("Expected permission to exist")
	}
	if perm.Outcome == nil {
		t.Fatal("Expected permission to have outcome")
	}
	if perm.Outcome.Selected == nil {
		t.Error("Expected selected outcome")
	}
}

func TestPermissionQueue_WaitForResolution(t *testing.T) {
	queue := NewPermissionQueue()

	sessionID := acpsdk.SessionId("test-session")
	req := acpsdk.RequestPermissionRequest{
		SessionId: sessionID,
		Options: []acpsdk.PermissionOption{
			{
				OptionId: "option1",
				Name:     "Allow",
				Kind:     "allow",
			},
		},
		ToolCall: acpsdk.RequestPermissionToolCall{
			ToolCallId: "tool1",
		},
	}

	requestID := queue.QueuePermission("task-123", sessionID, req)

	// Resolve in background after a delay
	go func() {
		time.Sleep(100 * time.Millisecond)
		outcome := acpsdk.NewRequestPermissionOutcomeSelected("option1")
		queue.ResolvePermission(requestID, outcome)
	}()

	// Wait for resolution
	ctx := context.Background()
	outcome, err := queue.WaitForResolution(ctx, requestID)
	if err != nil {
		t.Fatalf("WaitForResolution failed: %v", err)
	}

	if outcome.Selected == nil {
		t.Error("Expected selected outcome")
	}
}

func TestPermissionQueue_WaitForResolutionTimeout(t *testing.T) {
	queue := NewPermissionQueue()

	sessionID := acpsdk.SessionId("test-session")
	req := acpsdk.RequestPermissionRequest{
		SessionId: sessionID,
		Options: []acpsdk.PermissionOption{
			{
				OptionId: "option1",
				Name:     "Allow",
				Kind:     "allow",
			},
		},
		ToolCall: acpsdk.RequestPermissionToolCall{
			ToolCallId: "tool1",
		},
	}

	requestID := queue.QueuePermission("task-123", sessionID, req)

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// Wait for resolution (should timeout)
	_, err := queue.WaitForResolution(ctx, requestID)
	if err == nil {
		t.Error("Expected timeout error")
	}
	if err != context.DeadlineExceeded {
		t.Errorf("Expected DeadlineExceeded, got %v", err)
	}
}

func TestPermissionQueue_ResolveNonExistent(t *testing.T) {
	queue := NewPermissionQueue()

	outcome := acpsdk.NewRequestPermissionOutcomeSelected("option1")
	err := queue.ResolvePermission("non-existent", outcome)
	if err == nil {
		t.Error("Expected error when resolving non-existent permission")
	}
}

func TestPermissionQueue_ResolveAlreadyResolved(t *testing.T) {
	queue := NewPermissionQueue()

	sessionID := acpsdk.SessionId("test-session")
	req := acpsdk.RequestPermissionRequest{
		SessionId: sessionID,
		Options: []acpsdk.PermissionOption{
			{
				OptionId: "option1",
				Name:     "Allow",
				Kind:     "allow",
			},
		},
		ToolCall: acpsdk.RequestPermissionToolCall{
			ToolCallId: "tool1",
		},
	}

	requestID := queue.QueuePermission("task-123", sessionID, req)

	// Resolve once
	outcome := acpsdk.NewRequestPermissionOutcomeSelected("option1")
	err := queue.ResolvePermission(requestID, outcome)
	if err != nil {
		t.Fatalf("Failed to resolve permission: %v", err)
	}

	// Try to resolve again
	err = queue.ResolvePermission(requestID, outcome)
	if err == nil {
		t.Error("Expected error when resolving already resolved permission")
	}
}

func TestPermissionQueue_CleanupResolved(t *testing.T) {
	queue := NewPermissionQueue()

	sessionID := acpsdk.SessionId("test-session")
	req := acpsdk.RequestPermissionRequest{
		SessionId: sessionID,
		Options: []acpsdk.PermissionOption{
			{
				OptionId: "option1",
				Name:     "Allow",
				Kind:     "allow",
			},
		},
		ToolCall: acpsdk.RequestPermissionToolCall{
			ToolCallId: "tool1",
		},
	}

	// Queue and resolve 3 permissions
	for i := 0; i < 3; i++ {
		requestID := queue.QueuePermission("task-123", sessionID, req)
		outcome := acpsdk.NewRequestPermissionOutcomeSelected("option1")
		queue.ResolvePermission(requestID, outcome)
	}

	// Queue one more but don't resolve
	queue.QueuePermission("task-123", sessionID, req)

	// Cleanup with 0 age (should remove all resolved)
	removed := queue.CleanupResolved(0)
	if removed != 3 {
		t.Errorf("Expected to remove 3 resolved permissions, removed %d", removed)
	}

	// Should have 1 pending left
	pending := queue.GetPending("task-123")
	if len(pending) != 1 {
		t.Errorf("Expected 1 pending permission, got %d", len(pending))
	}
}
