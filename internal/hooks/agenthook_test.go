package hooks

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/wexinc/ralph/internal/agent"
	"github.com/wexinc/ralph/internal/config"
	"github.com/wexinc/ralph/internal/task"
)

// mockAgent implements agent.Agent for testing.
type mockAgent struct {
	name        string
	available   bool
	runResult   agent.Result
	runError    error
	lastPrompt  string
	lastOptions agent.RunOptions
}

func (m *mockAgent) Name() string                       { return m.name }
func (m *mockAgent) Description() string                { return "Mock agent for testing" }
func (m *mockAgent) IsAvailable() bool                  { return m.available }
func (m *mockAgent) CheckAuth() error                   { return nil }
func (m *mockAgent) ListModels() ([]agent.Model, error) { return nil, nil }
func (m *mockAgent) GetDefaultModel() agent.Model       { return agent.Model{ID: "default"} }
func (m *mockAgent) GetSessionID() string               { return "" }

func (m *mockAgent) Run(ctx context.Context, prompt string, opts agent.RunOptions) (agent.Result, error) {
	m.lastPrompt = prompt
	m.lastOptions = opts
	return m.runResult, m.runError
}

func (m *mockAgent) Continue(ctx context.Context, sessionID string, prompt string, opts agent.RunOptions) (agent.Result, error) {
	return m.Run(ctx, prompt, opts)
}

func TestNewAgentHook(t *testing.T) {
	def := config.HookDefinition{
		Type:      config.HookTypeAgent,
		Command:   "Review the code",
		Model:     "gpt-4",
		Agent:     "custom-agent",
		OnFailure: config.FailureModeSkipTask,
	}
	cfg := AgentHookConfig{
		Registry:     agent.NewRegistry(),
		DefaultAgent: "default-agent",
		WorkDir:      "/tmp/project",
	}

	hook := NewAgentHook("test-hook", HookPhasePre, def, cfg)

	if hook.Name() != "test-hook" {
		t.Errorf("Name() = %v, want test-hook", hook.Name())
	}
	if hook.Phase() != HookPhasePre {
		t.Errorf("Phase() = %v, want %v", hook.Phase(), HookPhasePre)
	}
	if hook.Type() != config.HookTypeAgent {
		t.Errorf("Type() = %v, want %v", hook.Type(), config.HookTypeAgent)
	}
}

func TestAgentHook_Execute_Success(t *testing.T) {
	registry := agent.NewRegistry()
	mockAg := &mockAgent{
		name:      "test-agent",
		available: true,
		runResult: agent.Result{
			Output:   "Task completed successfully",
			ExitCode: 0,
			Status:   agent.TaskStatusDone,
		},
	}
	registry.Register(mockAg)

	def := config.HookDefinition{
		Type:    config.HookTypeAgent,
		Command: "Review the changes",
	}
	cfg := AgentHookConfig{
		Registry: registry,
		WorkDir:  "/tmp/project",
	}

	hook := NewAgentHook("agent-hook", HookPhasePre, def, cfg)

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
	if !strings.Contains(result.Output, "Task completed successfully") {
		t.Errorf("Output = %v, want to contain 'Task completed successfully'", result.Output)
	}
	if mockAg.lastPrompt != "Review the changes" {
		t.Errorf("lastPrompt = %v, want 'Review the changes'", mockAg.lastPrompt)
	}
}

func TestAgentHook_Execute_Failure(t *testing.T) {
	registry := agent.NewRegistry()
	mockAg := &mockAgent{
		name:      "test-agent",
		available: true,
		runResult: agent.Result{
			Output:   "Failed to complete task",
			ExitCode: 1,
			Status:   agent.TaskStatusError,
			Error:    "Build failed",
		},
	}
	registry.Register(mockAg)

	def := config.HookDefinition{
		Type:      config.HookTypeAgent,
		Command:   "Review the changes",
		OnFailure: config.FailureModeAbortLoop,
	}
	cfg := AgentHookConfig{Registry: registry}

	hook := NewAgentHook("agent-hook", HookPhasePre, def, cfg)

	hookCtx := &HookContext{
		Task:       task.NewTask("TASK-001", "Test", "Desc"),
		Iteration:  1,
		ProjectDir: "/tmp",
	}

	result, err := hook.Execute(context.Background(), hookCtx)
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

func TestAgentHook_Execute_NilHookContext(t *testing.T) {
	def := config.HookDefinition{
		Type:    config.HookTypeAgent,
		Command: "Review code",
	}
	cfg := AgentHookConfig{Registry: agent.NewRegistry()}

	hook := NewAgentHook("test-hook", HookPhasePre, def, cfg)

	_, err := hook.Execute(context.Background(), nil)
	if err == nil {
		t.Error("Execute() with nil HookContext should return error")
	}
}

func TestAgentHook_Execute_EmptyPrompt(t *testing.T) {
	def := config.HookDefinition{
		Type:    config.HookTypeAgent,
		Command: "",
	}
	cfg := AgentHookConfig{Registry: agent.NewRegistry()}

	hook := NewAgentHook("empty-hook", HookPhasePre, def, cfg)

	hookCtx := &HookContext{
		Task:       task.NewTask("TASK-001", "Test", "Desc"),
		Iteration:  1,
		ProjectDir: "/tmp",
	}

	_, err := hook.Execute(context.Background(), hookCtx)
	if err == nil {
		t.Error("Execute() with empty prompt should return error")
	}
}

func TestAgentHook_Execute_NoRegistry(t *testing.T) {
	def := config.HookDefinition{
		Type:    config.HookTypeAgent,
		Command: "Review code",
	}
	cfg := AgentHookConfig{Registry: nil}

	hook := NewAgentHook("no-registry-hook", HookPhasePre, def, cfg)

	hookCtx := &HookContext{
		Task:       task.NewTask("TASK-001", "Test", "Desc"),
		Iteration:  1,
		ProjectDir: "/tmp",
	}

	result, err := hook.Execute(context.Background(), hookCtx)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if result.IsSuccess() {
		t.Error("Execute() with nil registry should not succeed")
	}
	if !strings.Contains(result.Error, "registry") {
		t.Errorf("Error should mention registry; got: %s", result.Error)
	}
}

func TestAgentHook_Execute_AgentNotAvailable(t *testing.T) {
	registry := agent.NewRegistry()
	mockAg := &mockAgent{
		name:      "unavailable-agent",
		available: false,
	}
	registry.Register(mockAg)

	def := config.HookDefinition{
		Type:    config.HookTypeAgent,
		Command: "Review code",
		Agent:   "unavailable-agent",
	}
	cfg := AgentHookConfig{Registry: registry}

	hook := NewAgentHook("unavailable-hook", HookPhasePre, def, cfg)

	hookCtx := &HookContext{
		Task:       task.NewTask("TASK-001", "Test", "Desc"),
		Iteration:  1,
		ProjectDir: "/tmp",
	}

	result, err := hook.Execute(context.Background(), hookCtx)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if result.IsSuccess() {
		t.Error("Execute() with unavailable agent should not succeed")
	}
	if !strings.Contains(result.Error, "not available") {
		t.Errorf("Error should mention unavailable; got: %s", result.Error)
	}
}

func TestAgentHook_Execute_AgentRunError(t *testing.T) {
	registry := agent.NewRegistry()
	mockAg := &mockAgent{
		name:      "error-agent",
		available: true,
		runError:  errors.New("connection timeout"),
	}
	registry.Register(mockAg)

	def := config.HookDefinition{
		Type:    config.HookTypeAgent,
		Command: "Review code",
	}
	cfg := AgentHookConfig{Registry: registry}

	hook := NewAgentHook("error-hook", HookPhasePre, def, cfg)

	hookCtx := &HookContext{
		Task:       task.NewTask("TASK-001", "Test", "Desc"),
		Iteration:  1,
		ProjectDir: "/tmp",
	}

	result, err := hook.Execute(context.Background(), hookCtx)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if result.IsSuccess() {
		t.Error("Execute() with agent error should not succeed")
	}
	if !strings.Contains(result.Error, "connection timeout") {
		t.Errorf("Error should contain run error; got: %s", result.Error)
	}
}


func TestAgentHook_Execute_PromptExpansion(t *testing.T) {
	registry := agent.NewRegistry()
	mockAg := &mockAgent{
		name:      "test-agent",
		available: true,
		runResult: agent.Result{
			Output:   "Done",
			ExitCode: 0,
			Status:   agent.TaskStatusDone,
		},
	}
	registry.Register(mockAg)

	def := config.HookDefinition{
		Type:    config.HookTypeAgent,
		Command: "Review ${TASK_ID}: ${TASK_NAME} (iteration ${ITERATION})",
	}
	cfg := AgentHookConfig{Registry: registry, WorkDir: "/tmp/project"}

	hook := NewAgentHook("expand-hook", HookPhasePre, def, cfg)

	hookCtx := &HookContext{
		Task:       task.NewTask("TASK-002", "Build Feature", "Desc"),
		Iteration:  3,
		ProjectDir: "/tmp/project",
	}

	result, err := hook.Execute(context.Background(), hookCtx)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	expectedPrompt := "Review TASK-002: Build Feature (iteration 3)"
	if mockAg.lastPrompt != expectedPrompt {
		t.Errorf("lastPrompt = %v, want %v", mockAg.lastPrompt, expectedPrompt)
	}
	if !result.IsSuccess() {
		t.Errorf("IsSuccess() = false; error=%s", result.Error)
	}
}

func TestAgentHook_Execute_ModelSelection(t *testing.T) {
	tests := []struct {
		name          string
		defModel      string
		cfgModel      string
		expectedModel string
	}{
		{"hook definition model", "gpt-4", "claude-3", "gpt-4"},
		{"config default model", "", "claude-3", "claude-3"},
		{"no model specified", "", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			registry := agent.NewRegistry()
			mockAg := &mockAgent{
				name:      "test-agent",
				available: true,
				runResult: agent.Result{ExitCode: 0, Status: agent.TaskStatusDone},
			}
			registry.Register(mockAg)

			def := config.HookDefinition{
				Type:    config.HookTypeAgent,
				Command: "Review code",
				Model:   tt.defModel,
			}
			cfg := AgentHookConfig{
				Registry:     registry,
				DefaultModel: tt.cfgModel,
			}

			hook := NewAgentHook("model-hook", HookPhasePre, def, cfg)

			hookCtx := &HookContext{
				Task:       task.NewTask("TASK-001", "Test", "Desc"),
				Iteration:  1,
				ProjectDir: "/tmp",
			}

			_, err := hook.Execute(context.Background(), hookCtx)
			if err != nil {
				t.Fatalf("Execute() error = %v", err)
			}

			if mockAg.lastOptions.Model != tt.expectedModel {
				t.Errorf("lastOptions.Model = %v, want %v", mockAg.lastOptions.Model, tt.expectedModel)
			}
		})
	}
}

func TestAgentHook_Execute_FailureModes(t *testing.T) {
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
			registry := agent.NewRegistry()
			mockAg := &mockAgent{
				name:      "test-agent",
				available: true,
				runResult: agent.Result{
					ExitCode: 1,
					Status:   agent.TaskStatusError,
					Error:    "Task failed",
				},
			}
			registry.Register(mockAg)

			def := config.HookDefinition{
				Type:      config.HookTypeAgent,
				Command:   "Review code",
				OnFailure: tt.failureMode,
			}
			cfg := AgentHookConfig{Registry: registry}

			hook := NewAgentHook("fail-hook", HookPhasePre, def, cfg)

			hookCtx := &HookContext{
				Task:       task.NewTask("TASK-001", "Test", "Desc"),
				Iteration:  1,
				ProjectDir: "/tmp",
			}

			result, err := hook.Execute(context.Background(), hookCtx)
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

func TestCreateHooksFromConfigWithAgents_AgentHooks(t *testing.T) {
	registry := agent.NewRegistry()
	mockAg := &mockAgent{
		name:      "test-agent",
		available: true,
		runResult: agent.Result{ExitCode: 0, Status: agent.TaskStatusDone, Output: "Done"},
	}
	registry.Register(mockAg)

	cfg := &config.HooksConfig{
		PreTask: []config.HookDefinition{
			{Type: config.HookTypeAgent, Command: "Pre-task review"},
		},
		PostTask: []config.HookDefinition{
			{Type: config.HookTypeAgent, Command: "Post-task verify"},
		},
	}

	agentCfg := AgentHookConfig{Registry: registry}

	preHooks, postHooks, err := CreateHooksFromConfigWithAgents(cfg, agentCfg)
	if err != nil {
		t.Fatalf("CreateHooksFromConfigWithAgents() error = %v", err)
	}

	if len(preHooks) != 1 {
		t.Errorf("preHooks count = %d, want 1", len(preHooks))
	}
	if len(postHooks) != 1 {
		t.Errorf("postHooks count = %d, want 1", len(postHooks))
	}

	// Verify hooks are AgentHook instances
	if _, ok := preHooks[0].(*AgentHook); !ok {
		t.Errorf("preHooks[0] is %T, want *AgentHook", preHooks[0])
	}
	if _, ok := postHooks[0].(*AgentHook); !ok {
		t.Errorf("postHooks[0] is %T, want *AgentHook", postHooks[0])
	}

	// Execute pre-hook
	hookCtx := &HookContext{
		Task:       task.NewTask("TASK-001", "Test", "Desc"),
		Iteration:  1,
		ProjectDir: "/tmp",
	}

	result, err := preHooks[0].Execute(context.Background(), hookCtx)
	if err != nil {
		t.Fatalf("preHooks[0].Execute() error = %v", err)
	}
	if !result.IsSuccess() {
		t.Errorf("preHooks[0].Execute() not successful; error=%s", result.Error)
	}
}

func TestAgentHook_Execute_PostTaskWithAgentResult(t *testing.T) {
	registry := agent.NewRegistry()
	mockAg := &mockAgent{
		name:      "test-agent",
		available: true,
		runResult: agent.Result{ExitCode: 0, Status: agent.TaskStatusDone},
	}
	registry.Register(mockAg)

	def := config.HookDefinition{
		Type:    config.HookTypeAgent,
		Command: "Verify task ${TASK_ID} with status ${AGENT_STATUS} (exit: ${AGENT_EXIT_CODE})",
	}
	cfg := AgentHookConfig{Registry: registry}

	hook := NewAgentHook("post-hook", HookPhasePost, def, cfg)

	hookCtx := &HookContext{
		Task:      task.NewTask("TASK-001", "Test", "Desc"),
		Iteration: 1,
		Result: &agent.Result{
			Output:   "Task completed",
			ExitCode: 0,
			Status:   agent.TaskStatusDone,
		},
		ProjectDir: "/tmp",
	}

	_, err := hook.Execute(context.Background(), hookCtx)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	expectedPrompt := "Verify task TASK-001 with status DONE (exit: 0)"
	if mockAg.lastPrompt != expectedPrompt {
		t.Errorf("lastPrompt = %v, want %v", mockAg.lastPrompt, expectedPrompt)
	}
}