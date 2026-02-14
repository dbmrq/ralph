package cmd

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
	"github.com/wexinc/ralph/internal/version"
)

// updateCmd represents the update command.
var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update ralph to the latest version",
	Long: `Update ralph to the latest version.

This command checks for the latest version of ralph and downloads/installs
it if a newer version is available.

Note: This updates the ralph binary in place. You may need sudo permissions
if ralph is installed in a system directory (e.g., /usr/local/bin).

Examples:
  ralph update          # Update to latest version
  ralph update --check  # Only check, don't install`,
	RunE: runUpdate,
}

func init() {
	rootCmd.AddCommand(updateCmd)
	updateCmd.Flags().BoolP("check", "c", false, "Only check for updates, don't install")
	updateCmd.Flags().BoolP("yes", "y", false, "Don't prompt for confirmation")
}

// runUpdate handles the update command.
func runUpdate(cmd *cobra.Command, args []string) error {
	checkOnly, _ := cmd.Flags().GetBool("check")
	skipPrompt, _ := cmd.Flags().GetBool("yes")

	cmd.Println("🔍 Checking for updates...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	checker := version.NewChecker()
	release, err := checker.CheckForUpdate(ctx, Version)
	if err != nil {
		return fmt.Errorf("failed to check for updates: %w", err)
	}

	if release == nil {
		cmd.Println("✓ You are already running the latest version:", Version)
		return nil
	}

	cmd.Printf("\n📦 New version available: %s (current: %s)\n", release.TagName, Version)

	if checkOnly {
		cmd.Printf("\nRelease notes: %s\n", release.HTMLURL)
		cmd.Println("\nRun 'ralph update' to install.")
		return nil
	}

	// Confirm update
	if !skipPrompt {
		cmd.Print("\nDo you want to update? [y/N]: ")
		var response string
		fmt.Scanln(&response)
		if response != "y" && response != "Y" {
			cmd.Println("Update cancelled.")
			return nil
		}
	}

	return performUpdate(cmd, release.TagName)
}

// performUpdate downloads and installs the update.
func performUpdate(cmd *cobra.Command, tagVersion string) error {
	cmd.Println("\n📥 Downloading...")

	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "ralph-update-*")
	if err != nil {
		return fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	// Download
	updater := version.NewUpdater()
	archivePath, err := updater.Download(ctx, tagVersion, tmpDir)
	if err != nil {
		return fmt.Errorf("failed to download update: %w", err)
	}

	cmd.Println("📦 Extracting...")

	// Extract
	binaryPath, err := version.Extract(archivePath, tmpDir)
	if err != nil {
		return fmt.Errorf("failed to extract update: %w", err)
	}

	// Get current executable path
	currentExe, err := version.GetCurrentExecutable()
	if err != nil {
		return fmt.Errorf("failed to get current executable: %w", err)
	}

	cmd.Printf("🔧 Installing to %s...\n", currentExe)

	// Install
	if err := version.InstallBinary(binaryPath, currentExe); err != nil {
		// Provide helpful message for permission errors
		cmd.Println("\n⚠️  Permission denied. Try running with sudo:")
		cmd.Printf("    sudo ralph update --yes\n")
		return err
	}

	cmd.Printf("\n✓ Successfully updated to %s!\n", tagVersion)
	return nil
}

