package cmd

import (
	"bytes"
	"testing"

	"github.com/spf13/cobra"
	"github.com/wexinc/ralph/internal/agent"
)

// newTestRoot creates a fresh command hierarchy for testing.
// This is necessary because Cobra commands maintain state between runs.
func newTestRoot() *cobra.Command {
	root := &cobra.Command{
		Use:   "ralph",
		Short: "Ralph Loop - AI-powered task automation",
		Long: `Ralph is an AI-powered task automation tool that runs in a loop,
completing tasks from a task list using AI agents.`,
	}
	root.Version = "test"
	root.SetVersionTemplate("ralph {{.Version}}\n")

	// Add run command
	run := &cobra.Command{
		Use:   "run",
		Short: "Start the Ralph loop to execute tasks",
		Long:  "Start the Ralph loop to execute tasks from the task list.",
		RunE:  runRun,
	}
	run.Flags().Bool("headless", false, "Run in headless mode without TUI")
	run.Flags().Bool("output", false, "Output in JSON format (requires --headless)")
	run.Flags().String("continue", "", "Continue a paused session by ID")
	root.AddCommand(run)

	// Add init command
	initC := &cobra.Command{
		Use:   "init",
		Short: "Initialize Ralph in the current project",
		Long:  "Initialize Ralph in the current project.",
		RunE:  runInit,
	}
	initC.Flags().BoolP("force", "f", false, "Overwrite existing configuration")
	root.AddCommand(initC)

	return root
}

func TestRootCommand(t *testing.T) {
	tests := []struct {
		name       string
		args       []string
		wantErr    bool
		wantOutput string
	}{
		{
			name:       "no args shows help",
			args:       []string{},
			wantErr:    false,
			wantOutput: "Ralph is an AI-powered task automation tool",
		},
		{
			name:       "help flag",
			args:       []string{"--help"},
			wantErr:    false,
			wantOutput: "Available Commands:",
		},
		{
			name:       "version flag",
			args:       []string{"--version"},
			wantErr:    false,
			wantOutput: "ralph",
		},
		{
			name:    "unknown command",
			args:    []string{"unknown"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := new(bytes.Buffer)
			cmd := newTestRoot()
			cmd.SetOut(buf)
			cmd.SetErr(buf)
			cmd.SetArgs(tt.args)

			err := cmd.Execute()
			if (err != nil) != tt.wantErr {
				t.Errorf("Execute() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantOutput != "" && !bytes.Contains(buf.Bytes(), []byte(tt.wantOutput)) {
				t.Errorf("Output = %q, want to contain %q", buf.String(), tt.wantOutput)
			}
		})
	}
}

func TestRunCommand(t *testing.T) {
	tests := []struct {
		name       string
		args       []string
		wantErr    bool
		wantOutput string
	}{
		{
			name:       "run without flags",
			args:       []string{"run"},
			wantErr:    false,
			wantOutput: "Starting Ralph in TUI mode",
		},
		{
			name:       "run with headless flag",
			args:       []string{"run", "--headless"},
			wantErr:    false,
			wantOutput: "Starting Ralph in headless mode",
		},
		{
			name:       "run help",
			args:       []string{"run", "--help"},
			wantErr:    false,
			wantOutput: "--headless",
		},
		{
			name:       "run with continue flag",
			args:       []string{"run", "--continue", "session-123"},
			wantErr:    false,
			wantOutput: "Continuing session: session-123",
		},
		{
			name:    "output requires headless",
			args:    []string{"run", "--output"},
			wantErr: true,
		},
		{
			name:       "output with headless works",
			args:       []string{"run", "--headless", "--output"},
			wantErr:    false,
			wantOutput: "headless mode",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := new(bytes.Buffer)
			cmd := newTestRoot()
			cmd.SetOut(buf)
			cmd.SetErr(buf)
			cmd.SetArgs(tt.args)

			err := cmd.Execute()
			if (err != nil) != tt.wantErr {
				t.Errorf("Execute() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantOutput != "" && !bytes.Contains(buf.Bytes(), []byte(tt.wantOutput)) {
				t.Errorf("Output = %q, want to contain %q", buf.String(), tt.wantOutput)
			}
		})
	}
}

func TestInitCommand(t *testing.T) {
	tests := []struct {
		name       string
		args       []string
		wantErr    bool
		wantOutput string
	}{
		{
			name:       "init without flags",
			args:       []string{"init"},
			wantErr:    false,
			wantOutput: "Initializing Ralph...",
		},
		{
			name:       "init with force flag",
			args:       []string{"init", "--force"},
			wantErr:    false,
			wantOutput: "force mode",
		},
		{
			name:       "init with force short flag",
			args:       []string{"init", "-f"},
			wantErr:    false,
			wantOutput: "force mode",
		},
		{
			name:       "init help",
			args:       []string{"init", "--help"},
			wantErr:    false,
			wantOutput: "--force",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := new(bytes.Buffer)
			cmd := newTestRoot()
			cmd.SetOut(buf)
			cmd.SetErr(buf)
			cmd.SetArgs(tt.args)

			err := cmd.Execute()
			if (err != nil) != tt.wantErr {
				t.Errorf("Execute() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantOutput != "" && !bytes.Contains(buf.Bytes(), []byte(tt.wantOutput)) {
				t.Errorf("Output = %q, want to contain %q", buf.String(), tt.wantOutput)
			}
		})
	}
}

func TestDefaultRegistry(t *testing.T) {
	t.Run("registry is initialized", func(t *testing.T) {
		if DefaultRegistry == nil {
			t.Fatal("DefaultRegistry should not be nil")
		}
	})

	t.Run("cursor agent is registered", func(t *testing.T) {
		a, ok := DefaultRegistry.Get("cursor")
		if !ok {
			t.Fatal("cursor agent should be registered")
		}
		if a.Name() != "cursor" {
			t.Errorf("agent.Name() = %q, want %q", a.Name(), "cursor")
		}
	})

	t.Run("auggie agent is registered", func(t *testing.T) {
		a, ok := DefaultRegistry.Get("auggie")
		if !ok {
			t.Fatal("auggie agent should be registered")
		}
		if a.Name() != "auggie" {
			t.Errorf("agent.Name() = %q, want %q", a.Name(), "auggie")
		}
	})

	t.Run("has expected agent count", func(t *testing.T) {
		all := DefaultRegistry.All()
		if len(all) != 2 {
			t.Errorf("Registry.All() count = %d, want 2", len(all))
		}
	})
}

func TestDefaultDiscovery(t *testing.T) {
	if DefaultDiscovery == nil {
		t.Fatal("DefaultDiscovery should not be nil")
	}
}

func TestRegisterDefaultAgents(t *testing.T) {
	// Test with a fresh registry
	r := agent.NewRegistry()
	RegisterDefaultAgents(r)

	if len(r.All()) != 2 {
		t.Errorf("RegisterDefaultAgents() registered %d agents, want 2", len(r.All()))
	}

	// Verify specific agents
	if _, ok := r.Get("cursor"); !ok {
		t.Error("cursor agent should be registered")
	}
	if _, ok := r.Get("auggie"); !ok {
		t.Error("auggie agent should be registered")
	}
}

