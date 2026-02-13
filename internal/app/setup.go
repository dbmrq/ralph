// Package app provides the main application orchestration for ralph.
// This package handles the first-run setup flow and application lifecycle.
package app

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/wexinc/ralph/internal/agent"
	"github.com/wexinc/ralph/internal/build"
	"github.com/wexinc/ralph/internal/config"
	"github.com/wexinc/ralph/internal/task"
)

// SetupResult contains the results of the setup flow.
type SetupResult struct {
	// Config is the confirmed configuration.
	Config *config.Config
	// Analysis is the confirmed project analysis.
	Analysis *build.ProjectAnalysis
	// Tasks is the list of imported/generated tasks.
	Tasks []*task.Task
	// TasksPath is the path where tasks were saved.
	TasksPath string
}

// SetupProgressFunc is called with progress updates during setup.
type SetupProgressFunc func(status string)

// Setup orchestrates the first-run setup flow.
type Setup struct {
	// ProjectDir is the root directory of the project.
	ProjectDir string
	// Agent is the AI agent to use for analysis and task generation.
	Agent agent.Agent
	// Model is the model to use (optional).
	Model string
	// OnProgress is called with status updates.
	OnProgress SetupProgressFunc
	// LogWriter receives real-time agent output (optional).
	LogWriter io.Writer
	// Headless indicates whether to skip interactive prompts.
	Headless bool
	// TasksPath is the path to import tasks from (for headless mode).
	TasksPath string
}

// NewSetup creates a new Setup orchestrator.
func NewSetup(projectDir string, ag agent.Agent) *Setup {
	return &Setup{
		ProjectDir: projectDir,
		Agent:      ag,
		OnProgress: func(status string) {}, // noop by default
	}
}

// NeedsSetup returns true if the project needs setup (no .ralph directory).
func NeedsSetup(projectDir string) bool {
	ralphDir := filepath.Join(projectDir, ".ralph")
	_, err := os.Stat(ralphDir)
	return os.IsNotExist(err)
}

// CreateRalphDir creates the .ralph directory structure.
func (s *Setup) CreateRalphDir() error {
	ralphDir := filepath.Join(s.ProjectDir, ".ralph")
	
	// Create main directory
	if err := os.MkdirAll(ralphDir, 0755); err != nil {
		return fmt.Errorf("failed to create .ralph directory: %w", err)
	}

	// Create subdirectories
	subdirs := []string{
		"sessions",
		"logs",
	}
	for _, subdir := range subdirs {
		path := filepath.Join(ralphDir, subdir)
		if err := os.MkdirAll(path, 0755); err != nil {
			return fmt.Errorf("failed to create %s directory: %w", subdir, err)
		}
	}

	s.report("Created .ralph directory structure")
	return nil
}

// RunAnalysis runs the Project Analysis Agent.
func (s *Setup) RunAnalysis(ctx context.Context) (*build.ProjectAnalysis, error) {
	s.report("Analyzing project...")
	
	analyzer := build.NewProjectAnalyzer(s.ProjectDir, s.Agent)
	analyzer.Model = s.Model
	analyzer.LogWriter = s.LogWriter
	analyzer.OnProgress = func(status string) {
		s.report(status)
	}
	
	analysis, err := analyzer.AnalyzeWithFallback(ctx)
	if err != nil {
		return nil, fmt.Errorf("analysis failed: %w", err)
	}
	
	return analysis, nil
}

// SaveAnalysis saves the analysis to the cache file.
func (s *Setup) SaveAnalysis(analysis *build.ProjectAnalysis) error {
	analyzer := build.NewProjectAnalyzer(s.ProjectDir, s.Agent)
	analyzer.Model = s.Model
	return analyzer.SaveCache(analysis)
}

// DetectTasks detects existing task lists in the project.
func (s *Setup) DetectTasks() *task.TaskListDetection {
	initializer := task.NewInitializer(s.ProjectDir, s.Agent)
	return initializer.DetectTaskList()
}

// ImportTasks imports tasks from a detection or file path.
func (s *Setup) ImportTasks(ctx context.Context, detection *task.TaskListDetection) ([]*task.Task, error) {
	initializer := task.NewInitializer(s.ProjectDir, s.Agent)
	initializer.Model = s.Model
	initializer.LogWriter = s.LogWriter
	initializer.OnProgress = func(status string) {
		s.report(status)
	}
	
	result, err := initializer.ImportFromDetection(detection)
	if err != nil {
		return nil, err
	}
	return result.Tasks, nil
}

// ImportTasksFromFile imports tasks from a file path.
func (s *Setup) ImportTasksFromFile(path string) ([]*task.Task, error) {
	initializer := task.NewInitializer(s.ProjectDir, s.Agent)
	initializer.OnProgress = func(status string) {
		s.report(status)
	}
	
	result, err := initializer.ImportFromFile(path)
	if err != nil {
		return nil, err
	}
	return result.Tasks, nil
}

// GenerateTasks generates tasks from a goal description.
func (s *Setup) GenerateTasks(ctx context.Context, goal string) ([]*task.Task, error) {
	initializer := task.NewInitializer(s.ProjectDir, s.Agent)
	initializer.Model = s.Model
	initializer.LogWriter = s.LogWriter
	initializer.OnProgress = func(status string) {
		s.report(status)
	}

	result, err := initializer.GenerateFromGoal(ctx, goal)
	if err != nil {
		return nil, err
	}
	return result.Tasks, nil
}

// SaveTasks saves tasks to the .ralph/tasks.json file.
func (s *Setup) SaveTasks(tasks []*task.Task) error {
	storePath := filepath.Join(s.ProjectDir, ".ralph", "tasks.json")
	initializer := task.NewInitializer(s.ProjectDir, s.Agent)
	return initializer.SaveToStore(tasks, storePath)
}

// SaveConfig saves the configuration to .ralph/config.yaml.
func (s *Setup) SaveConfig(cfg *config.Config) error {
	configPath := filepath.Join(s.ProjectDir, ".ralph", "config.yaml")
	return config.Save(cfg, configPath)
}

// BuildConfigFromAnalysis creates a Config from the analysis results.
func (s *Setup) BuildConfigFromAnalysis(analysis *build.ProjectAnalysis) *config.Config {
	cfg := config.NewConfig()

	// Set build command if detected
	if analysis.Build.Command != nil {
		cfg.Build.Command = *analysis.Build.Command
	}

	// Set test command if detected
	if analysis.Test.Command != nil {
		cfg.Test.Command = *analysis.Test.Command
	}

	// Set agent info if available
	if s.Agent != nil {
		cfg.Agent.Default = s.Agent.Name()
	}
	if s.Model != "" {
		cfg.Agent.Model = s.Model
	}

	return cfg
}

// RunHeadless runs the setup flow in headless mode.
// It requires either a tasks file or an existing task list in the project.
func (s *Setup) RunHeadless(ctx context.Context) (*SetupResult, error) {
	// Create .ralph directory
	if err := s.CreateRalphDir(); err != nil {
		return nil, err
	}

	// Run analysis
	analysis, err := s.RunAnalysis(ctx)
	if err != nil {
		return nil, fmt.Errorf("analysis failed: %w", err)
	}

	// Log analysis results
	s.logAnalysis(analysis)

	// Save analysis
	if err := s.SaveAnalysis(analysis); err != nil {
		// Non-fatal, just log
		s.report(fmt.Sprintf("Warning: failed to cache analysis: %v", err))
	}

	// Handle tasks
	var tasks []*task.Task
	var tasksPath string

	if s.TasksPath != "" {
		// Import from specified file
		tasks, err = s.ImportTasksFromFile(s.TasksPath)
		if err != nil {
			return nil, fmt.Errorf("failed to import tasks from %s: %w", s.TasksPath, err)
		}
		tasksPath = s.TasksPath
	} else if analysis.TaskList.Detected {
		// Use detected task list
		detection := &task.TaskListDetection{
			Detected:  analysis.TaskList.Detected,
			Path:      analysis.TaskList.Path,
			Format:    analysis.TaskList.Format,
			TaskCount: analysis.TaskList.TaskCount,
		}
		tasks, err = s.ImportTasks(ctx, detection)
		if err != nil {
			return nil, fmt.Errorf("failed to import detected tasks: %w", err)
		}
		tasksPath = analysis.TaskList.Path
	} else {
		return nil, fmt.Errorf("no tasks found: use --tasks flag to specify a task file")
	}

	// Save tasks
	if err := s.SaveTasks(tasks); err != nil {
		return nil, fmt.Errorf("failed to save tasks: %w", err)
	}

	// Build and save config
	cfg := s.BuildConfigFromAnalysis(analysis)
	if err := s.SaveConfig(cfg); err != nil {
		// Non-fatal, use defaults
		s.report(fmt.Sprintf("Warning: failed to save config: %v", err))
	}

	return &SetupResult{
		Config:    cfg,
		Analysis:  analysis,
		Tasks:     tasks,
		TasksPath: tasksPath,
	}, nil
}

// logAnalysis logs the analysis results for headless mode.
func (s *Setup) logAnalysis(analysis *build.ProjectAnalysis) {
	s.report(fmt.Sprintf("Project Analysis:"))
	s.report(fmt.Sprintf("  Type: %s", analysis.ProjectType))
	if len(analysis.Languages) > 0 {
		s.report(fmt.Sprintf("  Languages: %v", analysis.Languages))
	}
	if analysis.Build.Command != nil {
		readyStr := "not ready"
		if analysis.Build.Ready {
			readyStr = "ready"
		}
		s.report(fmt.Sprintf("  Build: %s (%s)", *analysis.Build.Command, readyStr))
	}
	if analysis.Test.Command != nil {
		readyStr := "not ready"
		if analysis.Test.Ready {
			readyStr = "ready"
		}
		s.report(fmt.Sprintf("  Test: %s (%s)", *analysis.Test.Command, readyStr))
	}
	if analysis.IsGreenfield {
		s.report("  Greenfield: yes")
	}
	if analysis.TaskList.Detected {
		s.report(fmt.Sprintf("  Task list found: %s (%d tasks)", analysis.TaskList.Path, analysis.TaskList.TaskCount))
	}
}

// report calls the progress callback.
func (s *Setup) report(status string) {
	if s.OnProgress != nil {
		s.OnProgress(status)
	}
}

