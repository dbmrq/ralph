package task

import (
	"path/filepath"
	"testing"
)

func newTestManager(t *testing.T) *Manager {
	t.Helper()
	tmpDir := t.TempDir()
	store := NewStore(filepath.Join(tmpDir, "tasks.json"))
	return NewManager(store)
}

func TestNewManager(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStore(filepath.Join(tmpDir, "tasks.json"))
	m := NewManager(store)

	if m.store != store {
		t.Error("manager store should be the provided store")
	}
}

func TestManager_Store(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStore(filepath.Join(tmpDir, "tasks.json"))
	m := NewManager(store)

	if m.Store() != store {
		t.Error("Store() should return the provided store")
	}
}

func TestManager_LoadAndSave(t *testing.T) {
	tmpDir := t.TempDir()
	storePath := filepath.Join(tmpDir, "tasks.json")

	// Create and save
	m1 := NewManager(NewStore(storePath))
	task := NewTask("TASK-001", "Test", "Description")
	if err := m1.AddTask(task); err != nil {
		t.Fatalf("AddTask: %v", err)
	}
	if err := m1.Save(); err != nil {
		t.Fatalf("Save: %v", err)
	}

	// Load in new manager
	m2 := NewManager(NewStore(storePath))
	if err := m2.Load(); err != nil {
		t.Fatalf("Load: %v", err)
	}

	if m2.CountTotal() != 1 {
		t.Errorf("CountTotal() = %d, want 1", m2.CountTotal())
	}
}

func TestManager_GetNext_EmptyStore(t *testing.T) {
	m := newTestManager(t)

	if m.GetNext() != nil {
		t.Error("GetNext() should return nil for empty store")
	}
}

func TestManager_GetNext_PendingTasks(t *testing.T) {
	m := newTestManager(t)

	task1 := NewTask("TASK-001", "First", "First task")
	task1.Order = 2
	task2 := NewTask("TASK-002", "Second", "Second task")
	task2.Order = 1

	m.AddTask(task1)
	m.AddTask(task2)

	next := m.GetNext()
	if next == nil {
		t.Fatal("GetNext() should return a task")
	}
	// TASK-002 has lower order, should be returned first
	if next.ID != "TASK-002" {
		t.Errorf("GetNext() = %q, want TASK-002", next.ID)
	}
}

func TestManager_GetNext_InProgressFirst(t *testing.T) {
	m := newTestManager(t)

	task1 := NewTask("TASK-001", "First", "First task")
	task1.Order = 1
	task2 := NewTask("TASK-002", "Second", "Second task")
	task2.Order = 2
	task2.Status = StatusInProgress

	m.AddTask(task1)
	m.AddTask(task2)

	next := m.GetNext()
	if next == nil {
		t.Fatal("GetNext() should return a task")
	}
	// In-progress tasks should be returned first
	if next.ID != "TASK-002" {
		t.Errorf("GetNext() = %q, want TASK-002 (in-progress)", next.ID)
	}
}

func TestManager_GetNext_PausedSecond(t *testing.T) {
	m := newTestManager(t)

	task1 := NewTask("TASK-001", "First", "First task")
	task1.Order = 1
	task2 := NewTask("TASK-002", "Second", "Second task")
	task2.Order = 2
	task2.Status = StatusPaused

	m.AddTask(task1)
	m.AddTask(task2)

	// Mark task1 as completed so paused task should be next
	m.MarkComplete("TASK-001")

	next := m.GetNext()
	if next == nil {
		t.Fatal("GetNext() should return a task")
	}
	// Paused tasks come before pending
	if next.ID != "TASK-002" {
		t.Errorf("GetNext() = %q, want TASK-002 (paused)", next.ID)
	}
}

func TestManager_GetNext_SkipsTerminal(t *testing.T) {
	m := newTestManager(t)

	task1 := NewTask("TASK-001", "First", "First task")
	task1.Order = 1
	task1.Status = StatusCompleted

	task2 := NewTask("TASK-002", "Second", "Second task")
	task2.Order = 2
	task2.Status = StatusSkipped

	task3 := NewTask("TASK-003", "Third", "Third task")
	task3.Order = 3

	m.AddTask(task1)
	m.AddTask(task2)
	m.AddTask(task3)

	next := m.GetNext()
	if next == nil {
		t.Fatal("GetNext() should return a task")
	}
	if next.ID != "TASK-003" {
		t.Errorf("GetNext() = %q, want TASK-003 (pending)", next.ID)
	}
}

func TestManager_GetByID(t *testing.T) {
	m := newTestManager(t)

	task := NewTask("TASK-001", "Test", "Description")
	m.AddTask(task)

	t.Run("existing", func(t *testing.T) {
		found, ok := m.GetByID("TASK-001")
		if !ok {
			t.Fatal("expected task to be found")
		}
		if found.ID != "TASK-001" {
			t.Errorf("ID = %q, want TASK-001", found.ID)
		}
	})

	t.Run("non-existing", func(t *testing.T) {
		_, ok := m.GetByID("NONEXISTENT")
		if ok {
			t.Error("expected task to not be found")
		}
	})
}

func TestManager_MarkComplete(t *testing.T) {
	m := newTestManager(t)

	task := NewTask("TASK-001", "Test", "Description")
	m.AddTask(task)

	t.Run("existing", func(t *testing.T) {
		err := m.MarkComplete("TASK-001")
		if err != nil {
			t.Fatalf("MarkComplete: %v", err)
		}

		updated, _ := m.GetByID("TASK-001")
		if updated.Status != StatusCompleted {
			t.Errorf("Status = %q, want completed", updated.Status)
		}
	})

	t.Run("non-existing", func(t *testing.T) {
		err := m.MarkComplete("NONEXISTENT")
		if err == nil {
			t.Error("expected error for non-existing task")
		}
	})
}

func TestManager_Skip(t *testing.T) {
	m := newTestManager(t)

	task := NewTask("TASK-001", "Test", "Description")
	m.AddTask(task)

	t.Run("existing", func(t *testing.T) {
		err := m.Skip("TASK-001")
		if err != nil {
			t.Fatalf("Skip: %v", err)
		}

		updated, _ := m.GetByID("TASK-001")
		if updated.Status != StatusSkipped {
			t.Errorf("Status = %q, want skipped", updated.Status)
		}
	})

	t.Run("non-existing", func(t *testing.T) {
		err := m.Skip("NONEXISTENT")
		if err == nil {
			t.Error("expected error for non-existing task")
		}
	})
}

func TestManager_Pause(t *testing.T) {
	m := newTestManager(t)

	task := NewTask("TASK-001", "Test", "Description")
	m.AddTask(task)

	t.Run("existing", func(t *testing.T) {
		err := m.Pause("TASK-001")
		if err != nil {
			t.Fatalf("Pause: %v", err)
		}

		updated, _ := m.GetByID("TASK-001")
		if updated.Status != StatusPaused {
			t.Errorf("Status = %q, want paused", updated.Status)
		}
	})

	t.Run("non-existing", func(t *testing.T) {
		err := m.Pause("NONEXISTENT")
		if err == nil {
			t.Error("expected error for non-existing task")
		}
	})
}

func TestManager_MarkFailed(t *testing.T) {
	m := newTestManager(t)

	task := NewTask("TASK-001", "Test", "Description")
	m.AddTask(task)

	t.Run("existing", func(t *testing.T) {
		err := m.MarkFailed("TASK-001")
		if err != nil {
			t.Fatalf("MarkFailed: %v", err)
		}

		updated, _ := m.GetByID("TASK-001")
		if updated.Status != StatusFailed {
			t.Errorf("Status = %q, want failed", updated.Status)
		}
	})

	t.Run("non-existing", func(t *testing.T) {
		err := m.MarkFailed("NONEXISTENT")
		if err == nil {
			t.Error("expected error for non-existing task")
		}
	})
}

func TestManager_CountRemaining(t *testing.T) {
	m := newTestManager(t)

	// No tasks
	if m.CountRemaining() != 0 {
		t.Errorf("CountRemaining() = %d, want 0", m.CountRemaining())
	}

	// Add tasks with various statuses
	task1 := NewTask("TASK-001", "T1", "D1")
	task1.Status = StatusPending
	task2 := NewTask("TASK-002", "T2", "D2")
	task2.Status = StatusInProgress
	task3 := NewTask("TASK-003", "T3", "D3")
	task3.Status = StatusPaused
	task4 := NewTask("TASK-004", "T4", "D4")
	task4.Status = StatusCompleted
	task5 := NewTask("TASK-005", "T5", "D5")
	task5.Status = StatusSkipped
	task6 := NewTask("TASK-006", "T6", "D6")
	task6.Status = StatusFailed

	m.AddTask(task1)
	m.AddTask(task2)
	m.AddTask(task3)
	m.AddTask(task4)
	m.AddTask(task5)
	m.AddTask(task6)

	// Remaining = pending + in_progress + paused = 3
	if m.CountRemaining() != 3 {
		t.Errorf("CountRemaining() = %d, want 3", m.CountRemaining())
	}
}

func TestManager_CountCompleted(t *testing.T) {
	m := newTestManager(t)

	task1 := NewTask("TASK-001", "T1", "D1")
	task1.Status = StatusCompleted
	task2 := NewTask("TASK-002", "T2", "D2")
	task2.Status = StatusPending

	m.AddTask(task1)
	m.AddTask(task2)

	if m.CountCompleted() != 1 {
		t.Errorf("CountCompleted() = %d, want 1", m.CountCompleted())
	}
}

func TestManager_CountTotal(t *testing.T) {
	m := newTestManager(t)

	if m.CountTotal() != 0 {
		t.Errorf("CountTotal() = %d, want 0", m.CountTotal())
	}

	m.AddTask(NewTask("TASK-001", "T1", "D1"))
	m.AddTask(NewTask("TASK-002", "T2", "D2"))

	if m.CountTotal() != 2 {
		t.Errorf("CountTotal() = %d, want 2", m.CountTotal())
	}
}

func TestManager_HasRemaining(t *testing.T) {
	m := newTestManager(t)

	if m.HasRemaining() {
		t.Error("HasRemaining() should be false for empty store")
	}

	task := NewTask("TASK-001", "Test", "Description")
	m.AddTask(task)

	if !m.HasRemaining() {
		t.Error("HasRemaining() should be true with pending task")
	}

	m.MarkComplete("TASK-001")

	if m.HasRemaining() {
		t.Error("HasRemaining() should be false when all tasks are complete")
	}
}

func TestManager_All(t *testing.T) {
	m := newTestManager(t)

	// Empty store
	if len(m.All()) != 0 {
		t.Errorf("All() length = %d, want 0", len(m.All()))
	}

	// Add tasks out of order
	task1 := NewTask("TASK-001", "T1", "D1")
	task1.Order = 3
	task2 := NewTask("TASK-002", "T2", "D2")
	task2.Order = 1
	task3 := NewTask("TASK-003", "T3", "D3")
	task3.Order = 2

	m.AddTask(task1)
	m.AddTask(task2)
	m.AddTask(task3)

	all := m.All()
	if len(all) != 3 {
		t.Fatalf("All() length = %d, want 3", len(all))
	}

	// Should be sorted by order
	if all[0].ID != "TASK-002" {
		t.Errorf("all[0].ID = %q, want TASK-002", all[0].ID)
	}
	if all[1].ID != "TASK-003" {
		t.Errorf("all[1].ID = %q, want TASK-003", all[1].ID)
	}
	if all[2].ID != "TASK-001" {
		t.Errorf("all[2].ID = %q, want TASK-001", all[2].ID)
	}
}

func TestManager_StartIteration(t *testing.T) {
	m := newTestManager(t)

	task := NewTask("TASK-001", "Test", "Description")
	m.AddTask(task)

	t.Run("existing", func(t *testing.T) {
		iter, err := m.StartIteration("TASK-001")
		if err != nil {
			t.Fatalf("StartIteration: %v", err)
		}
		if iter.Number != 1 {
			t.Errorf("Number = %d, want 1", iter.Number)
		}

		// Verify status changed
		updated, _ := m.GetByID("TASK-001")
		if updated.Status != StatusInProgress {
			t.Errorf("Status = %q, want in_progress", updated.Status)
		}
	})

	t.Run("non-existing", func(t *testing.T) {
		_, err := m.StartIteration("NONEXISTENT")
		if err == nil {
			t.Error("expected error for non-existing task")
		}
	})
}

func TestManager_EndIteration(t *testing.T) {
	m := newTestManager(t)

	task := NewTask("TASK-001", "Test", "Description")
	m.AddTask(task)
	m.StartIteration("TASK-001")

	t.Run("existing", func(t *testing.T) {
		err := m.EndIteration("TASK-001", "DONE", "test output", "session-123")
		if err != nil {
			t.Fatalf("EndIteration: %v", err)
		}

		updated, _ := m.GetByID("TASK-001")
		iter := updated.CurrentIteration()
		if iter.Result != "DONE" {
			t.Errorf("Result = %q, want DONE", iter.Result)
		}
		if iter.SessionID != "session-123" {
			t.Errorf("SessionID = %q, want session-123", iter.SessionID)
		}
	})

	t.Run("non-existing", func(t *testing.T) {
		err := m.EndIteration("NONEXISTENT", "DONE", "output", "session")
		if err == nil {
			t.Error("expected error for non-existing task")
		}
	})
}

func TestManager_UpdateTask(t *testing.T) {
	m := newTestManager(t)

	task := NewTask("TASK-001", "Original", "Original desc")
	m.AddTask(task)

	// Update the task
	updated := NewTask("TASK-001", "Updated", "Updated desc")
	err := m.UpdateTask(updated)
	if err != nil {
		t.Fatalf("UpdateTask: %v", err)
	}

	found, _ := m.GetByID("TASK-001")
	if found.Name != "Updated" {
		t.Errorf("Name = %q, want Updated", found.Name)
	}
}

func TestManager_AddTask(t *testing.T) {
	m := newTestManager(t)

	task := NewTask("TASK-001", "Test", "Description")
	err := m.AddTask(task)
	if err != nil {
		t.Fatalf("AddTask: %v", err)
	}

	if m.CountTotal() != 1 {
		t.Errorf("CountTotal() = %d, want 1", m.CountTotal())
	}

	// Adding duplicate should fail
	err = m.AddTask(task)
	if err == nil {
		t.Error("expected error for duplicate task")
	}
}

func TestManager_Progress(t *testing.T) {
	m := newTestManager(t)

	// Empty store
	completed, total := m.Progress()
	if completed != 0 || total != 0 {
		t.Errorf("Progress() = (%d, %d), want (0, 0)", completed, total)
	}

	// Add tasks
	task1 := NewTask("TASK-001", "T1", "D1")
	task1.Status = StatusCompleted
	task2 := NewTask("TASK-002", "T2", "D2")
	task2.Status = StatusPending
	task3 := NewTask("TASK-003", "T3", "D3")
	task3.Status = StatusCompleted

	m.AddTask(task1)
	m.AddTask(task2)
	m.AddTask(task3)

	completed, total = m.Progress()
	if completed != 2 || total != 3 {
		t.Errorf("Progress() = (%d, %d), want (2, 3)", completed, total)
	}
}

func TestManager_GetByStatus(t *testing.T) {
	m := newTestManager(t)

	task1 := NewTask("TASK-001", "T1", "D1")
	task1.Status = StatusCompleted
	task2 := NewTask("TASK-002", "T2", "D2")
	task2.Status = StatusPending
	task3 := NewTask("TASK-003", "T3", "D3")
	task3.Status = StatusPending

	m.AddTask(task1)
	m.AddTask(task2)
	m.AddTask(task3)

	pending := m.GetByStatus(StatusPending)
	if len(pending) != 2 {
		t.Errorf("GetByStatus(pending) length = %d, want 2", len(pending))
	}

	completed := m.GetByStatus(StatusCompleted)
	if len(completed) != 1 {
		t.Errorf("GetByStatus(completed) length = %d, want 1", len(completed))
	}
}

func TestManager_Resume(t *testing.T) {
	m := newTestManager(t)

	task := NewTask("TASK-001", "Test", "Description")
	task.Status = StatusPaused
	m.AddTask(task)

	t.Run("paused task", func(t *testing.T) {
		iter, err := m.Resume("TASK-001")
		if err != nil {
			t.Fatalf("Resume: %v", err)
		}
		if iter == nil {
			t.Fatal("expected non-nil iteration")
		}

		updated, _ := m.GetByID("TASK-001")
		if updated.Status != StatusInProgress {
			t.Errorf("Status = %q, want in_progress", updated.Status)
		}
	})

	t.Run("non-paused task", func(t *testing.T) {
		// Task is now in-progress from previous test
		_, err := m.Resume("TASK-001")
		if err == nil {
			t.Error("expected error for non-paused task")
		}
	})

	t.Run("non-existing", func(t *testing.T) {
		_, err := m.Resume("NONEXISTENT")
		if err == nil {
			t.Error("expected error for non-existing task")
		}
	})
}

func TestManager_ReorderTasks(t *testing.T) {
	m := newTestManager(t)

	task1 := NewTask("TASK-001", "T1", "D1")
	task1.Order = 1
	task2 := NewTask("TASK-002", "T2", "D2")
	task2.Order = 2
	task3 := NewTask("TASK-003", "T3", "D3")
	task3.Order = 3

	m.AddTask(task1)
	m.AddTask(task2)
	m.AddTask(task3)

	// Reorder: TASK-003 first, then TASK-001
	err := m.ReorderTasks([]string{"TASK-003", "TASK-001"})
	if err != nil {
		t.Fatalf("ReorderTasks: %v", err)
	}

	all := m.All()
	// Expected order: TASK-003 (1), TASK-001 (2), TASK-002 (3)
	if all[0].ID != "TASK-003" {
		t.Errorf("all[0].ID = %q, want TASK-003", all[0].ID)
	}
	if all[1].ID != "TASK-001" {
		t.Errorf("all[1].ID = %q, want TASK-001", all[1].ID)
	}
	if all[2].ID != "TASK-002" {
		t.Errorf("all[2].ID = %q, want TASK-002", all[2].ID)
	}
}

func TestManager_ReorderTasks_PartialList(t *testing.T) {
	m := newTestManager(t)

	task1 := NewTask("TASK-001", "T1", "D1")
	task1.Order = 1
	task2 := NewTask("TASK-002", "T2", "D2")
	task2.Order = 2
	task3 := NewTask("TASK-003", "T3", "D3")
	task3.Order = 3
	task4 := NewTask("TASK-004", "T4", "D4")
	task4.Order = 4

	m.AddTask(task1)
	m.AddTask(task2)
	m.AddTask(task3)
	m.AddTask(task4)

	// Only specify order for TASK-004 and TASK-002
	err := m.ReorderTasks([]string{"TASK-004", "TASK-002"})
	if err != nil {
		t.Fatalf("ReorderTasks: %v", err)
	}

	all := m.All()
	// Expected: TASK-004 (1), TASK-002 (2), then others
	if all[0].ID != "TASK-004" {
		t.Errorf("all[0].ID = %q, want TASK-004", all[0].ID)
	}
	if all[1].ID != "TASK-002" {
		t.Errorf("all[1].ID = %q, want TASK-002", all[1].ID)
	}
}

