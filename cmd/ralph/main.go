// Package main is the entry point for the ralph CLI application.
package main

import (
	"fmt"
	"os"
)

// Version information - will be set by build flags
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	// TODO: Initialize Cobra CLI in INIT-003
	fmt.Printf("ralph %s (commit: %s, built: %s)\n", version, commit, date)
	os.Exit(0)
}

