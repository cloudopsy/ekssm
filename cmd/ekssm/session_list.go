package main

import (
	"fmt"
	"os"
	"sort"
	"strings"

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

	// Get current KUBECONFIG value to determine active session
	currentKubeconfig := os.Getenv("KUBECONFIG")
	
	// Sort session IDs for consistent output
	ids := make([]string, 0, len(sessions))
	for id := range sessions {
		ids = append(ids, id)
	}
	sort.Strings(ids)

	// Build table rows
	var activeSession *state.SessionState
	for _, id := range ids {
		session := sessions[id]
		
		// Check if this is the active session
		isActive := currentKubeconfig != "" && strings.Contains(currentKubeconfig, session.SessionID)
		if isActive {
			activeSession = &session
		}
		
		// Only add non-active sessions to the regular table
		if !isActive {
			data = append(data, []string{
				session.SessionID,
				session.ClusterName,
				fmt.Sprintf("%d", session.PID),
				session.LocalPort,
				session.KubeconfigPath,
			})
		}
	}

	// Render table with custom styling
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Session ID", "Cluster", "PID", "Local Port", "Kubeconfig Path"})
	table.SetBorder(true)
	table.SetAutoWrapText(false)
	table.SetRowLine(false)
	table.SetHeaderAlignment(tablewriter.ALIGN_LEFT)
	table.SetAlignment(tablewriter.ALIGN_LEFT)
	table.SetHeaderColor(
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgGreenColor},
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgGreenColor},
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgGreenColor},
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgGreenColor},
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgGreenColor},
	)
	
	// Render active session with highlight (if exists)
	if activeSession != nil {
		fmt.Println("🟢 Active Session:")
		activeTable := tablewriter.NewWriter(os.Stdout)
		activeTable.SetHeader([]string{"Session ID", "Cluster", "PID", "Local Port", "Kubeconfig Path"})
		activeTable.SetBorder(true)
		activeTable.SetAutoWrapText(false)
		activeTable.SetRowLine(false)
		activeTable.SetHeaderAlignment(tablewriter.ALIGN_LEFT)
		activeTable.SetAlignment(tablewriter.ALIGN_LEFT)
		activeTable.SetHeaderColor(
			tablewriter.Colors{tablewriter.Bold, tablewriter.FgCyanColor},
			tablewriter.Colors{tablewriter.Bold, tablewriter.FgCyanColor},
			tablewriter.Colors{tablewriter.Bold, tablewriter.FgCyanColor},
			tablewriter.Colors{tablewriter.Bold, tablewriter.FgCyanColor},
			tablewriter.Colors{tablewriter.Bold, tablewriter.FgCyanColor},
		)
		activeTable.SetColumnColor(
			tablewriter.Colors{tablewriter.FgHiCyanColor},
			tablewriter.Colors{tablewriter.FgHiCyanColor},
			tablewriter.Colors{tablewriter.FgHiCyanColor},
			tablewriter.Colors{tablewriter.FgHiCyanColor},
			tablewriter.Colors{tablewriter.FgHiCyanColor},
		)
		activeTable.Append([]string{
			activeSession.SessionID,
			activeSession.ClusterName,
			fmt.Sprintf("%d", activeSession.PID),
			activeSession.LocalPort,
			activeSession.KubeconfigPath,
		})
		activeTable.Render()
		
		if len(data) > 0 {
			fmt.Println("\n📋 Other Sessions:")
		}
	}
	
	// Only render the table if there are other sessions
	if len(data) > 0 {
		table.AppendBulk(data)
		table.Render()
	}

	// If sessions exist, print helpful info
	if len(ids) > 0 {
		// Get the most recently added session (we'll use the last ID in the sorted list)
		latestSessionID := ids[len(ids)-1]
		fmt.Printf("\n📝 Latest session created: %s\n", latestSessionID)
		fmt.Printf("💡 Use 'ekssm session switch %s' to use this session\n", latestSessionID)
	}
}
