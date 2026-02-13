// Package hooks provides pre/post task hook functionality for ralph.
package hooks

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/wexinc/ralph/internal/config"
)

// ShellHook executes shell commands as hooks.
// It supports environment variable injection for task context and
// configurable failure handling modes.
type ShellHook struct {
	BaseHook
}

// NewShellHook creates a new shell hook with the given parameters.
func NewShellHook(name string, phase HookPhase, def config.HookDefinition) *ShellHook {
	return &ShellHook{
		BaseHook: NewBaseHook(name, phase, def),
	}
}

// Execute runs the shell command with the hook context.
// It sets environment variables based on the current task state and
// captures stdout/stderr. The result includes the exit code and output.
func (h *ShellHook) Execute(ctx context.Context, hookCtx *HookContext) (*HookResult, error) {
	if hookCtx == nil {
		return nil, fmt.Errorf("hook context is required")
	}

	command := h.definition.Command
	if command == "" {
		return nil, fmt.Errorf("shell hook command is empty")
	}

	// Expand environment variables in the command
	command = h.expandEnvVars(command, hookCtx)

	// Create the command
	cmd := exec.CommandContext(ctx, "sh", "-c", command)

	// Set up environment variables
	cmd.Env = h.buildEnv(hookCtx)

	// Capture output
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Run the command
	err := cmd.Run()

	// Combine stdout and stderr for output
	output := strings.TrimSpace(stdout.String())
	if stderr.Len() > 0 {
		stderrStr := strings.TrimSpace(stderr.String())
		if output != "" {
			output = output + "\n" + stderrStr
		} else {
			output = stderrStr
		}
	}

	// Determine success and exit code
	exitCode := 0
	var errMsg string
	success := true

	if err != nil {
		success = false
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			exitCode = 1
		}
		errMsg = err.Error()
	}

	return h.CreateHookResult(success, output, errMsg, exitCode), nil
}

// buildEnv creates the environment variables for the shell command.
// It starts with the current process environment and adds hook-specific variables.
func (h *ShellHook) buildEnv(hookCtx *HookContext) []string {
	// Start with current environment
	env := os.Environ()

	// Add hook-specific environment variables
	hookEnv := make(map[string]string)

	if hookCtx.Task != nil {
		hookEnv["TASK_ID"] = hookCtx.Task.ID
		hookEnv["TASK_NAME"] = hookCtx.Task.Name
		hookEnv["TASK_DESCRIPTION"] = hookCtx.Task.Description
		hookEnv["TASK_STATUS"] = string(hookCtx.Task.Status)
	}

	hookEnv["ITERATION"] = strconv.Itoa(hookCtx.Iteration)
	hookEnv["PROJECT_DIR"] = hookCtx.ProjectDir

	// Add result-specific variables for post-task hooks
	if hookCtx.Result != nil {
		hookEnv["AGENT_OUTPUT"] = hookCtx.Result.Output
		hookEnv["AGENT_EXIT_CODE"] = strconv.Itoa(hookCtx.Result.ExitCode)
		hookEnv["AGENT_STATUS"] = string(hookCtx.Result.Status)
	}

	// Append hook env vars to environment
	for key, value := range hookEnv {
		env = append(env, fmt.Sprintf("%s=%s", key, value))
	}

	return env
}

// expandEnvVars expands ${VAR} patterns in the command string using hook context.
// This allows commands like "echo 'Starting task: ${TASK_ID}'" to work.
func (h *ShellHook) expandEnvVars(command string, hookCtx *HookContext) string {
	// Build a map of variables for expansion
	vars := make(map[string]string)

	if hookCtx.Task != nil {
		vars["TASK_ID"] = hookCtx.Task.ID
		vars["TASK_NAME"] = hookCtx.Task.Name
		vars["TASK_DESCRIPTION"] = hookCtx.Task.Description
		vars["TASK_STATUS"] = string(hookCtx.Task.Status)
	}

	vars["ITERATION"] = strconv.Itoa(hookCtx.Iteration)
	vars["PROJECT_DIR"] = hookCtx.ProjectDir

	if hookCtx.Result != nil {
		vars["AGENT_OUTPUT"] = hookCtx.Result.Output
		vars["AGENT_EXIT_CODE"] = strconv.Itoa(hookCtx.Result.ExitCode)
		vars["AGENT_STATUS"] = string(hookCtx.Result.Status)
	}

	// Expand ${VAR} patterns
	result := command
	for key, value := range vars {
		result = strings.ReplaceAll(result, "${"+key+"}", value)
	}

	return result
}

