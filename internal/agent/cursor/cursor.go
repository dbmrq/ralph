// Package cursor provides the Cursor agent plugin for ralph.
// Cursor uses the `agent` CLI command for AI coding assistance.
package cursor

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"

	"github.com/dbmrq/ralph/internal/agent"
)

// DefaultModel is the default model for Cursor.
const DefaultModel = "claude-sonnet-4-20250514"

// Agent implements the agent.Agent interface for Cursor.
type Agent struct {
	// lastSessionID stores the session ID from the most recent run.
	lastSessionID string
	// defaultModel stores the default model to use.
	defaultModel string
}

// New creates a new Cursor agent.
func New() *Agent {
	return &Agent{
		defaultModel: DefaultModel,
	}
}

// Name returns the unique identifier for this agent.
func (a *Agent) Name() string {
	return "cursor"
}

// Description returns a human-readable description of the agent.
func (a *Agent) Description() string {
	return "Cursor AI coding assistant using the agent CLI"
}

// IsAvailable checks if the Cursor agent CLI is installed.
func (a *Agent) IsAvailable() bool {
	_, err := exec.LookPath("agent")
	return err == nil
}

// CheckAuth verifies that authentication is configured for Cursor.
// Cursor's agent CLI typically handles auth automatically through the Cursor app.
func (a *Agent) CheckAuth() error {
	if !a.IsAvailable() {
		return fmt.Errorf("cursor agent CLI not found: ensure Cursor is installed and the 'agent' command is in your PATH")
	}
	// Cursor's agent CLI doesn't have a separate auth check command,
	// so we just verify the command exists.
	return nil
}

// ListModels returns all available models for Cursor.
// It parses the output of `agent --list-models`.
func (a *Agent) ListModels() ([]agent.Model, error) {
	cmd := exec.Command("agent", "--list-models")
	output, err := cmd.Output()
	if err != nil {
		// If list-models fails, return default models
		return a.getDefaultModels(), nil
	}
	return parseModelsOutput(string(output), a.defaultModel), nil
}

// GetDefaultModel returns the default model for Cursor.
func (a *Agent) GetDefaultModel() agent.Model {
	return agent.Model{
		ID:        a.defaultModel,
		Name:      a.defaultModel,
		IsDefault: true,
	}
}

// Run executes a prompt and returns the result.
func (a *Agent) Run(ctx context.Context, prompt string, opts agent.RunOptions) (agent.Result, error) {
	return a.execute(ctx, prompt, opts, false)
}

// Continue resumes a previous session with a new prompt.
// Note: Cursor's agent CLI may have limited session support.
func (a *Agent) Continue(ctx context.Context, sessionID string, prompt string, opts agent.RunOptions) (agent.Result, error) {
	// Store the session ID for potential use
	opts.SessionID = sessionID
	return a.execute(ctx, prompt, opts, true)
}

// GetSessionID returns the session ID from the most recent run.
func (a *Agent) GetSessionID() string {
	return a.lastSessionID
}

// execute runs the agent command with the given parameters.
func (a *Agent) execute(ctx context.Context, prompt string, opts agent.RunOptions, isContinue bool) (agent.Result, error) {
	args := []string{"--print"}

	if opts.Force {
		args = append(args, "--force")
	}

	model := opts.Model
	if model == "" {
		model = a.defaultModel
	}
	if model != "" {
		args = append(args, "--model", model)
	}

	// Add the prompt as the last argument
	args = append(args, prompt)

	cmd := exec.CommandContext(ctx, "agent", args...)

	if opts.WorkDir != "" {
		cmd.Dir = opts.WorkDir
	}

	// Set up output capture
	var stdout, stderr bytes.Buffer
	if opts.LogWriter != nil {
		cmd.Stdout = io.MultiWriter(&stdout, opts.LogWriter)
	} else {
		cmd.Stdout = &stdout
	}
	cmd.Stderr = &stderr

	// Copy environment
	cmd.Env = os.Environ()

	startTime := time.Now()
	err := cmd.Run()
	duration := time.Since(startTime)

	exitCode := 0
	if cmd.ProcessState != nil {
		exitCode = cmd.ProcessState.ExitCode()
	}

	output := stdout.String()
	status := parseTaskStatus(output)
	sessionID := extractSessionID(output)
	a.lastSessionID = sessionID

	result := agent.Result{
		Output:    output,
		ExitCode:  exitCode,
		Duration:  duration,
		Status:    status,
		SessionID: sessionID,
	}

	if err != nil && exitCode != 0 {
		result.Error = stderr.String()
		if result.Error == "" {
			result.Error = err.Error()
		}
	}

	return result, nil
}

// getDefaultModels returns the default list of models when list-models fails.
func (a *Agent) getDefaultModels() []agent.Model {
	return []agent.Model{
		{ID: "claude-sonnet-4-20250514", Name: "Claude Sonnet 4", IsDefault: true},
		{ID: "claude-opus-4-20250514", Name: "Claude Opus 4"},
		{ID: "gpt-4o", Name: "GPT-4o"},
		{ID: "gpt-4.1", Name: "GPT-4.1"},
	}
}

// parseModelsOutput parses the output of `agent --list-models`.
// Expected format: one model name per line.
func parseModelsOutput(output string, defaultModel string) []agent.Model {
	var models []agent.Model
	lines := strings.Split(strings.TrimSpace(output), "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// Skip lines that look like headers or separators
		if strings.HasPrefix(line, "-") || strings.HasPrefix(line, "=") {
			continue
		}

		model := agent.Model{
			ID:        line,
			Name:      line,
			IsDefault: line == defaultModel,
		}
		models = append(models, model)
	}

	// If no models found, return empty slice
	if len(models) == 0 {
		return nil
	}

	return models
}

// parseTaskStatus extracts the task status from agent output.
// It looks for status markers at the end of the output: NEXT, DONE, ERROR, FIXED.
func parseTaskStatus(output string) agent.TaskStatus {
	// Look for status markers in the output (typically at the end)
	output = strings.TrimSpace(output)

	// Check for status markers, looking from the end of output
	lines := strings.Split(output, "\n")
	for i := len(lines) - 1; i >= 0 && i >= len(lines)-10; i-- {
		line := strings.TrimSpace(lines[i])

		// Check for exact status markers
		if strings.HasPrefix(line, "DONE") || line == "DONE" {
			return agent.TaskStatusDone
		}
		if strings.HasPrefix(line, "NEXT") || line == "NEXT" {
			return agent.TaskStatusNext
		}
		if strings.HasPrefix(line, "ERROR") || strings.HasPrefix(line, "ERROR:") {
			return agent.TaskStatusError
		}
		if strings.HasPrefix(line, "FIXED") || line == "FIXED" {
			return agent.TaskStatusFixed
		}
	}

	return agent.TaskStatusUnknown
}

// sessionIDPattern matches session IDs in agent output.
var sessionIDPattern = regexp.MustCompile(`session[_-]?id[:\s]+([a-zA-Z0-9_-]+)`)

// extractSessionID extracts a session ID from agent output.
// Returns empty string if no session ID is found.
func extractSessionID(output string) string {
	matches := sessionIDPattern.FindStringSubmatch(output)
	if len(matches) >= 2 {
		return matches[1]
	}
	return ""
}
