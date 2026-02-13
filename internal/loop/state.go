// Package loop provides the main execution loop for ralph.
// This package manages the state machine, session persistence, and orchestration
// of task execution through agents.
package loop

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/wexinc/ralph/internal/task"
)

// State represents the current state of the Ralph loop.
type State string

const (
	// StateIdle indicates the loop is not running and ready to start.
	StateIdle State = "idle"
	// StateRunning indicates the loop is actively executing tasks.
	StateRunning State = "running"
	// StatePaused indicates the loop was paused by user request.
	StatePaused State = "paused"
	// StateAwaitingFix indicates the loop is waiting for a fix attempt.
	StateAwaitingFix State = "awaiting_fix"
	// StateCompleted indicates all tasks were completed successfully.
	StateCompleted State = "completed"
	// StateFailed indicates the loop failed and cannot continue.
	StateFailed State = "failed"
)

// IsValid returns true if the state is a known valid state.
func (s State) IsValid() bool {
	switch s {
	case StateIdle, StateRunning, StatePaused, StateAwaitingFix, StateCompleted, StateFailed:
		return true
	default:
		return false
	}
}

// IsTerminal returns true if the state represents a terminal state.
func (s State) IsTerminal() bool {
	return s == StateCompleted || s == StateFailed
}

// CanResume returns true if the loop can be resumed from this state.
func (s State) CanResume() bool {
	return s == StatePaused || s == StateAwaitingFix
}

// String returns the string representation of the state.
func (s State) String() string {
	return string(s)
}

// ValidTransitions maps each state to its valid next states.
var ValidTransitions = map[State][]State{
	StateIdle:        {StateRunning},
	StateRunning:     {StatePaused, StateAwaitingFix, StateCompleted, StateFailed},
	StatePaused:      {StateRunning, StateFailed},
	StateAwaitingFix: {StateRunning, StateFailed},
	StateCompleted:   {}, // Terminal state
	StateFailed:      {StateIdle}, // Can restart from failed
}

// CanTransitionTo returns true if the transition from current state to next is valid.
func (s State) CanTransitionTo(next State) bool {
	valid, ok := ValidTransitions[s]
	if !ok {
		return false
	}
	for _, v := range valid {
		if v == next {
			return true
		}
	}
	return false
}

// TransitionError represents an invalid state transition.
type TransitionError struct {
	From State
	To   State
}

func (e *TransitionError) Error() string {
	return fmt.Sprintf("invalid state transition from %s to %s", e.From, e.To)
}

// LoopContext contains the current context of the loop execution.
type LoopContext struct {
	// State is the current loop state.
	State State `json:"state"`
	// SessionID is the unique identifier for this session.
	SessionID string `json:"session_id"`
	// CurrentTaskID is the ID of the task currently being executed.
	CurrentTaskID string `json:"current_task_id,omitempty"`
	// CurrentIteration is the iteration number for the current task.
	CurrentIteration int `json:"current_iteration"`
	// TotalIterations is the total number of iterations in this session.
	TotalIterations int `json:"total_iterations"`
	// TasksCompleted is the number of tasks completed in this session.
	TasksCompleted int `json:"tasks_completed"`
	// TasksFailed is the number of tasks that failed in this session.
	TasksFailed int `json:"tasks_failed"`
	// TasksSkipped is the number of tasks skipped in this session.
	TasksSkipped int `json:"tasks_skipped"`
	// FixAttempts is the number of fix attempts for the current issue.
	FixAttempts int `json:"fix_attempts"`
	// MaxFixAttempts is the maximum number of fix attempts allowed.
	MaxFixAttempts int `json:"max_fix_attempts"`
	// StartedAt is when the session started.
	StartedAt time.Time `json:"started_at"`
	// UpdatedAt is when the context was last updated.
	UpdatedAt time.Time `json:"updated_at"`
	// PausedAt is when the session was paused (zero if not paused).
	PausedAt time.Time `json:"paused_at,omitempty"`
	// LastError is the most recent error message, if any.
	LastError string `json:"last_error,omitempty"`
	// AgentSessionID is the session ID from the agent (for --continue).
	AgentSessionID string `json:"agent_session_id,omitempty"`
	// ProjectDir is the project directory for this session.
	ProjectDir string `json:"project_dir"`
	// AgentName is the name of the agent being used.
	AgentName string `json:"agent_name"`
	// ModelName is the name of the model being used.
	ModelName string `json:"model_name,omitempty"`
}

// NewLoopContext creates a new loop context with the given session ID.
func NewLoopContext(sessionID, projectDir, agentName string) *LoopContext {
	now := time.Now()
	return &LoopContext{
		State:          StateIdle,
		SessionID:      sessionID,
		MaxFixAttempts: 3, // Default max fix attempts
		StartedAt:      now,
		UpdatedAt:      now,
		ProjectDir:     projectDir,
		AgentName:      agentName,
	}
}

// Transition attempts to change the loop state.
// Returns an error if the transition is invalid.
func (c *LoopContext) Transition(to State) error {
	if !c.State.CanTransitionTo(to) {
		return &TransitionError{From: c.State, To: to}
	}

	c.State = to
	c.UpdatedAt = time.Now()

	// Handle state-specific logic
	switch to {
	case StatePaused:
		c.PausedAt = c.UpdatedAt
	case StateRunning:
		// Clear pause time when resuming
		c.PausedAt = time.Time{}
	case StateAwaitingFix:
		c.FixAttempts++
	}

	return nil
}

// SetCurrentTask updates the current task being executed.
func (c *LoopContext) SetCurrentTask(taskID string) {
	c.CurrentTaskID = taskID
	c.CurrentIteration = 0
	c.FixAttempts = 0
	c.UpdatedAt = time.Now()
}

// IncrementIteration increments the iteration counter.
func (c *LoopContext) IncrementIteration() {
	c.CurrentIteration++
	c.TotalIterations++
	c.UpdatedAt = time.Now()
}

// RecordTaskCompletion records that a task was completed.
func (c *LoopContext) RecordTaskCompletion(status task.TaskStatus) {
	switch status {
	case task.StatusCompleted:
		c.TasksCompleted++
	case task.StatusFailed:
		c.TasksFailed++
	case task.StatusSkipped:
		c.TasksSkipped++
	}
	c.CurrentTaskID = ""
	c.CurrentIteration = 0
	c.FixAttempts = 0
	c.UpdatedAt = time.Now()
}

// SetError records an error in the context.
func (c *LoopContext) SetError(err string) {
	c.LastError = err
	c.UpdatedAt = time.Now()
}

// ClearError clears the last error.
func (c *LoopContext) ClearError() {
	c.LastError = ""
	c.UpdatedAt = time.Now()
}

// SetAgentSession records the agent session ID for resume.
func (c *LoopContext) SetAgentSession(sessionID string) {
	c.AgentSessionID = sessionID
	c.UpdatedAt = time.Now()
}

// Duration returns the total duration of the session.
func (c *LoopContext) Duration() time.Duration {
	if c.State.IsTerminal() || c.State == StatePaused {
		return c.UpdatedAt.Sub(c.StartedAt)
	}
	return time.Since(c.StartedAt)
}

// CanAttemptFix returns true if more fix attempts are allowed.
func (c *LoopContext) CanAttemptFix() bool {
	return c.FixAttempts < c.MaxFixAttempts
}

// Clone creates a deep copy of the loop context.
func (c *LoopContext) Clone() *LoopContext {
	clone := *c
	return &clone
}

// StatePersistence manages saving and loading loop state.
type StatePersistence struct {
	// SessionsDir is the directory where session files are stored.
	SessionsDir string
}

// DefaultSessionsDir is the default directory for session files.
const DefaultSessionsDir = ".ralph/sessions"

// NewStatePersistence creates a new state persistence manager.
func NewStatePersistence(projectDir string) *StatePersistence {
	return &StatePersistence{
		SessionsDir: filepath.Join(projectDir, DefaultSessionsDir),
	}
}

// sessionPath returns the path to the session file for the given session ID.
func (p *StatePersistence) sessionPath(sessionID string) string {
	return filepath.Join(p.SessionsDir, sessionID+".json")
}

// Save persists the loop context to disk.
func (p *StatePersistence) Save(ctx *LoopContext) error {
	// Ensure sessions directory exists
	if err := os.MkdirAll(p.SessionsDir, 0755); err != nil {
		return fmt.Errorf("failed to create sessions directory: %w", err)
	}

	data, err := json.MarshalIndent(ctx, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal loop context: %w", err)
	}

	path := p.sessionPath(ctx.SessionID)
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write session file: %w", err)
	}

	return nil
}

// Load loads a loop context from disk by session ID.
func (p *StatePersistence) Load(sessionID string) (*LoopContext, error) {
	path := p.sessionPath(sessionID)

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("session not found: %s", sessionID)
		}
		return nil, fmt.Errorf("failed to read session file: %w", err)
	}

	var ctx LoopContext
	if err := json.Unmarshal(data, &ctx); err != nil {
		return nil, fmt.Errorf("failed to parse session file: %w", err)
	}

	return &ctx, nil
}

// LoadLatest loads the most recently updated session.
func (p *StatePersistence) LoadLatest() (*LoopContext, error) {
	entries, err := os.ReadDir(p.SessionsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("no sessions directory found")
		}
		return nil, fmt.Errorf("failed to read sessions directory: %w", err)
	}

	var latestCtx *LoopContext
	var latestTime time.Time

	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}

		sessionID := entry.Name()[:len(entry.Name())-5] // Remove .json
		ctx, err := p.Load(sessionID)
		if err != nil {
			continue // Skip invalid sessions
		}

		if ctx.UpdatedAt.After(latestTime) {
			latestTime = ctx.UpdatedAt
			latestCtx = ctx
		}
	}

	if latestCtx == nil {
		return nil, fmt.Errorf("no valid sessions found")
	}

	return latestCtx, nil
}

// LoadResumable loads the most recent resumable session (paused or awaiting_fix).
func (p *StatePersistence) LoadResumable() (*LoopContext, error) {
	entries, err := os.ReadDir(p.SessionsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("no sessions directory found")
		}
		return nil, fmt.Errorf("failed to read sessions directory: %w", err)
	}

	var latestCtx *LoopContext
	var latestTime time.Time

	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}

		sessionID := entry.Name()[:len(entry.Name())-5] // Remove .json
		ctx, err := p.Load(sessionID)
		if err != nil {
			continue // Skip invalid sessions
		}

		if !ctx.State.CanResume() {
			continue // Only resumable sessions
		}

		if ctx.UpdatedAt.After(latestTime) {
			latestTime = ctx.UpdatedAt
			latestCtx = ctx
		}
	}

	if latestCtx == nil {
		return nil, fmt.Errorf("no resumable sessions found")
	}

	return latestCtx, nil
}

// ListSessions returns a list of all session IDs.
func (p *StatePersistence) ListSessions() ([]string, error) {
	entries, err := os.ReadDir(p.SessionsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, fmt.Errorf("failed to read sessions directory: %w", err)
	}

	var sessions []string
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		sessionID := entry.Name()[:len(entry.Name())-5] // Remove .json
		sessions = append(sessions, sessionID)
	}

	return sessions, nil
}

// Delete removes a session file.
func (p *StatePersistence) Delete(sessionID string) error {
	path := p.sessionPath(sessionID)
	if err := os.Remove(path); err != nil {
		if os.IsNotExist(err) {
			return nil // Already deleted
		}
		return fmt.Errorf("failed to delete session: %w", err)
	}
	return nil
}

