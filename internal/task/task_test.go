package task

import (
	"testing"
	"time"
)

func TestTaskStatus_IsValid(t *testing.T) {
	tests := []struct {
		name   string
		status TaskStatus
		want   bool
	}{
		{"pending", StatusPending, true},
		{"in_progress", StatusInProgress, true},
		{"completed", StatusCompleted, true},
		{"skipped", StatusSkipped, true},
		{"paused", StatusPaused, true},
		{"failed", StatusFailed, true},
		{"invalid", TaskStatus("invalid"), false},
		{"empty", TaskStatus(""), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.status.IsValid(); got != tt.want {
				t.Errorf("TaskStatus.IsValid() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTaskStatus_IsTerminal(t *testing.T) {
	tests := []struct {
		name   string
		status TaskStatus
		want   bool
	}{
		{"pending", StatusPending, false},
		{"in_progress", StatusInProgress, false},
		{"completed", StatusCompleted, true},
		{"skipped", StatusSkipped, true},
		{"paused", StatusPaused, false},
		{"failed", StatusFailed, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.status.IsTerminal(); got != tt.want {
				t.Errorf("TaskStatus.IsTerminal() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTaskStatus_IsPending(t *testing.T) {
	tests := []struct {
		name   string
		status TaskStatus
		want   bool
	}{
		{"pending", StatusPending, true},
		{"in_progress", StatusInProgress, false},
		{"completed", StatusCompleted, false},
		{"skipped", StatusSkipped, false},
		{"paused", StatusPaused, true},
		{"failed", StatusFailed, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.status.IsPending(); got != tt.want {
				t.Errorf("TaskStatus.IsPending() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTaskStatus_String(t *testing.T) {
	tests := []struct {
		status TaskStatus
		want   string
	}{
		{StatusPending, "pending"},
		{StatusInProgress, "in_progress"},
		{StatusCompleted, "completed"},
		{StatusSkipped, "skipped"},
		{StatusPaused, "paused"},
		{StatusFailed, "failed"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := tt.status.String(); got != tt.want {
				t.Errorf("TaskStatus.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIteration_Duration(t *testing.T) {
	now := time.Now()

	t.Run("completed iteration", func(t *testing.T) {
		iter := Iteration{
			Number:    1,
			StartedAt: now,
			EndedAt:   now.Add(5 * time.Minute),
		}
		got := iter.Duration()
		if got != 5*time.Minute {
			t.Errorf("Duration() = %v, want %v", got, 5*time.Minute)
		}
	})

	t.Run("running iteration", func(t *testing.T) {
		iter := Iteration{
			Number:    1,
			StartedAt: now.Add(-1 * time.Second),
		}
		got := iter.Duration()
		if got < 1*time.Second {
			t.Errorf("Duration() = %v, want at least 1s", got)
		}
	})
}

func TestIteration_IsComplete(t *testing.T) {
	t.Run("complete", func(t *testing.T) {
		iter := Iteration{EndedAt: time.Now()}
		if !iter.IsComplete() {
			t.Error("expected IsComplete() to be true")
		}
	})

	t.Run("incomplete", func(t *testing.T) {
		iter := Iteration{}
		if iter.IsComplete() {
			t.Error("expected IsComplete() to be false")
		}
	})
}

func TestNewTask(t *testing.T) {
	task := NewTask("TASK-001", "Test Task", "Description here")

	if task.ID != "TASK-001" {
		t.Errorf("ID = %v, want TASK-001", task.ID)
	}
	if task.Name != "Test Task" {
		t.Errorf("Name = %v, want Test Task", task.Name)
	}
	if task.Description != "Description here" {
		t.Errorf("Description = %v, want Description here", task.Description)
	}
	if task.Status != StatusPending {
		t.Errorf("Status = %v, want pending", task.Status)
	}
	if task.Order != 0 {
		t.Errorf("Order = %v, want 0", task.Order)
	}
	if task.Iterations == nil {
		t.Error("Iterations should not be nil")
	}
	if len(task.Iterations) != 0 {
		t.Errorf("Iterations length = %v, want 0", len(task.Iterations))
	}
	if task.Metadata == nil {
		t.Error("Metadata should not be nil")
	}
	if task.CreatedAt.IsZero() {
		t.Error("CreatedAt should not be zero")
	}
	if task.UpdatedAt.IsZero() {
		t.Error("UpdatedAt should not be zero")
	}
}

func TestTask_IterationCount(t *testing.T) {
	task := NewTask("TASK-001", "Test", "Desc")
	if task.IterationCount() != 0 {
		t.Errorf("IterationCount() = %v, want 0", task.IterationCount())
	}

	task.StartIteration()
	if task.IterationCount() != 1 {
		t.Errorf("IterationCount() = %v, want 1", task.IterationCount())
	}

	task.StartIteration()
	if task.IterationCount() != 2 {
		t.Errorf("IterationCount() = %v, want 2", task.IterationCount())
	}
}

func TestTask_CurrentIteration(t *testing.T) {
	task := NewTask("TASK-001", "Test", "Desc")

	// No iterations yet
	if task.CurrentIteration() != nil {
		t.Error("expected CurrentIteration() to be nil")
	}

	// Start first iteration
	iter1 := task.StartIteration()
	current := task.CurrentIteration()
	if current == nil {
		t.Fatal("expected CurrentIteration() to not be nil")
	}
	if current.Number != iter1.Number {
		t.Errorf("Number = %v, want %v", current.Number, iter1.Number)
	}

	// Start second iteration
	task.StartIteration()
	current = task.CurrentIteration()
	if current.Number != 2 {
		t.Errorf("Number = %v, want 2", current.Number)
	}
}

func TestTask_StartIteration(t *testing.T) {
	task := NewTask("TASK-001", "Test", "Desc")

	iter := task.StartIteration()
	if iter == nil {
		t.Fatal("expected StartIteration() to return non-nil")
	}
	if iter.Number != 1 {
		t.Errorf("Number = %v, want 1", iter.Number)
	}
	if iter.StartedAt.IsZero() {
		t.Error("StartedAt should not be zero")
	}
	if task.Status != StatusInProgress {
		t.Errorf("Status = %v, want in_progress", task.Status)
	}

	// Start second iteration
	iter2 := task.StartIteration()
	if iter2.Number != 2 {
		t.Errorf("Number = %v, want 2", iter2.Number)
	}
}

func TestTask_EndIteration(t *testing.T) {
	t.Run("no iteration", func(t *testing.T) {
		task := NewTask("TASK-001", "Test", "Desc")
		err := task.EndIteration("DONE", "output", "session-123")
		if err == nil {
			t.Error("expected error for no active iteration")
		}
	})

	t.Run("already complete", func(t *testing.T) {
		task := NewTask("TASK-001", "Test", "Desc")
		task.StartIteration()
		task.EndIteration("DONE", "output", "session-123")
		err := task.EndIteration("DONE", "more output", "session-456")
		if err == nil {
			t.Error("expected error for already complete iteration")
		}
	})

	t.Run("success", func(t *testing.T) {
		task := NewTask("TASK-001", "Test", "Desc")
		task.StartIteration()

		err := task.EndIteration("DONE", "test output", "session-123")
		if err != nil {
			t.Fatalf("EndIteration() error = %v", err)
		}

		iter := task.CurrentIteration()
		if !iter.IsComplete() {
			t.Error("iteration should be complete")
		}
		if iter.Result != "DONE" {
			t.Errorf("Result = %v, want DONE", iter.Result)
		}
		if iter.AgentOutput != "test output" {
			t.Errorf("AgentOutput = %v, want test output", iter.AgentOutput)
		}
		if iter.SessionID != "session-123" {
			t.Errorf("SessionID = %v, want session-123", iter.SessionID)
		}
		if task.SessionID != "session-123" {
			t.Errorf("Task.SessionID = %v, want session-123", task.SessionID)
		}
	})
}

func TestTask_MarkCompleted(t *testing.T) {
	task := NewTask("TASK-001", "Test", "Desc")
	task.MarkCompleted()

	if task.Status != StatusCompleted {
		t.Errorf("Status = %v, want completed", task.Status)
	}
	if task.CompletedAt.IsZero() {
		t.Error("CompletedAt should not be zero")
	}
}

func TestTask_MarkFailed(t *testing.T) {
	task := NewTask("TASK-001", "Test", "Desc")
	task.MarkFailed()

	if task.Status != StatusFailed {
		t.Errorf("Status = %v, want failed", task.Status)
	}
}

func TestTask_MarkSkipped(t *testing.T) {
	task := NewTask("TASK-001", "Test", "Desc")
	task.MarkSkipped()

	if task.Status != StatusSkipped {
		t.Errorf("Status = %v, want skipped", task.Status)
	}
}

func TestTask_MarkPaused(t *testing.T) {
	task := NewTask("TASK-001", "Test", "Desc")
	task.StartIteration()
	task.MarkPaused()

	if task.Status != StatusPaused {
		t.Errorf("Status = %v, want paused", task.Status)
	}
}

func TestTask_Resume(t *testing.T) {
	t.Run("paused task", func(t *testing.T) {
		task := NewTask("TASK-001", "Test", "Desc")
		task.StartIteration()
		task.MarkPaused()

		iter := task.Resume()
		if iter == nil {
			t.Fatal("expected Resume() to return non-nil")
		}
		if iter.Number != 2 {
			t.Errorf("Number = %v, want 2", iter.Number)
		}
		if task.Status != StatusInProgress {
			t.Errorf("Status = %v, want in_progress", task.Status)
		}
	})

	t.Run("non-paused task", func(t *testing.T) {
		task := NewTask("TASK-001", "Test", "Desc")
		iter := task.Resume()
		if iter != nil {
			t.Error("expected Resume() to return nil for non-paused task")
		}
	})
}

func TestTask_Metadata(t *testing.T) {
	task := NewTask("TASK-001", "Test", "Desc")

	// Get non-existent key
	val, ok := task.GetMetadata("key")
	if ok {
		t.Error("expected GetMetadata() to return false for non-existent key")
	}
	if val != "" {
		t.Errorf("expected empty string, got %v", val)
	}

	// Set and get
	task.SetMetadata("key", "value")
	val, ok = task.GetMetadata("key")
	if !ok {
		t.Error("expected GetMetadata() to return true")
	}
	if val != "value" {
		t.Errorf("GetMetadata() = %v, want value", val)
	}

	// Overwrite
	task.SetMetadata("key", "newvalue")
	val, _ = task.GetMetadata("key")
	if val != "newvalue" {
		t.Errorf("GetMetadata() = %v, want newvalue", val)
	}
}

func TestTask_TotalDuration(t *testing.T) {
	task := NewTask("TASK-001", "Test", "Desc")

	if task.TotalDuration() != 0 {
		t.Errorf("TotalDuration() = %v, want 0", task.TotalDuration())
	}

	// Add a completed iteration with known duration
	now := time.Now()
	task.Iterations = append(task.Iterations, Iteration{
		Number:    1,
		StartedAt: now.Add(-5 * time.Minute),
		EndedAt:   now,
	})

	got := task.TotalDuration()
	if got != 5*time.Minute {
		t.Errorf("TotalDuration() = %v, want 5m", got)
	}

	// Add another iteration
	task.Iterations = append(task.Iterations, Iteration{
		Number:    2,
		StartedAt: now,
		EndedAt:   now.Add(3 * time.Minute),
	})

	got = task.TotalDuration()
	if got != 8*time.Minute {
		t.Errorf("TotalDuration() = %v, want 8m", got)
	}
}

func TestTask_Validate(t *testing.T) {
	tests := []struct {
		name    string
		task    *Task
		wantErr bool
	}{
		{
			name:    "valid task",
			task:    NewTask("TASK-001", "Test", "Desc"),
			wantErr: false,
		},
		{
			name: "missing ID",
			task: &Task{
				Name:   "Test",
				Status: StatusPending,
			},
			wantErr: true,
		},
		{
			name: "missing name",
			task: &Task{
				ID:     "TASK-001",
				Status: StatusPending,
			},
			wantErr: true,
		},
		{
			name: "invalid status",
			task: &Task{
				ID:     "TASK-001",
				Name:   "Test",
				Status: TaskStatus("invalid"),
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.task.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestTask_Clone(t *testing.T) {
	original := NewTask("TASK-001", "Test Task", "Description")
	original.Order = 5
	original.SessionID = "session-123"
	original.SetMetadata("key1", "value1")
	original.SetMetadata("key2", "value2")
	original.StartIteration()
	original.EndIteration("DONE", "output", "session-456")
	original.MarkCompleted()

	clone := original.Clone()

	// Verify basic fields
	if clone.ID != original.ID {
		t.Errorf("ID = %v, want %v", clone.ID, original.ID)
	}
	if clone.Name != original.Name {
		t.Errorf("Name = %v, want %v", clone.Name, original.Name)
	}
	if clone.Description != original.Description {
		t.Errorf("Description = %v, want %v", clone.Description, original.Description)
	}
	if clone.Status != original.Status {
		t.Errorf("Status = %v, want %v", clone.Status, original.Status)
	}
	if clone.Order != original.Order {
		t.Errorf("Order = %v, want %v", clone.Order, original.Order)
	}
	if clone.SessionID != original.SessionID {
		t.Errorf("SessionID = %v, want %v", clone.SessionID, original.SessionID)
	}

	// Verify iterations are deep copied
	if len(clone.Iterations) != len(original.Iterations) {
		t.Fatalf("Iterations length = %v, want %v", len(clone.Iterations), len(original.Iterations))
	}
	if clone.Iterations[0].Number != original.Iterations[0].Number {
		t.Errorf("Iteration.Number = %v, want %v", clone.Iterations[0].Number, original.Iterations[0].Number)
	}

	// Verify metadata is deep copied
	if len(clone.Metadata) != len(original.Metadata) {
		t.Fatalf("Metadata length = %v, want %v", len(clone.Metadata), len(original.Metadata))
	}

	// Verify independence (modifying clone doesn't affect original)
	clone.SetMetadata("key3", "value3")
	if _, ok := original.GetMetadata("key3"); ok {
		t.Error("modifying clone should not affect original metadata")
	}

	clone.ID = "TASK-002"
	if original.ID != "TASK-001" {
		t.Error("modifying clone should not affect original ID")
	}
}
