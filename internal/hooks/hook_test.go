package hooks

import (
	"context"
	"testing"

	"github.com/wexinc/ralph/internal/agent"
	"github.com/wexinc/ralph/internal/config"
	"github.com/wexinc/ralph/internal/task"
)

func TestHookPhase_String(t *testing.T) {
	tests := []struct {
		phase    HookPhase
		expected string
	}{
		{HookPhasePre, "pre"},
		{HookPhasePost, "post"},
	}

	for _, tt := range tests {
		if got := tt.phase.String(); got != tt.expected {
			t.Errorf("HookPhase.String() = %v, want %v", got, tt.expected)
		}
	}
}

func TestHookPhase_IsValid(t *testing.T) {
	tests := []struct {
		phase    HookPhase
		expected bool
	}{
		{HookPhasePre, true},
		{HookPhasePost, true},
		{HookPhase("invalid"), false},
		{HookPhase(""), false},
	}

	for _, tt := range tests {
		if got := tt.phase.IsValid(); got != tt.expected {
			t.Errorf("HookPhase(%q).IsValid() = %v, want %v", tt.phase, got, tt.expected)
		}
	}
}

func TestHookResult_IsSuccess(t *testing.T) {
	tests := []struct {
		name     string
		result   HookResult
		expected bool
	}{
		{
			name:     "success with zero exit code",
			result:   HookResult{Success: true, ExitCode: 0},
			expected: true,
		},
		{
			name:     "success with non-zero exit code",
			result:   HookResult{Success: true, ExitCode: 1},
			expected: false,
		},
		{
			name:     "failure with zero exit code",
			result:   HookResult{Success: false, ExitCode: 0},
			expected: false,
		},
		{
			name:     "failure with non-zero exit code",
			result:   HookResult{Success: false, ExitCode: 1},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.result.IsSuccess(); got != tt.expected {
				t.Errorf("HookResult.IsSuccess() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestHookResult_FailureHandling(t *testing.T) {
	tests := []struct {
		name        string
		result      HookResult
		shouldAbort bool
		shouldSkip  bool
		shouldAsk   bool
		shouldWarn  bool
	}{
		{
			name:        "success - no action needed",
			result:      HookResult{Success: true, ExitCode: 0, FailureMode: config.FailureModeAbortLoop},
			shouldAbort: false, shouldSkip: false, shouldAsk: false, shouldWarn: false,
		},
		{
			name:        "failure - abort loop",
			result:      HookResult{Success: false, ExitCode: 1, FailureMode: config.FailureModeAbortLoop},
			shouldAbort: true, shouldSkip: false, shouldAsk: false, shouldWarn: false,
		},
		{
			name:       "failure - skip task",
			result:     HookResult{Success: false, ExitCode: 1, FailureMode: config.FailureModeSkipTask},
			shouldSkip: true, shouldAbort: false, shouldAsk: false, shouldWarn: false,
		},
		{
			name:      "failure - ask agent",
			result:    HookResult{Success: false, ExitCode: 1, FailureMode: config.FailureModeAskAgent},
			shouldAsk: true, shouldAbort: false, shouldSkip: false, shouldWarn: false,
		},
		{
			name:       "failure - warn continue",
			result:     HookResult{Success: false, ExitCode: 1, FailureMode: config.FailureModeWarnContinue},
			shouldWarn: true, shouldAbort: false, shouldSkip: false, shouldAsk: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.result.ShouldAbort(); got != tt.shouldAbort {
				t.Errorf("ShouldAbort() = %v, want %v", got, tt.shouldAbort)
			}
			if got := tt.result.ShouldSkipTask(); got != tt.shouldSkip {
				t.Errorf("ShouldSkipTask() = %v, want %v", got, tt.shouldSkip)
			}
			if got := tt.result.ShouldAskAgent(); got != tt.shouldAsk {
				t.Errorf("ShouldAskAgent() = %v, want %v", got, tt.shouldAsk)
			}
			if got := tt.result.ShouldWarnAndContinue(); got != tt.shouldWarn {
				t.Errorf("ShouldWarnAndContinue() = %v, want %v", got, tt.shouldWarn)
			}
		})
	}
}

func TestBaseHook(t *testing.T) {
	def := config.HookDefinition{
		Type:      config.HookTypeShell,
		Command:   "echo hello",
		OnFailure: config.FailureModeSkipTask,
	}
	base := NewBaseHook("test-hook", HookPhasePre, def)

	t.Run("Name", func(t *testing.T) {
		if got := base.Name(); got != "test-hook" {
			t.Errorf("Name() = %v, want test-hook", got)
		}
	})

	t.Run("Phase", func(t *testing.T) {
		if got := base.Phase(); got != HookPhasePre {
			t.Errorf("Phase() = %v, want %v", got, HookPhasePre)
		}
	})

	t.Run("Type", func(t *testing.T) {
		if got := base.Type(); got != config.HookTypeShell {
			t.Errorf("Type() = %v, want %v", got, config.HookTypeShell)
		}
	})

	t.Run("Definition", func(t *testing.T) {
		if got := base.Definition(); got.Command != "echo hello" {
			t.Errorf("Definition().Command = %v, want echo hello", got.Command)
		}
	})

	t.Run("GetFailureMode", func(t *testing.T) {
		if got := base.GetFailureMode(); got != config.FailureModeSkipTask {
			t.Errorf("GetFailureMode() = %v, want %v", got, config.FailureModeSkipTask)
		}
	})

	t.Run("CreateHookResult", func(t *testing.T) {
		result := base.CreateHookResult(true, "output", "", 0)
		if !result.Success {
			t.Error("CreateHookResult().Success = false, want true")
		}
		if result.Output != "output" {
			t.Errorf("CreateHookResult().Output = %v, want output", result.Output)
		}
		if result.FailureMode != config.FailureModeSkipTask {
			t.Errorf("CreateHookResult().FailureMode = %v, want %v", result.FailureMode, config.FailureModeSkipTask)
		}
	})
}

func TestBaseHook_GetFailureMode_Default(t *testing.T) {
	// Test default failure mode when not specified
	def := config.HookDefinition{
		Type:    config.HookTypeShell,
		Command: "echo test",
		// OnFailure not set
	}
	base := NewBaseHook("test", HookPhasePre, def)

	if got := base.GetFailureMode(); got != config.FailureModeWarnContinue {
		t.Errorf("GetFailureMode() default = %v, want %v", got, config.FailureModeWarnContinue)
	}
}

func TestCreateHooksFromConfig(t *testing.T) {
	cfg := &config.HooksConfig{
		PreTask: []config.HookDefinition{
			{Type: config.HookTypeShell, Command: "echo pre1", OnFailure: config.FailureModeSkipTask},
			{Type: config.HookTypeShell, Command: "echo pre2"},
		},
		PostTask: []config.HookDefinition{
			{Type: config.HookTypeAgent, Command: "Review the changes", OnFailure: config.FailureModeAskAgent},
		},
	}

	preHooks, postHooks, err := CreateHooksFromConfig(cfg)
	if err != nil {
		t.Fatalf("CreateHooksFromConfig() error = %v", err)
	}

	if len(preHooks) != 2 {
		t.Errorf("len(preHooks) = %d, want 2", len(preHooks))
	}
	if len(postHooks) != 1 {
		t.Errorf("len(postHooks) = %d, want 1", len(postHooks))
	}

	// Verify pre-hooks
	if preHooks[0].Name() != "pre_task[0]" {
		t.Errorf("preHooks[0].Name() = %v, want pre_task[0]", preHooks[0].Name())
	}
	if preHooks[0].Phase() != HookPhasePre {
		t.Errorf("preHooks[0].Phase() = %v, want %v", preHooks[0].Phase(), HookPhasePre)
	}
	if preHooks[0].Type() != config.HookTypeShell {
		t.Errorf("preHooks[0].Type() = %v, want %v", preHooks[0].Type(), config.HookTypeShell)
	}

	// Verify post-hooks
	if postHooks[0].Phase() != HookPhasePost {
		t.Errorf("postHooks[0].Phase() = %v, want %v", postHooks[0].Phase(), HookPhasePost)
	}
	if postHooks[0].Type() != config.HookTypeAgent {
		t.Errorf("postHooks[0].Type() = %v, want %v", postHooks[0].Type(), config.HookTypeAgent)
	}
}

func TestCreateHooksFromConfig_EmptyConfig(t *testing.T) {
	cfg := &config.HooksConfig{
		PreTask:  []config.HookDefinition{},
		PostTask: []config.HookDefinition{},
	}

	preHooks, postHooks, err := CreateHooksFromConfig(cfg)
	if err != nil {
		t.Fatalf("CreateHooksFromConfig() error = %v", err)
	}

	if len(preHooks) != 0 {
		t.Errorf("len(preHooks) = %d, want 0", len(preHooks))
	}
	if len(postHooks) != 0 {
		t.Errorf("len(postHooks) = %d, want 0", len(postHooks))
	}
}

func TestCreateHooksFromConfig_DefaultType(t *testing.T) {
	// Test that empty type defaults to shell
	cfg := &config.HooksConfig{
		PreTask: []config.HookDefinition{
			{Command: "echo test"}, // Type not specified
		},
	}

	preHooks, _, err := CreateHooksFromConfig(cfg)
	if err != nil {
		t.Fatalf("CreateHooksFromConfig() error = %v", err)
	}

	if preHooks[0].Type() != config.HookTypeShell {
		t.Errorf("Type() = %v, want %v (should default to shell)", preHooks[0].Type(), config.HookTypeShell)
	}
}

func TestCreateHooksFromConfig_InvalidType(t *testing.T) {
	cfg := &config.HooksConfig{
		PreTask: []config.HookDefinition{
			{Type: config.HookType("invalid"), Command: "echo test"},
		},
	}

	_, _, err := CreateHooksFromConfig(cfg)
	if err == nil {
		t.Error("CreateHooksFromConfig() with invalid type should return error")
	}
}

func TestPlaceholderHook_Execute(t *testing.T) {
	def := config.HookDefinition{
		Type:    config.HookTypeShell,
		Command: "echo test",
	}
	hook := &placeholderHook{BaseHook: NewBaseHook("test", HookPhasePre, def)}

	ctx := context.Background()
	hookCtx := &HookContext{
		Task:       task.NewTask("TASK-001", "Test task", "Description"),
		Iteration:  1,
		ProjectDir: "/tmp/project",
	}

	_, err := hook.Execute(ctx, hookCtx)
	if err == nil {
		t.Error("placeholder Execute() should return error")
	}
}

func TestHookContext(t *testing.T) {
	tk := task.NewTask("TASK-001", "Test task", "Description")
	result := &agent.Result{
		Output:   "Agent output",
		ExitCode: 0,
		Status:   agent.TaskStatusDone,
	}

	ctx := &HookContext{
		Task:       tk,
		Result:     result,
		Iteration:  3,
		ProjectDir: "/home/user/project",
	}

	if ctx.Task.ID != "TASK-001" {
		t.Errorf("Task.ID = %v, want TASK-001", ctx.Task.ID)
	}
	if ctx.Result.Status != agent.TaskStatusDone {
		t.Errorf("Result.Status = %v, want %v", ctx.Result.Status, agent.TaskStatusDone)
	}
	if ctx.Iteration != 3 {
		t.Errorf("Iteration = %d, want 3", ctx.Iteration)
	}
	if ctx.ProjectDir != "/home/user/project" {
		t.Errorf("ProjectDir = %v, want /home/user/project", ctx.ProjectDir)
	}
}
