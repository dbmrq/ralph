package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// runCmd represents the run command.
var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Start the Ralph loop to execute tasks",
	Long: `Start the Ralph loop to execute tasks from the task list.

By default, Ralph runs in TUI mode with an interactive terminal interface.
Use --headless for non-interactive execution (e.g., in CI/GitHub Actions).

Examples:
  ralph run                    # Start in TUI mode
  ralph run --headless         # Start in headless mode
  ralph run --headless --output json  # Headless with JSON output
  ralph run --continue <id>    # Resume a paused session`,
	RunE: runRun,
}

func init() {
	rootCmd.AddCommand(runCmd)

	runCmd.Flags().Bool("headless", false, "Run in headless mode without TUI")
	runCmd.Flags().Bool("output", false, "Output in JSON format (requires --headless)")
	runCmd.Flags().String("continue", "", "Continue a paused session by ID")
}

// runRun is the main entry point for the run command.
func runRun(cmd *cobra.Command, args []string) error {
	headless, _ := cmd.Flags().GetBool("headless")
	outputJSON, _ := cmd.Flags().GetBool("output")
	continueID, _ := cmd.Flags().GetString("continue")

	if outputJSON && !headless {
		return fmt.Errorf("--output flag requires --headless mode")
	}

	// TODO: LOOP-001+ will implement actual loop execution
	if headless {
		cmd.Println("Starting Ralph in headless mode...")
		if continueID != "" {
			cmd.Printf("Continuing session: %s\n", continueID)
		}
	} else {
		cmd.Println("Starting Ralph in TUI mode...")
		if continueID != "" {
			cmd.Printf("Continuing session: %s\n", continueID)
		}
	}

	return nil
}

