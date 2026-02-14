// Package agent provides the agent plugin system for ralph.
// Agents are AI coding assistants (like Cursor, Auggie) that execute tasks.
package agent

import (
	"context"
	"io"
	"time"
)

// Model represents an AI model available through an agent.
type Model struct {
	// ID is the unique identifier for the model (e.g., "claude-opus-4").
	ID string `json:"id"`
	// Name is the human-readable name of the model.
	Name string `json:"name"`
	// Description is an optional description of the model's capabilities.
	Description string `json:"description,omitempty"`
	// IsDefault indicates if this is the agent's default model.
	IsDefault bool `json:"is_default,omitempty"`
}

// String returns a string representation of the model.
func (m Model) String() string {
	if m.Name != "" {
		return m.Name
	}
	return m.ID
}

// RunOptions configures how an agent executes a prompt.
type RunOptions struct {
	// Model is the model to use. Empty string uses agent's default.
	Model string
	// WorkDir is the working directory for the agent command.
	WorkDir string
	// LogPath is the path to write agent output logs.
	LogPath string
	// LogWriter is a writer for real-time output streaming.
	// Can be used to stream output to TUI while also capturing to log file.
	LogWriter io.Writer
	// Timeout is the maximum time to wait for the agent to complete.
	// Uses smart timeout system (default 2h if active, 30min if stuck).
	Timeout time.Duration
	// Force forces the agent to run without confirmation.
	Force bool
	// SessionID is the session ID for continuing a previous session.
	SessionID string
}

// TaskStatus represents the status reported by an agent.
type TaskStatus string

const (
	// TaskStatusNext indicates the task is in progress, more work needed.
	TaskStatusNext TaskStatus = "NEXT"
	// TaskStatusDone indicates the task was completed successfully.
	TaskStatusDone TaskStatus = "DONE"
	// TaskStatusError indicates the task encountered an error.
	TaskStatusError TaskStatus = "ERROR"
	// TaskStatusFixed indicates a previous error was fixed.
	TaskStatusFixed TaskStatus = "FIXED"
	// TaskStatusUnknown indicates the status could not be determined.
	TaskStatusUnknown TaskStatus = "UNKNOWN"
)

// String returns the string representation of the task status.
func (s TaskStatus) String() string {
	return string(s)
}

// IsTerminal returns true if the status represents a terminal state.
func (s TaskStatus) IsTerminal() bool {
	return s == TaskStatusDone || s == TaskStatusError
}

// IsSuccess returns true if the status indicates success (DONE or FIXED).
func (s TaskStatus) IsSuccess() bool {
	return s == TaskStatusDone || s == TaskStatusFixed
}

// Result represents the outcome of an agent execution.
type Result struct {
	// Output is the full output from the agent.
	Output string `json:"output"`
	// ExitCode is the exit code from the agent process.
	ExitCode int `json:"exit_code"`
	// Duration is how long the agent ran.
	Duration time.Duration `json:"duration"`
	// Status is the task status extracted from agent output (NEXT, DONE, ERROR, FIXED).
	Status TaskStatus `json:"status"`
	// SessionID is the agent session ID for continuing this work later.
	SessionID string `json:"session_id,omitempty"`
	// Error contains any error message if the agent failed.
	Error string `json:"error,omitempty"`
}

// IsSuccess returns true if the agent run completed successfully.
func (r Result) IsSuccess() bool {
	return r.ExitCode == 0 && r.Status.IsSuccess()
}

// Agent defines the interface that all agent plugins must implement.
type Agent interface {
	// Name returns the unique identifier for this agent (e.g., "cursor", "auggie").
	Name() string
	// Description returns a human-readable description of the agent.
	Description() string

	// IsAvailable checks if this agent is installed and available on the system.
	IsAvailable() bool
	// CheckAuth verifies that authentication is configured for this agent.
	// Returns an error with instructions if auth is not set up.
	CheckAuth() error

	// ListModels returns all available models for this agent.
	ListModels() ([]Model, error)
	// GetDefaultModel returns the default model for this agent.
	GetDefaultModel() Model

	// Run executes a prompt and returns the result.
	Run(ctx context.Context, prompt string, opts RunOptions) (Result, error)

	// Continue resumes a previous session with a new prompt.
	// This is used for pause/resume functionality.
	Continue(ctx context.Context, sessionID string, prompt string, opts RunOptions) (Result, error)

	// GetSessionID returns the session ID from the most recent run, if any.
	GetSessionID() string
}
