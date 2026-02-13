package agent

import (
	"context"
	"testing"
)

func TestModel_String(t *testing.T) {
	tests := []struct {
		name     string
		model    Model
		expected string
	}{
		{
			name:     "with name",
			model:    Model{ID: "claude-opus-4", Name: "Claude Opus 4"},
			expected: "Claude Opus 4",
		},
		{
			name:     "without name",
			model:    Model{ID: "claude-opus-4"},
			expected: "claude-opus-4",
		},
		{
			name:     "empty",
			model:    Model{},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.model.String(); got != tt.expected {
				t.Errorf("Model.String() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestTaskStatus_String(t *testing.T) {
	tests := []struct {
		status   TaskStatus
		expected string
	}{
		{TaskStatusNext, "NEXT"},
		{TaskStatusDone, "DONE"},
		{TaskStatusError, "ERROR"},
		{TaskStatusFixed, "FIXED"},
		{TaskStatusUnknown, "UNKNOWN"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if got := tt.status.String(); got != tt.expected {
				t.Errorf("TaskStatus.String() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestTaskStatus_IsTerminal(t *testing.T) {
	tests := []struct {
		status   TaskStatus
		expected bool
	}{
		{TaskStatusNext, false},
		{TaskStatusDone, true},
		{TaskStatusError, true},
		{TaskStatusFixed, false},
		{TaskStatusUnknown, false},
	}

	for _, tt := range tests {
		t.Run(tt.status.String(), func(t *testing.T) {
			if got := tt.status.IsTerminal(); got != tt.expected {
				t.Errorf("TaskStatus.IsTerminal() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestTaskStatus_IsSuccess(t *testing.T) {
	tests := []struct {
		status   TaskStatus
		expected bool
	}{
		{TaskStatusNext, false},
		{TaskStatusDone, true},
		{TaskStatusError, false},
		{TaskStatusFixed, true},
		{TaskStatusUnknown, false},
	}

	for _, tt := range tests {
		t.Run(tt.status.String(), func(t *testing.T) {
			if got := tt.status.IsSuccess(); got != tt.expected {
				t.Errorf("TaskStatus.IsSuccess() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestResult_IsSuccess(t *testing.T) {
	tests := []struct {
		name     string
		result   Result
		expected bool
	}{
		{
			name:     "success with DONE status",
			result:   Result{ExitCode: 0, Status: TaskStatusDone},
			expected: true,
		},
		{
			name:     "success with FIXED status",
			result:   Result{ExitCode: 0, Status: TaskStatusFixed},
			expected: true,
		},
		{
			name:     "non-zero exit code",
			result:   Result{ExitCode: 1, Status: TaskStatusDone},
			expected: false,
		},
		{
			name:     "error status",
			result:   Result{ExitCode: 0, Status: TaskStatusError},
			expected: false,
		},
		{
			name:     "next status",
			result:   Result{ExitCode: 0, Status: TaskStatusNext},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.result.IsSuccess(); got != tt.expected {
				t.Errorf("Result.IsSuccess() = %v, want %v", got, tt.expected)
			}
		})
	}
}

// mockAgent implements the Agent interface for testing.
type mockAgent struct {
	name        string
	description string
	available   bool
	authError   error
	models      []Model
	defaultMdl  Model
	runResult   Result
	runError    error
	sessionID   string
}

func (m *mockAgent) Name() string                 { return m.name }
func (m *mockAgent) Description() string          { return m.description }
func (m *mockAgent) IsAvailable() bool            { return m.available }
func (m *mockAgent) CheckAuth() error             { return m.authError }
func (m *mockAgent) ListModels() ([]Model, error) { return m.models, nil }
func (m *mockAgent) GetDefaultModel() Model       { return m.defaultMdl }
func (m *mockAgent) GetSessionID() string         { return m.sessionID }

func (m *mockAgent) Run(_ context.Context, _ string, _ RunOptions) (Result, error) {
	return m.runResult, m.runError
}

func (m *mockAgent) Continue(_ context.Context, _ string, _ string, _ RunOptions) (Result, error) {
	return m.runResult, m.runError
}

