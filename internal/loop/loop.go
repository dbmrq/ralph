// Package loop provides the main execution loop for ralph.
// This file implements LOOP-002: core loop execution logic that orchestrates
// task execution through agents with verification gates and hooks.
package loop

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/wexinc/ralph/internal/agent"
	"github.com/wexinc/ralph/internal/build"
	"github.com/wexinc/ralph/internal/config"
	"github.com/wexinc/ralph/internal/hooks"
	"github.com/wexinc/ralph/internal/prompt"
	"github.com/wexinc/ralph/internal/task"
)

// DefaultMaxIterations is the default maximum iterations per task.
const DefaultMaxIterations = 5

// DefaultMaxFixAttempts is the default maximum fix attempts per failure.
const DefaultMaxFixAttempts = 3

// EventType identifies the type of loop event.
type EventType string

const (
	EventAnalysisStarted   EventType = "analysis_started"
	EventAnalysisCompleted EventType = "analysis_completed"
	EventAnalysisFailed    EventType = "analysis_failed"
	EventLoopStarted       EventType = "loop_started"
	EventLoopCompleted     EventType = "loop_completed"
	EventLoopFailed        EventType = "loop_failed"
	EventLoopPaused        EventType = "loop_paused"
	EventTaskStarted       EventType = "task_started"
	EventTaskCompleted     EventType = "task_completed"
	EventTaskSkipped       EventType = "task_skipped"
	EventTaskFailed        EventType = "task_failed"
	EventIterationStarted  EventType = "iteration_started"
	EventIterationEnded    EventType = "iteration_ended"
	EventVerifyStarted     EventType = "verify_started"
	EventVerifyPassed      EventType = "verify_passed"
	EventVerifyFailed      EventType = "verify_failed"
	EventHooksStarted      EventType = "hooks_started"
	EventHooksCompleted    EventType = "hooks_completed"
	EventError             EventType = "error"
)

// Event represents a loop event for observers (TUI, logging, etc.).
type Event struct {
	Type      EventType
	TaskID    string
	TaskName  string
	Iteration int
	Message   string
	Error     error
	Timestamp time.Time
}

// EventHandler is a callback for loop events.
type EventHandler func(event Event)

// Options configures loop execution.
type Options struct {
	// MaxIterationsPerTask is the maximum iterations allowed per task.
	MaxIterationsPerTask int
	// MaxFixAttempts is the maximum fix attempts per verification failure.
	MaxFixAttempts int
	// LogWriter receives real-time agent output (optional).
	LogWriter io.Writer
	// OnEvent is called for each loop event (optional).
	OnEvent EventHandler
}

// DefaultOptions returns default loop options.
func DefaultOptions() *Options {
	return &Options{
		MaxIterationsPerTask: DefaultMaxIterations,
		MaxFixAttempts:       DefaultMaxFixAttempts,
	}
}

// Loop orchestrates the main execution flow for ralph.
// It integrates: analysis → task selection → hooks → agent → verify → state update.
type Loop struct {
	// Dependencies
	agent        agent.Agent
	taskManager  *task.Manager
	hookManager  *hooks.Manager
	config       *config.Config
	promptLoader *prompt.Loader

	// Project analysis (populated during Run)
	analysis *build.ProjectAnalysis

	// State
	context     *LoopContext
	persistence *StatePersistence
	projectDir  string

	// Options
	opts *Options
}

// NewLoop creates a new Loop with the given dependencies.
func NewLoop(
	ag agent.Agent,
	taskMgr *task.Manager,
	hookMgr *hooks.Manager,
	cfg *config.Config,
	projectDir string,
) *Loop {
	return &Loop{
		agent:        ag,
		taskManager:  taskMgr,
		hookManager:  hookMgr,
		config:       cfg,
		projectDir:   projectDir,
		promptLoader: prompt.NewLoader(projectDir + "/.ralph"),
		persistence:  NewStatePersistence(projectDir),
		opts:         DefaultOptions(),
	}
}

// SetOptions sets the loop options.
func (l *Loop) SetOptions(opts *Options) {
	if opts != nil {
		l.opts = opts
	}
}

// SetAnalysis sets a pre-computed project analysis (skips analysis phase).
// This is useful when analysis was already performed during setup.
func (l *Loop) SetAnalysis(analysis *build.ProjectAnalysis) {
	l.analysis = analysis
}

// Context returns the current loop context.
func (l *Loop) Context() *LoopContext {
	return l.context
}

// emit sends an event to the event handler if configured.
func (l *Loop) emit(eventType EventType, taskID, taskName string, iteration int, message string, err error) {
	if l.opts.OnEvent != nil {
		l.opts.OnEvent(Event{
			Type:      eventType,
			TaskID:    taskID,
			TaskName:  taskName,
			Iteration: iteration,
			Message:   message,
			Error:     err,
			Timestamp: time.Now(),
		})
	}
}

// Run executes the main loop until all tasks are complete or an error occurs.
// The flow is:
// 1. Run Project Analysis Agent (if not already set)
// 2. Initialize loop context
// 3. For each task: pre-hooks → agent → verify → post-hooks → state update
// 4. Handle errors, pauses, and completions
func (l *Loop) Run(ctx context.Context, sessionID string) error {
	// Initialize context
	l.context = NewLoopContext(sessionID, l.projectDir, l.agent.Name())
	l.context.MaxFixAttempts = l.opts.MaxFixAttempts

	// Transition to running
	if err := l.context.Transition(StateRunning); err != nil {
		return fmt.Errorf("failed to start loop: %w", err)
	}
	l.emit(EventLoopStarted, "", "", 0, "Loop started", nil)

	// Step 1: Run project analysis if not already set
	if l.analysis == nil {
		if err := l.runAnalysis(ctx); err != nil {
			l.context.SetError(err.Error())
			if transErr := l.context.Transition(StateFailed); transErr != nil {
				return fmt.Errorf("analysis failed: %w (also failed to transition: %v)", err, transErr)
			}
			l.emit(EventAnalysisFailed, "", "", 0, "Analysis failed", err)
			return fmt.Errorf("analysis failed: %w", err)
		}
	}

	// Step 2: Main task loop
	for {
		// Check for cancellation
		if ctx.Err() != nil {
			l.context.SetError("cancelled")
			_ = l.context.Transition(StateFailed)
			return ctx.Err()
		}

		// Get next task
		nextTask := l.taskManager.GetNext()
		if nextTask == nil {
			// All tasks complete
			if err := l.context.Transition(StateCompleted); err != nil {
				return fmt.Errorf("failed to complete loop: %w", err)
			}
			l.emit(EventLoopCompleted, "", "", 0, "All tasks completed", nil)
			return nil
		}

		// Run the task
		result, err := l.runTask(ctx, nextTask)
		if err != nil {
			// Handle task-level errors
			if ctx.Err() != nil {
				return ctx.Err() // Cancelled
			}

			// Record failure
			l.context.SetError(err.Error())
			l.context.RecordTaskCompletion(task.StatusFailed)

			// If we can't continue, fail the loop
			if !l.canContinueAfterError(err) {
				if transErr := l.context.Transition(StateFailed); transErr != nil {
					return fmt.Errorf("task failed: %w (also failed to transition: %v)", err, transErr)
				}
				l.emit(EventLoopFailed, nextTask.ID, nextTask.Name, 0, "Loop failed", err)
				return err
			}

			// Mark task as failed but continue
			if markErr := l.taskManager.MarkFailed(nextTask.ID); markErr != nil {
				return fmt.Errorf("failed to mark task as failed: %w", markErr)
			}
			l.emit(EventTaskFailed, nextTask.ID, nextTask.Name, 0, err.Error(), err)
			continue
		}

		// Handle task result
		if err := l.handleTaskResult(ctx, nextTask, result); err != nil {
			return err
		}

		// Save state after each task
		if err := l.persistence.Save(l.context); err != nil {
			l.emit(EventError, "", "", 0, "Failed to save state", err)
			// Non-fatal, continue
		}
	}
}

// runAnalysis runs the Project Analysis Agent.
func (l *Loop) runAnalysis(ctx context.Context) error {
	l.emit(EventAnalysisStarted, "", "", 0, "Running project analysis", nil)

	analyzer := build.NewProjectAnalyzer(l.projectDir, l.agent)
	if l.opts.LogWriter != nil {
		analyzer.LogWriter = l.opts.LogWriter
	}
	analyzer.OnProgress = func(status string) {
		l.emit(EventAnalysisStarted, "", "", 0, status, nil)
	}

	analysis, err := analyzer.AnalyzeWithFallback(ctx)
	if err != nil {
		return err
	}

	l.analysis = analysis

	// Cache the analysis
	if err := analyzer.SaveCache(analysis); err != nil {
		l.emit(EventError, "", "", 0, "Failed to cache analysis", err)
		// Non-fatal, continue
	}

	l.emit(EventAnalysisCompleted, "", "", 0, "Analysis complete", nil)
	return nil
}

// runTask executes a single task through all phases.
// Returns the last agent result (may be nil if task was skipped) and any error.
func (l *Loop) runTask(ctx context.Context, t *task.Task) (*agent.Result, error) {
	l.context.SetCurrentTask(t.ID)
	l.emit(EventTaskStarted, t.ID, t.Name, 0, "Starting task", nil)

	var lastResult *agent.Result

	for iteration := 1; iteration <= l.opts.MaxIterationsPerTask; iteration++ {
		l.context.IncrementIteration()
		l.emit(EventIterationStarted, t.ID, t.Name, iteration, fmt.Sprintf("Iteration %d", iteration), nil)

		// Start iteration tracking
		if _, err := l.taskManager.StartIteration(t.ID); err != nil {
			return nil, fmt.Errorf("failed to start iteration: %w", err)
		}

		// Phase 1: Pre-task hooks
		if l.hookManager != nil && l.hookManager.HasPreTaskHooks() {
			l.emit(EventHooksStarted, t.ID, t.Name, iteration, "Running pre-task hooks", nil)
			hookCtx := hooks.BuildHookContextForPreTask(t, iteration, l.projectDir)
			hookResult := l.hookManager.ExecutePreTaskHooks(ctx, hookCtx)

			if hookResult.Action == hooks.ManagerActionAbortLoop {
				return nil, fmt.Errorf("pre-task hook aborted loop: %s", l.hookManager.GetFailedHookInfo(hookResult))
			}
			if hookResult.Action == hooks.ManagerActionSkipTask {
				if err := l.taskManager.Skip(t.ID); err != nil {
					return nil, fmt.Errorf("failed to skip task: %w", err)
				}
				l.emit(EventTaskSkipped, t.ID, t.Name, iteration, "Skipped by pre-task hook", nil)
				return nil, nil // Task skipped, not an error
			}
			l.emit(EventHooksCompleted, t.ID, t.Name, iteration, "Pre-task hooks completed", nil)
		}

		// Phase 2: Run agent
		result, err := l.runAgentForTask(ctx, t, iteration)
		if err != nil {
			if endErr := l.taskManager.EndIteration(t.ID, "ERROR", err.Error(), ""); endErr != nil {
				l.emit(EventError, t.ID, t.Name, iteration, "Failed to end iteration", endErr)
			}
			return nil, err
		}
		// Store as pointer for return value and hooks
		lastResult = &result

		// End iteration tracking
		if err := l.taskManager.EndIteration(t.ID, string(result.Status), truncateOutput(result.Output), result.SessionID); err != nil {
			l.emit(EventError, t.ID, t.Name, iteration, "Failed to end iteration", err)
		}

		l.emit(EventIterationEnded, t.ID, t.Name, iteration, fmt.Sprintf("Agent returned: %s", result.Status), nil)

		// Phase 3: Verification
		if result.Status.IsSuccess() {
			passed, err := l.runVerification(ctx, t)
			if err != nil {
				return lastResult, fmt.Errorf("verification error: %w", err)
			}
			if !passed {
				// Verification failed, but we might retry
				if iteration < l.opts.MaxIterationsPerTask && l.context.CanAttemptFix() {
					l.emit(EventVerifyFailed, t.ID, t.Name, iteration, "Verification failed, will retry", nil)
					continue // Retry
				}
				return lastResult, fmt.Errorf("verification failed after %d iterations", iteration)
			}
		}

		// Phase 4: Post-task hooks
		if l.hookManager != nil && l.hookManager.HasPostTaskHooks() {
			l.emit(EventHooksStarted, t.ID, t.Name, iteration, "Running post-task hooks", nil)
			hookCtx := hooks.BuildHookContextForPostTask(t, lastResult, iteration, l.projectDir)
			hookResult := l.hookManager.ExecutePostTaskHooks(ctx, hookCtx)

			if hookResult.Action == hooks.ManagerActionAbortLoop {
				return lastResult, fmt.Errorf("post-task hook aborted loop: %s", l.hookManager.GetFailedHookInfo(hookResult))
			}
			l.emit(EventHooksCompleted, t.ID, t.Name, iteration, "Post-task hooks completed", nil)
		}

		// Check agent status
		switch result.Status {
		case agent.TaskStatusDone:
			return lastResult, nil // Task complete
		case agent.TaskStatusNext:
			if iteration < l.opts.MaxIterationsPerTask {
				continue // More work needed
			}
			return lastResult, fmt.Errorf("task not completed after %d iterations", l.opts.MaxIterationsPerTask)
		case agent.TaskStatusError:
			return lastResult, fmt.Errorf("agent reported error: %s", result.Error)
		case agent.TaskStatusFixed:
			// Fixed a previous issue, continue
			continue
		default:
			// Unknown status, treat as needing more work
			if iteration < l.opts.MaxIterationsPerTask {
				continue
			}
		}
	}

	return lastResult, fmt.Errorf("maximum iterations (%d) reached", l.opts.MaxIterationsPerTask)
}

// runAgentForTask runs the agent with the task prompt.
func (l *Loop) runAgentForTask(ctx context.Context, t *task.Task, iteration int) (agent.Result, error) {
	// Build prompt
	taskPrompt, err := l.buildTaskPrompt(t, iteration)
	if err != nil {
		return agent.Result{}, fmt.Errorf("failed to build prompt: %w", err)
	}

	// Configure agent options
	opts := agent.RunOptions{
		Model:     l.config.Agent.Model,
		WorkDir:   l.projectDir,
		Timeout:   l.config.Timeout.Active,
		Force:     true,
		LogWriter: l.opts.LogWriter,
	}

	// Check if this is a continuation
	if iteration > 1 && t.SessionID != "" {
		l.context.SetAgentSession(t.SessionID)
		return l.agent.Continue(ctx, t.SessionID, taskPrompt, opts)
	}

	// New run
	result, err := l.agent.Run(ctx, taskPrompt, opts)
	if err != nil {
		return agent.Result{}, err
	}

	// Save session ID for potential continuation
	if result.SessionID != "" {
		l.context.SetAgentSession(result.SessionID)
	}

	return result, nil
}

// buildTaskPrompt constructs the full prompt for a task.
func (l *Loop) buildTaskPrompt(t *task.Task, iteration int) (string, error) {
	// Load prompt templates
	templates, err := l.promptLoader.Load()
	if err != nil {
		return "", fmt.Errorf("failed to load prompts: %w", err)
	}

	// Build variables
	vars := &prompt.Variables{
		TaskID:          t.ID,
		TaskName:        t.Name,
		TaskDescription: t.Description,
		TaskStatus:      string(t.Status),
		Iteration:       iteration,
		ProjectDir:      l.projectDir,
		AgentName:       l.agent.Name(),
		Model:           l.config.Agent.Model,
		SessionID:       l.context.SessionID,
	}

	// Build base prompt
	builder := prompt.NewBuilder(templates)
	basePrompt := builder.Build(vars)

	// Add project analysis context
	analysisContext := l.buildAnalysisContext()

	// Build task content
	taskContent := l.buildTaskContent(t, iteration)

	// Combine all parts
	fullPrompt := basePrompt + "\n\n---\n\n# Project Context (from analysis)\n\n" + analysisContext
	fullPrompt += "\n\n---\n\n# Current Task\n\n" + taskContent

	return fullPrompt, nil
}

// buildAnalysisContext creates the analysis context section for prompts.
func (l *Loop) buildAnalysisContext() string {
	if l.analysis == nil {
		return "Project analysis not available."
	}

	var parts []string

	parts = append(parts, fmt.Sprintf("Project Type: %s", l.analysis.ProjectType))

	if len(l.analysis.Languages) > 0 {
		parts = append(parts, fmt.Sprintf("Languages: %s", strings.Join(l.analysis.Languages, ", ")))
	}

	if l.analysis.Build.Command != nil {
		parts = append(parts, fmt.Sprintf("Build Command: %s", *l.analysis.Build.Command))
	}

	if l.analysis.Test.Command != nil {
		parts = append(parts, fmt.Sprintf("Test Command: %s", *l.analysis.Test.Command))
	}

	if l.analysis.Dependencies.Manager != "" {
		installed := "No"
		if l.analysis.Dependencies.Installed {
			installed = "Yes"
		}
		parts = append(parts, fmt.Sprintf("Package Manager: %s (installed: %s)", l.analysis.Dependencies.Manager, installed))
	}

	if l.analysis.IsGreenfield {
		parts = append(parts, "Status: Greenfield project (no buildable code yet)")
	}

	if l.analysis.ProjectContext != "" {
		parts = append(parts, fmt.Sprintf("\n%s", l.analysis.ProjectContext))
	}

	return strings.Join(parts, "\n")
}

// buildTaskContent creates the task-specific content for prompts.
func (l *Loop) buildTaskContent(t *task.Task, iteration int) string {
	var parts []string

	parts = append(parts, fmt.Sprintf("**Task ID:** %s", t.ID))
	parts = append(parts, fmt.Sprintf("**Task:** %s", t.Name))

	if t.Description != "" {
		parts = append(parts, fmt.Sprintf("\n**Description:**\n%s", t.Description))
	}

	if iteration > 1 {
		parts = append(parts, fmt.Sprintf("\n**Iteration:** %d (previous attempt did not complete the task)", iteration))

		// Include last iteration result if available
		if lastIter := t.CurrentIteration(); lastIter != nil && lastIter.Result != "" {
			parts = append(parts, fmt.Sprintf("**Previous result:** %s", lastIter.Result))
		}
	}

	return strings.Join(parts, "\n")
}

// runVerification runs build and test verification gates.
func (l *Loop) runVerification(ctx context.Context, t *task.Task) (bool, error) {
	l.emit(EventVerifyStarted, t.ID, t.Name, l.context.CurrentIteration, "Running verification", nil)

	gate := build.NewVerificationGate(l.projectDir, l.config.Build, l.config.Test, l.analysis)
	gate.SessionID = l.context.SessionID

	result, err := gate.Verify(ctx, t)
	if err != nil {
		return false, err
	}

	if result.Passed() {
		l.emit(EventVerifyPassed, t.ID, t.Name, l.context.CurrentIteration, result.Reason, nil)
		return true, nil
	}

	l.emit(EventVerifyFailed, t.ID, t.Name, l.context.CurrentIteration, result.Reason, nil)
	return false, nil
}

// handleTaskResult processes the result of a task execution.
func (l *Loop) handleTaskResult(ctx context.Context, t *task.Task, result *agent.Result) error {
	if result == nil {
		// Task was skipped
		l.context.RecordTaskCompletion(task.StatusSkipped)
		return nil
	}

	switch result.Status {
	case agent.TaskStatusDone, agent.TaskStatusFixed:
		// Task completed successfully
		if err := l.taskManager.MarkComplete(t.ID); err != nil {
			return fmt.Errorf("failed to mark task complete: %w", err)
		}
		l.context.RecordTaskCompletion(task.StatusCompleted)
		l.emit(EventTaskCompleted, t.ID, t.Name, l.context.CurrentIteration, "Task completed", nil)

	case agent.TaskStatusError:
		// Agent reported error
		if err := l.taskManager.MarkFailed(t.ID); err != nil {
			return fmt.Errorf("failed to mark task failed: %w", err)
		}
		l.context.RecordTaskCompletion(task.StatusFailed)
		l.emit(EventTaskFailed, t.ID, t.Name, l.context.CurrentIteration, result.Error, nil)

	default:
		// Unexpected state after all iterations
		if err := l.taskManager.MarkFailed(t.ID); err != nil {
			return fmt.Errorf("failed to mark task failed: %w", err)
		}
		l.context.RecordTaskCompletion(task.StatusFailed)
		l.emit(EventTaskFailed, t.ID, t.Name, l.context.CurrentIteration, "Task did not complete", nil)
	}

	return nil
}

// canContinueAfterError determines if the loop can continue after an error.
func (l *Loop) canContinueAfterError(err error) bool {
	// For now, most errors are recoverable - we mark the task as failed and continue
	// Context cancellation is not recoverable
	if l.context.State != StateRunning {
		return false
	}
	return true
}

// Pause pauses the loop after the current task.
func (l *Loop) Pause() error {
	if l.context == nil || l.context.State != StateRunning {
		return fmt.Errorf("loop is not running")
	}
	return l.context.Transition(StatePaused)
}

// Resume resumes a paused loop.
func (l *Loop) Resume(ctx context.Context) error {
	if l.context == nil {
		return fmt.Errorf("no loop context")
	}
	if !l.context.State.CanResume() {
		return fmt.Errorf("loop cannot be resumed from state: %s", l.context.State)
	}

	if err := l.context.Transition(StateRunning); err != nil {
		return err
	}

	// Continue with the main loop (this is a simplified resume - full implementation
	// would restore all state and continue from where we left off)
	return l.Run(ctx, l.context.SessionID)
}

// truncateOutput truncates output to a reasonable size for storage.
func truncateOutput(output string) string {
	const maxLen = 1000
	if len(output) <= maxLen {
		return output
	}
	return "..." + output[len(output)-maxLen:]
}

