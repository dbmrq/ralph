// Package prompt provides prompt template loading and building for ralph.
// It implements the 3-level prompt system: base_prompt.txt, platform_prompt.txt, project_prompt.txt.
package prompt

import (
	"fmt"
	"regexp"
)

// Default file names for prompt templates.
const (
	BasePromptFile     = "base_prompt.txt"
	PlatformPromptFile = "platform_prompt.txt"
	ProjectPromptFile  = "project_prompt.txt"
)

// Default template directory relative to project root.
const DefaultTemplateDir = ".ralph"

// PromptLevel represents the level of a prompt template.
type PromptLevel int

const (
	// LevelBase is the base agent instructions (universal).
	LevelBase PromptLevel = iota
	// LevelPlatform is platform-specific guidelines (Go, Node, Python, etc.).
	LevelPlatform
	// LevelProject is project-specific instructions.
	LevelProject
)

func (l PromptLevel) String() string {
	switch l {
	case LevelBase:
		return "base"
	case LevelPlatform:
		return "platform"
	case LevelProject:
		return "project"
	default:
		return "unknown"
	}
}

// Template represents a loaded prompt template.
type Template struct {
	// Level indicates which level this template is (base, platform, project).
	Level PromptLevel
	// Content is the raw template content before variable substitution.
	Content string
	// Path is the file path this template was loaded from.
	Path string
}

// Prompt represents the assembled prompt with all levels combined.
type Prompt struct {
	// Base is the base agent instructions.
	Base *Template
	// Platform is the platform-specific guidelines (optional).
	Platform *Template
	// Project is the project-specific instructions (optional).
	Project *Template
}

// Variables represents the available template variables for substitution.
// These are commonly used across hooks and prompts.
type Variables struct {
	// TaskID is the current task identifier.
	TaskID string
	// TaskName is the task name.
	TaskName string
	// TaskDescription is the task description.
	TaskDescription string
	// TaskStatus is the result status (pending, completed, failed, etc.).
	TaskStatus string
	// Iteration is the current iteration number.
	Iteration int
	// ProjectDir is the project root directory.
	ProjectDir string
	// AgentName is the name of the agent being used.
	AgentName string
	// Model is the model being used.
	Model string
	// SessionID is the current session identifier.
	SessionID string
	// Custom allows additional custom variables.
	Custom map[string]string
}

// ToMap converts Variables to a map for substitution.
func (v *Variables) ToMap() map[string]string {
	m := map[string]string{
		"TASK_ID":          v.TaskID,
		"TASK_NAME":        v.TaskName,
		"TASK_DESCRIPTION": v.TaskDescription,
		"TASK_STATUS":      v.TaskStatus,
		"ITERATION":        fmt.Sprintf("%d", v.Iteration),
		"PROJECT_DIR":      v.ProjectDir,
		"AGENT_NAME":       v.AgentName,
		"MODEL":            v.Model,
		"SESSION_ID":       v.SessionID,
	}
	// Add custom variables
	for k, val := range v.Custom {
		m[k] = val
	}
	return m
}

// LoadError represents an error that occurred while loading a template.
type LoadError struct {
	Path    string
	Message string
	Err     error
}

func (e *LoadError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("prompt template %s: %s: %v", e.Path, e.Message, e.Err)
	}
	return fmt.Sprintf("prompt template %s: %s", e.Path, e.Message)
}

func (e *LoadError) Unwrap() error {
	return e.Err
}

// variablePattern matches ${VARIABLE_NAME} patterns.
var variablePattern = regexp.MustCompile(`\$\{([A-Z_][A-Z0-9_]*)\}`)

// Substitute replaces template variables in the given content with values from vars.
// Variables are in the format ${VARIABLE_NAME}.
// Unknown variables are left unchanged.
func Substitute(content string, vars map[string]string) string {
	return variablePattern.ReplaceAllStringFunc(content, func(match string) string {
		// Extract variable name from ${NAME}
		name := match[2 : len(match)-1]
		if val, ok := vars[name]; ok {
			return val
		}
		// Leave unknown variables unchanged
		return match
	})
}

// SubstituteVariables replaces template variables using a Variables struct.
func SubstituteVariables(content string, vars *Variables) string {
	if vars == nil {
		return content
	}
	return Substitute(content, vars.ToMap())
}

