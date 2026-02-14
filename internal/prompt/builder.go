package prompt

import "strings"

// Separator between prompt levels.
const LevelSeparator = "\n\n---\n\n"

// Builder assembles prompts from templates.
type Builder struct {
	prompt *Prompt
}

// NewBuilder creates a new prompt builder with the given templates.
func NewBuilder(prompt *Prompt) *Builder {
	return &Builder{prompt: prompt}
}

// Build assembles the full prompt by combining all levels.
// The optional Variables are used for template variable substitution.
// Returns the combined prompt with levels separated by dividers.
func (b *Builder) Build(vars *Variables) string {
	if b.prompt == nil {
		return ""
	}

	var parts []string

	// Add base prompt (required)
	if b.prompt.Base != nil {
		content := b.prompt.Base.Content
		if vars != nil {
			content = SubstituteVariables(content, vars)
		}
		parts = append(parts, content)
	}

	// Add level separator and header comments between sections
	levelHeader := func(name string) string {
		return "# Level " + name + "\n\n"
	}

	// Add platform prompt (optional)
	if b.prompt.Platform != nil && b.prompt.Platform.Content != "" {
		content := b.prompt.Platform.Content
		if vars != nil {
			content = SubstituteVariables(content, vars)
		}
		parts = append(parts, levelHeader("2: Platform Guidelines")+content)
	}

	// Add project prompt (optional)
	if b.prompt.Project != nil && b.prompt.Project.Content != "" {
		content := b.prompt.Project.Content
		if vars != nil {
			content = SubstituteVariables(content, vars)
		}
		parts = append(parts, levelHeader("3: Project-Specific Instructions")+content)
	}

	return strings.Join(parts, LevelSeparator)
}

// BuildWithTask assembles the prompt and appends task-specific content.
// This is a convenience method for the common case of adding task details.
func (b *Builder) BuildWithTask(vars *Variables, taskContent string) string {
	prompt := b.Build(vars)
	if taskContent == "" {
		return prompt
	}

	// Add task section
	taskSection := LevelSeparator + "# Current Task\n\n" + taskContent
	if vars != nil {
		taskSection = SubstituteVariables(taskSection, vars)
	}

	return prompt + taskSection
}

// HasPlatformPrompt returns true if a platform prompt is loaded.
func (b *Builder) HasPlatformPrompt() bool {
	return b.prompt != nil && b.prompt.Platform != nil && b.prompt.Platform.Content != ""
}

// HasProjectPrompt returns true if a project prompt is loaded.
func (b *Builder) HasProjectPrompt() bool {
	return b.prompt != nil && b.prompt.Project != nil && b.prompt.Project.Content != ""
}

// GetBase returns the base template, or nil if not set.
func (b *Builder) GetBase() *Template {
	if b.prompt == nil {
		return nil
	}
	return b.prompt.Base
}

// GetPlatform returns the platform template, or nil if not set.
func (b *Builder) GetPlatform() *Template {
	if b.prompt == nil {
		return nil
	}
	return b.prompt.Platform
}

// GetProject returns the project template, or nil if not set.
func (b *Builder) GetProject() *Template {
	if b.prompt == nil {
		return nil
	}
	return b.prompt.Project
}

// CountLevels returns the number of prompt levels that are loaded.
func (b *Builder) CountLevels() int {
	if b.prompt == nil {
		return 0
	}
	count := 0
	if b.prompt.Base != nil && b.prompt.Base.Content != "" {
		count++
	}
	if b.prompt.Platform != nil && b.prompt.Platform.Content != "" {
		count++
	}
	if b.prompt.Project != nil && b.prompt.Project.Content != "" {
		count++
	}
	return count
}
