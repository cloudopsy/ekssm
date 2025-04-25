// Package main implements the command-line interface for ekssm.
package main

import (
	"fmt"
	"os"
)

// main is the entry point for the ekssm command-line tool
func main() {
	// Set global options for Cobra command error handling
	rootCmd.SilenceUsage = true
	rootCmd.SilenceErrors = true

	// Execute will handle command line processing and execution
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
