// Package build provides build and test verification logic for ralph.
package build

import (
	"context"
	"fmt"
	"os/exec"

	"github.com/wexinc/ralph/internal/config"
)

// BootstrapState represents the current bootstrap/greenfield state of a project.
type BootstrapState struct {
	// BuildReady indicates whether the project has buildable code.
	BuildReady bool `json:"build_ready"`
	// TestReady indicates whether the project has test files.
	TestReady bool `json:"test_ready"`
	// Reason provides a human-readable explanation of the state.
	Reason string `json:"reason"`
}

// BuildAnalysis contains build-related analysis results from the AI.
type BuildAnalysis struct {
	// Ready indicates whether the project can be built.
	Ready bool `json:"ready"`
	// Command is the detected build command, nil if not detected.
	Command *string `json:"command"`
	// Reason provides a human-readable explanation.
	Reason string `json:"reason"`
}

// TestAnalysis contains test-related analysis results from the AI.
type TestAnalysis struct {
	// Ready indicates whether tests can be run.
	Ready bool `json:"ready"`
	// Command is the detected test command, nil if not detected.
	Command *string `json:"command"`
	// HasTestFiles indicates whether test files were found.
	HasTestFiles bool `json:"has_test_files"`
	// Reason provides a human-readable explanation.
	Reason string `json:"reason"`
}

// LintAnalysis contains linting-related analysis results from the AI.
type LintAnalysis struct {
	// Command is the detected lint command, nil if not detected.
	Command *string `json:"command"`
	// Available indicates whether linting is available.
	Available bool `json:"available"`
}

// DependencyAnalysis contains dependency-related analysis results from the AI.
type DependencyAnalysis struct {
	// Manager is the detected package manager (e.g., "go mod", "npm", "pip").
	Manager string `json:"manager"`
	// Installed indicates whether dependencies are installed.
	Installed bool `json:"installed"`
}

// TaskListAnalysis contains task list detection results from the AI.
type TaskListAnalysis struct {
	// Detected indicates whether a task list was found.
	Detected bool `json:"detected"`
	// Path is the path to the detected task list file.
	Path string `json:"path"`
	// Format is the format of the task list (e.g., "markdown", "json").
	Format string `json:"format"`
	// TaskCount is the number of tasks detected.
	TaskCount int `json:"task_count"`
}

// ProjectAnalysis contains the AI-driven analysis results for a project.
// This struct is populated by the Project Analysis Agent (BUILD-001) before
// the task loop starts. It replaces the previous hardcoded pattern-based detection.
type ProjectAnalysis struct {
	// ProjectType is the detected project type (e.g., "go", "node", "python", "rust", "mixed", "unknown").
	ProjectType string `json:"project_type"`
	// Languages is a list of programming languages detected in the project.
	Languages []string `json:"languages"`
	// IsGreenfield indicates whether this is a new project with no buildable code yet.
	IsGreenfield bool `json:"is_greenfield"`
	// IsMonorepo indicates whether this is a monorepo with multiple packages/projects.
	IsMonorepo bool `json:"is_monorepo"`

	// Build contains build-related analysis.
	Build BuildAnalysis `json:"build"`
	// Test contains test-related analysis.
	Test TestAnalysis `json:"test"`
	// Lint contains linting-related analysis.
	Lint LintAnalysis `json:"lint"`
	// Dependencies contains dependency-related analysis.
	Dependencies DependencyAnalysis `json:"dependencies"`
	// TaskList contains task list detection results.
	TaskList TaskListAnalysis `json:"task_list"`

	// ProjectContext is a human-readable description of the project for injection into prompts.
	ProjectContext string `json:"project_context"`
}

// ToBootstrapState converts the ProjectAnalysis to a BootstrapState.
// This allows the rest of the system to work with the simplified BootstrapState
// while the AI-driven analysis provides the detailed ProjectAnalysis.
func (p *ProjectAnalysis) ToBootstrapState() *BootstrapState {
	var reason string
	if p.IsGreenfield {
		reason = "greenfield project (no buildable code yet)"
	} else {
		reasons := []string{}
		if p.Build.Ready {
			reasons = append(reasons, "build ready")
		} else {
			reasons = append(reasons, "build not ready: "+p.Build.Reason)
		}
		if p.Test.Ready {
			reasons = append(reasons, "tests ready")
		} else {
			reasons = append(reasons, "tests not ready: "+p.Test.Reason)
		}
		reason = joinReasons(reasons)
	}

	return &BootstrapState{
		BuildReady: p.Build.Ready,
		TestReady:  p.Test.Ready,
		Reason:     reason,
	}
}

// joinReasons joins multiple reasons with semicolons.
func joinReasons(reasons []string) string {
	if len(reasons) == 0 {
		return ""
	}
	result := reasons[0]
	for i := 1; i < len(reasons); i++ {
		result += "; " + reasons[i]
	}
	return result
}

// BootstrapDetector detects the bootstrap state of a project.
// It supports three modes:
//   - "auto" (default): Uses AI-driven ProjectAnalysis for detection
//   - "manual": Uses a custom bootstrap_check command
//   - "disabled": Always considers the project ready
type BootstrapDetector struct {
	// ProjectDir is the root directory of the project.
	ProjectDir string
	// Config is the build configuration.
	Config config.BuildConfig
	// Analysis is the AI-driven project analysis (used in "auto" mode).
	// When nil in "auto" mode, the detector will return an error indicating
	// that analysis is required.
	Analysis *ProjectAnalysis
}

// NewBootstrapDetector creates a new BootstrapDetector.
// For "auto" mode detection, call SetAnalysis() before calling Detect().
func NewBootstrapDetector(projectDir string, cfg config.BuildConfig) *BootstrapDetector {
	return &BootstrapDetector{
		ProjectDir: projectDir,
		Config:     cfg,
	}
}

// SetAnalysis sets the AI-driven project analysis for use in "auto" mode.
func (d *BootstrapDetector) SetAnalysis(analysis *ProjectAnalysis) {
	d.Analysis = analysis
}

// Detect returns the bootstrap state of the project.
// In "auto" mode, this uses the ProjectAnalysis set via SetAnalysis().
// In "manual" mode, this runs the configured bootstrap_check command.
// In "disabled" mode, this always returns a ready state.
func (d *BootstrapDetector) Detect(ctx context.Context) (*BootstrapState, error) {
	switch d.Config.BootstrapDetection {
	case config.BootstrapDetectionDisabled:
		return &BootstrapState{
			BuildReady: true,
			TestReady:  true,
			Reason:     "bootstrap detection disabled",
		}, nil

	case config.BootstrapDetectionManual:
		return d.detectManual(ctx)

	case config.BootstrapDetectionAuto, "":
		return d.detectAuto()

	default:
		return nil, fmt.Errorf("unknown bootstrap_detection mode: %s", d.Config.BootstrapDetection)
	}
}

// detectManual runs the custom bootstrap_check command.
func (d *BootstrapDetector) detectManual(ctx context.Context) (*BootstrapState, error) {
	if d.Config.BootstrapCheck == "" {
		return nil, fmt.Errorf("bootstrap_detection is 'manual' but bootstrap_check command is not set")
	}

	cmd := exec.CommandContext(ctx, "sh", "-c", d.Config.BootstrapCheck)
	cmd.Dir = d.ProjectDir

	err := cmd.Run()
	if err != nil {
		// Non-zero exit = project is ready for verification
		if _, ok := err.(*exec.ExitError); ok {
			return &BootstrapState{
				BuildReady: true,
				TestReady:  true,
				Reason:     "bootstrap_check command returned non-zero (project ready)",
			}, nil
		}
		return nil, fmt.Errorf("failed to run bootstrap_check: %w", err)
	}

	// Exit 0 = still in bootstrap phase
	return &BootstrapState{
		BuildReady: false,
		TestReady:  false,
		Reason:     "bootstrap_check command returned 0 (still bootstrapping)",
	}, nil
}

// detectAuto uses the AI-driven ProjectAnalysis to determine bootstrap state.
// If no analysis is available, it returns an error indicating that analysis is required.
func (d *BootstrapDetector) detectAuto() (*BootstrapState, error) {
	if d.Analysis == nil {
		return nil, fmt.Errorf("bootstrap_detection is 'auto' but no ProjectAnalysis is available; run Project Analysis Agent first")
	}

	return d.Analysis.ToBootstrapState(), nil
}

