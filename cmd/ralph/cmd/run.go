package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"

	"github.com/dbmrq/ralph/internal/agent"
	"github.com/dbmrq/ralph/internal/agent/auggie"
	"github.com/dbmrq/ralph/internal/agent/cursor"
	"github.com/dbmrq/ralph/internal/app"
	"github.com/dbmrq/ralph/internal/config"
	"github.com/dbmrq/ralph/internal/hooks"
	"github.com/dbmrq/ralph/internal/logging"
	"github.com/dbmrq/ralph/internal/loop"
	"github.com/dbmrq/ralph/internal/project"
	"github.com/dbmrq/ralph/internal/task"
	"github.com/dbmrq/ralph/internal/tui"
	"github.com/dbmrq/ralph/internal/tui/components"
	"github.com/dbmrq/ralph/internal/version"
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
	runCmd.Flags().Bool("init-only", false, "Only run setup, do not start the task loop")
}

// runRun is the main entry point for the run command.
func runRun(cmd *cobra.Command, args []string) error {
	headless, _ := cmd.Flags().GetBool("headless")
	outputFormat, _ := cmd.Flags().GetString("output")
	continueID, _ := cmd.Flags().GetString("continue")
	tasksPath, _ := cmd.Flags().GetString("tasks")
	verbose, _ := cmd.Flags().GetBool("verbose")
	initOnly, _ := cmd.Flags().GetBool("init-only")

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

	// Check if we need to prompt for directory selection (TUI mode only)
	if !headless {
		projectDir, err = detectOrSelectProject(cmd, projectDir)
		if err != nil {
			return err
		}
	}

	// Initialize logging
	logLevel := logging.LevelInfo
	if verbose {
		logLevel = logging.LevelDebug
	}
	logConfig := &logging.Config{
		Level:       logLevel,
		LogDir:      filepath.Join(projectDir, ".ralph", "logs"),
		MaxLogFiles: 10,
		MaxLogAge:   7 * 24 * time.Hour,
		Console:     false, // Don't mix console output with TUI
		JSONFormat:  false,
	}
	if err := logging.InitGlobal(logConfig); err != nil {
		// Non-fatal: warn but continue without file logging
		fmt.Fprintf(cmd.ErrOrStderr(), "Warning: failed to initialize logging: %v\n", err)
	} else {
		defer func() { _ = logging.CloseGlobal() }()
		logging.Info("Ralph starting", "version", Version, "verbose", verbose)
	}

	// Check if setup is needed (no .ralph directory) or legacy .ralph detected
	needsSetup := app.NeedsSetup(projectDir)
	isLegacy := app.IsLegacyRalph(projectDir)

	if needsSetup || isLegacy {
		if headless {
			return runHeadlessSetup(cmd, projectDir, outputFormat, tasksPath, verbose, initOnly)
		}
		return runTUISetup(cmd, projectDir, continueID, tasksPath, verbose, initOnly, isLegacy)
	}

	// If --init-only was specified but setup already done, just exit
	if initOnly {
		cmd.Println("Project already initialized (.ralph directory exists)")
		return nil
	}

	if headless {
		return runHeadless(cmd, projectDir, outputFormat, continueID, tasksPath, verbose)
	}

	// TUI mode
	return runTUI(cmd, projectDir, continueID, tasksPath, verbose)
}

// runTUI executes ralph in TUI mode with interactive terminal interface.
func runTUI(cmd *cobra.Command, projectDir, continueID, tasksPath string, verbose bool) error {
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

	// Get all tasks for the TUI
	tasks := taskMgr.All()

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

	// Determine session ID
	var sessionID string
	if continueID != "" {
		sessionID = continueID
	} else {
		sessionID = loop.GenerateSessionID()
	}

	// Create TUI runner with session info
	sessionInfo := tui.SessionInfo{
		ProjectName: filepath.Base(projectDir),
		AgentName:   selectedAgent.Name(),
		ModelName:   cfg.Agent.Model,
		SessionID:   sessionID,
	}
	tuiRunner := tui.NewTUIRunner(mainLoop, tasks, sessionInfo)

	// Configure loop options with TUI event handling
	loopOpts := loop.DefaultOptions()
	tuiRunner.ConfigureLoop(loopOpts)
	mainLoop.SetOptions(loopOpts)

	// Create loop controller adapter and set it on the model
	controller := NewLoopControllerAdapter(mainLoop, cancel)
	tuiRunner.Model().SetLoopController(controller)

	// Run TUI and Loop concurrently
	var loopErr error
	if continueID != "" {
		loopErr = tuiRunner.Run(func() error {
			return mainLoop.Resume(ctx, continueID)
		})
	} else {
		loopErr = tuiRunner.Run(func() error {
			return mainLoop.Run(ctx, sessionID)
		})
	}

	return loopErr
}

// runTUISetup runs the first-run setup flow in TUI mode with seamless transition to loop.
// If initOnly is true, exits after setup without starting the task loop.
// If isLegacy is true, shows the legacy migration screen first.
func runTUISetup(cmd *cobra.Command, projectDir, continueID, tasksPath string, verbose, initOnly, isLegacy bool) error {
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

	// Initialize agent for setup (may fail if no agents available)
	selectedAgent, err := selectAgentForSetup()
	noAgents := err != nil

	// Build setup options based on detected edge cases
	setupOpts := tui.SetupTUIOptions{
		IsLegacy: isLegacy,
		NoAgents: noAgents,
	}

	// If initOnly, use the standard setup TUI that exits after completion
	if initOnly {
		result, err := tui.RunSetupTUIWithOptions(ctx, selectedAgent, projectDir, setupOpts)
		if err != nil {
			return fmt.Errorf("setup failed: %w", err)
		}
		cmd.Printf("\n✓ Setup complete! Found %d tasks.\n", len(result.Tasks))
		cmd.Println("\nRun 'ralph run' to start the task loop.")
		return nil
	}

	// Determine session ID
	var sessionID string
	if continueID != "" {
		sessionID = continueID
	} else {
		sessionID = loop.GenerateSessionID()
	}

	// Variables to hold loop components created after setup
	var mainLoop *loop.Loop
	var loopController *LoopControllerAdapter

	// Session info will be populated after setup
	// Agent name may be empty if no agents are available
	agentName := ""
	if selectedAgent != nil {
		agentName = selectedAgent.Name()
	}
	sessionInfo := tui.SessionInfo{
		ProjectName: filepath.Base(projectDir),
		AgentName:   agentName,
		SessionID:   sessionID,
	}

	// Run combined setup-to-loop TUI
	result, err := tui.RunCombinedTUIWithOptions(
		ctx,
		selectedAgent,
		projectDir,
		nil, // tasks come from setup result
		sessionInfo,
		func(setupResult *app.SetupResult, loopModel *tui.Model, program *tea.Program) error {
			// This function is called after setup completes, inside the TUI
			// We need to create and run the loop here

			if setupResult == nil {
				return fmt.Errorf("setup did not complete")
			}

			// Use the config from setup result
			cfg := setupResult.Config

			// Update session info with model from config (agent may have been selected during setup)
			agentNameForLoop := agentName
			if selectedAgent != nil {
				agentNameForLoop = selectedAgent.Name()
			}
			loopModel.SetSessionInfo(
				sessionInfo.ProjectName,
				agentNameForLoop,
				cfg.Agent.Model,
				sessionID,
			)

			// Create task manager from setup tasks
			storePath := filepath.Join(projectDir, ".ralph", "tasks.json")
			store := task.NewStore(storePath)
			taskMgr := task.NewManager(store)
			if err := taskMgr.Load(); err != nil {
				return fmt.Errorf("failed to load tasks: %w", err)
			}

			// Set tasks on the loop model
			loopModel.SetTasks(taskMgr.All())

			// Create hook manager (optional)
			var hookMgr *hooks.Manager
			if len(cfg.Hooks.PreTask) > 0 || len(cfg.Hooks.PostTask) > 0 {
				hookMgr, err = hooks.NewManagerFromConfig(&cfg.Hooks)
				if err != nil {
					return fmt.Errorf("failed to create hook manager: %w", err)
				}
			}

			// Create the main loop
			mainLoop = loop.NewLoop(selectedAgent, taskMgr, hookMgr, cfg, projectDir)

			// Set analysis from setup result
			if setupResult.Analysis != nil {
				mainLoop.SetAnalysis(setupResult.Analysis)
			}

			// Create event handler and output writer
			eventHandler := tui.NewTUIEventHandler(program)
			eventHandler.SetTasks(taskMgr.All())
			outputWriter := tui.NewTUIOutputWriter(program)

			// Configure loop options with TUI event handling
			loopOpts := loop.DefaultOptions()
			loopOpts.OnEvent = eventHandler.HandleEvent
			loopOpts.LogWriter = outputWriter
			mainLoop.SetOptions(loopOpts)

			// Create loop controller adapter and set it on the model
			loopController = NewLoopControllerAdapter(mainLoop, cancel)
			loopModel.SetLoopController(loopController)

			// Run the loop
			if continueID != "" {
				return mainLoop.Resume(ctx, continueID)
			}
			return mainLoop.Run(ctx, sessionID)
		},
		setupOpts,
	)

	if err != nil {
		return fmt.Errorf("TUI failed: %w", err)
	}

	if result.Canceled {
		return fmt.Errorf("setup canceled")
	}

	if result.Error != nil {
		return result.Error
	}

	return nil
}

// runHeadlessSetup runs the first-run setup flow in headless mode.
// If initOnly is true, exits after setup without starting the task loop.
func runHeadlessSetup(cmd *cobra.Command, projectDir, outputFormat, tasksPath string, verbose, initOnly bool) error {
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

	// If initOnly, just exit after setup
	if initOnly {
		cmd.Println("Run 'ralph run' to start the task loop.")
		return nil
	}

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

// LoopControllerAdapter adapts the Loop to the TUI's LoopController interface.
// This bridges the TUI's control interface (pause/resume/skip/abort) to the Loop methods.
type LoopControllerAdapter struct {
	loop       *loop.Loop
	cancelFunc context.CancelFunc
}

// NewLoopControllerAdapter creates a new adapter.
func NewLoopControllerAdapter(l *loop.Loop, cancelFunc context.CancelFunc) *LoopControllerAdapter {
	return &LoopControllerAdapter{
		loop:       l,
		cancelFunc: cancelFunc,
	}
}

// Pause pauses the loop after the current task.
func (a *LoopControllerAdapter) Pause() error {
	return a.loop.Pause()
}

// Resume resumes a paused loop.
func (a *LoopControllerAdapter) Resume() error {
	ctx := a.loop.Context()
	if ctx == nil {
		return fmt.Errorf("loop has no context")
	}
	if ctx.State != loop.StatePaused {
		return fmt.Errorf("loop is not paused (current state: %s)", ctx.State)
	}
	return ctx.Transition(loop.StateRunning)
}

// Skip skips the current or specified task.
// The loop will process this request at its next check point.
func (a *LoopControllerAdapter) Skip(taskID string) error {
	if a.loop == nil {
		return fmt.Errorf("loop not initialized")
	}
	return a.loop.Skip(taskID)
}

// Abort aborts the loop cleanly.
// The loop will save state and exit at its next check point.
func (a *LoopControllerAdapter) Abort() error {
	if a.loop == nil {
		// Fallback to context cancellation if loop is nil
		if a.cancelFunc != nil {
			a.cancelFunc()
		}
		return fmt.Errorf("loop not initialized")
	}
	// Use the loop's Abort method for clean shutdown
	if err := a.loop.Abort(""); err != nil {
		// Fallback to context cancellation if abort fails
		if a.cancelFunc != nil {
			a.cancelFunc()
		}
		return err
	}
	return nil
}

// detectOrSelectProject checks if the current directory is a valid project.
// If not, it shows a directory picker TUI for the user to select a project.
func detectOrSelectProject(cmd *cobra.Command, currentDir string) (string, error) {
	detector := project.NewDetector()

	// Check if we should prompt for directory selection
	if !detector.ShouldPromptForDirectory(currentDir) {
		// Current directory is a valid project, use it
		return currentDir, nil
	}

	// Load recent projects
	recent, err := project.LoadRecentProjects()
	if err != nil {
		// Non-fatal: continue without recent projects
		recent = &project.RecentProjects{}
	}

	// Create and run the directory picker TUI
	picker := components.NewDirPicker()
	picker.Init(currentDir, recent)

	model := &dirPickerModel{
		picker: picker,
	}

	p := tea.NewProgram(model, tea.WithAltScreen())
	finalModel, err := p.Run()
	if err != nil {
		return "", fmt.Errorf("directory picker error: %w", err)
	}

	result, ok := finalModel.(*dirPickerModel)
	if !ok {
		return "", fmt.Errorf("unexpected model type from directory picker")
	}
	if result.canceled {
		return "", fmt.Errorf("directory selection canceled")
	}

	if result.selectedPath == "" {
		return "", fmt.Errorf("no directory selected")
	}

	// Update recent projects with the selected project
	if result.selectedProject != nil {
		recent.Add(result.selectedProject)
		if err := recent.Save(); err != nil {
			// Non-fatal: warn but continue
			fmt.Fprintf(cmd.ErrOrStderr(), "Warning: failed to save recent projects: %v\n", err)
		}
	}

	return result.selectedPath, nil
}

// dirPickerModel wraps the DirPicker component for standalone TUI execution.
type dirPickerModel struct {
	picker          *components.DirPicker
	selectedPath    string
	selectedProject *project.ProjectInfo
	canceled        bool
}

func (m *dirPickerModel) Init() tea.Cmd {
	return nil
}

func (m *dirPickerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			m.canceled = true
			return m, tea.Quit
		}
	case tea.WindowSizeMsg:
		m.picker.SetSize(msg.Width, msg.Height)
		return m, nil
	case components.DirSelectedMsg:
		m.selectedPath = msg.Path
		m.selectedProject = msg.Project
		return m, tea.Quit
	case components.DirCanceledMsg:
		m.canceled = true
		return m, tea.Quit
	}

	var cmd tea.Cmd
	m.picker, cmd = m.picker.Update(msg)
	return m, cmd
}

func (m *dirPickerModel) View() string {
	return m.picker.View()
}
