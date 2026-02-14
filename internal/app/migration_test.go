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

