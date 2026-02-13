package prompt

import (
	"os"
	"path/filepath"
)

// Loader handles loading prompt templates from the filesystem.
type Loader struct {
	// templateDir is the directory containing prompt templates.
	templateDir string
}

// NewLoader creates a new prompt template loader.
// If templateDir is empty, it defaults to DefaultTemplateDir.
func NewLoader(templateDir string) *Loader {
	if templateDir == "" {
		templateDir = DefaultTemplateDir
	}
	return &Loader{templateDir: templateDir}
}

// LoadTemplate loads a single template file.
// Returns nil (not an error) if the file doesn't exist.
func (l *Loader) LoadTemplate(filename string, level PromptLevel) (*Template, error) {
	path := filepath.Join(l.templateDir, filename)

	content, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, &LoadError{
			Path:    path,
			Message: "failed to read file",
			Err:     err,
		}
	}

	return &Template{
		Level:   level,
		Content: string(content),
		Path:    path,
	}, nil
}

// Load loads all prompt templates from the template directory.
// The base prompt is required; platform and project prompts are optional.
func (l *Loader) Load() (*Prompt, error) {
	prompt := &Prompt{}
	var err error

	// Load base prompt (required)
	prompt.Base, err = l.LoadTemplate(BasePromptFile, LevelBase)
	if err != nil {
		return nil, err
	}
	if prompt.Base == nil {
		return nil, &LoadError{
			Path:    filepath.Join(l.templateDir, BasePromptFile),
			Message: "base prompt not found (required)",
		}
	}

	// Load platform prompt (optional)
	prompt.Platform, err = l.LoadTemplate(PlatformPromptFile, LevelPlatform)
	if err != nil {
		return nil, err
	}

	// Load project prompt (optional)
	prompt.Project, err = l.LoadTemplate(ProjectPromptFile, LevelProject)
	if err != nil {
		return nil, err
	}

	return prompt, nil
}

// LoadFromDir loads prompt templates from a specific directory.
// This is a convenience method that creates a new loader and loads templates.
func LoadFromDir(dir string) (*Prompt, error) {
	return NewLoader(dir).Load()
}

// LoadWithFallback loads templates from the primary directory,
// falling back to the fallback directory for missing files.
func LoadWithFallback(primaryDir, fallbackDir string) (*Prompt, error) {
	primary := NewLoader(primaryDir)
	fallback := NewLoader(fallbackDir)

	prompt := &Prompt{}
	var err error

	// Load base prompt with fallback
	prompt.Base, err = primary.LoadTemplate(BasePromptFile, LevelBase)
	if err != nil {
		return nil, err
	}
	if prompt.Base == nil {
		prompt.Base, err = fallback.LoadTemplate(BasePromptFile, LevelBase)
		if err != nil {
			return nil, err
		}
	}
	if prompt.Base == nil {
		return nil, &LoadError{
			Path:    filepath.Join(primaryDir, BasePromptFile),
			Message: "base prompt not found (required)",
		}
	}

	// Load platform prompt with fallback
	prompt.Platform, err = primary.LoadTemplate(PlatformPromptFile, LevelPlatform)
	if err != nil {
		return nil, err
	}
	if prompt.Platform == nil {
		prompt.Platform, err = fallback.LoadTemplate(PlatformPromptFile, LevelPlatform)
		if err != nil {
			return nil, err
		}
	}

	// Load project prompt with fallback
	prompt.Project, err = primary.LoadTemplate(ProjectPromptFile, LevelProject)
	if err != nil {
		return nil, err
	}
	if prompt.Project == nil {
		prompt.Project, err = fallback.LoadTemplate(ProjectPromptFile, LevelProject)
		if err != nil {
			return nil, err
		}
	}

	return prompt, nil
}

