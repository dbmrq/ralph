// Package task provides task data model and management for ralph.
package task

import (
	"fmt"
	"time"
)

// TaskStatus represents the current state of a task.
type TaskStatus string

const (
	// StatusPending indicates the task has not been started.
	StatusPending TaskStatus = "pending"
	// StatusInProgress indicates the task is currently being worked on.
	StatusInProgress TaskStatus = "in_progress"
	// StatusCompleted indicates the task was successfully completed.
	StatusCompleted TaskStatus = "completed"
	// StatusSkipped indicates the task was skipped by user or system.
	StatusSkipped TaskStatus = "skipped"
	// StatusPaused indicates the task was paused and can be resumed.
	StatusPaused TaskStatus = "paused"
	// StatusFailed indicates the task failed after all retry attempts.
	StatusFailed TaskStatus = "failed"
)

// IsValid returns true if the status is a known valid status.
func (s TaskStatus) IsValid() bool {
	switch s {
	case StatusPending, StatusInProgress, StatusCompleted, StatusSkipped, StatusPaused, StatusFailed:
		return true
	default:
		return false
	}
}

// IsTerminal returns true if the status represents a terminal state (completed, skipped, or failed).
func (s TaskStatus) IsTerminal() bool {
	switch s {
	case StatusCompleted, StatusSkipped, StatusFailed:
		return true
	default:
		return false
	}
}

// IsPending returns true if the task has not yet started or is paused.
func (s TaskStatus) IsPending() bool {
	return s == StatusPending || s == StatusPaused
}

// String returns the string representation of the status.
func (s TaskStatus) String() string {
	return string(s)
}

// Iteration represents a single attempt at completing a task.
type Iteration struct {
	// Number is the iteration number (1-indexed).
	Number int `json:"number"`
	// StartedAt is when this iteration began.
	StartedAt time.Time `json:"started_at"`
	// EndedAt is when this iteration ended (zero if still running).
	EndedAt time.Time `json:"ended_at,omitempty"`
	// Result is the outcome of this iteration (NEXT, DONE, ERROR, FIXED).
	Result string `json:"result,omitempty"`
	// AgentOutput is a summary or last portion of agent output.
	AgentOutput string `json:"agent_output,omitempty"`
	// SessionID is the agent session ID for this iteration (for --continue).
	SessionID string `json:"session_id,omitempty"`
}

// Duration returns the duration of the iteration.
// If the iteration is still running, returns duration until now.
func (i Iteration) Duration() time.Duration {
	if i.EndedAt.IsZero() {
		return time.Since(i.StartedAt)
	}
	return i.EndedAt.Sub(i.StartedAt)
}

// IsComplete returns true if the iteration has ended.
func (i Iteration) IsComplete() bool {
	return !i.EndedAt.IsZero()
}

// Task represents a single task in the task list.
type Task struct {
	// ID is the unique identifier for the task (e.g., "TASK-001").
	ID string `json:"id"`
	// Name is a short description of the task.
	Name string `json:"name"`
	// Description is the full description including context and requirements.
	Description string `json:"description"`
	// Status is the current state of the task.
	Status TaskStatus `json:"status"`
	// Order is the execution order (lower numbers execute first).
	Order int `json:"order"`
	// SessionID is the current or most recent agent session ID (for pause/resume).
	SessionID string `json:"session_id,omitempty"`
	// Iterations is the history of all attempts on this task.
	Iterations []Iteration `json:"iterations,omitempty"`
	// Metadata stores additional key-value data about the task.
	Metadata map[string]string `json:"metadata,omitempty"`
	// CreatedAt is when the task was created.
	CreatedAt time.Time `json:"created_at"`
	// UpdatedAt is when the task was last updated.
	UpdatedAt time.Time `json:"updated_at"`
	// CompletedAt is when the task was completed (zero if not completed).
	CompletedAt time.Time `json:"completed_at,omitempty"`
}

// NewTask creates a new task with the given ID, name, and description.
// The task is initialized with pending status and current timestamps.
func NewTask(id, name, description string) *Task {
	now := time.Now()
	return &Task{
		ID:          id,
		Name:        name,
		Description: description,
		Status:      StatusPending,
		Order:       0,
		Iterations:  []Iteration{},
		Metadata:    make(map[string]string),
		CreatedAt:   now,
		UpdatedAt:   now,
	}
}

// IterationCount returns the number of iterations attempted on this task.
func (t *Task) IterationCount() int {
	return len(t.Iterations)
}

// CurrentIteration returns the current (most recent) iteration, or nil if none.
func (t *Task) CurrentIteration() *Iteration {
	if len(t.Iterations) == 0 {
		return nil
	}
	return &t.Iterations[len(t.Iterations)-1]
}

// StartIteration starts a new iteration and returns it.
func (t *Task) StartIteration() *Iteration {
	t.Status = StatusInProgress
	t.UpdatedAt = time.Now()
	iteration := Iteration{
		Number:    len(t.Iterations) + 1,
		StartedAt: t.UpdatedAt,
	}
	t.Iterations = append(t.Iterations, iteration)
	return &t.Iterations[len(t.Iterations)-1]
}

// EndIteration ends the current iteration with the given result.
// Returns an error if there is no current iteration.
func (t *Task) EndIteration(result, output, sessionID string) error {
	current := t.CurrentIteration()
	if current == nil {
		return fmt.Errorf("no active iteration to end")
	}
	if current.IsComplete() {
		return fmt.Errorf("current iteration is already complete")
	}

	now := time.Now()
	current.EndedAt = now
	current.Result = result
	current.AgentOutput = output
	if sessionID != "" {
		current.SessionID = sessionID
	}
	t.SessionID = sessionID
	t.UpdatedAt = now

	return nil
}

// MarkCompleted marks the task as completed.
func (t *Task) MarkCompleted() {
	now := time.Now()
	t.Status = StatusCompleted
	t.CompletedAt = now
	t.UpdatedAt = now
}

// MarkFailed marks the task as failed.
func (t *Task) MarkFailed() {
	t.Status = StatusFailed
	t.UpdatedAt = time.Now()
}

// MarkSkipped marks the task as skipped.
func (t *Task) MarkSkipped() {
	t.Status = StatusSkipped
	t.UpdatedAt = time.Now()
}

// MarkPaused marks the task as paused for later resumption.
func (t *Task) MarkPaused() {
	t.Status = StatusPaused
	t.UpdatedAt = time.Now()
}

// Resume resumes a paused task by starting a new iteration.
func (t *Task) Resume() *Iteration {
	if t.Status == StatusPaused {
		return t.StartIteration()
	}
	return nil
}

// SetMetadata sets a metadata key-value pair.
func (t *Task) SetMetadata(key, value string) {
	if t.Metadata == nil {
		t.Metadata = make(map[string]string)
	}
	t.Metadata[key] = value
	t.UpdatedAt = time.Now()
}

// GetMetadata retrieves a metadata value by key.
// Returns empty string and false if the key doesn't exist.
func (t *Task) GetMetadata(key string) (string, bool) {
	if t.Metadata == nil {
		return "", false
	}
	v, ok := t.Metadata[key]
	return v, ok
}

// TotalDuration returns the total time spent on all iterations.
func (t *Task) TotalDuration() time.Duration {
	var total time.Duration
	for _, iter := range t.Iterations {
		total += iter.Duration()
	}
	return total
}

// Validate validates the task data and returns an error if invalid.
func (t *Task) Validate() error {
	if t.ID == "" {
		return fmt.Errorf("task ID is required")
	}
	if t.Name == "" {
		return fmt.Errorf("task name is required")
	}
	if !t.Status.IsValid() {
		return fmt.Errorf("invalid task status: %s", t.Status)
	}
	return nil
}

// Clone returns a deep copy of the task.
func (t *Task) Clone() *Task {
	clone := &Task{
		ID:          t.ID,
		Name:        t.Name,
		Description: t.Description,
		Status:      t.Status,
		Order:       t.Order,
		SessionID:   t.SessionID,
		CreatedAt:   t.CreatedAt,
		UpdatedAt:   t.UpdatedAt,
		CompletedAt: t.CompletedAt,
	}

	// Deep copy iterations
	if t.Iterations != nil {
		clone.Iterations = make([]Iteration, len(t.Iterations))
		copy(clone.Iterations, t.Iterations)
	}

	// Deep copy metadata
	if t.Metadata != nil {
		clone.Metadata = make(map[string]string, len(t.Metadata))
		for k, v := range t.Metadata {
			clone.Metadata[k] = v
		}
	}

	return clone
}
