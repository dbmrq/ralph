package custom

import (
	"context"
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

