package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/dbmrq/ralph/internal/version"
)

// versionCmd represents the version command.
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show version information",
	Long: `Show detailed version information for ralph.

Displays the current version, commit hash, build date,
and Go/platform information.

Examples:
  ralph version           # Show detailed version info
  ralph version --check   # Check for updates`,
	RunE: runVersion,
}

func init() {
	rootCmd.AddCommand(versionCmd)
	versionCmd.Flags().BoolP("check", "c", false, "Check for available updates")
}

// runVersion handles the version command.
func runVersion(cmd *cobra.Command, args []string) error {
	info := version.NewInfo(Version, Commit, Date)
	cmd.Println(info.FullString())

	check, _ := cmd.Flags().GetBool("check")
	if check {
		return checkForUpdate(cmd)
	}

	return nil
}

// checkForUpdate checks for available updates and reports.
func checkForUpdate(cmd *cobra.Command) error {
	cmd.Println("")
	cmd.Println("Checking for updates...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	checker := version.NewChecker()
	release, err := checker.CheckForUpdate(ctx, Version)
	if err != nil {
		return fmt.Errorf("failed to check for updates: %w", err)
	}

	if release == nil {
		cmd.Println("✓ You are running the latest version.")
		return nil
	}

	cmd.Println("")
	cmd.Printf("📦 A new version is available: %s (current: %s)\n", release.TagName, Version)
	cmd.Println("")
	cmd.Println("To update, run:")
	cmd.Println("  ralph update")
	cmd.Println("")
	cmd.Printf("Release notes: %s\n", release.HTMLURL)

	return nil
}
