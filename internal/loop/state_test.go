package loop

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/wexinc/ralph/internal/task"
)

func TestState_IsValid(t *testing.T) {
	tests := []struct {
		state State
		want  bool
	}{
		{StateIdle, true},
		{StateRunning, true},
		{StatePaused, true},
		{StateAwaitingFix, true},
		{StateCompleted, true},
		{StateFailed, true},
		{State("invalid"), false},
		{State(""), false},
	}

	for _, tt := range tests {
		t.Run(string(tt.state), func(t *testing.T) {
			if got := tt.state.IsValid(); got != tt.want {
				t.Errorf("State(%q).IsValid() = %v, want %v", tt.state, got, tt.want)
			}
		})
	}
}

func TestState_IsTerminal(t *testing.T) {
	tests := []struct {
		state State
		want  bool
	}{
		{StateIdle, false},
		{StateRunning, false},
		{StatePaused, false},
		{StateAwaitingFix, false},
		{StateCompleted, true},
		{StateFailed, true},
	}

	for _, tt := range tests {
		t.Run(string(tt.state), func(t *testing.T) {
			if got := tt.state.IsTerminal(); got != tt.want {
				t.Errorf("State(%q).IsTerminal() = %v, want %v", tt.state, got, tt.want)
			}
		})
	}
}

func TestState_CanResume(t *testing.T) {
	tests := []struct {
		state State
		want  bool
	}{
		{StateIdle, false},
		{StateRunning, false},
		{StatePaused, true},
		{StateAwaitingFix, true},
		{StateCompleted, false},
		{StateFailed, false},
	}

	for _, tt := range tests {
		t.Run(string(tt.state), func(t *testing.T) {
			if got := tt.state.CanResume(); got != tt.want {
				t.Errorf("State(%q).CanResume() = %v, want %v", tt.state, got, tt.want)
			}
		})
	}
}

func TestState_CanTransitionTo(t *testing.T) {
	tests := []struct {
		from State
		to   State
		want bool
	}{
		// Idle can only go to Running
		{StateIdle, StateRunning, true},
		{StateIdle, StatePaused, false},
		{StateIdle, StateCompleted, false},

		// Running can go to multiple states
		{StateRunning, StatePaused, true},
		{StateRunning, StateAwaitingFix, true},
		{StateRunning, StateCompleted, true},
		{StateRunning, StateFailed, true},
		{StateRunning, StateIdle, false},

		// Paused can go back to Running or fail
		{StatePaused, StateRunning, true},
		{StatePaused, StateFailed, true},
		{StatePaused, StateCompleted, false},

		// AwaitingFix can go back to Running or fail
		{StateAwaitingFix, StateRunning, true},
		{StateAwaitingFix, StateFailed, true},
		{StateAwaitingFix, StateCompleted, false},

		// Completed is terminal
		{StateCompleted, StateIdle, false},
		{StateCompleted, StateRunning, false},

		// Failed can restart
		{StateFailed, StateIdle, true},
		{StateFailed, StateRunning, false},
	}

	for _, tt := range tests {
		name := string(tt.from) + "->" + string(tt.to)
		t.Run(name, func(t *testing.T) {
			if got := tt.from.CanTransitionTo(tt.to); got != tt.want {
				t.Errorf("CanTransitionTo(%q, %q) = %v, want %v", tt.from, tt.to, got, tt.want)
			}
		})
	}
}

func TestNewLoopContext(t *testing.T) {
	ctx := NewLoopContext("session-123", "/project", "auggie")

	if ctx.State != StateIdle {
		t.Errorf("State = %v, want %v", ctx.State, StateIdle)
	}
	if ctx.SessionID != "session-123" {
		t.Errorf("SessionID = %v, want session-123", ctx.SessionID)
	}
	if ctx.ProjectDir != "/project" {
		t.Errorf("ProjectDir = %v, want /project", ctx.ProjectDir)
	}
	if ctx.AgentName != "auggie" {
		t.Errorf("AgentName = %v, want auggie", ctx.AgentName)
	}
	if ctx.MaxFixAttempts != 3 {
		t.Errorf("MaxFixAttempts = %v, want 3", ctx.MaxFixAttempts)
	}
	if ctx.StartedAt.IsZero() {
		t.Error("StartedAt should not be zero")
	}
}

func TestLoopContext_Transition(t *testing.T) {
	t.Run("valid transition", func(t *testing.T) {
		ctx := NewLoopContext("session-123", "/project", "auggie")

		err := ctx.Transition(StateRunning)
		if err != nil {
			t.Fatalf("Transition to Running failed: %v", err)
		}
		if ctx.State != StateRunning {
			t.Errorf("State = %v, want %v", ctx.State, StateRunning)
		}
	})

	t.Run("invalid transition", func(t *testing.T) {
		ctx := NewLoopContext("session-123", "/project", "auggie")

		err := ctx.Transition(StateCompleted)
		if err == nil {
			t.Error("Expected error for invalid transition")
		}
		if _, ok := err.(*TransitionError); !ok {
			t.Errorf("Expected TransitionError, got %T", err)
		}
		if ctx.State != StateIdle {
			t.Errorf("State should remain %v, got %v", StateIdle, ctx.State)
		}
	})

	t.Run("pause sets PausedAt", func(t *testing.T) {
		ctx := NewLoopContext("session-123", "/project", "auggie")
		ctx.Transition(StateRunning)

		before := time.Now()
		ctx.Transition(StatePaused)
		after := time.Now()

		if ctx.PausedAt.Before(before) || ctx.PausedAt.After(after) {
			t.Error("PausedAt should be set to current time")
		}
	})

	t.Run("resume clears PausedAt", func(t *testing.T) {
		ctx := NewLoopContext("session-123", "/project", "auggie")
		ctx.Transition(StateRunning)
		ctx.Transition(StatePaused)

		if ctx.PausedAt.IsZero() {
			t.Fatal("PausedAt should be set")
		}

		ctx.Transition(StateRunning)
		if !ctx.PausedAt.IsZero() {
			t.Error("PausedAt should be cleared on resume")
		}
	})

	t.Run("awaiting fix increments FixAttempts", func(t *testing.T) {
		ctx := NewLoopContext("session-123", "/project", "auggie")
		ctx.Transition(StateRunning)

		ctx.Transition(StateAwaitingFix)
		if ctx.FixAttempts != 1 {
			t.Errorf("FixAttempts = %v, want 1", ctx.FixAttempts)
		}

		ctx.Transition(StateRunning)
		ctx.Transition(StateAwaitingFix)
		if ctx.FixAttempts != 2 {
			t.Errorf("FixAttempts = %v, want 2", ctx.FixAttempts)
		}
	})
}

func TestLoopContext_SetCurrentTask(t *testing.T) {
	ctx := NewLoopContext("session-123", "/project", "auggie")
	ctx.CurrentIteration = 5
	ctx.FixAttempts = 2

	ctx.SetCurrentTask("TASK-001")

	if ctx.CurrentTaskID != "TASK-001" {
		t.Errorf("CurrentTaskID = %v, want TASK-001", ctx.CurrentTaskID)
	}
	if ctx.CurrentIteration != 0 {
		t.Errorf("CurrentIteration = %v, want 0", ctx.CurrentIteration)
	}
	if ctx.FixAttempts != 0 {
		t.Errorf("FixAttempts = %v, want 0", ctx.FixAttempts)
	}
}

func TestLoopContext_IncrementIteration(t *testing.T) {
	ctx := NewLoopContext("session-123", "/project", "auggie")

	ctx.IncrementIteration()
	if ctx.CurrentIteration != 1 {
		t.Errorf("CurrentIteration = %v, want 1", ctx.CurrentIteration)
	}
	if ctx.TotalIterations != 1 {
		t.Errorf("TotalIterations = %v, want 1", ctx.TotalIterations)
	}

	ctx.IncrementIteration()
	if ctx.CurrentIteration != 2 {
		t.Errorf("CurrentIteration = %v, want 2", ctx.CurrentIteration)
	}
	if ctx.TotalIterations != 2 {
		t.Errorf("TotalIterations = %v, want 2", ctx.TotalIterations)
	}
}

func TestLoopContext_RecordTaskCompletion(t *testing.T) {
	tests := []struct {
		status        task.TaskStatus
		wantCompleted int
		wantFailed    int
		wantSkipped   int
	}{
		{task.StatusCompleted, 1, 0, 0},
		{task.StatusFailed, 0, 1, 0},
		{task.StatusSkipped, 0, 0, 1},
	}

	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			ctx := NewLoopContext("session-123", "/project", "auggie")
			ctx.CurrentTaskID = "TASK-001"
			ctx.CurrentIteration = 5

			ctx.RecordTaskCompletion(tt.status)

			if ctx.TasksCompleted != tt.wantCompleted {
				t.Errorf("TasksCompleted = %v, want %v", ctx.TasksCompleted, tt.wantCompleted)
			}
			if ctx.TasksFailed != tt.wantFailed {
				t.Errorf("TasksFailed = %v, want %v", ctx.TasksFailed, tt.wantFailed)
			}
			if ctx.TasksSkipped != tt.wantSkipped {
				t.Errorf("TasksSkipped = %v, want %v", ctx.TasksSkipped, tt.wantSkipped)
			}
			if ctx.CurrentTaskID != "" {
				t.Errorf("CurrentTaskID should be cleared")
			}
			if ctx.CurrentIteration != 0 {
				t.Errorf("CurrentIteration should be reset")
			}
		})
	}
}

func TestLoopContext_CanAttemptFix(t *testing.T) {
	ctx := NewLoopContext("session-123", "/project", "auggie")
	ctx.MaxFixAttempts = 3

	tests := []struct {
		attempts int
		want     bool
	}{
		{0, true},
		{1, true},
		{2, true},
		{3, false},
		{4, false},
	}

	for _, tt := range tests {
		ctx.FixAttempts = tt.attempts
		if got := ctx.CanAttemptFix(); got != tt.want {
			t.Errorf("CanAttemptFix() with %d attempts = %v, want %v", tt.attempts, got, tt.want)
		}
	}
}

func TestLoopContext_Clone(t *testing.T) {
	ctx := NewLoopContext("session-123", "/project", "auggie")
	ctx.State = StateRunning
	ctx.CurrentTaskID = "TASK-001"
	ctx.TasksCompleted = 5

	clone := ctx.Clone()

	// Verify clone has same values
	if clone.SessionID != ctx.SessionID {
		t.Errorf("Clone SessionID = %v, want %v", clone.SessionID, ctx.SessionID)
	}
	if clone.State != ctx.State {
		t.Errorf("Clone State = %v, want %v", clone.State, ctx.State)
	}
	if clone.CurrentTaskID != ctx.CurrentTaskID {
		t.Errorf("Clone CurrentTaskID = %v, want %v", clone.CurrentTaskID, ctx.CurrentTaskID)
	}

	// Verify it's a separate instance
	clone.TasksCompleted = 10
	if ctx.TasksCompleted == 10 {
		t.Error("Modifying clone should not affect original")
	}
}

func TestStatePersistence_SaveAndLoad(t *testing.T) {
	tmpDir := t.TempDir()
	persistence := NewStatePersistence(tmpDir)

	ctx := NewLoopContext("test-session", tmpDir, "auggie")
	ctx.State = StateRunning
	ctx.CurrentTaskID = "TASK-001"
	ctx.TasksCompleted = 3

	// Save
	err := persistence.Save(ctx)
	if err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// Verify file exists
	sessionPath := filepath.Join(tmpDir, DefaultSessionsDir, "test-session.json")
	if _, err := os.Stat(sessionPath); os.IsNotExist(err) {
		t.Fatal("Session file was not created")
	}

	// Load
	loaded, err := persistence.Load("test-session")
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if loaded.SessionID != ctx.SessionID {
		t.Errorf("Loaded SessionID = %v, want %v", loaded.SessionID, ctx.SessionID)
	}
	if loaded.State != ctx.State {
		t.Errorf("Loaded State = %v, want %v", loaded.State, ctx.State)
	}
	if loaded.CurrentTaskID != ctx.CurrentTaskID {
		t.Errorf("Loaded CurrentTaskID = %v, want %v", loaded.CurrentTaskID, ctx.CurrentTaskID)
	}
	if loaded.TasksCompleted != ctx.TasksCompleted {
		t.Errorf("Loaded TasksCompleted = %v, want %v", loaded.TasksCompleted, ctx.TasksCompleted)
	}
}

func TestStatePersistence_LoadNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	persistence := NewStatePersistence(tmpDir)

	_, err := persistence.Load("nonexistent")
	if err == nil {
		t.Error("Expected error for nonexistent session")
	}
}

func TestStatePersistence_LoadLatest(t *testing.T) {
	tmpDir := t.TempDir()
	persistence := NewStatePersistence(tmpDir)

	// Create older session
	ctx1 := NewLoopContext("session-1", tmpDir, "auggie")
	ctx1.UpdatedAt = time.Now().Add(-time.Hour)
	persistence.Save(ctx1)

	// Create newer session
	ctx2 := NewLoopContext("session-2", tmpDir, "cursor")
	ctx2.UpdatedAt = time.Now()
	persistence.Save(ctx2)

	// LoadLatest should return session-2
	latest, err := persistence.LoadLatest()
	if err != nil {
		t.Fatalf("LoadLatest failed: %v", err)
	}

	if latest.SessionID != "session-2" {
		t.Errorf("LoadLatest returned %v, want session-2", latest.SessionID)
	}
}

func TestStatePersistence_LoadResumable(t *testing.T) {
	tmpDir := t.TempDir()
	persistence := NewStatePersistence(tmpDir)

	// Create completed session (not resumable)
	ctx1 := NewLoopContext("session-1", tmpDir, "auggie")
	ctx1.State = StateCompleted
	ctx1.UpdatedAt = time.Now()
	persistence.Save(ctx1)

	// Create paused session (resumable)
	ctx2 := NewLoopContext("session-2", tmpDir, "cursor")
	ctx2.State = StatePaused
	ctx2.UpdatedAt = time.Now().Add(-time.Hour)
	persistence.Save(ctx2)

	// LoadResumable should return session-2 (the only resumable one)
	resumable, err := persistence.LoadResumable()
	if err != nil {
		t.Fatalf("LoadResumable failed: %v", err)
	}

	if resumable.SessionID != "session-2" {
		t.Errorf("LoadResumable returned %v, want session-2", resumable.SessionID)
	}
}

func TestStatePersistence_LoadResumable_NoResumable(t *testing.T) {
	tmpDir := t.TempDir()
	persistence := NewStatePersistence(tmpDir)

	// Create completed session (not resumable)
	ctx := NewLoopContext("session-1", tmpDir, "auggie")
	ctx.State = StateCompleted
	persistence.Save(ctx)

	// LoadResumable should return error
	_, err := persistence.LoadResumable()
	if err == nil {
		t.Error("Expected error when no resumable sessions exist")
	}
}

func TestStatePersistence_ListSessions(t *testing.T) {
	tmpDir := t.TempDir()
	persistence := NewStatePersistence(tmpDir)

	// Initially empty
	sessions, err := persistence.ListSessions()
	if err != nil {
		t.Fatalf("ListSessions failed: %v", err)
	}
	if len(sessions) != 0 {
		t.Errorf("Expected 0 sessions, got %d", len(sessions))
	}

	// Create sessions
	persistence.Save(NewLoopContext("session-1", tmpDir, "auggie"))
	persistence.Save(NewLoopContext("session-2", tmpDir, "cursor"))

	sessions, err = persistence.ListSessions()
	if err != nil {
		t.Fatalf("ListSessions failed: %v", err)
	}
	if len(sessions) != 2 {
		t.Errorf("Expected 2 sessions, got %d", len(sessions))
	}
}

func TestStatePersistence_Delete(t *testing.T) {
	tmpDir := t.TempDir()
	persistence := NewStatePersistence(tmpDir)

	ctx := NewLoopContext("session-to-delete", tmpDir, "auggie")
	persistence.Save(ctx)

	// Verify it exists
	_, err := persistence.Load("session-to-delete")
	if err != nil {
		t.Fatalf("Session should exist before delete: %v", err)
	}

	// Delete
	err = persistence.Delete("session-to-delete")
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	// Verify it's gone
	_, err = persistence.Load("session-to-delete")
	if err == nil {
		t.Error("Session should not exist after delete")
	}

	// Delete nonexistent should not error
	err = persistence.Delete("nonexistent")
	if err != nil {
		t.Errorf("Delete nonexistent should not error: %v", err)
	}
}

func TestTransitionError_Error(t *testing.T) {
	err := &TransitionError{From: StateIdle, To: StateCompleted}
	expected := "invalid state transition from idle to completed"
	if err.Error() != expected {
		t.Errorf("Error() = %q, want %q", err.Error(), expected)
	}
}
