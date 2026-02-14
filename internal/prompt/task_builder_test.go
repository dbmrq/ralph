package prompt

import (
	"strings"
	"testing"

	"github.com/wexinc/ralph/internal/build"
	"github.com/wexinc/ralph/internal/task"
)

func TestNewTaskPromptBuilder(t *testing.T) {
	templates := &Prompt{Base: &Template{Content: "base"}}
	builder := NewTaskPromptBuilder(templates)

	if builder == nil {
		t.Fatal("NewTaskPromptBuilder returned nil")
	}
	if builder.templates != templates {
		t.Error("templates not set correctly")
	}
	if builder.analysis != nil {
		t.Error("analysis should be nil initially")
	}
}

func TestTaskPromptBuilder_SetAnalysis(t *testing.T) {
	builder := NewTaskPromptBuilder(nil)
	buildCmd := "go build ./..."
	analysis := &build.ProjectAnalysis{
		ProjectType: "go",
		Build: build.BuildAnalysis{
			Command: &buildCmd,
		},
	}

	result := builder.SetAnalysis(analysis)

	if result != builder {
		t.Error("SetAnalysis should return self for chaining")
	}
	if builder.analysis != analysis {
		t.Error("analysis not set correctly")
	}
	if !builder.HasAnalysis() {
		t.Error("HasAnalysis should return true")
	}
}

func TestTaskPromptBuilder_SetDocsContext(t *testing.T) {
	builder := NewTaskPromptBuilder(nil)
	result := builder.SetDocsContext("Some docs context")

	if result != builder {
		t.Error("SetDocsContext should return self for chaining")
	}
	if builder.docsContext != "Some docs context" {
		t.Error("docsContext not set correctly")
	}
}

func TestTaskPromptBuilder_SetPreviousChanges(t *testing.T) {
	builder := NewTaskPromptBuilder(nil)
	result := builder.SetPreviousChanges("Changed file.go")

	if result != builder {
		t.Error("SetPreviousChanges should return self for chaining")
	}
	if builder.previousChanges != "Changed file.go" {
		t.Error("previousChanges not set correctly")
	}
}

func TestTaskPromptBuilder_BuildForTask_AllParts(t *testing.T) {
	templates := &Prompt{
		Base:     &Template{Content: "Base instructions"},
		Platform: &Template{Content: "Platform guidelines"},
		Project:  &Template{Content: "Project rules"},
	}

	buildCmd := "go build ./..."
	testCmd := "go test ./..."
	analysis := &build.ProjectAnalysis{
		ProjectType:    "go",
		Languages:      []string{"go", "yaml"},
		Build:          build.BuildAnalysis{Command: &buildCmd, Ready: true},
		Test:           build.TestAnalysis{Command: &testCmd, Ready: true},
		ProjectContext: "A CLI application",
	}

	testTask := task.NewTask("TASK-001", "Do something", "Full description here")
	vars := &Variables{TaskID: "TASK-001", TaskName: "Do something"}

	builder := NewTaskPromptBuilder(templates).
		SetAnalysis(analysis).
		SetDocsContext("Reference: ARCHITECTURE.md").
		SetPreviousChanges("Modified main.go")

	result := builder.BuildForTask(testTask, vars, 1)

	// Check base templates
	if !strings.Contains(result, "Base instructions") {
		t.Error("missing base instructions")
	}
	if !strings.Contains(result, "Platform guidelines") {
		t.Error("missing platform guidelines")
	}
	if !strings.Contains(result, "Project rules") {
		t.Error("missing project rules")
	}

	// Check analysis context
	if !strings.Contains(result, "Project Type: go") {
		t.Error("missing project type")
	}
	if !strings.Contains(result, "Build Command: go build ./...") {
		t.Error("missing build command")
	}
	if !strings.Contains(result, "Test Command: go test ./...") {
		t.Error("missing test command")
	}
	if !strings.Contains(result, "Languages: go, yaml") {
		t.Error("missing languages")
	}

	// Check docs context
	if !strings.Contains(result, "# Documentation Context") {
		t.Error("missing docs header")
	}
	if !strings.Contains(result, "Reference: ARCHITECTURE.md") {
		t.Error("missing docs content")
	}

	// Check previous changes
	if !strings.Contains(result, "# Recent Changes") {
		t.Error("missing changes header")
	}
	if !strings.Contains(result, "Modified main.go") {
		t.Error("missing changes content")
	}

	// Check task content
	if !strings.Contains(result, "# Current Task") {
		t.Error("missing task header")
	}
	if !strings.Contains(result, "**Task ID:** TASK-001") {
		t.Error("missing task ID")
	}
	if !strings.Contains(result, "**Task:** Do something") {
		t.Error("missing task name")
	}
	if !strings.Contains(result, "**Description:**") {
		t.Error("missing description label")
	}
}

func TestTaskPromptBuilder_BuildForTask_MinimalParts(t *testing.T) {
	templates := &Prompt{
		Base: &Template{Content: "Base only"},
	}

	testTask := task.NewTask("T1", "Task", "")

	builder := NewTaskPromptBuilder(templates)
	result := builder.BuildForTask(testTask, nil, 1)

	if !strings.Contains(result, "Base only") {
		t.Error("missing base content")
	}
	if !strings.Contains(result, "Project analysis not available") {
		t.Error("missing analysis fallback message")
	}
	if !strings.Contains(result, "**Task ID:** T1") {
		t.Error("missing task ID")
	}
	// Should not have empty sections
	if strings.Contains(result, "# Documentation Context") {
		t.Error("should not have empty docs section")
	}
	if strings.Contains(result, "# Recent Changes") {
		t.Error("should not have empty changes section")
	}
}

func TestTaskPromptBuilder_BuildForTask_WithIteration(t *testing.T) {
	templates := &Prompt{Base: &Template{Content: "Base"}}
	testTask := task.NewTask("T1", "Task", "Description")

	// Start and end an iteration to simulate retry
	testTask.StartIteration()
	testTask.EndIteration("NEXT", "output", "")

	builder := NewTaskPromptBuilder(templates)
	result := builder.BuildForTask(testTask, nil, 2)

	if !strings.Contains(result, "**Iteration:** 2") {
		t.Error("missing iteration number")
	}
	if !strings.Contains(result, "previous attempt did not complete") {
		t.Error("missing retry context")
	}
	if !strings.Contains(result, "**Previous result:** NEXT") {
		t.Error("missing previous result")
	}
}

func TestTaskPromptBuilder_BuildAnalysisContext(t *testing.T) {
	t.Run("nil analysis", func(t *testing.T) {
		builder := NewTaskPromptBuilder(nil)
		result := builder.FormatAnalysisContext()

		if result != "Project analysis not available." {
			t.Errorf("unexpected: %s", result)
		}
	})

	t.Run("greenfield project", func(t *testing.T) {
		analysis := &build.ProjectAnalysis{
			ProjectType:  "go",
			IsGreenfield: true,
		}
		builder := NewTaskPromptBuilder(nil).SetAnalysis(analysis)
		result := builder.FormatAnalysisContext()

		if !strings.Contains(result, "Greenfield project") {
			t.Error("missing greenfield indicator")
		}
	})

	t.Run("monorepo", func(t *testing.T) {
		analysis := &build.ProjectAnalysis{
			ProjectType: "node",
			IsMonorepo:  true,
		}
		builder := NewTaskPromptBuilder(nil).SetAnalysis(analysis)
		result := builder.FormatAnalysisContext()

		if !strings.Contains(result, "Structure: Monorepo") {
			t.Error("missing monorepo indicator")
		}
	})

	t.Run("with lint command", func(t *testing.T) {
		lintCmd := "golangci-lint run"
		analysis := &build.ProjectAnalysis{
			ProjectType: "go",
			Lint: build.LintAnalysis{
				Command:   &lintCmd,
				Available: true,
			},
		}
		builder := NewTaskPromptBuilder(nil).SetAnalysis(analysis)
		result := builder.FormatAnalysisContext()

		if !strings.Contains(result, "Lint Command: golangci-lint run") {
			t.Error("missing lint command")
		}
	})

	t.Run("dependencies info", func(t *testing.T) {
		analysis := &build.ProjectAnalysis{
			ProjectType: "go",
			Dependencies: build.DependencyAnalysis{
				Manager:   "go mod",
				Installed: true,
			},
		}
		builder := NewTaskPromptBuilder(nil).SetAnalysis(analysis)
		result := builder.FormatAnalysisContext()

		if !strings.Contains(result, "Package Manager: go mod (installed: Yes)") {
			t.Error("missing dependency info")
		}
	})
}

func TestTaskPromptBuilder_Clone(t *testing.T) {
	buildCmd := "make"
	analysis := &build.ProjectAnalysis{ProjectType: "c"}

	original := NewTaskPromptBuilder(&Prompt{Base: &Template{Content: "base"}}).
		SetAnalysis(analysis).
		SetDocsContext("docs").
		SetPreviousChanges("changes")

	cloned := original.Clone()

	if cloned == original {
		t.Error("Clone should return new instance")
	}
	if cloned.templates != original.templates {
		t.Error("templates should be same reference")
	}
	if cloned.analysis != original.analysis {
		t.Error("analysis should be same reference")
	}
	if cloned.docsContext != original.docsContext {
		t.Error("docsContext should match")
	}
	if cloned.previousChanges != original.previousChanges {
		t.Error("previousChanges should match")
	}

	// Modify clone should not affect original
	newCmd := "ninja"
	cloned.analysis = &build.ProjectAnalysis{Build: build.BuildAnalysis{Command: &newCmd}}
	if original.analysis.Build.Command != nil && *original.analysis.Build.Command == "ninja" {
		t.Error("modifying clone should not affect original")
	}

	_ = buildCmd // use the variable
}

func TestTaskPromptBuilder_FormatTaskContent(t *testing.T) {
	builder := NewTaskPromptBuilder(nil)

	testTask := task.NewTask("TASK-123", "Implement feature", "Detailed description")
	result := builder.FormatTaskContent(testTask, 1)

	if !strings.Contains(result, "**Task ID:** TASK-123") {
		t.Error("missing task ID")
	}
	if !strings.Contains(result, "**Task:** Implement feature") {
		t.Error("missing task name")
	}
	if !strings.Contains(result, "Detailed description") {
		t.Error("missing description")
	}
}
