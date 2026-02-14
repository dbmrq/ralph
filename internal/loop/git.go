// Package loop provides the main execution loop for ralph.
// This file implements LOOP-003: automatic commit logic for git operations.
package loop

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"

	"github.com/dbmrq/ralph/internal/config"
	"github.com/dbmrq/ralph/internal/task"
)

// GitOperations handles git operations for automatic commits.
type GitOperations struct {
	// WorkDir is the working directory (project root).
	WorkDir string
	// Config contains git configuration settings.
	Config config.GitConfig
}

// NewGitOperations creates a new GitOperations instance.
func NewGitOperations(workDir string, cfg config.GitConfig) *GitOperations {
	return &GitOperations{
		WorkDir: workDir,
		Config:  cfg,
	}
}

// CommitResult contains the result of a commit operation.
type CommitResult struct {
	// Committed indicates if a commit was made.
	Committed bool
	// Message is the commit message used.
	Message string
	// Pushed indicates if the commit was pushed.
	Pushed bool
	// Error is any error that occurred.
	Error error
}

// IsGitRepo checks if the working directory is a git repository.
func (g *GitOperations) IsGitRepo(ctx context.Context) bool {
	cmd := exec.CommandContext(ctx, "git", "rev-parse", "--git-dir")
	cmd.Dir = g.WorkDir
	return cmd.Run() == nil
}

// HasChanges checks if there are any uncommitted changes.
// Returns true if there are staged, unstaged, or untracked changes.
func (g *GitOperations) HasChanges(ctx context.Context) (bool, error) {
	// Check for staged or unstaged changes
	cmd := exec.CommandContext(ctx, "git", "status", "--porcelain")
	cmd.Dir = g.WorkDir
	output, err := cmd.Output()
	if err != nil {
		return false, fmt.Errorf("git status failed: %w", err)
	}

	// If output is non-empty, there are changes
	return len(bytes.TrimSpace(output)) > 0, nil
}

// StageAll stages all changes (including untracked files).
func (g *GitOperations) StageAll(ctx context.Context) error {
	cmd := exec.CommandContext(ctx, "git", "add", "-A")
	cmd.Dir = g.WorkDir
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git add failed: %w", err)
	}
	return nil
}

// BuildCommitMessage creates a commit message for a task.
func (g *GitOperations) BuildCommitMessage(t *task.Task) string {
	prefix := g.Config.CommitPrefix
	if prefix == "" {
		prefix = config.DefaultCommitPrefix
	}

	// Format: "[ralph] TASK-001 - Task name" or "feat: TASK-001 - Task name"
	return fmt.Sprintf("%s %s - %s", prefix, t.ID, t.Name)
}

// Commit creates a commit with the given message.
func (g *GitOperations) Commit(ctx context.Context, message string) error {
	cmd := exec.CommandContext(ctx, "git", "commit", "-m", message)
	cmd.Dir = g.WorkDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		// Check if the error is because there's nothing to commit
		if strings.Contains(string(output), "nothing to commit") {
			return nil
		}
		return fmt.Errorf("git commit failed: %w: %s", err, string(output))
	}
	return nil
}

// Push pushes commits to the remote.
func (g *GitOperations) Push(ctx context.Context) error {
	cmd := exec.CommandContext(ctx, "git", "push")
	cmd.Dir = g.WorkDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git push failed: %w: %s", err, string(output))
	}
	return nil
}

// CommitTask stages, commits, and optionally pushes changes for a completed task.
// It respects the auto_commit and push configuration settings.
func (g *GitOperations) CommitTask(ctx context.Context, t *task.Task) CommitResult {
	result := CommitResult{}

	// Check if auto-commit is enabled
	if !g.Config.AutoCommit {
		return result
	}

	// Check if this is a git repo
	if !g.IsGitRepo(ctx) {
		return result
	}

	// Check for changes
	hasChanges, err := g.HasChanges(ctx)
	if err != nil {
		result.Error = err
		return result
	}
	if !hasChanges {
		return result
	}

	// Stage all changes
	if err := g.StageAll(ctx); err != nil {
		result.Error = err
		return result
	}

	// Build and execute commit
	result.Message = g.BuildCommitMessage(t)
	if err := g.Commit(ctx, result.Message); err != nil {
		result.Error = err
		return result
	}
	result.Committed = true

	// Push if configured
	if g.Config.Push {
		if err := g.Push(ctx); err != nil {
			result.Error = err
			return result
		}
		result.Pushed = true
	}

	return result
}
