package hooks

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/wexinc/ralph/internal/agent"
	"github.com/wexinc/ralph/internal/config"
	"github.com/wexinc/ralph/internal/task"
)

// mockHook is a test double for the Hook interface.
type mockHook struct {
	name        string
	phase       HookPhase
	hookType    config.HookType
	definition  config.HookDefinition
	executeFunc func(ctx context.Context, hookCtx *HookContext) (*HookResult, error)
}

func (m *mockHook) Name() string                      { return m.name }
func (m *mockHook) Phase() HookPhase                  { return m.phase }
func (m *mockHook) Type() config.HookType             { return m.hookType }
func (m *mockHook) Definition() config.HookDefinition { return m.definition }
func (m *mockHook) Execute(ctx context.Context, hookCtx *HookContext) (*HookResult, error) {
	if m.executeFunc != nil {
		return m.executeFunc(ctx, hookCtx)
	}
	return &HookResult{Success: true, ExitCode: 0}, nil
}

func newSuccessHook(name string, phase HookPhase) *mockHook {
	return &mockHook{
		name:     name,
		phase:    phase,
		hookType: config.HookTypeShell,
		executeFunc: func(ctx context.Context, hookCtx *HookContext) (*HookResult, error) {
			return &HookResult{Success: true, ExitCode: 0, Output: "ok"}, nil
		},
	}
}

func newFailureHook(name string, phase HookPhase, failureMode config.FailureMode) *mockHook {
	return &mockHook{
		name:       name,
		phase:      phase,
		hookType:   config.HookTypeShell,
		definition: config.HookDefinition{OnFailure: failureMode},
		executeFunc: func(ctx context.Context, hookCtx *HookContext) (*HookResult, error) {
			return &HookResult{
				Success:     false,
				ExitCode:    1,
				Error:       "hook failed",
				FailureMode: failureMode,
			}, nil
		},
	}
}

func newErrorHook(name string, phase HookPhase) *mockHook {
	return &mockHook{
		name:     name,
		phase:    phase,
		hookType: config.HookTypeShell,
		executeFunc: func(ctx context.Context, hookCtx *HookContext) (*HookResult, error) {
			return nil, errors.New("execution error")
		},
	}
}

func TestNewManager(t *testing.T) {
	preHooks := []Hook{newSuccessHook("pre1", HookPhasePre)}
	postHooks := []Hook{newSuccessHook("post1", HookPhasePost)}

	m := NewManager(preHooks, postHooks)

	if len(m.PreTaskHooks()) != 1 {
		t.Errorf("PreTaskHooks() len = %d, want 1", len(m.PreTaskHooks()))
	}
	if len(m.PostTaskHooks()) != 1 {
		t.Errorf("PostTaskHooks() len = %d, want 1", len(m.PostTaskHooks()))
	}
}

func TestManager_HasHooks(t *testing.T) {
	tests := []struct {
		name    string
		pre     []Hook
		post    []Hook
		hasPre  bool
		hasPost bool
	}{
		{"empty", nil, nil, false, false},
		{"pre only", []Hook{newSuccessHook("h", HookPhasePre)}, nil, true, false},
		{"post only", nil, []Hook{newSuccessHook("h", HookPhasePost)}, false, true},
		{"both", []Hook{newSuccessHook("h", HookPhasePre)}, []Hook{newSuccessHook("h", HookPhasePost)}, true, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := NewManager(tt.pre, tt.post)
			if got := m.HasPreTaskHooks(); got != tt.hasPre {
				t.Errorf("HasPreTaskHooks() = %v, want %v", got, tt.hasPre)
			}
			if got := m.HasPostTaskHooks(); got != tt.hasPost {
				t.Errorf("HasPostTaskHooks() = %v, want %v", got, tt.hasPost)
			}
		})
	}
}

func TestManager_ExecutePreTaskHooks_AllSuccess(t *testing.T) {
	hooks := []Hook{
		newSuccessHook("pre1", HookPhasePre),
		newSuccessHook("pre2", HookPhasePre),
	}
	m := NewManager(hooks, nil)

	ctx := context.Background()
	hookCtx := &HookContext{
		Task:       task.NewTask("TASK-001", "Test", "Desc"),
		Iteration:  1,
		ProjectDir: "/tmp",
	}

	result := m.ExecutePreTaskHooks(ctx, hookCtx)

	if !result.AllSuccess {
		t.Error("AllSuccess = false, want true")
	}
	if result.Action != ManagerActionContinue {
		t.Errorf("Action = %v, want %v", result.Action, ManagerActionContinue)
	}
	if len(result.Results) != 2 {
		t.Errorf("len(Results) = %d, want 2", len(result.Results))
	}
}

func TestManager_ExecutePreTaskHooks_Empty(t *testing.T) {
	m := NewManager(nil, nil)
	ctx := context.Background()
	hookCtx := &HookContext{
		Task:       task.NewTask("TASK-001", "Test", "Desc"),
		Iteration:  1,
		ProjectDir: "/tmp",
	}

	result := m.ExecutePreTaskHooks(ctx, hookCtx)

	if !result.AllSuccess {
		t.Error("AllSuccess = false, want true (empty hooks should succeed)")
	}
	if result.Action != ManagerActionContinue {
		t.Errorf("Action = %v, want %v", result.Action, ManagerActionContinue)
	}
}

func TestManager_ExecuteHooks_FailureAbortLoop(t *testing.T) {
	hooks := []Hook{
		newSuccessHook("pre1", HookPhasePre),
		newFailureHook("pre2", HookPhasePre, config.FailureModeAbortLoop),
		newSuccessHook("pre3", HookPhasePre), // should not execute
	}
	m := NewManager(hooks, nil)

	ctx := context.Background()
	hookCtx := &HookContext{
		Task:       task.NewTask("TASK-001", "Test", "Desc"),
		Iteration:  1,
		ProjectDir: "/tmp",
	}

	result := m.ExecutePreTaskHooks(ctx, hookCtx)

	if result.AllSuccess {
		t.Error("AllSuccess = true, want false")
	}
	if result.Action != ManagerActionAbortLoop {
		t.Errorf("Action = %v, want %v", result.Action, ManagerActionAbortLoop)
	}
	if len(result.Results) != 2 {
		t.Errorf("len(Results) = %d, want 2 (third hook should not execute)", len(result.Results))
	}
	if result.FailedHook == nil || result.FailedHook.Name() != "pre2" {
		t.Error("FailedHook should be 'pre2'")
	}
}

func TestManager_ExecuteHooks_FailureSkipTask(t *testing.T) {
	hooks := []Hook{
		newSuccessHook("pre1", HookPhasePre),
		newFailureHook("pre2", HookPhasePre, config.FailureModeSkipTask),
		newSuccessHook("pre3", HookPhasePre), // should not execute
	}
	m := NewManager(hooks, nil)

	ctx := context.Background()
	hookCtx := &HookContext{
		Task:       task.NewTask("TASK-001", "Test", "Desc"),
		Iteration:  1,
		ProjectDir: "/tmp",
	}

	result := m.ExecutePreTaskHooks(ctx, hookCtx)

	if result.Action != ManagerActionSkipTask {
		t.Errorf("Action = %v, want %v", result.Action, ManagerActionSkipTask)
	}
	if len(result.Results) != 2 {
		t.Errorf("len(Results) = %d, want 2 (third hook should not execute)", len(result.Results))
	}
}

func TestManager_ExecuteHooks_FailureAskAgent(t *testing.T) {
	hooks := []Hook{
		newFailureHook("pre1", HookPhasePre, config.FailureModeAskAgent),
	}
	m := NewManager(hooks, nil)

	ctx := context.Background()
	hookCtx := &HookContext{
		Task:       task.NewTask("TASK-001", "Test", "Desc"),
		Iteration:  1,
		ProjectDir: "/tmp",
	}

	result := m.ExecutePreTaskHooks(ctx, hookCtx)

	if result.Action != ManagerActionAskAgent {
		t.Errorf("Action = %v, want %v", result.Action, ManagerActionAskAgent)
	}
	if result.FailedHook == nil {
		t.Error("FailedHook should not be nil")
	}
}

func TestManager_ExecuteHooks_FailureWarnContinue(t *testing.T) {
	hooks := []Hook{
		newFailureHook("pre1", HookPhasePre, config.FailureModeWarnContinue),
		newSuccessHook("pre2", HookPhasePre),
		newFailureHook("pre3", HookPhasePre, config.FailureModeWarnContinue),
	}
	m := NewManager(hooks, nil)

	ctx := context.Background()
	hookCtx := &HookContext{
		Task:       task.NewTask("TASK-001", "Test", "Desc"),
		Iteration:  1,
		ProjectDir: "/tmp",
	}

	result := m.ExecutePreTaskHooks(ctx, hookCtx)

	// Should continue despite failures
	if result.Action != ManagerActionContinue {
		t.Errorf("Action = %v, want %v", result.Action, ManagerActionContinue)
	}
	// But AllSuccess should be false
	if result.AllSuccess {
		t.Error("AllSuccess = true, want false (some hooks failed)")
	}
	// All hooks should have executed
	if len(result.Results) != 3 {
		t.Errorf("len(Results) = %d, want 3", len(result.Results))
	}
}

func TestManager_ExecuteHooks_ExecutionError(t *testing.T) {
	hooks := []Hook{
		newSuccessHook("pre1", HookPhasePre),
		newErrorHook("pre2", HookPhasePre),
	}
	m := NewManager(hooks, nil)

	ctx := context.Background()
	hookCtx := &HookContext{
		Task:       task.NewTask("TASK-001", "Test", "Desc"),
		Iteration:  1,
		ProjectDir: "/tmp",
	}

	result := m.ExecutePreTaskHooks(ctx, hookCtx)

	// Execution errors should be treated as abort_loop
	if result.Action != ManagerActionAbortLoop {
		t.Errorf("Action = %v, want %v", result.Action, ManagerActionAbortLoop)
	}
	if result.AllSuccess {
		t.Error("AllSuccess = true, want false")
	}
}

func TestManager_ExecuteHooks_ContextCancellation(t *testing.T) {
	// Create a slow hook that will be canceled
	slowHook := &mockHook{
		name:     "slow",
		phase:    HookPhasePre,
		hookType: config.HookTypeShell,
		executeFunc: func(ctx context.Context, hookCtx *HookContext) (*HookResult, error) {
			select {
			case <-ctx.Done():
				return &HookResult{Success: false, Error: "canceled"}, nil
			case <-time.After(10 * time.Second):
				return &HookResult{Success: true}, nil
			}
		},
	}

	m := NewManager([]Hook{slowHook}, nil)

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	hookCtx := &HookContext{
		Task:       task.NewTask("TASK-001", "Test", "Desc"),
		Iteration:  1,
		ProjectDir: "/tmp",
	}

	result := m.ExecutePreTaskHooks(ctx, hookCtx)

	if result.AllSuccess {
		t.Error("AllSuccess = true, want false (canceled)")
	}
}

func TestManager_ExecutePostTaskHooks(t *testing.T) {
	hooks := []Hook{
		newSuccessHook("post1", HookPhasePost),
		newSuccessHook("post2", HookPhasePost),
	}
	m := NewManager(nil, hooks)

	ctx := context.Background()
	hookCtx := &HookContext{
		Task:      task.NewTask("TASK-001", "Test", "Desc"),
		Iteration: 1,
		Result: &agent.Result{
			Output:   "task completed",
			ExitCode: 0,
			Status:   agent.TaskStatusDone,
		},
		ProjectDir: "/tmp",
	}

	result := m.ExecutePostTaskHooks(ctx, hookCtx)

	if !result.AllSuccess {
		t.Error("AllSuccess = false, want true")
	}
	if result.Action != ManagerActionContinue {
		t.Errorf("Action = %v, want %v", result.Action, ManagerActionContinue)
	}
	if len(result.Results) != 2 {
		t.Errorf("len(Results) = %d, want 2", len(result.Results))
	}
}

func TestManager_Logger(t *testing.T) {
	hooks := []Hook{
		newSuccessHook("pre1", HookPhasePre),
		newSuccessHook("pre2", HookPhasePre),
	}
	m := NewManager(hooks, nil)

	var loggedHooks []string
	m.Logger = func(phase HookPhase, hook Hook, result *HookResult) {
		loggedHooks = append(loggedHooks, hook.Name())
	}

	ctx := context.Background()
	hookCtx := &HookContext{
		Task:       task.NewTask("TASK-001", "Test", "Desc"),
		Iteration:  1,
		ProjectDir: "/tmp",
	}

	m.ExecutePreTaskHooks(ctx, hookCtx)

	if len(loggedHooks) != 2 {
		t.Errorf("logged %d hooks, want 2", len(loggedHooks))
	}
	if loggedHooks[0] != "pre1" || loggedHooks[1] != "pre2" {
		t.Errorf("logged hooks = %v, want [pre1, pre2]", loggedHooks)
	}
}

func TestManager_GetFailedHookInfo(t *testing.T) {
	hook := newFailureHook("test-hook", HookPhasePre, config.FailureModeSkipTask)
	m := NewManager([]Hook{hook}, nil)

	ctx := context.Background()
	hookCtx := &HookContext{
		Task:       task.NewTask("TASK-001", "Test", "Desc"),
		Iteration:  1,
		ProjectDir: "/tmp",
	}

	result := m.ExecutePreTaskHooks(ctx, hookCtx)

	info := m.GetFailedHookInfo(result)
	if info == "" {
		t.Error("GetFailedHookInfo() returned empty string")
	}
	if result.FailedHook == nil {
		t.Error("FailedHook should not be nil")
		return
	}
	// Check that info contains relevant details
	if !containsAll(info, "test-hook", "shell", "pre", "exit code 1") {
		t.Errorf("GetFailedHookInfo() = %q, missing expected content", info)
	}
}

func TestManager_GetFailedHookInfo_NoFailure(t *testing.T) {
	m := NewManager(nil, nil)

	result := &ManagerResult{
		AllSuccess: true,
		Action:     ManagerActionContinue,
	}

	info := m.GetFailedHookInfo(result)
	if info != "" {
		t.Errorf("GetFailedHookInfo() = %q, want empty string for no failure", info)
	}
}

func TestBuildHookContextForPreTask(t *testing.T) {
	tk := task.NewTask("TASK-001", "Test Task", "Description")
	hookCtx := BuildHookContextForPreTask(tk, 3, "/project")

	if hookCtx.Task != tk {
		t.Error("Task not set correctly")
	}
	if hookCtx.Result != nil {
		t.Error("Result should be nil for pre-task hooks")
	}
	if hookCtx.Iteration != 3 {
		t.Errorf("Iteration = %d, want 3", hookCtx.Iteration)
	}
	if hookCtx.ProjectDir != "/project" {
		t.Errorf("ProjectDir = %q, want /project", hookCtx.ProjectDir)
	}
}

func TestBuildHookContextForPostTask(t *testing.T) {
	tk := task.NewTask("TASK-001", "Test Task", "Description")
	result := &agent.Result{
		Output:   "task output",
		ExitCode: 0,
		Status:   agent.TaskStatusDone,
	}
	hookCtx := BuildHookContextForPostTask(tk, result, 2, "/project")

	if hookCtx.Task != tk {
		t.Error("Task not set correctly")
	}
	if hookCtx.Result != result {
		t.Error("Result not set correctly")
	}
	if hookCtx.Iteration != 2 {
		t.Errorf("Iteration = %d, want 2", hookCtx.Iteration)
	}
	if hookCtx.ProjectDir != "/project" {
		t.Errorf("ProjectDir = %q, want /project", hookCtx.ProjectDir)
	}
}

func TestNewManagerFromConfig(t *testing.T) {
	cfg := &config.HooksConfig{
		PreTask: []config.HookDefinition{
			{Type: config.HookTypeShell, Command: "echo pre"},
		},
		PostTask: []config.HookDefinition{
			{Type: config.HookTypeShell, Command: "echo post"},
		},
	}

	m, err := NewManagerFromConfig(cfg)
	if err != nil {
		t.Fatalf("NewManagerFromConfig() error = %v", err)
	}

	if len(m.PreTaskHooks()) != 1 {
		t.Errorf("PreTaskHooks() len = %d, want 1", len(m.PreTaskHooks()))
	}
	if len(m.PostTaskHooks()) != 1 {
		t.Errorf("PostTaskHooks() len = %d, want 1", len(m.PostTaskHooks()))
	}
}

func TestNewManagerFromConfig_InvalidType(t *testing.T) {
	cfg := &config.HooksConfig{
		PreTask: []config.HookDefinition{
			{Type: config.HookType("invalid"), Command: "echo test"},
		},
	}

	_, err := NewManagerFromConfig(cfg)
	if err == nil {
		t.Error("NewManagerFromConfig() with invalid type should return error")
	}
}

func TestNewManagerFromConfigWithAgents(t *testing.T) {
	cfg := &config.HooksConfig{
		PreTask: []config.HookDefinition{
			{Type: config.HookTypeShell, Command: "echo pre"},
		},
	}

	agentCfg := AgentHookConfig{
		WorkDir: "/tmp",
	}

	m, err := NewManagerFromConfigWithAgents(cfg, agentCfg)
	if err != nil {
		t.Fatalf("NewManagerFromConfigWithAgents() error = %v", err)
	}

	if len(m.PreTaskHooks()) != 1 {
		t.Errorf("PreTaskHooks() len = %d, want 1", len(m.PreTaskHooks()))
	}
}

// Helper function to check if a string contains all substrings
func containsAll(s string, substrings ...string) bool {
	for _, sub := range substrings {
		found := false
		for i := 0; i <= len(s)-len(sub); i++ {
			if s[i:i+len(sub)] == sub {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

func TestManager_ExecuteHooks_ContextCanceledBeforeStart(t *testing.T) {
	hooks := []Hook{
		newSuccessHook("pre1", HookPhasePre),
	}
	m := NewManager(hooks, nil)

	// Cancel context immediately
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	hookCtx := &HookContext{
		Task:       task.NewTask("TASK-001", "Test", "Desc"),
		Iteration:  1,
		ProjectDir: "/tmp",
	}

	result := m.ExecutePreTaskHooks(ctx, hookCtx)

	if result.AllSuccess {
		t.Error("AllSuccess = true, want false (context canceled)")
	}
	if result.Action != ManagerActionAbortLoop {
		t.Errorf("Action = %v, want %v", result.Action, ManagerActionAbortLoop)
	}
	// No hooks should have been executed
	if len(result.Results) != 0 {
		t.Errorf("len(Results) = %d, want 0 (context canceled before any execution)", len(result.Results))
	}
}

func TestNewManagerFromConfigWithAgents_Error(t *testing.T) {
	// Test error handling in NewManagerFromConfigWithAgents
	cfg := &config.HooksConfig{
		PreTask: []config.HookDefinition{
			{Type: config.HookType("invalid-type"), Command: "echo test"},
		},
	}

	agentCfg := AgentHookConfig{
		WorkDir: "/tmp",
	}

	_, err := NewManagerFromConfigWithAgents(cfg, agentCfg)
	if err == nil {
		t.Error("NewManagerFromConfigWithAgents() with invalid type should return error")
	}
}

func TestCreateHooksFromConfigWithAgents_PostTaskError(t *testing.T) {
	// Test error handling for invalid post-task hook type
	cfg := &config.HooksConfig{
		PreTask: []config.HookDefinition{
			{Type: config.HookTypeShell, Command: "echo pre"},
		},
		PostTask: []config.HookDefinition{
			{Type: config.HookType("bad-type"), Command: "echo post"},
		},
	}

	_, _, err := CreateHooksFromConfigWithAgents(cfg, AgentHookConfig{})
	if err == nil {
		t.Error("CreateHooksFromConfigWithAgents() with invalid post-task type should return error")
	}
	if !containsAll(err.Error(), "post-task", "hook") {
		t.Errorf("Error should mention post-task hook; got: %v", err)
	}
}

func TestManager_ExecuteHooks_MixedFailureModes(t *testing.T) {
	// Test multiple hooks with different failure modes
	// warn_continue followed by skip_task
	hooks := []Hook{
		newFailureHook("pre1", HookPhasePre, config.FailureModeWarnContinue),
		newSuccessHook("pre2", HookPhasePre),
		newFailureHook("pre3", HookPhasePre, config.FailureModeSkipTask),
		newSuccessHook("pre4", HookPhasePre), // should not execute
	}
	m := NewManager(hooks, nil)

	ctx := context.Background()
	hookCtx := &HookContext{
		Task:       task.NewTask("TASK-001", "Test", "Desc"),
		Iteration:  1,
		ProjectDir: "/tmp",
	}

	result := m.ExecutePreTaskHooks(ctx, hookCtx)

	if result.AllSuccess {
		t.Error("AllSuccess = true, want false")
	}
	if result.Action != ManagerActionSkipTask {
		t.Errorf("Action = %v, want %v", result.Action, ManagerActionSkipTask)
	}
	// First three hooks should have executed
	if len(result.Results) != 3 {
		t.Errorf("len(Results) = %d, want 3", len(result.Results))
	}
	if result.FailedHook == nil || result.FailedHook.Name() != "pre3" {
		t.Error("FailedHook should be 'pre3'")
	}
}

func TestManager_LoggerCalledForFailures(t *testing.T) {
	hooks := []Hook{
		newFailureHook("fail1", HookPhasePre, config.FailureModeWarnContinue),
	}
	m := NewManager(hooks, nil)

	var loggedPhases []HookPhase
	var loggedResults []*HookResult
	m.Logger = func(phase HookPhase, hook Hook, result *HookResult) {
		loggedPhases = append(loggedPhases, phase)
		loggedResults = append(loggedResults, result)
	}

	ctx := context.Background()
	hookCtx := &HookContext{
		Task:       task.NewTask("TASK-001", "Test", "Desc"),
		Iteration:  1,
		ProjectDir: "/tmp",
	}

	m.ExecutePreTaskHooks(ctx, hookCtx)

	if len(loggedPhases) != 1 {
		t.Errorf("logged %d hooks, want 1", len(loggedPhases))
	}
	if loggedPhases[0] != HookPhasePre {
		t.Errorf("logged phase = %v, want %v", loggedPhases[0], HookPhasePre)
	}
	if loggedResults[0].IsSuccess() {
		t.Error("logged result should be failure")
	}
}

func TestManagerResult_AllCombinations(t *testing.T) {
	tests := []struct {
		name   string
		result ManagerResult
	}{
		{
			name: "all success with continue",
			result: ManagerResult{
				AllSuccess: true,
				Action:     ManagerActionContinue,
				Results:    []*HookResult{{Success: true}},
			},
		},
		{
			name: "partial success with skip",
			result: ManagerResult{
				AllSuccess:   false,
				Action:       ManagerActionSkipTask,
				Results:      []*HookResult{{Success: true}, {Success: false}},
				FailedHook:   newFailureHook("h", HookPhasePre, config.FailureModeSkipTask),
				FailedResult: &HookResult{Success: false, ExitCode: 1},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Just verify the struct is valid
			if tt.result.Action == "" {
				t.Error("Action should not be empty")
			}
		})
	}
}

func TestBuildHookContextForPreTask_NilTask(t *testing.T) {
	hookCtx := BuildHookContextForPreTask(nil, 1, "/project")

	if hookCtx.Task != nil {
		t.Error("Task should be nil")
	}
	if hookCtx.Result != nil {
		t.Error("Result should be nil for pre-task")
	}
	if hookCtx.Iteration != 1 {
		t.Errorf("Iteration = %d, want 1", hookCtx.Iteration)
	}
	if hookCtx.ProjectDir != "/project" {
		t.Errorf("ProjectDir = %q, want /project", hookCtx.ProjectDir)
	}
}

func TestBuildHookContextForPostTask_NilResult(t *testing.T) {
	tk := task.NewTask("TASK-001", "Test", "Desc")
	hookCtx := BuildHookContextForPostTask(tk, nil, 2, "/project")

	if hookCtx.Task != tk {
		t.Error("Task not set correctly")
	}
	if hookCtx.Result != nil {
		t.Error("Result should be nil when passed nil")
	}
	if hookCtx.Iteration != 2 {
		t.Errorf("Iteration = %d, want 2", hookCtx.Iteration)
	}
}

func TestManager_GetFailedHookInfo_NilFailedResult(t *testing.T) {
	m := NewManager(nil, nil)

	result := &ManagerResult{
		AllSuccess: false,
		Action:     ManagerActionSkipTask,
		FailedHook: newFailureHook("h", HookPhasePre, config.FailureModeSkipTask),
		// FailedResult is nil
	}

	info := m.GetFailedHookInfo(result)
	if info != "" {
		t.Errorf("GetFailedHookInfo() = %q, want empty string when FailedResult is nil", info)
	}
}
