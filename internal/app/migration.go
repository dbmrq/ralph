// Package app provides the main application orchestration for ralph.
// This file handles legacy .ralph directory detection and migration.
package app

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/wexinc/ralph/internal/config"
)

// LegacyMarkers are files that indicate a legacy shell-based .ralph directory.
var LegacyMarkers = []string{
	"ralph_loop.sh",
	"build.sh",
	"test.sh",
	"config.sh",
}

// MigrationResult contains the results of a legacy migration.
type MigrationResult struct {
	// ConfigCreated is true if a new config.yaml was created.
	ConfigCreated bool
	// TasksPreserved is true if TASKS.md was preserved.
	TasksPreserved bool
	// PromptsPreserved lists prompt files that were preserved.
	PromptsPreserved []string
	// FilesRemoved lists legacy files that were removed.
	FilesRemoved []string
	// Warnings contains any non-fatal issues during migration.
	Warnings []string
}

// IsLegacyRalph checks if the .ralph directory is from the legacy shell version.
// Legacy format has ralph_loop.sh, build.sh, etc. but no config.yaml.
func IsLegacyRalph(projectDir string) bool {
	ralphDir := filepath.Join(projectDir, ".ralph")

	// Check if .ralph exists
	info, err := os.Stat(ralphDir)
	if err != nil || !info.IsDir() {
		return false
	}

	// Check for config.yaml - if it exists, this is the new format
	configPath := filepath.Join(ralphDir, "config.yaml")
	if _, err := os.Stat(configPath); err == nil {
		return false
	}

	// Check for any legacy marker files
	for _, marker := range LegacyMarkers {
		markerPath := filepath.Join(ralphDir, marker)
		if _, err := os.Stat(markerPath); err == nil {
			return true
		}
	}

	return false
}

// HasRalphDirectory checks if a .ralph directory exists.
func HasRalphDirectory(projectDir string) bool {
	ralphDir := filepath.Join(projectDir, ".ralph")
	info, err := os.Stat(ralphDir)
	return err == nil && info.IsDir()
}

// MigrateFromLegacy migrates a legacy .ralph directory to the new format.
// It preserves TASKS.md, prompt files, and creates a new config.yaml.
// Legacy shell script files are removed.
func MigrateFromLegacy(projectDir string) (*MigrationResult, error) {
	ralphDir := filepath.Join(projectDir, ".ralph")
	result := &MigrationResult{}

	// Verify this is indeed a legacy directory
	if !IsLegacyRalph(projectDir) {
		return nil, fmt.Errorf("not a legacy ralph directory")
	}

	// Preserve TASKS.md
	tasksPath := filepath.Join(ralphDir, "TASKS.md")
	if _, err := os.Stat(tasksPath); err == nil {
		result.TasksPreserved = true
	}

	// Preserve prompt files
	promptFiles := []string{"base_prompt.txt", "platform_prompt.txt", "project_prompt.txt"}
	for _, pf := range promptFiles {
		pfPath := filepath.Join(ralphDir, pf)
		if _, err := os.Stat(pfPath); err == nil {
			result.PromptsPreserved = append(result.PromptsPreserved, pf)
		}
	}

	// Create new config.yaml with defaults
	cfg := config.NewConfig()

	// Try to parse build command from build.sh
	buildCmd := parseShellCommand(filepath.Join(ralphDir, "build.sh"))
	if buildCmd != "" {
		cfg.Build.Command = buildCmd
	}

	// Try to parse test command from test.sh
	testCmd := parseShellCommand(filepath.Join(ralphDir, "test.sh"))
	if testCmd != "" {
		cfg.Test.Command = testCmd
	}

	// Save the new config
	configPath := filepath.Join(ralphDir, "config.yaml")
	if err := config.Save(cfg, configPath); err != nil {
		return nil, fmt.Errorf("failed to create config.yaml: %w", err)
	}
	result.ConfigCreated = true

	// Create required subdirectories
	subdirs := []string{"sessions", "logs"}
	for _, subdir := range subdirs {
		path := filepath.Join(ralphDir, subdir)
		if err := os.MkdirAll(path, 0755); err != nil {
			result.Warnings = append(result.Warnings, fmt.Sprintf("failed to create %s: %v", subdir, err))
		}
	}

	// Remove legacy shell script files (optionally, we could back them up)
	legacyFiles := append(LegacyMarkers, "config.sh")
	for _, lf := range legacyFiles {
		lfPath := filepath.Join(ralphDir, lf)
		if _, err := os.Stat(lfPath); err == nil {
			if err := os.Remove(lfPath); err != nil {
				result.Warnings = append(result.Warnings, fmt.Sprintf("failed to remove %s: %v", lf, err))
			} else {
				result.FilesRemoved = append(result.FilesRemoved, lf)
			}
		}
	}

	return result, nil
}

// parseShellCommand attempts to extract a command from a shell script.
// Returns empty string if the file doesn't exist or can't be parsed.
func parseShellCommand(path string) string {
	content, err := os.ReadFile(path)
	if err != nil {
		return ""
	}

	// Look for common patterns in the shell script
	// This is a simple heuristic - just return empty for now
	// A full implementation would parse the script
	_ = content
	return ""
}

