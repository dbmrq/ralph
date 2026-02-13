package prompt

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewLoader(t *testing.T) {
	t.Run("default dir", func(t *testing.T) {
		l := NewLoader("")
		if l.templateDir != DefaultTemplateDir {
			t.Errorf("NewLoader(\"\").templateDir = %q, want %q", l.templateDir, DefaultTemplateDir)
		}
	})

	t.Run("custom dir", func(t *testing.T) {
		l := NewLoader("/custom/dir")
		if l.templateDir != "/custom/dir" {
			t.Errorf("NewLoader().templateDir = %q, want %q", l.templateDir, "/custom/dir")
		}
	})
}

func TestLoader_LoadTemplate(t *testing.T) {
	// Create temp directory with test files
	tmpDir := t.TempDir()

	// Create a test template file
	testContent := "# Test Template\nHello ${NAME}!"
	testFile := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	loader := NewLoader(tmpDir)

	t.Run("existing file", func(t *testing.T) {
		tmpl, err := loader.LoadTemplate("test.txt", LevelBase)
		if err != nil {
			t.Fatalf("LoadTemplate() error = %v", err)
		}
		if tmpl == nil {
			t.Fatal("LoadTemplate() returned nil")
		}
		if tmpl.Content != testContent {
			t.Errorf("LoadTemplate().Content = %q, want %q", tmpl.Content, testContent)
		}
		if tmpl.Level != LevelBase {
			t.Errorf("LoadTemplate().Level = %v, want %v", tmpl.Level, LevelBase)
		}
		if tmpl.Path != testFile {
			t.Errorf("LoadTemplate().Path = %q, want %q", tmpl.Path, testFile)
		}
	})

	t.Run("non-existent file", func(t *testing.T) {
		tmpl, err := loader.LoadTemplate("nonexistent.txt", LevelBase)
		if err != nil {
			t.Errorf("LoadTemplate() error = %v, want nil", err)
		}
		if tmpl != nil {
			t.Errorf("LoadTemplate() = %v, want nil", tmpl)
		}
	})
}

func TestLoader_Load(t *testing.T) {
	t.Run("all files present", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create all three template files
		files := map[string]string{
			BasePromptFile:     "Base content",
			PlatformPromptFile: "Platform content",
			ProjectPromptFile:  "Project content",
		}
		for name, content := range files {
			if err := os.WriteFile(filepath.Join(tmpDir, name), []byte(content), 0644); err != nil {
				t.Fatalf("Failed to create %s: %v", name, err)
			}
		}

		loader := NewLoader(tmpDir)
		prompt, err := loader.Load()
		if err != nil {
			t.Fatalf("Load() error = %v", err)
		}

		if prompt.Base == nil || prompt.Base.Content != "Base content" {
			t.Error("Load() Base not loaded correctly")
		}
		if prompt.Platform == nil || prompt.Platform.Content != "Platform content" {
			t.Error("Load() Platform not loaded correctly")
		}
		if prompt.Project == nil || prompt.Project.Content != "Project content" {
			t.Error("Load() Project not loaded correctly")
		}
	})

	t.Run("only base present", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create only base file
		if err := os.WriteFile(filepath.Join(tmpDir, BasePromptFile), []byte("Base only"), 0644); err != nil {
			t.Fatalf("Failed to create base file: %v", err)
		}

		loader := NewLoader(tmpDir)
		prompt, err := loader.Load()
		if err != nil {
			t.Fatalf("Load() error = %v", err)
		}

		if prompt.Base == nil {
			t.Error("Load() Base is nil")
		}
		if prompt.Platform != nil {
			t.Error("Load() Platform should be nil")
		}
		if prompt.Project != nil {
			t.Error("Load() Project should be nil")
		}
	})

	t.Run("missing base returns error", func(t *testing.T) {
		tmpDir := t.TempDir()

		loader := NewLoader(tmpDir)
		_, err := loader.Load()
		if err == nil {
			t.Error("Load() expected error for missing base, got nil")
		}

		loadErr, ok := err.(*LoadError)
		if !ok {
			t.Errorf("Load() error type = %T, want *LoadError", err)
		} else if loadErr.Message != "base prompt not found (required)" {
			t.Errorf("LoadError.Message = %q, want base prompt not found", loadErr.Message)
		}
	})
}

func TestLoadWithFallback(t *testing.T) {
	t.Run("primary takes precedence", func(t *testing.T) {
		primaryDir := t.TempDir()
		fallbackDir := t.TempDir()

		// Create files in both dirs
		os.WriteFile(filepath.Join(primaryDir, BasePromptFile), []byte("Primary base"), 0644)
		os.WriteFile(filepath.Join(fallbackDir, BasePromptFile), []byte("Fallback base"), 0644)

		prompt, err := LoadWithFallback(primaryDir, fallbackDir)
		if err != nil {
			t.Fatalf("LoadWithFallback() error = %v", err)
		}
		if prompt.Base.Content != "Primary base" {
			t.Errorf("Expected primary to take precedence, got %q", prompt.Base.Content)
		}
	})

	t.Run("fallback used when missing", func(t *testing.T) {
		primaryDir := t.TempDir()
		fallbackDir := t.TempDir()

		// Create only fallback base
		os.WriteFile(filepath.Join(fallbackDir, BasePromptFile), []byte("Fallback base"), 0644)
		os.WriteFile(filepath.Join(fallbackDir, PlatformPromptFile), []byte("Fallback plat"), 0644)

		prompt, err := LoadWithFallback(primaryDir, fallbackDir)
		if err != nil {
			t.Fatalf("LoadWithFallback() error = %v", err)
		}
		if prompt.Base.Content != "Fallback base" {
			t.Errorf("Base = %q, want fallback", prompt.Base.Content)
		}
		if prompt.Platform.Content != "Fallback plat" {
			t.Errorf("Platform = %q, want fallback", prompt.Platform.Content)
		}
	})

	t.Run("error when both missing base", func(t *testing.T) {
		primaryDir := t.TempDir()
		fallbackDir := t.TempDir()

		_, err := LoadWithFallback(primaryDir, fallbackDir)
		if err == nil {
			t.Error("Expected error when base missing from both")
		}
	})
}

func TestLoadFromDir(t *testing.T) {
	tmpDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(tmpDir, BasePromptFile), []byte("Test"), 0644); err != nil {
		t.Fatalf("Failed to create file: %v", err)
	}

	prompt, err := LoadFromDir(tmpDir)
	if err != nil {
		t.Fatalf("LoadFromDir() error = %v", err)
	}
	if prompt.Base == nil {
		t.Error("LoadFromDir() Base is nil")
	}
}

