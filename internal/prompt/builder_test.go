package prompt

import (
	"strings"
	"testing"
)

func TestNewBuilder(t *testing.T) {
	prompt := &Prompt{Base: &Template{Content: "test"}}
	builder := NewBuilder(prompt)
	if builder == nil {
		t.Fatal("NewBuilder() returned nil")
	}
	if builder.prompt != prompt {
		t.Error("NewBuilder() did not set prompt correctly")
	}
}

func TestBuilder_Build(t *testing.T) {
	t.Run("all levels", func(t *testing.T) {
		prompt := &Prompt{
			Base:     &Template{Content: "Base content", Level: LevelBase},
			Platform: &Template{Content: "Platform content", Level: LevelPlatform},
			Project:  &Template{Content: "Project content", Level: LevelProject},
		}
		builder := NewBuilder(prompt)
		result := builder.Build(nil)

		if !strings.Contains(result, "Base content") {
			t.Error("Build() missing base content")
		}
		if !strings.Contains(result, "Platform content") {
			t.Error("Build() missing platform content")
		}
		if !strings.Contains(result, "Project content") {
			t.Error("Build() missing project content")
		}
		if !strings.Contains(result, "Level 2: Platform Guidelines") {
			t.Error("Build() missing platform header")
		}
		if !strings.Contains(result, "Level 3: Project-Specific Instructions") {
			t.Error("Build() missing project header")
		}
	})

	t.Run("base only", func(t *testing.T) {
		prompt := &Prompt{
			Base: &Template{Content: "Base only", Level: LevelBase},
		}
		builder := NewBuilder(prompt)
		result := builder.Build(nil)

		if result != "Base only" {
			t.Errorf("Build() = %q, want %q", result, "Base only")
		}
	})

	t.Run("nil prompt", func(t *testing.T) {
		builder := NewBuilder(nil)
		result := builder.Build(nil)
		if result != "" {
			t.Errorf("Build() with nil prompt = %q, want empty", result)
		}
	})

	t.Run("with variable substitution", func(t *testing.T) {
		prompt := &Prompt{
			Base: &Template{Content: "Task: ${TASK_ID}", Level: LevelBase},
		}
		vars := &Variables{TaskID: "TASK-001"}
		builder := NewBuilder(prompt)
		result := builder.Build(vars)

		if result != "Task: TASK-001" {
			t.Errorf("Build() = %q, want %q", result, "Task: TASK-001")
		}
	})

	t.Run("empty templates skipped", func(t *testing.T) {
		prompt := &Prompt{
			Base:     &Template{Content: "Base", Level: LevelBase},
			Platform: &Template{Content: "", Level: LevelPlatform}, // Empty
			Project:  &Template{Content: "Project", Level: LevelProject},
		}
		builder := NewBuilder(prompt)
		result := builder.Build(nil)

		if strings.Contains(result, "Level 2: Platform Guidelines") {
			t.Error("Build() should skip empty platform template")
		}
		if !strings.Contains(result, "Level 3: Project-Specific Instructions") {
			t.Error("Build() should include non-empty project")
		}
	})
}

func TestBuilder_BuildWithTask(t *testing.T) {
	prompt := &Prompt{
		Base: &Template{Content: "Base", Level: LevelBase},
	}
	builder := NewBuilder(prompt)

	t.Run("with task content", func(t *testing.T) {
		result := builder.BuildWithTask(nil, "Do this task")
		if !strings.Contains(result, "Base") {
			t.Error("BuildWithTask() missing base")
		}
		if !strings.Contains(result, "# Current Task") {
			t.Error("BuildWithTask() missing task header")
		}
		if !strings.Contains(result, "Do this task") {
			t.Error("BuildWithTask() missing task content")
		}
	})

	t.Run("empty task content", func(t *testing.T) {
		result := builder.BuildWithTask(nil, "")
		if result != "Base" {
			t.Errorf("BuildWithTask() = %q, want %q", result, "Base")
		}
	})

	t.Run("with substitution in task", func(t *testing.T) {
		vars := &Variables{TaskID: "T1"}
		result := builder.BuildWithTask(vars, "Work on ${TASK_ID}")
		if !strings.Contains(result, "Work on T1") {
			t.Error("BuildWithTask() did not substitute task content")
		}
	})
}

func TestBuilder_HasPlatformPrompt(t *testing.T) {
	tests := []struct {
		name     string
		prompt   *Prompt
		expected bool
	}{
		{"nil prompt", nil, false},
		{"nil platform", &Prompt{Base: &Template{Content: "b"}}, false},
		{"empty platform", &Prompt{Platform: &Template{Content: ""}}, false},
		{"has platform", &Prompt{Platform: &Template{Content: "p"}}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			builder := NewBuilder(tt.prompt)
			if got := builder.HasPlatformPrompt(); got != tt.expected {
				t.Errorf("HasPlatformPrompt() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestBuilder_HasProjectPrompt(t *testing.T) {
	tests := []struct {
		name     string
		prompt   *Prompt
		expected bool
	}{
		{"nil prompt", nil, false},
		{"nil project", &Prompt{Base: &Template{Content: "b"}}, false},
		{"empty project", &Prompt{Project: &Template{Content: ""}}, false},
		{"has project", &Prompt{Project: &Template{Content: "p"}}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			builder := NewBuilder(tt.prompt)
			if got := builder.HasProjectPrompt(); got != tt.expected {
				t.Errorf("HasProjectPrompt() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestBuilder_GetBase(t *testing.T) {
	base := &Template{Content: "base"}
	prompt := &Prompt{Base: base}
	builder := NewBuilder(prompt)

	if got := builder.GetBase(); got != base {
		t.Errorf("GetBase() = %v, want %v", got, base)
	}

	nilBuilder := NewBuilder(nil)
	if got := nilBuilder.GetBase(); got != nil {
		t.Errorf("GetBase() with nil prompt = %v, want nil", got)
	}
}

func TestBuilder_CountLevels(t *testing.T) {
	tests := []struct {
		name     string
		prompt   *Prompt
		expected int
	}{
		{"nil prompt", nil, 0},
		{"base only", &Prompt{Base: &Template{Content: "b"}}, 1},
		{"base and platform", &Prompt{
			Base:     &Template{Content: "b"},
			Platform: &Template{Content: "p"},
		}, 2},
		{"all three", &Prompt{
			Base:     &Template{Content: "b"},
			Platform: &Template{Content: "p"},
			Project:  &Template{Content: "r"},
		}, 3},
		{"empty templates not counted", &Prompt{
			Base:     &Template{Content: "b"},
			Platform: &Template{Content: ""},
			Project:  &Template{Content: "r"},
		}, 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			builder := NewBuilder(tt.prompt)
			if got := builder.CountLevels(); got != tt.expected {
				t.Errorf("CountLevels() = %d, want %d", got, tt.expected)
			}
		})
	}
}
