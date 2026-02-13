package cursor

import (
	"strings"
	"testing"

	"github.com/wexinc/ralph/internal/agent"
)

func TestAgent_Name(t *testing.T) {
	a := New()
	if got := a.Name(); got != "cursor" {
		t.Errorf("Agent.Name() = %q, want %q", got, "cursor")
	}
}

func TestAgent_Description(t *testing.T) {
	a := New()
	if got := a.Description(); got == "" {
		t.Error("Agent.Description() should not be empty")
	}
}

func TestAgent_GetDefaultModel(t *testing.T) {
	a := New()
	model := a.GetDefaultModel()

	if model.ID == "" {
		t.Error("GetDefaultModel().ID should not be empty")
	}
	if !model.IsDefault {
		t.Error("GetDefaultModel().IsDefault should be true")
	}
}

func TestAgent_GetSessionID(t *testing.T) {
	a := New()
	// Initially should be empty
	if got := a.GetSessionID(); got != "" {
		t.Errorf("Initial GetSessionID() = %q, want empty", got)
	}

	// Set the session ID directly for testing
	a.lastSessionID = "test-session-123"
	if got := a.GetSessionID(); got != "test-session-123" {
		t.Errorf("GetSessionID() = %q, want %q", got, "test-session-123")
	}
}

func TestParseModelsOutput(t *testing.T) {
	tests := []struct {
		name         string
		output       string
		defaultModel string
		wantCount    int
		wantFirst    string
		wantDefault  bool
	}{
		{
			name:         "single model",
			output:       "claude-sonnet-4-20250514\n",
			defaultModel: "claude-sonnet-4-20250514",
			wantCount:    1,
			wantFirst:    "claude-sonnet-4-20250514",
			wantDefault:  true,
		},
		{
			name:         "multiple models",
			output:       "claude-sonnet-4-20250514\nclaude-opus-4\ngpt-4o\n",
			defaultModel: "claude-sonnet-4-20250514",
			wantCount:    3,
			wantFirst:    "claude-sonnet-4-20250514",
			wantDefault:  true,
		},
		{
			name:         "with empty lines",
			output:       "\nclaude-sonnet-4\n\ngpt-4o\n\n",
			defaultModel: "gpt-4o",
			wantCount:    2,
			wantFirst:    "claude-sonnet-4",
			wantDefault:  false, // first is not default
		},
		{
			name:         "skip separators",
			output:       "---\nclaude-sonnet-4\n===\ngpt-4o\n",
			defaultModel: "claude-sonnet-4",
			wantCount:    2,
			wantFirst:    "claude-sonnet-4",
			wantDefault:  true,
		},
		{
			name:         "empty output",
			output:       "",
			defaultModel: "claude-sonnet-4",
			wantCount:    0,
		},
		{
			name:         "only whitespace",
			output:       "   \n  \n  ",
			defaultModel: "claude-sonnet-4",
			wantCount:    0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			models := parseModelsOutput(tt.output, tt.defaultModel)

			if len(models) != tt.wantCount {
				t.Errorf("parseModelsOutput() returned %d models, want %d", len(models), tt.wantCount)
				return
			}

			if tt.wantCount > 0 {
				if models[0].ID != tt.wantFirst {
					t.Errorf("first model ID = %q, want %q", models[0].ID, tt.wantFirst)
				}
				if models[0].IsDefault != tt.wantDefault {
					t.Errorf("first model IsDefault = %v, want %v", models[0].IsDefault, tt.wantDefault)
				}
			}
		})
	}
}

func TestParseTaskStatus(t *testing.T) {
	tests := []struct {
		name   string
		output string
		want   agent.TaskStatus
	}{
		{
			name:   "DONE at end",
			output: "Task completed successfully.\n\nDONE",
			want:   agent.TaskStatusDone,
		},
		{
			name:   "DONE with newline",
			output: "Task completed.\nDONE\n",
			want:   agent.TaskStatusDone,
		},
		{
			name:   "NEXT status",
			output: "Still working on it.\n\nNEXT",
			want:   agent.TaskStatusNext,
		},
		{
			name:   "ERROR status",
			output: "Failed to complete.\n\nERROR: something went wrong",
			want:   agent.TaskStatusError,
		},
		{
			name:   "FIXED status",
			output: "Fixed the issue.\n\nFIXED",
			want:   agent.TaskStatusFixed,
		},
		{
			name:   "no status marker",
			output: "Just some output without status",
			want:   agent.TaskStatusUnknown,
		},
		{
			name:   "empty output",
			output: "",
			want:   agent.TaskStatusUnknown,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := parseTaskStatus(tt.output); got != tt.want {
				t.Errorf("parseTaskStatus() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestExtractSessionID(t *testing.T) {
	tests := []struct {
		name   string
		output string
		want   string
	}{
		{
			name:   "session_id with colon",
			output: "Starting work...\nsession_id: abc123-def456\nDone.",
			want:   "abc123-def456",
		},
		{
			name:   "session-id with colon",
			output: "session-id: xyz789",
			want:   "xyz789",
		},
		{
			name:   "sessionid with space",
			output: "sessionid abc_123",
			want:   "abc_123",
		},
		{
			name:   "no session id",
			output: "Just regular output",
			want:   "",
		},
		{
			name:   "empty output",
			output: "",
			want:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := extractSessionID(tt.output); got != tt.want {
				t.Errorf("extractSessionID() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestAgent_GetDefaultModels(t *testing.T) {
	a := New()
	models := a.getDefaultModels()

	if len(models) == 0 {
		t.Error("getDefaultModels() should return at least one model")
	}

	// Check that at least one model is marked as default
	hasDefault := false
	for _, m := range models {
		if m.IsDefault {
			hasDefault = true
			break
		}
	}
	if !hasDefault {
		t.Error("getDefaultModels() should have at least one default model")
	}
}

// TestAgentImplementsInterface verifies that Agent implements agent.Agent interface.
func TestAgentImplementsInterface(t *testing.T) {
	var _ agent.Agent = (*Agent)(nil)
}

func TestNew(t *testing.T) {
	a := New()
	if a == nil {
		t.Fatal("New() returned nil")
	}
	if a.defaultModel != DefaultModel {
		t.Errorf("New().defaultModel = %q, want %q", a.defaultModel, DefaultModel)
	}
	if a.lastSessionID != "" {
		t.Error("New().lastSessionID should be empty")
	}
}

func TestAgent_CheckAuth_NotAvailable(t *testing.T) {
	// Create a new agent
	a := New()

	// If agent CLI is not available, CheckAuth should return an error
	if !a.IsAvailable() {
		err := a.CheckAuth()
		if err == nil {
			t.Error("CheckAuth() should return error when agent CLI is not available")
		}
		if !strings.Contains(err.Error(), "cursor agent CLI not found") {
			t.Errorf("CheckAuth() error = %v, should contain 'cursor agent CLI not found'", err)
		}
	} else {
		// If available, CheckAuth should succeed
		err := a.CheckAuth()
		if err != nil {
			t.Errorf("CheckAuth() error = %v, want nil when agent is available", err)
		}
	}
}

func TestParseModelsOutput_WithDefault(t *testing.T) {
	output := "claude-sonnet-4-20250514\nclaude-opus-4-20250514\ngpt-4o"
	defaultModel := "claude-opus-4-20250514"

	models := parseModelsOutput(output, defaultModel)

	if len(models) != 3 {
		t.Fatalf("parseModelsOutput() returned %d models, want 3", len(models))
	}

	// Check default is marked correctly
	for _, m := range models {
		if m.ID == defaultModel && !m.IsDefault {
			t.Errorf("Model %q should be marked as default", defaultModel)
		}
		if m.ID != defaultModel && m.IsDefault {
			t.Errorf("Model %q should not be marked as default", m.ID)
		}
	}
}

func TestParseModelsOutput_NilOnEmpty(t *testing.T) {
	// Verify empty input returns nil, not empty slice
	models := parseModelsOutput("", "default")
	if models != nil {
		t.Errorf("parseModelsOutput() = %v, want nil for empty input", models)
	}

	models = parseModelsOutput("   \n  ", "default")
	if models != nil {
		t.Errorf("parseModelsOutput() = %v, want nil for whitespace-only input", models)
	}
}

func TestParseTaskStatus_EdgeCases(t *testing.T) {
	tests := []struct {
		name   string
		output string
		want   agent.TaskStatus
	}{
		{
			name:   "DONE in middle of last lines",
			output: "line1\nline2\nline3\nline4\nline5\nDONE\nline7\nline8",
			want:   agent.TaskStatusDone,
		},
		{
			name:   "NEXT with colon",
			output: "work continues\nNEXT: more tasks remain",
			want:   agent.TaskStatusNext,
		},
		{
			name:   "ERROR with message",
			output: "failed\nERROR: Build failed",
			want:   agent.TaskStatusError,
		},
		{
			name:   "FIXED marker",
			output: "fixed the issue\nFIXED: resolved compilation error",
			want:   agent.TaskStatusFixed,
		},
		{
			name:   "whitespace around DONE",
			output: "task done\n  DONE  \n",
			want:   agent.TaskStatusDone,
		},
		{
			name:   "many lines only checks last 10",
			output: "1\n2\n3\n4\n5\n6\n7\n8\n9\n10\n11\nDONE",
			want:   agent.TaskStatusDone, // DONE is within last 10 lines
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := parseTaskStatus(tt.output); got != tt.want {
				t.Errorf("parseTaskStatus() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestExtractSessionID_Variants(t *testing.T) {
	tests := []struct {
		name   string
		output string
		want   string
	}{
		{
			name:   "session_id format",
			output: "session_id: abc-123-def",
			want:   "abc-123-def",
		},
		{
			name:   "session-id format",
			output: "session-id: xyz_456",
			want:   "xyz_456",
		},
		{
			name:   "sessionid single word",
			output: "sessionid abc123",
			want:   "abc123",
		},
		{
			name:   "session id with underscores",
			output: "session_id: test_session_id_123",
			want:   "test_session_id_123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := extractSessionID(tt.output); got != tt.want {
				t.Errorf("extractSessionID() = %q, want %q", got, tt.want)
			}
		})
	}
}
