// Package project provides project directory detection and selection.
package project

import (
	"os"
	"path/filepath"
)

// ProjectInfo contains information about a detected project.
type ProjectInfo struct {
	// Path is the absolute path to the project directory.
	Path string `json:"path"`
	// Name is the project name (usually the directory name).
	Name string `json:"name"`
	// IsGitRepo indicates whether this is a git repository.
	IsGitRepo bool `json:"is_git_repo"`
	// ProjectType is the detected project type (go, node, python, rust, etc.).
	ProjectType string `json:"project_type,omitempty"`
	// HasRalph indicates whether the project has a .ralph directory.
	HasRalph bool `json:"has_ralph"`
	// Markers are the project markers found (go.mod, package.json, etc.).
	Markers []string `json:"markers,omitempty"`
}

// ProjectMarker represents a file or directory that indicates a project type.
type ProjectMarker struct {
	// Name is the file or directory name to look for.
	Name string
	// IsDir indicates whether this is a directory marker.
	IsDir bool
	// ProjectType is the project type this marker indicates.
	ProjectType string
}

// DefaultMarkers are the project markers checked during detection.
var DefaultMarkers = []ProjectMarker{
	// Version control
	{Name: ".git", IsDir: true, ProjectType: ""},
	// Go projects
	{Name: "go.mod", IsDir: false, ProjectType: "go"},
	{Name: "go.sum", IsDir: false, ProjectType: "go"},
	// Node.js projects
	{Name: "package.json", IsDir: false, ProjectType: "node"},
	{Name: "package-lock.json", IsDir: false, ProjectType: "node"},
	{Name: "yarn.lock", IsDir: false, ProjectType: "node"},
	{Name: "pnpm-lock.yaml", IsDir: false, ProjectType: "node"},
	// Python projects
	{Name: "pyproject.toml", IsDir: false, ProjectType: "python"},
	{Name: "setup.py", IsDir: false, ProjectType: "python"},
	{Name: "requirements.txt", IsDir: false, ProjectType: "python"},
	{Name: "Pipfile", IsDir: false, ProjectType: "python"},
	// Rust projects
	{Name: "Cargo.toml", IsDir: false, ProjectType: "rust"},
	{Name: "Cargo.lock", IsDir: false, ProjectType: "rust"},
	// Java/Kotlin/Gradle projects
	{Name: "build.gradle", IsDir: false, ProjectType: "gradle"},
	{Name: "build.gradle.kts", IsDir: false, ProjectType: "gradle"},
	{Name: "pom.xml", IsDir: false, ProjectType: "maven"},
	// .NET projects
	{Name: "*.csproj", IsDir: false, ProjectType: "dotnet"},
	{Name: "*.fsproj", IsDir: false, ProjectType: "dotnet"},
	// Ruby projects
	{Name: "Gemfile", IsDir: false, ProjectType: "ruby"},
	{Name: "Gemfile.lock", IsDir: false, ProjectType: "ruby"},
	// PHP projects
	{Name: "composer.json", IsDir: false, ProjectType: "php"},
	// Swift/iOS projects
	{Name: "Package.swift", IsDir: false, ProjectType: "swift"},
	{Name: "*.xcodeproj", IsDir: true, ProjectType: "xcode"},
	{Name: "*.xcworkspace", IsDir: true, ProjectType: "xcode"},
	// Generic project markers
	{Name: "Makefile", IsDir: false, ProjectType: "make"},
	{Name: "CMakeLists.txt", IsDir: false, ProjectType: "cmake"},
	// Ralph specific
	{Name: ".ralph", IsDir: true, ProjectType: ""},
}

// Detector detects project directories.
type Detector struct {
	// Markers are the project markers to check.
	Markers []ProjectMarker
}

// NewDetector creates a new Detector with default markers.
func NewDetector() *Detector {
	return &Detector{
		Markers: DefaultMarkers,
	}
}

// DetectProject checks if a directory is a valid project directory.
// Returns ProjectInfo if the directory appears to be a project, or nil if not.
func (d *Detector) DetectProject(dir string) (*ProjectInfo, error) {
	absPath, err := filepath.Abs(dir)
	if err != nil {
		return nil, err
	}

	info, err := os.Stat(absPath)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return nil, os.ErrNotExist
	}

	project := &ProjectInfo{
		Path:    absPath,
		Name:    filepath.Base(absPath),
		Markers: []string{},
	}

	// Check all markers
	for _, marker := range d.Markers {
		if d.checkMarker(absPath, marker) {
			project.Markers = append(project.Markers, marker.Name)
			// Set specific flags
			switch marker.Name {
			case ".git":
				project.IsGitRepo = true
			case ".ralph":
				project.HasRalph = true
			default:
				if marker.ProjectType != "" && project.ProjectType == "" {
					project.ProjectType = marker.ProjectType
				}
			}
		}
	}

	// If no markers found, this might not be a project directory
	if len(project.Markers) == 0 {
		return nil, nil
	}

	return project, nil
}

// checkMarker checks if a specific marker exists in the directory.
func (d *Detector) checkMarker(dir string, marker ProjectMarker) bool {
	// Handle glob patterns
	if containsGlob(marker.Name) {
		matches, err := filepath.Glob(filepath.Join(dir, marker.Name))
		return err == nil && len(matches) > 0
	}
	// Check exact match
	path := filepath.Join(dir, marker.Name)
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.IsDir() == marker.IsDir
}

// containsGlob checks if a string contains glob characters.
func containsGlob(s string) bool {
	for _, c := range s {
		if c == '*' || c == '?' || c == '[' {
			return true
		}
	}
	return false
}

// IsProjectDirectory returns true if the directory appears to be a project.
func (d *Detector) IsProjectDirectory(dir string) bool {
	project, err := d.DetectProject(dir)
	return err == nil && project != nil
}

// IsHomeDirectory returns true if the directory is the user's home directory.
func IsHomeDirectory(dir string) bool {
	home, err := os.UserHomeDir()
	if err != nil {
		return false
	}
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return false
	}
	absHome, err := filepath.Abs(home)
	if err != nil {
		return false
	}
	return absDir == absHome
}

// IsRootDirectory returns true if the directory is the root directory.
func IsRootDirectory(dir string) bool {
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return false
	}
	return absDir == "/" || absDir == filepath.VolumeName(absDir)+"\\"
}

// ShouldPromptForDirectory returns true if ralph should prompt for a project directory.
// This is true when the current directory doesn't look like a project.
func (d *Detector) ShouldPromptForDirectory(dir string) bool {
	// Always prompt in home or root directory
	if IsHomeDirectory(dir) || IsRootDirectory(dir) {
		return true
	}
	// Check if this looks like a project
	return !d.IsProjectDirectory(dir)
}
