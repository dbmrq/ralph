package loop

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/wexinc/ralph/internal/config"
	"github.com/wexinc/ralph/internal/task"
)

// setupTestRepo creates a temporary git repository for testing.
func setupTestRepo(t *testing.T) (string, func()) {
	t.Helper()

	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "git-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}

	// Initialize git repo
	cmd := exec.Command("git", "init")
	cmd.Dir = tmpDir
	if err := cmd.Run(); err != nil {
		os.RemoveAll(tmpDir)
		t.Fatalf("failed to init git repo: %v", err)
	}

	// Configure git for commits
	cmd = exec.Command("git", "config", "user.email", "test@example.com")
	cmd.Dir = tmpDir
	cmd.Run()

	cmd = exec.Command("git", "config", "user.name", "Test User")
	cmd.Dir = tmpDir
	cmd.Run()

	// Create initial commit so we have a valid HEAD
	readmePath := filepath.Join(tmpDir, "README.md")
	if err := os.WriteFile(readmePath, []byte("# Test"), 0644); err != nil {
		os.RemoveAll(tmpDir)
		t.Fatalf("failed to create README: %v", err)
	}

	cmd = exec.Command("git", "add", "-A")
	cmd.Dir = tmpDir
	cmd.Run()

	cmd = exec.Command("git", "commit", "-m", "Initial commit")
	cmd.Dir = tmpDir
	cmd.Run()

	cleanup := func() {
		os.RemoveAll(tmpDir)
	}

	return tmpDir, cleanup
}

func TestGitOperations_IsGitRepo(t *testing.T) {
	repoDir, cleanup := setupTestRepo(t)
	defer cleanup()

	nonRepoDir, err := os.MkdirTemp("", "non-git-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(nonRepoDir)

	ctx := context.Background()

	tests := []struct {
		name     string
		workDir  string
		expected bool
	}{
		{
			name:     "is a git repo",
			workDir:  repoDir,
			expected: true,
		},
		{
			name:     "not a git repo",
			workDir:  nonRepoDir,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := NewGitOperations(tt.workDir, config.GitConfig{})
			result := g.IsGitRepo(ctx)
			if result != tt.expected {
				t.Errorf("IsGitRepo() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestGitOperations_HasChanges(t *testing.T) {
	repoDir, cleanup := setupTestRepo(t)
	defer cleanup()

	ctx := context.Background()
	g := NewGitOperations(repoDir, config.GitConfig{})

	// Initially no changes
	hasChanges, err := g.HasChanges(ctx)
	if err != nil {
		t.Fatalf("HasChanges() error = %v", err)
	}
	if hasChanges {
		t.Error("HasChanges() = true, want false (no changes after initial commit)")
	}

	// Create a new file
	testFile := filepath.Join(repoDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("test content"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	// Now there should be changes
	hasChanges, err = g.HasChanges(ctx)
	if err != nil {
		t.Fatalf("HasChanges() error = %v", err)
	}
	if !hasChanges {
		t.Error("HasChanges() = false, want true (untracked file exists)")
	}
}

func TestGitOperations_StageAll(t *testing.T) {
	repoDir, cleanup := setupTestRepo(t)
	defer cleanup()

	ctx := context.Background()
	g := NewGitOperations(repoDir, config.GitConfig{})

	// Create a new file
	testFile := filepath.Join(repoDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("test content"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	// Stage all
	if err := g.StageAll(ctx); err != nil {
		t.Fatalf("StageAll() error = %v", err)
	}

	// Verify file is staged
	cmd := exec.Command("git", "diff", "--cached", "--name-only")
	cmd.Dir = repoDir
	output, _ := cmd.Output()
	if len(output) == 0 {
		t.Error("StageAll() did not stage the file")
	}
}

func TestGitOperations_BuildCommitMessage(t *testing.T) {
	tests := []struct {
		name     string
		config   config.GitConfig
		task     *task.Task
		expected string
	}{
		{
			name:   "default prefix",
			config: config.GitConfig{CommitPrefix: ""},
			task: &task.Task{
				ID:   "TASK-001",
				Name: "Implement feature",
			},
			expected: "[ralph] TASK-001 - Implement feature",
		},
		{
			name:   "custom prefix",
			config: config.GitConfig{CommitPrefix: "feat:"},
			task: &task.Task{
				ID:   "BUILD-005",
				Name: "Create verification gate logic",
			},
			expected: "feat: BUILD-005 - Create verification gate logic",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := NewGitOperations("/tmp", tt.config)
			result := g.BuildCommitMessage(tt.task)
			if result != tt.expected {
				t.Errorf("BuildCommitMessage() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestGitOperations_Commit(t *testing.T) {
	repoDir, cleanup := setupTestRepo(t)
	defer cleanup()

	ctx := context.Background()
	g := NewGitOperations(repoDir, config.GitConfig{})

	// Create and stage a file
	testFile := filepath.Join(repoDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("test content"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}
	if err := g.StageAll(ctx); err != nil {
		t.Fatalf("StageAll() error = %v", err)
	}

	// Commit
	message := "[ralph] TASK-001 - Test task"
	if err := g.Commit(ctx, message); err != nil {
		t.Fatalf("Commit() error = %v", err)
	}

	// Verify commit was made
	cmd := exec.Command("git", "log", "-1", "--pretty=%s")
	cmd.Dir = repoDir
	output, err := cmd.Output()
	if err != nil {
		t.Fatalf("git log failed: %v", err)
	}

	commitMsg := string(output)
	if commitMsg[:len(commitMsg)-1] != message { // strip newline
		t.Errorf("Commit message = %q, want %q", commitMsg, message)
	}
}

func TestGitOperations_CommitTask(t *testing.T) {
	t.Run("auto-commit disabled", func(t *testing.T) {
		g := NewGitOperations("/tmp", config.GitConfig{AutoCommit: false})
		testTask := task.NewTask("TASK-001", "Test", "Description")

		result := g.CommitTask(context.Background(), testTask)

		if result.Committed {
			t.Error("CommitTask() committed when auto-commit is disabled")
		}
		if result.Error != nil {
			t.Errorf("CommitTask() error = %v, want nil", result.Error)
		}
	})

	t.Run("not a git repo", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "non-git-*")
		if err != nil {
			t.Fatalf("failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tmpDir)

		g := NewGitOperations(tmpDir, config.GitConfig{AutoCommit: true})
		testTask := task.NewTask("TASK-001", "Test", "Description")

		result := g.CommitTask(context.Background(), testTask)

		if result.Committed {
			t.Error("CommitTask() committed in non-git repo")
		}
	})

	t.Run("no changes to commit", func(t *testing.T) {
		repoDir, cleanup := setupTestRepo(t)
		defer cleanup()

		g := NewGitOperations(repoDir, config.GitConfig{AutoCommit: true})
		testTask := task.NewTask("TASK-001", "Test", "Description")

		result := g.CommitTask(context.Background(), testTask)

		if result.Committed {
			t.Error("CommitTask() committed when there are no changes")
		}
		if result.Error != nil {
			t.Errorf("CommitTask() error = %v, want nil", result.Error)
		}
	})

	t.Run("successful commit", func(t *testing.T) {
		repoDir, cleanup := setupTestRepo(t)
		defer cleanup()

		// Create a change
		testFile := filepath.Join(repoDir, "feature.go")
		if err := os.WriteFile(testFile, []byte("package main"), 0644); err != nil {
			t.Fatalf("failed to create test file: %v", err)
		}

		g := NewGitOperations(repoDir, config.GitConfig{
			AutoCommit:   true,
			CommitPrefix: "[ralph]",
		})
		testTask := task.NewTask("LOOP-003", "Add git support", "Description")

		result := g.CommitTask(context.Background(), testTask)

		if !result.Committed {
			t.Error("CommitTask() did not commit")
		}
		if result.Error != nil {
			t.Errorf("CommitTask() error = %v, want nil", result.Error)
		}
		if result.Message != "[ralph] LOOP-003 - Add git support" {
			t.Errorf("CommitTask() message = %q, want %q", result.Message, "[ralph] LOOP-003 - Add git support")
		}

		// Verify commit exists
		cmd := exec.Command("git", "log", "-1", "--pretty=%s")
		cmd.Dir = repoDir
		output, _ := cmd.Output()
		if string(output[:len(output)-1]) != result.Message {
			t.Errorf("Git log shows %q, want %q", string(output), result.Message)
		}
	})
}
