// Package main implements the command-line interface for ekssm.
package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	// rootCmd is the root command for the ekssm CLI
	rootCmd = &cobra.Command{
		Use:   "ekssm",
		Short: "EKS SSM Proxy allows running commands against an EKS cluster via an SSM-enabled instance.",
		Long: `EKS SSM Proxy allows running commands against an EKS cluster via an SSM-enabled instance.
Primarily used for kubectl, but can support any command that can use the KUBECONFIG environment.

Use 'ekssm run --help' or 'ekssm session --help' for more details on subcommands.`,
	}

	// debug flag is used to enable debug logging across all commands
	debug bool
)

// Execute runs the root command and handles any errors
func Execute() {
	// Cobra already handles signals internally, so we don't need custom signal handling at the root level
	
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func init() {
	// Add the debug flag to all commands
	rootCmd.PersistentFlags().BoolVar(&debug, "debug", false, "Enable debug logging")
}
