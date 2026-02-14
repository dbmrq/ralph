package config

import (
	"testing"
	"time"
)

func TestNewConfig(t *testing.T) {
	cfg := NewConfig()

	// Verify default timeout values
	if cfg.Timeout.Active != DefaultActiveTimeout {
		t.Errorf("expected Active timeout %v, got %v", DefaultActiveTimeout, cfg.Timeout.Active)
	}
	if cfg.Timeout.Stuck != DefaultStuckTimeout {
		t.Errorf("expected Stuck timeout %v, got %v", DefaultStuckTimeout, cfg.Timeout.Stuck)
	}

	// Verify default git values
	if cfg.Git.AutoCommit != true {
		t.Error("expected AutoCommit to be true by default")
	}
	if cfg.Git.CommitPrefix != DefaultCommitPrefix {
		t.Errorf("expected CommitPrefix %q, got %q", DefaultCommitPrefix, cfg.Git.CommitPrefix)
	}
	if cfg.Git.Push != false {
		t.Error("expected Push to be false by default")
	}

	// Verify default build values
	if cfg.Build.BootstrapDetection != BootstrapDetectionAuto {
		t.Errorf("expected BootstrapDetection %q, got %q", BootstrapDetectionAuto, cfg.Build.BootstrapDetection)
	}

	// Verify default test values
	if cfg.Test.Mode != TestModeGate {
		t.Errorf("expected Test.Mode %q, got %q", TestModeGate, cfg.Test.Mode)
	}
	if cfg.Test.BaselineFile != DefaultBaselineFile {
		t.Errorf("expected BaselineFile %q, got %q", DefaultBaselineFile, cfg.Test.BaselineFile)
	}
	if cfg.Test.BaselineScope != BaselineScopeGlobal {
		t.Errorf("expected BaselineScope %q, got %q", BaselineScopeGlobal, cfg.Test.BaselineScope)
	}

	// Verify hooks are initialized as empty slices (not nil)
	if cfg.Hooks.PreTask == nil {
		t.Error("expected PreTask to be initialized, got nil")
	}
	if cfg.Hooks.PostTask == nil {
		t.Error("expected PostTask to be initialized, got nil")
	}
}

func TestConfig_ApplyDefaults(t *testing.T) {
	// Start with empty config
	cfg := &Config{}

	// Apply defaults
	cfg.ApplyDefaults()

	// Verify defaults were applied
	if cfg.Timeout.Active != DefaultActiveTimeout {
		t.Errorf("expected Active timeout %v, got %v", DefaultActiveTimeout, cfg.Timeout.Active)
	}
	if cfg.Timeout.Stuck != DefaultStuckTimeout {
		t.Errorf("expected Stuck timeout %v, got %v", DefaultStuckTimeout, cfg.Timeout.Stuck)
	}
	if cfg.Git.CommitPrefix != DefaultCommitPrefix {
		t.Errorf("expected CommitPrefix %q, got %q", DefaultCommitPrefix, cfg.Git.CommitPrefix)
	}
	if cfg.Build.BootstrapDetection != BootstrapDetectionAuto {
		t.Errorf("expected BootstrapDetection %q, got %q", BootstrapDetectionAuto, cfg.Build.BootstrapDetection)
	}
	if cfg.Test.Mode != TestModeGate {
		t.Errorf("expected Test.Mode %q, got %q", TestModeGate, cfg.Test.Mode)
	}
	if cfg.Hooks.PreTask == nil {
		t.Error("expected PreTask to be initialized, got nil")
	}
	if cfg.Hooks.PostTask == nil {
		t.Error("expected PostTask to be initialized, got nil")
	}
}

func TestConfig_ApplyDefaults_PreservesExistingValues(t *testing.T) {
	cfg := &Config{
		Timeout: TimeoutConfig{
			Active: 1 * time.Hour,
			Stuck:  10 * time.Minute,
		},
		Git: GitConfig{
			CommitPrefix: "[custom]",
		},
		Test: TestConfig{
			Mode: TestModeTDD,
		},
	}

	cfg.ApplyDefaults()

	// Existing values should be preserved
	if cfg.Timeout.Active != 1*time.Hour {
		t.Errorf("expected Active timeout to be preserved, got %v", cfg.Timeout.Active)
	}
	if cfg.Timeout.Stuck != 10*time.Minute {
		t.Errorf("expected Stuck timeout to be preserved, got %v", cfg.Timeout.Stuck)
	}
	if cfg.Git.CommitPrefix != "[custom]" {
		t.Errorf("expected CommitPrefix to be preserved, got %q", cfg.Git.CommitPrefix)
	}
	if cfg.Test.Mode != TestModeTDD {
		t.Errorf("expected Test.Mode to be preserved, got %q", cfg.Test.Mode)
	}
}

func TestConfig_Validate_ValidConfig(t *testing.T) {
	cfg := NewConfig()

	err := cfg.Validate()
	if err != nil {
		t.Errorf("expected valid config to pass validation, got error: %v", err)
	}
}

func TestConfig_Validate_InvalidTimeouts(t *testing.T) {
	tests := []struct {
		name    string
		cfg     *Config
		wantErr string
	}{
		{
			name: "negative active timeout",
			cfg: &Config{
				Timeout: TimeoutConfig{Active: -1 * time.Hour},
			},
			wantErr: "timeout.active: must be non-negative",
		},
		{
			name: "negative stuck timeout",
			cfg: &Config{
				Timeout: TimeoutConfig{Stuck: -1 * time.Minute},
			},
			wantErr: "timeout.stuck: must be non-negative",
		},
		{
			name: "stuck greater than active",
			cfg: &Config{
				Timeout: TimeoutConfig{Active: 30 * time.Minute, Stuck: 1 * time.Hour},
			},
			wantErr: "timeout.stuck: should be less than timeout.active",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate()
			if err == nil {
				t.Fatal("expected validation error")
			}
			if err.Error() != tt.wantErr {
				t.Errorf("expected error %q, got %q", tt.wantErr, err.Error())
			}
		})
	}
}

func TestConfig_Validate_InvalidBootstrapDetection(t *testing.T) {
	cfg := &Config{
		Build: BuildConfig{
			BootstrapDetection: "invalid",
		},
	}

	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected validation error")
	}
	expected := "build.bootstrap_detection: must be 'auto', 'manual', or 'disabled'"
	if err.Error() != expected {
		t.Errorf("expected error %q, got %q", expected, err.Error())
	}
}

func TestConfig_Validate_InvalidTestMode(t *testing.T) {
	cfg := &Config{
		Test: TestConfig{
			Mode: "invalid",
		},
	}

	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected validation error")
	}
	expected := "test.mode: must be 'gate', 'tdd', or 'report'"
	if err.Error() != expected {
		t.Errorf("expected error %q, got %q", expected, err.Error())
	}
}

func TestConfig_Validate_InvalidBaselineScope(t *testing.T) {
	cfg := &Config{
		Test: TestConfig{
			BaselineScope: "invalid",
		},
	}

	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected validation error")
	}
	expected := "test.baseline_scope: must be 'global', 'session', or 'task'"
	if err.Error() != expected {
		t.Errorf("expected error %q, got %q", expected, err.Error())
	}
}

func TestConfig_Validate_InvalidHookType(t *testing.T) {
	cfg := &Config{
		Hooks: HooksConfig{
			PreTask: []HookDefinition{
				{Type: "invalid", Command: "echo test"},
			},
		},
	}

	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected validation error")
	}
	expected := "hooks.pre_task[0].type: must be 'shell' or 'agent'"
	if err.Error() != expected {
		t.Errorf("expected error %q, got %q", expected, err.Error())
	}
}

func TestConfig_Validate_InvalidFailureMode(t *testing.T) {
	cfg := &Config{
		Hooks: HooksConfig{
			PostTask: []HookDefinition{
				{Type: HookTypeShell, Command: "echo test", OnFailure: "invalid"},
			},
		},
	}

	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected validation error")
	}
	expected := "hooks.post_task[0].on_failure: must be 'skip_task', 'warn_continue', 'abort_loop', or 'ask_agent'"
	if err.Error() != expected {
		t.Errorf("expected error %q, got %q", expected, err.Error())
	}
}

func TestConfig_Validate_MultipleErrors(t *testing.T) {
	cfg := &Config{
		Timeout: TimeoutConfig{Active: -1 * time.Hour, Stuck: -1 * time.Minute},
	}

	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected validation error")
	}
	verrs, ok := err.(ValidationErrors)
	if !ok {
		t.Fatalf("expected ValidationErrors, got %T", err)
	}
	if len(verrs) != 2 {
		t.Errorf("expected 2 validation errors, got %d", len(verrs))
	}
}

func TestValidationError_Error(t *testing.T) {
	err := &ValidationError{Field: "test.field", Message: "is invalid"}
	expected := "test.field: is invalid"
	if err.Error() != expected {
		t.Errorf("expected %q, got %q", expected, err.Error())
	}
}

func TestValidationErrors_Error(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		var errs ValidationErrors
		if errs.Error() != "" {
			t.Errorf("expected empty string, got %q", errs.Error())
		}
	})

	t.Run("single error", func(t *testing.T) {
		errs := ValidationErrors{
			{Field: "field1", Message: "error1"},
		}
		expected := "field1: error1"
		if errs.Error() != expected {
			t.Errorf("expected %q, got %q", expected, errs.Error())
		}
	})

	t.Run("multiple errors", func(t *testing.T) {
		errs := ValidationErrors{
			{Field: "field1", Message: "error1"},
			{Field: "field2", Message: "error2"},
		}
		result := errs.Error()
		if result == "" {
			t.Error("expected non-empty error message")
		}
		// Just verify it contains both errors
		if !containsAll(result, "field1: error1", "field2: error2") {
			t.Errorf("expected error to contain both messages, got %q", result)
		}
	})
}

func containsAll(s string, substrs ...string) bool {
	for _, sub := range substrs {
		found := false
		for i := 0; i <= len(s)-len(sub); i++ {
			if s[i:i+len(sub)] == sub {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

// TEST-001: Additional comprehensive tests for configuration

func TestConfig_Validate_CustomAgentMissingName(t *testing.T) {
	cfg := &Config{
		Agent: AgentConfig{
			Custom: []CustomAgentConfig{
				{
					Name:    "", // Missing name
					Command: "my-agent",
				},
			},
		},
	}

	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected validation error for missing agent name")
	}
	expected := "agent.custom[0].name: is required"
	if err.Error() != expected {
		t.Errorf("expected error %q, got %q", expected, err.Error())
	}
}

func TestConfig_Validate_CustomAgentMissingCommand(t *testing.T) {
	cfg := &Config{
		Agent: AgentConfig{
			Custom: []CustomAgentConfig{
				{
					Name:    "my-agent",
					Command: "", // Missing command
				},
			},
		},
	}

	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected validation error for missing agent command")
	}
	expected := "agent.custom[0].command: is required"
	if err.Error() != expected {
		t.Errorf("expected error %q, got %q", expected, err.Error())
	}
}

func TestConfig_Validate_CustomAgentInvalidDetectionMethod(t *testing.T) {
	cfg := &Config{
		Agent: AgentConfig{
			Custom: []CustomAgentConfig{
				{
					Name:            "my-agent",
					Command:         "my-agent",
					DetectionMethod: "invalid",
				},
			},
		},
	}

	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected validation error for invalid detection method")
	}
	expected := "agent.custom[0].detection_method: must be 'command', 'path', 'env', or 'always'"
	if err.Error() != expected {
		t.Errorf("expected error %q, got %q", expected, err.Error())
	}
}

func TestConfig_Validate_CustomAgentValidDetectionMethods(t *testing.T) {
	methods := []DetectionMethod{
		DetectionMethodCommand,
		DetectionMethodPath,
		DetectionMethodEnv,
		DetectionMethodAlways,
	}

	for _, method := range methods {
		t.Run(string(method), func(t *testing.T) {
			cfg := &Config{
				Agent: AgentConfig{
					Custom: []CustomAgentConfig{
						{
							Name:            "my-agent",
							Command:         "my-agent",
							DetectionMethod: method,
						},
					},
				},
			}

			err := cfg.Validate()
			if err != nil {
				t.Errorf("expected valid config for detection method %q, got error: %v", method, err)
			}
		})
	}
}

func TestConfig_Validate_ValidHookTypes(t *testing.T) {
	hookTypes := []HookType{
		HookTypeShell,
		HookTypeAgent,
	}

	for _, ht := range hookTypes {
		t.Run(string(ht), func(t *testing.T) {
			cfg := &Config{
				Hooks: HooksConfig{
					PreTask: []HookDefinition{
						{Type: ht, Command: "echo test"},
					},
				},
			}

			err := cfg.Validate()
			if err != nil {
				t.Errorf("expected valid config for hook type %q, got error: %v", ht, err)
			}
		})
	}
}

func TestConfig_Validate_ValidFailureModes(t *testing.T) {
	modes := []FailureMode{
		FailureModeSkipTask,
		FailureModeWarnContinue,
		FailureModeAbortLoop,
		FailureModeAskAgent,
	}

	for _, mode := range modes {
		t.Run(string(mode), func(t *testing.T) {
			cfg := &Config{
				Hooks: HooksConfig{
					PostTask: []HookDefinition{
						{Type: HookTypeShell, Command: "echo test", OnFailure: mode},
					},
				},
			}

			err := cfg.Validate()
			if err != nil {
				t.Errorf("expected valid config for failure mode %q, got error: %v", mode, err)
			}
		})
	}
}

func TestConfig_Validate_ValidBootstrapDetectionModes(t *testing.T) {
	modes := []BootstrapDetection{
		BootstrapDetectionAuto,
		BootstrapDetectionManual,
		BootstrapDetectionDisabled,
	}

	for _, mode := range modes {
		t.Run(string(mode), func(t *testing.T) {
			cfg := &Config{
				Build: BuildConfig{
					BootstrapDetection: mode,
				},
			}

			err := cfg.Validate()
			if err != nil {
				t.Errorf("expected valid config for bootstrap detection %q, got error: %v", mode, err)
			}
		})
	}
}

func TestConfig_Validate_ValidTestModes(t *testing.T) {
	modes := []TestMode{
		TestModeGate,
		TestModeTDD,
		TestModeReport,
	}

	for _, mode := range modes {
		t.Run(string(mode), func(t *testing.T) {
			cfg := &Config{
				Test: TestConfig{
					Mode: mode,
				},
			}

			err := cfg.Validate()
			if err != nil {
				t.Errorf("expected valid config for test mode %q, got error: %v", mode, err)
			}
		})
	}
}

func TestConfig_Validate_ValidBaselineScopes(t *testing.T) {
	scopes := []BaselineScope{
		BaselineScopeGlobal,
		BaselineScopeSession,
		BaselineScopeTask,
	}

	for _, scope := range scopes {
		t.Run(string(scope), func(t *testing.T) {
			cfg := &Config{
				Test: TestConfig{
					BaselineScope: scope,
				},
			}

			err := cfg.Validate()
			if err != nil {
				t.Errorf("expected valid config for baseline scope %q, got error: %v", scope, err)
			}
		})
	}
}

func TestConfig_Validate_MultipleCustomAgents(t *testing.T) {
	cfg := &Config{
		Agent: AgentConfig{
			Custom: []CustomAgentConfig{
				{Name: "agent1", Command: "agent1-cmd"},
				{Name: "", Command: "agent2-cmd"}, // Missing name
				{Name: "agent3", Command: ""},     // Missing command
			},
		},
	}

	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected validation error for invalid custom agents")
	}

	// Validation catches all errors, so both missing name and missing command should be reported
	verrs, ok := err.(ValidationErrors)
	if !ok {
		t.Fatalf("expected ValidationErrors, got %T", err)
	}
	if len(verrs) != 2 {
		t.Errorf("expected 2 validation errors, got %d", len(verrs))
	}

	// Verify both errors are present
	errStr := err.Error()
	if !containsAll(errStr, "agent.custom[1].name: is required", "agent.custom[2].command: is required") {
		t.Errorf("expected both validation errors in message, got %q", errStr)
	}
}

func TestConfig_Validate_PostTaskHookInvalidFailureMode(t *testing.T) {
	cfg := &Config{
		Hooks: HooksConfig{
			PostTask: []HookDefinition{
				{Type: HookTypeShell, Command: "echo 1"},
				{Type: HookTypeShell, Command: "echo 2", OnFailure: "invalid"},
			},
		},
	}

	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected validation error")
	}
	expected := "hooks.post_task[1].on_failure: must be 'skip_task', 'warn_continue', 'abort_loop', or 'ask_agent'"
	if err.Error() != expected {
		t.Errorf("expected error %q, got %q", expected, err.Error())
	}
}

func TestConfig_ApplyDefaults_ZeroValues(t *testing.T) {
	// Test that ApplyDefaults doesn't override with zero values when already set
	cfg := &Config{
		Timeout: TimeoutConfig{
			Active: 0, // Should get default
			Stuck:  0, // Should get default
		},
		Git: GitConfig{
			CommitPrefix: "", // Should get default
		},
		Build: BuildConfig{
			BootstrapDetection: "", // Should get default
		},
		Test: TestConfig{
			Mode:          "", // Should get default
			BaselineFile:  "", // Should get default
			BaselineScope: "", // Should get default
		},
	}

	cfg.ApplyDefaults()

	if cfg.Timeout.Active != DefaultActiveTimeout {
		t.Errorf("expected Active timeout %v, got %v", DefaultActiveTimeout, cfg.Timeout.Active)
	}
	if cfg.Timeout.Stuck != DefaultStuckTimeout {
		t.Errorf("expected Stuck timeout %v, got %v", DefaultStuckTimeout, cfg.Timeout.Stuck)
	}
	if cfg.Git.CommitPrefix != DefaultCommitPrefix {
		t.Errorf("expected CommitPrefix %q, got %q", DefaultCommitPrefix, cfg.Git.CommitPrefix)
	}
	if cfg.Build.BootstrapDetection != BootstrapDetectionAuto {
		t.Errorf("expected BootstrapDetection %q, got %q", BootstrapDetectionAuto, cfg.Build.BootstrapDetection)
	}
	if cfg.Test.Mode != TestModeGate {
		t.Errorf("expected Test.Mode %q, got %q", TestModeGate, cfg.Test.Mode)
	}
	if cfg.Test.BaselineFile != DefaultBaselineFile {
		t.Errorf("expected BaselineFile %q, got %q", DefaultBaselineFile, cfg.Test.BaselineFile)
	}
	if cfg.Test.BaselineScope != BaselineScopeGlobal {
		t.Errorf("expected BaselineScope %q, got %q", BaselineScopeGlobal, cfg.Test.BaselineScope)
	}
}

func TestConfig_Validate_ZeroTimeoutsAreValid(t *testing.T) {
	// Zero timeouts should be valid (they will get defaults applied)
	cfg := &Config{
		Timeout: TimeoutConfig{
			Active: 0,
			Stuck:  0,
		},
	}

	err := cfg.Validate()
	if err != nil {
		t.Errorf("expected zero timeouts to be valid, got error: %v", err)
	}
}

func TestConfig_Validate_EqualTimeoutsAreValid(t *testing.T) {
	// Equal timeouts should be valid
	cfg := &Config{
		Timeout: TimeoutConfig{
			Active: 30 * time.Minute,
			Stuck:  30 * time.Minute,
		},
	}

	err := cfg.Validate()
	if err != nil {
		t.Errorf("expected equal timeouts to be valid, got error: %v", err)
	}
}

func TestConfig_Validate_EmptyConfig(t *testing.T) {
	// Completely empty config should be valid (will get defaults)
	cfg := &Config{}

	err := cfg.Validate()
	if err != nil {
		t.Errorf("expected empty config to be valid, got error: %v", err)
	}
}

func TestConfig_Validate_EmptyHooks(t *testing.T) {
	cfg := &Config{
		Hooks: HooksConfig{
			PreTask:  []HookDefinition{},
			PostTask: []HookDefinition{},
		},
	}

	err := cfg.Validate()
	if err != nil {
		t.Errorf("expected config with empty hooks to be valid, got error: %v", err)
	}
}
