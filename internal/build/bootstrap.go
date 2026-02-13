// Package build provides build and test verification logic for ralph.
package build

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

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

// ProjectType represents the detected project type.
type ProjectType string

const (
	ProjectTypeGo      ProjectType = "go"
	ProjectTypeNode    ProjectType = "node"
	ProjectTypePython  ProjectType = "python"
	ProjectTypeRust    ProjectType = "rust"
	ProjectTypeUnknown ProjectType = "unknown"
)

// BootstrapDetector detects the bootstrap state of a project.
type BootstrapDetector struct {
	// ProjectDir is the root directory of the project.
	ProjectDir string
	// Config is the build configuration.
	Config config.BuildConfig
}

// NewBootstrapDetector creates a new BootstrapDetector.
func NewBootstrapDetector(projectDir string, cfg config.BuildConfig) *BootstrapDetector {
	return &BootstrapDetector{
		ProjectDir: projectDir,
		Config:     cfg,
	}
}

// Detect returns the bootstrap state of the project.
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

// detectAuto automatically detects project state based on project type.
func (d *BootstrapDetector) detectAuto() (*BootstrapState, error) {
	projectType := d.DetectProjectType()

	var buildReady, testReady bool
	var reasons []string

	switch projectType {
	case ProjectTypeGo:
		buildReady, testReady, reasons = d.detectGoProject()
	case ProjectTypeNode:
		buildReady, testReady, reasons = d.detectNodeProject()
	case ProjectTypePython:
		buildReady, testReady, reasons = d.detectPythonProject()
	case ProjectTypeRust:
		buildReady, testReady, reasons = d.detectRustProject()
	default:
		buildReady, testReady, reasons = d.detectGenericProject()
	}

	return &BootstrapState{
		BuildReady: buildReady,
		TestReady:  testReady,
		Reason:     strings.Join(reasons, "; "),
	}, nil
}

// DetectProjectType determines the project type based on markers.
func (d *BootstrapDetector) DetectProjectType() ProjectType {
	// Check for Go
	if d.fileExists("go.mod") {
		return ProjectTypeGo
	}
	// Check for Node/JavaScript
	if d.fileExists("package.json") {
		return ProjectTypeNode
	}
	// Check for Python
	if d.fileExists("pyproject.toml") || d.fileExists("setup.py") || d.fileExists("requirements.txt") {
		return ProjectTypePython
	}
	// Check for Rust
	if d.fileExists("Cargo.toml") {
		return ProjectTypeRust
	}
	return ProjectTypeUnknown
}

// detectGoProject checks if a Go project is build/test ready.
func (d *BootstrapDetector) detectGoProject() (buildReady, testReady bool, reasons []string) {
	// Build ready: go.mod exists AND at least one .go file exists
	if !d.fileExists("go.mod") {
		reasons = append(reasons, "no go.mod found")
		return false, false, reasons
	}

	goFiles := d.findFiles("*.go")
	if len(goFiles) == 0 {
		reasons = append(reasons, "go.mod exists but no .go files found")
		return false, false, reasons
	}

	buildReady = true
	reasons = append(reasons, fmt.Sprintf("Go project: go.mod and %d .go file(s) found", len(goFiles)))

	// Test ready: at least one *_test.go file exists
	testFiles := d.findFiles("*_test.go")
	if len(testFiles) > 0 {
		testReady = true
		reasons = append(reasons, fmt.Sprintf("%d test file(s) found", len(testFiles)))
	} else {
		reasons = append(reasons, "no *_test.go files found")
	}

	return buildReady, testReady, reasons
}

// detectNodeProject checks if a Node.js project is build/test ready.
func (d *BootstrapDetector) detectNodeProject() (buildReady, testReady bool, reasons []string) {
	// Build ready: package.json exists
	if !d.fileExists("package.json") {
		reasons = append(reasons, "no package.json found")
		return false, false, reasons
	}

	// Check for node_modules (dependencies installed)
	if !d.dirExists("node_modules") {
		reasons = append(reasons, "package.json exists but node_modules not found (run npm install)")
		return false, false, reasons
	}

	buildReady = true
	reasons = append(reasons, "Node project: package.json and node_modules found")

	// Test ready: check for test files
	testPatterns := []string{"*.test.js", "*.spec.js", "*.test.ts", "*.spec.ts", "*.test.jsx", "*.spec.jsx", "*.test.tsx", "*.spec.tsx"}
	totalTests := 0
	for _, pattern := range testPatterns {
		totalTests += len(d.findFiles(pattern))
	}

	if totalTests > 0 {
		testReady = true
		reasons = append(reasons, fmt.Sprintf("%d test file(s) found", totalTests))
	} else {
		reasons = append(reasons, "no test files found (*.test.js, *.spec.js, etc.)")
	}

	return buildReady, testReady, reasons
}

// detectPythonProject checks if a Python project is build/test ready.
func (d *BootstrapDetector) detectPythonProject() (buildReady, testReady bool, reasons []string) {
	// Build ready: has pyproject.toml, setup.py, or at least one .py file
	hasPyproject := d.fileExists("pyproject.toml")
	hasSetupPy := d.fileExists("setup.py")
	pyFiles := d.findFiles("*.py")

	if !hasPyproject && !hasSetupPy && len(pyFiles) == 0 {
		reasons = append(reasons, "no Python project markers found")
		return false, false, reasons
	}

	buildReady = true
	if hasPyproject {
		reasons = append(reasons, "Python project: pyproject.toml found")
	} else if hasSetupPy {
		reasons = append(reasons, "Python project: setup.py found")
	} else {
		reasons = append(reasons, fmt.Sprintf("Python project: %d .py file(s) found", len(pyFiles)))
	}

	// Test ready: check for test files (test_*.py or *_test.py)
	testFiles := d.findFiles("test_*.py")
	testFiles = append(testFiles, d.findFiles("*_test.py")...)

	if len(testFiles) > 0 {
		testReady = true
		reasons = append(reasons, fmt.Sprintf("%d test file(s) found", len(testFiles)))
	} else {
		reasons = append(reasons, "no test files found (test_*.py, *_test.py)")
	}

	return buildReady, testReady, reasons
}

// detectRustProject checks if a Rust project is build/test ready.
func (d *BootstrapDetector) detectRustProject() (buildReady, testReady bool, reasons []string) {
	// Build ready: Cargo.toml exists AND at least one .rs file in src/
	if !d.fileExists("Cargo.toml") {
		reasons = append(reasons, "no Cargo.toml found")
		return false, false, reasons
	}

	rsFiles := d.findFilesInDir("src", "*.rs")
	if len(rsFiles) == 0 {
		reasons = append(reasons, "Cargo.toml exists but no .rs files in src/")
		return false, false, reasons
	}

	buildReady = true
	reasons = append(reasons, fmt.Sprintf("Rust project: Cargo.toml and %d .rs file(s) found", len(rsFiles)))

	// Test ready: check for tests directory or #[test] in source (we check tests/ dir for simplicity)
	testFiles := d.findFilesInDir("tests", "*.rs")
	if len(testFiles) > 0 {
		testReady = true
		reasons = append(reasons, fmt.Sprintf("%d integration test file(s) found", len(testFiles)))
	} else {
		// Rust tests can also be inline; assume test ready if we have source files
		testReady = true
		reasons = append(reasons, "assuming unit tests may exist inline in source files")
	}

	return buildReady, testReady, reasons
}

// detectGenericProject provides a fallback for unknown project types.
func (d *BootstrapDetector) detectGenericProject() (buildReady, testReady bool, reasons []string) {
	// Look for any source-like files
	sourcePatterns := []string{"*.go", "*.py", "*.js", "*.ts", "*.rs", "*.java", "*.c", "*.cpp", "*.rb"}
	var foundSources []string
	for _, pattern := range sourcePatterns {
		files := d.findFiles(pattern)
		if len(files) > 0 {
			foundSources = append(foundSources, fmt.Sprintf("%d %s", len(files), pattern))
		}
	}

	if len(foundSources) == 0 {
		reasons = append(reasons, "no source files found")
		return false, false, reasons
	}

	buildReady = true
	reasons = append(reasons, fmt.Sprintf("Generic project: found %s", strings.Join(foundSources, ", ")))

	// Assume tests might exist but we can't reliably detect them
	testReady = false
	reasons = append(reasons, "unknown project type, cannot detect tests")

	return buildReady, testReady, reasons
}


// Helper methods

// fileExists checks if a file exists in the project directory.
func (d *BootstrapDetector) fileExists(name string) bool {
	path := filepath.Join(d.ProjectDir, name)
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !info.IsDir()
}

// dirExists checks if a directory exists in the project directory.
func (d *BootstrapDetector) dirExists(name string) bool {
	path := filepath.Join(d.ProjectDir, name)
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.IsDir()
}

// findFiles finds files matching a pattern recursively in the project directory.
// It skips common non-source directories (vendor, node_modules, .git, etc.)
func (d *BootstrapDetector) findFiles(pattern string) []string {
	var matches []string

	skipDirs := map[string]bool{
		"vendor":       true,
		"node_modules": true,
		".git":         true,
		".ralph":       true,
		"__pycache__":  true,
		".venv":        true,
		"venv":         true,
		"target":       true, // Rust build dir
	}

	_ = filepath.Walk(d.ProjectDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // skip errors
		}
		if info.IsDir() {
			if skipDirs[info.Name()] {
				return filepath.SkipDir
			}
			return nil
		}
		matched, _ := filepath.Match(pattern, info.Name())
		if matched {
			matches = append(matches, path)
		}
		return nil
	})

	return matches
}

// findFilesInDir finds files matching a pattern in a specific subdirectory.
func (d *BootstrapDetector) findFilesInDir(dir, pattern string) []string {
	var matches []string
	searchDir := filepath.Join(d.ProjectDir, dir)

	if !d.dirExists(dir) {
		return matches
	}

	_ = filepath.Walk(searchDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		matched, _ := filepath.Match(pattern, info.Name())
		if matched {
			matches = append(matches, path)
		}
		return nil
	})

	return matches
}

