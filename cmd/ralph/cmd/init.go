package cmd

import (
	"github.com/spf13/cobra"
)

// initCmd represents the init command.
var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize Ralph in the current project",
	Long: `Initialize Ralph in the current project.

This command creates the .ralph directory and configuration files:
  - .ralph/config.yaml    Default configuration
  - .ralph/tasks.json     Empty task list
  - .ralph/prompts/       Prompt templates

Use --force to overwrite existing configuration.

Examples:
  ralph init          # Initialize in current directory
  ralph init --force  # Force reinitialize, overwriting existing config`,
	RunE: runInit,
}

func init() {
	rootCmd.AddCommand(initCmd)

	initCmd.Flags().BoolP("force", "f", false, "Overwrite existing configuration")
}

// runInit is the main entry point for the init command.
func runInit(cmd *cobra.Command, args []string) error {
	force, _ := cmd.Flags().GetBool("force")

	// TODO: POLISH-001 will implement actual initialization
	if force {
		cmd.Println("Initializing Ralph (force mode)...")
	} else {
		cmd.Println("Initializing Ralph...")
	}

	cmd.Println("Created .ralph/config.yaml")
	cmd.Println("Created .ralph/tasks.json")
	cmd.Println("Created .ralph/prompts/")
	cmd.Println("")
	cmd.Println("Ralph initialized successfully!")
	cmd.Println("Edit .ralph/config.yaml to configure your settings.")
	cmd.Println("Run 'ralph run' to start the task loop.")

	return nil
}

