package prompt

import (
	"fmt"
	"strings"

	"github.com/dbmrq/ralph/internal/build"
	"github.com/dbmrq/ralph/internal/task"
)

// TaskPromptBuilder builds complete prompts for task execution.
// It combines template layers, project analysis context, and task-specific content.
type TaskPromptBuilder struct {
	// templates is the loaded prompt templates
	templates *Prompt
	// analysis is the AI-driven project analysis
	analysis *build.ProjectAnalysis
	// docsContext is optional context from project documentation
	docsContext string
	// previousChanges is optional summary of recent changes
	previousChanges string
}

// NewTaskPromptBuilder creates a new TaskPromptBuilder with the given templates.
func NewTaskPromptBuilder(templates *Prompt) *TaskPromptBuilder {
	return &TaskPromptBuilder{
		templates: templates,
	}
}

// SetAnalysis sets the project analysis for context injection.
func (b *TaskPromptBuilder) SetAnalysis(analysis *build.ProjectAnalysis) *TaskPromptBuilder {
	b.analysis = analysis
	return b
}

// SetDocsContext sets additional context from project documentation.
func (b *TaskPromptBuilder) SetDocsContext(docs string) *TaskPromptBuilder {
	b.docsContext = docs
	return b
}

// SetPreviousChanges sets a summary of recent changes for context.
func (b *TaskPromptBuilder) SetPreviousChanges(changes string) *TaskPromptBuilder {
	b.previousChanges = changes
	return b
}

// BuildForTask constructs the full prompt for a task execution.
// It combines:
// 1. Base prompt templates (base + platform + project)
// 2. Project analysis context (build commands, project type, etc.)
// 3. Optional documentation context
// 4. Task-specific content
func (b *TaskPromptBuilder) BuildForTask(t *task.Task, vars *Variables, iteration int) string {
	var parts []string

	// 1. Build base prompt from templates
	baseBuilder := NewBuilder(b.templates)
	basePrompt := baseBuilder.Build(vars)
	if basePrompt != "" {
		parts = append(parts, basePrompt)
	}

	// 2. Add project analysis context
	analysisContext := b.buildAnalysisContext()
	if analysisContext != "" {
		parts = append(parts, "# Project Context (from analysis)\n\n"+analysisContext)
	}

	// 3. Add documentation context if available
	if b.docsContext != "" {
		parts = append(parts, "# Documentation Context\n\n"+b.docsContext)
	}

	// 4. Add previous changes if available
	if b.previousChanges != "" {
		parts = append(parts, "# Recent Changes\n\n"+b.previousChanges)
	}

	// 5. Add task content
	taskContent := b.buildTaskContent(t, iteration)
	if taskContent != "" {
		parts = append(parts, "# Current Task\n\n"+taskContent)
	}

	return strings.Join(parts, LevelSeparator)
}

// buildAnalysisContext creates the project analysis section for prompts.
func (b *TaskPromptBuilder) buildAnalysisContext() string {
	if b.analysis == nil {
		return "Project analysis not available."
	}

	var parts []string

	parts = append(parts, fmt.Sprintf("Project Type: %s", b.analysis.ProjectType))

	if len(b.analysis.Languages) > 0 {
		parts = append(parts, fmt.Sprintf("Languages: %s", strings.Join(b.analysis.Languages, ", ")))
	}

	if b.analysis.Build.Command != nil {
		parts = append(parts, fmt.Sprintf("Build Command: %s", *b.analysis.Build.Command))
	}

	if b.analysis.Test.Command != nil {
		parts = append(parts, fmt.Sprintf("Test Command: %s", *b.analysis.Test.Command))
	}

	if b.analysis.Lint.Available && b.analysis.Lint.Command != nil {
		parts = append(parts, fmt.Sprintf("Lint Command: %s", *b.analysis.Lint.Command))
	}

	if b.analysis.Dependencies.Manager != "" {
		installed := "No"
		if b.analysis.Dependencies.Installed {
			installed = "Yes"
		}
		parts = append(parts, fmt.Sprintf("Package Manager: %s (installed: %s)",
			b.analysis.Dependencies.Manager, installed))
	}

	if b.analysis.IsGreenfield {
		parts = append(parts, "Status: Greenfield project (no buildable code yet)")
	}

	if b.analysis.IsMonorepo {
		parts = append(parts, "Structure: Monorepo")
	}

	if b.analysis.ProjectContext != "" {
		parts = append(parts, fmt.Sprintf("\n%s", b.analysis.ProjectContext))
	}

	return strings.Join(parts, "\n")
}

// buildTaskContent creates the task-specific content for prompts.
func (b *TaskPromptBuilder) buildTaskContent(t *task.Task, iteration int) string {
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

// GetAnalysis returns the current project analysis, or nil if not set.
func (b *TaskPromptBuilder) GetAnalysis() *build.ProjectAnalysis {
	return b.analysis
}

// HasAnalysis returns true if project analysis is set.
func (b *TaskPromptBuilder) HasAnalysis() bool {
	return b.analysis != nil
}

// FormatAnalysisContext returns the formatted analysis context string.
// This is useful for inspecting what will be injected.
func (b *TaskPromptBuilder) FormatAnalysisContext() string {
	return b.buildAnalysisContext()
}

// FormatTaskContent returns the formatted task content string.
// This is useful for inspecting what will be injected.
func (b *TaskPromptBuilder) FormatTaskContent(t *task.Task, iteration int) string {
	return b.buildTaskContent(t, iteration)
}

// Clone creates a copy of the TaskPromptBuilder with the same settings.
// This allows creating variations without modifying the original.
func (b *TaskPromptBuilder) Clone() *TaskPromptBuilder {
	return &TaskPromptBuilder{
		templates:       b.templates,
		analysis:        b.analysis,
		docsContext:     b.docsContext,
		previousChanges: b.previousChanges,
	}
}
