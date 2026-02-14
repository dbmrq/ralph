// Package app provides the main application orchestration for ralph.
// This file handles legacy .ralph directory detection and migration.
package app

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

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

// SetupState represents the state of a partial or interrupted setup.
// This is saved to .ralph/setup_state.json to allow resuming setup.
type SetupState struct {
	// Phase indicates where in the setup flow we are.
	Phase string `json:"phase"`
	// AnalysisDone indicates if project analysis completed.
	AnalysisDone bool `json:"analysis_done"`
	// AnalysisPath is the path to the cached analysis file.
	AnalysisPath string `json:"analysis_path,omitempty"`
	// TasksDone indicates if task import/generation completed.
	TasksDone bool `json:"tasks_done"`
	// TasksPath is the path to tasks file.
	TasksPath string `json:"tasks_path,omitempty"`
	// ConfigDone indicates if config was saved.
	ConfigDone bool `json:"config_done"`
	// StartedAt is when setup started.
	StartedAt string `json:"started_at"`
	// LastUpdated is when the state was last updated.
	LastUpdated string `json:"last_updated"`
}

const setupStateFile = "setup_state.json"

// SaveSetupState saves the current setup state to .ralph/setup_state.json.
func SaveSetupState(projectDir string, state *SetupState) error {
	ralphDir := filepath.Join(projectDir, ".ralph")
	if _, err := os.Stat(ralphDir); os.IsNotExist(err) {
		return nil // .ralph doesn't exist yet, nothing to save
	}

	statePath := filepath.Join(ralphDir, setupStateFile)
	return saveSetupStateToFile(statePath, state)
}

// LoadSetupState loads the setup state from .ralph/setup_state.json.
// Returns nil if no state file exists.
func LoadSetupState(projectDir string) (*SetupState, error) {
	statePath := filepath.Join(projectDir, ".ralph", setupStateFile)
	state, err := loadSetupStateFromFile(statePath)
	if os.IsNotExist(err) {
		return nil, nil // No state file, not an error
	}
	if err != nil {
		return nil, fmt.Errorf("failed to load setup state: %w", err)
	}
	return state, nil
}

// ClearSetupState removes the setup state file.
// Called when setup completes successfully.
func ClearSetupState(projectDir string) error {
	statePath := filepath.Join(projectDir, ".ralph", setupStateFile)
	if err := os.Remove(statePath); os.IsNotExist(err) {
		return nil // Already gone, not an error
	} else if err != nil {
		return fmt.Errorf("failed to clear setup state: %w", err)
	}
	return nil
}

// HasPartialSetup checks if there's a partial setup that can be resumed.
func HasPartialSetup(projectDir string) bool {
	state, err := LoadSetupState(projectDir)
	return err == nil && state != nil
}

// CleanupPartialSetup removes a partial .ralph directory.
// This is called when the user chooses not to resume an interrupted setup.
func CleanupPartialSetup(projectDir string) error {
	ralphDir := filepath.Join(projectDir, ".ralph")
	if _, err := os.Stat(ralphDir); os.IsNotExist(err) {
		return nil // Nothing to clean up
	}

	// Check if this is truly a partial setup (has setup_state.json)
	state, err := LoadSetupState(projectDir)
	if err != nil {
		return err
	}
	if state == nil {
		// No state file means this is a complete setup, don't clean up
		return fmt.Errorf("setup state not found: directory may be complete, not cleaning up")
	}

	// Remove the entire .ralph directory
	return os.RemoveAll(ralphDir)
}

// NewSetupState creates a new SetupState with initial values.
func NewSetupState(phase string) *SetupState {
	now := time.Now().Format(time.RFC3339)
	return &SetupState{
		Phase:       phase,
		StartedAt:   now,
		LastUpdated: now,
	}
}

// UpdatePhase updates the phase and last updated time.
func (s *SetupState) UpdatePhase(phase string) {
	s.Phase = phase
	s.LastUpdated = time.Now().Format(time.RFC3339)
}

// MarkAnalysisDone marks analysis as complete.
func (s *SetupState) MarkAnalysisDone(analysisPath string) {
	s.AnalysisDone = true
	s.AnalysisPath = analysisPath
	s.LastUpdated = time.Now().Format(time.RFC3339)
}

// MarkTasksDone marks tasks as complete.
func (s *SetupState) MarkTasksDone(tasksPath string) {
	s.TasksDone = true
	s.TasksPath = tasksPath
	s.LastUpdated = time.Now().Format(time.RFC3339)
}

// MarkConfigDone marks config as saved.
func (s *SetupState) MarkConfigDone() {
	s.ConfigDone = true
	s.LastUpdated = time.Now().Format(time.RFC3339)
}

// saveSetupStateToFile saves the state to a JSON file.
func saveSetupStateToFile(path string, state *SetupState) error {
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

// loadSetupStateFromFile loads the state from a JSON file.
func loadSetupStateFromFile(path string) (*SetupState, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var state SetupState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, err
	}
	return &state, nil
}

