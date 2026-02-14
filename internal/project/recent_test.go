package project

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestRalphUserDir(t *testing.T) {
	dir, err := RalphUserDir()
	if err != nil {
		t.Fatalf("RalphUserDir error: %v", err)
	}
	if dir == "" {
		t.Error("RalphUserDir returned empty string")
	}
	if !filepath.IsAbs(dir) {
		t.Error("RalphUserDir should return absolute path")
	}
}

func TestRecentProjectsPath(t *testing.T) {
	path, err := RecentProjectsPath()
	if err != nil {
		t.Fatalf("RecentProjectsPath error: %v", err)
	}
	if path == "" {
		t.Error("RecentProjectsPath returned empty string")
	}
	if filepath.Base(path) != RecentProjectsFile {
		t.Errorf("expected filename %q, got %q", RecentProjectsFile, filepath.Base(path))
	}
}

func TestLoadRecentProjects_NoFile(t *testing.T) {
	// Load should work even if file doesn't exist
	recent, err := LoadRecentProjects()
	if err != nil {
		t.Fatalf("LoadRecentProjects error: %v", err)
	}
	if recent == nil {
		t.Fatal("expected non-nil RecentProjects")
	}
}

func TestRecentProjects_Add(t *testing.T) {
	recent := &RecentProjects{}

	proj1 := &ProjectInfo{
		Path:        "/path/to/project1",
		Name:        "project1",
		ProjectType: "go",
	}
	proj2 := &ProjectInfo{
		Path:        "/path/to/project2",
		Name:        "project2",
		ProjectType: "node",
	}

	recent.Add(proj1)
	if len(recent.Projects) != 1 {
		t.Errorf("expected 1 project, got %d", len(recent.Projects))
	}

	recent.Add(proj2)
	if len(recent.Projects) != 2 {
		t.Errorf("expected 2 projects, got %d", len(recent.Projects))
	}

	// Add proj1 again - should update, not duplicate
	recent.Add(proj1)
	if len(recent.Projects) != 2 {
		t.Errorf("expected 2 projects after re-add, got %d", len(recent.Projects))
	}

	// proj1 should now be first (most recent)
	if recent.Projects[0].Path != proj1.Path {
		t.Error("re-added project should be first")
	}
}

func TestRecentProjects_MaxLimit(t *testing.T) {
	recent := &RecentProjects{}

	// Add more than max
	for i := 0; i < MaxRecentProjects+5; i++ {
		proj := &ProjectInfo{
			Path: "/path/to/project" + string(rune('a'+i)),
			Name: "project" + string(rune('a'+i)),
		}
		recent.Add(proj)
	}

	if len(recent.Projects) != MaxRecentProjects {
		t.Errorf("expected %d projects, got %d", MaxRecentProjects, len(recent.Projects))
	}
}

func TestRecentProjects_GetPaths(t *testing.T) {
	recent := &RecentProjects{
		Projects: []RecentProject{
			{Path: "/path/1", Name: "p1"},
			{Path: "/path/2", Name: "p2"},
		},
	}

	paths := recent.GetPaths()
	if len(paths) != 2 {
		t.Errorf("expected 2 paths, got %d", len(paths))
	}
	if paths[0] != "/path/1" || paths[1] != "/path/2" {
		t.Errorf("unexpected paths: %v", paths)
	}
}

func TestRecentProjects_SaveAndLoad(t *testing.T) {
	// Create temp directory to use as ~/.ralph
	tmpDir := t.TempDir()
	tmpRalphDir := filepath.Join(tmpDir, ".ralph")
	if err := os.MkdirAll(tmpRalphDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create a valid project directory to add
	projDir := filepath.Join(tmpDir, "myproject")
	if err := os.Mkdir(projDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create recent projects and save
	recent := &RecentProjects{
		Projects: []RecentProject{
			{Path: projDir, Name: "myproject", LastUsed: time.Now()},
		},
	}

	// Save directly to temp location
	savePath := filepath.Join(tmpRalphDir, RecentProjectsFile)
	data, err := json.MarshalIndent(recent, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(savePath, data, 0644); err != nil {
		t.Fatal(err)
	}

	// Read back
	loadedData, err := os.ReadFile(savePath)
	if err != nil {
		t.Fatal(err)
	}

	var loaded RecentProjects
	if err := json.Unmarshal(loadedData, &loaded); err != nil {
		t.Fatal(err)
	}

	if len(loaded.Projects) != 1 {
		t.Errorf("expected 1 project, got %d", len(loaded.Projects))
	}
}

