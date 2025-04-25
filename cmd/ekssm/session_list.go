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
		// Check if the error is because the file doesn't exist (no sessions ever started)
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

	// Prepare data for table
	data := [][]string{}
	ids := make([]string, 0, len(allSessions)) // For sorting
	for id := range allSessions {
		ids = append(ids, id)
	}
	sort.Strings(ids) // Sort by session ID for consistent output

	for _, id := range ids {
		session := allSessions[id]
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
	table.SetBorder(true)       // Set border to true
	table.AppendBulk(data)      // Add Bulk Data
	table.Render()

	return nil
}

// No init() needed here as the command is added in session.go
