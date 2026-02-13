// Package loop provides integration tests for the ralph loop.
// These tests verify the full end-to-end loop execution with mock agents,
// session persistence/resume, and headless mode output.
package loop

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/wexinc/ralph/internal/agent"
	"github.com/wexinc/ralph/internal/build"
	"github.com/wexinc/ralph/internal/config"
	"github.com/wexinc/ralph/internal/hooks"
	"github.com/wexinc/ralph/internal/task"
)

// =============================================================================
// Integration Test Helpers
// =============================================================================

// scenarioAgent is a mock agent that follows a predefined scenario.
// It returns different results based on task ID and iteration number.
type scenarioAgent struct {
	name      string
	scenarios map[string][]agent.Result // taskID -> results per iteration
	mu        sync.Mutex
	calls     []agentCall
}

type agentCall struct {
	TaskID    string
	Prompt    string
	Iteration int
	IsContinue bool
}

func newScenarioAgent(name string) *scenarioAgent {
	return &scenarioAgent{
		name:      name,
		scenarios: make(map[string][]agent.Result),
	}
}

func (a *scenarioAgent) SetScenario(taskID string, results ...agent.Result) {
	a.scenarios[taskID] = results
}

func (a *scenarioAgent) Name() string                       { return a.name }
func (a *scenarioAgent) Description() string                { return "Scenario-based mock agent" }
func (a *scenarioAgent) IsAvailable() bool                  { return true }
func (a *scenarioAgent) CheckAuth() error                   { return nil }
func (a *scenarioAgent) ListModels() ([]agent.Model, error) { return nil, nil }
func (a *scenarioAgent) GetDefaultModel() agent.Model       { return agent.Model{ID: "mock"} }
func (a *scenarioAgent) GetSessionID() string               { return "scenario-session" }

func (a *scenarioAgent) Run(ctx context.Context, prompt string, opts agent.RunOptions) (agent.Result, error) {
	return a.handleCall(prompt, false)
}

func (a *scenarioAgent) Continue(ctx context.Context, sessionID string, prompt string, opts agent.RunOptions) (agent.Result, error) {
	return a.handleCall(prompt, true)
}

func (a *scenarioAgent) handleCall(prompt string, isContinue bool) (agent.Result, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	// Extract task ID from prompt (look for TASK- pattern)
	taskID := extractTaskIDFromPrompt(prompt)

	// Find iteration for this task
	iteration := 0
	for _, c := range a.calls {
		if c.TaskID == taskID {
			iteration++
		}
	}

	a.calls = append(a.calls, agentCall{
		TaskID:     taskID,
		Prompt:     prompt,
		Iteration:  iteration + 1,
		IsContinue: isContinue,
	})

	// Get result from scenario
	results, ok := a.scenarios[taskID]
	if !ok || iteration >= len(results) {
		return agent.Result{
			Status:    agent.TaskStatusDone,
			SessionID: "session-" + taskID,
		}, nil
	}

	return results[iteration], nil
}

func (a *scenarioAgent) GetCalls() []agentCall {
	a.mu.Lock()
	defer a.mu.Unlock()
	return append([]agentCall{}, a.calls...)
}

func extractTaskIDFromPrompt(prompt string) string {
	// Look for "Task ID: XXX" or "TASK-XXX" patterns
	if idx := strings.Index(prompt, "Task ID:"); idx >= 0 {
		end := strings.Index(prompt[idx:], "\n")
		if end > 0 {
			taskLine := prompt[idx : idx+end]
			parts := strings.Fields(taskLine)
			if len(parts) >= 3 {
				return strings.TrimRight(parts[2], "**")
			}
		}
	}
	return "UNKNOWN"
}

// setupIntegrationProject creates a complete test project directory.
func setupIntegrationProject(t *testing.T) string {
	t.Helper()
	tmpDir := t.TempDir()

	// Create .ralph directory structure
	ralphDir := filepath.Join(tmpDir, ".ralph")
	dirs := []string{ralphDir, filepath.Join(ralphDir, "sessions"), filepath.Join(ralphDir, "docs")}
	for _, d := range dirs {
		if err := os.MkdirAll(d, 0755); err != nil {
			t.Fatalf("failed to create dir %s: %v", d, err)
		}
	}

	// Create base_prompt.txt
	basePrompt := `# Ralph Base Prompt
You are an AI agent. Complete the assigned task and report status.
Report DONE when the task is complete, NEXT if more work is needed.
`
	if err := os.WriteFile(filepath.Join(ralphDir, "base_prompt.txt"), []byte(basePrompt), 0644); err != nil {
		t.Fatalf("failed to write base_prompt.txt: %v", err)
	}

	return tmpDir
}

// createIntegrationTaskManager creates a task manager with test tasks.
func createIntegrationTaskManager(t *testing.T, projectDir string, tasks ...*task.Task) *task.Manager {
	t.Helper()
	store := task.NewStore(filepath.Join(projectDir, ".ralph", "tasks.json"))
	mgr := task.NewManager(store)
	for _, tsk := range tasks {
		if err := mgr.AddTask(tsk); err != nil {
			t.Fatalf("failed to add task: %v", err)
		}
	}
	return mgr
}

// greenfieldAnalysis returns a ProjectAnalysis that skips verification.
func greenfieldAnalysis() *build.ProjectAnalysis {
	return &build.ProjectAnalysis{
		ProjectType:  "test",
		IsGreenfield: true,
		Build:        build.BuildAnalysis{Ready: false},
		Test:         build.TestAnalysis{Ready: false},
	}
}

// =============================================================================
// Full Loop Integration Tests
// =============================================================================

func TestIntegration_FullLoop_MultipleTasks(t *testing.T) {
	projectDir := setupIntegrationProject(t)

	// Create 3 tasks
	task1 := task.NewTask("TASK-001", "First task", "Description for task 1")
	task2 := task.NewTask("TASK-002", "Second task", "Description for task 2")
	task3 := task.NewTask("TASK-003", "Third task", "Description for task 3")
	taskMgr := createIntegrationTaskManager(t, projectDir, task1, task2, task3)

	// Create scenario agent that completes each task immediately
	ag := newScenarioAgent("test-agent")
	ag.SetScenario("TASK-001", agent.Result{Status: agent.TaskStatusDone, SessionID: "s1"})
	ag.SetScenario("TASK-002", agent.Result{Status: agent.TaskStatusDone, SessionID: "s2"})
	ag.SetScenario("TASK-003", agent.Result{Status: agent.TaskStatusDone, SessionID: "s3"})

	cfg := config.NewConfig()
	l := NewLoop(ag, taskMgr, nil, cfg, projectDir)
	l.SetAnalysis(greenfieldAnalysis())

	// Track events
	var events []Event
	l.SetOptions(&Options{
		MaxIterationsPerTask: 5,
		OnEvent: func(e Event) {
			events = append(events, e)
		},
	})

	ctx := context.Background()
	err := l.Run(ctx, "int-test-session")
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	// Verify all 3 tasks were completed
	if l.context.TasksCompleted != 3 {
		t.Errorf("TasksCompleted = %d, want 3", l.context.TasksCompleted)
	}

	// Verify agent was called 3 times
	calls := ag.GetCalls()
	if len(calls) != 3 {
		t.Errorf("Agent called %d times, want 3", len(calls))
	}

	// Verify final state
	if l.context.State != StateCompleted {
		t.Errorf("State = %q, want %q", l.context.State, StateCompleted)
	}

	// Verify event sequence
	foundStart, foundComplete := false, false
	for _, e := range events {
		if e.Type == EventLoopStarted {
			foundStart = true
		}
		if e.Type == EventLoopCompleted {
			foundComplete = true
		}
	}
	if !foundStart || !foundComplete {
		t.Error("missing EventLoopStarted or EventLoopCompleted")
	}
}

func TestIntegration_FullLoop_TaskWithMultipleIterations(t *testing.T) {
	projectDir := setupIntegrationProject(t)

	tsk := task.NewTask("TASK-001", "Multi-iteration task", "Needs multiple attempts")
	taskMgr := createIntegrationTaskManager(t, projectDir, tsk)

	// Task needs 3 iterations: NEXT, NEXT, DONE
	ag := newScenarioAgent("test-agent")
	ag.SetScenario("TASK-001",
		agent.Result{Status: agent.TaskStatusNext, SessionID: "s1-1"},
		agent.Result{Status: agent.TaskStatusNext, SessionID: "s1-2"},
		agent.Result{Status: agent.TaskStatusDone, SessionID: "s1-3"},
	)

	cfg := config.NewConfig()
	l := NewLoop(ag, taskMgr, nil, cfg, projectDir)
	l.SetAnalysis(greenfieldAnalysis())

	l.SetOptions(&Options{
		MaxIterationsPerTask: 5,
	})

	err := l.Run(context.Background(), "int-test-session")
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	// Agent should have been called 3 times for the same task
	calls := ag.GetCalls()
	if len(calls) != 3 {
		t.Errorf("Agent called %d times, want 3", len(calls))
	}

	// Task should be completed
	updated, _ := taskMgr.GetByID("TASK-001")
	if updated.Status != task.StatusCompleted {
		t.Errorf("Task status = %v, want completed", updated.Status)
	}

	// Context should show 3 iterations
	if l.context.TotalIterations != 3 {
		t.Errorf("TotalIterations = %d, want 3", l.context.TotalIterations)
	}
}

// =============================================================================
// Session Persistence and Resume Integration Tests
// =============================================================================

func TestIntegration_SessionPersistence_SaveAndResume(t *testing.T) {
	projectDir := setupIntegrationProject(t)

	// Create 2 tasks
	task1 := task.NewTask("TASK-001", "First task", "Will complete")
	task2 := task.NewTask("TASK-002", "Second task", "Will pause before this")
	taskMgr := createIntegrationTaskManager(t, projectDir, task1, task2)

	// Agent completes first task, then we'll pause
	ag := newScenarioAgent("test-agent")
	ag.SetScenario("TASK-001", agent.Result{Status: agent.TaskStatusDone, SessionID: "s1"})
	ag.SetScenario("TASK-002", agent.Result{Status: agent.TaskStatusDone, SessionID: "s2"})

	cfg := config.NewConfig()
	l := NewLoop(ag, taskMgr, nil, cfg, projectDir)
	l.SetAnalysis(greenfieldAnalysis())

	// Run the loop with a custom context that we control
	ctx, cancel := context.WithCancel(context.Background())

	var taskEvents []Event
	l.SetOptions(&Options{
		MaxIterationsPerTask: 3,
		OnEvent: func(e Event) {
			taskEvents = append(taskEvents, e)
			// After first task completes, cancel to simulate pause
			if e.Type == EventTaskCompleted && e.TaskID == "TASK-001" {
				cancel()
			}
		},
	})

	// First run - should complete TASK-001 then cancel before TASK-002
	err := l.Run(ctx, "persist-test-session")
	if err != context.Canceled {
		// It's ok if no error (task completed before next iteration)
		if err != nil && l.context.TasksCompleted < 1 {
			t.Fatalf("Run() error = %v, want context.Canceled or completion", err)
		}
	}

	// Verify first task was completed
	if l.context.TasksCompleted < 1 {
		t.Errorf("TasksCompleted = %d, want >= 1", l.context.TasksCompleted)
	}

	// Save session state
	sessionMgr := NewSessionManager(projectDir)
	if err := sessionMgr.SaveSession(l.context); err != nil {
		t.Fatalf("SaveSession() error = %v", err)
	}

	// Verify session file exists
	sessionFile := filepath.Join(projectDir, ".ralph", "sessions", l.context.SessionID+".json")
	if _, err := os.Stat(sessionFile); os.IsNotExist(err) {
		t.Error("session file was not created")
	}

	// Load and verify session state
	loaded, err := sessionMgr.GetSession(l.context.SessionID)
	if err != nil {
		t.Fatalf("GetSession() error = %v", err)
	}

	if loaded.SessionID != l.context.SessionID {
		t.Errorf("loaded SessionID = %q, want %q", loaded.SessionID, l.context.SessionID)
	}
	if loaded.TasksCompleted != l.context.TasksCompleted {
		t.Errorf("loaded TasksCompleted = %d, want %d", loaded.TasksCompleted, l.context.TasksCompleted)
	}
}

func TestIntegration_SessionResume_ContinuesFromPausedState(t *testing.T) {
	projectDir := setupIntegrationProject(t)

	// Create 2 tasks
	task1 := task.NewTask("TASK-001", "First task", "Already completed")
	task2 := task.NewTask("TASK-002", "Second task", "Will complete on resume")
	taskMgr := createIntegrationTaskManager(t, projectDir, task1, task2)

	// Mark first task as already completed
	taskMgr.MarkComplete("TASK-001")

	// Create and save a paused session
	sessionMgr := NewSessionManager(projectDir)
	pausedCtx, _ := sessionMgr.CreateSession("test-agent")
	pausedCtx.Transition(StateRunning)
	pausedCtx.Transition(StatePaused)
	pausedCtx.TasksCompleted = 1
	pausedCtx.AgentSessionID = "agent-session-123"
	sessionMgr.SaveSession(pausedCtx)

	// Create agent for resume
	ag := newScenarioAgent("test-agent")
	ag.SetScenario("TASK-002", agent.Result{Status: agent.TaskStatusDone, SessionID: "s2"})

	cfg := config.NewConfig()
	l := NewLoop(ag, taskMgr, nil, cfg, projectDir)
	l.SetAnalysis(greenfieldAnalysis())

	// Resume the session
	err := l.ResumeFromContext(context.Background(), pausedCtx)
	if err != nil {
		t.Fatalf("ResumeFromContext() error = %v", err)
	}

	// Verify session completed all tasks
	if l.context.State != StateCompleted {
		t.Errorf("State = %q, want %q", l.context.State, StateCompleted)
	}

	// Second task should be completed
	updated, _ := taskMgr.GetByID("TASK-002")
	if updated.Status != task.StatusCompleted {
		t.Errorf("TASK-002 status = %v, want completed", updated.Status)
	}
}

func TestIntegration_SessionManager_ResumeLatest(t *testing.T) {
	projectDir := setupIntegrationProject(t)
	sessionMgr := NewSessionManager(projectDir)

	// Create and save multiple sessions with different states
	completed, _ := sessionMgr.CreateSession("agent1")
	completed.Transition(StateRunning)
	completed.Transition(StateCompleted)
	sessionMgr.SaveSession(completed)

	paused1, _ := sessionMgr.CreateSession("agent2")
	paused1.Transition(StateRunning)
	paused1.Transition(StatePaused)
	paused1.UpdatedAt = time.Now().Add(-time.Hour)
	sessionMgr.SaveSession(paused1)

	paused2, _ := sessionMgr.CreateSession("agent3")
	paused2.Transition(StateRunning)
	paused2.Transition(StatePaused)
	paused2.UpdatedAt = time.Now()
	sessionMgr.SaveSession(paused2)

	// Resume without ID should get most recent resumable (paused2)
	resumed, err := sessionMgr.ResumeSession("")
	if err != nil {
		t.Fatalf("ResumeSession('') error = %v", err)
	}

	if resumed.SessionID != paused2.SessionID {
		t.Errorf("resumed SessionID = %q, want %q (most recent)", resumed.SessionID, paused2.SessionID)
	}

	// Verify we can list resumable sessions
	resumable, err := sessionMgr.GetResumableSessions()
	if err != nil {
		t.Fatalf("GetResumableSessions() error = %v", err)
	}
	if len(resumable) != 2 {
		t.Errorf("GetResumableSessions() returned %d, want 2", len(resumable))
	}
}

func TestIntegration_FullLoop_TaskExceedsMaxIterations(t *testing.T) {
	projectDir := setupIntegrationProject(t)

	tsk := task.NewTask("TASK-001", "Never-ending task", "Always returns NEXT")
	taskMgr := createIntegrationTaskManager(t, projectDir, tsk)

	// Task always returns NEXT
	ag := newScenarioAgent("test-agent")
	ag.SetScenario("TASK-001",
		agent.Result{Status: agent.TaskStatusNext},
		agent.Result{Status: agent.TaskStatusNext},
		agent.Result{Status: agent.TaskStatusNext},
		agent.Result{Status: agent.TaskStatusNext},
		agent.Result{Status: agent.TaskStatusNext},
	)

	cfg := config.NewConfig()
	l := NewLoop(ag, taskMgr, nil, cfg, projectDir)
	l.SetAnalysis(greenfieldAnalysis())

	l.SetOptions(&Options{
		MaxIterationsPerTask: 3, // Only allow 3 iterations
	})

	err := l.Run(context.Background(), "int-test-session")
	// Loop should complete (no more tasks) after marking task as failed
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	// Task should be marked as failed
	updated, _ := taskMgr.GetByID("TASK-001")
	if updated.Status != task.StatusFailed {
		t.Errorf("Task status = %v, want failed", updated.Status)
	}

	// Agent should have been called exactly 3 times
	calls := ag.GetCalls()
	if len(calls) != 3 {
		t.Errorf("Agent called %d times, want 3", len(calls))
	}
}

// =============================================================================
// Headless Mode Integration Tests
// =============================================================================

func TestIntegration_HeadlessMode_TextOutput(t *testing.T) {
	projectDir := setupIntegrationProject(t)

	task1 := task.NewTask("TASK-001", "Test task", "Description")
	taskMgr := createIntegrationTaskManager(t, projectDir, task1)

	ag := newScenarioAgent("test-agent")
	ag.SetScenario("TASK-001", agent.Result{Status: agent.TaskStatusDone, SessionID: "s1"})

	cfg := config.NewConfig()
	l := NewLoop(ag, taskMgr, nil, cfg, projectDir)
	l.SetAnalysis(greenfieldAnalysis())

	// Capture headless text output
	var buf bytes.Buffer
	headless := NewHeadlessRunner(&HeadlessConfig{
		OutputFormat: OutputFormatText,
		Writer:       &buf,
	})

	l.SetOptions(&Options{
		MaxIterationsPerTask: 3,
		OnEvent: func(e Event) {
			headless.HandleEvent(e)
		},
	})

	err := l.Run(context.Background(), "headless-test-session")
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	// Verify text output contains expected elements
	output := buf.String()
	// The headless runner uses TaskName in its output, not TaskID
	if !strings.Contains(output, "Test task") {
		t.Errorf("text output missing task name, got: %q", output)
	}
	// Check for loop start and completion events
	if !strings.Contains(output, "Loop started") || !strings.Contains(output, "Loop completed") {
		t.Errorf("text output missing loop status updates, got: %q", output)
	}
}

func TestIntegration_HeadlessMode_JSONOutput(t *testing.T) {
	projectDir := setupIntegrationProject(t)

	task1 := task.NewTask("TASK-001", "Test task", "Description")
	task2 := task.NewTask("TASK-002", "Test task 2", "Description 2")
	taskMgr := createIntegrationTaskManager(t, projectDir, task1, task2)

	ag := newScenarioAgent("test-agent")
	ag.SetScenario("TASK-001", agent.Result{Status: agent.TaskStatusDone, SessionID: "s1"})
	ag.SetScenario("TASK-002", agent.Result{Status: agent.TaskStatusDone, SessionID: "s2"})

	cfg := config.NewConfig()
	l := NewLoop(ag, taskMgr, nil, cfg, projectDir)
	l.SetAnalysis(greenfieldAnalysis())

	// Capture headless JSON output
	var buf bytes.Buffer
	headless := NewHeadlessRunner(&HeadlessConfig{
		OutputFormat: OutputFormatJSON,
		Writer:       &buf,
	})

	l.SetOptions(&Options{
		MaxIterationsPerTask: 3,
		OnEvent: func(e Event) {
			headless.HandleEvent(e)
		},
	})

	err := l.Run(context.Background(), "headless-json-session")
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	// Write final JSON output using the loop context
	headless.WriteJSONOutput(l.context)

	// Parse and verify JSON output
	var jsonOut JSONOutput
	if err := json.Unmarshal(buf.Bytes(), &jsonOut); err != nil {
		t.Fatalf("failed to parse JSON output: %v", err)
	}

	if jsonOut.SessionID != "headless-json-session" {
		t.Errorf("SessionID = %q, want %q", jsonOut.SessionID, "headless-json-session")
	}
	if jsonOut.TasksComplete != 2 {
		t.Errorf("TasksComplete = %d, want 2", jsonOut.TasksComplete)
	}
	if jsonOut.FinalState != "completed" {
		t.Errorf("FinalState = %q, want %q", jsonOut.FinalState, "completed")
	}
	// Verify events were collected
	if len(jsonOut.Events) == 0 {
		t.Error("Events should not be empty")
	}
}

func TestIntegration_HeadlessMode_WithHooks(t *testing.T) {
	projectDir := setupIntegrationProject(t)

	// Create a marker file that hooks will write to
	markerFile := filepath.Join(projectDir, "hook_ran.txt")

	task1 := task.NewTask("TASK-001", "Test task", "Description")
	taskMgr := createIntegrationTaskManager(t, projectDir, task1)

	ag := newScenarioAgent("test-agent")
	ag.SetScenario("TASK-001", agent.Result{Status: agent.TaskStatusDone, SessionID: "s1"})

	// Create a shell hook definition
	hookDef := config.HookDefinition{
		Type:    config.HookTypeShell,
		Command: fmt.Sprintf("echo 'post-task hook ran' > %s", markerFile),
	}
	postHook := hooks.NewShellHook("marker-hook", hooks.HookPhasePost, hookDef)

	// Create hooks manager with the shell hook
	hooksMgr := hooks.NewManager(nil, []hooks.Hook{postHook})

	cfg := config.NewConfig()
	l := NewLoop(ag, taskMgr, hooksMgr, cfg, projectDir)
	l.SetAnalysis(greenfieldAnalysis())

	var buf bytes.Buffer
	headless := NewHeadlessRunner(&HeadlessConfig{
		OutputFormat: OutputFormatText,
		Writer:       &buf,
	})

	l.SetOptions(&Options{
		MaxIterationsPerTask: 3,
		OnEvent: func(e Event) {
			headless.HandleEvent(e)
		},
	})

	err := l.Run(context.Background(), "hooks-test-session")
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	// Verify hook was executed (marker file exists)
	if _, err := os.Stat(markerFile); os.IsNotExist(err) {
		t.Error("post-task hook did not run (marker file not created)")
	}
}
