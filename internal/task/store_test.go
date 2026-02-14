package task

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestNewStore(t *testing.T) {
	s := NewStore("/tmp/test.json")

	if s.path != "/tmp/test.json" {
		t.Errorf("path = %q, want %q", s.path, "/tmp/test.json")
	}
	if s.store == nil {
		t.Error("store should not be nil")
	}
	if s.store.Metadata.Version != "1.0" {
		t.Errorf("Version = %q, want %q", s.store.Metadata.Version, "1.0")
	}
	if len(s.store.Tasks) != 0 {
		t.Errorf("Tasks length = %d, want 0", len(s.store.Tasks))
	}
}

func TestNewStoreInDir(t *testing.T) {
	s := NewStoreInDir("/tmp/.ralph")

	expected := "/tmp/.ralph/tasks.json"
	if s.path != expected {
		t.Errorf("path = %q, want %q", s.path, expected)
	}
}

func TestStore_Path(t *testing.T) {
	s := NewStore("/some/path/tasks.json")
	if s.Path() != "/some/path/tasks.json" {
		t.Errorf("Path() = %q, want %q", s.Path(), "/some/path/tasks.json")
	}
}

func TestStore_LoadAndSave(t *testing.T) {
	tmpDir := t.TempDir()
	storePath := filepath.Join(tmpDir, "tasks.json")

	// Create a store, add tasks, save
	s1 := NewStore(storePath)
	task1 := NewTask("TASK-001", "First Task", "Description 1")
	task2 := NewTask("TASK-002", "Second Task", "Description 2")
	task2.MarkCompleted()

	if err := s1.Add(task1); err != nil {
		t.Fatalf("Add task1: %v", err)
	}
	if err := s1.Add(task2); err != nil {
		t.Fatalf("Add task2: %v", err)
	}
	if err := s1.Save(); err != nil {
		t.Fatalf("Save: %v", err)
	}

	// Load into a new store
	s2 := NewStore(storePath)
	if err := s2.Load(); err != nil {
		t.Fatalf("Load: %v", err)
	}

	if s2.Count() != 2 {
		t.Errorf("Count() = %d, want 2", s2.Count())
	}

	loaded1, ok := s2.Get("TASK-001")
	if !ok {
		t.Fatal("task TASK-001 not found")
	}
	if loaded1.Name != "First Task" {
		t.Errorf("Name = %q, want %q", loaded1.Name, "First Task")
	}

	loaded2, ok := s2.Get("TASK-002")
	if !ok {
		t.Fatal("task TASK-002 not found")
	}
	if loaded2.Status != StatusCompleted {
		t.Errorf("Status = %q, want %q", loaded2.Status, StatusCompleted)
	}
}

func TestStore_LoadNonExistent(t *testing.T) {
	tmpDir := t.TempDir()
	storePath := filepath.Join(tmpDir, "nonexistent.json")

	s := NewStore(storePath)
	if err := s.Load(); err != nil {
		t.Fatalf("Load should not error for nonexistent file: %v", err)
	}
	if s.Count() != 0 {
		t.Errorf("Count() = %d, want 0", s.Count())
	}
}

func TestStore_LoadInvalidJSON(t *testing.T) {
	tmpDir := t.TempDir()
	storePath := filepath.Join(tmpDir, "invalid.json")

	if err := os.WriteFile(storePath, []byte("not valid json"), 0644); err != nil {
		t.Fatal(err)
	}

	s := NewStore(storePath)
	if err := s.Load(); err == nil {
		t.Error("Load should error for invalid JSON")
	}
}

func TestStore_SaveCreatesDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	storePath := filepath.Join(tmpDir, "nested", "dir", "tasks.json")

	s := NewStore(storePath)
	s.Add(NewTask("TASK-001", "Test", "Desc"))

	if err := s.Save(); err != nil {
		t.Fatalf("Save: %v", err)
	}

	if _, err := os.Stat(storePath); os.IsNotExist(err) {
		t.Error("file should exist after save")
	}
}

func TestStore_Add(t *testing.T) {
	s := NewStore("/tmp/test.json")

	task := NewTask("TASK-001", "Test Task", "Description")
	if err := s.Add(task); err != nil {
		t.Fatalf("Add: %v", err)
	}

	if s.Count() != 1 {
		t.Errorf("Count() = %d, want 1", s.Count())
	}

	// Duplicate ID should fail
	task2 := NewTask("TASK-001", "Duplicate", "Should fail")
	if err := s.Add(task2); err == nil {
		t.Error("Add should error for duplicate ID")
	}
}

func TestStore_AddSetsOrder(t *testing.T) {
	s := NewStore("/tmp/test.json")

	task1 := NewTask("TASK-001", "First", "")
	task2 := NewTask("TASK-002", "Second", "")

	s.Add(task1)
	s.Add(task2)

	got1, _ := s.Get("TASK-001")
	got2, _ := s.Get("TASK-002")

	// First task should have order 0 (default)
	// Second task should have order 1
	if got2.Order <= got1.Order {
		t.Errorf("Second task order (%d) should be greater than first (%d)", got2.Order, got1.Order)
	}
}

func TestStore_AddAll(t *testing.T) {
	s := NewStore("/tmp/test.json")

	tasks := []*Task{
		NewTask("TASK-001", "First", ""),
		NewTask("TASK-002", "Second", ""),
		NewTask("TASK-003", "Third", ""),
	}

	if err := s.AddAll(tasks); err != nil {
		t.Fatalf("AddAll: %v", err)
	}

	if s.Count() != 3 {
		t.Errorf("Count() = %d, want 3", s.Count())
	}

	// Adding with duplicate should fail
	dupTasks := []*Task{
		NewTask("TASK-004", "Fourth", ""),
		NewTask("TASK-001", "Duplicate", ""),
	}
	if err := s.AddAll(dupTasks); err == nil {
		t.Error("AddAll should error for duplicate ID")
	}
}

func TestStore_Update(t *testing.T) {
	s := NewStore("/tmp/test.json")

	task := NewTask("TASK-001", "Original Name", "Original Desc")
	s.Add(task)

	// Update the task
	updated := task.Clone()
	updated.Name = "Updated Name"
	updated.MarkCompleted()

	if err := s.Update(updated); err != nil {
		t.Fatalf("Update: %v", err)
	}

	got, _ := s.Get("TASK-001")
	if got.Name != "Updated Name" {
		t.Errorf("Name = %q, want %q", got.Name, "Updated Name")
	}
	if got.Status != StatusCompleted {
		t.Errorf("Status = %q, want %q", got.Status, StatusCompleted)
	}

	// Update non-existent task should fail
	nonExistent := NewTask("TASK-999", "Not Found", "")
	if err := s.Update(nonExistent); err == nil {
		t.Error("Update should error for non-existent task")
	}
}

func TestStore_Delete(t *testing.T) {
	s := NewStore("/tmp/test.json")

	s.Add(NewTask("TASK-001", "First", ""))
	s.Add(NewTask("TASK-002", "Second", ""))

	if err := s.Delete("TASK-001"); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	if s.Count() != 1 {
		t.Errorf("Count() = %d, want 1", s.Count())
	}

	if s.Exists("TASK-001") {
		t.Error("TASK-001 should not exist after delete")
	}

	// Delete non-existent should fail
	if err := s.Delete("TASK-999"); err == nil {
		t.Error("Delete should error for non-existent task")
	}
}

func TestStore_Clear(t *testing.T) {
	s := NewStore("/tmp/test.json")

	s.Add(NewTask("TASK-001", "First", ""))
	s.Add(NewTask("TASK-002", "Second", ""))
	s.Clear()

	if s.Count() != 0 {
		t.Errorf("Count() = %d, want 0", s.Count())
	}
}

func TestStore_Exists(t *testing.T) {
	s := NewStore("/tmp/test.json")

	s.Add(NewTask("TASK-001", "First", ""))

	if !s.Exists("TASK-001") {
		t.Error("Exists should return true for TASK-001")
	}
	if s.Exists("TASK-999") {
		t.Error("Exists should return false for TASK-999")
	}
}

func TestStore_CountByStatus(t *testing.T) {
	s := NewStore("/tmp/test.json")

	task1 := NewTask("TASK-001", "First", "")
	task2 := NewTask("TASK-002", "Second", "")
	task2.MarkCompleted()
	task3 := NewTask("TASK-003", "Third", "")
	task3.MarkCompleted()

	s.Add(task1)
	s.Add(task2)
	s.Add(task3)

	if s.CountByStatus(StatusPending) != 1 {
		t.Errorf("CountByStatus(pending) = %d, want 1", s.CountByStatus(StatusPending))
	}
	if s.CountByStatus(StatusCompleted) != 2 {
		t.Errorf("CountByStatus(completed) = %d, want 2", s.CountByStatus(StatusCompleted))
	}
	if s.CountByStatus(StatusFailed) != 0 {
		t.Errorf("CountByStatus(failed) = %d, want 0", s.CountByStatus(StatusFailed))
	}
}

func TestStore_GetByStatus(t *testing.T) {
	s := NewStore("/tmp/test.json")

	task1 := NewTask("TASK-001", "First", "")
	task2 := NewTask("TASK-002", "Second", "")
	task2.MarkCompleted()
	task3 := NewTask("TASK-003", "Third", "")

	s.Add(task1)
	s.Add(task2)
	s.Add(task3)

	pending := s.GetByStatus(StatusPending)
	if len(pending) != 2 {
		t.Errorf("GetByStatus(pending) len = %d, want 2", len(pending))
	}

	completed := s.GetByStatus(StatusCompleted)
	if len(completed) != 1 {
		t.Errorf("GetByStatus(completed) len = %d, want 1", len(completed))
	}
	if completed[0].ID != "TASK-002" {
		t.Errorf("completed task ID = %q, want TASK-002", completed[0].ID)
	}
}

func TestStore_SetTasks(t *testing.T) {
	s := NewStore("/tmp/test.json")

	// Add initial tasks
	s.Add(NewTask("OLD-001", "Old Task", ""))

	// Replace with new tasks
	newTasks := []*Task{
		NewTask("NEW-001", "New First", ""),
		NewTask("NEW-002", "New Second", ""),
	}
	s.SetTasks(newTasks)

	if s.Count() != 2 {
		t.Errorf("Count() = %d, want 2", s.Count())
	}
	if s.Exists("OLD-001") {
		t.Error("OLD-001 should not exist after SetTasks")
	}
	if !s.Exists("NEW-001") {
		t.Error("NEW-001 should exist after SetTasks")
	}
}

func TestStore_Metadata(t *testing.T) {
	s := NewStore("/tmp/test.json")

	meta := s.Metadata()
	if meta.Version != "1.0" {
		t.Errorf("Version = %q, want %q", meta.Version, "1.0")
	}
	if meta.CreatedAt.IsZero() {
		t.Error("CreatedAt should not be zero")
	}
}

func TestStore_TasksReturnsClones(t *testing.T) {
	s := NewStore("/tmp/test.json")
	s.Add(NewTask("TASK-001", "Test", ""))

	tasks := s.Tasks()
	tasks[0].Name = "Modified"

	// Original should be unchanged
	got, _ := s.Get("TASK-001")
	if got.Name == "Modified" {
		t.Error("Tasks() should return clones, not references")
	}
}

func TestStore_GetReturnsClone(t *testing.T) {
	s := NewStore("/tmp/test.json")
	s.Add(NewTask("TASK-001", "Test", ""))

	got, _ := s.Get("TASK-001")
	got.Name = "Modified"

	// Original should be unchanged
	got2, _ := s.Get("TASK-001")
	if got2.Name == "Modified" {
		t.Error("Get() should return a clone, not reference")
	}
}

func TestStore_LoadFromJSONWithMetadata(t *testing.T) {
	tmpDir := t.TempDir()
	storePath := filepath.Join(tmpDir, "tasks.json")

	// Create a JSON file with metadata
	content := `{
		"metadata": {
			"version": "1.0",
			"created_at": "2024-01-01T00:00:00Z",
			"updated_at": "2024-01-02T00:00:00Z"
		},
		"tasks": [
			{
				"id": "TASK-001",
				"name": "First Task",
				"description": "Description 1",
				"status": "pending",
				"order": 1,
				"metadata": {"key": "value"},
				"iterations": []
			},
			{
				"id": "TASK-002",
				"name": "Second Task",
				"description": "Description 2",
				"status": "completed",
				"order": 2,
				"metadata": {},
				"iterations": [{"number": 1, "result": "DONE"}]
			}
		]
	}`

	if err := os.WriteFile(storePath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	s := NewStore(storePath)
	if err := s.Load(); err != nil {
		t.Fatalf("Load: %v", err)
	}

	// Verify metadata
	meta := s.Metadata()
	if meta.Version != "1.0" {
		t.Errorf("Version = %q, want %q", meta.Version, "1.0")
	}

	// Verify tasks
	if s.Count() != 2 {
		t.Errorf("Count() = %d, want 2", s.Count())
	}

	task1, ok := s.Get("TASK-001")
	if !ok {
		t.Fatal("TASK-001 not found")
	}
	if task1.Status != StatusPending {
		t.Errorf("Status = %q, want pending", task1.Status)
	}

	task2, ok := s.Get("TASK-002")
	if !ok {
		t.Fatal("TASK-002 not found")
	}
	if task2.Status != StatusCompleted {
		t.Errorf("Status = %q, want completed", task2.Status)
	}
	if len(task2.Iterations) != 1 {
		t.Errorf("Iterations length = %d, want 1", len(task2.Iterations))
	}
}

func TestStore_SaveUpdatesMedataTimestamp(t *testing.T) {
	tmpDir := t.TempDir()
	storePath := filepath.Join(tmpDir, "tasks.json")

	s := NewStore(storePath)
	originalUpdatedAt := s.Metadata().UpdatedAt

	// Wait briefly and save
	s.Add(NewTask("TASK-001", "Test", ""))
	if err := s.Save(); err != nil {
		t.Fatalf("Save: %v", err)
	}

	// Load into new store to verify timestamp updated
	s2 := NewStore(storePath)
	if err := s2.Load(); err != nil {
		t.Fatalf("Load: %v", err)
	}

	// The UpdatedAt should be set (might be same if test runs fast)
	if s2.Metadata().UpdatedAt.IsZero() {
		t.Error("UpdatedAt should not be zero")
	}
	// Should be at or after original
	if s2.Metadata().UpdatedAt.Before(originalUpdatedAt) {
		t.Error("UpdatedAt should be at or after original")
	}
}

func TestStore_AddPreservesExplicitOrder(t *testing.T) {
	s := NewStore("/tmp/test.json")

	task1 := NewTask("TASK-001", "First", "")
	task1.Order = 10
	s.Add(task1)

	got, _ := s.Get("TASK-001")
	if got.Order != 10 {
		t.Errorf("Order = %d, want 10 (explicit order should be preserved)", got.Order)
	}
}

func TestStore_AddMultipleTasks_OrderProgression(t *testing.T) {
	s := NewStore("/tmp/test.json")

	// Add first task with explicit order
	task1 := NewTask("TASK-001", "First", "")
	task1.Order = 5
	s.Add(task1)

	// Add second task without order - should get max(existing) + 1
	task2 := NewTask("TASK-002", "Second", "")
	s.Add(task2)

	got2, _ := s.Get("TASK-002")
	if got2.Order <= 5 {
		t.Errorf("Order = %d, want > 5", got2.Order)
	}
}

func TestStore_ConcurrentAccess(t *testing.T) {
	s := NewStore("/tmp/test.json")

	// Add some tasks
	for i := 1; i <= 10; i++ {
		s.Add(NewTask(fmt.Sprintf("TASK-%03d", i), "Task", ""))
	}

	// Concurrent reads
	done := make(chan bool)
	for i := 0; i < 5; i++ {
		go func() {
			for j := 0; j < 100; j++ {
				s.Count()
				s.Tasks()
				s.Get("TASK-001")
				s.Exists("TASK-005")
			}
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 5; i++ {
		<-done
	}
}
