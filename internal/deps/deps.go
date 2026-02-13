// Package deps imports project dependencies to ensure they are tracked in go.mod.
// This file ensures that the core dependencies are marked as direct dependencies
// and won't be removed by `go mod tidy` before actual usage begins.
//
// Once features are implemented using these packages, this file can be removed.
package deps

import (
	// TUI framework for building terminal user interfaces
	_ "github.com/charmbracelet/bubbletea"

	// Terminal styling library
	_ "github.com/charmbracelet/lipgloss"

	// CLI framework
	_ "github.com/spf13/cobra"

	// Configuration management
	_ "github.com/spf13/viper"

	// YAML parsing
	_ "gopkg.in/yaml.v3"
)

