package cmd

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/spf13/cobra"
	"github.com/wexinc/ralph/internal/agent"
	"github.com/wexinc/ralph/internal/agent/auggie"
	"github.com/wexinc/ralph/internal/agent/cursor"
	"github.com/wexinc/ralph/internal/app"
	"github.com/wexinc/ralph/internal/config"
	"github.com/wexinc/ralph/internal/tui"
)

// initCmd represents the init command.
var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize Ralph in the current project",
	Long: `Initialize Ralph in the current project.

This command creates the .ralph directory and runs project analysis to configure
build/test commands and detect/import task lists.

Modes:
  ralph init              Interactive setup with TUI confirmation
  ralph init --yes        Non-interactive, use AI defaults
  ralph init --config X   Use provided config file
  ralph init --tasks X    Point to existing task file

If .ralph/ exists, you'll be prompted to reconfigure or exit.
Use --force to overwrite existing configuration without prompting.

Examples:
  ralph init                        # Interactive setup
  ralph init --yes                  # Non-interactive with AI defaults
  ralph init --config config.yaml   # Use existing config
  ralph init --tasks TASKS.md       # Import tasks from file
  ralph init --force                # Reinitialize, overwriting existing`,
	RunE: runInit,
}

func init() {
	rootCmd.AddCommand(initCmd)

	initCmd.Flags().BoolP("force", "f", false, "Overwrite existing configuration without prompting")
	initCmd.Flags().BoolP("yes", "y", false, "Non-interactive mode, use AI defaults")
	initCmd.Flags().StringP("config", "c", "", "Path to config file to use")
	initCmd.Flags().StringP("tasks", "t", "", "Path to task file to import")
}

// runInit is the main entry point for the init command.
func runInit(cmd *cobra.Command, args []string) error {
	force, _ := cmd.Flags().GetBool("force")
	yes, _ := cmd.Flags().GetBool("yes")
	configPath, _ := cmd.Flags().GetString("config")
	tasksPath, _ := cmd.Flags().GetString("tasks")

	// Get the current working directory as project dir
	projectDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get working directory: %w", err)
	}

	// Check if .ralph directory already exists
	ralphDir := filepath.Join(projectDir, ".ralph")
	if _, err := os.Stat(ralphDir); err == nil {
		// .ralph exists
		if !force {
			shouldContinue, err := promptReconfigure(cmd, yes)
			if err != nil {
				return err
			}
			if !shouldContinue {
				cmd.Println("Initialization cancelled.")
				return nil
			}
		}
		// Remove existing .ralph directory for reinitialize
		if err := os.RemoveAll(ralphDir); err != nil {
			return fmt.Errorf("failed to remove existing .ralph directory: %w", err)
		}
		cmd.Println("Removed existing .ralph configuration.")
	}

	// If --config is provided, use that config directly
	if configPath != "" {
		return runInitWithConfig(cmd, projectDir, configPath, tasksPath)
	}

	// Run interactive or headless setup
	if yes {
		return runInitHeadless(cmd, projectDir, tasksPath)
	}

	return runInitInteractive(cmd, projectDir, tasksPath)
}

// promptReconfigure prompts the user whether to reconfigure existing .ralph.
func promptReconfigure(cmd *cobra.Command, nonInteractive bool) (bool, error) {
	if nonInteractive {
		// In non-interactive mode without --force, don't overwrite
		return false, fmt.Errorf(".ralph directory already exists, use --force to overwrite")
	}

	cmd.Println("A .ralph directory already exists in this project.")
	cmd.Print("Do you want to reconfigure? This will remove existing settings. [y/N]: ")

	reader := bufio.NewReader(cmd.InOrStdin())
	response, err := reader.ReadString('\n')
	if err != nil {
		return false, fmt.Errorf("failed to read response: %w", err)
	}

	response = strings.TrimSpace(strings.ToLower(response))
	return response == "y" || response == "yes", nil
}

// runInitWithConfig initializes using a provided config file.
func runInitWithConfig(cmd *cobra.Command, projectDir, configPath, tasksPath string) error {
	// Load the provided config
	loader := config.NewLoader()
	cfg, err := loader.LoadConfig(configPath)
	if err != nil {
		return fmt.Errorf("failed to load config from %s: %w", configPath, err)
	}

	// Create .ralph directory structure
	setup := app.NewSetup(projectDir, nil)
	if err := setup.CreateRalphDir(); err != nil {
		return err
	}

	// Save the config to .ralph/config.yaml
	if err := setup.SaveConfig(cfg); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	cmd.Println("Created .ralph/config.yaml from", configPath)

	// Handle tasks if provided
	if tasksPath != "" {
		tasks, err := setup.ImportTasksFromFile(tasksPath)
		if err != nil {
			return fmt.Errorf("failed to import tasks: %w", err)
		}
		if err := setup.SaveTasks(tasks); err != nil {
			return fmt.Errorf("failed to save tasks: %w", err)
		}
		cmd.Printf("Imported %d tasks from %s\n", len(tasks), tasksPath)
	}

	printInitSuccess(cmd)
	return nil
}

// runInitHeadless runs initialization in non-interactive mode with AI defaults.
func runInitHeadless(cmd *cobra.Command, projectDir, tasksPath string) error {
	// Set up cancellation context
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle signals for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigChan
		cancel()
	}()

	// Select agent for analysis
	selectedAgent, err := selectAgentForInit()
	if err != nil {
		return fmt.Errorf("failed to select agent: %w", err)
	}

	cmd.Println("🐺 Initializing Ralph (non-interactive mode)...")
	cmd.Printf("Using agent: %s\n", selectedAgent.Name())

	// Create setup with headless configuration
	setup := app.NewSetup(projectDir, selectedAgent)
	setup.Headless = true
	setup.TasksPath = tasksPath
	setup.LogWriter = cmd.OutOrStdout()
	setup.OnProgress = func(status string) {
		fmt.Fprintf(cmd.OutOrStdout(), "  %s\n", status)
	}

	// Run headless setup
	result, err := setup.RunHeadless(ctx)
	if err != nil {
		return fmt.Errorf("setup failed: %w", err)
	}

	cmd.Printf("\n✓ Project analysis complete\n")
	cmd.Printf("  Type: %s\n", result.Analysis.ProjectType)
	if result.Analysis.Build.Command != nil {
		cmd.Printf("  Build: %s\n", *result.Analysis.Build.Command)
	}
	if result.Analysis.Test.Command != nil {
		cmd.Printf("  Test: %s\n", *result.Analysis.Test.Command)
	}
	cmd.Printf("  Tasks: %d\n", len(result.Tasks))

	printInitSuccess(cmd)
	return nil
}

// runInitInteractive runs initialization with TUI confirmation.
func runInitInteractive(cmd *cobra.Command, projectDir, tasksPath string) error {
	// Set up cancellation context
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle signals for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigChan
		cancel()
	}()

	// Select agent for analysis
	selectedAgent, err := selectAgentForInit()
	if err != nil {
		return fmt.Errorf("failed to select agent: %w", err)
	}

	// Run the TUI setup flow (stops at completion, doesn't start loop)
	result, err := tui.RunSetupTUI(ctx, selectedAgent, projectDir)
	if err != nil {
		return fmt.Errorf("setup failed: %w", err)
	}

	cmd.Printf("\n✓ Setup complete! Found %d tasks.\n", len(result.Tasks))
	printInitSuccess(cmd)
	return nil
}

// selectAgentForInit selects an agent for the init process.
func selectAgentForInit() (agent.Agent, error) {
	registry := agent.NewRegistry()
	registry.Register(cursor.New())
	registry.Register(auggie.New())
	return registry.GetOrDefault("")
}

// printInitSuccess prints the success message after initialization.
func printInitSuccess(cmd *cobra.Command) {
	cmd.Println("")
	cmd.Println("✓ Ralph initialized successfully!")
	cmd.Println("")
	cmd.Println("Next steps:")
	cmd.Println("  1. Review .ralph/config.yaml to customize settings")
	cmd.Println("  2. Run 'ralph run' to start the task loop")
}
