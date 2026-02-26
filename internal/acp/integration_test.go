package acp

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	acpsdk "github.com/coder/acp-go-sdk"
)

// TestACPClientFileOperations tests file read/write operations with security constraints.
func TestACPClientFileOperations(t *testing.T) {
	// Create temp workspace
	tempDir := t.TempDir()
	logFile, err := os.CreateTemp("", "acp-test-*.log")
	if err != nil {
		t.Fatalf("Failed to create log file: %v", err)
	}
	defer logFile.Close()

	// Create client
	client := NewMesnadaACPClient("test-task", tempDir, logFile, nil)

	t.Run("ReadTextFile_ValidPath", func(t *testing.T) {
		// Write test file
		testFile := filepath.Join(tempDir, "test.txt")
		testContent := "Hello, ACP!"
		if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
			t.Fatalf("Failed to write test file: %v", err)
		}

		// Read file
		ctx := context.Background()
		resp, err := client.ReadTextFile(ctx, acpsdk.ReadTextFileRequest{
			Path: "test.txt",
		})

		if err != nil {
			t.Errorf("ReadTextFile failed: %v", err)
		}

		if resp.Content != testContent {
			t.Errorf("Expected content %q, got %q", testContent, resp.Content)
		}
	})

	t.Run("ReadTextFile_PathTraversal", func(t *testing.T) {
		// Attempt path traversal
		ctx := context.Background()
		_, err := client.ReadTextFile(ctx, acpsdk.ReadTextFileRequest{
			Path: "../../../etc/passwd",
		})

		if err == nil {
			t.Error("Expected error for path traversal, got nil")
		}
	})

	t.Run("WriteTextFile_ValidPath", func(t *testing.T) {
		// Write file
		ctx := context.Background()
		testContent := "New file content"
		_, err := client.WriteTextFile(ctx, acpsdk.WriteTextFileRequest{
			Path:    "newfile.txt",
			Content: testContent,
		})

		if err != nil {
			t.Errorf("WriteTextFile failed: %v", err)
		}

		// Verify file was written
		writtenContent, err := os.ReadFile(filepath.Join(tempDir, "newfile.txt"))
		if err != nil {
			t.Errorf("Failed to read written file: %v", err)
		}

		if string(writtenContent) != testContent {
			t.Errorf("Expected content %q, got %q", testContent, string(writtenContent))
		}
	})

	t.Run("WriteTextFile_PathTraversal", func(t *testing.T) {
		// Attempt path traversal
		ctx := context.Background()
		_, err := client.WriteTextFile(ctx, acpsdk.WriteTextFileRequest{
			Path:    "../../../tmp/evil.txt",
			Content: "malicious",
		})

		if err == nil {
			t.Error("Expected error for path traversal, got nil")
		}
	})
}

// TestACPClientTerminalOperations tests terminal creation and management.
func TestACPClientTerminalOperations(t *testing.T) {
	tempDir := t.TempDir()
	logFile, err := os.CreateTemp("", "acp-test-*.log")
	if err != nil {
		t.Fatalf("Failed to create log file: %v", err)
	}
	defer logFile.Close()

	client := NewMesnadaACPClient("test-task", tempDir, logFile, nil)

	t.Run("CreateTerminal_Success", func(t *testing.T) {
		ctx := context.Background()
		resp, err := client.CreateTerminal(ctx, acpsdk.CreateTerminalRequest{
			Command: "echo",
			Args:    []string{"hello"},
		})

		if err != nil {
			t.Fatalf("CreateTerminal failed: %v", err)
		}

		if resp.TerminalId == "" {
			t.Error("Expected terminal ID, got empty string")
		}

		// Wait for terminal to complete
		time.Sleep(100 * time.Millisecond)

		// Get output
		outputResp, err := client.TerminalOutput(ctx, acpsdk.TerminalOutputRequest{
			TerminalId: resp.TerminalId,
		})

		if err != nil {
			t.Errorf("TerminalOutput failed: %v", err)
		}

		if outputResp.Output != "hello\n" {
			t.Errorf("Expected output 'hello\\n', got %q", outputResp.Output)
		}
	})

	t.Run("CreateTerminal_WaitForExit", func(t *testing.T) {
		ctx := context.Background()
		resp, err := client.CreateTerminal(ctx, acpsdk.CreateTerminalRequest{
			Command: "sleep",
			Args:    []string{"0.1"},
		})

		if err != nil {
			t.Fatalf("CreateTerminal failed: %v", err)
		}

		// Wait for exit
		exitResp, err := client.WaitForTerminalExit(ctx, acpsdk.WaitForTerminalExitRequest{
			TerminalId: resp.TerminalId,
		})

		if err != nil {
			t.Errorf("WaitForTerminalExit failed: %v", err)
		}

		if exitResp.ExitCode == nil {
			t.Error("Expected exit code, got nil")
		} else if *exitResp.ExitCode != 0 {
			t.Errorf("Expected exit code 0, got %d", *exitResp.ExitCode)
		}
	})

	t.Run("CreateTerminal_Kill", func(t *testing.T) {
		ctx := context.Background()
		resp, err := client.CreateTerminal(ctx, acpsdk.CreateTerminalRequest{
			Command: "sleep",
			Args:    []string{"10"},
		})

		if err != nil {
			t.Fatalf("CreateTerminal failed: %v", err)
		}

		// Kill terminal
		_, err = client.KillTerminalCommand(ctx, acpsdk.KillTerminalCommandRequest{
			TerminalId: resp.TerminalId,
		})

		if err != nil {
			t.Errorf("KillTerminalCommand failed: %v", err)
		}
	})
}

// TestACPClientSessionUpdates tests session update handling.
func TestACPClientSessionUpdates(t *testing.T) {
	tempDir := t.TempDir()
	logFile, err := os.CreateTemp("", "acp-test-*.log")
	if err != nil {
		t.Fatalf("Failed to create log file: %v", err)
	}
	defer logFile.Close()

	var updates []SessionUpdateInfo
	onUpdate := func(update SessionUpdateInfo) {
		updates = append(updates, update)
	}

	client := NewMesnadaACPClient("test-task", tempDir, logFile, onUpdate)

	t.Run("SessionUpdate_AgentMessage", func(t *testing.T) {
		ctx := context.Background()
		updates = nil

		// Simulate agent message
		err := client.SessionUpdate(ctx, acpsdk.SessionNotification{
			SessionId: "test-session",
			Update: acpsdk.SessionUpdate{
				AgentMessageChunk: &acpsdk.SessionUpdateAgentMessageChunk{
					Content: acpsdk.TextBlock("Test message"),
				},
			},
		})

		if err != nil {
			t.Errorf("SessionUpdate failed: %v", err)
		}

		if len(updates) != 1 {
			t.Errorf("Expected 1 update, got %d", len(updates))
		}

		if len(updates) > 0 && updates[0].MessageText != "Test message" {
			t.Errorf("Expected message 'Test message', got %q", updates[0].MessageText)
		}
	})

	t.Run("SessionUpdate_ToolCall", func(t *testing.T) {
		ctx := context.Background()
		updates = nil

		// Simulate tool call
		err := client.SessionUpdate(ctx, acpsdk.SessionNotification{
			SessionId: "test-session",
			Update: acpsdk.SessionUpdate{
				ToolCall: &acpsdk.SessionUpdateToolCall{
					ToolCallId: "tool-1",
					Title:      "Test Tool",
					Status:     "started",
					RawInput: map[string]interface{}{
						"param": "value",
					},
				},
			},
		})

		if err != nil {
			t.Errorf("SessionUpdate failed: %v", err)
		}

		if len(updates) != 1 {
			t.Errorf("Expected 1 update, got %d", len(updates))
		}

		if len(updates) > 0 {
			if updates[0].ToolCall == nil {
				t.Error("Expected tool call info, got nil")
			} else {
				if updates[0].ToolCall.Name != "Test Tool" {
					t.Errorf("Expected tool name 'Test Tool', got %q", updates[0].ToolCall.Name)
				}
				if updates[0].ToolCall.Status != "started" {
					t.Errorf("Expected status 'started', got %q", updates[0].ToolCall.Status)
				}
			}
		}

		// Verify tool call count
		if count := client.GetToolCallCount(); count != 1 {
			t.Errorf("Expected tool call count 1, got %d", count)
		}
	})
}

// TestACPClientPermissions tests permission request handling.
func TestACPClientPermissions(t *testing.T) {
	tempDir := t.TempDir()
	logFile, err := os.CreateTemp("", "acp-test-*.log")
	if err != nil {
		t.Fatalf("Failed to create log file: %v", err)
	}
	defer logFile.Close()

	client := NewMesnadaACPClient("test-task", tempDir, logFile, nil)

	t.Run("AutoPermission_Enabled", func(t *testing.T) {
		client.SetAutoPermission(true)

		ctx := context.Background()
		option1 := acpsdk.PermissionOptionId("option-1")
		resp, err := client.RequestPermission(ctx, acpsdk.RequestPermissionRequest{
			SessionId: "test-session",
			ToolCall: acpsdk.RequestPermissionToolCall{
				Title: ptrString("Test Permission"),
			},
			Options: []acpsdk.PermissionOption{
				{
					OptionId: option1,
					Name:     "Approve",
				},
			},
		})

		if err != nil {
			t.Errorf("RequestPermission failed: %v", err)
		}

		if resp.Outcome.Selected == nil {
			t.Error("Expected selected outcome, got nil")
		} else if resp.Outcome.Selected.OptionId != option1 {
			t.Errorf("Expected option %s, got %s", option1, resp.Outcome.Selected.OptionId)
		}
	})

	t.Run("AutoPermission_Disabled", func(t *testing.T) {
		client.SetAutoPermission(false)

		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()

		go func() {
			time.Sleep(50 * time.Millisecond)
			// Resolve permission after delay
			queue := client.GetPermissionQueue()
			pending := queue.GetPending("test-task")
			if len(pending) > 0 {
				option1 := acpsdk.PermissionOptionId("option-1")
				queue.ResolvePermission(pending[0].RequestID, acpsdk.NewRequestPermissionOutcomeSelected(option1))
			}
		}()

		option1 := acpsdk.PermissionOptionId("option-1")
		_, err := client.RequestPermission(ctx, acpsdk.RequestPermissionRequest{
			SessionId: "test-session",
			ToolCall: acpsdk.RequestPermissionToolCall{
				Title: ptrString("Test Permission"),
			},
			Options: []acpsdk.PermissionOption{
				{
					OptionId: option1,
					Name:     "Approve",
				},
			},
		})

		if err != nil {
			t.Errorf("RequestPermission failed: %v", err)
		}
	})
}

// TestACPClientCapabilities tests capability enforcement.
func TestACPClientCapabilities(t *testing.T) {
	tempDir := t.TempDir()
	logFile, err := os.CreateTemp("", "acp-test-*.log")
	if err != nil {
		t.Fatalf("Failed to create log file: %v", err)
	}
	defer logFile.Close()

	client := NewMesnadaACPClient("test-task", tempDir, logFile, nil)

	t.Run("FileAccess_Disabled", func(t *testing.T) {
		// Disable file access
		client.SetCapabilities(ACPCapabilities{
			Terminals:   true,
			FileAccess:  false,
			Permissions: true,
		})

		ctx := context.Background()
		_, err := client.ReadTextFile(ctx, acpsdk.ReadTextFileRequest{
			Path: "test.txt",
		})

		if err == nil {
			t.Error("Expected error when file access is disabled, got nil")
		}
	})

	t.Run("Terminals_Disabled", func(t *testing.T) {
		// Disable terminals
		client.SetCapabilities(ACPCapabilities{
			Terminals:   false,
			FileAccess:  true,
			Permissions: true,
		})

		ctx := context.Background()
		_, err := client.CreateTerminal(ctx, acpsdk.CreateTerminalRequest{
			Command: "echo",
			Args:    []string{"hello"},
		})

		if err == nil {
			t.Error("Expected error when terminals are disabled, got nil")
		}
	})
}

// ptrString returns a pointer to a string.
func ptrString(s string) *string {
	return &s
}
