// Package errors provides comprehensive error types with actionable suggestions
// for the ralph application. Errors include contextual information to help users
// resolve issues quickly.
package errors

import (
	"errors"
	"fmt"
	"strings"
)

// Common sentinel errors for use with errors.Is().
var (
	// ErrAuth indicates an authentication failure.
	ErrAuth = errors.New("authentication error")
	// ErrConfig indicates a configuration error.
	ErrConfig = errors.New("configuration error")
	// ErrAgent indicates an agent-related error.
	ErrAgent = errors.New("agent error")
	// ErrTask indicates a task-related error.
	ErrTask = errors.New("task error")
	// ErrBuild indicates a build failure.
	ErrBuild = errors.New("build error")
	// ErrTest indicates a test failure.
	ErrTest = errors.New("test error")
	// ErrGit indicates a git operation failure.
	ErrGit = errors.New("git error")
	// ErrNetwork indicates a network-related error.
	ErrNetwork = errors.New("network error")
	// ErrTimeout indicates a timeout occurred.
	ErrTimeout = errors.New("timeout error")
	// ErrNotFound indicates a resource was not found.
	ErrNotFound = errors.New("not found")
)

// RalphError is the base error type for ralph errors.
// It wraps an underlying error and provides additional context.
type RalphError struct {
	// Kind is the category of error (e.g., ErrAuth, ErrConfig).
	Kind error
	// Message is the human-readable error message.
	Message string
	// Suggestion provides actionable advice for resolving the error.
	Suggestion string
	// DocLink is a URL to relevant documentation.
	DocLink string
	// Cause is the underlying error that caused this error.
	Cause error
	// Details provides additional context (e.g., file path, command output).
	Details map[string]string
}

// Error implements the error interface.
func (e *RalphError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Cause)
	}
	return e.Message
}

// Unwrap returns the underlying cause for use with errors.Is/errors.As.
func (e *RalphError) Unwrap() error {
	if e.Cause != nil {
		return e.Cause
	}
	return e.Kind
}

// Is reports whether any error in err's chain matches the target.
func (e *RalphError) Is(target error) bool {
	return errors.Is(e.Kind, target)
}

// Format returns a formatted error message with suggestions and doc links.
func (e *RalphError) Format() string {
	var sb strings.Builder

	// Main error message
	sb.WriteString("Error: ")
	sb.WriteString(e.Error())
	sb.WriteString("\n")

	// Details
	if len(e.Details) > 0 {
		sb.WriteString("\nDetails:\n")
		for k, v := range e.Details {
			sb.WriteString(fmt.Sprintf("  %s: %s\n", k, v))
		}
	}

	// Suggestion
	if e.Suggestion != "" {
		sb.WriteString("\n💡 Suggestion: ")
		sb.WriteString(e.Suggestion)
		sb.WriteString("\n")
	}

	// Documentation link
	if e.DocLink != "" {
		sb.WriteString("\n📚 Documentation: ")
		sb.WriteString(e.DocLink)
		sb.WriteString("\n")
	}

	return sb.String()
}

// WithDetails adds details to the error.
func (e *RalphError) WithDetails(key, value string) *RalphError {
	if e.Details == nil {
		e.Details = make(map[string]string)
	}
	e.Details[key] = value
	return e
}

// WithCause sets the underlying cause of the error.
func (e *RalphError) WithCause(cause error) *RalphError {
	e.Cause = cause
	return e
}

// New creates a new RalphError with the given kind and message.
func New(kind error, message string) *RalphError {
	return &RalphError{
		Kind:    kind,
		Message: message,
	}
}

// Wrap wraps an existing error with additional context.
func Wrap(err error, kind error, message string) *RalphError {
	return &RalphError{
		Kind:    kind,
		Message: message,
		Cause:   err,
	}
}

// WithSuggestion creates a new error with a suggestion.
func WithSuggestion(kind error, message, suggestion string) *RalphError {
	return &RalphError{
		Kind:       kind,
		Message:    message,
		Suggestion: suggestion,
	}
}

// WithDoc creates a new error with documentation link.
func WithDoc(kind error, message, docLink string) *RalphError {
	return &RalphError{
		Kind:    kind,
		Message: message,
		DocLink: docLink,
	}
}
