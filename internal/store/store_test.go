package store

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/sevir/mesnada/pkg/models"
)

func TestFileStore(t *testing.T) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "mesnada-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	storePath := filepath.Join(tmpDir, "tasks.json")

	// Create store
	store, err := NewFileStore(storePath)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer store.Close()

	t.Run("Save and Get", func(t *testing.T) {
		task := &models.Task{
			ID:        "test-1",
			Prompt:    "Test prompt",
			WorkDir:   "/test",
			Status:    models.TaskStatusPending,
			CreatedAt: time.Now(),
		}

		if err := store.Save(task); err != nil {
			t.Fatalf("Failed to save task: %v", err)
		}

		retrieved, err := store.Get("test-1")
		if err != nil {
			t.Fatalf("Failed to get task: %v", err)
		}

		if retrieved.ID != task.ID {
			t.Errorf("Expected ID %s, got %s", task.ID, retrieved.ID)
		}
		if retrieved.Prompt != task.Prompt {
			t.Errorf("Expected Prompt %s, got %s", task.Prompt, retrieved.Prompt)
		}
	})

	t.Run("Get non-existent", func(t *testing.T) {
		_, err := store.Get("non-existent")
		if err == nil {
			t.Error("Expected error for non-existent task")
		}
	})

	t.Run("List with filter", func(t *testing.T) {
		// Add more tasks
		tasks := []*models.Task{
			{
				ID:        "test-2",
				Prompt:    "Task 2",
				Status:    models.TaskStatusRunning,
				Tags:      []string{"important"},
				CreatedAt: time.Now(),
			},
			{
				ID:        "test-3",
				Prompt:    "Task 3",
				Status:    models.TaskStatusCompleted,
				Tags:      []string{"important", "done"},
				CreatedAt: time.Now(),
			},
			{
				ID:        "test-4",
				Prompt:    "Task 4",
				Status:    models.TaskStatusCompleted,
				CreatedAt: time.Now(),
			},
		}

		for _, task := range tasks {
			if err := store.Save(task); err != nil {
				t.Fatalf("Failed to save task: %v", err)
			}
		}

		// Filter by status
		result, err := store.List(ListFilter{
			Status: []models.TaskStatus{models.TaskStatusCompleted},
		})
		if err != nil {
			t.Fatalf("Failed to list tasks: %v", err)
		}
		if len(result) != 2 {
			t.Errorf("Expected 2 completed tasks, got %d", len(result))
		}

		// Filter by tags
		result, err = store.List(ListFilter{
			Tags: []string{"important"},
		})
		if err != nil {
			t.Fatalf("Failed to list tasks: %v", err)
		}
		if len(result) != 2 {
			t.Errorf("Expected 2 tasks with 'important' tag, got %d", len(result))
		}

		// Filter by status and tags
		result, err = store.List(ListFilter{
			Status: []models.TaskStatus{models.TaskStatusCompleted},
			Tags:   []string{"important"},
		})
		if err != nil {
			t.Fatalf("Failed to list tasks: %v", err)
		}
		if len(result) != 1 {
			t.Errorf("Expected 1 completed+important task, got %d", len(result))
		}
	})

	t.Run("List with limit and offset", func(t *testing.T) {
		result, err := store.List(ListFilter{Limit: 2})
		if err != nil {
			t.Fatalf("Failed to list tasks: %v", err)
		}
		if len(result) != 2 {
			t.Errorf("Expected 2 tasks with limit, got %d", len(result))
		}

		result, err = store.List(ListFilter{Offset: 2, Limit: 2})
		if err != nil {
			t.Fatalf("Failed to list tasks: %v", err)
		}
		if len(result) != 2 {
			t.Errorf("Expected 2 tasks with offset+limit, got %d", len(result))
		}
	})

	t.Run("UpdateStatus", func(t *testing.T) {
		if err := store.UpdateStatus("test-1", models.TaskStatusRunning); err != nil {
			t.Fatalf("Failed to update status: %v", err)
		}

		task, err := store.Get("test-1")
		if err != nil {
			t.Fatalf("Failed to get task: %v", err)
		}
		if task.Status != models.TaskStatusRunning {
			t.Errorf("Expected status %s, got %s", models.TaskStatusRunning, task.Status)
		}
	})

	t.Run("Delete", func(t *testing.T) {
		if err := store.Delete("test-1"); err != nil {
			t.Fatalf("Failed to delete task: %v", err)
		}

		_, err := store.Get("test-1")
		if err == nil {
			t.Error("Expected error for deleted task")
		}
	})

	t.Run("Delete non-existent", func(t *testing.T) {
		err := store.Delete("non-existent")
		if err == nil {
			t.Error("Expected error for non-existent task")
		}
	})
}

func TestFileStorePersistence(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "mesnada-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	storePath := filepath.Join(tmpDir, "tasks.json")

	// Create store and save task
	store1, err := NewFileStore(storePath)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}

	task := &models.Task{
		ID:        "persist-test",
		Prompt:    "Persistence test",
		Status:    models.TaskStatusPending,
		CreatedAt: time.Now(),
	}

	if err := store1.Save(task); err != nil {
		t.Fatalf("Failed to save task: %v", err)
	}

	// Force save and close
	if err := store1.ForceSave(); err != nil {
		t.Fatalf("Failed to force save: %v", err)
	}
	store1.Close()

	// Create new store and verify task exists
	store2, err := NewFileStore(storePath)
	if err != nil {
		t.Fatalf("Failed to create second store: %v", err)
	}
	defer store2.Close()

	retrieved, err := store2.Get("persist-test")
	if err != nil {
		t.Fatalf("Failed to get persisted task: %v", err)
	}

	if retrieved.ID != task.ID {
		t.Errorf("Expected ID %s, got %s", task.ID, retrieved.ID)
	}
	if retrieved.Prompt != task.Prompt {
		t.Errorf("Expected Prompt %s, got %s", task.Prompt, retrieved.Prompt)
	}
}
