package errors

import (
	"errors"
	"strings"
	"testing"
)

func TestBuildFailed(t *testing.T) {
	err := BuildFailed("go build ./...", 1, 3)

	if !errors.Is(err, ErrBuild) {
		t.Error("BuildFailed should return ErrBuild")
	}
	if err.Details["command"] != "go build ./..." {
		t.Error("Should include build command")
	}
	if err.Details["exit_code"] != "1" {
		t.Error("Should include exit code")
	}
	if err.Details["errors"] != "3 errors found" {
		t.Error("Should include error count")
	}
}

func TestBuildFailed_NoErrorCount(t *testing.T) {
	err := BuildFailed("npm run build", 1, 0)

	if _, ok := err.Details["errors"]; ok {
		t.Error("Should not include errors key when count is 0")
	}
}

func TestTestFailed(t *testing.T) {
	err := TestFailed("go test ./...", 3, 10)

	if !errors.Is(err, ErrTest) {
		t.Error("TestFailed should return ErrTest")
	}
	if !strings.Contains(err.Message, "3 of 10") {
		t.Error("Message should show failed/total")
	}
	if err.Details["failed"] != "3" {
		t.Error("Should include failed count")
	}
	if err.Details["total"] != "10" {
		t.Error("Should include total count")
	}
	if !strings.Contains(err.Details["passing_rate"], "70") {
		t.Error("Should calculate passing rate")
	}
}

func TestTestRegression(t *testing.T) {
	tests := []string{"TestAuth", "TestUser", "TestLogin"}
	err := TestRegression(tests)

	if !errors.Is(err, ErrTest) {
		t.Error("TestRegression should return ErrTest")
	}
	if !strings.Contains(err.Message, "3") {
		t.Error("Message should include regression count")
	}
	if !strings.Contains(err.Details["regressed_tests"], "TestAuth") {
		t.Error("Should list regressed tests")
	}
}

func TestTestRegression_ManyTests(t *testing.T) {
	tests := []string{"Test1", "Test2", "Test3", "Test4", "Test5", "Test6", "Test7"}
	err := TestRegression(tests)

	if !strings.Contains(err.Details["regressed_tests"], "and 2 more") {
		t.Error("Should truncate long test lists")
	}
}

func TestNoTestsFound(t *testing.T) {
	err := NoTestsFound("/project")

	if !errors.Is(err, ErrTest) {
		t.Error("NoTestsFound should return ErrTest")
	}
	if !strings.Contains(err.Message, "bootstrap") {
		t.Error("Message should mention bootstrap phase")
	}
}

func TestGitNotInitialized(t *testing.T) {
	err := GitNotInitialized("/project")

	if !errors.Is(err, ErrGit) {
		t.Error("GitNotInitialized should return ErrGit")
	}
	if !strings.Contains(err.Suggestion, "git init") {
		t.Error("Suggestion should mention git init")
	}
}

func TestGitDirtyState(t *testing.T) {
	err := GitDirtyState(5)

	if !errors.Is(err, ErrGit) {
		t.Error("GitDirtyState should return ErrGit")
	}
	if !strings.Contains(err.Message, "5 files") {
		t.Error("Message should include file count")
	}
	if !strings.Contains(err.Suggestion, "stash") {
		t.Error("Suggestion should mention stash option")
	}
}

func TestGitConflict(t *testing.T) {
	files := []string{"file1.go", "file2.go"}
	err := GitConflict(files)

	if !errors.Is(err, ErrGit) {
		t.Error("GitConflict should return ErrGit")
	}
	if !strings.Contains(err.Details["conflict_files"], "file1.go") {
		t.Error("Should list conflict files")
	}
}

func TestGitConflict_ManyFiles(t *testing.T) {
	files := []string{"f1", "f2", "f3", "f4", "f5", "f6", "f7"}
	err := GitConflict(files)

	if !strings.Contains(err.Details["conflict_files"], "and 2 more") {
		t.Error("Should truncate long file lists")
	}
}

func TestCommitFailed(t *testing.T) {
	err := CommitFailed("author identity unknown")

	if !errors.Is(err, ErrGit) {
		t.Error("CommitFailed should return ErrGit")
	}
	if err.Details["output"] != "author identity unknown" {
		t.Error("Should include stderr output")
	}
	if !strings.Contains(err.Suggestion, "git config") {
		t.Error("Suggestion should mention git config")
	}
}

func TestCommitFailed_EmptyStderr(t *testing.T) {
	err := CommitFailed("")

	if err.Details != nil {
		t.Error("Should not include empty details")
	}
}

