// Package build provides build and test verification logic for ralph.
package build

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/wexinc/ralph/internal/agent"
)

// CacheFile is the default path for cached project analysis.
const CacheFile = ".ralph/project_analysis.json"

// CacheMaxAge is the maximum age for cached analysis before re-analysis is suggested.
const CacheMaxAge = 24 * time.Hour

// AnalysisProgress is a callback for reporting analysis progress.
type AnalysisProgress func(status string)

// CachedAnalysis wraps ProjectAnalysis with metadata for caching.
type CachedAnalysis struct {
	Analysis   *ProjectAnalysis `json:"analysis"`
	CachedAt   time.Time        `json:"cached_at"`
	AgentName  string           `json:"agent_name"`
	AgentModel string           `json:"agent_model"`
}

// ProjectAnalyzer runs AI-driven project analysis.
type ProjectAnalyzer struct {
	// ProjectDir is the root directory of the project.
	ProjectDir string
	// Agent is the AI agent to use for analysis.
	Agent agent.Agent
	// Model is the model to use (optional, uses agent default if empty).
	Model string
	// OnProgress is called with status updates during analysis.
	OnProgress AnalysisProgress
	// LogWriter receives real-time agent output (optional).
	LogWriter io.Writer
}

// NewProjectAnalyzer creates a new ProjectAnalyzer.
func NewProjectAnalyzer(projectDir string, ag agent.Agent) *ProjectAnalyzer {
	return &ProjectAnalyzer{
		ProjectDir: projectDir,
		Agent:      ag,
		OnProgress: func(status string) {}, // noop by default
	}
}

// Analyze runs the AI agent to analyze the project and returns ProjectAnalysis.
// This is the main entry point for AI-driven project detection.
func (p *ProjectAnalyzer) Analyze(ctx context.Context) (*ProjectAnalysis, error) {
	p.report("Running AI analysis...")

	prompt := p.buildAnalysisPrompt()

	opts := agent.RunOptions{
		Model:     p.Model,
		WorkDir:   p.ProjectDir,
		Timeout:   5 * time.Minute, // Analysis should be quick
		Force:     true,
		LogWriter: p.LogWriter,
	}

	p.report("Examining project structure...")
	result, err := p.Agent.Run(ctx, prompt, opts)
	if err != nil {
		return nil, fmt.Errorf("agent failed: %w", err)
	}

	if result.ExitCode != 0 {
		return nil, fmt.Errorf("agent exited with code %d: %s", result.ExitCode, result.Error)
	}

	p.report("Parsing analysis results...")
	analysis, err := parseAnalysisOutput(result.Output)
	if err != nil {
		return nil, fmt.Errorf("failed to parse analysis: %w", err)
	}

	p.report("Analysis complete")
	return analysis, nil
}

// AnalyzeWithFallback runs analysis and falls back to minimal defaults if it fails.
// This ensures the loop can continue even if AI analysis is unavailable.
func (p *ProjectAnalyzer) AnalyzeWithFallback(ctx context.Context) (*ProjectAnalysis, error) {
	analysis, err := p.Analyze(ctx)
	if err != nil {
		p.report(fmt.Sprintf("Analysis failed, using fallback: %s", err))
		return p.fallbackAnalysis(), nil
	}
	return analysis, nil
}

// LoadCached loads cached analysis from disk if available and fresh.
func (p *ProjectAnalyzer) LoadCached() (*CachedAnalysis, error) {
	cachePath := filepath.Join(p.ProjectDir, CacheFile)
	data, err := os.ReadFile(cachePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // No cache, not an error
		}
		return nil, fmt.Errorf("failed to read cache: %w", err)
	}

	var cached CachedAnalysis
	if err := json.Unmarshal(data, &cached); err != nil {
		return nil, fmt.Errorf("failed to parse cache: %w", err)
	}

	return &cached, nil
}

// IsCacheFresh returns true if the cached analysis is recent enough.
func (c *CachedAnalysis) IsCacheFresh() bool {
	return time.Since(c.CachedAt) < CacheMaxAge
}

// SaveCache saves the analysis to the cache file.
func (p *ProjectAnalyzer) SaveCache(analysis *ProjectAnalysis) error {
	cached := CachedAnalysis{
		Analysis:   analysis,
		CachedAt:   time.Now(),
		AgentName:  p.Agent.Name(),
		AgentModel: p.Model,
	}

	data, err := json.MarshalIndent(cached, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal cache: %w", err)
	}

	cachePath := filepath.Join(p.ProjectDir, CacheFile)
	// Ensure .ralph directory exists
	if err := os.MkdirAll(filepath.Dir(cachePath), 0755); err != nil {
		return fmt.Errorf("failed to create cache directory: %w", err)
	}

	if err := os.WriteFile(cachePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write cache: %w", err)
	}

	return nil
}

// report calls the progress callback if set.
func (p *ProjectAnalyzer) report(status string) {
	if p.OnProgress != nil {
		p.OnProgress(status)
	}
}

// buildAnalysisPrompt creates the prompt for the AI agent.
func (p *ProjectAnalyzer) buildAnalysisPrompt() string {
	return analysisPrompt
}

// fallbackAnalysis returns minimal defaults when AI analysis fails.
func (p *ProjectAnalyzer) fallbackAnalysis() *ProjectAnalysis {
	return &ProjectAnalysis{
		ProjectType:  "unknown",
		Languages:    []string{},
		IsGreenfield: true,
		IsMonorepo:   false,
		Build: BuildAnalysis{
			Ready:   false,
			Command: nil,
			Reason:  "AI analysis unavailable, using fallback",
		},
		Test: TestAnalysis{
			Ready:        false,
			Command:      nil,
			HasTestFiles: false,
			Reason:       "AI analysis unavailable, using fallback",
		},
		Lint: LintAnalysis{
			Command:   nil,
			Available: false,
		},
		Dependencies: DependencyAnalysis{
			Manager:   "",
			Installed: false,
		},
		TaskList: TaskListAnalysis{
			Detected:  false,
			Path:      "",
			Format:    "",
			TaskCount: 0,
		},
		ProjectContext: "Project analysis unavailable. Manual configuration may be required.",
	}
}

// analysisPrompt is the prompt sent to the AI agent for project analysis.
const analysisPrompt = `Analyze this project and return a JSON object with the following structure.
Return ONLY the JSON object, no other text, no markdown code blocks, no explanations.

{
  "project_type": "go|node|python|rust|java|mixed|unknown",
  "languages": ["go", "typescript"],
  "is_greenfield": true,
  "is_monorepo": false,
  "build": {
    "ready": false,
    "command": "go build ./...",
    "reason": "No source files yet"
  },
  "test": {
    "ready": false,
    "command": "go test ./...",
    "has_test_files": false,
    "reason": "No test files found"
  },
  "lint": {
    "command": "golangci-lint run ./...",
    "available": true
  },
  "dependencies": {
    "manager": "go mod",
    "installed": true
  },
  "task_list": {
    "detected": false,
    "path": "",
    "format": "",
    "task_count": 0
  },
  "project_context": "Brief description of the project..."
}

Instructions:
1. Examine the project structure (files, directories, config files)
2. Look for build system markers (go.mod, package.json, Cargo.toml, pyproject.toml, etc.)
3. Detect what commands would build/test this project
4. Determine if the project is in a "greenfield" state (no buildable code yet)
5. Look for existing task lists (TASKS.md, TODO.md, .ralph/tasks.json, etc.)
6. For "command" fields, use null if not detected, or the actual command string
7. Write a brief project_context describing the project type and structure
8. Return ONLY the JSON object`

// parseAnalysisOutput extracts ProjectAnalysis from agent output.
// It handles various output formats including raw JSON and JSON wrapped in text.
func parseAnalysisOutput(output string) (*ProjectAnalysis, error) {
	// Try to extract JSON from the output
	jsonStr := extractJSON(output)
	if jsonStr == "" {
		return nil, fmt.Errorf("no JSON found in agent output")
	}

	var analysis ProjectAnalysis
	if err := json.Unmarshal([]byte(jsonStr), &analysis); err != nil {
		return nil, fmt.Errorf("invalid JSON: %w", err)
	}

	// Validate required fields
	if analysis.ProjectType == "" {
		analysis.ProjectType = "unknown"
	}
	if analysis.Languages == nil {
		analysis.Languages = []string{}
	}

	return &analysis, nil
}

// extractJSON finds and extracts a JSON object from text.
// It handles cases where the JSON is wrapped in markdown code blocks or other text.
func extractJSON(text string) string {
	// First, try to find JSON object boundaries
	start := -1
	braceCount := 0
	inString := false
	escape := false

	for i, c := range text {
		if escape {
			escape = false
			continue
		}
		if c == '\\' && inString {
			escape = true
			continue
		}
		if c == '"' && !escape {
			inString = !inString
			continue
		}
		if inString {
			continue
		}
		if c == '{' {
			if start == -1 {
				start = i
			}
			braceCount++
		} else if c == '}' {
			braceCount--
			if braceCount == 0 && start != -1 {
				return text[start : i+1]
			}
		}
	}

	return ""
}

