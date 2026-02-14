// Package loop provides the main execution loop for ralph.
// This file implements LOOP-002: core loop execution logic that orchestrates
// task execution through agents with verification gates and hooks.
package loop

import (
	"context"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/wexinc/ralph/internal/agent"
	"github.com/wexinc/ralph/internal/build"
	"github.com/wexinc/ralph/internal/config"
	"github.com/wexinc/ralph/internal/hooks"
	"github.com/wexinc/ralph/internal/logging"
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
	EventCommitStarted     EventType = "commit_started"
	EventCommitCompleted   EventType = "commit_completed"
	EventCommitSkipped     EventType = "commit_skipped"
	EventCommitFailed      EventType = "commit_failed"
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
// It integrates: analysis → task selection → hooks → agent → verify → commit → state update.
type Loop struct {
	// Dependencies
	agent        agent.Agent
	taskManager  *task.Manager
	hookManager  *hooks.Manager
	config       *config.Config
	promptLoader *prompt.Loader
	gitOps       *GitOperations

	// Project analysis (populated during Run)
	analysis *build.ProjectAnalysis

	// State
	context     *LoopContext
	persistence *StatePersistence
	projectDir  string

	// Error recovery
	recovery *ErrorRecovery

	// Options
	opts *Options

	// Control signals for Skip and Abort (LOOP-007)
	// Protected by controlMu for thread-safe access from TUI
	controlMu    sync.Mutex
	skipRequests map[string]bool // taskID -> true if skip requested
	abortRequest bool            // true if abort was requested
	abortReason  string          // reason for abort
}

// NewLoop creates a new Loop with the given dependencies.
func NewLoop(
	ag agent.Agent,
	taskMgr *task.Manager,
	hookMgr *hooks.Manager,
	cfg *config.Config,
	projectDir string,
) *Loop {
	l := &Loop{
		agent:        ag,
		taskManager:  taskMgr,
		hookManager:  hookMgr,
		config:       cfg,
		projectDir:   projectDir,
		promptLoader: prompt.NewLoader(projectDir + "/.ralph"),
		persistence:  NewStatePersistence(projectDir),
		gitOps:       NewGitOperations(projectDir, cfg.Git),
		opts:         DefaultOptions(),
		skipRequests: make(map[string]bool),
	}
	// Initialize error recovery with default config
	l.recovery = NewErrorRecovery(l, nil)
	return l
}

// SetOptions sets the loop options.
func (l *Loop) SetOptions(opts *Options) {
	if opts != nil {
		l.opts = opts
	}
}

// SetRecoveryConfig sets the error recovery configuration.
func (l *Loop) SetRecoveryConfig(cfg *RecoveryConfig) {
	l.recovery = NewErrorRecovery(l, cfg)
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
	log := logging.Global().With("session_id", sessionID)
	log.Info("Loop starting", "project_dir", l.projectDir, "agent", l.agent.Name())

	// Setup signal handler for graceful shutdown
	if l.recovery != nil {
		cleanup := l.recovery.SetupSignalHandler()
		defer cleanup()
	}

	// Initialize context
	l.context = NewLoopContext(sessionID, l.projectDir, l.agent.Name())
	l.context.MaxFixAttempts = l.opts.MaxFixAttempts

	// Transition to running
	if err := l.context.Transition(StateRunning); err != nil {
		log.Error("Failed to start loop", "error", err)
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
		// Check for abort request (LOOP-007)
		if aborted, reason := l.checkAbort(); aborted {
			l.context.SetError(reason)
			if err := l.persistence.Save(l.context); err != nil {
				l.emit(EventError, "", "", 0, "Failed to save state on abort", err)
			}
			if transErr := l.context.Transition(StateFailed); transErr != nil {
				l.emit(EventError, "", "", 0, "Failed to transition on abort", transErr)
			}
			l.emit(EventLoopFailed, "", "", 0, reason, fmt.Errorf("%s", reason))
			return fmt.Errorf("%s", reason)
		}

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

		// Check for skip request before running the task (LOOP-007)
		if l.checkSkip(nextTask.ID) {
			if err := l.taskManager.Skip(nextTask.ID); err != nil {
				l.emit(EventError, nextTask.ID, nextTask.Name, 0, "Failed to skip task", err)
			} else {
				l.context.RecordTaskCompletion(task.StatusSkipped)
				l.emit(EventTaskSkipped, nextTask.ID, nextTask.Name, 0, "Skipped by user request", nil)
			}
			continue
		}

		// Run the task
		result, err := l.runTask(ctx, nextTask)
		if err != nil {
			// Clear any pending skip request for this task
			l.clearSkipRequest(nextTask.ID)

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

		// Clear any pending skip request for this task
		l.clearSkipRequest(nextTask.ID)

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
	log := logging.Global().With("task_id", t.ID, "task_name", t.Name)
	log.Info("Task started")

	l.context.SetCurrentTask(t.ID)
	l.emit(EventTaskStarted, t.ID, t.Name, 0, "Starting task", nil)

	var lastResult *agent.Result

	for iteration := 1; iteration <= l.opts.MaxIterationsPerTask; iteration++ {
		log.Debug("Iteration started", "iteration", iteration)
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
			passed, gateResult, err := l.runVerification(ctx, t)
			if err != nil {
				return lastResult, fmt.Errorf("verification error: %w", err)
			}
			if !passed {
				// Verification failed - attempt auto-fix if enabled
				if l.recovery != nil && l.recovery.config.EnableAutoFix &&
					iteration < l.opts.MaxIterationsPerTask && l.context.CanAttemptFix() {

					// Transition to awaiting fix state and increment fix attempts
					_ = l.context.Transition(StateAwaitingFix)

					// Run auto-fix attempt
					fixResult, fixErr := l.runAutoFix(ctx, t, gateResult, iteration)
					if fixErr != nil {
						l.emit(EventError, t.ID, t.Name, iteration, "Auto-fix failed", fixErr)
						// Continue to next iteration anyway
					} else if fixResult != nil && fixResult.Status == agent.TaskStatusFixed {
						// Fix was successful, update lastResult and continue to verify again
						lastResult = fixResult
						// Transition back to running
						_ = l.context.Transition(StateRunning)
						continue
					}

					// Transition back to running for next iteration
					_ = l.context.Transition(StateRunning)
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

// runAgentForTask runs the agent with the task prompt, including retry logic.
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
		return l.runAgentWithRetry(ctx, t.SessionID, taskPrompt, opts, true)
	}

	// New run with retry
	return l.runAgentWithRetry(ctx, "", taskPrompt, opts, false)
}

// runAgentWithRetry executes agent with retry logic for transient failures.
func (l *Loop) runAgentWithRetry(ctx context.Context, sessionID, prompt string, opts agent.RunOptions, isContinue bool) (agent.Result, error) {
	if l.recovery == nil {
		// No recovery configured, run directly
		if isContinue {
			return l.agent.Continue(ctx, sessionID, prompt, opts)
		}
		return l.agent.Run(ctx, prompt, opts)
	}

	var result agent.Result
	err := l.recovery.RetryWithBackoff(ctx, "agent execution", func() error {
		var runErr error
		if isContinue {
			result, runErr = l.agent.Continue(ctx, sessionID, prompt, opts)
		} else {
			result, runErr = l.agent.Run(ctx, prompt, opts)
		}
		return runErr
	})

	if err != nil {
		return result, err
	}

	// Save session ID for potential continuation
	if result.SessionID != "" {
		l.context.SetAgentSession(result.SessionID)
	}

	return result, nil
}

// buildTaskPrompt constructs the full prompt for a task.
// It uses the TaskPromptBuilder to combine template layers, project analysis,
// and task-specific content into a complete prompt.
func (l *Loop) buildTaskPrompt(t *task.Task, iteration int) (string, error) {
	// Load prompt templates
	templates, err := l.promptLoader.Load()
	if err != nil {
		return "", fmt.Errorf("failed to load prompts: %w", err)
	}

	// Build variables for template substitution
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

	// Use TaskPromptBuilder to combine all prompt components
	builder := prompt.NewTaskPromptBuilder(templates).
		SetAnalysis(l.analysis)

	// TODO: In the future, we can add documentation context and previous changes
	// builder.SetDocsContext(l.loadRelevantDocs())
	// builder.SetPreviousChanges(l.getRecentChanges())

	return builder.BuildForTask(t, vars, iteration), nil
}

// runVerification runs build and test verification gates.
// Returns (passed, gateResult, error) where gateResult is used for fix prompts.
func (l *Loop) runVerification(ctx context.Context, t *task.Task) (bool, *build.GateResult, error) {
	l.emit(EventVerifyStarted, t.ID, t.Name, l.context.CurrentIteration, "Running verification", nil)

	gate := build.NewVerificationGate(l.projectDir, l.config.Build, l.config.Test, l.analysis)
	gate.SessionID = l.context.SessionID

	result, err := gate.Verify(ctx, t)
	if err != nil {
		return false, nil, err
	}

	if result.Passed() {
		l.emit(EventVerifyPassed, t.ID, t.Name, l.context.CurrentIteration, result.Reason, nil)
		return true, result, nil
	}

	l.emit(EventVerifyFailed, t.ID, t.Name, l.context.CurrentIteration, result.Reason, nil)
	return false, result, nil
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

		// Commit changes if auto-commit is enabled
		l.commitTaskChanges(ctx, t)

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

// commitTaskChanges commits the changes made for a task if auto-commit is enabled.
func (l *Loop) commitTaskChanges(ctx context.Context, t *task.Task) {
	if l.gitOps == nil {
		return
	}

	l.emit(EventCommitStarted, t.ID, t.Name, l.context.CurrentIteration, "Committing changes", nil)

	result := l.gitOps.CommitTask(ctx, t)

	if result.Error != nil {
		l.emit(EventCommitFailed, t.ID, t.Name, l.context.CurrentIteration, result.Error.Error(), result.Error)
		return
	}

	if !result.Committed {
		reason := "No changes to commit"
		if !l.config.Git.AutoCommit {
			reason = "Auto-commit disabled"
		}
		l.emit(EventCommitSkipped, t.ID, t.Name, l.context.CurrentIteration, reason, nil)
		return
	}

	message := fmt.Sprintf("Committed: %s", result.Message)
	if result.Pushed {
		message += " (pushed)"
	}
	l.emit(EventCommitCompleted, t.ID, t.Name, l.context.CurrentIteration, message, nil)
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
	if err := l.context.Transition(StatePaused); err != nil {
		return err
	}
	// Save state for resume
	if err := l.persistence.Save(l.context); err != nil {
		l.emit(EventError, "", "", 0, "Failed to save paused state", err)
		// Non-fatal, the pause still happened
	}
	l.emit(EventLoopPaused, "", "", 0,
		fmt.Sprintf("Session %s paused, can resume with --continue", l.context.SessionID), nil)
	return nil
}

// Skip requests that a task be skipped.
// If taskID matches the current task, the skip will be processed at the next check point.
// If taskID is empty, the current task will be skipped.
// The actual skip happens asynchronously when the loop checks for skip requests.
func (l *Loop) Skip(taskID string) error {
	l.controlMu.Lock()
	defer l.controlMu.Unlock()

	if l.context == nil {
		return fmt.Errorf("loop has no context")
	}

	// If no taskID specified, use current task
	if taskID == "" {
		taskID = l.context.CurrentTaskID
		if taskID == "" {
			return fmt.Errorf("no task is currently running")
		}
	}

	// Verify the task exists
	if _, ok := l.taskManager.GetByID(taskID); !ok {
		return fmt.Errorf("task %q not found", taskID)
	}

	// Request the skip - the loop will process this
	l.skipRequests[taskID] = true
	return nil
}

// Abort requests that the loop stop immediately.
// The loop will save state and exit cleanly at the next check point.
// If reason is empty, a default reason will be used.
func (l *Loop) Abort(reason string) error {
	l.controlMu.Lock()
	defer l.controlMu.Unlock()

	if l.context == nil {
		return fmt.Errorf("loop has no context")
	}

	if reason == "" {
		reason = "aborted by user"
	}

	l.abortRequest = true
	l.abortReason = reason
	return nil
}

// checkAbort checks if an abort has been requested.
// Returns true and the reason if abort was requested.
func (l *Loop) checkAbort() (bool, string) {
	l.controlMu.Lock()
	defer l.controlMu.Unlock()
	return l.abortRequest, l.abortReason
}

// checkSkip checks if a skip has been requested for the given task.
// Returns true if skip was requested and clears the request.
func (l *Loop) checkSkip(taskID string) bool {
	l.controlMu.Lock()
	defer l.controlMu.Unlock()
	if l.skipRequests[taskID] {
		delete(l.skipRequests, taskID)
		return true
	}
	return false
}

// clearSkipRequest removes a skip request for a task (called when task ends for any reason).
func (l *Loop) clearSkipRequest(taskID string) {
	l.controlMu.Lock()
	defer l.controlMu.Unlock()
	delete(l.skipRequests, taskID)
}

// Resume resumes a paused session.
// If sessionID is empty, resumes the most recent resumable session.
func (l *Loop) Resume(ctx context.Context, sessionID string) error {
	mgr := NewSessionManager(l.projectDir)

	// Load the session
	resumedCtx, err := mgr.ResumeSession(sessionID)
	if err != nil {
		return fmt.Errorf("cannot resume session: %w", err)
	}

	// Restore the context
	l.context = resumedCtx

	// Transition to running
	if err := l.context.Transition(StateRunning); err != nil {
		return fmt.Errorf("failed to resume loop: %w", err)
	}
	l.emit(EventLoopStarted, "", "", 0,
		fmt.Sprintf("Resuming session %s", l.context.SessionID), nil)

	// Continue the loop from where it left off
	return l.continueLoop(ctx)
}

// ResumeFromContext resumes with an already-loaded context.
// Used when the caller has already loaded the session.
func (l *Loop) ResumeFromContext(ctx context.Context, loopCtx *LoopContext) error {
	if loopCtx == nil {
		return fmt.Errorf("no loop context provided")
	}
	if !loopCtx.State.CanResume() {
		return fmt.Errorf("session cannot be resumed from state: %s", loopCtx.State)
	}

	// Restore the context
	l.context = loopCtx

	// Transition to running
	if err := l.context.Transition(StateRunning); err != nil {
		return fmt.Errorf("failed to resume loop: %w", err)
	}
	l.emit(EventLoopStarted, "", "", 0,
		fmt.Sprintf("Resuming session %s", l.context.SessionID), nil)

	// Continue the loop from where it left off
	return l.continueLoop(ctx)
}

// continueLoop continues the main task loop after resuming.
func (l *Loop) continueLoop(ctx context.Context) error {
	// Setup signal handler for graceful shutdown
	if l.recovery != nil {
		cleanup := l.recovery.SetupSignalHandler()
		defer cleanup()
	}

	// Restore model name from config if set
	if l.config.Agent.Model != "" {
		l.context.ModelName = l.config.Agent.Model
	}

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

	// Step 2: Main task loop (same as Run)
	for {
		// Check for abort request (LOOP-007)
		if aborted, reason := l.checkAbort(); aborted {
			l.context.SetError(reason)
			if err := l.persistence.Save(l.context); err != nil {
				l.emit(EventError, "", "", 0, "Failed to save state on abort", err)
			}
			if transErr := l.context.Transition(StateFailed); transErr != nil {
				l.emit(EventError, "", "", 0, "Failed to transition on abort", transErr)
			}
			l.emit(EventLoopFailed, "", "", 0, reason, fmt.Errorf("%s", reason))
			return fmt.Errorf("%s", reason)
		}

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

		// Check for skip request before running the task (LOOP-007)
		if l.checkSkip(nextTask.ID) {
			if err := l.taskManager.Skip(nextTask.ID); err != nil {
				l.emit(EventError, nextTask.ID, nextTask.Name, 0, "Failed to skip task", err)
			} else {
				l.context.RecordTaskCompletion(task.StatusSkipped)
				l.emit(EventTaskSkipped, nextTask.ID, nextTask.Name, 0, "Skipped by user request", nil)
			}
			continue
		}

		// Run the task
		result, err := l.runTask(ctx, nextTask)
		if err != nil {
			// Clear any pending skip request for this task
			l.clearSkipRequest(nextTask.ID)

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

		// Clear any pending skip request for this task
		l.clearSkipRequest(nextTask.ID)

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

// SessionID returns the current session ID.
func (l *Loop) SessionID() string {
	if l.context == nil {
		return ""
	}
	return l.context.SessionID
}

// AgentSessionID returns the agent's session ID for continuation (e.g., auggie --continue).
func (l *Loop) AgentSessionID() string {
	if l.context == nil {
		return ""
	}
	return l.context.AgentSessionID
}

// truncateOutput truncates output to a reasonable size for storage.
func truncateOutput(output string) string {
	const maxLen = 1000
	if len(output) <= maxLen {
		return output
	}
	return "..." + output[len(output)-maxLen:]
}

// runAutoFix runs an automatic fix attempt for verification failures.
// It builds a fix prompt with failure details and asks the agent to fix the issues.
func (l *Loop) runAutoFix(ctx context.Context, t *task.Task, gateResult *build.GateResult, iteration int) (*agent.Result, error) {
	if gateResult == nil {
		return nil, fmt.Errorf("no gate result for fix prompt")
	}

	// Build fix prompt
	fixBuilder := NewFixPromptBuilder(l.projectDir)
	fixPrompt := fixBuilder.BuildVerificationFixPrompt(t, gateResult)

	l.emit(EventIterationStarted, t.ID, t.Name, iteration,
		fmt.Sprintf("Auto-fix attempt %d for verification failure", l.context.FixAttempts), nil)

	// Configure agent options
	opts := agent.RunOptions{
		Model:     l.config.Agent.Model,
		WorkDir:   l.projectDir,
		Timeout:   l.config.Timeout.Active,
		Force:     true,
		LogWriter: l.opts.LogWriter,
	}

	// Use session continuation if available
	if t.SessionID != "" {
		result, err := l.runAgentWithRetry(ctx, t.SessionID, fixPrompt, opts, true)
		if err != nil {
			return nil, err
		}
		return &result, nil
	}

	// New run
	result, err := l.runAgentWithRetry(ctx, "", fixPrompt, opts, false)
	if err != nil {
		return nil, err
	}

	return &result, nil
}
