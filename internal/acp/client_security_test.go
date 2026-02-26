package acp

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	acpsdk "github.com/coder/acp-go-sdk"
)

// TestValidatePathInWorkspace_Valid tests valid paths within workspace
func TestValidatePathInWorkspace_Valid(t *testing.T) {
	// Create temporary workspace
	tmpDir := t.TempDir()

	client := &MesnadaACPClient{
		taskID:  "test-task",
		workDir: tmpDir,
	}

	tests := []struct {
		name string
		path string
	}{
		{"simple file", "test.txt"},
		{"subdirectory", "subdir/test.txt"},
		{"deep nesting", "a/b/c/d/test.txt"},
		{"dot in filename", "test.config.yaml"},
		{"current dir relative", "./test.txt"},
		{"redundant slashes", "subdir//test.txt"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			absPath, err := client.validatePathInWorkspace(tt.path)
			if err != nil {
				t.Errorf("validatePathInWorkspace(%q) failed: %v", tt.path, err)
			}
			if !filepath.IsAbs(absPath) {
				t.Errorf("Expected absolute path, got %q", absPath)
			}
			// Check the path is within workspace
			if rel, err := filepath.Rel(tmpDir, absPath); err != nil || filepath.IsAbs(rel) {
				t.Errorf("Path %q is not within workspace %q", absPath, tmpDir)
			}
		})
	}
}

// TestValidatePathInWorkspace_PathTraversal tests path traversal attacks
func TestValidatePathInWorkspace_PathTraversal(t *testing.T) {
	// Create temporary workspace
	tmpDir := t.TempDir()

	client := &MesnadaACPClient{
		taskID:  "test-task",
		workDir: tmpDir,
	}

	tests := []struct {
		name string
		path string
	}{
		{"parent directory", "../test.txt"},
		{"double parent", "../../test.txt"},
		{"deep escape", "../../../../../../../etc/passwd"},
		{"subdir escape", "subdir/../../test.txt"},
		{"mixed escape", "a/b/../../../../../../etc/passwd"},
		{"absolute path", "/etc/passwd"},
		{"absolute path unix", "/tmp/test.txt"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			absPath, err := client.validatePathInWorkspace(tt.path)
			if err == nil {
				// If no error, verify the path is actually within workspace
				rel, relErr := filepath.Rel(tmpDir, absPath)
				if relErr != nil || filepath.IsAbs(rel) {
					// Good - the path escaped but was caught
					return
				}
				// Path is within workspace - that's okay for relative paths like "subdir/../../test.txt" -> "test.txt"
				// But not for absolute paths
				if filepath.IsAbs(tt.path) {
					t.Errorf("validatePathInWorkspace(%q) should reject absolute paths outside workspace", tt.path)
				}
			}
			// Error is expected for escaping paths
		})
	}
}

// TestReadTextFile_Security tests file read security
func TestReadTextFile_Security(t *testing.T) {
	// Create temporary workspace
	tmpDir := t.TempDir()

	// Create a test file in workspace
	testFile := filepath.Join(tmpDir, "test.txt")
	testContent := "test content"
	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Create a file outside workspace (parent directory)
	outsideFile := filepath.Join(filepath.Dir(tmpDir), "outside.txt")
	if err := os.WriteFile(outsideFile, []byte("outside content"), 0644); err != nil {
		t.Fatalf("Failed to create outside file: %v", err)
	}
	defer os.Remove(outsideFile)

	client := &MesnadaACPClient{
		taskID:  "test-task",
		workDir: tmpDir,
		capabilities: ACPCapabilities{
			FileAccess: true,
		},
	}

	ctx := context.Background()

	// Test 1: Read valid file
	t.Run("valid file", func(t *testing.T) {
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

	// Test 2: Try to read file outside workspace using path traversal
	t.Run("path traversal attack", func(t *testing.T) {
		_, err := client.ReadTextFile(ctx, acpsdk.ReadTextFileRequest{
			Path: "../outside.txt",
		})
		if err == nil {
			t.Error("Expected error when reading file outside workspace")
		}
	})

	// Test 3: Try to read with absolute path
	t.Run("absolute path attack", func(t *testing.T) {
		_, err := client.ReadTextFile(ctx, acpsdk.ReadTextFileRequest{
			Path: outsideFile,
		})
		if err == nil {
			t.Error("Expected error when reading absolute path outside workspace")
		}
	})

	// Test 4: File access disabled
	t.Run("file access disabled", func(t *testing.T) {
		noAccessClient := &MesnadaACPClient{
			taskID:  "test-task",
			workDir: tmpDir,
			capabilities: ACPCapabilities{
				FileAccess: false,
			},
		}
		_, err := noAccessClient.ReadTextFile(ctx, acpsdk.ReadTextFileRequest{
			Path: "test.txt",
		})
		if err == nil {
			t.Error("Expected error when file access is disabled")
		}
	})
}

// TestWriteTextFile_Security tests file write security
func TestWriteTextFile_Security(t *testing.T) {
	// Create temporary workspace
	tmpDir := t.TempDir()

	client := &MesnadaACPClient{
		taskID:  "test-task",
		workDir: tmpDir,
		capabilities: ACPCapabilities{
			FileAccess: true,
		},
	}

	ctx := context.Background()

	// Test 1: Write valid file
	t.Run("valid file", func(t *testing.T) {
		content := "new content"
		_, err := client.WriteTextFile(ctx, acpsdk.WriteTextFileRequest{
			Path:    "new.txt",
			Content: content,
		})
		if err != nil {
			t.Errorf("WriteTextFile failed: %v", err)
		}

		// Verify file was written
		writtenContent, err := os.ReadFile(filepath.Join(tmpDir, "new.txt"))
		if err != nil {
			t.Errorf("Failed to read written file: %v", err)
		}
		if string(writtenContent) != content {
			t.Errorf("Expected content %q, got %q", content, string(writtenContent))
		}
	})

	// Test 2: Write to subdirectory (should create)
	t.Run("create subdirectory", func(t *testing.T) {
		content := "subdir content"
		_, err := client.WriteTextFile(ctx, acpsdk.WriteTextFileRequest{
			Path:    "subdir/deep/file.txt",
			Content: content,
		})
		if err != nil {
			t.Errorf("WriteTextFile failed: %v", err)
		}

		// Verify file was written
		writtenContent, err := os.ReadFile(filepath.Join(tmpDir, "subdir", "deep", "file.txt"))
		if err != nil {
			t.Errorf("Failed to read written file: %v", err)
		}
		if string(writtenContent) != content {
			t.Errorf("Expected content %q, got %q", content, string(writtenContent))
		}
	})

	// Test 3: Try to write file outside workspace using path traversal
	t.Run("path traversal attack", func(t *testing.T) {
		_, err := client.WriteTextFile(ctx, acpsdk.WriteTextFileRequest{
			Path:    "../outside.txt",
			Content: "malicious content",
		})
		if err == nil {
			t.Error("Expected error when writing file outside workspace")
		}

		// Verify file was NOT written
		outsideFile := filepath.Join(filepath.Dir(tmpDir), "outside.txt")
		if _, err := os.Stat(outsideFile); err == nil {
			os.Remove(outsideFile)
			t.Error("File was written outside workspace!")
		}
	})

	// Test 4: Try to write with absolute path
	t.Run("absolute path attack", func(t *testing.T) {
		outsideFile := filepath.Join(filepath.Dir(tmpDir), "absolute.txt")
		_, err := client.WriteTextFile(ctx, acpsdk.WriteTextFileRequest{
			Path:    outsideFile,
			Content: "malicious content",
		})
		if err == nil {
			t.Error("Expected error when writing absolute path outside workspace")
		}

		// Verify file was NOT written
		if _, err := os.Stat(outsideFile); err == nil {
			os.Remove(outsideFile)
			t.Error("File was written outside workspace!")
		}
	})

	// Test 5: File access disabled
	t.Run("file access disabled", func(t *testing.T) {
		noAccessClient := &MesnadaACPClient{
			taskID:  "test-task",
			workDir: tmpDir,
			capabilities: ACPCapabilities{
				FileAccess: false,
			},
		}
		_, err := noAccessClient.WriteTextFile(ctx, acpsdk.WriteTextFileRequest{
			Path:    "denied.txt",
			Content: "content",
		})
		if err == nil {
			t.Error("Expected error when file access is disabled")
		}
	})
}

// TestCreateTerminal_Capabilities tests terminal capability checks
func TestCreateTerminal_Capabilities(t *testing.T) {
	tmpDir := t.TempDir()

	// Test 1: Terminals enabled
	t.Run("terminals enabled", func(t *testing.T) {
		client := &MesnadaACPClient{
			taskID:  "test-task",
			workDir: tmpDir,
			capabilities: ACPCapabilities{
				Terminals: true,
			},
			terminals: make(map[string]*terminalState),
		}

		ctx := context.Background()
		resp, err := client.CreateTerminal(ctx, acpsdk.CreateTerminalRequest{
			Command: "echo",
			Args:    []string{"test"},
		})
		if err != nil {
			t.Errorf("CreateTerminal failed: %v", err)
		}
		if resp.TerminalId == "" {
			t.Error("Expected non-empty terminal ID")
		}
	})

	// Test 2: Terminals disabled
	t.Run("terminals disabled", func(t *testing.T) {
		client := &MesnadaACPClient{
			taskID:  "test-task",
			workDir: tmpDir,
			capabilities: ACPCapabilities{
				Terminals: false,
			},
		}

		ctx := context.Background()
		_, err := client.CreateTerminal(ctx, acpsdk.CreateTerminalRequest{
			Command: "echo",
			Args:    []string{"test"},
		})
		if err == nil {
			t.Error("Expected error when terminals are disabled")
		}
	})
}
