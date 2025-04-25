// Package main implements the command-line interface for ekssm.
package main

import (
	"github.com/spf13/cobra"
)

// sessionCmd represents the base command for session management
var sessionCmd = &cobra.Command{
	Use:   "session",
	Short: "Manage background SSM proxy sessions for EKS access",
	Long: `Allows starting, stopping, listing, and switching background SSM proxy sessions for EKS access.
These sessions are useful for running multiple commands against a cluster without restarting the proxy each time.

Available subcommands:
  start       - Start a new background session
  stop        - Stop one or all sessions
  list        - List all active sessions
  switch      - Get command to switch to a specific session

TIP: For automatic KUBECONFIG setting without manual export, use shell integration:
  eval "$(ekssm shell bash)"  # Add to ~/.bashrc or ~/.zshrc`,
}

func init() {
	rootCmd.AddCommand(sessionCmd)

	// Add list and switch commands here
	// sessionStartCmd and sessionStopCmd add themselves via their own init() functions
	sessionCmd.AddCommand(sessionListCmd)
	sessionCmd.AddCommand(sessionSwitchCmd)
}
