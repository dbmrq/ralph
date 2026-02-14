// Package main is the entry point for the ralph CLI application.
package main

import (
	"github.com/wexinc/ralph/cmd/ralph/cmd"
)

// Version information - set via ldflags at build time.
// These are passed to the cmd package for the version command.
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	// Set version info in cmd package
	cmd.Version = version
	cmd.Commit = commit
	cmd.Date = date

	cmd.Execute()
}
