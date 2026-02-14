// Package errors provides comprehensive error types for ralph.
// This file contains agent-specific errors and authentication helpers.
package errors

import (
	"fmt"
	"strings"
)

// Agent-related error constructors.

// AgentNotAvailable creates an error for when an agent is not installed.
func AgentNotAvailable(agentName string) *RalphError {
	return &RalphError{
		Kind:    ErrAgent,
		Message: fmt.Sprintf("agent %q is not available on this system", agentName),
		Suggestion: fmt.Sprintf(`Install the %s CLI tool:
  • Cursor: Install the Cursor IDE from https://cursor.com
  • Auggie: Run 'npm install -g @anthropic/auggie' or 'brew install auggie'
  
Alternatively, configure a custom agent in .ralph/config.yaml`, agentName),
	}
}

// AgentNotFound creates an error for when a requested agent doesn't exist.
func AgentNotFound(agentName string, availableAgents []string) *RalphError {
	available := "none"
	if len(availableAgents) > 0 {
		available = strings.Join(availableAgents, ", ")
	}
	return &RalphError{
		Kind:    ErrAgent,
		Message: fmt.Sprintf("agent %q not found", agentName),
		Details: map[string]string{
			"requested": agentName,
			"available": available,
		},
		Suggestion: fmt.Sprintf("Use one of the available agents: %s\n"+
			"Or configure a custom agent in .ralph/config.yaml", available),
	}
}

// NoAgentsAvailable creates an error when no agents can be found.
func NoAgentsAvailable() *RalphError {
	return &RalphError{
		Kind:    ErrAgent,
		Message: "no AI agents available",
		Suggestion: `Install an AI coding agent:
  
  For Cursor users:
    1. Install Cursor IDE from https://cursor.com
    2. Ensure the 'agent' command is in your PATH
  
  For Auggie users:
    1. Run: npm install -g @anthropic/auggie
    2. Run: auggie login
    3. Verify: auggie --version
  
  For custom agents:
    1. Add custom agent config to .ralph/config.yaml
    2. Example:
       agent:
         custom:
           - name: my-agent
             command: my-agent-cli
             detection_method: command`,
		DocLink: "https://github.com/dbmrq/ralph#agents",
	}
}

// MultipleAgentsNeedSelection creates an error when user needs to select.
func MultipleAgentsNeedSelection(agents []string) *RalphError {
	return &RalphError{
		Kind:    ErrAgent,
		Message: "multiple agents available, selection required",
		Details: map[string]string{
			"available": strings.Join(agents, ", "),
		},
		Suggestion: fmt.Sprintf(`Specify which agent to use:
  
  In config (.ralph/config.yaml):
    agent:
      default: %s
  
  Or via environment:
    export RALPH_AGENT=%s
  
  Or run interactively to be prompted for selection.`, agents[0], agents[0]),
	}
}

// Authentication-related error constructors.

// AuthNotConfigured creates an error for missing authentication.
func AuthNotConfigured(agentName string) *RalphError {
	suggestion := ""
	switch strings.ToLower(agentName) {
	case "auggie":
		suggestion = `Authenticate with Auggie:
  1. Run: auggie login
  2. Follow the browser prompts
  3. Verify: auggie tokens print
  
Alternatively, set the AUGMENT_SESSION_AUTH environment variable.`
	case "cursor":
		suggestion = `Authenticate with Cursor:
  1. Open Cursor IDE
  2. Sign in to your account
  3. The 'agent' command will use your session`
	default:
		suggestion = fmt.Sprintf(`Configure authentication for %s:
  1. Check the agent's documentation for auth setup
  2. Ensure required environment variables are set
  3. Test with: %s --version`, agentName, agentName)
	}

	return &RalphError{
		Kind:       ErrAuth,
		Message:    fmt.Sprintf("%s authentication not configured", agentName),
		Suggestion: suggestion,
	}
}

// AuthExpired creates an error for expired authentication.
func AuthExpired(agentName string) *RalphError {
	suggestion := ""
	switch strings.ToLower(agentName) {
	case "auggie":
		suggestion = `Your Auggie session has expired. Re-authenticate:
  1. Run: auggie login
  2. Follow the browser prompts`
	default:
		suggestion = fmt.Sprintf("Re-authenticate with %s. Check the agent's documentation.", agentName)
	}

	return &RalphError{
		Kind:       ErrAuth,
		Message:    fmt.Sprintf("%s authentication expired", agentName),
		Suggestion: suggestion,
	}
}

// AgentExecutionFailed creates an error for agent execution failures.
func AgentExecutionFailed(agentName string, exitCode int, stderr string) *RalphError {
	err := &RalphError{
		Kind:    ErrAgent,
		Message: fmt.Sprintf("%s execution failed", agentName),
		Details: map[string]string{
			"agent":     agentName,
			"exit_code": fmt.Sprintf("%d", exitCode),
		},
	}

	// Truncate stderr if too long
	if len(stderr) > 500 {
		stderr = stderr[:500] + "..."
	}
	if stderr != "" {
		err.Details["output"] = stderr
	}

	// Provide context-specific suggestions
	stderrLower := strings.ToLower(stderr)
	if strings.Contains(stderrLower, "not authenticated") || strings.Contains(stderrLower, "unauthorized") {
		err.Suggestion = fmt.Sprintf("Authentication issue detected. Run '%s login' or check your credentials.", agentName)
	} else if strings.Contains(stderrLower, "rate limit") {
		err.Suggestion = "Rate limit reached. Wait a few minutes and try again."
	} else if strings.Contains(stderrLower, "timeout") {
		err.Suggestion = "The agent timed out. Check your network connection or increase the timeout in config."
	}

	return err
}
