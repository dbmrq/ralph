package project

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewDetector(t *testing.T) {
	d := NewDetector()
	if d == nil {
		t.Fatal("NewDetector returned nil")
	}
	if len(d.Markers) == 0 {
		t.Error("Detector should have default markers")
	}
}

func TestDetectProject(t *testing.T) {
	// Create temp directory structure
	tmpDir := t.TempDir()

	tests := []struct {
		name        string
		setup       func(dir string)
		wantIsGit   bool
		wantType    string
		wantMarkers []string
		wantNil     bool
	}{
		{
			name:    "empty directory",
			setup:   func(_ string) {},
			wantNil: true,
		},
		{
			name: "go project",
			setup: func(dir string) {
				_ = os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module test"), 0644)
			},
			wantType:    "go",
			wantMarkers: []string{"go.mod"},
		},
		{
			name: "node project",
			setup: func(dir string) {
				_ = os.WriteFile(filepath.Join(dir, "package.json"), []byte("{}"), 0644)
			},
			wantType:    "node",
			wantMarkers: []string{"package.json"},
		},
		{
			name: "python project",
			setup: func(dir string) {
				_ = os.WriteFile(filepath.Join(dir, "pyproject.toml"), []byte(""), 0644)
			},
			wantType:    "python",
			wantMarkers: []string{"pyproject.toml"},
		},
		{
			name: "rust project",
			setup: func(dir string) {
				_ = os.WriteFile(filepath.Join(dir, "Cargo.toml"), []byte(""), 0644)
			},
			wantType:    "rust",
			wantMarkers: []string{"Cargo.toml"},
		},
		{
			name: "git repo",
			setup: func(dir string) {
				_ = os.Mkdir(filepath.Join(dir, ".git"), 0755)
			},
			wantIsGit:   true,
			wantMarkers: []string{".git"},
		},
		{
			name: "git + go project",
			setup: func(dir string) {
				_ = os.Mkdir(filepath.Join(dir, ".git"), 0755)
				_ = os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module test"), 0644)
			},
			wantIsGit:   true,
			wantType:    "go",
			wantMarkers: []string{".git", "go.mod"},
		},
		{
			name: "ralph project",
			setup: func(dir string) {
				_ = os.Mkdir(filepath.Join(dir, ".ralph"), 0755)
			},
			wantMarkers: []string{".ralph"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a subdirectory for this test
			testDir := filepath.Join(tmpDir, tt.name)
			if err := os.Mkdir(testDir, 0755); err != nil {
				t.Fatal(err)
			}

			tt.setup(testDir)

			d := NewDetector()
			proj, err := d.DetectProject(testDir)
			if err != nil {
				t.Fatalf("DetectProject error: %v", err)
			}

			if tt.wantNil {
				if proj != nil {
					t.Error("expected nil project")
				}
				return
			}

			if proj == nil {
				t.Fatal("expected non-nil project")
			}

			if proj.IsGitRepo != tt.wantIsGit {
				t.Errorf("IsGitRepo = %v, want %v", proj.IsGitRepo, tt.wantIsGit)
			}
			if proj.ProjectType != tt.wantType {
				t.Errorf("ProjectType = %q, want %q", proj.ProjectType, tt.wantType)
			}
			// Check that expected markers are present
			for _, m := range tt.wantMarkers {
				found := false
				for _, pm := range proj.Markers {
					if pm == m {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("marker %q not found in %v", m, proj.Markers)
				}
			}
		})
	}
}

func TestDetectProject_InvalidPath(t *testing.T) {
	d := NewDetector()

	// Non-existent path
	_, err := d.DetectProject("/nonexistent/path")
	if err == nil {
		t.Error("expected error for non-existent path")
	}

	// File instead of directory
	tmpFile := filepath.Join(t.TempDir(), "file.txt")
	_ = os.WriteFile(tmpFile, []byte("test"), 0644)
	_, err = d.DetectProject(tmpFile)
	if err == nil {
		t.Error("expected error for file path")
	}
}

func TestIsProjectDirectory(t *testing.T) {
	d := NewDetector()
	tmpDir := t.TempDir()

	// Empty directory is not a project
	if d.IsProjectDirectory(tmpDir) {
		t.Error("empty directory should not be a project")
	}

	// Add a marker
	_ = os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte("module test"), 0644)

	if !d.IsProjectDirectory(tmpDir) {
		t.Error("directory with go.mod should be a project")
	}
}

func TestIsHomeDirectory(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("could not get home directory")
	}

	if !IsHomeDirectory(home) {
		t.Error("home directory should be detected as home")
	}

	if IsHomeDirectory("/tmp") {
		t.Error("/tmp should not be detected as home")
	}

	if IsHomeDirectory("/nonexistent") {
		t.Error("nonexistent path should not be detected as home")
	}
}

func TestIsRootDirectory(t *testing.T) {
	if !IsRootDirectory("/") {
		t.Error("/ should be detected as root")
	}

	if IsRootDirectory("/tmp") {
		t.Error("/tmp should not be detected as root")
	}
}

func TestShouldPromptForDirectory(t *testing.T) {
	d := NewDetector()

	// Home directory should prompt
	home, err := os.UserHomeDir()
	if err == nil {
		if !d.ShouldPromptForDirectory(home) {
			t.Error("home directory should prompt for directory")
		}
	}

	// Root should prompt
	if !d.ShouldPromptForDirectory("/") {
		t.Error("root directory should prompt for directory")
	}

	// Project directory should not prompt
	tmpDir := t.TempDir()
	_ = os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte("module test"), 0644)
	if d.ShouldPromptForDirectory(tmpDir) {
		t.Error("project directory should not prompt")
	}
}

func TestContainsGlob(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"file.txt", false},
		{"*.txt", true},
		{"file?.txt", true},
		{"[abc]", true},
		{"normal/path", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := containsGlob(tt.input)
			if got != tt.want {
				t.Errorf("containsGlob(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestDetectProject_WithGlobMarkers(t *testing.T) {
	tmpDir := t.TempDir()

	// Create an .xcodeproj directory (glob pattern in markers)
	xcodeDir := filepath.Join(tmpDir, "MyApp.xcodeproj")
	if err := os.Mkdir(xcodeDir, 0755); err != nil {
		t.Fatal(err)
	}

	d := NewDetector()
	proj, err := d.DetectProject(tmpDir)
	if err != nil {
		t.Fatalf("DetectProject error: %v", err)
	}

	if proj == nil {
		t.Fatal("expected non-nil project for xcode project")
	}

	if proj.ProjectType != "xcode" {
		t.Errorf("ProjectType = %q, want %q", proj.ProjectType, "xcode")
	}
}
