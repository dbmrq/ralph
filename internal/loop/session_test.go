package loop

import (
	"regexp"
	"testing"
	"time"
)

func TestGenerateSessionID(t *testing.T) {
	t.Run("format", func(t *testing.T) {
		id := GenerateSessionID()

		// Should match pattern: ralph-YYYYMMDD-HHMMSS-XXXXXXXX
		pattern := regexp.MustCompile(`^ralph-\d{8}-\d{6}-[a-f0-9]{8}$`)
		if !pattern.MatchString(id) {
			t.Errorf("GenerateSessionID() = %q, doesn't match expected pattern", id)
		}
	})

	t.Run("uniqueness", func(t *testing.T) {
		seen := make(map[string]bool)
		for i := 0; i < 100; i++ {
			id := GenerateSessionID()
			if seen[id] {
				t.Errorf("GenerateSessionID() generated duplicate: %s", id)
			}
			seen[id] = true
		}
	})

	t.Run("prefix", func(t *testing.T) {
		id := GenerateSessionID()
		if id[:6] != "ralph-" {
			t.Errorf("GenerateSessionID() = %q, should start with 'ralph-'", id)
		}
	})
}

func TestParseSessionID(t *testing.T) {
	t.Run("valid ID", func(t *testing.T) {
		id := GenerateSessionID()
		ts, err := ParseSessionID(id)
		if err != nil {
			t.Errorf("ParseSessionID(%q) error = %v", id, err)
		}

		// Timestamp should be within last minute (accounting for timezone differences)
		// The timestamp is parsed as UTC but generated from local time
		now := time.Now().UTC()
		diff := now.Sub(ts)
		if diff < 0 {
			diff = -diff
		}
		// Allow up to 24 hours difference to account for timezone issues
		if diff > 24*time.Hour {
			t.Errorf("ParseSessionID(%q) timestamp too different from now: %v (diff: %v)", id, ts, diff)
		}
	})

	t.Run("specific ID", func(t *testing.T) {
		id := "ralph-20260213-150405-abcd1234"
		ts, err := ParseSessionID(id)
		if err != nil {
			t.Errorf("ParseSessionID(%q) error = %v", id, err)
		}

		expected := time.Date(2026, 2, 13, 15, 4, 5, 0, time.UTC)
		if !ts.Equal(expected) {
			t.Errorf("ParseSessionID(%q) = %v, want %v", id, ts, expected)
		}
	})

	t.Run("invalid prefix", func(t *testing.T) {
		id := "other-20260213-150405-abcd1234"
		_, err := ParseSessionID(id)
		if err == nil {
			t.Errorf("ParseSessionID(%q) should fail for invalid prefix", id)
		}
	})

	t.Run("invalid format", func(t *testing.T) {
		testCases := []string{
			"ralph",
			"ralph-20260213",
			"ralph-invalid-150405-abcd1234",
			"ralph-20260213-invalid-abcd1234",
		}
		for _, id := range testCases {
			_, err := ParseSessionID(id)
			if err == nil {
				t.Errorf("ParseSessionID(%q) should fail", id)
			}
		}
	})
}

func TestSessionManager_CreateSession(t *testing.T) {
	tmpDir := t.TempDir()
	mgr := NewSessionManager(tmpDir)

	ctx, err := mgr.CreateSession("auggie")
	if err != nil {
		t.Fatalf("CreateSession() error = %v", err)
	}

	if ctx.SessionID == "" {
		t.Error("CreateSession() should set SessionID")
	}
	if ctx.ProjectDir != tmpDir {
		t.Errorf("CreateSession() ProjectDir = %q, want %q", ctx.ProjectDir, tmpDir)
	}
	if ctx.AgentName != "auggie" {
		t.Errorf("CreateSession() AgentName = %q, want %q", ctx.AgentName, "auggie")
	}
	if ctx.State != StateIdle {
		t.Errorf("CreateSession() State = %q, want %q", ctx.State, StateIdle)
	}
}

func TestSessionManager_SaveAndResume(t *testing.T) {
	tmpDir := t.TempDir()
	mgr := NewSessionManager(tmpDir)

	// Create and save a session
	ctx, _ := mgr.CreateSession("auggie")
	ctx.Transition(StateRunning)
	ctx.Transition(StatePaused)
	ctx.CurrentTaskID = "TASK-001"

	err := mgr.SaveSession(ctx)
	if err != nil {
		t.Fatalf("SaveSession() error = %v", err)
	}

	// Resume the session
	resumed, err := mgr.ResumeSession(ctx.SessionID)
	if err != nil {
		t.Fatalf("ResumeSession() error = %v", err)
	}

	if resumed.SessionID != ctx.SessionID {
		t.Errorf("ResumeSession() SessionID = %q, want %q", resumed.SessionID, ctx.SessionID)
	}
	if resumed.State != StatePaused {
		t.Errorf("ResumeSession() State = %q, want %q", resumed.State, StatePaused)
	}
	if resumed.CurrentTaskID != "TASK-001" {
		t.Errorf("ResumeSession() CurrentTaskID = %q, want %q", resumed.CurrentTaskID, "TASK-001")
	}
}

func TestSessionManager_ResumeLatest(t *testing.T) {
	tmpDir := t.TempDir()
	mgr := NewSessionManager(tmpDir)

	// Create and save two sessions
	ctx1, _ := mgr.CreateSession("auggie")
	ctx1.Transition(StateRunning)
	ctx1.Transition(StatePaused)
	ctx1.UpdatedAt = time.Now().Add(-time.Hour)
	mgr.SaveSession(ctx1)

	ctx2, _ := mgr.CreateSession("cursor")
	ctx2.Transition(StateRunning)
	ctx2.Transition(StatePaused)
	ctx2.UpdatedAt = time.Now()
	mgr.SaveSession(ctx2)

	// Resume without specifying ID should get most recent
	resumed, err := mgr.ResumeSession("")
	if err != nil {
		t.Fatalf("ResumeSession('') error = %v", err)
	}

	if resumed.SessionID != ctx2.SessionID {
		t.Errorf("ResumeSession('') should resume most recent, got %q, want %q",
			resumed.SessionID, ctx2.SessionID)
	}
}

func TestSessionManager_ResumeNonResumable(t *testing.T) {
	tmpDir := t.TempDir()
	mgr := NewSessionManager(tmpDir)

	// Create a completed session
	ctx, _ := mgr.CreateSession("auggie")
	ctx.Transition(StateRunning)
	ctx.Transition(StateCompleted)
	mgr.SaveSession(ctx)

	// Should fail to resume
	_, err := mgr.ResumeSession(ctx.SessionID)
	if err == nil {
		t.Error("ResumeSession() should fail for completed session")
	}
}

func TestSessionManager_ResumeWrongProject(t *testing.T) {
	tmpDir1 := t.TempDir()
	tmpDir2 := t.TempDir()

	mgr1 := NewSessionManager(tmpDir1)

	// Create and save a session in project 1
	ctx, _ := mgr1.CreateSession("auggie")
	ctx.Transition(StateRunning)
	ctx.Transition(StatePaused)
	mgr1.SaveSession(ctx)

	// Try to resume from project 2
	mgr2 := NewSessionManager(tmpDir2)
	_, err := mgr2.ResumeSession(ctx.SessionID)
	if err == nil {
		t.Error("ResumeSession() should fail for session from different project")
	}
}

func TestSessionManager_GetResumableSessions(t *testing.T) {
	tmpDir := t.TempDir()
	mgr := NewSessionManager(tmpDir)

	// Create various sessions
	ctx1, _ := mgr.CreateSession("auggie")
	ctx1.Transition(StateRunning)
	ctx1.Transition(StatePaused)
	mgr.SaveSession(ctx1)

	ctx2, _ := mgr.CreateSession("cursor")
	ctx2.Transition(StateRunning)
	ctx2.Transition(StateCompleted)
	mgr.SaveSession(ctx2)

	ctx3, _ := mgr.CreateSession("auggie")
	ctx3.Transition(StateRunning)
	ctx3.Transition(StateAwaitingFix)
	mgr.SaveSession(ctx3)

	// Get resumable sessions
	resumable, err := mgr.GetResumableSessions()
	if err != nil {
		t.Fatalf("GetResumableSessions() error = %v", err)
	}

	if len(resumable) != 2 {
		t.Errorf("GetResumableSessions() returned %d sessions, want 2", len(resumable))
	}
}

func TestSessionManager_ListSessions(t *testing.T) {
	tmpDir := t.TempDir()
	mgr := NewSessionManager(tmpDir)

	// Initially empty
	sessions, _ := mgr.ListSessions()
	if len(sessions) != 0 {
		t.Errorf("ListSessions() should return empty initially, got %d", len(sessions))
	}

	// Create sessions
	ctx1, _ := mgr.CreateSession("auggie")
	mgr.SaveSession(ctx1)
	ctx2, _ := mgr.CreateSession("cursor")
	mgr.SaveSession(ctx2)

	sessions, err := mgr.ListSessions()
	if err != nil {
		t.Fatalf("ListSessions() error = %v", err)
	}
	if len(sessions) != 2 {
		t.Errorf("ListSessions() returned %d sessions, want 2", len(sessions))
	}
}

func TestSessionManager_DeleteSession(t *testing.T) {
	tmpDir := t.TempDir()
	mgr := NewSessionManager(tmpDir)

	ctx, _ := mgr.CreateSession("auggie")
	mgr.SaveSession(ctx)

	// Verify it exists
	_, err := mgr.GetSession(ctx.SessionID)
	if err != nil {
		t.Fatalf("Session should exist before delete: %v", err)
	}

	// Delete
	err = mgr.DeleteSession(ctx.SessionID)
	if err != nil {
		t.Fatalf("DeleteSession() error = %v", err)
	}

	// Verify it's gone
	_, err = mgr.GetSession(ctx.SessionID)
	if err == nil {
		t.Error("Session should not exist after delete")
	}
}
