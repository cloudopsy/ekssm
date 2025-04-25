package main

import (
	"github.com/spf13/cobra"
)

var sessionCmd = &cobra.Command{
	Use:   "session",
	Short: "Manage persistent SSM proxy sessions for EKS",
	Long:  `Allows starting and stopping a background SSM proxy session for EKS access, useful for running multiple commands without restarting the proxy.`,
}

func init() {
	rootCmd.AddCommand(sessionCmd)
	// Subcommands (start, stop) will be added to sessionCmd in their respective files
}
