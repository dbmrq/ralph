// Package errors provides comprehensive error types for ralph.
// This file contains network and timeout-related errors.
package errors

import (
	"fmt"
	"time"
)

// Network-related error constructors.

// NetworkUnavailable creates an error for network connectivity issues.
func NetworkUnavailable(host string, cause error) *RalphError {
	err := &RalphError{
		Kind:    ErrNetwork,
		Message: "network unavailable",
		Cause:   cause,
		Suggestion: `Check your network connection:
  
  1. Verify internet connectivity
  2. Check if VPN or firewall is blocking access
  3. Try: curl -I https://api.anthropic.com
  
If you're behind a proxy:
  export HTTP_PROXY=http://proxy:port
  export HTTPS_PROXY=http://proxy:port`,
	}
	if host != "" {
		err.Details = map[string]string{"host": host}
	}
	return err
}

// RateLimited creates an error for API rate limiting.
func RateLimited(retryAfter time.Duration) *RalphError {
	suggestion := "Wait before retrying."
	if retryAfter > 0 {
		suggestion = fmt.Sprintf("Wait %v before retrying.", retryAfter.Round(time.Second))
	}
	return &RalphError{
		Kind:       ErrNetwork,
		Message:    "rate limit exceeded",
		Suggestion: suggestion + "\n\nRalph will automatically retry with exponential backoff.",
	}
}

// Timeout-related error constructors.

// AgentTimeout creates an error for agent execution timeout.
func AgentTimeout(elapsed, limit time.Duration, isStuck bool) *RalphError {
	message := fmt.Sprintf("agent timed out after %v", elapsed.Round(time.Second))
	if isStuck {
		message = fmt.Sprintf("agent appears stuck (no output for %v)", elapsed.Round(time.Second))
	}

	suggestion := `The agent took too long to respond.

Possible causes:
  • Complex task requiring more time
  • Agent is stuck or unresponsive
  • Network latency issues

Adjust timeouts in .ralph/config.yaml:
  timeout:
    active: 2h    # Max time while agent is producing output
    stuck: 30m    # Max time with no output (stuck detection)`

	return &RalphError{
		Kind:    ErrTimeout,
		Message: message,
		Details: map[string]string{
			"elapsed": elapsed.Round(time.Second).String(),
			"limit":   limit.Round(time.Second).String(),
			"stuck":   fmt.Sprintf("%t", isStuck),
		},
		Suggestion: suggestion,
	}
}

// OperationTimeout creates a generic timeout error.
func OperationTimeout(operation string, elapsed time.Duration) *RalphError {
	return &RalphError{
		Kind:    ErrTimeout,
		Message: fmt.Sprintf("%s timed out after %v", operation, elapsed.Round(time.Second)),
		Details: map[string]string{
			"operation": operation,
			"elapsed":   elapsed.Round(time.Second).String(),
		},
		Suggestion: "The operation took too long. Check if the system is overloaded or try again later.",
	}
}

// ContextCancelled creates an error for cancelled operations.
func ContextCancelled(operation string) *RalphError {
	return &RalphError{
		Kind:    ErrTimeout,
		Message: fmt.Sprintf("%s was cancelled", operation),
		Details: map[string]string{
			"operation": operation,
		},
		Suggestion: `The operation was interrupted.

If you pressed Ctrl+C:
  • The session state has been saved
  • Resume with: ralph run --continue

If this was unexpected:
  • Check system resources
  • Review logs for errors`,
	}
}

// Helper functions for error detection.

// IsRetryable returns true if the error is likely transient and retrying may succeed.
func IsRetryable(err error) bool {
	if err == nil {
		return false
	}

	// Check if it's a RalphError with a retryable kind
	if re, ok := err.(*RalphError); ok {
		switch re.Kind {
		case ErrNetwork, ErrTimeout:
			return true
		case ErrAuth:
			// Auth expired might be retryable after re-auth
			return false
		default:
			return false
		}
	}

	// Check underlying error for common retryable patterns
	return false
}

// IsUserError returns true if the error is due to user misconfiguration.
func IsUserError(err error) bool {
	if re, ok := err.(*RalphError); ok {
		switch re.Kind {
		case ErrConfig, ErrAuth:
			return true
		default:
			return false
		}
	}
	return false
}

