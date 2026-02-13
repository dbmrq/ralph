package auggie

import (
	"testing"

	"github.com/wexinc/ralph/internal/agent"
)

func TestAgent_Name(t *testing.T) {
	a := New()
	if got := a.Name(); got != "auggie" {
		t.Errorf("Agent.Name() = %q, want %q", got, "auggie")
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
	if model.ID != DefaultModel {
		t.Errorf("GetDefaultModel().ID = %q, want %q", model.ID, DefaultModel)
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
			output:       "claude-sonnet-4\n",
			defaultModel: "claude-sonnet-4",
			wantCount:    1,
			wantFirst:    "claude-sonnet-4",
			wantDefault:  true,
		},
		{
			name:         "multiple models",
			output:       "claude-sonnet-4\nclaude-opus-4\ngpt-4o\n",
			defaultModel: "claude-sonnet-4",
			wantCount:    3,
			wantFirst:    "claude-sonnet-4",
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
	if a.sessionToken != "" {
		t.Error("New().sessionToken should be empty")
	}
}

