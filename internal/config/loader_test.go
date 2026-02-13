package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLoad_MissingFile(t *testing.T) {
	_, err := Load("nonexistent/config.yaml")
	if err == nil {
		t.Fatal("expected error for missing file")
	}

	loadErr, ok := err.(*LoadError)
	if !ok {
		t.Fatalf("expected *LoadError, got %T", err)
	}
	if loadErr.Path != "nonexistent/config.yaml" {
		t.Errorf("expected path 'nonexistent/config.yaml', got %q", loadErr.Path)
	}
	if loadErr.Message != "config file not found" {
		t.Errorf("expected message 'config file not found', got %q", loadErr.Message)
	}
}

func TestLoad_ValidConfig(t *testing.T) {
	// Create a temporary directory and config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	configContent := `
agent:
  default: cursor
  model: claude-opus

timeout:
  active: 1h
  stuck: 15m

git:
  auto_commit: true
  commit_prefix: "[test]"
  push: false

build:
  command: "go build ./..."
  bootstrap_detection: auto

test:
  command: "go test ./..."
  mode: tdd
  baseline_file: .ralph/test_baseline.json
  baseline_scope: global
`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	// Verify agent settings
	if cfg.Agent.Default != "cursor" {
		t.Errorf("expected agent.default 'cursor', got %q", cfg.Agent.Default)
	}
	if cfg.Agent.Model != "claude-opus" {
		t.Errorf("expected agent.model 'claude-opus', got %q", cfg.Agent.Model)
	}

	// Verify timeout settings
	if cfg.Timeout.Active != 1*time.Hour {
		t.Errorf("expected timeout.active 1h, got %v", cfg.Timeout.Active)
	}
	if cfg.Timeout.Stuck != 15*time.Minute {
		t.Errorf("expected timeout.stuck 15m, got %v", cfg.Timeout.Stuck)
	}

	// Verify git settings
	if cfg.Git.AutoCommit != true {
		t.Error("expected git.auto_commit to be true")
	}
	if cfg.Git.CommitPrefix != "[test]" {
		t.Errorf("expected git.commit_prefix '[test]', got %q", cfg.Git.CommitPrefix)
	}
	if cfg.Git.Push != false {
		t.Error("expected git.push to be false")
	}

	// Verify build settings
	if cfg.Build.Command != "go build ./..." {
		t.Errorf("expected build.command 'go build ./...', got %q", cfg.Build.Command)
	}
	if cfg.Build.BootstrapDetection != BootstrapDetectionAuto {
		t.Errorf("expected build.bootstrap_detection 'auto', got %q", cfg.Build.BootstrapDetection)
	}

	// Verify test settings
	if cfg.Test.Command != "go test ./..." {
		t.Errorf("expected test.command 'go test ./...', got %q", cfg.Test.Command)
	}
	if cfg.Test.Mode != TestModeTDD {
		t.Errorf("expected test.mode 'tdd', got %q", cfg.Test.Mode)
	}
}

func TestLoad_DefaultsApplied(t *testing.T) {
	// Create a minimal config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	// Minimal config - just agent settings
	configContent := `
agent:
  default: auggie
`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	// Verify agent settings from file
	if cfg.Agent.Default != "auggie" {
		t.Errorf("expected agent.default 'auggie', got %q", cfg.Agent.Default)
	}

	// Verify defaults were applied
	if cfg.Timeout.Active != DefaultActiveTimeout {
		t.Errorf("expected default timeout.active %v, got %v", DefaultActiveTimeout, cfg.Timeout.Active)
	}
	if cfg.Timeout.Stuck != DefaultStuckTimeout {
		t.Errorf("expected default timeout.stuck %v, got %v", DefaultStuckTimeout, cfg.Timeout.Stuck)
	}
	if cfg.Git.CommitPrefix != DefaultCommitPrefix {
		t.Errorf("expected default git.commit_prefix %q, got %q", DefaultCommitPrefix, cfg.Git.CommitPrefix)
	}
	if cfg.Test.Mode != TestModeGate {
		t.Errorf("expected default test.mode 'gate', got %q", cfg.Test.Mode)
	}
}

func TestLoad_EnvOverrides(t *testing.T) {
	// Create a config file with defaults
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	configContent := `
agent:
  default: cursor
  model: claude-opus

timeout:
  active: 1h
  stuck: 15m

git:
  auto_commit: true
  commit_prefix: "[test]"
`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	// Set environment variables to override
	envVars := map[string]string{
		"RALPH_AGENT_DEFAULT":   "auggie",
		"RALPH_AGENT_MODEL":     "gpt-4",
		"RALPH_TIMEOUT_ACTIVE":  "3h",
		"RALPH_TIMEOUT_STUCK":   "45m",
		"RALPH_GIT_AUTO_COMMIT": "false",
		"RALPH_GIT_PUSH":        "true",
		"RALPH_TEST_MODE":       "tdd",
	}

	// Set env vars and defer cleanup
	for k, v := range envVars {
		os.Setenv(k, v)
	}
	defer func() {
		for k := range envVars {
			os.Unsetenv(k)
		}
	}()

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	// Verify environment overrides
	if cfg.Agent.Default != "auggie" {
		t.Errorf("expected agent.default 'auggie' from env, got %q", cfg.Agent.Default)
	}
	if cfg.Agent.Model != "gpt-4" {
		t.Errorf("expected agent.model 'gpt-4' from env, got %q", cfg.Agent.Model)
	}
	if cfg.Timeout.Active != 3*time.Hour {
		t.Errorf("expected timeout.active 3h from env, got %v", cfg.Timeout.Active)
	}
	if cfg.Timeout.Stuck != 45*time.Minute {
		t.Errorf("expected timeout.stuck 45m from env, got %v", cfg.Timeout.Stuck)
	}
	if cfg.Git.AutoCommit != false {
		t.Error("expected git.auto_commit false from env")
	}
	if cfg.Git.Push != true {
		t.Error("expected git.push true from env")
	}
	if cfg.Test.Mode != TestModeTDD {
		t.Errorf("expected test.mode 'tdd' from env, got %q", cfg.Test.Mode)
	}
}

func TestLoad_ValidationError(t *testing.T) {
	// Create a config file with invalid values
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	configContent := `
test:
  mode: invalid_mode
`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	_, err := Load(configPath)
	if err == nil {
		t.Fatal("expected validation error")
	}

	loadErr, ok := err.(*LoadError)
	if !ok {
		t.Fatalf("expected *LoadError, got %T", err)
	}
	if loadErr.Message != "configuration validation failed" {
		t.Errorf("expected message 'configuration validation failed', got %q", loadErr.Message)
	}
}

func TestLoad_InvalidYAML(t *testing.T) {
	// Create a config file with invalid YAML
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	configContent := `
agent:
  default: cursor
  model: [invalid yaml
`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	_, err := Load(configPath)
	if err == nil {
		t.Fatal("expected error for invalid YAML")
	}

	loadErr, ok := err.(*LoadError)
	if !ok {
		t.Fatalf("expected *LoadError, got %T", err)
	}
	if loadErr.Message != "failed to read config file" {
		t.Errorf("expected message 'failed to read config file', got %q", loadErr.Message)
	}
}

func TestLoadFromDir(t *testing.T) {
	// Create a .ralph directory structure
	tmpDir := t.TempDir()
	ralphDir := filepath.Join(tmpDir, ".ralph")
	if err := os.MkdirAll(ralphDir, 0755); err != nil {
		t.Fatalf("failed to create .ralph directory: %v", err)
	}

	configPath := filepath.Join(ralphDir, "config.yaml")
	configContent := `
agent:
  default: cursor
`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	cfg, err := LoadFromDir(tmpDir)
	if err != nil {
		t.Fatalf("failed to load config from dir: %v", err)
	}

	if cfg.Agent.Default != "cursor" {
		t.Errorf("expected agent.default 'cursor', got %q", cfg.Agent.Default)
	}
}

func TestParseBool(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"true", true},
		{"TRUE", true},
		{"True", true},
		{"1", true},
		{"yes", true},
		{"YES", true},
		{"false", false},
		{"FALSE", false},
		{"0", false},
		{"no", false},
		{"", false},
		{"invalid", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := parseBool(tt.input)
			if result != tt.expected {
				t.Errorf("parseBool(%q) = %v, expected %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestLoadError_Error(t *testing.T) {
	t.Run("with underlying error", func(t *testing.T) {
		err := &LoadError{
			Path:    "config.yaml",
			Message: "failed to parse",
			Err:     os.ErrNotExist,
		}
		expected := "config.yaml: failed to parse: file does not exist"
		if err.Error() != expected {
			t.Errorf("expected %q, got %q", expected, err.Error())
		}
	})

	t.Run("without underlying error", func(t *testing.T) {
		err := &LoadError{
			Path:    "config.yaml",
			Message: "invalid format",
		}
		expected := "config.yaml: invalid format"
		if err.Error() != expected {
			t.Errorf("expected %q, got %q", expected, err.Error())
		}
	})
}

func TestLoadError_Unwrap(t *testing.T) {
	underlyingErr := os.ErrNotExist
	err := &LoadError{
		Path:    "config.yaml",
		Message: "failed to read",
		Err:     underlyingErr,
	}

	if err.Unwrap() != underlyingErr {
		t.Error("Unwrap should return the underlying error")
	}
}

func TestLoad_HooksConfig(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	configContent := `
hooks:
  pre_task:
    - type: shell
      command: "echo starting"
      on_failure: warn_continue
    - type: agent
      command: "review the task"
      model: claude-opus
      on_failure: skip_task
  post_task:
    - type: shell
      command: "echo done"
      on_failure: abort_loop
`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	// Verify pre_task hooks
	if len(cfg.Hooks.PreTask) != 2 {
		t.Fatalf("expected 2 pre_task hooks, got %d", len(cfg.Hooks.PreTask))
	}

	hook1 := cfg.Hooks.PreTask[0]
	if hook1.Type != HookTypeShell {
		t.Errorf("expected hook type 'shell', got %q", hook1.Type)
	}
	if hook1.Command != "echo starting" {
		t.Errorf("expected hook command 'echo starting', got %q", hook1.Command)
	}
	if hook1.OnFailure != FailureModeWarnContinue {
		t.Errorf("expected on_failure 'warn_continue', got %q", hook1.OnFailure)
	}

	hook2 := cfg.Hooks.PreTask[1]
	if hook2.Type != HookTypeAgent {
		t.Errorf("expected hook type 'agent', got %q", hook2.Type)
	}
	if hook2.Model != "claude-opus" {
		t.Errorf("expected model 'claude-opus', got %q", hook2.Model)
	}

	// Verify post_task hooks
	if len(cfg.Hooks.PostTask) != 1 {
		t.Fatalf("expected 1 post_task hook, got %d", len(cfg.Hooks.PostTask))
	}

	postHook := cfg.Hooks.PostTask[0]
	if postHook.OnFailure != FailureModeAbortLoop {
		t.Errorf("expected on_failure 'abort_loop', got %q", postHook.OnFailure)
	}
}

// TEST-001: Additional comprehensive tests for configuration loading

func TestLoad_CustomAgents(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	configContent := `
agent:
  default: my-custom-agent
  custom:
    - name: my-custom-agent
      description: "A custom agent for testing"
      command: my-agent-cli
      detection_method: command
      detection_value: my-agent-cli
      model_list_command: "my-agent-cli models"
      default_model: gpt-4
      args:
        - "--verbose"
        - "--format=json"
`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	if len(cfg.Agent.Custom) != 1 {
		t.Fatalf("expected 1 custom agent, got %d", len(cfg.Agent.Custom))
	}

	agent := cfg.Agent.Custom[0]
	if agent.Name != "my-custom-agent" {
		t.Errorf("expected name 'my-custom-agent', got %q", agent.Name)
	}
	if agent.Description != "A custom agent for testing" {
		t.Errorf("expected description 'A custom agent for testing', got %q", agent.Description)
	}
	if agent.Command != "my-agent-cli" {
		t.Errorf("expected command 'my-agent-cli', got %q", agent.Command)
	}
	if agent.DetectionMethod != DetectionMethodCommand {
		t.Errorf("expected detection_method 'command', got %q", agent.DetectionMethod)
	}
	if agent.DetectionValue != "my-agent-cli" {
		t.Errorf("expected detection_value 'my-agent-cli', got %q", agent.DetectionValue)
	}
	if agent.ModelListCommand != "my-agent-cli models" {
		t.Errorf("expected model_list_command 'my-agent-cli models', got %q", agent.ModelListCommand)
	}
	if agent.DefaultModel != "gpt-4" {
		t.Errorf("expected default_model 'gpt-4', got %q", agent.DefaultModel)
	}
	if len(agent.Args) != 2 {
		t.Fatalf("expected 2 args, got %d", len(agent.Args))
	}
	if agent.Args[0] != "--verbose" || agent.Args[1] != "--format=json" {
		t.Errorf("unexpected args: %v", agent.Args)
	}
}

func TestLoad_EnvOverrides_BuildAndTestSettings(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	configContent := `
agent:
  default: cursor
build:
  command: "make build"
  bootstrap_detection: auto
test:
  command: "make test"
  baseline_file: .ralph/baseline.json
  baseline_scope: global
`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	// Set environment variables to override
	envVars := map[string]string{
		"RALPH_BUILD_COMMAND":             "go build ./...",
		"RALPH_BUILD_BOOTSTRAP_DETECTION": "disabled",
		"RALPH_BUILD_BOOTSTRAP_CHECK":     "test -f go.mod",
		"RALPH_TEST_COMMAND":              "go test ./...",
		"RALPH_TEST_BASELINE_FILE":        ".ralph/my_baseline.json",
		"RALPH_TEST_BASELINE_SCOPE":       "session",
	}

	for k, v := range envVars {
		os.Setenv(k, v)
	}
	defer func() {
		for k := range envVars {
			os.Unsetenv(k)
		}
	}()

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	// Verify build settings from environment
	if cfg.Build.Command != "go build ./..." {
		t.Errorf("expected build.command 'go build ./...' from env, got %q", cfg.Build.Command)
	}
	if cfg.Build.BootstrapDetection != BootstrapDetectionDisabled {
		t.Errorf("expected build.bootstrap_detection 'disabled' from env, got %q", cfg.Build.BootstrapDetection)
	}
	if cfg.Build.BootstrapCheck != "test -f go.mod" {
		t.Errorf("expected build.bootstrap_check 'test -f go.mod' from env, got %q", cfg.Build.BootstrapCheck)
	}

	// Verify test settings from environment
	if cfg.Test.Command != "go test ./..." {
		t.Errorf("expected test.command 'go test ./...' from env, got %q", cfg.Test.Command)
	}
	if cfg.Test.BaselineFile != ".ralph/my_baseline.json" {
		t.Errorf("expected test.baseline_file '.ralph/my_baseline.json' from env, got %q", cfg.Test.BaselineFile)
	}
	if cfg.Test.BaselineScope != BaselineScopeSession {
		t.Errorf("expected test.baseline_scope 'session' from env, got %q", cfg.Test.BaselineScope)
	}
}

func TestLoad_EnvOverrides_GitCommitPrefix(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	configContent := `
agent:
  default: cursor
git:
  commit_prefix: "[original]"
`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	os.Setenv("RALPH_GIT_COMMIT_PREFIX", "[overridden]")
	defer os.Unsetenv("RALPH_GIT_COMMIT_PREFIX")

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	if cfg.Git.CommitPrefix != "[overridden]" {
		t.Errorf("expected git.commit_prefix '[overridden]' from env, got %q", cfg.Git.CommitPrefix)
	}
}

func TestLoad_EnvOverrides_DurationParsing(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	configContent := `
agent:
  default: cursor
timeout:
  active: 1h
  stuck: 15m
`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	// Set valid duration in env - should override file values
	os.Setenv("RALPH_TIMEOUT_ACTIVE", "4h")
	os.Setenv("RALPH_TIMEOUT_STUCK", "20m")
	defer func() {
		os.Unsetenv("RALPH_TIMEOUT_ACTIVE")
		os.Unsetenv("RALPH_TIMEOUT_STUCK")
	}()

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	// Environment values should override file values
	if cfg.Timeout.Active != 4*time.Hour {
		t.Errorf("expected timeout.active 4h from env, got %v", cfg.Timeout.Active)
	}
	if cfg.Timeout.Stuck != 20*time.Minute {
		t.Errorf("expected timeout.stuck 20m from env, got %v", cfg.Timeout.Stuck)
	}
}

func TestSave_BasicConfig(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	cfg := &Config{
		Agent: AgentConfig{
			Default: "cursor",
			Model:   "claude-opus",
		},
		Timeout: TimeoutConfig{
			Active: 2 * time.Hour,
			Stuck:  30 * time.Minute,
		},
		Git: GitConfig{
			AutoCommit:   true,
			CommitPrefix: "[ralph]",
			Push:         false,
		},
	}

	err := Save(cfg, configPath)
	if err != nil {
		t.Fatalf("failed to save config: %v", err)
	}

	// Verify file was created
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Fatal("config file was not created")
	}

	// Load the saved config and verify
	loadedCfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("failed to load saved config: %v", err)
	}

	if loadedCfg.Agent.Default != "cursor" {
		t.Errorf("expected agent.default 'cursor', got %q", loadedCfg.Agent.Default)
	}
	if loadedCfg.Agent.Model != "claude-opus" {
		t.Errorf("expected agent.model 'claude-opus', got %q", loadedCfg.Agent.Model)
	}
}

func TestSave_CreatesParentDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "nested", "dir", "config.yaml")

	cfg := NewConfig()
	cfg.Agent.Default = "auggie"

	err := Save(cfg, configPath)
	if err != nil {
		t.Fatalf("failed to save config: %v", err)
	}

	// Verify parent directories were created
	if _, err := os.Stat(filepath.Dir(configPath)); os.IsNotExist(err) {
		t.Fatal("parent directory was not created")
	}

	// Verify file was created
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Fatal("config file was not created")
	}
}

func TestSave_DefaultPath(t *testing.T) {
	// Save to current directory's .ralph/config.yaml
	tmpDir := t.TempDir()
	originalWd, _ := os.Getwd()
	defer os.Chdir(originalWd)
	os.Chdir(tmpDir)

	cfg := NewConfig()
	cfg.Agent.Default = "test-agent"

	err := Save(cfg, "")
	if err != nil {
		t.Fatalf("failed to save config with default path: %v", err)
	}

	// Verify file was created at default path
	if _, err := os.Stat(DefaultConfigPath); os.IsNotExist(err) {
		t.Fatal("config file was not created at default path")
	}
}

func TestSave_WithHooks(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	cfg := &Config{
		Agent: AgentConfig{Default: "cursor"},
		Hooks: HooksConfig{
			PreTask: []HookDefinition{
				{Type: HookTypeShell, Command: "echo pre", OnFailure: FailureModeWarnContinue},
			},
			PostTask: []HookDefinition{
				{Type: HookTypeAgent, Command: "review", Model: "gpt-4", OnFailure: FailureModeSkipTask},
			},
		},
	}

	err := Save(cfg, configPath)
	if err != nil {
		t.Fatalf("failed to save config: %v", err)
	}

	loadedCfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("failed to load saved config: %v", err)
	}

	if len(loadedCfg.Hooks.PreTask) != 1 {
		t.Fatalf("expected 1 pre_task hook, got %d", len(loadedCfg.Hooks.PreTask))
	}
	if loadedCfg.Hooks.PreTask[0].Type != HookTypeShell {
		t.Errorf("expected hook type 'shell', got %q", loadedCfg.Hooks.PreTask[0].Type)
	}
	if loadedCfg.Hooks.PreTask[0].Command != "echo pre" {
		t.Errorf("expected hook command 'echo pre', got %q", loadedCfg.Hooks.PreTask[0].Command)
	}
}

func TestSave_WithCustomAgents(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	cfg := &Config{
		Agent: AgentConfig{
			Default: "my-agent",
			Custom: []CustomAgentConfig{
				{
					Name:            "my-agent",
					Description:     "My custom agent",
					Command:         "my-agent-cmd",
					DetectionMethod: DetectionMethodCommand,
					DetectionValue:  "my-agent-cmd",
					Args:            []string{"--flag1", "--flag2"},
				},
			},
		},
	}

	err := Save(cfg, configPath)
	if err != nil {
		t.Fatalf("failed to save config: %v", err)
	}

	loadedCfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("failed to load saved config: %v", err)
	}

	if len(loadedCfg.Agent.Custom) != 1 {
		t.Fatalf("expected 1 custom agent, got %d", len(loadedCfg.Agent.Custom))
	}
	if loadedCfg.Agent.Custom[0].Name != "my-agent" {
		t.Errorf("expected name 'my-agent', got %q", loadedCfg.Agent.Custom[0].Name)
	}
	if len(loadedCfg.Agent.Custom[0].Args) != 2 {
		t.Errorf("expected 2 args, got %d", len(loadedCfg.Agent.Custom[0].Args))
	}
}

func TestLoad_EmptyConfigFile(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	// Write an empty config file
	if err := os.WriteFile(configPath, []byte(""), 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("failed to load empty config: %v", err)
	}

	// Should have all defaults applied
	if cfg.Timeout.Active != DefaultActiveTimeout {
		t.Errorf("expected default active timeout, got %v", cfg.Timeout.Active)
	}
	if cfg.Git.CommitPrefix != DefaultCommitPrefix {
		t.Errorf("expected default commit prefix, got %q", cfg.Git.CommitPrefix)
	}
}

func TestLoad_PartialConfig(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	// Only set some values, rest should get defaults
	configContent := `
timeout:
  active: 3h
`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	// Set value should be used
	if cfg.Timeout.Active != 3*time.Hour {
		t.Errorf("expected timeout.active 3h, got %v", cfg.Timeout.Active)
	}

	// Unset value should get default
	if cfg.Timeout.Stuck != DefaultStuckTimeout {
		t.Errorf("expected default stuck timeout %v, got %v", DefaultStuckTimeout, cfg.Timeout.Stuck)
	}
}

func TestNewLoader(t *testing.T) {
	loader := NewLoader()
	if loader == nil {
		t.Fatal("NewLoader returned nil")
	}
	if loader.v == nil {
		t.Fatal("NewLoader did not initialize viper instance")
	}
}

func TestLoad_AllTestModes(t *testing.T) {
	modes := []struct {
		mode     string
		expected TestMode
	}{
		{"gate", TestModeGate},
		{"tdd", TestModeTDD},
		{"report", TestModeReport},
	}

	for _, tt := range modes {
		t.Run(tt.mode, func(t *testing.T) {
			tmpDir := t.TempDir()
			configPath := filepath.Join(tmpDir, "config.yaml")

			configContent := "test:\n  mode: " + tt.mode
			if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
				t.Fatalf("failed to write config file: %v", err)
			}

			cfg, err := Load(configPath)
			if err != nil {
				t.Fatalf("failed to load config: %v", err)
			}

			if cfg.Test.Mode != tt.expected {
				t.Errorf("expected mode %q, got %q", tt.expected, cfg.Test.Mode)
			}
		})
	}
}

func TestLoad_AllBootstrapDetectionModes(t *testing.T) {
	modes := []struct {
		mode     string
		expected BootstrapDetection
	}{
		{"auto", BootstrapDetectionAuto},
		{"manual", BootstrapDetectionManual},
		{"disabled", BootstrapDetectionDisabled},
	}

	for _, tt := range modes {
		t.Run(tt.mode, func(t *testing.T) {
			tmpDir := t.TempDir()
			configPath := filepath.Join(tmpDir, "config.yaml")

			configContent := "build:\n  bootstrap_detection: " + tt.mode
			if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
				t.Fatalf("failed to write config file: %v", err)
			}

			cfg, err := Load(configPath)
			if err != nil {
				t.Fatalf("failed to load config: %v", err)
			}

			if cfg.Build.BootstrapDetection != tt.expected {
				t.Errorf("expected bootstrap_detection %q, got %q", tt.expected, cfg.Build.BootstrapDetection)
			}
		})
	}
}

func TestLoad_AllDetectionMethods(t *testing.T) {
	methods := []struct {
		method   string
		expected DetectionMethod
	}{
		{"command", DetectionMethodCommand},
		{"path", DetectionMethodPath},
		{"env", DetectionMethodEnv},
		{"always", DetectionMethodAlways},
	}

	for _, tt := range methods {
		t.Run(tt.method, func(t *testing.T) {
			tmpDir := t.TempDir()
			configPath := filepath.Join(tmpDir, "config.yaml")

			configContent := `
agent:
  custom:
    - name: test-agent
      command: test-cmd
      detection_method: ` + tt.method
			if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
				t.Fatalf("failed to write config file: %v", err)
			}

			cfg, err := Load(configPath)
			if err != nil {
				t.Fatalf("failed to load config: %v", err)
			}

			if len(cfg.Agent.Custom) != 1 {
				t.Fatalf("expected 1 custom agent, got %d", len(cfg.Agent.Custom))
			}
			if cfg.Agent.Custom[0].DetectionMethod != tt.expected {
				t.Errorf("expected detection_method %q, got %q", tt.expected, cfg.Agent.Custom[0].DetectionMethod)
			}
		})
	}
}

