// Package loop provides the main execution loop for ralph.
// This file implements LOOP-005: session management including unique session ID
// generation, session state persistence, and session resumption support.
package loop

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
	"time"
)

// SessionIDPrefix is the prefix for session IDs.
const SessionIDPrefix = "ralph"

// SessionIDLength is the number of random bytes in a session ID.
const SessionIDLength = 8

// GenerateSessionID creates a unique session ID.
// Format: ralph-YYYYMMDD-HHMMSS-XXXXXXXX
// Where XXXXXXXX is 8 random hex characters.
func GenerateSessionID() string {
	now := time.Now()
	datePart := now.Format("20060102-150405")

	// Generate random suffix
	randomBytes := make([]byte, SessionIDLength/2)
	if _, err := rand.Read(randomBytes); err != nil {
		// Fallback to timestamp-based suffix if crypto/rand fails
		return fmt.Sprintf("%s-%s-%d", SessionIDPrefix, datePart, now.UnixNano()%100000000)
	}

	randomHex := hex.EncodeToString(randomBytes)
	return fmt.Sprintf("%s-%s-%s", SessionIDPrefix, datePart, randomHex)
}

// ParseSessionID parses a session ID and validates its format.
// Returns the timestamp portion if valid.
func ParseSessionID(sessionID string) (time.Time, error) {
	parts := strings.Split(sessionID, "-")
	if len(parts) < 4 {
		return time.Time{}, fmt.Errorf("invalid session ID format: %s", sessionID)
	}

	if parts[0] != SessionIDPrefix {
		return time.Time{}, fmt.Errorf("invalid session ID prefix: %s", sessionID)
	}

	// Parse date and time parts
	dateStr := parts[1] + parts[2]
	t, err := time.Parse("20060102150405", dateStr)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid session ID timestamp: %s", sessionID)
	}

	return t, nil
}

// SessionManager manages session lifecycle including creation, resumption, and continuation.
type SessionManager struct {
	persistence *StatePersistence
	projectDir  string
}

// NewSessionManager creates a new session manager.
func NewSessionManager(projectDir string) *SessionManager {
	return &SessionManager{
		persistence: NewStatePersistence(projectDir),
		projectDir:  projectDir,
	}
}

// CreateSession creates a new session with a unique ID.
func (m *SessionManager) CreateSession(agentName string) (*LoopContext, error) {
	sessionID := GenerateSessionID()
	ctx := NewLoopContext(sessionID, m.projectDir, agentName)
	return ctx, nil
}

// ResumeSession loads a session by ID and validates it can be resumed.
// If sessionID is empty, loads the most recent resumable session.
func (m *SessionManager) ResumeSession(sessionID string) (*LoopContext, error) {
	var ctx *LoopContext
	var err error

	if sessionID == "" {
		// Load most recent resumable session
		ctx, err = m.persistence.LoadResumable()
		if err != nil {
			return nil, fmt.Errorf("no resumable session found: %w", err)
		}
	} else {
		// Load specific session
		ctx, err = m.persistence.Load(sessionID)
		if err != nil {
			return nil, fmt.Errorf("session not found: %w", err)
		}
	}

	// Validate session can be resumed
	if !ctx.State.CanResume() {
		return nil, fmt.Errorf("session %s cannot be resumed (state: %s)", ctx.SessionID, ctx.State)
	}

	// Validate project directory matches
	if ctx.ProjectDir != m.projectDir {
		return nil, fmt.Errorf("session %s is for different project: %s (current: %s)",
			ctx.SessionID, ctx.ProjectDir, m.projectDir)
	}

	return ctx, nil
}

// SaveSession persists the session state.
func (m *SessionManager) SaveSession(ctx *LoopContext) error {
	return m.persistence.Save(ctx)
}

// GetSession loads a session by ID without validation.
func (m *SessionManager) GetSession(sessionID string) (*LoopContext, error) {
	return m.persistence.Load(sessionID)
}

// ListSessions returns all session IDs.
func (m *SessionManager) ListSessions() ([]string, error) {
	return m.persistence.ListSessions()
}

// GetResumableSessions returns all sessions that can be resumed.
func (m *SessionManager) GetResumableSessions() ([]*LoopContext, error) {
	sessions, err := m.persistence.ListSessions()
	if err != nil {
		return nil, err
	}

	var resumable []*LoopContext
	for _, id := range sessions {
		ctx, err := m.persistence.Load(id)
		if err != nil {
			continue
		}
		if ctx.State.CanResume() && ctx.ProjectDir == m.projectDir {
			resumable = append(resumable, ctx)
		}
	}

	return resumable, nil
}

// DeleteSession removes a session.
func (m *SessionManager) DeleteSession(sessionID string) error {
	return m.persistence.Delete(sessionID)
}

