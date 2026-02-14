// Package custom provides a configurable custom agent plugin for ralph.
// Custom agents are defined via configuration rather than code.
package custom

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
	"github.com/dbmrq/ralph/internal/config"
)

// Agent implements the agent.Agent interface for custom user-defined agents.
type Agent struct {
	config        config.CustomAgentConfig
	lastSessionID string
}

// New creates a new custom agent from configuration.
func New(cfg config.CustomAgentConfig) *Agent {
	return &Agent{
		config: cfg,
	}
}

// Name returns the unique identifier for this agent.
func (a *Agent) Name() string {
	return a.config.Name
}

// Description returns a human-readable description of the agent.
func (a *Agent) Description() string {
	if a.config.Description != "" {
		return a.config.Description
	}
	return fmt.Sprintf("Custom agent using %s", a.config.Command)
}

// IsAvailable checks if this agent is available on the system.
func (a *Agent) IsAvailable() bool {
	switch a.config.DetectionMethod {
	case config.DetectionMethodCommand, "":
		// Default: check if command exists in PATH
		cmdName := a.config.DetectionValue
		if cmdName == "" {
			// Use the base command name
			cmdName = strings.Fields(a.config.Command)[0]
		}
		_, err := exec.LookPath(cmdName)
		return err == nil
	case config.DetectionMethodPath:
		// Check if a specific path exists
		if a.config.DetectionValue == "" {
			return false
		}
		_, err := os.Stat(a.config.DetectionValue)
		return err == nil
	case config.DetectionMethodEnv:
		// Check if environment variable is set
		if a.config.DetectionValue == "" {
			return false
		}
		_, exists := os.LookupEnv(a.config.DetectionValue)
		return exists
	case config.DetectionMethodAlways:
		return true
	default:
		return false
	}
}

// CheckAuth verifies that authentication is configured for this agent.
// For custom agents, we assume auth is handled externally.
func (a *Agent) CheckAuth() error {
	if !a.IsAvailable() {
		return fmt.Errorf("custom agent %q is not available", a.config.Name)
	}
	return nil
}

// ListModels returns all available models for this agent.
func (a *Agent) ListModels() ([]agent.Model, error) {
	if a.config.ModelListCommand == "" {
		// No model list command, return default model only
		return []agent.Model{a.GetDefaultModel()}, nil
	}

	// Execute the model list command
	cmd := exec.Command("sh", "-c", a.config.ModelListCommand)
	output, err := cmd.Output()
	if err != nil {
		// Fall back to default model
		return []agent.Model{a.GetDefaultModel()}, nil
	}

	return parseModelsOutput(string(output), a.config.DefaultModel), nil
}

// GetDefaultModel returns the default model for this agent.
func (a *Agent) GetDefaultModel() agent.Model {
	modelID := a.config.DefaultModel
	if modelID == "" {
		modelID = "default"
	}
	return agent.Model{
		ID:        modelID,
		Name:      modelID,
		IsDefault: true,
	}
}

// Run executes a prompt and returns the result.
func (a *Agent) Run(ctx context.Context, prompt string, opts agent.RunOptions) (agent.Result, error) {
	return a.execute(ctx, prompt, opts)
}

// Continue resumes a previous session with a new prompt.
func (a *Agent) Continue(ctx context.Context, sessionID string, prompt string, opts agent.RunOptions) (agent.Result, error) {
	opts.SessionID = sessionID
	return a.execute(ctx, prompt, opts)
}

// GetSessionID returns the session ID from the most recent run.
func (a *Agent) GetSessionID() string {
	return a.lastSessionID
}

// execute runs the agent command with the given parameters.
func (a *Agent) execute(ctx context.Context, prompt string, opts agent.RunOptions) (agent.Result, error) {
	// Build the command
	cmdParts := strings.Fields(a.config.Command)
	if len(cmdParts) == 0 {
		return agent.Result{}, fmt.Errorf("custom agent %q has empty command", a.config.Name)
	}

	args := append(cmdParts[1:], a.config.Args...)
	args = append(args, prompt)

	cmd := exec.CommandContext(ctx, cmdParts[0], args...)

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

// parseModelsOutput parses the output of a model list command.
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

	if len(models) == 0 {
		return nil
	}

	return models
}

// parseTaskStatus extracts the task status from agent output.
func parseTaskStatus(output string) agent.TaskStatus {
	output = strings.TrimSpace(output)
	lines := strings.Split(output, "\n")

	for i := len(lines) - 1; i >= 0 && i >= len(lines)-10; i-- {
		line := strings.TrimSpace(lines[i])

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
func extractSessionID(output string) string {
	matches := sessionIDPattern.FindStringSubmatch(output)
	if len(matches) >= 2 {
		return matches[1]
	}
	return ""
}
