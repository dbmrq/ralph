// Package version provides version checking and update functionality for ralph.
package version

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// GitHubRepo is the GitHub repository for ralph.
const GitHubRepo = "wexinc/ralph"

// ReleaseAPIURL is the GitHub API URL for latest release.
const ReleaseAPIURL = "https://api.github.com/repos/%s/releases/latest"

// Info contains version information about ralph.
type Info struct {
	Version string `json:"version"`
	Commit  string `json:"commit"`
	Date    string `json:"date"`
	GoVer   string `json:"go_version"`
	OS      string `json:"os"`
	Arch    string `json:"arch"`
}

// NewInfo creates a new Info from the build variables.
func NewInfo(version, commit, date string) *Info {
	return &Info{
		Version: version,
		Commit:  commit,
		Date:    date,
		GoVer:   runtime.Version(),
		OS:      runtime.GOOS,
		Arch:    runtime.GOARCH,
	}
}

// String returns a formatted version string.
func (i *Info) String() string {
	return fmt.Sprintf("ralph %s (commit: %s, built: %s)", i.Version, i.Commit, i.Date)
}

// FullString returns a detailed version string.
func (i *Info) FullString() string {
	return fmt.Sprintf(`ralph %s
  Commit:   %s
  Built:    %s
  Go:       %s
  OS/Arch:  %s/%s`, i.Version, i.Commit, i.Date, i.GoVer, i.OS, i.Arch)
}

// Release represents a GitHub release.
type Release struct {
	TagName     string `json:"tag_name"`
	Name        string `json:"name"`
	Body        string `json:"body"`
	PublishedAt string `json:"published_at"`
	HTMLURL     string `json:"html_url"`
}

// Checker checks for new versions.
type Checker struct {
	HTTPClient *http.Client
	Repo       string
}

// NewChecker creates a new version checker.
func NewChecker() *Checker {
	return &Checker{
		HTTPClient: &http.Client{Timeout: 10 * time.Second},
		Repo:       GitHubRepo,
	}
}

// GetLatestRelease fetches the latest release from GitHub.
func (c *Checker) GetLatestRelease(ctx context.Context) (*Release, error) {
	url := fmt.Sprintf(ReleaseAPIURL, c.Repo)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("User-Agent", "ralph-version-checker")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch release: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("GitHub API returned %d: %s", resp.StatusCode, string(body))
	}

	var release Release
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, fmt.Errorf("failed to decode release: %w", err)
	}

	return &release, nil
}

// CheckForUpdate compares current version with latest release.
// Returns the release if an update is available, nil if current.
func (c *Checker) CheckForUpdate(ctx context.Context, currentVersion string) (*Release, error) {
	release, err := c.GetLatestRelease(ctx)
	if err != nil {
		return nil, err
	}

	latestVersion := strings.TrimPrefix(release.TagName, "v")
	currentVersion = strings.TrimPrefix(currentVersion, "v")

	if CompareVersions(latestVersion, currentVersion) > 0 {
		return release, nil
	}

	return nil, nil
}

// CompareVersions compares two semantic version strings.
// Returns: 1 if a > b, -1 if a < b, 0 if equal.
func CompareVersions(a, b string) int {
	aParts := parseVersion(a)
	bParts := parseVersion(b)

	for i := 0; i < 3; i++ {
		if aParts[i] > bParts[i] {
			return 1
		}
		if aParts[i] < bParts[i] {
			return -1
		}
	}
	return 0
}

// parseVersion parses a version string into major, minor, patch integers.
func parseVersion(v string) [3]int {
	// Strip 'v' prefix if present
	v = strings.TrimPrefix(v, "v")
	parts := strings.Split(v, ".")
	var result [3]int
	for i := 0; i < 3 && i < len(parts); i++ {
		// Remove any pre-release suffix (e.g., "-rc1")
		part := strings.Split(parts[i], "-")[0]
		fmt.Sscanf(part, "%d", &result[i])
	}
	return result
}

// ProjectVersion stores version info for a project.
type ProjectVersion struct {
	RalphVersion string    `json:"ralph_version"`
	InitializedAt time.Time `json:"initialized_at"`
	LastRunAt     time.Time `json:"last_run_at,omitempty"`
}

// VersionFilePath is the path to the version file within .ralph.
const VersionFilePath = ".ralph/version.json"

// LoadProjectVersion loads the project version from .ralph/version.json.
func LoadProjectVersion(projectDir string) (*ProjectVersion, error) {
	path := filepath.Join(projectDir, VersionFilePath)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var pv ProjectVersion
	if err := json.Unmarshal(data, &pv); err != nil {
		return nil, fmt.Errorf("failed to parse version.json: %w", err)
	}

	return &pv, nil
}

// SaveProjectVersion saves the project version to .ralph/version.json.
func SaveProjectVersion(projectDir string, pv *ProjectVersion) error {
	path := filepath.Join(projectDir, VersionFilePath)
	data, err := json.MarshalIndent(pv, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal version.json: %w", err)
	}

	return os.WriteFile(path, data, 0644)
}

// UpdateLastRun updates the last_run_at timestamp.
func UpdateLastRun(projectDir, version string) error {
	pv, err := LoadProjectVersion(projectDir)
	if err != nil {
		// Create new if doesn't exist
		pv = &ProjectVersion{
			RalphVersion:  version,
			InitializedAt: time.Now(),
		}
	}
	pv.LastRunAt = time.Now()
	pv.RalphVersion = version
	return SaveProjectVersion(projectDir, pv)
}

