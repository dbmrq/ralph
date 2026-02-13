// Package config provides configuration loading and management for ralph.
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"time"

	"github.com/go-viper/mapstructure/v2"
	"github.com/spf13/viper"
)

const (
	// DefaultConfigPath is the default path to the config file relative to project root.
	DefaultConfigPath = ".ralph/config.yaml"

	// EnvPrefix is the prefix for environment variable overrides.
	EnvPrefix = "RALPH"
)

// Loader handles loading configuration from files and environment.
type Loader struct {
	v *viper.Viper
}

// NewLoader creates a new configuration loader.
func NewLoader() *Loader {
	v := viper.New()

	// Set up viper
	v.SetConfigType("yaml")

	// Set up environment variable support
	v.SetEnvPrefix(EnvPrefix)
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	return &Loader{v: v}
}

// LoadConfig loads configuration from the specified path, applies defaults,
// merges environment variables, and validates the result.
// If path is empty, it uses DefaultConfigPath relative to the working directory.
func (l *Loader) LoadConfig(path string) (*Config, error) {
	if path == "" {
		path = DefaultConfigPath
	}

	// Check if the config file exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil, &LoadError{
			Path:    path,
			Message: "config file not found",
			Err:     err,
		}
	}

	// Set the config file path
	l.v.SetConfigFile(path)

	// Read the config file
	if err := l.v.ReadInConfig(); err != nil {
		return nil, &LoadError{
			Path:    path,
			Message: "failed to read config file",
			Err:     err,
		}
	}

	// Start with defaults
	cfg := NewConfig()

	// Unmarshal into the config struct
	if err := l.v.Unmarshal(cfg, viperDecodeHook); err != nil {
		return nil, &LoadError{
			Path:    path,
			Message: "failed to parse config file",
			Err:     err,
		}
	}

	// Apply environment variable overrides
	l.applyEnvOverrides(cfg)

	// Apply defaults for any unset values
	cfg.ApplyDefaults()

	// Validate the configuration
	if err := cfg.Validate(); err != nil {
		return nil, &LoadError{
			Path:    path,
			Message: "configuration validation failed",
			Err:     err,
		}
	}

	return cfg, nil
}

// LoadConfigFromDir loads configuration from .ralph/config.yaml in the specified directory.
func (l *Loader) LoadConfigFromDir(dir string) (*Config, error) {
	path := filepath.Join(dir, DefaultConfigPath)
	return l.LoadConfig(path)
}

// applyEnvOverrides applies environment variable overrides to the config.
func (l *Loader) applyEnvOverrides(cfg *Config) {
	// Agent settings
	if v := os.Getenv(EnvPrefix + "_AGENT_DEFAULT"); v != "" {
		cfg.Agent.Default = v
	}
	if v := os.Getenv(EnvPrefix + "_AGENT_MODEL"); v != "" {
		cfg.Agent.Model = v
	}

	// Timeout settings (parse durations)
	if v := os.Getenv(EnvPrefix + "_TIMEOUT_ACTIVE"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			cfg.Timeout.Active = d
		}
	}
	if v := os.Getenv(EnvPrefix + "_TIMEOUT_STUCK"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			cfg.Timeout.Stuck = d
		}
	}

	// Git settings
	if v := os.Getenv(EnvPrefix + "_GIT_AUTO_COMMIT"); v != "" {
		cfg.Git.AutoCommit = parseBool(v)
	}
	if v := os.Getenv(EnvPrefix + "_GIT_COMMIT_PREFIX"); v != "" {
		cfg.Git.CommitPrefix = v
	}
	if v := os.Getenv(EnvPrefix + "_GIT_PUSH"); v != "" {
		cfg.Git.Push = parseBool(v)
	}

	// Build settings
	if v := os.Getenv(EnvPrefix + "_BUILD_COMMAND"); v != "" {
		cfg.Build.Command = v
	}
	if v := os.Getenv(EnvPrefix + "_BUILD_BOOTSTRAP_DETECTION"); v != "" {
		cfg.Build.BootstrapDetection = BootstrapDetection(v)
	}
	if v := os.Getenv(EnvPrefix + "_BUILD_BOOTSTRAP_CHECK"); v != "" {
		cfg.Build.BootstrapCheck = v
	}

	// Test settings
	if v := os.Getenv(EnvPrefix + "_TEST_COMMAND"); v != "" {
		cfg.Test.Command = v
	}
	if v := os.Getenv(EnvPrefix + "_TEST_MODE"); v != "" {
		cfg.Test.Mode = TestMode(v)
	}
	if v := os.Getenv(EnvPrefix + "_TEST_BASELINE_FILE"); v != "" {
		cfg.Test.BaselineFile = v
	}
	if v := os.Getenv(EnvPrefix + "_TEST_BASELINE_SCOPE"); v != "" {
		cfg.Test.BaselineScope = BaselineScope(v)
	}
}

// parseBool parses a string as a boolean value.
// Returns true for "true", "1", "yes" (case-insensitive).
// Returns false for anything else.
func parseBool(s string) bool {
	s = strings.ToLower(strings.TrimSpace(s))
	return s == "true" || s == "1" || s == "yes"
}

// viperDecodeHook provides custom decoding for viper unmarshaling.
// It composes the standard mapstructure hooks with our custom ones.
func viperDecodeHook(dc *mapstructure.DecoderConfig) {
	dc.DecodeHook = mapstructure.ComposeDecodeHookFunc(
		mapstructure.StringToTimeDurationHookFunc(),
		stringToCustomTypeHookFunc(),
	)
}

// stringToCustomTypeHookFunc creates a decode hook for our custom types.
func stringToCustomTypeHookFunc() mapstructure.DecodeHookFunc {
	return func(from reflect.Type, to reflect.Type, data interface{}) (interface{}, error) {
		if from.Kind() != reflect.String {
			return data, nil
		}

		// Handle our custom string types
		switch to {
		case reflect.TypeOf(BootstrapDetection("")):
			return BootstrapDetection(data.(string)), nil
		case reflect.TypeOf(TestMode("")):
			return TestMode(data.(string)), nil
		case reflect.TypeOf(BaselineScope("")):
			return BaselineScope(data.(string)), nil
		case reflect.TypeOf(FailureMode("")):
			return FailureMode(data.(string)), nil
		case reflect.TypeOf(HookType("")):
			return HookType(data.(string)), nil
		}

		return data, nil
	}
}

// LoadError represents an error that occurred while loading configuration.
type LoadError struct {
	Path    string
	Message string
	Err     error
}

func (e *LoadError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %s: %v", e.Path, e.Message, e.Err)
	}
	return fmt.Sprintf("%s: %s", e.Path, e.Message)
}

func (e *LoadError) Unwrap() error {
	return e.Err
}

// Load is a convenience function that creates a new Loader and loads configuration.
// If path is empty, it uses DefaultConfigPath.
func Load(path string) (*Config, error) {
	return NewLoader().LoadConfig(path)
}

// LoadFromDir is a convenience function that loads configuration from a directory.
func LoadFromDir(dir string) (*Config, error) {
	return NewLoader().LoadConfigFromDir(dir)
}

