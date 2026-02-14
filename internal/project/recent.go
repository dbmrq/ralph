package project

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"time"
)

// RecentProject represents a recently used project.
type RecentProject struct {
	// Path is the absolute path to the project.
	Path string `json:"path"`
	// Name is the project name.
	Name string `json:"name"`
	// LastUsed is when the project was last used.
	LastUsed time.Time `json:"last_used"`
	// ProjectType is the detected project type.
	ProjectType string `json:"project_type,omitempty"`
}

// RecentProjects manages the list of recently used projects.
type RecentProjects struct {
	// Projects is the list of recent projects, sorted by last used.
	Projects []RecentProject `json:"projects"`
}

const (
	// MaxRecentProjects is the maximum number of recent projects to store.
	MaxRecentProjects = 10
	// RecentProjectsFile is the file name for storing recent projects.
	RecentProjectsFile = "recent.json"
)

// RalphUserDir returns the path to the user's ralph directory (~/.ralph).
func RalphUserDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".ralph"), nil
}

// RecentProjectsPath returns the path to the recent projects file.
func RecentProjectsPath() (string, error) {
	userDir, err := RalphUserDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(userDir, RecentProjectsFile), nil
}

// LoadRecentProjects loads the recent projects list from disk.
func LoadRecentProjects() (*RecentProjects, error) {
	path, err := RecentProjectsPath()
	if err != nil {
		return &RecentProjects{}, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &RecentProjects{}, nil
		}
		return nil, err
	}

	var recent RecentProjects
	if err := json.Unmarshal(data, &recent); err != nil {
		// If file is corrupted, start fresh
		return &RecentProjects{}, nil
	}

	// Filter out non-existent projects
	valid := make([]RecentProject, 0, len(recent.Projects))
	for _, p := range recent.Projects {
		if _, err := os.Stat(p.Path); err == nil {
			valid = append(valid, p)
		}
	}
	recent.Projects = valid

	return &recent, nil
}

// Save saves the recent projects list to disk.
func (r *RecentProjects) Save() error {
	userDir, err := RalphUserDir()
	if err != nil {
		return err
	}

	// Create ~/.ralph if it doesn't exist
	if err := os.MkdirAll(userDir, 0755); err != nil {
		return err
	}

	path, err := RecentProjectsPath()
	if err != nil {
		return err
	}

	data, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}

// Add adds or updates a project in the recent list.
func (r *RecentProjects) Add(project *ProjectInfo) {
	now := time.Now()

	// Check if project already exists
	for i := range r.Projects {
		if r.Projects[i].Path == project.Path {
			// Update existing entry
			r.Projects[i].LastUsed = now
			r.Projects[i].Name = project.Name
			r.Projects[i].ProjectType = project.ProjectType
			r.sortAndTrim()
			return
		}
	}

	// Add new entry
	r.Projects = append(r.Projects, RecentProject{
		Path:        project.Path,
		Name:        project.Name,
		LastUsed:    now,
		ProjectType: project.ProjectType,
	})
	r.sortAndTrim()
}

// sortAndTrim sorts projects by last used (newest first) and trims to max.
func (r *RecentProjects) sortAndTrim() {
	sort.Slice(r.Projects, func(i, j int) bool {
		return r.Projects[i].LastUsed.After(r.Projects[j].LastUsed)
	})
	if len(r.Projects) > MaxRecentProjects {
		r.Projects = r.Projects[:MaxRecentProjects]
	}
}

// GetPaths returns the paths of all recent projects.
func (r *RecentProjects) GetPaths() []string {
	paths := make([]string, len(r.Projects))
	for i, p := range r.Projects {
		paths[i] = p.Path
	}
	return paths
}

