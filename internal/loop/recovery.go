// Package loop provides the main execution loop for ralph.
// This file implements LOOP-004: error recovery logic including retry mechanisms,
// automatic fix attempts for verification failures, and graceful shutdown handling.
package loop

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/wexinc/ralph/internal/agent"
	"github.com/wexinc/ralph/internal/build"
	ralpherrors "github.com/wexinc/ralph/internal/errors"
	"github.com/wexinc/ralph/internal/task"
)

// RecoveryConfig configures error recovery behavior.
type RecoveryConfig struct {
	// MaxAgentRetries is the maximum number of retries for agent execution failures.
	MaxAgentRetries int
	// RetryBackoff is the initial backoff duration between retries.
	RetryBackoff time.Duration
	// MaxRetryBackoff is the maximum backoff duration between retries.
	MaxRetryBackoff time.Duration
	// EnableAutoFix enables automatic fix attempts for build/test failures.
	EnableAutoFix bool
}

// DefaultRecoveryConfig returns the default recovery configuration.
func DefaultRecoveryConfig() *RecoveryConfig {
	return &RecoveryConfig{
		MaxAgentRetries: 3,
		RetryBackoff:    5 * time.Second,
		MaxRetryBackoff: 60 * time.Second,
		EnableAutoFix:   true,
	}
}

// ErrorRecovery handles error recovery for the loop.
type ErrorRecovery struct {
	config      *RecoveryConfig
	loop        *Loop
	signalChan  chan os.Signal
	shutdownCtx context.Context
	cancelFunc  context.CancelFunc
}

// NewErrorRecovery creates a new error recovery handler.
func NewErrorRecovery(loop *Loop, config *RecoveryConfig) *ErrorRecovery {
	if config == nil {
		config = DefaultRecoveryConfig()
	}
	ctx, cancel := context.WithCancel(context.Background())
	return &ErrorRecovery{
		config:      config,
		loop:        loop,
		signalChan:  make(chan os.Signal, 1),
		shutdownCtx: ctx,
		cancelFunc:  cancel,
	}
}

// SetupSignalHandler sets up handlers for SIGINT and SIGTERM.
// Returns a cleanup function that should be deferred.
func (r *ErrorRecovery) SetupSignalHandler() func() {
	signal.Notify(r.signalChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		select {
		case sig := <-r.signalChan:
			r.handleShutdown(sig)
		case <-r.shutdownCtx.Done():
			return
		}
	}()

	return func() {
		signal.Stop(r.signalChan)
		r.cancelFunc()
		close(r.signalChan)
	}
}

// handleShutdown handles graceful shutdown on signal.
func (r *ErrorRecovery) handleShutdown(sig os.Signal) {
	if r.loop.opts.OnEvent != nil {
		r.loop.emit(EventError, "", "", 0, fmt.Sprintf("Received %s signal, saving state...", sig), nil)
	}

	// Save current state
	if r.loop.context != nil {
		// Mark as paused for resume
		if r.loop.context.State == StateRunning {
			_ = r.loop.context.Transition(StatePaused)
		}

		if err := r.loop.persistence.Save(r.loop.context); err != nil {
			if r.loop.opts.OnEvent != nil {
				r.loop.emit(EventError, "", "", 0, "Failed to save state on shutdown", err)
			}
		} else if r.loop.opts.OnEvent != nil {
			r.loop.emit(EventLoopPaused, "", "", 0, "State saved, can resume with --continue", nil)
		}
	}

	// Cancel the shutdown context to stop the signal handler goroutine
	r.cancelFunc()
}

// IsRetryableError determines if an error is retryable (transient).
func IsRetryableError(err error) bool {
	if err == nil {
		return false
	}

	// Check if it's a RalphError with retryable kind
	if ralpherrors.IsRetryable(err) {
		return true
	}

	errStr := strings.ToLower(err.Error())

	// Network/timeout errors (fallback for non-RalphError types)
	retryablePatterns := []string{
		"timeout",
		"connection refused",
		"connection reset",
		"temporary failure",
		"network unreachable",
		"no such host",
		"eof",
		"broken pipe",
		"context deadline exceeded",
	}

	for _, pattern := range retryablePatterns {
		if strings.Contains(errStr, pattern) {
			return true
		}
	}

	return false
}

// RetryWithBackoff executes a function with exponential backoff retry.
func (r *ErrorRecovery) RetryWithBackoff(ctx context.Context, name string, fn func() error) error {
	var lastErr error
	backoff := r.config.RetryBackoff

	for attempt := 0; attempt <= r.config.MaxAgentRetries; attempt++ {
		if attempt > 0 {
			// Emit retry event
			if r.loop.opts.OnEvent != nil {
				r.loop.emit(EventError, "", "", 0,
					fmt.Sprintf("Retry %d/%d for %s (waiting %v)", attempt, r.config.MaxAgentRetries, name, backoff),
					lastErr)
			}

			// Wait with backoff
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(backoff):
			}

			// Exponential backoff with cap
			backoff = backoff * 2
			if backoff > r.config.MaxRetryBackoff {
				backoff = r.config.MaxRetryBackoff
			}
		}

		lastErr = fn()
		if lastErr == nil {
			return nil
		}

		// Don't retry non-retryable errors
		if !IsRetryableError(lastErr) {
			return lastErr
		}
	}

	return fmt.Errorf("max retries (%d) exceeded: %w", r.config.MaxAgentRetries, lastErr)
}

// FixPromptBuilder builds prompts for automatic fix attempts.
type FixPromptBuilder struct {
	projectDir string
}

// NewFixPromptBuilder creates a new fix prompt builder.
func NewFixPromptBuilder(projectDir string) *FixPromptBuilder {
	return &FixPromptBuilder{projectDir: projectDir}
}

// BuildBuildFixPrompt builds a prompt to fix build failures.
func (b *FixPromptBuilder) BuildBuildFixPrompt(t *task.Task, result *build.BuildResult) string {
	var sb strings.Builder

	sb.WriteString("# Build Fix Required\n\n")
	sb.WriteString(fmt.Sprintf("The build failed after completing task **%s** (%s).\n\n", t.Name, t.ID))
	sb.WriteString("## Build Errors\n\n")

	if len(result.Errors) > 0 {
		for _, err := range result.Errors {
			if err.File != "" {
				sb.WriteString(fmt.Sprintf("- **%s", err.File))
				if err.Line > 0 {
					sb.WriteString(fmt.Sprintf(":%d", err.Line))
				}
				sb.WriteString(fmt.Sprintf("**: %s\n", err.Message))
			} else {
				sb.WriteString(fmt.Sprintf("- %s\n", err.Message))
			}
		}
	} else {
		sb.WriteString(fmt.Sprintf("Build command failed:\n```\n%s\n```\n", result.Output))
	}

	sb.WriteString("\n## Instructions\n\n")
	sb.WriteString("1. Analyze the build errors above\n")
	sb.WriteString("2. Fix the errors in the code\n")
	sb.WriteString("3. Verify the build passes\n")
	sb.WriteString("4. Report FIXED when done\n")

	return sb.String()
}

// BuildTestFixPrompt builds a prompt to fix test failures.
func (b *FixPromptBuilder) BuildTestFixPrompt(t *task.Task, result *build.TestResult) string {
	var sb strings.Builder

	sb.WriteString("# Test Fix Required\n\n")
	sb.WriteString(fmt.Sprintf("Tests failed after completing task **%s** (%s).\n\n", t.Name, t.ID))
	sb.WriteString("## Test Failures\n\n")

	if len(result.Failures) > 0 {
		for _, failure := range result.Failures {
			// Use TestName field (Name doesn't exist)
			testName := failure.TestName
			if testName == "" {
				testName = "Unknown test"
			}
			sb.WriteString(fmt.Sprintf("### %s\n", testName))
			if failure.Package != "" {
				sb.WriteString(fmt.Sprintf("- Package: `%s`\n", failure.Package))
			}
			if failure.File != "" {
				location := failure.File
				if failure.Line > 0 {
					location = fmt.Sprintf("%s:%d", failure.File, failure.Line)
				}
				sb.WriteString(fmt.Sprintf("- Location: `%s`\n", location))
			}
			if failure.Message != "" {
				sb.WriteString(fmt.Sprintf("- Error: %s\n", failure.Message))
			}
			sb.WriteString("\n")
		}
	} else {
		// Use truncated output if no parsed failures
		output := result.Output
		if len(output) > 500 {
			output = output[:500] + "..."
		}
		sb.WriteString(fmt.Sprintf("Test command failed:\n```\n%s\n```\n", output))
	}

	sb.WriteString("## Instructions\n\n")
	sb.WriteString("1. Analyze the test failures above\n")
	sb.WriteString("2. Fix the issues in the code or tests\n")
	sb.WriteString("3. Verify all tests pass\n")
	sb.WriteString("4. Report FIXED when done\n")

	return sb.String()
}

// BuildVerificationFixPrompt builds a prompt to fix verification failures.
func (b *FixPromptBuilder) BuildVerificationFixPrompt(t *task.Task, gateResult *build.GateResult) string {
	// If build failed, use build fix prompt
	if gateResult.BuildResult != nil && !gateResult.BuildResult.Success {
		return b.BuildBuildFixPrompt(t, gateResult.BuildResult)
	}

	// If tests failed, use test fix prompt
	if gateResult.TestResult != nil && !gateResult.TestResult.Success {
		return b.BuildTestFixPrompt(t, gateResult.TestResult)
	}

	// Generic failure prompt
	return fmt.Sprintf(`# Verification Fix Required

The verification gate failed after completing task **%s** (%s).

Reason: %s

## Instructions

1. Investigate the failure reason
2. Fix any issues in the code
3. Verify the build passes: %s
4. Verify tests pass
5. Report FIXED when done
`, t.Name, t.ID, gateResult.Reason, b.projectDir)
}

// RunAgentWithRetry runs an agent execution with retry logic.
func (r *ErrorRecovery) RunAgentWithRetry(
	ctx context.Context,
	ag agent.Agent,
	prompt string,
	opts agent.RunOptions,
) (agent.Result, error) {
	var result agent.Result
	var lastErr error

	err := r.RetryWithBackoff(ctx, "agent execution", func() error {
		var runErr error
		result, runErr = ag.Run(ctx, prompt, opts)
		if runErr != nil {
			lastErr = runErr
			return runErr
		}
		return nil
	})

	if err != nil {
		return result, err
	}

	return result, lastErr
}

