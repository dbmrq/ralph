package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"github.com/wexinc/ralph/internal/agent"
	"github.com/wexinc/ralph/internal/agent/auggie"
	"github.com/wexinc/ralph/internal/agent/cursor"
	"github.com/wexinc/ralph/internal/app"
	"github.com/wexinc/ralph/internal/config"
	"github.com/wexinc/ralph/internal/hooks"
	"github.com/wexinc/ralph/internal/loop"
	"github.com/wexinc/ralph/internal/task"
	"github.com/wexinc/ralph/internal/tui"
	"github.com/wexinc/ralph/internal/version"
)

// runCmd represents the run command.
var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Start the Ralph loop to execute tasks",
	Long: `Start the Ralph loop to execute tasks from the task list.

By default, Ralph runs in TUI mode with an interactive terminal interface.
Use --headless for non-interactive execution (e.g., in CI/GitHub Actions).

Examples:
  ralph run                    # Start in TUI mode
  ralph run --headless         # Start in headless mode
  ralph run --headless --output json  # Headless with JSON output
  ralph run --headless --tasks ./TASKS.md  # Use specific task file
  ralph run --continue <id>    # Resume a paused session`,
	RunE: runRun,
}

func init() {
	rootCmd.AddCommand(runCmd)

	runCmd.Flags().Bool("headless", false, "Run in headless mode without TUI")
	runCmd.Flags().String("output", "", "Output format: json for structured output (requires --headless)")
	runCmd.Flags().String("continue", "", "Continue a paused session by ID")
	runCmd.Flags().String("tasks", "", "Path to task file (required for headless mode if no tasks.json exists)")
	runCmd.Flags().BoolP("verbose", "v", false, "Enable verbose output")
}

// runRun is the main entry point for the run command.
func runRun(cmd *cobra.Command, args []string) error {
	headless, _ := cmd.Flags().GetBool("headless")
	outputFormat, _ := cmd.Flags().GetString("output")
	continueID, _ := cmd.Flags().GetString("continue")
	tasksPath, _ := cmd.Flags().GetString("tasks")
	verbose, _ := cmd.Flags().GetBool("verbose")

	if outputFormat != "" && !headless {
		return fmt.Errorf("--output flag requires --headless mode")
	}

	// Check for updates in the background (non-blocking)
	go checkUpdateBackground(cmd)

	// Get the current working directory as project dir
	projectDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get working directory: %w", err)
	}

	// Check if setup is needed (no .ralph directory)
	if app.NeedsSetup(projectDir) {
		if headless {
			return runHeadlessSetup(cmd, projectDir, outputFormat, tasksPath, verbose)
		}
		return runTUISetup(cmd, projectDir, continueID, tasksPath, verbose)
	}

	if headless {
		return runHeadless(cmd, projectDir, outputFormat, continueID, tasksPath, verbose)
	}

	// TUI mode (placeholder - will be implemented in TUI-006)
	cmd.Println("Starting Ralph in TUI mode...")
	if continueID != "" {
		cmd.Printf("Continuing session: %s\n", continueID)
	}
	cmd.Println("TUI mode not yet implemented. Use --headless for now.")
	return nil
}

// runTUISetup runs the first-run setup flow in TUI mode.
func runTUISetup(cmd *cobra.Command, projectDir, continueID, tasksPath string, verbose bool) error {
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

	// Initialize agent for setup
	selectedAgent, err := selectAgentForSetup()
	if err != nil {
		return fmt.Errorf("failed to select agent: %w", err)
	}

	// Run the TUI setup flow
	result, err := tui.RunSetupTUI(ctx, selectedAgent, projectDir)
	if err != nil {
		return fmt.Errorf("setup failed: %w", err)
	}

	cmd.Printf("Setup complete! Found %d tasks.\n", len(result.Tasks))
	cmd.Println("Starting Ralph loop...")

	// Continue to run the loop with the setup result
	return runWithSetupResult(cmd, projectDir, result, continueID, verbose)
}

// runHeadlessSetup runs the first-run setup flow in headless mode.
func runHeadlessSetup(cmd *cobra.Command, projectDir, outputFormat, tasksPath string, verbose bool) error {
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

	// Initialize agent for setup
	selectedAgent, err := selectAgentForSetup()
	if err != nil {
		return fmt.Errorf("failed to select agent: %w", err)
	}

	// Create setup with headless configuration
	setup := app.NewSetup(projectDir, selectedAgent)
	setup.Headless = true
	setup.TasksPath = tasksPath
	setup.LogWriter = cmd.OutOrStdout()
	setup.OnProgress = func(status string) {
		if verbose {
			fmt.Fprintf(cmd.OutOrStdout(), "[setup] %s\n", status)
		}
	}

	// Run headless setup
	result, err := setup.RunHeadless(ctx)
	if err != nil {
		return fmt.Errorf("setup failed: %w", err)
	}

	cmd.Printf("Setup complete! Found %d tasks.\n", len(result.Tasks))

	// Continue to run the loop with the setup result
	return runHeadlessWithSetupResult(cmd, projectDir, result, outputFormat, verbose)
}

// selectAgentForSetup selects an agent for the setup process.
// It uses GetOrDefault to automatically select the first available agent.
func selectAgentForSetup() (agent.Agent, error) {
	registry := agent.NewRegistry()
	registry.Register(cursor.New())
	registry.Register(auggie.New())
	return registry.GetOrDefault("")
}

// runWithSetupResult runs the main loop after setup completes (TUI mode).
func runWithSetupResult(cmd *cobra.Command, projectDir string, result *app.SetupResult, continueID string, verbose bool) error {
	// For now, fall through to TUI mode placeholder
	// Full TUI loop will be implemented in TUI-006
	if continueID != "" {
		cmd.Printf("Continuing session: %s\n", continueID)
	}
	cmd.Println("TUI mode not yet fully implemented. Use --headless for now.")
	return nil
}

// runHeadlessWithSetupResult runs the main loop after setup completes (headless mode).
func runHeadlessWithSetupResult(cmd *cobra.Command, projectDir string, result *app.SetupResult, outputFormat string, verbose bool) error {
	// Configure headless output
	headlessConfig := loop.DefaultHeadlessConfig()
	headlessConfig.Writer = cmd.OutOrStdout()
	headlessConfig.ErrorWriter = cmd.ErrOrStderr()
	headlessConfig.Verbose = verbose

	if outputFormat == "json" {
		headlessConfig.OutputFormat = loop.OutputFormatJSON
	}

	runner := loop.NewHeadlessRunner(headlessConfig)

	// Use the config from setup result
	cfg := result.Config

	// Initialize agent
	selectedAgent, err := selectAgent(cfg)
	if err != nil {
		return fmt.Errorf("failed to select agent: %w", err)
	}

	// Create task manager from setup tasks
	storePath := filepath.Join(projectDir, ".ralph", "tasks.json")
	store := task.NewStore(storePath)
	taskMgr := task.NewManager(store)
	if err := taskMgr.Load(); err != nil {
		return fmt.Errorf("failed to load tasks: %w", err)
	}

	// Create hook manager (optional)
	var hookMgr *hooks.Manager
	if len(cfg.Hooks.PreTask) > 0 || len(cfg.Hooks.PostTask) > 0 {
		hookMgr, err = hooks.NewManagerFromConfig(&cfg.Hooks)
		if err != nil {
			return fmt.Errorf("failed to create hook manager: %w", err)
		}
	}

	// Create the main loop
	mainLoop := loop.NewLoop(selectedAgent, taskMgr, hookMgr, cfg, projectDir)

	// Configure loop options
	loopOpts := loop.DefaultOptions()
	loopOpts.OnEvent = runner.HandleEvent
	loopOpts.LogWriter = cmd.OutOrStdout()
	mainLoop.SetOptions(loopOpts)

	// Set up cancellation context
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigChan
		cancel()
	}()

	// Run the loop
	sessionID := loop.GenerateSessionID()
	loopErr := mainLoop.Run(ctx, sessionID)

	// Output final results
	loopCtx := mainLoop.Context()
	if loopCtx != nil {
		if headlessConfig.OutputFormat == loop.OutputFormatJSON {
			if err := runner.WriteJSONOutput(loopCtx); err != nil {
				fmt.Fprintf(cmd.ErrOrStderr(), "Failed to write JSON output: %v\n", err)
			}
		} else {
			runner.PrintSummary(loopCtx)
		}
	}

	return loopErr
}

// runHeadless executes ralph in headless mode for CI/GitHub Actions.
func runHeadless(cmd *cobra.Command, projectDir, outputFormat, continueID, tasksPath string, verbose bool) error {
	// Configure headless output
	headlessConfig := loop.DefaultHeadlessConfig()
	headlessConfig.Writer = cmd.OutOrStdout()
	headlessConfig.ErrorWriter = cmd.ErrOrStderr()
	headlessConfig.Verbose = verbose

	if outputFormat == "json" {
		headlessConfig.OutputFormat = loop.OutputFormatJSON
	}

	runner := loop.NewHeadlessRunner(headlessConfig)

	// Load configuration
	cfg, err := loadConfig(projectDir)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Initialize agent registry and select agent
	selectedAgent, err := selectAgent(cfg)
	if err != nil {
		return fmt.Errorf("failed to select agent: %w", err)
	}

	// Load task manager
	taskMgr, err := loadTasks(projectDir, tasksPath)
	if err != nil {
		return fmt.Errorf("failed to load tasks: %w", err)
	}

	// Create hook manager (optional - hooks may not be configured)
	var hookMgr *hooks.Manager
	if len(cfg.Hooks.PreTask) > 0 || len(cfg.Hooks.PostTask) > 0 {
		hookMgr, err = hooks.NewManagerFromConfig(&cfg.Hooks)
		if err != nil {
			return fmt.Errorf("failed to create hook manager: %w", err)
		}
	}

	// Create the main loop
	mainLoop := loop.NewLoop(selectedAgent, taskMgr, hookMgr, cfg, projectDir)

	// Configure loop options with headless event handler
	loopOpts := loop.DefaultOptions()
	loopOpts.OnEvent = runner.HandleEvent
	loopOpts.LogWriter = cmd.OutOrStdout() // Agent output to stdout
	mainLoop.SetOptions(loopOpts)

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

	// Run or resume the loop
	var loopErr error
	if continueID != "" {
		loopErr = mainLoop.Resume(ctx, continueID)
	} else {
		sessionID := loop.GenerateSessionID()
		loopErr = mainLoop.Run(ctx, sessionID)
	}

	// Output final results
	loopCtx := mainLoop.Context()
	if loopCtx != nil {
		if headlessConfig.OutputFormat == loop.OutputFormatJSON {
			if err := runner.WriteJSONOutput(loopCtx); err != nil {
				fmt.Fprintf(cmd.ErrOrStderr(), "Failed to write JSON output: %v\n", err)
			}
		} else {
			runner.PrintSummary(loopCtx)
		}
	}

	return loopErr
}

// loadConfig loads the ralph configuration from the project directory.
func loadConfig(projectDir string) (*config.Config, error) {
	configPath := filepath.Join(projectDir, ".ralph", "config.yaml")

	// Check if config file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		// Use default config if no config file exists
		return config.NewConfig(), nil
	}

	loader := config.NewLoader()
	return loader.LoadConfig(configPath)
}

// selectAgent initializes the agent registry and selects an agent.
func selectAgent(cfg *config.Config) (agent.Agent, error) {
	registry := agent.NewRegistry()

	// Register built-in agents
	registry.Register(cursor.New())
	registry.Register(auggie.New())

	// Select agent based on config or availability
	agentName := cfg.Agent.Default
	return registry.SelectAgent(agentName)
}

// loadTasks loads the task manager from the project directory.
func loadTasks(projectDir, tasksPath string) (*task.Manager, error) {
	ralphDir := filepath.Join(projectDir, ".ralph")

	// Determine task store path
	var storePath string
	if tasksPath != "" {
		// If a specific tasks file is provided, import it
		storePath = filepath.Join(ralphDir, "tasks.json")
		store := task.NewStore(storePath)
		mgr := task.NewManager(store)

		// Import tasks from the provided file
		if err := importTasks(mgr, tasksPath); err != nil {
			return nil, fmt.Errorf("failed to import tasks from %s: %w", tasksPath, err)
		}
		return mgr, nil
	}

	// Use default tasks.json location
	storePath = filepath.Join(ralphDir, "tasks.json")
	if _, err := os.Stat(storePath); os.IsNotExist(err) {
		return nil, fmt.Errorf("no tasks found: run 'ralph init' or use --tasks flag")
	}

	store := task.NewStore(storePath)
	mgr := task.NewManager(store)
	if err := mgr.Load(); err != nil {
		return nil, fmt.Errorf("failed to load tasks: %w", err)
	}

	return mgr, nil
}

// importTasks imports tasks from a markdown or text file.
func importTasks(mgr *task.Manager, path string) error {
	importer := task.NewImporter()

	// Auto-detect format based on file extension
	format := task.FormatMarkdown
	if filepath.Ext(path) == ".txt" {
		format = task.FormatPlainText
	}

	result, err := importer.ImportFromFile(path, format)
	if err != nil {
		return err
	}

	for _, t := range result.Tasks {
		if err := mgr.AddTask(t); err != nil {
			return fmt.Errorf("failed to add task %s: %w", t.ID, err)
		}
	}

	return mgr.Save()
}

// checkUpdateBackground checks for updates in the background and displays a notification.
// This is non-blocking and runs in a goroutine.
func checkUpdateBackground(cmd *cobra.Command) {
	// Use a short timeout for background check
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Skip if we're in dev mode
	if Version == "dev" {
		return
	}

	checker := version.NewChecker()
	release, err := checker.CheckForUpdate(ctx, Version)
	if err != nil {
		// Silently ignore errors in background check
		return
	}

	if release != nil {
		// Print update notification (non-blocking, happens async)
		fmt.Fprintf(cmd.ErrOrStderr(), "\n💡 Update available: %s → %s (run 'ralph update')\n\n",
			Version, release.TagName)
	}
}
