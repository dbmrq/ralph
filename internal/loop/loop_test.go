package loop

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/wexinc/ralph/internal/agent"
	"github.com/wexinc/ralph/internal/build"
	"github.com/wexinc/ralph/internal/config"
	"github.com/wexinc/ralph/internal/task"
)

// mockAgent implements agent.Agent for testing.
type mockAgent struct {
	name        string
	runResult   agent.Result
	runError    error
	runCount    int
	lastPrompt  string
	lastOptions agent.RunOptions
}

func (m *mockAgent) Name() string                       { return m.name }
func (m *mockAgent) Description() string                { return "Mock agent for testing" }
func (m *mockAgent) IsAvailable() bool                  { return true }
func (m *mockAgent) CheckAuth() error                   { return nil }
func (m *mockAgent) ListModels() ([]agent.Model, error) { return nil, nil }
func (m *mockAgent) GetDefaultModel() agent.Model       { return agent.Model{ID: "mock-model"} }
func (m *mockAgent) GetSessionID() string               { return "mock-session" }

func (m *mockAgent) Run(ctx context.Context, prompt string, opts agent.RunOptions) (agent.Result, error) {
	m.runCount++
	m.lastPrompt = prompt
	m.lastOptions = opts
	return m.runResult, m.runError
}

func (m *mockAgent) Continue(ctx context.Context, sessionID string, prompt string, opts agent.RunOptions) (agent.Result, error) {
	m.runCount++
	m.lastPrompt = prompt
	m.lastOptions = opts
	return m.runResult, m.runError
}

// newTestManager creates a task manager for testing.
func newTestManager(t *testing.T) *task.Manager {
	t.Helper()
	tmpDir := t.TempDir()
	store := task.NewStore(tmpDir + "/tasks.json")
	return task.NewManager(store)
}

// setupTestProjectDir creates a temporary project directory with required files.
func setupTestProjectDir(t *testing.T) string {
	t.Helper()
	tmpDir := t.TempDir()

	// Create .ralph directory with base prompt file
	ralphDir := filepath.Join(tmpDir, ".ralph")
	if err := os.MkdirAll(ralphDir, 0755); err != nil {
		t.Fatalf("failed to create .ralph dir: %v", err)
	}

	basePrompt := `# Ralph Base Prompt
You are an AI agent working on a task. Complete the task and report your status.
`
	if err := os.WriteFile(filepath.Join(ralphDir, "base_prompt.txt"), []byte(basePrompt), 0644); err != nil {
		t.Fatalf("failed to write base_prompt.txt: %v", err)
	}

	return tmpDir
}

// newTestConfig creates a minimal config for testing.
func newTestConfig() *config.Config {
	return &config.Config{
		Agent: config.AgentConfig{
			Default: "mock",
			Model:   "test-model",
		},
		Timeout: config.TimeoutConfig{
			Active: 10 * time.Minute,
		},
	}
}

// newTestAnalysis creates a mock project analysis.
func newTestAnalysis() *build.ProjectAnalysis {
	buildCmd := "go build ./..."
	testCmd := "go test ./..."
	return &build.ProjectAnalysis{
		ProjectType:  "go",
		Languages:    []string{"go"},
		IsGreenfield: false,
		Build: build.BuildAnalysis{
			Ready:   true,
			Command: &buildCmd,
		},
		Test: build.TestAnalysis{
			Ready:        true,
			Command:      &testCmd,
			HasTestFiles: true,
		},
		ProjectContext: "Test Go project",
	}
}

func TestNewLoop(t *testing.T) {
	mockAg := &mockAgent{name: "test-agent"}
	taskMgr := newTestManager(t)
	cfg := newTestConfig()

	l := NewLoop(mockAg, taskMgr, nil, cfg, "/tmp/project")

	if l.agent != mockAg {
		t.Error("agent not set")
	}
	if l.taskManager != taskMgr {
		t.Error("task manager not set")
	}
	if l.config != cfg {
		t.Error("config not set")
	}
	if l.projectDir != "/tmp/project" {
		t.Errorf("projectDir = %q, want /tmp/project", l.projectDir)
	}
	if l.opts == nil {
		t.Error("opts should be initialized with defaults")
	}
	if l.opts.MaxIterationsPerTask != DefaultMaxIterations {
		t.Errorf("MaxIterationsPerTask = %d, want %d", l.opts.MaxIterationsPerTask, DefaultMaxIterations)
	}
}

func TestLoop_SetOptions(t *testing.T) {
	mockAg := &mockAgent{name: "test-agent"}
	taskMgr := newTestManager(t)
	cfg := newTestConfig()

	l := NewLoop(mockAg, taskMgr, nil, cfg, "/tmp/project")

	customOpts := &Options{
		MaxIterationsPerTask: 10,
		MaxFixAttempts:       5,
	}
	l.SetOptions(customOpts)

	if l.opts.MaxIterationsPerTask != 10 {
		t.Errorf("MaxIterationsPerTask = %d, want 10", l.opts.MaxIterationsPerTask)
	}
	if l.opts.MaxFixAttempts != 5 {
		t.Errorf("MaxFixAttempts = %d, want 5", l.opts.MaxFixAttempts)
	}
}

func TestLoop_SetAnalysis(t *testing.T) {
	mockAg := &mockAgent{name: "test-agent"}
	taskMgr := newTestManager(t)
	cfg := newTestConfig()

	l := NewLoop(mockAg, taskMgr, nil, cfg, "/tmp/project")

	analysis := newTestAnalysis()
	l.SetAnalysis(analysis)

	if l.analysis != analysis {
		t.Error("analysis not set")
	}
}

func TestLoop_Run_NoTasks(t *testing.T) {
	mockAg := &mockAgent{name: "test-agent"}
	taskMgr := newTestManager(t)
	cfg := newTestConfig()
	tmpDir := setupTestProjectDir(t)

	l := NewLoop(mockAg, taskMgr, nil, cfg, tmpDir)
	l.SetAnalysis(newTestAnalysis())

	ctx := context.Background()
	err := l.Run(ctx, "test-session")

	if err != nil {
		t.Fatalf("Run() error = %v, want nil", err)
	}
	if l.context.State != StateCompleted {
		t.Errorf("State = %v, want %v", l.context.State, StateCompleted)
	}
}

func TestLoop_Run_SingleTaskDone(t *testing.T) {
	mockAg := &mockAgent{
		name: "test-agent",
		runResult: agent.Result{
			Output:    "Task completed",
			ExitCode:  0,
			Status:    agent.TaskStatusDone,
			SessionID: "agent-session-1",
		},
	}
	taskMgr := newTestManager(t)
	tsk := task.NewTask("TASK-001", "Test task", "Test description")
	taskMgr.AddTask(tsk)

	cfg := newTestConfig()
	tmpDir := setupTestProjectDir(t)

	l := NewLoop(mockAg, taskMgr, nil, cfg, tmpDir)

	// Use greenfield analysis to skip verification
	greenfield := newTestAnalysis()
	greenfield.IsGreenfield = true
	greenfield.Build.Ready = false
	greenfield.Test.Ready = false
	l.SetAnalysis(greenfield)

	ctx := context.Background()
	err := l.Run(ctx, "test-session")

	if err != nil {
		t.Fatalf("Run() error = %v, want nil", err)
	}
	if l.context.State != StateCompleted {
		t.Errorf("State = %v, want %v", l.context.State, StateCompleted)
	}
	if l.context.TasksCompleted != 1 {
		t.Errorf("TasksCompleted = %d, want 1", l.context.TasksCompleted)
	}
	if mockAg.runCount != 1 {
		t.Errorf("Agent run count = %d, want 1", mockAg.runCount)
	}

	// Verify task status was updated
	updated, _ := taskMgr.GetByID("TASK-001")
	if updated.Status != task.StatusCompleted {
		t.Errorf("Task status = %v, want completed", updated.Status)
	}
}

func TestLoop_Run_AgentError(t *testing.T) {
	mockAg := &mockAgent{
		name:     "test-agent",
		runError: errors.New("agent failed"),
	}
	taskMgr := newTestManager(t)
	tsk := task.NewTask("TASK-001", "Test task", "Test description")
	taskMgr.AddTask(tsk)

	cfg := newTestConfig()
	tmpDir := setupTestProjectDir(t)

	l := NewLoop(mockAg, taskMgr, nil, cfg, tmpDir)

	// Use greenfield analysis to skip verification
	greenfield := newTestAnalysis()
	greenfield.IsGreenfield = true
	greenfield.Build.Ready = false
	greenfield.Test.Ready = false
	l.SetAnalysis(greenfield)

	ctx := context.Background()
	err := l.Run(ctx, "test-session")

	// Agent error should mark task as failed but loop continues (no more tasks)
	if err != nil {
		t.Fatalf("Run() error = %v, want nil (loop should continue after failed task)", err)
	}

	// Task should be marked as failed
	updated, _ := taskMgr.GetByID("TASK-001")
	if updated.Status != task.StatusFailed {
		t.Errorf("Task status = %v, want failed", updated.Status)
	}
}

func TestLoop_Run_Cancellation(t *testing.T) {
	mockAg := &mockAgent{
		name: "test-agent",
		runResult: agent.Result{
			Output:   "Running...",
			ExitCode: 0,
			Status:   agent.TaskStatusNext,
		},
	}
	taskMgr := newTestManager(t)
	tsk := task.NewTask("TASK-001", "Test task", "Test description")
	taskMgr.AddTask(tsk)

	cfg := newTestConfig()
	tmpDir := setupTestProjectDir(t)

	l := NewLoop(mockAg, taskMgr, nil, cfg, tmpDir)

	// Use greenfield analysis to skip verification
	greenfield := newTestAnalysis()
	greenfield.IsGreenfield = true
	greenfield.Build.Ready = false
	greenfield.Test.Ready = false
	l.SetAnalysis(greenfield)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	err := l.Run(ctx, "test-session")

	if err != context.Canceled {
		t.Errorf("Run() error = %v, want context.Canceled", err)
	}
}


func TestLoop_Run_EventEmission(t *testing.T) {
	mockAg := &mockAgent{
		name: "test-agent",
		runResult: agent.Result{
			Output:   "Done",
			ExitCode: 0,
			Status:   agent.TaskStatusDone,
		},
	}
	taskMgr := newTestManager(t)
	tsk := task.NewTask("TASK-001", "Test task", "Test description")
	taskMgr.AddTask(tsk)

	cfg := newTestConfig()
	tmpDir := setupTestProjectDir(t)

	l := NewLoop(mockAg, taskMgr, nil, cfg, tmpDir)

	greenfield := newTestAnalysis()
	greenfield.IsGreenfield = true
	greenfield.Build.Ready = false
	greenfield.Test.Ready = false
	l.SetAnalysis(greenfield)

	var events []Event
	l.SetOptions(&Options{
		MaxIterationsPerTask: 5,
		MaxFixAttempts:       3,
		OnEvent: func(event Event) {
			events = append(events, event)
		},
	})

	ctx := context.Background()
	err := l.Run(ctx, "test-session")
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	// Check for expected events
	eventTypes := make([]EventType, len(events))
	for i, e := range events {
		eventTypes[i] = e.Type
	}

	// Should have: loop_started, task_started, iteration_started, iteration_ended, task_completed, loop_completed
	hasLoopStarted := false
	hasTaskStarted := false
	hasIterationStarted := false
	hasTaskCompleted := false
	hasLoopCompleted := false

	for _, et := range eventTypes {
		switch et {
		case EventLoopStarted:
			hasLoopStarted = true
		case EventTaskStarted:
			hasTaskStarted = true
		case EventIterationStarted:
			hasIterationStarted = true
		case EventTaskCompleted:
			hasTaskCompleted = true
		case EventLoopCompleted:
			hasLoopCompleted = true
		}
	}

	if !hasLoopStarted {
		t.Error("missing EventLoopStarted")
	}
	if !hasTaskStarted {
		t.Error("missing EventTaskStarted")
	}
	if !hasIterationStarted {
		t.Error("missing EventIterationStarted")
	}
	if !hasTaskCompleted {
		t.Error("missing EventTaskCompleted")
	}
	if !hasLoopCompleted {
		t.Error("missing EventLoopCompleted")
	}
}

func TestLoop_Run_MaxIterations(t *testing.T) {
	// Agent always returns NEXT (never completes)
	mockAg := &mockAgent{
		name: "test-agent",
		runResult: agent.Result{
			Output:   "Still working...",
			ExitCode: 0,
			Status:   agent.TaskStatusNext,
		},
	}
	taskMgr := newTestManager(t)
	tsk := task.NewTask("TASK-001", "Test task", "Test description")
	taskMgr.AddTask(tsk)

	cfg := newTestConfig()
	tmpDir := setupTestProjectDir(t)

	l := NewLoop(mockAg, taskMgr, nil, cfg, tmpDir)

	greenfield := newTestAnalysis()
	greenfield.IsGreenfield = true
	greenfield.Build.Ready = false
	greenfield.Test.Ready = false
	l.SetAnalysis(greenfield)

	l.SetOptions(&Options{
		MaxIterationsPerTask: 3,
		MaxFixAttempts:       3,
	})

	ctx := context.Background()
	_ = l.Run(ctx, "test-session")

	// Agent should have been called 3 times
	if mockAg.runCount != 3 {
		t.Errorf("Agent run count = %d, want 3", mockAg.runCount)
	}

	// Task should be marked as failed (exceeded iterations)
	updated, _ := taskMgr.GetByID("TASK-001")
	if updated.Status != task.StatusFailed {
		t.Errorf("Task status = %v, want failed", updated.Status)
	}
}

func TestLoop_Pause_NotRunning(t *testing.T) {
	mockAg := &mockAgent{name: "test-agent"}
	taskMgr := newTestManager(t)
	cfg := newTestConfig()

	l := NewLoop(mockAg, taskMgr, nil, cfg, "/tmp/project")

	err := l.Pause()
	if err == nil {
		t.Error("Pause() should error when loop is not running")
	}
}

func TestLoop_Context(t *testing.T) {
	mockAg := &mockAgent{name: "test-agent"}
	taskMgr := newTestManager(t)
	cfg := newTestConfig()

	l := NewLoop(mockAg, taskMgr, nil, cfg, "/tmp/project")

	// Before run, context should be nil
	if l.Context() != nil {
		t.Error("Context() should be nil before Run()")
	}

	l.SetAnalysis(newTestAnalysis())
	l.Run(context.Background(), "test-session")

	// After run, context should exist
	if l.Context() == nil {
		t.Error("Context() should not be nil after Run()")
	}
	if l.Context().SessionID != "test-session" {
		t.Errorf("SessionID = %q, want test-session", l.Context().SessionID)
	}
}

func TestLoop_buildAnalysisContext(t *testing.T) {
	mockAg := &mockAgent{name: "test-agent"}
	taskMgr := newTestManager(t)
	cfg := newTestConfig()

	l := NewLoop(mockAg, taskMgr, nil, cfg, "/tmp/project")

	t.Run("nil analysis", func(t *testing.T) {
		ctx := l.buildAnalysisContext()
		if ctx != "Project analysis not available." {
			t.Errorf("buildAnalysisContext() = %q, want 'Project analysis not available.'", ctx)
		}
	})

	t.Run("with analysis", func(t *testing.T) {
		l.SetAnalysis(newTestAnalysis())
		ctx := l.buildAnalysisContext()

		// Should contain project type
		if !contains(ctx, "Project Type: go") {
			t.Error("missing Project Type in context")
		}
		// Should contain languages
		if !contains(ctx, "Languages: go") {
			t.Error("missing Languages in context")
		}
		// Should contain build command
		if !contains(ctx, "Build Command: go build ./...") {
			t.Error("missing Build Command in context")
		}
	})

	t.Run("greenfield project", func(t *testing.T) {
		greenfield := newTestAnalysis()
		greenfield.IsGreenfield = true
		l.SetAnalysis(greenfield)

		ctx := l.buildAnalysisContext()
		if !contains(ctx, "Greenfield project") {
			t.Error("missing Greenfield indicator in context")
		}
	})
}

func TestLoop_buildTaskContent(t *testing.T) {
	mockAg := &mockAgent{name: "test-agent"}
	taskMgr := newTestManager(t)
	cfg := newTestConfig()

	l := NewLoop(mockAg, taskMgr, nil, cfg, "/tmp/project")

	tsk := task.NewTask("TASK-001", "Test task", "Test description for the task")

	t.Run("first iteration", func(t *testing.T) {
		content := l.buildTaskContent(tsk, 1)

		if !contains(content, "**Task ID:** TASK-001") {
			t.Error("missing Task ID")
		}
		if !contains(content, "**Task:** Test task") {
			t.Error("missing Task name")
		}
		if !contains(content, "**Description:**") {
			t.Error("missing Description")
		}
		// First iteration should not mention previous attempts
		if contains(content, "Iteration:") {
			t.Error("first iteration should not mention iteration number")
		}
	})

	t.Run("later iteration", func(t *testing.T) {
		content := l.buildTaskContent(tsk, 3)

		if !contains(content, "**Iteration:** 3") {
			t.Error("missing Iteration indicator")
		}
		if !contains(content, "previous attempt") {
			t.Error("missing previous attempt message")
		}
	})
}

func TestDefaultOptions(t *testing.T) {
	opts := DefaultOptions()

	if opts.MaxIterationsPerTask != DefaultMaxIterations {
		t.Errorf("MaxIterationsPerTask = %d, want %d", opts.MaxIterationsPerTask, DefaultMaxIterations)
	}
	if opts.MaxFixAttempts != DefaultMaxFixAttempts {
		t.Errorf("MaxFixAttempts = %d, want %d", opts.MaxFixAttempts, DefaultMaxFixAttempts)
	}
}

func TestTruncateOutput(t *testing.T) {
	t.Run("short output", func(t *testing.T) {
		output := "short output"
		result := truncateOutput(output)
		if result != output {
			t.Errorf("truncateOutput() = %q, want %q", result, output)
		}
	})

	t.Run("long output", func(t *testing.T) {
		// Create output longer than 1000 chars
		long := make([]byte, 2000)
		for i := range long {
			long[i] = 'x'
		}
		output := string(long)
		result := truncateOutput(output)

		// Should start with "..."
		if result[:3] != "..." {
			t.Error("truncated output should start with ...")
		}
		// Should be 1003 chars (1000 + "...")
		if len(result) != 1003 {
			t.Errorf("truncated length = %d, want 1003", len(result))
		}
	})
}

// Helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestLoop_CommitTaskChanges(t *testing.T) {
	t.Run("emits commit events", func(t *testing.T) {
		// Create temp git repo
		tmpDir, err := os.MkdirTemp("", "loop-commit-test-*")
		if err != nil {
			t.Fatalf("failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tmpDir)

		// Initialize git repo with initial commit
		runGitCommand(t, tmpDir, "init")
		runGitCommand(t, tmpDir, "config", "user.email", "test@example.com")
		runGitCommand(t, tmpDir, "config", "user.name", "Test User")
		if err := os.WriteFile(filepath.Join(tmpDir, "README.md"), []byte("# Test"), 0644); err != nil {
			t.Fatalf("failed to create README: %v", err)
		}
		runGitCommand(t, tmpDir, "add", "-A")
		runGitCommand(t, tmpDir, "commit", "-m", "Initial commit")

		// Create .ralph directory with required files
		ralphDir := filepath.Join(tmpDir, ".ralph")
		if err := os.MkdirAll(ralphDir, 0755); err != nil {
			t.Fatalf("failed to create .ralph dir: %v", err)
		}

		// Create base prompt file (required by prompt loader)
		basePrompt := `# Ralph Base Prompt
You are an AI agent working on a task. Complete the task and report your status.
`
		if err := os.WriteFile(filepath.Join(ralphDir, "base_prompt.txt"), []byte(basePrompt), 0644); err != nil {
			t.Fatalf("failed to write base_prompt.txt: %v", err)
		}

		// Create task manager
		store := task.NewStore(filepath.Join(ralphDir, "tasks.json"))
		mgr := task.NewManager(store)

		testTask := task.NewTask("TEST-001", "Test Task", "Description")
		if err := mgr.AddTask(testTask); err != nil {
			t.Fatalf("failed to add task: %v", err)
		}

		// Config with auto-commit enabled
		cfg := config.NewConfig()
		cfg.Git.AutoCommit = true
		cfg.Git.CommitPrefix = "[test]"

		// Create mock agent
		mockAg := &mockAgent{
			name: "test-agent",
			runResult: agent.Result{
				Status:    agent.TaskStatusDone,
				SessionID: "test-session",
			},
		}

		// Create loop
		loop := NewLoop(mockAg, mgr, nil, cfg, tmpDir)

		// Set analysis to skip analysis phase
		loop.SetAnalysis(&build.ProjectAnalysis{
			ProjectType: "test",
		})

		// Track events
		var events []Event
		loop.SetOptions(&Options{
			MaxIterationsPerTask: 1,
			OnEvent: func(e Event) {
				events = append(events, e)
			},
		})

		// Create a file change
		if err := os.WriteFile(filepath.Join(tmpDir, "feature.go"), []byte("package main"), 0644); err != nil {
			t.Fatalf("failed to create test file: %v", err)
		}

		// Run the loop
		ctx := context.Background()
		err = loop.Run(ctx, "test-session")
		if err != nil {
			t.Fatalf("Run() error = %v", err)
		}

		// Check for commit events
		var foundCommitStarted, foundCommitCompleted bool
		for _, e := range events {
			switch e.Type {
			case EventCommitStarted:
				foundCommitStarted = true
			case EventCommitCompleted:
				foundCommitCompleted = true
				if e.TaskID != "TEST-001" {
					t.Errorf("commit event TaskID = %q, want %q", e.TaskID, "TEST-001")
				}
			}
		}

		if !foundCommitStarted {
			t.Error("expected EventCommitStarted event")
		}
		if !foundCommitCompleted {
			t.Error("expected EventCommitCompleted event")
		}
	})

	t.Run("skips commit when auto-commit disabled", func(t *testing.T) {
		// Create temp dir (not a git repo)
		tmpDir, err := os.MkdirTemp("", "loop-nocommit-test-*")
		if err != nil {
			t.Fatalf("failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tmpDir)

		// Create .ralph directory with required files
		ralphDir := filepath.Join(tmpDir, ".ralph")
		if err := os.MkdirAll(ralphDir, 0755); err != nil {
			t.Fatalf("failed to create .ralph dir: %v", err)
		}

		// Create base prompt file (required by prompt loader)
		basePrompt := `# Ralph Base Prompt
You are an AI agent working on a task. Complete the task and report your status.
`
		if err := os.WriteFile(filepath.Join(ralphDir, "base_prompt.txt"), []byte(basePrompt), 0644); err != nil {
			t.Fatalf("failed to write base_prompt.txt: %v", err)
		}

		// Create task manager
		store := task.NewStore(filepath.Join(ralphDir, "tasks.json"))
		mgr := task.NewManager(store)

		testTask := task.NewTask("TEST-002", "Test Task", "Description")
		if err := mgr.AddTask(testTask); err != nil {
			t.Fatalf("failed to add task: %v", err)
		}

		// Config with auto-commit disabled
		cfg := config.NewConfig()
		cfg.Git.AutoCommit = false

		// Create mock agent
		mockAg := &mockAgent{
			name: "test-agent",
			runResult: agent.Result{
				Status:    agent.TaskStatusDone,
				SessionID: "test-session",
			},
		}

		// Create loop
		loop := NewLoop(mockAg, mgr, nil, cfg, tmpDir)

		// Set analysis to skip analysis phase
		loop.SetAnalysis(&build.ProjectAnalysis{
			ProjectType: "test",
		})

		// Track events
		var events []Event
		loop.SetOptions(&Options{
			MaxIterationsPerTask: 1,
			OnEvent: func(e Event) {
				events = append(events, e)
			},
		})

		// Run the loop
		ctx := context.Background()
		err = loop.Run(ctx, "test-session")
		if err != nil {
			t.Fatalf("Run() error = %v", err)
		}

		// Check for commit skipped event
		var foundCommitSkipped bool
		for _, e := range events {
			if e.Type == EventCommitSkipped {
				foundCommitSkipped = true
				break
			}
		}

		if !foundCommitSkipped {
			t.Error("expected EventCommitSkipped event when auto-commit is disabled")
		}
	})
}

func runGitCommand(t *testing.T, dir string, args ...string) {
	t.Helper()
	c := exec.Command("git", args...)
	c.Dir = dir
	if err := c.Run(); err != nil {
		t.Fatalf("git %v failed: %v", args, err)
	}
}

// ============================================
// Error Recovery Tests (LOOP-004)
// ============================================

func TestIsRetryableError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
		{
			name:     "timeout error",
			err:      errors.New("connection timeout"),
			expected: true,
		},
		{
			name:     "connection refused",
			err:      errors.New("dial tcp: connection refused"),
			expected: true,
		},
		{
			name:     "connection reset",
			err:      errors.New("connection reset by peer"),
			expected: true,
		},
		{
			name:     "context deadline exceeded",
			err:      errors.New("context deadline exceeded"),
			expected: true,
		},
		{
			name:     "EOF error",
			err:      errors.New("unexpected EOF"),
			expected: true,
		},
		{
			name:     "broken pipe",
			err:      errors.New("write: broken pipe"),
			expected: true,
		},
		{
			name:     "network unreachable",
			err:      errors.New("network unreachable"),
			expected: true,
		},
		{
			name:     "non-retryable error",
			err:      errors.New("file not found"),
			expected: false,
		},
		{
			name:     "syntax error",
			err:      errors.New("syntax error in code"),
			expected: false,
		},
		{
			name:     "agent error",
			err:      errors.New("agent returned error status"),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsRetryableError(tt.err)
			if result != tt.expected {
				t.Errorf("IsRetryableError(%v) = %v, want %v", tt.err, result, tt.expected)
			}
		})
	}
}

func TestDefaultRecoveryConfig(t *testing.T) {
	cfg := DefaultRecoveryConfig()

	if cfg.MaxAgentRetries != 3 {
		t.Errorf("MaxAgentRetries = %d, want 3", cfg.MaxAgentRetries)
	}
	if cfg.RetryBackoff != 5*time.Second {
		t.Errorf("RetryBackoff = %v, want 5s", cfg.RetryBackoff)
	}
	if cfg.MaxRetryBackoff != 60*time.Second {
		t.Errorf("MaxRetryBackoff = %v, want 60s", cfg.MaxRetryBackoff)
	}
	if !cfg.EnableAutoFix {
		t.Error("EnableAutoFix should be true by default")
	}
}

func TestNewErrorRecovery(t *testing.T) {
	mockAg := &mockAgent{name: "test-agent"}
	taskMgr := newTestManager(t)
	cfg := newTestConfig()
	l := NewLoop(mockAg, taskMgr, nil, cfg, "/tmp/project")

	t.Run("with nil config uses defaults", func(t *testing.T) {
		recovery := NewErrorRecovery(l, nil)
		if recovery.config.MaxAgentRetries != 3 {
			t.Errorf("MaxAgentRetries = %d, want 3", recovery.config.MaxAgentRetries)
		}
	})

	t.Run("with custom config", func(t *testing.T) {
		customCfg := &RecoveryConfig{
			MaxAgentRetries: 5,
			RetryBackoff:    10 * time.Second,
			MaxRetryBackoff: 120 * time.Second,
			EnableAutoFix:   false,
		}
		recovery := NewErrorRecovery(l, customCfg)
		if recovery.config.MaxAgentRetries != 5 {
			t.Errorf("MaxAgentRetries = %d, want 5", recovery.config.MaxAgentRetries)
		}
		if recovery.config.EnableAutoFix {
			t.Error("EnableAutoFix should be false")
		}
	})
}

func TestLoop_SetRecoveryConfig(t *testing.T) {
	mockAg := &mockAgent{name: "test-agent"}
	taskMgr := newTestManager(t)
	cfg := newTestConfig()
	l := NewLoop(mockAg, taskMgr, nil, cfg, "/tmp/project")

	customCfg := &RecoveryConfig{
		MaxAgentRetries: 7,
		EnableAutoFix:   false,
	}
	l.SetRecoveryConfig(customCfg)

	if l.recovery.config.MaxAgentRetries != 7 {
		t.Errorf("MaxAgentRetries = %d, want 7", l.recovery.config.MaxAgentRetries)
	}
}

func TestRetryWithBackoff_Success(t *testing.T) {
	mockAg := &mockAgent{name: "test-agent"}
	taskMgr := newTestManager(t)
	cfg := newTestConfig()
	l := NewLoop(mockAg, taskMgr, nil, cfg, "/tmp/project")

	recoveryCfg := &RecoveryConfig{
		MaxAgentRetries: 3,
		RetryBackoff:    10 * time.Millisecond,
		MaxRetryBackoff: 50 * time.Millisecond,
	}
	recovery := NewErrorRecovery(l, recoveryCfg)

	callCount := 0
	err := recovery.RetryWithBackoff(context.Background(), "test-op", func() error {
		callCount++
		return nil
	})

	if err != nil {
		t.Errorf("RetryWithBackoff() error = %v, want nil", err)
	}
	if callCount != 1 {
		t.Errorf("function called %d times, want 1", callCount)
	}
}

func TestRetryWithBackoff_RetryableError(t *testing.T) {
	mockAg := &mockAgent{name: "test-agent"}
	taskMgr := newTestManager(t)
	cfg := newTestConfig()
	l := NewLoop(mockAg, taskMgr, nil, cfg, "/tmp/project")

	recoveryCfg := &RecoveryConfig{
		MaxAgentRetries: 3,
		RetryBackoff:    10 * time.Millisecond,
		MaxRetryBackoff: 50 * time.Millisecond,
	}
	recovery := NewErrorRecovery(l, recoveryCfg)

	callCount := 0
	err := recovery.RetryWithBackoff(context.Background(), "test-op", func() error {
		callCount++
		if callCount < 3 {
			return errors.New("connection timeout")
		}
		return nil
	})

	if err != nil {
		t.Errorf("RetryWithBackoff() error = %v, want nil", err)
	}
	if callCount != 3 {
		t.Errorf("function called %d times, want 3", callCount)
	}
}

func TestRetryWithBackoff_MaxRetriesExceeded(t *testing.T) {
	mockAg := &mockAgent{name: "test-agent"}
	taskMgr := newTestManager(t)
	cfg := newTestConfig()
	l := NewLoop(mockAg, taskMgr, nil, cfg, "/tmp/project")

	recoveryCfg := &RecoveryConfig{
		MaxAgentRetries: 2,
		RetryBackoff:    10 * time.Millisecond,
		MaxRetryBackoff: 50 * time.Millisecond,
	}
	recovery := NewErrorRecovery(l, recoveryCfg)

	callCount := 0
	err := recovery.RetryWithBackoff(context.Background(), "test-op", func() error {
		callCount++
		return errors.New("connection timeout")
	})

	if err == nil {
		t.Error("RetryWithBackoff() expected error, got nil")
	}
	if !contains(err.Error(), "max retries") {
		t.Errorf("error = %v, should contain 'max retries'", err)
	}
	// Should be called MaxAgentRetries + 1 times (initial + retries)
	if callCount != 3 {
		t.Errorf("function called %d times, want 3", callCount)
	}
}

func TestRetryWithBackoff_NonRetryableError(t *testing.T) {
	mockAg := &mockAgent{name: "test-agent"}
	taskMgr := newTestManager(t)
	cfg := newTestConfig()
	l := NewLoop(mockAg, taskMgr, nil, cfg, "/tmp/project")

	recoveryCfg := &RecoveryConfig{
		MaxAgentRetries: 3,
		RetryBackoff:    10 * time.Millisecond,
		MaxRetryBackoff: 50 * time.Millisecond,
	}
	recovery := NewErrorRecovery(l, recoveryCfg)

	callCount := 0
	err := recovery.RetryWithBackoff(context.Background(), "test-op", func() error {
		callCount++
		return errors.New("file not found")
	})

	if err == nil {
		t.Error("RetryWithBackoff() expected error, got nil")
	}
	// Non-retryable errors should not be retried
	if callCount != 1 {
		t.Errorf("function called %d times, want 1 (non-retryable)", callCount)
	}
}

func TestRetryWithBackoff_ContextCancellation(t *testing.T) {
	mockAg := &mockAgent{name: "test-agent"}
	taskMgr := newTestManager(t)
	cfg := newTestConfig()
	l := NewLoop(mockAg, taskMgr, nil, cfg, "/tmp/project")

	recoveryCfg := &RecoveryConfig{
		MaxAgentRetries: 3,
		RetryBackoff:    100 * time.Millisecond, // Longer backoff
		MaxRetryBackoff: 500 * time.Millisecond,
	}
	recovery := NewErrorRecovery(l, recoveryCfg)

	ctx, cancel := context.WithCancel(context.Background())
	callCount := 0

	// Start retry in goroutine
	errChan := make(chan error, 1)
	go func() {
		err := recovery.RetryWithBackoff(ctx, "test-op", func() error {
			callCount++
			return errors.New("timeout")
		})
		errChan <- err
	}()

	// Wait for first call, then cancel
	time.Sleep(50 * time.Millisecond)
	cancel()

	err := <-errChan
	if err != context.Canceled {
		t.Errorf("RetryWithBackoff() error = %v, want context.Canceled", err)
	}
}

// ============================================
// Fix Prompt Builder Tests
// ============================================

func TestFixPromptBuilder_BuildBuildFixPrompt(t *testing.T) {
	builder := NewFixPromptBuilder("/tmp/project")
	tsk := task.NewTask("TEST-001", "Test Task", "Test description")

	t.Run("with build errors", func(t *testing.T) {
		result := &build.BuildResult{
			Success: false,
			Errors: []build.BuildError{
				{
					File:    "main.go",
					Line:    10,
					Message: "undefined: foo",
				},
				{
					File:    "util.go",
					Line:    25,
					Message: "syntax error",
				},
			},
		}

		prompt := builder.BuildBuildFixPrompt(tsk, result)

		if !contains(prompt, "Build Fix Required") {
			t.Error("missing 'Build Fix Required' header")
		}
		if !contains(prompt, "TEST-001") {
			t.Error("missing task ID")
		}
		if !contains(prompt, "main.go:10") {
			t.Error("missing first error file:line")
		}
		if !contains(prompt, "undefined: foo") {
			t.Error("missing first error message")
		}
		if !contains(prompt, "util.go:25") {
			t.Error("missing second error file:line")
		}
		if !contains(prompt, "Report FIXED when done") {
			t.Error("missing instruction to report FIXED")
		}
	})

	t.Run("without parsed errors", func(t *testing.T) {
		result := &build.BuildResult{
			Success: false,
			Output:  "build failed: some error",
			Errors:  []build.BuildError{},
		}

		prompt := builder.BuildBuildFixPrompt(tsk, result)

		if !contains(prompt, "Build command failed") {
			t.Error("missing 'Build command failed' section")
		}
		if !contains(prompt, "build failed: some error") {
			t.Error("missing raw output")
		}
	})
}

func TestFixPromptBuilder_BuildTestFixPrompt(t *testing.T) {
	builder := NewFixPromptBuilder("/tmp/project")
	tsk := task.NewTask("TEST-002", "Test Task", "Test description")

	t.Run("with test failures", func(t *testing.T) {
		result := &build.TestResult{
			Success: false,
			Failures: []build.TestFailure{
				{
					TestName: "TestFoo",
					Package:  "pkg/foo",
					File:     "foo_test.go",
					Line:     42,
					Message:  "expected true, got false",
				},
				{
					TestName: "TestBar",
					Package:  "pkg/bar",
					Message:  "assertion failed",
				},
			},
		}

		prompt := builder.BuildTestFixPrompt(tsk, result)

		if !contains(prompt, "Test Fix Required") {
			t.Error("missing 'Test Fix Required' header")
		}
		if !contains(prompt, "TEST-002") {
			t.Error("missing task ID")
		}
		if !contains(prompt, "### TestFoo") {
			t.Error("missing first test name")
		}
		if !contains(prompt, "Package: `pkg/foo`") {
			t.Error("missing first package")
		}
		if !contains(prompt, "foo_test.go:42") {
			t.Error("missing first test location")
		}
		if !contains(prompt, "expected true, got false") {
			t.Error("missing first error message")
		}
		if !contains(prompt, "### TestBar") {
			t.Error("missing second test name")
		}
	})

	t.Run("with unknown test name", func(t *testing.T) {
		result := &build.TestResult{
			Success: false,
			Failures: []build.TestFailure{
				{
					Message: "some failure",
				},
			},
		}

		prompt := builder.BuildTestFixPrompt(tsk, result)

		if !contains(prompt, "Unknown test") {
			t.Error("missing 'Unknown test' fallback for empty test name")
		}
	})

	t.Run("without parsed failures", func(t *testing.T) {
		result := &build.TestResult{
			Success:  false,
			Output:   "FAIL: tests failed with some error",
			Failures: []build.TestFailure{},
		}

		prompt := builder.BuildTestFixPrompt(tsk, result)

		if !contains(prompt, "Test command failed") {
			t.Error("missing 'Test command failed' section")
		}
		if !contains(prompt, "FAIL: tests failed") {
			t.Error("missing raw output")
		}
	})

	t.Run("long output is truncated", func(t *testing.T) {
		// Create output longer than 500 chars
		longOutput := make([]byte, 600)
		for i := range longOutput {
			longOutput[i] = 'x'
		}

		result := &build.TestResult{
			Success:  false,
			Output:   string(longOutput),
			Failures: []build.TestFailure{},
		}

		prompt := builder.BuildTestFixPrompt(tsk, result)

		if !contains(prompt, "...") {
			t.Error("long output should be truncated with ...")
		}
	})
}

func TestFixPromptBuilder_BuildVerificationFixPrompt(t *testing.T) {
	builder := NewFixPromptBuilder("/tmp/project")
	tsk := task.NewTask("TEST-003", "Test Task", "Test description")

	t.Run("build failure uses build fix prompt", func(t *testing.T) {
		gateResult := &build.GateResult{
			BuildResult: &build.BuildResult{
				Success: false,
				Errors:  []build.BuildError{{File: "main.go", Message: "error"}},
			},
			Reason: "build failed",
		}

		prompt := builder.BuildVerificationFixPrompt(tsk, gateResult)

		if !contains(prompt, "Build Fix Required") {
			t.Error("should use build fix prompt for build failures")
		}
	})

	t.Run("test failure uses test fix prompt", func(t *testing.T) {
		gateResult := &build.GateResult{
			BuildResult: &build.BuildResult{Success: true},
			TestResult: &build.TestResult{
				Success:  false,
				Failures: []build.TestFailure{{TestName: "TestFoo"}},
			},
			Reason: "tests failed",
		}

		prompt := builder.BuildVerificationFixPrompt(tsk, gateResult)

		if !contains(prompt, "Test Fix Required") {
			t.Error("should use test fix prompt for test failures")
		}
	})

	t.Run("generic failure", func(t *testing.T) {
		gateResult := &build.GateResult{
			BuildResult: &build.BuildResult{Success: true},
			TestResult:  &build.TestResult{Success: true},
			Reason:      "TDD regression detected",
		}

		prompt := builder.BuildVerificationFixPrompt(tsk, gateResult)

		if !contains(prompt, "Verification Fix Required") {
			t.Error("should use generic verification prompt")
		}
		if !contains(prompt, "TDD regression detected") {
			t.Error("should include reason")
		}
	})
}
