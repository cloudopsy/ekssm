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

var sessionListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all active ekssm sessions",
	Long:  `Reads the session state and displays details of all currently running ekssm proxy sessions.`,
	RunE:  listSessions,
}

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

	// If sessions exist, print the latest one for easy access
	if len(ids) > 0 {
		// Get the most recently added session (we'll use the last ID in the sorted list)
		latestSessionID := ids[len(ids)-1]
		fmt.Printf("\nLatest session created: %s\n", latestSessionID)
		fmt.Printf("Use 'ekssm session switch %s' to use this session\n", latestSessionID)
	}
}
