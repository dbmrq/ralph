// Package cmd provides the CLI commands for ralph.
package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// Version information - set via ldflags at build time.
var (
	Version = "dev"
	Commit  = "none"
	Date    = "unknown"
)

// rootCmd represents the base command when called without any subcommands.
var rootCmd = &cobra.Command{
	Use:   "ralph",
	Short: "Ralph Loop - AI-powered task automation",
	Long: `Ralph is an AI-powered task automation tool that runs in a loop,
completing tasks from a task list using AI agents.

It supports multiple AI agents (Cursor, Auggie, custom), provides both
TUI and headless modes, and includes features like automatic commits,
hooks, and TDD support.`,
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	// Add --version flag that prints version info and exits.
	rootCmd.Version = fmt.Sprintf("%s (commit: %s, built: %s)", Version, Commit, Date)
	rootCmd.SetVersionTemplate("ralph {{.Version}}\n")
}

// Root returns the root command for testing purposes.
func Root() *cobra.Command {
	return rootCmd
}

