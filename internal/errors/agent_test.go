package errors

import (
	"errors"
	"strings"
	"testing"
)

func TestAgentNotAvailable(t *testing.T) {
	err := AgentNotAvailable("cursor")

	if !errors.Is(err, ErrAgent) {
		t.Error("AgentNotAvailable should return ErrAgent")
	}
	if !strings.Contains(err.Message, "cursor") {
		t.Error("Error message should contain agent name")
	}
	if !strings.Contains(err.Suggestion, "Install") {
		t.Error("Suggestion should mention installation")
	}
}

func TestAgentNotFound(t *testing.T) {
	err := AgentNotFound("unknown", []string{"cursor", "auggie"})

	if !errors.Is(err, ErrAgent) {
		t.Error("AgentNotFound should return ErrAgent")
	}
	if !strings.Contains(err.Message, "unknown") {
		t.Error("Error message should contain requested agent")
	}
	if err.Details["available"] != "cursor, auggie" {
		t.Errorf("Details should list available agents, got %v", err.Details["available"])
	}
}

func TestAgentNotFound_NoAvailable(t *testing.T) {
	err := AgentNotFound("unknown", nil)

	if err.Details["available"] != "none" {
		t.Error("Should show 'none' when no agents available")
	}
}

func TestNoAgentsAvailable(t *testing.T) {
	err := NoAgentsAvailable()

	if !errors.Is(err, ErrAgent) {
		t.Error("NoAgentsAvailable should return ErrAgent")
	}
	if !strings.Contains(err.Suggestion, "Install an AI coding agent") {
		t.Error("Suggestion should help user install an agent")
	}
	if err.DocLink == "" {
		t.Error("Should include documentation link")
	}
}

func TestMultipleAgentsNeedSelection(t *testing.T) {
	err := MultipleAgentsNeedSelection([]string{"cursor", "auggie"})

	if !errors.Is(err, ErrAgent) {
		t.Error("MultipleAgentsNeedSelection should return ErrAgent")
	}
	if !strings.Contains(err.Suggestion, "config") {
		t.Error("Suggestion should mention configuration")
	}
	if !strings.Contains(err.Details["available"], "cursor") {
		t.Error("Should list available agents")
	}
}

func TestAuthNotConfigured(t *testing.T) {
	tests := []struct {
		agent      string
		shouldHave string
	}{
		{"auggie", "auggie login"},
		{"cursor", "Cursor IDE"},
		{"custom", "documentation"},
	}

	for _, tt := range tests {
		t.Run(tt.agent, func(t *testing.T) {
			err := AuthNotConfigured(tt.agent)

			if !errors.Is(err, ErrAuth) {
				t.Error("AuthNotConfigured should return ErrAuth")
			}
			if !strings.Contains(err.Suggestion, tt.shouldHave) {
				t.Errorf("Suggestion for %s should contain %q", tt.agent, tt.shouldHave)
			}
		})
	}
}

func TestAuthExpired(t *testing.T) {
	err := AuthExpired("auggie")

	if !errors.Is(err, ErrAuth) {
		t.Error("AuthExpired should return ErrAuth")
	}
	if !strings.Contains(err.Message, "expired") {
		t.Error("Message should mention expiration")
	}
}

func TestAgentExecutionFailed(t *testing.T) {
	err := AgentExecutionFailed("auggie", 1, "command failed")

	if !errors.Is(err, ErrAgent) {
		t.Error("AgentExecutionFailed should return ErrAgent")
	}
	if err.Details["exit_code"] != "1" {
		t.Error("Should include exit code")
	}
	if err.Details["output"] != "command failed" {
		t.Error("Should include output")
	}
}

func TestAgentExecutionFailed_WithAuthError(t *testing.T) {
	err := AgentExecutionFailed("auggie", 1, "not authenticated")

	if !strings.Contains(err.Suggestion, "Authentication") {
		t.Error("Should detect auth issue and suggest fix")
	}
}

func TestAgentExecutionFailed_WithRateLimit(t *testing.T) {
	err := AgentExecutionFailed("auggie", 1, "rate limit exceeded")

	if !strings.Contains(err.Suggestion, "Rate limit") {
		t.Error("Should detect rate limit and suggest waiting")
	}
}

func TestAgentExecutionFailed_LongStderr(t *testing.T) {
	longOutput := strings.Repeat("x", 600)
	err := AgentExecutionFailed("auggie", 1, longOutput)

	if len(err.Details["output"]) > 550 {
		t.Error("Long output should be truncated")
	}
	if !strings.HasSuffix(err.Details["output"], "...") {
		t.Error("Truncated output should end with ...")
	}
}
