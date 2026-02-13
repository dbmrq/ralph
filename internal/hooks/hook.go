// Package hooks provides pre/post task hook functionality for ralph.
// Hooks can execute shell commands or agent calls before or after each task.
package hooks

import (
	"context"
	"fmt"

	"github.com/wexinc/ralph/internal/agent"
	"github.com/wexinc/ralph/internal/config"
	"github.com/wexinc/ralph/internal/task"
)

// HookPhase indicates when a hook runs relative to task execution.
type HookPhase string

const (
	// HookPhasePre indicates the hook runs before the task.
	HookPhasePre HookPhase = "pre"
	// HookPhasePost indicates the hook runs after the task.
	HookPhasePost HookPhase = "post"
)

// String returns the string representation of the hook phase.
func (p HookPhase) String() string {
	return string(p)
}

// IsValid returns true if the hook phase is valid.
func (p HookPhase) IsValid() bool {
	return p == HookPhasePre || p == HookPhasePost
}

// HookContext provides context information for hook execution.
// This includes the current task state and agent result (for post-task hooks).
type HookContext struct {
	// Task is the current task being processed.
	Task *task.Task
	// Result is the agent result (only populated for post-task hooks).
	Result *agent.Result
	// Iteration is the current iteration number for the task.
	Iteration int
	// ProjectDir is the project root directory.
	ProjectDir string
}

// HookResult represents the outcome of a hook execution.
type HookResult struct {
	// Success indicates whether the hook completed successfully.
	Success bool
	// Output is the captured output from the hook.
	Output string
	// Error contains any error message if the hook failed.
	Error string
	// ExitCode is the exit code for shell hooks (0 = success).
	ExitCode int
	// FailureMode is the configured failure handling mode.
	FailureMode config.FailureMode
}

// IsSuccess returns true if the hook executed successfully.
func (r HookResult) IsSuccess() bool {
	return r.Success && r.ExitCode == 0
}

// ShouldAbort returns true if the hook failure should abort the loop.
func (r HookResult) ShouldAbort() bool {
	return !r.IsSuccess() && r.FailureMode == config.FailureModeAbortLoop
}

// ShouldSkipTask returns true if the hook failure should skip the current task.
func (r HookResult) ShouldSkipTask() bool {
	return !r.IsSuccess() && r.FailureMode == config.FailureModeSkipTask
}

// ShouldAskAgent returns true if the agent should decide how to handle the failure.
func (r HookResult) ShouldAskAgent() bool {
	return !r.IsSuccess() && r.FailureMode == config.FailureModeAskAgent
}

// ShouldWarnAndContinue returns true if the hook failure should log a warning and continue.
func (r HookResult) ShouldWarnAndContinue() bool {
	return !r.IsSuccess() && r.FailureMode == config.FailureModeWarnContinue
}

// Hook defines the interface that all hook implementations must satisfy.
type Hook interface {
	// Name returns a descriptive name for this hook (for logging/debugging).
	Name() string

	// Phase returns whether this is a pre-task or post-task hook.
	Phase() HookPhase

	// Type returns the hook type (shell or agent).
	Type() config.HookType

	// Definition returns the underlying hook definition from config.
	Definition() config.HookDefinition

	// Execute runs the hook with the given context.
	// Returns a HookResult with the outcome of the execution.
	Execute(ctx context.Context, hookCtx *HookContext) (*HookResult, error)
}

// BaseHook provides common functionality for hook implementations.
// Embed this in concrete hook types (ShellHook, AgentHook).
type BaseHook struct {
	// name is a descriptive name for this hook instance.
	name string
	// phase indicates when this hook runs (pre/post task).
	phase HookPhase
	// definition is the hook configuration from config.yaml.
	definition config.HookDefinition
}

// NewBaseHook creates a new BaseHook with the given parameters.
func NewBaseHook(name string, phase HookPhase, def config.HookDefinition) BaseHook {
	return BaseHook{
		name:       name,
		phase:      phase,
		definition: def,
	}
}

// Name returns the hook name.
func (h *BaseHook) Name() string {
	return h.name
}

// Phase returns the hook phase (pre/post).
func (h *BaseHook) Phase() HookPhase {
	return h.phase
}

// Type returns the hook type from the definition.
func (h *BaseHook) Type() config.HookType {
	return h.definition.Type
}

// Definition returns the hook definition.
func (h *BaseHook) Definition() config.HookDefinition {
	return h.definition
}

// GetFailureMode returns the failure mode, defaulting to WarnContinue if not set.
func (h *BaseHook) GetFailureMode() config.FailureMode {
	if h.definition.OnFailure == "" {
		return config.FailureModeWarnContinue
	}
	return h.definition.OnFailure
}

// CreateHookResult creates a HookResult with the hook's failure mode.
func (h *BaseHook) CreateHookResult(success bool, output, errMsg string, exitCode int) *HookResult {
	return &HookResult{
		Success:     success,
		Output:      output,
		Error:       errMsg,
		ExitCode:    exitCode,
		FailureMode: h.GetFailureMode(),
	}
}

// CreateHooksFromConfig creates Hook instances from the configuration.
// This is a factory function that creates either ShellHook or AgentHook based on type.
// Note: The actual ShellHook and AgentHook types will be implemented in HOOK-002 and HOOK-003.
func CreateHooksFromConfig(cfg *config.HooksConfig) (preHooks, postHooks []Hook, err error) {
	preHooks = make([]Hook, 0, len(cfg.PreTask))
	postHooks = make([]Hook, 0, len(cfg.PostTask))

	for i, def := range cfg.PreTask {
		name := fmt.Sprintf("pre_task[%d]", i)
		hook, err := createHookFromDefinition(name, HookPhasePre, def)
		if err != nil {
			return nil, nil, fmt.Errorf("creating pre-task hook %d: %w", i, err)
		}
		preHooks = append(preHooks, hook)
	}

	for i, def := range cfg.PostTask {
		name := fmt.Sprintf("post_task[%d]", i)
		hook, err := createHookFromDefinition(name, HookPhasePost, def)
		if err != nil {
			return nil, nil, fmt.Errorf("creating post-task hook %d: %w", i, err)
		}
		postHooks = append(postHooks, hook)
	}

	return preHooks, postHooks, nil
}

// createHookFromDefinition creates a single hook from a definition.
// Returns a placeholder hook for now - actual implementations come in HOOK-002/003.
func createHookFromDefinition(name string, phase HookPhase, def config.HookDefinition) (Hook, error) {
	switch def.Type {
	case config.HookTypeShell:
		return &placeholderHook{BaseHook: NewBaseHook(name, phase, def)}, nil
	case config.HookTypeAgent:
		return &placeholderHook{BaseHook: NewBaseHook(name, phase, def)}, nil
	case "":
		// Default to shell if type not specified
		def.Type = config.HookTypeShell
		return &placeholderHook{BaseHook: NewBaseHook(name, phase, def)}, nil
	default:
		return nil, fmt.Errorf("unknown hook type: %s", def.Type)
	}
}

// placeholderHook is a temporary implementation until HOOK-002 and HOOK-003.
type placeholderHook struct {
	BaseHook
}

// Execute returns a not-implemented error for placeholder hooks.
func (h *placeholderHook) Execute(ctx context.Context, hookCtx *HookContext) (*HookResult, error) {
	return nil, fmt.Errorf("hook type %s not yet implemented", h.Type())
}
