package custom

import (
	"bytes"
	"context"
	"os"
	"strings"
	"testing"

	"github.com/wexinc/ralph/internal/agent"
	"github.com/wexinc/ralph/internal/config"
)

func TestAgent_Name(t *testing.T) {
	a := New(config.CustomAgentConfig{
		Name:    "test-agent",
		Command: "echo",
	})

	if got := a.Name(); got != "test-agent" {
		t.Errorf("Name() = %q, want %q", got, "test-agent")
	}
}

func TestAgent_Description(t *testing.T) {
	tests := []struct {
		name   string
		config config.CustomAgentConfig
		want   string
	}{
		{
			name: "custom description",
			config: config.CustomAgentConfig{
				Name:        "test",
				Command:     "cmd",
				Description: "My custom agent",
			},
			want: "My custom agent",
		},
		{
			name: "default description",
			config: config.CustomAgentConfig{
				Name:    "test",
				Command: "my-cli",
			},
			want: "Custom agent using my-cli",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := New(tt.config)
			if got := a.Description(); got != tt.want {
				t.Errorf("Description() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestAgent_IsAvailable(t *testing.T) {
	tests := []struct {
		name   string
		config config.CustomAgentConfig
		want   bool
	}{
		{
			name: "command detection - exists",
			config: config.CustomAgentConfig{
				Name:            "test",
				Command:         "echo hello",
				DetectionMethod: config.DetectionMethodCommand,
			},
			want: true, // echo should exist
		},
		{
			name: "command detection - not exists",
			config: config.CustomAgentConfig{
				Name:            "test",
				Command:         "nonexistent_command_xyz_123",
				DetectionMethod: config.DetectionMethodCommand,
			},
			want: false,
		},
		{
			name: "always detection",
			config: config.CustomAgentConfig{
				Name:            "test",
				Command:         "nonexistent",
				DetectionMethod: config.DetectionMethodAlways,
			},
			want: true,
		},
		{
			name: "env detection - missing",
			config: config.CustomAgentConfig{
				Name:            "test",
				Command:         "cmd",
				DetectionMethod: config.DetectionMethodEnv,
				DetectionValue:  "UNLIKELY_ENV_VAR_XYZ_123",
			},
			want: false,
		},
		{
			name: "path detection - missing value",
			config: config.CustomAgentConfig{
				Name:            "test",
				Command:         "cmd",
				DetectionMethod: config.DetectionMethodPath,
				DetectionValue:  "",
			},
			want: false,
		},
		{
			name: "default (command) detection with existing command",
			config: config.CustomAgentConfig{
				Name:    "test",
				Command: "echo test",
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := New(tt.config)
			if got := a.IsAvailable(); got != tt.want {
				t.Errorf("IsAvailable() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAgent_GetDefaultModel(t *testing.T) {
	tests := []struct {
		name         string
		defaultModel string
		wantID       string
	}{
		{
			name:         "with default model",
			defaultModel: "gpt-4",
			wantID:       "gpt-4",
		},
		{
			name:         "without default model",
			defaultModel: "",
			wantID:       "default",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := New(config.CustomAgentConfig{
				Name:         "test",
				Command:      "cmd",
				DefaultModel: tt.defaultModel,
			})
			model := a.GetDefaultModel()
			if model.ID != tt.wantID {
				t.Errorf("GetDefaultModel().ID = %q, want %q", model.ID, tt.wantID)
			}
			if !model.IsDefault {
				t.Error("GetDefaultModel().IsDefault should be true")
			}
		})
	}
}

func TestAgent_ListModels_NoCommand(t *testing.T) {
	a := New(config.CustomAgentConfig{
		Name:         "test",
		Command:      "cmd",
		DefaultModel: "my-model",
	})

	models, err := a.ListModels()
	if err != nil {
		t.Fatalf("ListModels() error = %v", err)
	}

	if len(models) != 1 {
		t.Fatalf("ListModels() returned %d models, want 1", len(models))
	}

	if models[0].ID != "my-model" {
		t.Errorf("ListModels()[0].ID = %q, want %q", models[0].ID, "my-model")
	}
}

func TestAgent_CheckAuth(t *testing.T) {
	tests := []struct {
		name    string
		config  config.CustomAgentConfig
		wantErr bool
	}{
		{
			name: "available agent",
			config: config.CustomAgentConfig{
				Name:            "test",
				Command:         "echo",
				DetectionMethod: config.DetectionMethodCommand,
			},
			wantErr: false,
		},
		{
			name: "unavailable agent",
			config: config.CustomAgentConfig{
				Name:            "test",
				Command:         "nonexistent_xyz_123",
				DetectionMethod: config.DetectionMethodCommand,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := New(tt.config)
			err := a.CheckAuth()
			if (err != nil) != tt.wantErr {
				t.Errorf("CheckAuth() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestAgent_Run(t *testing.T) {
	a := New(config.CustomAgentConfig{
		Name:    "test",
		Command: "echo",
	})

	result, err := a.Run(context.Background(), "DONE", agent.RunOptions{})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	if result.ExitCode != 0 {
		t.Errorf("Run().ExitCode = %d, want 0", result.ExitCode)
	}

	if result.Status != agent.TaskStatusDone {
		t.Errorf("Run().Status = %q, want %q", result.Status, agent.TaskStatusDone)
	}
}

func TestAgent_GetSessionID(t *testing.T) {
	a := New(config.CustomAgentConfig{
		Name:    "test",
		Command: "echo",
	})

	// Initially should be empty
	if got := a.GetSessionID(); got != "" {
		t.Errorf("GetSessionID() before run = %q, want empty", got)
	}
}

func TestParseTaskStatus(t *testing.T) {
	tests := []struct {
		name   string
		output string
		want   agent.TaskStatus
	}{
		{name: "done", output: "work done\nDONE", want: agent.TaskStatusDone},
		{name: "next", output: "more work\nNEXT", want: agent.TaskStatusNext},
		{name: "error", output: "failed\nERROR: something wrong", want: agent.TaskStatusError},
		{name: "fixed", output: "resolved\nFIXED", want: agent.TaskStatusFixed},
		{name: "unknown", output: "no status marker", want: agent.TaskStatusUnknown},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := parseTaskStatus(tt.output); got != tt.want {
				t.Errorf("parseTaskStatus() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestParseModelsOutput(t *testing.T) {
	tests := []struct {
		name         string
		output       string
		defaultModel string
		wantCount    int
	}{
		{
			name:         "multiple models",
			output:       "model-a\nmodel-b\nmodel-c",
			defaultModel: "model-b",
			wantCount:    3,
		},
		{
			name:         "with separators",
			output:       "---\nmodel-a\n===\nmodel-b",
			defaultModel: "",
			wantCount:    2,
		},
		{
			name:         "empty",
			output:       "",
			defaultModel: "",
			wantCount:    0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			models := parseModelsOutput(tt.output, tt.defaultModel)
			if len(models) != tt.wantCount {
				t.Errorf("parseModelsOutput() returned %d models, want %d", len(models), tt.wantCount)
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
			name:   "with session id",
			output: "Starting...\nsession_id: abc123\nDone",
			want:   "abc123",
		},
		{
			name:   "no session id",
			output: "Just some output",
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

// Verify that Agent implements the agent.Agent interface
var _ agent.Agent = (*Agent)(nil)

func TestAgent_Continue(t *testing.T) {
	a := New(config.CustomAgentConfig{
		Name:    "test",
		Command: "echo",
	})

	result, err := a.Continue(context.Background(), "session-123", "DONE", agent.RunOptions{})
	if err != nil {
		t.Fatalf("Continue() error = %v", err)
	}

	if result.ExitCode != 0 {
		t.Errorf("Continue().ExitCode = %d, want 0", result.ExitCode)
	}

	if result.Status != agent.TaskStatusDone {
		t.Errorf("Continue().Status = %q, want %q", result.Status, agent.TaskStatusDone)
	}
}

func TestAgent_IsAvailable_PathDetection(t *testing.T) {
	// Create a temporary file to test path detection
	tmpFile, err := os.CreateTemp("", "test-agent-*")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	tests := []struct {
		name   string
		config config.CustomAgentConfig
		want   bool
	}{
		{
			name: "path detection - file exists",
			config: config.CustomAgentConfig{
				Name:            "test",
				Command:         "echo",
				DetectionMethod: config.DetectionMethodPath,
				DetectionValue:  tmpFile.Name(),
			},
			want: true,
		},
		{
			name: "path detection - file does not exist",
			config: config.CustomAgentConfig{
				Name:            "test",
				Command:         "echo",
				DetectionMethod: config.DetectionMethodPath,
				DetectionValue:  "/nonexistent/path/to/file",
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := New(tt.config)
			if got := a.IsAvailable(); got != tt.want {
				t.Errorf("IsAvailable() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAgent_IsAvailable_EnvDetection(t *testing.T) {
	// Set test environment variable
	t.Setenv("TEST_AGENT_ENV_VAR", "value")

	tests := []struct {
		name   string
		config config.CustomAgentConfig
		want   bool
	}{
		{
			name: "env detection - var exists",
			config: config.CustomAgentConfig{
				Name:            "test",
				Command:         "echo",
				DetectionMethod: config.DetectionMethodEnv,
				DetectionValue:  "TEST_AGENT_ENV_VAR",
			},
			want: true,
		},
		{
			name: "env detection - var does not exist",
			config: config.CustomAgentConfig{
				Name:            "test",
				Command:         "echo",
				DetectionMethod: config.DetectionMethodEnv,
				DetectionValue:  "NONEXISTENT_ENV_VAR_XYZ_123_456",
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := New(tt.config)
			if got := a.IsAvailable(); got != tt.want {
				t.Errorf("IsAvailable() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAgent_IsAvailable_UnknownMethod(t *testing.T) {
	a := New(config.CustomAgentConfig{
		Name:            "test",
		Command:         "echo",
		DetectionMethod: "unknown_method",
	})

	if a.IsAvailable() {
		t.Error("IsAvailable() should return false for unknown detection method")
	}
}

func TestAgent_ListModels_WithCommand(t *testing.T) {
	// Use echo to simulate a model list command
	a := New(config.CustomAgentConfig{
		Name:             "test",
		Command:          "echo",
		DefaultModel:     "model-b",
		ModelListCommand: "echo -e 'model-a\nmodel-b\nmodel-c'",
	})

	models, err := a.ListModels()
	if err != nil {
		t.Fatalf("ListModels() error = %v", err)
	}

	// The echo command should output model names
	if len(models) == 0 {
		t.Error("ListModels() should return at least one model")
	}
}

func TestAgent_Run_WithLogWriter(t *testing.T) {
	a := New(config.CustomAgentConfig{
		Name:    "test",
		Command: "echo",
	})

	var logBuf bytes.Buffer
	result, err := a.Run(context.Background(), "DONE", agent.RunOptions{
		LogWriter: &logBuf,
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	if result.ExitCode != 0 {
		t.Errorf("Run().ExitCode = %d, want 0", result.ExitCode)
	}

	// LogWriter should have received the output
	if logBuf.Len() == 0 {
		t.Error("LogWriter should have received output")
	}
}

func TestAgent_Run_WithWorkDir(t *testing.T) {
	tmpDir := t.TempDir()

	a := New(config.CustomAgentConfig{
		Name:    "test",
		Command: "sh -c 'pwd'",
	})

	result, err := a.Run(context.Background(), "", agent.RunOptions{
		WorkDir: tmpDir,
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	// Verify exit code is 0
	if result.ExitCode != 0 {
		t.Errorf("Run().ExitCode = %d, want 0", result.ExitCode)
	}
}

func TestAgent_Run_EmptyCommand(t *testing.T) {
	a := New(config.CustomAgentConfig{
		Name:    "test",
		Command: "",
	})

	_, err := a.Run(context.Background(), "prompt", agent.RunOptions{})
	if err == nil {
		t.Error("Run() with empty command should return error")
	}
}

func TestAgent_Run_NonExistentCommand(t *testing.T) {
	a := New(config.CustomAgentConfig{
		Name:    "test",
		Command: "nonexistent_command_xyz_123",
	})

	result, err := a.Run(context.Background(), "prompt", agent.RunOptions{})
	// The execute method returns nil error even for command failures
	// (error handling is designed to not interrupt the loop)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	// Note: When command doesn't exist, cmd.ProcessState is nil so exitCode stays 0.
	// The current implementation doesn't capture this case as an error.
	// This test verifies the actual behavior - the method completes without error.
	// A future improvement could handle this case better.
	if result.Status != agent.TaskStatusUnknown {
		t.Errorf("Run().Status = %v, want TaskStatusUnknown for failed command", result.Status)
	}
}

func TestAgent_Run_ExtractsSessionID(t *testing.T) {
	a := New(config.CustomAgentConfig{
		Name:    "test",
		Command: "echo",
	})

	result, err := a.Run(context.Background(), "session_id: test-session-abc", agent.RunOptions{})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	if result.SessionID != "test-session-abc" {
		t.Errorf("Run().SessionID = %q, want %q", result.SessionID, "test-session-abc")
	}

	// GetSessionID should return the same value
	if a.GetSessionID() != "test-session-abc" {
		t.Errorf("GetSessionID() = %q, want %q", a.GetSessionID(), "test-session-abc")
	}
}

func TestAgent_Run_WithArgs(t *testing.T) {
	a := New(config.CustomAgentConfig{
		Name:    "test",
		Command: "echo",
		Args:    []string{"arg1", "arg2"},
	})

	result, err := a.Run(context.Background(), "prompt", agent.RunOptions{})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	// Output should contain the args
	if !strings.Contains(result.Output, "arg1") {
		t.Errorf("Run() output = %q, should contain args", result.Output)
	}
}

func TestParseModelsOutput_NilOnEmpty(t *testing.T) {
	models := parseModelsOutput("", "default")
	if models != nil {
		t.Errorf("parseModelsOutput() = %v, want nil for empty input", models)
	}
}

func TestParseModelsOutput_SkipsSeparators(t *testing.T) {
	// Only model names, no header - the function doesn't skip headers
	output := "---\nmodel-a\n===\nmodel-b\n---"
	models := parseModelsOutput(output, "model-a")

	if len(models) != 2 {
		t.Errorf("parseModelsOutput() returned %d models, want 2", len(models))
	}

	for _, m := range models {
		if strings.HasPrefix(m.ID, "-") || strings.HasPrefix(m.ID, "=") {
			t.Errorf("Model ID should not be a separator: %q", m.ID)
		}
	}
}
