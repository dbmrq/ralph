// Package config provides configuration data structures for ralph.
package config

import (
	"fmt"
	"time"
)

// Config represents the complete ralph configuration loaded from .ralph/config.yaml.
type Config struct {
	Agent   AgentConfig   `yaml:"agent"   json:"agent"`
	Timeout TimeoutConfig `yaml:"timeout" json:"timeout"`
	Git     GitConfig     `yaml:"git"     json:"git"`
	Build   BuildConfig   `yaml:"build"   json:"build"`
	Test    TestConfig    `yaml:"test"    json:"test"`
	Hooks   HooksConfig   `yaml:"hooks"   json:"hooks"`
}

// AgentConfig configures the AI agent settings.
type AgentConfig struct {
	// Default is the default agent to use. Empty string means prompt user if multiple available.
	Default string `yaml:"default" json:"default"`
	// Model is the default model to use. Empty string means use agent's default.
	Model string `yaml:"model" json:"model"`
}

// TimeoutConfig configures the smart timeout system.
type TimeoutConfig struct {
	// Active is the timeout while agent is producing output (default: 2h).
	Active time.Duration `yaml:"active" json:"active"`
	// Stuck is the timeout when no output is being produced (default: 30m).
	Stuck time.Duration `yaml:"stuck" json:"stuck"`
}

// GitConfig configures git integration settings.
type GitConfig struct {
	// AutoCommit enables automatic commit after each successful task (default: true).
	AutoCommit bool `yaml:"auto_commit" json:"auto_commit"`
	// CommitPrefix is the prefix for commit messages (default: "[ralph]").
	CommitPrefix string `yaml:"commit_prefix" json:"commit_prefix"`
	// Push enables automatic push after commit (default: false).
	Push bool `yaml:"push" json:"push"`
}

// BootstrapDetection defines how bootstrap/greenfield state is detected.
type BootstrapDetection string

const (
	// BootstrapDetectionAuto auto-detects based on project type markers.
	BootstrapDetectionAuto BootstrapDetection = "auto"
	// BootstrapDetectionManual uses a custom command for detection.
	BootstrapDetectionManual BootstrapDetection = "manual"
	// BootstrapDetectionDisabled always runs build/test commands.
	BootstrapDetectionDisabled BootstrapDetection = "disabled"
)

// BuildConfig configures build verification settings.
type BuildConfig struct {
	// Command is the build command. Empty means auto-detect based on project type.
	Command string `yaml:"command" json:"command"`
	// BootstrapDetection configures how bootstrap state is detected (default: auto).
	BootstrapDetection BootstrapDetection `yaml:"bootstrap_detection" json:"bootstrap_detection"`
	// BootstrapCheck is the custom command for manual mode (exit 0 = bootstrap, non-zero = ready).
	BootstrapCheck string `yaml:"bootstrap_check" json:"bootstrap_check"`
}

// TestMode defines the test gate behavior.
type TestMode string

const (
	// TestModeGate blocks on any test failure.
	TestModeGate TestMode = "gate"
	// TestModeTDD allows initial failures, blocks only on regressions.
	TestModeTDD TestMode = "tdd"
	// TestModeReport reports test results but doesn't block.
	TestModeReport TestMode = "report"
)

// BaselineScope defines when test baselines are captured.
type BaselineScope string

const (
	// BaselineScopeGlobal captures baseline once at loop start (default).
	BaselineScopeGlobal BaselineScope = "global"
	// BaselineScopeSession captures baseline once per session.
	BaselineScopeSession BaselineScope = "session"
	// BaselineScopeTask captures baseline before each task.
	BaselineScopeTask BaselineScope = "task"
)

// TestConfig configures test verification settings.
type TestConfig struct {
	// Command is the test command. Empty means auto-detect based on project type.
	Command string `yaml:"command" json:"command"`
	// Mode controls test gate behavior (default: gate).
	Mode TestMode `yaml:"mode" json:"mode"`
	// BaselineFile is the path for test baseline storage (default: .ralph/test_baseline.json).
	BaselineFile string `yaml:"baseline_file" json:"baseline_file"`
	// BaselineScope controls when baselines are captured (default: global).
	BaselineScope BaselineScope `yaml:"baseline_scope" json:"baseline_scope"`
}

// FailureMode defines how hook failures are handled.
type FailureMode string

const (
	// FailureModeSkipTask skips the current task and moves to next.
	FailureModeSkipTask FailureMode = "skip_task"
	// FailureModeWarnContinue logs a warning but continues with the task.
	FailureModeWarnContinue FailureMode = "warn_continue"
	// FailureModeAbortLoop stops the entire loop.
	FailureModeAbortLoop FailureMode = "abort_loop"
	// FailureModeAskAgent includes failure info in agent prompt, lets agent decide.
	FailureModeAskAgent FailureMode = "ask_agent"
)

// HookType defines the type of hook execution.
type HookType string

const (
	// HookTypeShell executes a shell command.
	HookTypeShell HookType = "shell"
	// HookTypeAgent runs an agent with a prompt.
	HookTypeAgent HookType = "agent"
)

// HookDefinition defines a single hook configuration.
type HookDefinition struct {
	// Type is the hook type: "shell" or "agent".
	Type HookType `yaml:"type" json:"type"`
	// Command is the shell command (for shell hooks) or prompt (for agent hooks).
	Command string `yaml:"command" json:"command"`
	// Model is the model to use (for agent hooks, optional - uses main agent's model if empty).
	Model string `yaml:"model,omitempty" json:"model,omitempty"`
	// Agent is the agent to use (for agent hooks, optional - uses main agent if empty).
	Agent string `yaml:"agent,omitempty" json:"agent,omitempty"`
	// OnFailure defines how to handle hook failures (default: warn_continue).
	OnFailure FailureMode `yaml:"on_failure" json:"on_failure"`
}

// HooksConfig configures pre/post task hooks.
type HooksConfig struct {
	// PreTask hooks run before each task.
	PreTask []HookDefinition `yaml:"pre_task" json:"pre_task"`
	// PostTask hooks run after each task.
	PostTask []HookDefinition `yaml:"post_task" json:"post_task"`
}

// Default values.
const (
	DefaultActiveTimeout   = 2 * time.Hour
	DefaultStuckTimeout    = 30 * time.Minute
	DefaultCommitPrefix    = "[ralph]"
	DefaultBaselineFile    = ".ralph/test_baseline.json"
)

// NewConfig returns a new Config with default values applied.
func NewConfig() *Config {
	return &Config{
		Agent: AgentConfig{
			Default: "",
			Model:   "",
		},
		Timeout: TimeoutConfig{
			Active: DefaultActiveTimeout,
			Stuck:  DefaultStuckTimeout,
		},
		Git: GitConfig{
			AutoCommit:   true,
			CommitPrefix: DefaultCommitPrefix,
			Push:         false,
		},
		Build: BuildConfig{
			Command:            "",
			BootstrapDetection: BootstrapDetectionAuto,
			BootstrapCheck:     "",
		},
		Test: TestConfig{
			Command:       "",
			Mode:          TestModeGate,
			BaselineFile:  DefaultBaselineFile,
			BaselineScope: BaselineScopeGlobal,
		},
		Hooks: HooksConfig{
			PreTask:  []HookDefinition{},
			PostTask: []HookDefinition{},
		},
	}
}

// ApplyDefaults applies default values to any unset fields.
// This is used after loading config from file to fill in missing values.
func (c *Config) ApplyDefaults() {
	defaults := NewConfig()

	// Apply timeout defaults
	if c.Timeout.Active == 0 {
		c.Timeout.Active = defaults.Timeout.Active
	}
	if c.Timeout.Stuck == 0 {
		c.Timeout.Stuck = defaults.Timeout.Stuck
	}

	// Apply git defaults
	if c.Git.CommitPrefix == "" {
		c.Git.CommitPrefix = defaults.Git.CommitPrefix
	}
	// Note: AutoCommit defaults to true but we can't detect if it was explicitly set to false
	// vs never set. The loader handles this by using the default config as base.

	// Apply build defaults
	if c.Build.BootstrapDetection == "" {
		c.Build.BootstrapDetection = defaults.Build.BootstrapDetection
	}

	// Apply test defaults
	if c.Test.Mode == "" {
		c.Test.Mode = defaults.Test.Mode
	}
	if c.Test.BaselineFile == "" {
		c.Test.BaselineFile = defaults.Test.BaselineFile
	}
	if c.Test.BaselineScope == "" {
		c.Test.BaselineScope = defaults.Test.BaselineScope
	}

	// Initialize nil slices
	if c.Hooks.PreTask == nil {
		c.Hooks.PreTask = []HookDefinition{}
	}
	if c.Hooks.PostTask == nil {
		c.Hooks.PostTask = []HookDefinition{}
	}
}

// ValidationError represents a configuration validation error.
type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return e.Field + ": " + e.Message
}

// ValidationErrors is a collection of validation errors.
type ValidationErrors []*ValidationError

func (e ValidationErrors) Error() string {
	if len(e) == 0 {
		return ""
	}
	if len(e) == 1 {
		return e[0].Error()
	}
	msg := "multiple validation errors:"
	for _, err := range e {
		msg += "\n  - " + err.Error()
	}
	return msg
}

// Validate validates the configuration and returns any errors.
func (c *Config) Validate() error {
	var errs ValidationErrors

	// Validate timeout config
	if c.Timeout.Active < 0 {
		errs = append(errs, &ValidationError{Field: "timeout.active", Message: "must be non-negative"})
	}
	if c.Timeout.Stuck < 0 {
		errs = append(errs, &ValidationError{Field: "timeout.stuck", Message: "must be non-negative"})
	}
	if c.Timeout.Active > 0 && c.Timeout.Stuck > 0 && c.Timeout.Stuck > c.Timeout.Active {
		errs = append(errs, &ValidationError{
			Field:   "timeout.stuck",
			Message: "should be less than timeout.active",
		})
	}

	// Validate bootstrap detection
	if c.Build.BootstrapDetection != "" {
		switch c.Build.BootstrapDetection {
		case BootstrapDetectionAuto, BootstrapDetectionManual, BootstrapDetectionDisabled:
			// valid
		default:
			errs = append(errs, &ValidationError{
				Field:   "build.bootstrap_detection",
				Message: "must be 'auto', 'manual', or 'disabled'",
			})
		}
	}

	// Validate test mode
	if c.Test.Mode != "" {
		switch c.Test.Mode {
		case TestModeGate, TestModeTDD, TestModeReport:
			// valid
		default:
			errs = append(errs, &ValidationError{
				Field:   "test.mode",
				Message: "must be 'gate', 'tdd', or 'report'",
			})
		}
	}

	// Validate baseline scope
	if c.Test.BaselineScope != "" {
		switch c.Test.BaselineScope {
		case BaselineScopeGlobal, BaselineScopeSession, BaselineScopeTask:
			// valid
		default:
			errs = append(errs, &ValidationError{
				Field:   "test.baseline_scope",
				Message: "must be 'global', 'session', or 'task'",
			})
		}
	}

	// Validate hooks
	for i, hook := range c.Hooks.PreTask {
		if err := validateHook(hook, "hooks.pre_task", i); err != nil {
			errs = append(errs, err)
		}
	}
	for i, hook := range c.Hooks.PostTask {
		if err := validateHook(hook, "hooks.post_task", i); err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		return errs
	}
	return nil
}

func validateHook(hook HookDefinition, prefix string, index int) *ValidationError {
	field := fmt.Sprintf("%s[%d]", prefix, index)

	// Validate hook type
	if hook.Type != "" {
		switch hook.Type {
		case HookTypeShell, HookTypeAgent:
			// valid
		default:
			return &ValidationError{
				Field:   field + ".type",
				Message: "must be 'shell' or 'agent'",
			}
		}
	}

	// Validate on_failure mode
	if hook.OnFailure != "" {
		switch hook.OnFailure {
		case FailureModeSkipTask, FailureModeWarnContinue, FailureModeAbortLoop, FailureModeAskAgent:
			// valid
		default:
			return &ValidationError{
				Field:   field + ".on_failure",
				Message: "must be 'skip_task', 'warn_continue', 'abort_loop', or 'ask_agent'",
			}
		}
	}

	return nil
}

