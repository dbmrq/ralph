package errors

import (
	"errors"
	"strings"
	"testing"
	"time"
)

func TestNetworkUnavailable(t *testing.T) {
	cause := errors.New("connection refused")
	err := NetworkUnavailable("api.example.com", cause)

	if !errors.Is(err, ErrNetwork) {
		t.Error("NetworkUnavailable should return ErrNetwork")
	}
	if !errors.Is(err.Cause, cause) {
		t.Error("Should wrap the cause")
	}
	if err.Details["host"] != "api.example.com" {
		t.Error("Should include host in details")
	}
	if !strings.Contains(err.Suggestion, "VPN") {
		t.Error("Suggestion should mention common network issues")
	}
}

func TestNetworkUnavailable_NoHost(t *testing.T) {
	err := NetworkUnavailable("", nil)

	if err.Details != nil {
		t.Error("Should not include details when host is empty")
	}
}

func TestRateLimited(t *testing.T) {
	err := RateLimited(30 * time.Second)

	if !errors.Is(err, ErrNetwork) {
		t.Error("RateLimited should return ErrNetwork")
	}
	if !strings.Contains(err.Suggestion, "30s") {
		t.Error("Suggestion should include retry time")
	}
}

func TestRateLimited_NoRetryAfter(t *testing.T) {
	err := RateLimited(0)

	if !strings.Contains(err.Suggestion, "Wait before retrying") {
		t.Error("Should provide generic wait message")
	}
}

func TestAgentTimeout(t *testing.T) {
	err := AgentTimeout(35*time.Minute, 30*time.Minute, false)

	if !errors.Is(err, ErrTimeout) {
		t.Error("AgentTimeout should return ErrTimeout")
	}
	if !strings.Contains(err.Message, "35m") {
		t.Error("Message should include elapsed time")
	}
	if !strings.Contains(err.Suggestion, "timeout:") {
		t.Error("Suggestion should mention config options")
	}
}

func TestAgentTimeout_Stuck(t *testing.T) {
	err := AgentTimeout(15*time.Minute, 30*time.Minute, true)

	if !strings.Contains(err.Message, "stuck") {
		t.Error("Message should indicate stuck state")
	}
	if err.Details["stuck"] != "true" {
		t.Error("Details should show stuck status")
	}
}

func TestOperationTimeout(t *testing.T) {
	err := OperationTimeout("build", 5*time.Minute)

	if !errors.Is(err, ErrTimeout) {
		t.Error("OperationTimeout should return ErrTimeout")
	}
	if !strings.Contains(err.Message, "build") {
		t.Error("Message should include operation name")
	}
	if err.Details["operation"] != "build" {
		t.Error("Details should include operation")
	}
}

func TestContextCancelled(t *testing.T) {
	err := ContextCancelled("task execution")

	if !errors.Is(err, ErrTimeout) {
		t.Error("ContextCancelled should return ErrTimeout")
	}
	if !strings.Contains(err.Message, "cancelled") {
		t.Error("Message should indicate cancellation")
	}
	if !strings.Contains(err.Suggestion, "--continue") {
		t.Error("Suggestion should mention resume option")
	}
}

func TestIsRetryable(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
		{
			name:     "network error",
			err:      &RalphError{Kind: ErrNetwork, Message: "network"},
			expected: true,
		},
		{
			name:     "timeout error",
			err:      &RalphError{Kind: ErrTimeout, Message: "timeout"},
			expected: true,
		},
		{
			name:     "auth error",
			err:      &RalphError{Kind: ErrAuth, Message: "auth"},
			expected: false,
		},
		{
			name:     "config error",
			err:      &RalphError{Kind: ErrConfig, Message: "config"},
			expected: false,
		},
		{
			name:     "plain error",
			err:      errors.New("plain"),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsRetryable(tt.err); got != tt.expected {
				t.Errorf("IsRetryable() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestIsUserError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "config error",
			err:      &RalphError{Kind: ErrConfig, Message: "config"},
			expected: true,
		},
		{
			name:     "auth error",
			err:      &RalphError{Kind: ErrAuth, Message: "auth"},
			expected: true,
		},
		{
			name:     "build error",
			err:      &RalphError{Kind: ErrBuild, Message: "build"},
			expected: false,
		},
		{
			name:     "plain error",
			err:      errors.New("plain"),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsUserError(tt.err); got != tt.expected {
				t.Errorf("IsUserError() = %v, want %v", got, tt.expected)
			}
		})
	}
}

