package errors

import (
	"errors"
	"strings"
	"testing"
)

func TestRalphError_Error(t *testing.T) {
	tests := []struct {
		name     string
		err      *RalphError
		expected string
	}{
		{
			name:     "simple message",
			err:      New(ErrAuth, "authentication failed"),
			expected: "authentication failed",
		},
		{
			name: "with cause",
			err: &RalphError{
				Kind:    ErrConfig,
				Message: "config error",
				Cause:   errors.New("parse error"),
			},
			expected: "config error: parse error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.err.Error(); got != tt.expected {
				t.Errorf("Error() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestRalphError_Unwrap(t *testing.T) {
	cause := errors.New("underlying error")
	err := Wrap(cause, ErrAgent, "wrapped error")

	unwrapped := errors.Unwrap(err)
	if unwrapped != cause {
		t.Errorf("Unwrap() = %v, want %v", unwrapped, cause)
	}

	// Without cause, should return Kind
	errNoWrap := New(ErrAuth, "no cause")
	unwrapped = errors.Unwrap(errNoWrap)
	if !errors.Is(unwrapped, ErrAuth) {
		t.Errorf("Unwrap() should return Kind when no cause")
	}
}

func TestRalphError_Is(t *testing.T) {
	err := New(ErrAuth, "auth failed")

	if !errors.Is(err, ErrAuth) {
		t.Error("errors.Is should return true for matching Kind")
	}

	if errors.Is(err, ErrConfig) {
		t.Error("errors.Is should return false for non-matching Kind")
	}

	// Wrapped errors should still match
	wrapped := Wrap(err, ErrAgent, "wrapped")
	if !errors.Is(wrapped, ErrAgent) {
		t.Error("errors.Is should return true for wrapped error Kind")
	}
}

func TestRalphError_Format(t *testing.T) {
	err := &RalphError{
		Kind:       ErrAuth,
		Message:    "authentication failed",
		Suggestion: "Run 'agent login'",
		DocLink:    "https://example.com/docs",
		Details: map[string]string{
			"agent": "auggie",
		},
	}

	formatted := err.Format()

	// Check all parts are present
	if !strings.Contains(formatted, "Error: authentication failed") {
		t.Error("Format() should contain error message")
	}
	if !strings.Contains(formatted, "💡 Suggestion:") {
		t.Error("Format() should contain suggestion")
	}
	if !strings.Contains(formatted, "Run 'agent login'") {
		t.Error("Format() should contain suggestion text")
	}
	if !strings.Contains(formatted, "📚 Documentation:") {
		t.Error("Format() should contain doc link header")
	}
	if !strings.Contains(formatted, "https://example.com/docs") {
		t.Error("Format() should contain doc link URL")
	}
	if !strings.Contains(formatted, "agent: auggie") {
		t.Error("Format() should contain details")
	}
}

func TestRalphError_WithDetails(t *testing.T) {
	err := New(ErrConfig, "config error")
	err.WithDetails("file", "config.yaml").WithDetails("line", "42")

	if err.Details["file"] != "config.yaml" {
		t.Error("WithDetails should set key")
	}
	if err.Details["line"] != "42" {
		t.Error("WithDetails should allow chaining")
	}
}

func TestRalphError_WithCause(t *testing.T) {
	cause := errors.New("root cause")
	err := New(ErrAgent, "agent error").WithCause(cause)

	if !errors.Is(err.Cause, cause) {
		t.Error("WithCause should set cause")
	}
}

func TestNew(t *testing.T) {
	err := New(ErrBuild, "build failed")

	if !errors.Is(err, ErrBuild) {
		t.Error("New should set Kind")
	}
	if err.Message != "build failed" {
		t.Error("New should set Message")
	}
}

func TestWrap(t *testing.T) {
	cause := errors.New("underlying")
	err := Wrap(cause, ErrTest, "test wrapper")

	if !errors.Is(err, ErrTest) {
		t.Error("Wrap should set Kind")
	}
	if err.Message != "test wrapper" {
		t.Error("Wrap should set Message")
	}
	if err.Cause != cause {
		t.Error("Wrap should set Cause")
	}
}

func TestWithSuggestion(t *testing.T) {
	err := WithSuggestion(ErrGit, "git error", "Run git init")

	if err.Suggestion != "Run git init" {
		t.Error("WithSuggestion should set Suggestion")
	}
}

func TestWithDoc(t *testing.T) {
	err := WithDoc(ErrNetwork, "network error", "https://docs.example.com")

	if err.DocLink != "https://docs.example.com" {
		t.Error("WithDoc should set DocLink")
	}
}
