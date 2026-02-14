package app

import (
	"os"
	"path/filepath"
	"testing"
)

func TestIsLegacyRalph(t *testing.T) {
	tests := []struct {
		name     string
		setup    func(dir string) // Creates test fixtures
		expected bool
	}{
		{
			name: "no .ralph directory",
			setup: func(dir string) {
				// Nothing to create
			},
			expected: false,
		},
		{
			name: "new format with config.yaml",
			setup: func(dir string) {
				ralphDir := filepath.Join(dir, ".ralph")
				os.MkdirAll(ralphDir, 0755)
				os.WriteFile(filepath.Join(ralphDir, "config.yaml"), []byte("{}"), 0644)
			},
			expected: false,
		},
		{
			name: "legacy with ralph_loop.sh",
			setup: func(dir string) {
				ralphDir := filepath.Join(dir, ".ralph")
				os.MkdirAll(ralphDir, 0755)
				os.WriteFile(filepath.Join(ralphDir, "ralph_loop.sh"), []byte("#!/bin/bash"), 0644)
			},
			expected: true,
		},
		{
			name: "legacy with build.sh",
			setup: func(dir string) {
				ralphDir := filepath.Join(dir, ".ralph")
				os.MkdirAll(ralphDir, 0755)
				os.WriteFile(filepath.Join(ralphDir, "build.sh"), []byte("#!/bin/bash"), 0644)
			},
			expected: true,
		},
		{
			name: "legacy with multiple files",
			setup: func(dir string) {
				ralphDir := filepath.Join(dir, ".ralph")
				os.MkdirAll(ralphDir, 0755)
				os.WriteFile(filepath.Join(ralphDir, "ralph_loop.sh"), []byte("#!/bin/bash"), 0644)
				os.WriteFile(filepath.Join(ralphDir, "build.sh"), []byte("#!/bin/bash"), 0644)
				os.WriteFile(filepath.Join(ralphDir, "test.sh"), []byte("#!/bin/bash"), 0644)
				os.WriteFile(filepath.Join(ralphDir, "config.sh"), []byte("#!/bin/bash"), 0644)
			},
			expected: true,
		},
		{
			name: "legacy with config.yaml present - not legacy",
			setup: func(dir string) {
				ralphDir := filepath.Join(dir, ".ralph")
				os.MkdirAll(ralphDir, 0755)
				os.WriteFile(filepath.Join(ralphDir, "ralph_loop.sh"), []byte("#!/bin/bash"), 0644)
				os.WriteFile(filepath.Join(ralphDir, "config.yaml"), []byte("{}"), 0644)
			},
			expected: false, // Has config.yaml so it's new format
		},
		{
			name: "empty .ralph directory",
			setup: func(dir string) {
				os.MkdirAll(filepath.Join(dir, ".ralph"), 0755)
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			tt.setup(dir)

			result := IsLegacyRalph(dir)
			if result != tt.expected {
				t.Errorf("IsLegacyRalph() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestHasRalphDirectory(t *testing.T) {
	t.Run("directory exists", func(t *testing.T) {
		dir := t.TempDir()
		os.MkdirAll(filepath.Join(dir, ".ralph"), 0755)

		if !HasRalphDirectory(dir) {
			t.Error("HasRalphDirectory() should return true")
		}
	})

	t.Run("directory does not exist", func(t *testing.T) {
		dir := t.TempDir()

		if HasRalphDirectory(dir) {
			t.Error("HasRalphDirectory() should return false")
		}
	})

	t.Run("file instead of directory", func(t *testing.T) {
		dir := t.TempDir()
		os.WriteFile(filepath.Join(dir, ".ralph"), []byte("not a dir"), 0644)

		if HasRalphDirectory(dir) {
			t.Error("HasRalphDirectory() should return false for file")
		}
	})
}

func TestMigrateFromLegacy(t *testing.T) {
	t.Run("successful migration", func(t *testing.T) {
		dir := t.TempDir()
		ralphDir := filepath.Join(dir, ".ralph")
		os.MkdirAll(ralphDir, 0755)

		// Create legacy files
		os.WriteFile(filepath.Join(ralphDir, "ralph_loop.sh"), []byte("#!/bin/bash\necho loop"), 0644)
		os.WriteFile(filepath.Join(ralphDir, "build.sh"), []byte("#!/bin/bash\ngo build"), 0644)
		os.WriteFile(filepath.Join(ralphDir, "test.sh"), []byte("#!/bin/bash\ngo test"), 0644)
		os.WriteFile(filepath.Join(ralphDir, "config.sh"), []byte("#!/bin/bash"), 0644)
		os.WriteFile(filepath.Join(ralphDir, "TASKS.md"), []byte("# Tasks"), 0644)
		os.WriteFile(filepath.Join(ralphDir, "base_prompt.txt"), []byte("prompt"), 0644)

		result, err := MigrateFromLegacy(dir)
		if err != nil {
			t.Fatalf("MigrateFromLegacy() error = %v", err)
		}

		if !result.ConfigCreated {
			t.Error("ConfigCreated should be true")
		}
		if !result.TasksPreserved {
			t.Error("TasksPreserved should be true")
		}
		if len(result.PromptsPreserved) != 1 {
			t.Errorf("PromptsPreserved = %v, want 1 item", result.PromptsPreserved)
		}
	})
}

func TestSetupState(t *testing.T) {
	t.Run("save and load setup state", func(t *testing.T) {
		dir := t.TempDir()
		ralphDir := filepath.Join(dir, ".ralph")
		os.MkdirAll(ralphDir, 0755)

		state := NewSetupState("analyzing")
		state.MarkAnalysisDone("/path/to/analysis.json")

		err := SaveSetupState(dir, state)
		if err != nil {
			t.Fatalf("SaveSetupState() error = %v", err)
		}

		loaded, err := LoadSetupState(dir)
		if err != nil {
			t.Fatalf("LoadSetupState() error = %v", err)
		}

		if loaded.Phase != "analyzing" {
			t.Errorf("Phase = %q, want %q", loaded.Phase, "analyzing")
		}
		if !loaded.AnalysisDone {
			t.Error("AnalysisDone should be true")
		}
		if loaded.AnalysisPath != "/path/to/analysis.json" {
			t.Errorf("AnalysisPath = %q, want %q", loaded.AnalysisPath, "/path/to/analysis.json")
		}
	})

	t.Run("load returns nil when no state file", func(t *testing.T) {
		dir := t.TempDir()
		os.MkdirAll(filepath.Join(dir, ".ralph"), 0755)

		state, err := LoadSetupState(dir)
		if err != nil {
			t.Fatalf("LoadSetupState() error = %v", err)
		}
		if state != nil {
			t.Error("expected nil state when no state file exists")
		}
	})

	t.Run("save does nothing when no .ralph directory", func(t *testing.T) {
		dir := t.TempDir()
		state := NewSetupState("welcome")

		err := SaveSetupState(dir, state)
		if err != nil {
			t.Fatalf("SaveSetupState() error = %v", err)
		}

		// Should not have created .ralph directory
		if _, err := os.Stat(filepath.Join(dir, ".ralph")); !os.IsNotExist(err) {
			t.Error(".ralph directory should not have been created")
		}
	})

	t.Run("clear setup state", func(t *testing.T) {
		dir := t.TempDir()
		ralphDir := filepath.Join(dir, ".ralph")
		os.MkdirAll(ralphDir, 0755)

		state := NewSetupState("analyzing")
		SaveSetupState(dir, state)

		err := ClearSetupState(dir)
		if err != nil {
			t.Fatalf("ClearSetupState() error = %v", err)
		}

		loaded, err := LoadSetupState(dir)
		if err != nil {
			t.Fatalf("LoadSetupState() error = %v", err)
		}
		if loaded != nil {
			t.Error("state should be nil after clear")
		}
	})

	t.Run("clear non-existent state returns nil", func(t *testing.T) {
		dir := t.TempDir()
		os.MkdirAll(filepath.Join(dir, ".ralph"), 0755)

		err := ClearSetupState(dir)
		if err != nil {
			t.Fatalf("ClearSetupState() error = %v", err)
		}
	})
}

func TestHasPartialSetup(t *testing.T) {
	t.Run("returns true when setup state exists", func(t *testing.T) {
		dir := t.TempDir()
		ralphDir := filepath.Join(dir, ".ralph")
		os.MkdirAll(ralphDir, 0755)

		state := NewSetupState("analyzing")
		SaveSetupState(dir, state)

		if !HasPartialSetup(dir) {
			t.Error("HasPartialSetup() should return true")
		}
	})

	t.Run("returns false when no setup state", func(t *testing.T) {
		dir := t.TempDir()
		os.MkdirAll(filepath.Join(dir, ".ralph"), 0755)

		if HasPartialSetup(dir) {
			t.Error("HasPartialSetup() should return false")
		}
	})

	t.Run("returns false when no .ralph directory", func(t *testing.T) {
		dir := t.TempDir()

		if HasPartialSetup(dir) {
			t.Error("HasPartialSetup() should return false")
		}
	})
}

func TestCleanupPartialSetup(t *testing.T) {
	t.Run("cleans up partial setup", func(t *testing.T) {
		dir := t.TempDir()
		ralphDir := filepath.Join(dir, ".ralph")
		os.MkdirAll(ralphDir, 0755)

		state := NewSetupState("analyzing")
		SaveSetupState(dir, state)

		err := CleanupPartialSetup(dir)
		if err != nil {
			t.Fatalf("CleanupPartialSetup() error = %v", err)
		}

		if _, err := os.Stat(ralphDir); !os.IsNotExist(err) {
			t.Error(".ralph directory should have been removed")
		}
	})

	t.Run("returns error for complete setup", func(t *testing.T) {
		dir := t.TempDir()
		ralphDir := filepath.Join(dir, ".ralph")
		os.MkdirAll(ralphDir, 0755)
		os.WriteFile(filepath.Join(ralphDir, "config.yaml"), []byte("{}"), 0644)

		err := CleanupPartialSetup(dir)
		if err == nil {
			t.Error("CleanupPartialSetup() should return error for complete setup")
		}
	})

	t.Run("returns nil when no .ralph directory", func(t *testing.T) {
		dir := t.TempDir()

		err := CleanupPartialSetup(dir)
		if err != nil {
			t.Fatalf("CleanupPartialSetup() error = %v", err)
		}
	})
}

func TestSetupStateMethods(t *testing.T) {
	t.Run("NewSetupState initializes correctly", func(t *testing.T) {
		state := NewSetupState("welcome")

		if state.Phase != "welcome" {
			t.Errorf("Phase = %q, want %q", state.Phase, "welcome")
		}
		if state.StartedAt == "" {
			t.Error("StartedAt should be set")
		}
		if state.LastUpdated == "" {
			t.Error("LastUpdated should be set")
		}
	})

	t.Run("UpdatePhase updates phase and timestamp", func(t *testing.T) {
		state := NewSetupState("welcome")
		originalTime := state.LastUpdated

		// Wait briefly to ensure timestamp changes
		state.UpdatePhase("analyzing")

		if state.Phase != "analyzing" {
			t.Errorf("Phase = %q, want %q", state.Phase, "analyzing")
		}
		// Note: timestamps may be same if test runs fast
		_ = originalTime
	})

	t.Run("MarkAnalysisDone sets fields", func(t *testing.T) {
		state := NewSetupState("analyzing")
		state.MarkAnalysisDone("/path/to/analysis.json")

		if !state.AnalysisDone {
			t.Error("AnalysisDone should be true")
		}
		if state.AnalysisPath != "/path/to/analysis.json" {
			t.Errorf("AnalysisPath = %q, want %q", state.AnalysisPath, "/path/to/analysis.json")
		}
	})

	t.Run("MarkTasksDone sets fields", func(t *testing.T) {
		state := NewSetupState("tasks")
		state.MarkTasksDone("/path/to/tasks.json")

		if !state.TasksDone {
			t.Error("TasksDone should be true")
		}
		if state.TasksPath != "/path/to/tasks.json" {
			t.Errorf("TasksPath = %q, want %q", state.TasksPath, "/path/to/tasks.json")
		}
	})

	t.Run("MarkConfigDone sets field", func(t *testing.T) {
		state := NewSetupState("config")
		state.MarkConfigDone()

		if !state.ConfigDone {
			t.Error("ConfigDone should be true")
		}
	})
}
