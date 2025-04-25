// Package main implements the command-line interface for ekssm.
package main

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/cloudopsy/ekssm/internal/logging"
	"github.com/cloudopsy/ekssm/internal/state"
)

// sessionSwitchCmd represents the session switch command
var sessionSwitchCmd = &cobra.Command{
	Use:   "switch <session_id>",
	Short: "Show command to switch KUBECONFIG to a specific session",
	Long: `Looks up the specified active ekssm session and prints the shell command 
required to set the KUBECONFIG environment variable to that session's dedicated kubeconfig file. 

You need to run the output command in your shell to actually switch the context.
Example: $(ekssm session switch <some-session-id>)
Or copy-paste the output.`,
	Args:  cobra.ExactArgs(1), // Requires exactly one argument: the session_id
	RunE:  switchSession,
}

// switchSession retrieves and outputs the command to switch to a specific session
func switchSession(cmd *cobra.Command, args []string) error {
	sessionID := args[0]

	debug, _ := cmd.Flags().GetBool("debug")
	logging.SetDebug(debug)

	stateManager, err := state.NewManager()
	if err != nil {
		return fmt.Errorf("failed to initialize state manager: %w", err)
	}

	session, err := stateManager.GetSession(sessionID)
	if err != nil {
		logging.Errorf("Session ID '%s' not found or error reading state: %v", sessionID, err)
		
		// Provide helpful hints to the user
		allSessions, _ := stateManager.GetAllSessions()
		if len(allSessions) == 0 {
			fmt.Println("Hint: No active sessions found. Use 'ekssm session start' to create one.")
		} else {
			fmt.Println("Hint: Use 'ekssm session list' to see available session IDs.")
		}
		
		return fmt.Errorf("active session with ID '%s' not found", sessionID)
	}

	if session.KubeconfigPath == "" {
		return fmt.Errorf("session '%s' exists but has no associated kubeconfig path in state", sessionID)
	}

	// Print the export command for the user to execute
	fmt.Printf("export KUBECONFIG='%s'\n", session.KubeconfigPath)
	logging.Infof("Use the above command in your shell to switch KUBECONFIG for session %s (Cluster: %s)", 
		session.SessionID, session.ClusterName)

	return nil
}
