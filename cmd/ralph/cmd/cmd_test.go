package cmd

import (
	"bytes"
	"os"
	"testing"

	"github.com/spf13/cobra"
	"github.com/wexinc/ralph/internal/agent"
	"github.com/wexinc/ralph/internal/config"
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
	run.Flags().String("output", "", "Output format: json for structured output (requires --headless)")
	run.Flags().String("continue", "", "Continue a paused session by ID")
	run.Flags().String("tasks", "", "Path to task file")
	run.Flags().BoolP("verbose", "v", false, "Enable verbose output")
	root.AddCommand(run)

	// Add init command
	initC := &cobra.Command{
		Use:   "init",
		Short: "Initialize Ralph in the current project",
		Long:  "Initialize Ralph in the current project.",
		RunE:  runInit,
	}
	initC.Flags().BoolP("force", "f", false, "Overwrite existing configuration without prompting")
	initC.Flags().BoolP("yes", "y", false, "Non-interactive mode, use AI defaults")
	initC.Flags().StringP("config", "c", "", "Path to config file to use")
	initC.Flags().StringP("tasks", "t", "", "Path to task file to import")
	root.AddCommand(initC)

	// Add agent command group
	agentC := &cobra.Command{
		Use:   "agent",
		Short: "Manage AI agents",
		Long:  "Commands for managing AI agents including listing, adding, and configuring agents.",
	}
	root.AddCommand(agentC)

	// Add agent list command
	agentListC := &cobra.Command{
		Use:   "list",
		Short: "List available agents",
		Long:  "List all available agents including built-in and custom agents.",
		RunE:  runAgentList,
	}
	agentC.AddCommand(agentListC)

	// Add agent add command
	agentAddC := &cobra.Command{
		Use:   "add",
		Short: "Add a custom agent",
		Long:  "Add a custom agent to Ralph.",
		RunE:  runAgentAdd,
	}
	agentAddC.Flags().StringP("name", "n", "", "Agent name")
	agentAddC.Flags().StringP("command", "c", "", "Agent command")
	agentAddC.Flags().StringP("description", "d", "", "Agent description")
	agentAddC.Flags().String("detection", "command", "Detection method")
	agentAddC.Flags().String("detection-value", "", "Value for detection")
	agentAddC.Flags().String("model-list-cmd", "", "Command to list available models")
	agentAddC.Flags().String("default-model", "", "Default model for the agent")
	agentC.AddCommand(agentAddC)

	// Add version command
	versionC := &cobra.Command{
		Use:   "version",
		Short: "Show version information",
		Long:  "Show detailed version information for ralph.",
		RunE:  runVersion,
	}
	versionC.Flags().BoolP("check", "c", false, "Check for available updates")
	root.AddCommand(versionC)

	// Add update command
	updateC := &cobra.Command{
		Use:   "update",
		Short: "Update ralph to the latest version",
		Long:  "Update ralph to the latest version.",
		RunE:  runUpdate,
	}
	updateC.Flags().BoolP("check", "c", false, "Only check for updates, don't install")
	updateC.Flags().BoolP("yes", "y", false, "Don't prompt for confirmation")
	root.AddCommand(updateC)

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
		skipSetup  bool // if true, create .ralph dir to skip setup flow
	}{
		// Note: TUI mode and headless mode now actually try to run the loop,
		// which requires agents, config, tasks, etc. This is tested in integration tests.
		// Here we only test the help and flag parsing.
		{
			name:       "run help",
			args:       []string{"run", "--help"},
			wantErr:    false,
			wantOutput: "--headless",
			skipSetup:  false, // help doesn't run the command
		},
		{
			name:      "output requires headless",
			args:      []string{"run", "--output", "json"},
			wantErr:   true,
			skipSetup: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set up temp directory if we need to skip setup flow
			if tt.skipSetup {
				tmpDir := t.TempDir()
				// Create .ralph directory to skip setup flow
				if err := os.MkdirAll(tmpDir+"/.ralph", 0755); err != nil {
					t.Fatalf("failed to create .ralph dir: %v", err)
				}
				// Change to temp directory for the test
				oldWd, _ := os.Getwd()
				if err := os.Chdir(tmpDir); err != nil {
					t.Fatalf("failed to chdir: %v", err)
				}
				defer os.Chdir(oldWd)
			}

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
			name:       "init help",
			args:       []string{"init", "--help"},
			wantErr:    false,
			wantOutput: "--force",
		},
		{
			name:       "init help shows yes flag",
			args:       []string{"init", "--help"},
			wantErr:    false,
			wantOutput: "--yes",
		},
		{
			name:       "init help shows config flag",
			args:       []string{"init", "--help"},
			wantErr:    false,
			wantOutput: "--config",
		},
		{
			name:       "init help shows tasks flag",
			args:       []string{"init", "--help"},
			wantErr:    false,
			wantOutput: "--tasks",
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

func TestInitCommandWithConfig(t *testing.T) {
	// Test init --config with a valid config file
	t.Run("init with config file", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create a config file
		configPath := tmpDir + "/test-config.yaml"
		configContent := `
agent:
  default: cursor
build:
  command: go build ./...
test:
  command: go test ./...
`
		if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
			t.Fatalf("failed to write config: %v", err)
		}

		// Change to temp directory
		oldWd, _ := os.Getwd()
		if err := os.Chdir(tmpDir); err != nil {
			t.Fatalf("failed to chdir: %v", err)
		}
		defer os.Chdir(oldWd)

		buf := new(bytes.Buffer)
		cmd := newTestRoot()
		cmd.SetOut(buf)
		cmd.SetErr(buf)
		cmd.SetArgs([]string{"init", "--config", configPath})

		err := cmd.Execute()
		if err != nil {
			t.Fatalf("Execute() error = %v", err)
		}

		// Check that .ralph directory was created
		if _, err := os.Stat(tmpDir + "/.ralph"); os.IsNotExist(err) {
			t.Error(".ralph directory was not created")
		}

		// Check that config.yaml was created
		if _, err := os.Stat(tmpDir + "/.ralph/config.yaml"); os.IsNotExist(err) {
			t.Error(".ralph/config.yaml was not created")
		}

		// Check output
		if !bytes.Contains(buf.Bytes(), []byte("initialized successfully")) {
			t.Errorf("Output = %q, want to contain 'initialized successfully'", buf.String())
		}
	})

	// Test init --config with non-existent file
	t.Run("init with missing config file", func(t *testing.T) {
		tmpDir := t.TempDir()

		oldWd, _ := os.Getwd()
		if err := os.Chdir(tmpDir); err != nil {
			t.Fatalf("failed to chdir: %v", err)
		}
		defer os.Chdir(oldWd)

		buf := new(bytes.Buffer)
		cmd := newTestRoot()
		cmd.SetOut(buf)
		cmd.SetErr(buf)
		cmd.SetArgs([]string{"init", "--config", "/nonexistent/config.yaml"})

		err := cmd.Execute()
		if err == nil {
			t.Error("Expected error for missing config file")
		}
	})
}

func TestInitCommandExistingRalph(t *testing.T) {
	// Test that init with --yes and existing .ralph errors without --force
	t.Run("init --yes with existing .ralph errors", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create .ralph directory
		if err := os.MkdirAll(tmpDir+"/.ralph", 0755); err != nil {
			t.Fatalf("failed to create .ralph dir: %v", err)
		}

		oldWd, _ := os.Getwd()
		if err := os.Chdir(tmpDir); err != nil {
			t.Fatalf("failed to chdir: %v", err)
		}
		defer os.Chdir(oldWd)

		buf := new(bytes.Buffer)
		cmd := newTestRoot()
		cmd.SetOut(buf)
		cmd.SetErr(buf)
		cmd.SetArgs([]string{"init", "--yes"})

		err := cmd.Execute()
		if err == nil {
			t.Error("Expected error when .ralph exists without --force")
		}
		if err != nil && !bytes.Contains([]byte(err.Error()), []byte("already exists")) {
			t.Errorf("Error = %q, want to contain 'already exists'", err.Error())
		}
	})

	// Test that init with --force and existing .ralph succeeds
	t.Run("init --force removes existing .ralph", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create .ralph directory with a file
		ralphDir := tmpDir + "/.ralph"
		if err := os.MkdirAll(ralphDir, 0755); err != nil {
			t.Fatalf("failed to create .ralph dir: %v", err)
		}
		if err := os.WriteFile(ralphDir+"/old-config.yaml", []byte("old"), 0644); err != nil {
			t.Fatalf("failed to write old config: %v", err)
		}

		// Create a config file to use
		configPath := tmpDir + "/test-config.yaml"
		if err := os.WriteFile(configPath, []byte("agent:\n  default: cursor\n"), 0644); err != nil {
			t.Fatalf("failed to write config: %v", err)
		}

		oldWd, _ := os.Getwd()
		if err := os.Chdir(tmpDir); err != nil {
			t.Fatalf("failed to chdir: %v", err)
		}
		defer os.Chdir(oldWd)

		buf := new(bytes.Buffer)
		cmd := newTestRoot()
		cmd.SetOut(buf)
		cmd.SetErr(buf)
		cmd.SetArgs([]string{"init", "--force", "--config", configPath})

		err := cmd.Execute()
		if err != nil {
			t.Fatalf("Execute() error = %v", err)
		}

		// Check that old file is gone
		if _, err := os.Stat(ralphDir + "/old-config.yaml"); !os.IsNotExist(err) {
			t.Error("old-config.yaml should have been removed")
		}

		// Check output mentions removal
		if !bytes.Contains(buf.Bytes(), []byte("Removed existing")) {
			t.Errorf("Output = %q, want to contain 'Removed existing'", buf.String())
		}
	})
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

func TestAgentCommand(t *testing.T) {
	tests := []struct {
		name       string
		args       []string
		wantErr    bool
		wantOutput string
	}{
		{
			name:       "agent help",
			args:       []string{"agent", "--help"},
			wantErr:    false,
			wantOutput: "managing AI agents",
		},
		{
			name:       "agent list",
			args:       []string{"agent", "list"},
			wantErr:    false,
			wantOutput: "Registered agents:",
		},
		{
			name:       "agent add help",
			args:       []string{"agent", "add", "--help"},
			wantErr:    false,
			wantOutput: "Add a custom agent",
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

func TestRegisterCustomAgentsFromConfig(t *testing.T) {
	r := agent.NewRegistry()

	cfg := &config.Config{
		Agent: config.AgentConfig{
			Custom: []config.CustomAgentConfig{
				{
					Name:            "test-custom",
					Command:         "echo",
					Description:     "Test custom agent",
					DetectionMethod: config.DetectionMethodAlways,
				},
			},
		},
	}

	RegisterCustomAgentsFromConfig(r, cfg)

	// Verify the custom agent was registered
	a, ok := r.Get("test-custom")
	if !ok {
		t.Fatal("test-custom agent should be registered")
	}

	if a.Name() != "test-custom" {
		t.Errorf("agent.Name() = %q, want %q", a.Name(), "test-custom")
	}

	if a.Description() != "Test custom agent" {
		t.Errorf("agent.Description() = %q, want %q", a.Description(), "Test custom agent")
	}

	if !a.IsAvailable() {
		t.Error("agent should be available (detection method is 'always')")
	}
}

func TestVersionCommand(t *testing.T) {
	tests := []struct {
		name       string
		args       []string
		wantErr    bool
		wantOutput string
	}{
		{
			name:       "version shows info",
			args:       []string{"version"},
			wantErr:    false,
			wantOutput: "ralph",
		},
		{
			name:       "version help",
			args:       []string{"version", "--help"},
			wantErr:    false,
			wantOutput: "--check",
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

func TestUpdateCommand(t *testing.T) {
	tests := []struct {
		name       string
		args       []string
		wantErr    bool
		wantOutput string
	}{
		{
			name:       "update help",
			args:       []string{"update", "--help"},
			wantErr:    false,
			wantOutput: "Update ralph to the latest version",
		},
		{
			name:       "update help shows check flag",
			args:       []string{"update", "--help"},
			wantErr:    false,
			wantOutput: "--check",
		},
		{
			name:       "update help shows yes flag",
			args:       []string{"update", "--help"},
			wantErr:    false,
			wantOutput: "--yes",
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
