// Package hooks provides pre/post task hook functionality for ralph.
package hooks

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/dbmrq/ralph/internal/agent"
	"github.com/dbmrq/ralph/internal/config"
)

// AgentHookConfig provides configuration for agent hook execution.
type AgentHookConfig struct {
	// Registry is the agent registry to look up agents.
	Registry *agent.Registry
	// DefaultAgent is the default agent to use if none specified in hook.
	DefaultAgent string
	// DefaultModel is the default model to use if none specified in hook.
	DefaultModel string
	// WorkDir is the working directory for agent execution.
	WorkDir string
	// Timeout is the maximum time for hook execution.
	Timeout time.Duration
}

// AgentHook executes agent prompts as hooks.
// It runs an AI agent with a custom prompt and captures the result.
type AgentHook struct {
	BaseHook
	config AgentHookConfig
}

// NewAgentHook creates a new agent hook with the given parameters.
func NewAgentHook(name string, phase HookPhase, def config.HookDefinition, cfg AgentHookConfig) *AgentHook {
	return &AgentHook{
		BaseHook: NewBaseHook(name, phase, def),
		config:   cfg,
	}
}

// Execute runs the agent with the configured prompt.
// It builds the prompt using task context and executes via the agent interface.
func (h *AgentHook) Execute(ctx context.Context, hookCtx *HookContext) (*HookResult, error) {
	if hookCtx == nil {
		return nil, fmt.Errorf("hook context is required")
	}

	prompt := h.definition.Command
	if prompt == "" {
		return nil, fmt.Errorf("agent hook prompt (command) is empty")
	}

	// Expand template variables in the prompt
	prompt = h.expandPromptVariables(prompt, hookCtx)

	// Get the agent to use
	selectedAgent, err := h.selectAgent()
	if err != nil {
		return h.CreateHookResult(false, "", fmt.Sprintf("failed to select agent: %v", err), 1), nil
	}

	// Check agent availability
	if !selectedAgent.IsAvailable() {
		return h.CreateHookResult(false, "", fmt.Sprintf("agent %q is not available", selectedAgent.Name()), 1), nil
	}

	// Build run options
	opts := agent.RunOptions{
		WorkDir: h.config.WorkDir,
		Force:   true,
		Timeout: h.config.Timeout,
	}

	// Use model from hook definition, config, or agent default
	if h.definition.Model != "" {
		opts.Model = h.definition.Model
	} else if h.config.DefaultModel != "" {
		opts.Model = h.config.DefaultModel
	}

	// Execute the agent
	result, err := selectedAgent.Run(ctx, prompt, opts)
	if err != nil {
		return h.CreateHookResult(false, "", fmt.Sprintf("agent execution failed: %v", err), 1), nil
	}

	// Create hook result from agent result
	success := result.ExitCode == 0 && result.Status.IsSuccess()
	var errMsg string
	if !success {
		errMsg = result.Error
		if errMsg == "" && result.ExitCode != 0 {
			errMsg = fmt.Sprintf("agent exited with code %d", result.ExitCode)
		}
	}

	return h.CreateHookResult(success, result.Output, errMsg, result.ExitCode), nil
}

// selectAgent selects the agent to use for this hook.
// Priority: hook definition > config default > registry default
func (h *AgentHook) selectAgent() (agent.Agent, error) {
	if h.config.Registry == nil {
		return nil, fmt.Errorf("agent registry is not configured")
	}

	// Try hook-specific agent first
	agentName := h.definition.Agent
	if agentName == "" {
		agentName = h.config.DefaultAgent
	}

	if agentName != "" {
		return h.config.Registry.SelectAgent(agentName)
	}

	// Fall back to any available agent
	return h.config.Registry.GetOrDefault("")
}

// expandPromptVariables expands ${VAR} patterns in the prompt using hook context.
// This allows prompts like "Review the changes for ${TASK_ID}: ${TASK_NAME}".
func (h *AgentHook) expandPromptVariables(prompt string, hookCtx *HookContext) string {
	vars := make(map[string]string)

	if hookCtx.Task != nil {
		vars["TASK_ID"] = hookCtx.Task.ID
		vars["TASK_NAME"] = hookCtx.Task.Name
		vars["TASK_DESCRIPTION"] = hookCtx.Task.Description
		vars["TASK_STATUS"] = string(hookCtx.Task.Status)
	}

	vars["ITERATION"] = fmt.Sprintf("%d", hookCtx.Iteration)
	vars["PROJECT_DIR"] = hookCtx.ProjectDir

	if hookCtx.Result != nil {
		vars["AGENT_OUTPUT"] = hookCtx.Result.Output
		vars["AGENT_EXIT_CODE"] = fmt.Sprintf("%d", hookCtx.Result.ExitCode)
		vars["AGENT_STATUS"] = string(hookCtx.Result.Status)
	}

	result := prompt
	for key, value := range vars {
		result = strings.ReplaceAll(result, "${"+key+"}", value)
	}

	return result
}
