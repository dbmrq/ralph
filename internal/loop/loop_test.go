package loop

import (
	"context"
	"errors"
	"os"
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

