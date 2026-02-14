// Package tui provides the terminal user interface for ralph.
package tui

import (
	"time"

	"github.com/dbmrq/ralph/internal/task"
	"github.com/dbmrq/ralph/internal/tui/components"
)

// Message types for TUI state updates.
// These are sent to the TUI to trigger updates.

// TasksUpdatedMsg is sent when the task list changes.
type TasksUpdatedMsg struct {
	Tasks     []*task.Task
	Completed int
	Total     int
}

// TaskStartedMsg is sent when a task starts execution.
type TaskStartedMsg struct {
	TaskID    string
	TaskName  string
	Iteration int
}

// TaskCompletedMsg is sent when a task completes.
type TaskCompletedMsg struct {
	TaskID   string
	TaskName string
	Status   string
	Duration time.Duration
}

// TaskFailedMsg is sent when a task fails.
type TaskFailedMsg struct {
	TaskID    string
	TaskName  string
	Error     string
	Iteration int
}

// AgentOutputMsg is sent for real-time agent output streaming.
type AgentOutputMsg struct {
	Line string
}

// BuildStatusMsg reports build verification status.
type BuildStatusMsg struct {
	Running bool
	Passed  bool
	Output  string
	Error   string
}

// TestStatusMsg reports test verification status.
type TestStatusMsg struct {
	Running bool
	Passed  bool
	Output  string
	Error   string
	Passed_ int // Number of passing tests
	Failed_ int // Number of failing tests
}

// SessionInfoMsg updates session information.
type SessionInfoMsg struct {
	SessionID   string
	ProjectName string
	AgentName   string
	ModelName   string
}

// LoopStateMsg reports the current loop state.
type LoopStateMsg struct {
	State      string // idle, running, paused, awaiting_fix, completed, failed
	Iteration  int
	ElapsedAt  time.Time
	CurrentMsg string // Short status message
}

// ErrorMsg is sent when an error occurs.
type ErrorMsg struct {
	Error string
}

// QuitMsg signals the TUI should quit.
type QuitMsg struct {
	Reason string
}

// TickMsg is sent periodically for time-based updates.
type TickMsg struct {
	Time time.Time
}

// WindowSizeMsg is sent when the terminal size changes.
type WindowSizeMsg struct {
	Width  int
	Height int
}

// KeyPressMsg represents a key press event.
type KeyPressMsg struct {
	Key string
}

// ConfirmMsg is sent when user confirms an action.
type ConfirmMsg struct {
	Action   string
	Accepted bool
}

// HookStatusMsg reports hook execution status.
type HookStatusMsg struct {
	HookName  string
	HookType  string // pre_task or post_task
	Running   bool
	Succeeded bool
	Error     string
}

// AnalysisStatusMsg reports project analysis status.
type AnalysisStatusMsg struct {
	Running  bool
	Complete bool
	Status   string // Status message
	Error    string
}

// FormSubmitMsg is sent when a form is submitted.
type FormSubmitMsg struct {
	FormID string
	Values map[string]interface{}
}

// FormCancelMsg is sent when a form is canceled.
type FormCancelMsg struct {
	FormID string
}

// AnalysisConfirmedMsg is sent when the user confirms the project analysis.
// The Analysis field contains the (potentially modified) analysis data.
type AnalysisConfirmedMsg = components.AnalysisConfirmedMsg

// ReanalyzeRequestedMsg is sent when the user requests to re-run project analysis.
type ReanalyzeRequestedMsg = components.ReanalyzeRequestedMsg

// TaskInitSelectedMsg is sent when the user selects a task init mode.
type TaskInitSelectedMsg = components.TaskInitSelectedMsg

// TaskInitCanceledMsg is sent when the user cancels task init.
type TaskInitCanceledMsg = components.TaskInitCanceledMsg

// TaskListConfirmedMsg is sent when the user confirms the task list.
type TaskListConfirmedMsg = components.TaskListConfirmedMsg

// TaskListReparseMsg is sent when the user wants to re-parse the task list.
type TaskListReparseMsg = components.TaskListReparseMsg
