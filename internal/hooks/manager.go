// Package hooks provides pre/post task hook functionality for ralph.
package hooks

import (
	"context"
	"fmt"

	"github.com/dbmrq/ralph/internal/agent"
	"github.com/dbmrq/ralph/internal/config"
	"github.com/dbmrq/ralph/internal/task"
)

// ManagerResult represents the aggregate outcome of executing a phase of hooks.
type ManagerResult struct {
	// AllSuccess is true if all hooks succeeded.
	AllSuccess bool
	// Results contains the individual result for each hook.
	Results []*HookResult
	// Action is the recommended action based on hook results.
	Action ManagerAction
	// FailedHook is the hook that caused a non-continue action (if any).
	FailedHook Hook
	// FailedResult is the result of the failed hook (if any).
	FailedResult *HookResult
}

// ManagerAction defines the action the manager recommends after hook execution.
type ManagerAction string

const (
	// ManagerActionContinue indicates all hooks passed or failed with warn_continue.
	ManagerActionContinue ManagerAction = "continue"
	// ManagerActionSkipTask indicates a hook failed with skip_task mode.
	ManagerActionSkipTask ManagerAction = "skip_task"
	// ManagerActionAbortLoop indicates a hook failed with abort_loop mode.
	ManagerActionAbortLoop ManagerAction = "abort_loop"
	// ManagerActionAskAgent indicates a hook failed with ask_agent mode.
	ManagerActionAskAgent ManagerAction = "ask_agent"
)

// Manager orchestrates hook execution for the ralph loop.
// It manages pre-task and post-task hooks, executing them in order
// and handling failures according to each hook's configured failure mode.
type Manager struct {
	preHooks  []Hook
	postHooks []Hook
	// Logger is called for each hook execution (optional).
	Logger func(phase HookPhase, hook Hook, result *HookResult)
}

// NewManager creates a new hook manager with the given hooks.
func NewManager(preHooks, postHooks []Hook) *Manager {
	return &Manager{
		preHooks:  preHooks,
		postHooks: postHooks,
	}
}

// NewManagerFromConfig creates a Manager from configuration.
// This is a convenience constructor that creates hooks from config.
func NewManagerFromConfig(cfg *config.HooksConfig) (*Manager, error) {
	preHooks, postHooks, err := CreateHooksFromConfig(cfg)
	if err != nil {
		return nil, fmt.Errorf("creating hooks from config: %w", err)
	}
	return NewManager(preHooks, postHooks), nil
}

// NewManagerFromConfigWithAgents creates a Manager from configuration with agent support.
func NewManagerFromConfigWithAgents(cfg *config.HooksConfig, agentCfg AgentHookConfig) (*Manager, error) {
	preHooks, postHooks, err := CreateHooksFromConfigWithAgents(cfg, agentCfg)
	if err != nil {
		return nil, fmt.Errorf("creating hooks from config: %w", err)
	}
	return NewManager(preHooks, postHooks), nil
}

// PreTaskHooks returns the pre-task hooks.
func (m *Manager) PreTaskHooks() []Hook {
	return m.preHooks
}

// PostTaskHooks returns the post-task hooks.
func (m *Manager) PostTaskHooks() []Hook {
	return m.postHooks
}

// HasPreTaskHooks returns true if there are pre-task hooks configured.
func (m *Manager) HasPreTaskHooks() bool {
	return len(m.preHooks) > 0
}

// HasPostTaskHooks returns true if there are post-task hooks configured.
func (m *Manager) HasPostTaskHooks() bool {
	return len(m.postHooks) > 0
}

// ExecutePreTaskHooks runs all pre-task hooks in order.
// It stops execution if a hook fails with skip_task or abort_loop mode.
// Returns a ManagerResult with the aggregate outcome.
func (m *Manager) ExecutePreTaskHooks(ctx context.Context, hookCtx *HookContext) *ManagerResult {
	return m.executeHooks(ctx, m.preHooks, hookCtx, HookPhasePre)
}

// ExecutePostTaskHooks runs all post-task hooks in order.
// It stops execution if a hook fails with skip_task or abort_loop mode.
// Returns a ManagerResult with the aggregate outcome.
func (m *Manager) ExecutePostTaskHooks(ctx context.Context, hookCtx *HookContext) *ManagerResult {
	return m.executeHooks(ctx, m.postHooks, hookCtx, HookPhasePost)
}

// BuildHookContextForPreTask creates a HookContext for pre-task hooks.
// The task parameter is the current task being executed.
// Result will be nil since the task hasn't run yet.
func BuildHookContextForPreTask(t *task.Task, iteration int, projectDir string) *HookContext {
	return &HookContext{
		Task:       t,
		Iteration:  iteration,
		ProjectDir: projectDir,
	}
}

// BuildHookContextForPostTask creates a HookContext for post-task hooks.
// The task parameter is the current task, result is the agent's execution result.
func BuildHookContextForPostTask(t *task.Task, result *agent.Result, iteration int, projectDir string) *HookContext {
	return &HookContext{
		Task:       t,
		Result:     result,
		Iteration:  iteration,
		ProjectDir: projectDir,
	}
}

// executeHooks runs a list of hooks in order, handling failures.
func (m *Manager) executeHooks(ctx context.Context, hooks []Hook, hookCtx *HookContext, phase HookPhase) *ManagerResult {
	result := &ManagerResult{
		AllSuccess: true,
		Results:    make([]*HookResult, 0, len(hooks)),
		Action:     ManagerActionContinue,
	}

	for _, hook := range hooks {
		// Check context cancellation
		if ctx.Err() != nil {
			result.AllSuccess = false
			result.Action = ManagerActionAbortLoop
			break
		}

		hookResult, err := hook.Execute(ctx, hookCtx)
		if err != nil {
			// Execution error (not hook failure) - treat as abort
			hookResult = &HookResult{
				Success:     false,
				Error:       fmt.Sprintf("execution error: %v", err),
				ExitCode:    1,
				FailureMode: config.FailureModeAbortLoop,
			}
		}

		result.Results = append(result.Results, hookResult)

		// Log the result if logger is configured
		if m.Logger != nil {
			m.Logger(phase, hook, hookResult)
		}

		// Handle failure modes
		if !hookResult.IsSuccess() {
			result.AllSuccess = false

			switch {
			case hookResult.ShouldAbort():
				result.Action = ManagerActionAbortLoop
				result.FailedHook = hook
				result.FailedResult = hookResult
				return result

			case hookResult.ShouldSkipTask():
				result.Action = ManagerActionSkipTask
				result.FailedHook = hook
				result.FailedResult = hookResult
				return result

			case hookResult.ShouldAskAgent():
				result.Action = ManagerActionAskAgent
				result.FailedHook = hook
				result.FailedResult = hookResult
				return result

			case hookResult.ShouldWarnAndContinue():
				// Continue with next hook
				continue
			}
		}
	}

	return result
}

// GetFailedHookInfo returns a formatted string describing the failed hook for agent prompts.
// This is useful for the ask_agent failure mode.
func (m *Manager) GetFailedHookInfo(result *ManagerResult) string {
	if result.FailedHook == nil || result.FailedResult == nil {
		return ""
	}

	return fmt.Sprintf(
		"Hook '%s' (type: %s, phase: %s) failed with exit code %d.\nError: %s\nOutput: %s",
		result.FailedHook.Name(),
		result.FailedHook.Type(),
		result.FailedHook.Phase(),
		result.FailedResult.ExitCode,
		result.FailedResult.Error,
		result.FailedResult.Output,
	)
}
