// Package main implements the command-line interface for ekssm.
package main

import (
	"fmt"
	"os"
	"sort"

	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"

	"github.com/cloudopsy/ekssm/internal/logging"
	"github.com/cloudopsy/ekssm/internal/state"
)

// sessionListCmd represents the session list command
var sessionListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all active ekssm sessions",
	Long:  `Reads the session state and displays details of all currently running ekssm proxy sessions.`,
	RunE:  listSessions,
}

// listSessions retrieves and displays all active sessions
func listSessions(cmd *cobra.Command, args []string) error {
	debug, _ := cmd.Flags().GetBool("debug")
	logging.SetDebug(debug)

	stateManager, err := state.NewManager()
	if err != nil {
		return fmt.Errorf("failed to initialize state manager: %w", err)
	}

	allSessions, err := stateManager.GetAllSessions()
	if err != nil {
		// Check if the error is because the file doesn't exist
		if os.IsNotExist(err) {
			fmt.Println("No active ekssm sessions found.")
			return nil
		}
		return fmt.Errorf("failed to load session states: %w", err)
	}

	if len(allSessions) == 0 {
		fmt.Println("No active ekssm sessions found.")
		return nil
	}

	renderSessionTable(allSessions)
	return nil
}

// renderSessionTable formats and displays session data in a table
func renderSessionTable(sessions state.SessionMap) {
	// Prepare session data for display
	data := [][]string{}
	
	// Sort session IDs for consistent output
	ids := make([]string, 0, len(sessions))
	for id := range sessions {
		ids = append(ids, id)
	}
	sort.Strings(ids)

	// Build table rows
	for _, id := range ids {
		session := sessions[id]
		data = append(data, []string{
			session.SessionID,
			session.ClusterName,
			fmt.Sprintf("%d", session.PID),
			session.LocalPort,
			session.KubeconfigPath,
		})
	}

	// Render table
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Session ID", "Cluster", "PID", "Local Port", "Kubeconfig Path"})
	table.SetBorder(true)
	table.AppendBulk(data)
	table.Render()
}
