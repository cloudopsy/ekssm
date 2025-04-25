package main

import (
	"github.com/spf13/cobra"
)

// sessionCmd represents the base command when called without any subcommands
var sessionCmd = &cobra.Command{
	Use:   "session",
	Short: "Manage background SSM proxy sessions for EKS access",
	Long: `Allows starting, stopping, listing, and switching background SSM proxy sessions for EKS access.
These sessions are useful for running multiple commands against a cluster without restarting the proxy each time.`,
	// Run: func(cmd *cobra.Command, args []string) { }, // Base command doesn't do anything itself
}

func init() {
	rootCmd.AddCommand(sessionCmd)

	// Add subcommands to sessionCmd
	// sessionStartCmd is defined in session_start.go
	// sessionStopCmd is defined in session_stop.go
	// sessionListCmd is defined in session_list.go
	// sessionSwitchCmd is defined in session_switch.go
	sessionCmd.AddCommand(sessionListCmd)   // Add the list command
	sessionCmd.AddCommand(sessionSwitchCmd) // Add the switch command

	// Note: sessionStartCmd and sessionStopCmd add themselves via their own init() functions.
	// Cobra handles calling all init() functions.
}
