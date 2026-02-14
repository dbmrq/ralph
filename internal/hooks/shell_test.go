package hooks

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/wexinc/ralph/internal/agent"
	"github.com/wexinc/ralph/internal/config"
	"github.com/wexinc/ralph/internal/task"
)

func TestNewShellHook(t *testing.T) {
	def := config.HookDefinition{
		Type:      config.HookTypeShell,
		Command:   "echo hello",
		OnFailure: config.FailureModeSkipTask,
	}
	hook := NewShellHook("test-hook", HookPhasePre, def)

	if hook.Name() != "test-hook" {
		t.Errorf("Name() = %v, want test-hook", hook.Name())
	}
	if hook.Phase() != HookPhasePre {
		t.Errorf("Phase() = %v, want %v", hook.Phase(), HookPhasePre)
	}
	if hook.Type() != config.HookTypeShell {
		t.Errorf("Type() = %v, want %v", hook.Type(), config.HookTypeShell)
	}
}

func TestShellHook_Execute_Success(t *testing.T) {
	def := config.HookDefinition{
		Type:    config.HookTypeShell,
		Command: "echo 'hello world'",
	}
	hook := NewShellHook("echo-hook", HookPhasePre, def)

	ctx := context.Background()
	hookCtx := &HookContext{
		Task:       task.NewTask("TASK-001", "Test task", "Test description"),
		Iteration:  1,
		ProjectDir: "/tmp/project",
	}

	result, err := hook.Execute(ctx, hookCtx)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if !result.IsSuccess() {
		t.Errorf("IsSuccess() = false, want true; error=%s", result.Error)
	}
	if result.ExitCode != 0 {
		t.Errorf("ExitCode = %d, want 0", result.ExitCode)
	}
	if !strings.Contains(result.Output, "hello world") {
		t.Errorf("Output = %v, want to contain 'hello world'", result.Output)
	}
}

func TestShellHook_Execute_Failure(t *testing.T) {
	def := config.HookDefinition{
		Type:      config.HookTypeShell,
		Command:   "exit 1",
		OnFailure: config.FailureModeAbortLoop,
	}
	hook := NewShellHook("fail-hook", HookPhasePre, def)

	ctx := context.Background()
	hookCtx := &HookContext{
		Task:       task.NewTask("TASK-001", "Test", "Desc"),
		Iteration:  1,
		ProjectDir: "/tmp",
	}

	result, err := hook.Execute(ctx, hookCtx)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if result.IsSuccess() {
		t.Error("IsSuccess() = true, want false")
	}
	if result.ExitCode != 1 {
		t.Errorf("ExitCode = %d, want 1", result.ExitCode)
	}
	if result.FailureMode != config.FailureModeAbortLoop {
		t.Errorf("FailureMode = %v, want %v", result.FailureMode, config.FailureModeAbortLoop)
	}
}

func TestShellHook_Execute_ContextCancellation(t *testing.T) {
	def := config.HookDefinition{
		Type:    config.HookTypeShell,
		Command: "sleep 10",
	}
	hook := NewShellHook("sleep-hook", HookPhasePre, def)

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	hookCtx := &HookContext{
		Task:       task.NewTask("TASK-001", "Test", "Desc"),
		Iteration:  1,
		ProjectDir: "/tmp",
	}

	result, err := hook.Execute(ctx, hookCtx)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if result.IsSuccess() {
		t.Error("expected failure due to context cancellation")
	}
}

func TestShellHook_Execute_NilContext(t *testing.T) {
	def := config.HookDefinition{
		Type:    config.HookTypeShell,
		Command: "echo test",
	}
	hook := NewShellHook("test-hook", HookPhasePre, def)

	_, err := hook.Execute(context.Background(), nil)
	if err == nil {
		t.Error("Execute() with nil HookContext should return error")
	}
}

func TestShellHook_Execute_EmptyCommand(t *testing.T) {
	def := config.HookDefinition{
		Type:    config.HookTypeShell,
		Command: "",
	}
	hook := NewShellHook("empty-hook", HookPhasePre, def)

	hookCtx := &HookContext{
		Task:       task.NewTask("TASK-001", "Test", "Desc"),
		Iteration:  1,
		ProjectDir: "/tmp",
	}

	_, err := hook.Execute(context.Background(), hookCtx)
	if err == nil {
		t.Error("Execute() with empty command should return error")
	}
}

func TestShellHook_Execute_EnvironmentVariables(t *testing.T) {
	def := config.HookDefinition{
		Type:    config.HookTypeShell,
		Command: "echo $TASK_ID $TASK_NAME $ITERATION $PROJECT_DIR",
	}
	hook := NewShellHook("env-hook", HookPhasePre, def)

	ctx := context.Background()
	hookCtx := &HookContext{
		Task:       task.NewTask("TASK-001", "Test Task", "Desc"),
		Iteration:  3,
		ProjectDir: "/home/user/project",
	}

	result, err := hook.Execute(ctx, hookCtx)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if !result.IsSuccess() {
		t.Errorf("IsSuccess() = false; error=%s", result.Error)
	}

	// Check that environment variables are available
	if !strings.Contains(result.Output, "TASK-001") {
		t.Errorf("Output should contain TASK_ID; got: %s", result.Output)
	}
	if !strings.Contains(result.Output, "Test Task") {
		t.Errorf("Output should contain TASK_NAME; got: %s", result.Output)
	}
	if !strings.Contains(result.Output, "3") {
		t.Errorf("Output should contain ITERATION; got: %s", result.Output)
	}
	if !strings.Contains(result.Output, "/home/user/project") {
		t.Errorf("Output should contain PROJECT_DIR; got: %s", result.Output)
	}
}

func TestShellHook_Execute_VariableExpansion(t *testing.T) {
	def := config.HookDefinition{
		Type:    config.HookTypeShell,
		Command: "echo 'Starting ${TASK_ID}: ${TASK_NAME}'",
	}
	hook := NewShellHook("expand-hook", HookPhasePre, def)

	ctx := context.Background()
	hookCtx := &HookContext{
		Task:       task.NewTask("TASK-002", "Build Feature", "Desc"),
		Iteration:  1,
		ProjectDir: "/tmp",
	}

	result, err := hook.Execute(ctx, hookCtx)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if !result.IsSuccess() {
		t.Errorf("IsSuccess() = false; error=%s", result.Error)
	}

	expected := "Starting TASK-002: Build Feature"
	if !strings.Contains(result.Output, expected) {
		t.Errorf("Output = %v, want to contain %q", result.Output, expected)
	}
}

func TestShellHook_Execute_PostTaskWithAgentResult(t *testing.T) {
	def := config.HookDefinition{
		Type:    config.HookTypeShell,
		Command: "echo $AGENT_STATUS $AGENT_EXIT_CODE",
	}
	hook := NewShellHook("post-hook", HookPhasePost, def)

	ctx := context.Background()
	hookCtx := &HookContext{
		Task:      task.NewTask("TASK-001", "Test", "Desc"),
		Iteration: 1,
		Result: &agent.Result{
			Output:   "Agent completed successfully",
			ExitCode: 0,
			Status:   agent.TaskStatusDone,
		},
		ProjectDir: "/tmp",
	}

	result, err := hook.Execute(ctx, hookCtx)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if !result.IsSuccess() {
		t.Errorf("IsSuccess() = false; error=%s", result.Error)
	}

	if !strings.Contains(result.Output, "DONE") {
		t.Errorf("Output should contain agent status (DONE); got: %s", result.Output)
	}
	if !strings.Contains(result.Output, "0") {
		t.Errorf("Output should contain agent exit code; got: %s", result.Output)
	}
}

func TestShellHook_Execute_NilTask(t *testing.T) {
	def := config.HookDefinition{
		Type:    config.HookTypeShell,
		Command: "echo $ITERATION",
	}
	hook := NewShellHook("nil-task-hook", HookPhasePre, def)

	ctx := context.Background()
	hookCtx := &HookContext{
		Task:       nil,
		Iteration:  5,
		ProjectDir: "/tmp",
	}

	result, err := hook.Execute(ctx, hookCtx)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if !result.IsSuccess() {
		t.Errorf("IsSuccess() = false; error=%s", result.Error)
	}

	if !strings.Contains(result.Output, "5") {
		t.Errorf("Output should contain ITERATION; got: %s", result.Output)
	}
}

func TestShellHook_Execute_Stderr(t *testing.T) {
	def := config.HookDefinition{
		Type:    config.HookTypeShell,
		Command: "echo 'stdout output'; echo 'stderr output' >&2",
	}
	hook := NewShellHook("stderr-hook", HookPhasePre, def)

	ctx := context.Background()
	hookCtx := &HookContext{
		Task:       task.NewTask("TASK-001", "Test", "Desc"),
		Iteration:  1,
		ProjectDir: "/tmp",
	}

	result, err := hook.Execute(ctx, hookCtx)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if !result.IsSuccess() {
		t.Errorf("IsSuccess() = false; error=%s", result.Error)
	}

	if !strings.Contains(result.Output, "stdout output") {
		t.Errorf("Output should contain stdout; got: %s", result.Output)
	}
	if !strings.Contains(result.Output, "stderr output") {
		t.Errorf("Output should contain stderr; got: %s", result.Output)
	}
}

func TestShellHook_Execute_FailureModes(t *testing.T) {
	tests := []struct {
		name        string
		failureMode config.FailureMode
		shouldAbort bool
		shouldSkip  bool
		shouldAsk   bool
		shouldWarn  bool
	}{
		{"abort_loop", config.FailureModeAbortLoop, true, false, false, false},
		{"skip_task", config.FailureModeSkipTask, false, true, false, false},
		{"ask_agent", config.FailureModeAskAgent, false, false, true, false},
		{"warn_continue", config.FailureModeWarnContinue, false, false, false, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			def := config.HookDefinition{
				Type:      config.HookTypeShell,
				Command:   "exit 1",
				OnFailure: tt.failureMode,
			}
			hook := NewShellHook("fail-hook", HookPhasePre, def)

			ctx := context.Background()
			hookCtx := &HookContext{
				Task:       task.NewTask("TASK-001", "Test", "Desc"),
				Iteration:  1,
				ProjectDir: "/tmp",
			}

			result, err := hook.Execute(ctx, hookCtx)
			if err != nil {
				t.Fatalf("Execute() error = %v", err)
			}

			if result.ShouldAbort() != tt.shouldAbort {
				t.Errorf("ShouldAbort() = %v, want %v", result.ShouldAbort(), tt.shouldAbort)
			}
			if result.ShouldSkipTask() != tt.shouldSkip {
				t.Errorf("ShouldSkipTask() = %v, want %v", result.ShouldSkipTask(), tt.shouldSkip)
			}
			if result.ShouldAskAgent() != tt.shouldAsk {
				t.Errorf("ShouldAskAgent() = %v, want %v", result.ShouldAskAgent(), tt.shouldAsk)
			}
			if result.ShouldWarnAndContinue() != tt.shouldWarn {
				t.Errorf("ShouldWarnAndContinue() = %v, want %v", result.ShouldWarnAndContinue(), tt.shouldWarn)
			}
		})
	}
}

func TestCreateHooksFromConfig_ShellHooks(t *testing.T) {
	cfg := &config.HooksConfig{
		PreTask: []config.HookDefinition{
			{Type: config.HookTypeShell, Command: "echo pre"},
			{Command: "echo default"}, // should default to shell
		},
	}

	preHooks, _, err := CreateHooksFromConfig(cfg)
	if err != nil {
		t.Fatalf("CreateHooksFromConfig() error = %v", err)
	}

	// Verify hooks are ShellHook instances
	for i, hook := range preHooks {
		if _, ok := hook.(*ShellHook); !ok {
			t.Errorf("preHooks[%d] is %T, want *ShellHook", i, hook)
		}
	}

	// Execute the hooks to verify they work
	ctx := context.Background()
	hookCtx := &HookContext{
		Task:       task.NewTask("TASK-001", "Test", "Desc"),
		Iteration:  1,
		ProjectDir: "/tmp",
	}

	for i, hook := range preHooks {
		result, err := hook.Execute(ctx, hookCtx)
		if err != nil {
			t.Errorf("preHooks[%d].Execute() error = %v", i, err)
		}
		if !result.IsSuccess() {
			t.Errorf("preHooks[%d].Execute() not successful", i)
		}
	}
}

func TestShellHook_Execute_StderrOnly(t *testing.T) {
	// Test when there's only stderr output, no stdout
	def := config.HookDefinition{
		Type:    config.HookTypeShell,
		Command: "echo 'error message' >&2",
	}
	hook := NewShellHook("stderr-only-hook", HookPhasePre, def)

	ctx := context.Background()
	hookCtx := &HookContext{
		Task:       task.NewTask("TASK-001", "Test", "Desc"),
		Iteration:  1,
		ProjectDir: "/tmp",
	}

	result, err := hook.Execute(ctx, hookCtx)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if !result.IsSuccess() {
		t.Errorf("IsSuccess() = false; error=%s", result.Error)
	}

	if !strings.Contains(result.Output, "error message") {
		t.Errorf("Output should contain stderr; got: %s", result.Output)
	}
}

func TestShellHook_Execute_ExpandVariablesWithNilResult(t *testing.T) {
	// Test variable expansion when Result is nil (pre-task hooks)
	def := config.HookDefinition{
		Type:    config.HookTypeShell,
		Command: "echo '${TASK_ID} ${TASK_NAME} ${TASK_STATUS} ${ITERATION}'",
	}
	hook := NewShellHook("expand-hook", HookPhasePre, def)

	ctx := context.Background()
	tk := task.NewTask("TASK-005", "My Task", "Description")
	hookCtx := &HookContext{
		Task:       tk,
		Result:     nil, // Pre-task, no result
		Iteration:  7,
		ProjectDir: "/tmp",
	}

	result, err := hook.Execute(ctx, hookCtx)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if !result.IsSuccess() {
		t.Errorf("IsSuccess() = false; error=%s", result.Error)
	}

	if !strings.Contains(result.Output, "TASK-005") {
		t.Errorf("Output should contain TASK_ID; got: %s", result.Output)
	}
	if !strings.Contains(result.Output, "My Task") {
		t.Errorf("Output should contain TASK_NAME; got: %s", result.Output)
	}
	if !strings.Contains(result.Output, "7") {
		t.Errorf("Output should contain ITERATION; got: %s", result.Output)
	}
}

func TestShellHook_Execute_AllEnvironmentVariables(t *testing.T) {
	// Test all available environment variables
	def := config.HookDefinition{
		Type:    config.HookTypeShell,
		Command: "echo \"ID:$TASK_ID NAME:$TASK_NAME DESC:$TASK_DESCRIPTION STATUS:$TASK_STATUS ITER:$ITERATION DIR:$PROJECT_DIR OUT:$AGENT_OUTPUT EXIT:$AGENT_EXIT_CODE ASTATUS:$AGENT_STATUS\"",
	}
	hook := NewShellHook("all-env-hook", HookPhasePost, def)

	ctx := context.Background()
	tk := task.NewTask("T-001", "TaskName", "TaskDesc")
	hookCtx := &HookContext{
		Task:      tk,
		Iteration: 2,
		Result: &agent.Result{
			Output:   "agent output text",
			ExitCode: 0,
			Status:   agent.TaskStatusNext,
		},
		ProjectDir: "/my/project",
	}

	result, err := hook.Execute(ctx, hookCtx)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if !result.IsSuccess() {
		t.Errorf("IsSuccess() = false; error=%s", result.Error)
	}

	checks := []string{
		"ID:T-001",
		"NAME:TaskName",
		"DESC:TaskDesc",
		"ITER:2",
		"DIR:/my/project",
		"EXIT:0",
		"ASTATUS:NEXT",
	}

	for _, check := range checks {
		if !strings.Contains(result.Output, check) {
			t.Errorf("Output should contain %q; got: %s", check, result.Output)
		}
	}
}

func TestShellHook_Execute_NonExitError(t *testing.T) {
	// Test when command itself fails to execute (not found)
	def := config.HookDefinition{
		Type:      config.HookTypeShell,
		Command:   "nonexistent_command_12345",
		OnFailure: config.FailureModeWarnContinue,
	}
	hook := NewShellHook("nonexist-hook", HookPhasePre, def)

	ctx := context.Background()
	hookCtx := &HookContext{
		Task:       task.NewTask("TASK-001", "Test", "Desc"),
		Iteration:  1,
		ProjectDir: "/tmp",
	}

	result, err := hook.Execute(ctx, hookCtx)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if result.IsSuccess() {
		t.Error("IsSuccess() = true, want false (command not found)")
	}
	// Should have non-zero exit code
	if result.ExitCode == 0 {
		t.Error("ExitCode = 0, want non-zero")
	}
}

func TestShellHook_Execute_DefaultFailureMode(t *testing.T) {
	// Test that default failure mode is warn_continue
	def := config.HookDefinition{
		Type:    config.HookTypeShell,
		Command: "exit 1",
		// OnFailure not set - should default to warn_continue
	}
	hook := NewShellHook("default-mode-hook", HookPhasePre, def)

	ctx := context.Background()
	hookCtx := &HookContext{
		Task:       task.NewTask("TASK-001", "Test", "Desc"),
		Iteration:  1,
		ProjectDir: "/tmp",
	}

	result, err := hook.Execute(ctx, hookCtx)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if result.IsSuccess() {
		t.Error("IsSuccess() = true, want false")
	}
	// Default failure mode should be warn_continue
	if !result.ShouldWarnAndContinue() {
		t.Errorf("ShouldWarnAndContinue() = false, want true (default); FailureMode = %v", result.FailureMode)
	}
}
