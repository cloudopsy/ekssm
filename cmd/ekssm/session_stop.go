package main

import (
	"fmt"
	"os"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"github.com/cloudopsy/ekssm/internal/logging"
	"github.com/cloudopsy/ekssm/internal/state"
)

var stopSessionID string // Flag variable

var sessionStopCmd = &cobra.Command{
	Use:   "stop [--session-id <id>]",
	Short: "Stop background SSM proxy session(s)",
	Long: `Terminates running SSM proxy process(es) identified by the session state file(s).
Removes the generated kubeconfig file(s) for the session(s).

If --session-id is provided, only that specific session is stopped.
If no --session-id is provided, all active sessions are stopped.`,
	RunE: stopSession,
}

func init() {
	sessionCmd.AddCommand(sessionStopCmd)
	// Add the optional session-id flag
	sessionStopCmd.Flags().StringVar(&stopSessionID, "session-id", "", "Optional ID of the specific session to stop.")
}

func stopSession(cmd *cobra.Command, args []string) error {
	stateManager, err := state.NewManager()
	if err != nil {
		return fmt.Errorf("failed to initialize state manager: %w", err)
	}

	if stopSessionID != "" {
		// Stop a specific session
		logging.Infof("Attempting to stop session with ID: %s", stopSessionID)
		session, err := stateManager.GetSession(stopSessionID)
		if err != nil {
			logging.Errorf("Failed to find session with ID '%s': %v", stopSessionID, err)
			return err // Return the error indicating session not found
		}
		return stopAndCleanupSession(stateManager, *session)
	}

	// Stop all sessions
	logging.Info("Attempting to stop all active sessions...")
	allSessions, err := stateManager.GetAllSessions()
	if err != nil {
		return fmt.Errorf("failed to load session states: %w", err)
	}

	if len(allSessions) == 0 {
		logging.Info("No active sessions found to stop.")
		// Ensure state file is empty/cleared even if no sessions were running
		if clearErr := stateManager.ClearAllSessions(); clearErr != nil {
			logging.Warnf("Failed to ensure state file is cleared: %v", clearErr)
		}
		return nil
	}

	stoppedCount := 0
	var firstErr error // Keep track of the first error encountered

	for id, session := range allSessions {
		logging.Infof("Stopping session %s (PID: %d)...", id, session.PID)
		err := stopAndCleanupSession(stateManager, session)
		if err != nil {
			logging.Errorf("Failed to fully stop session %s: %v", id, err)
			if firstErr == nil {
				firstErr = err // Record the first error
			}
		} else {
			stoppedCount++
		}
	}

	// Always attempt to clear the state file after stopping all,
	// even if errors occurred during individual stops.
	if clearErr := stateManager.ClearAllSessions(); clearErr != nil {
		logging.Warnf("Failed to clear session state file after stopping sessions: %v", clearErr)
		if firstErr == nil {
			firstErr = fmt.Errorf("failed to clear session state: %w", clearErr)
		}
	}

	logging.Infof("Finished stopping sessions. %d sessions were stopped.", stoppedCount)
	if firstErr != nil {
		return fmt.Errorf("encountered errors while stopping sessions: %w", firstErr)
	}

	return nil
}

// stopAndCleanupSession handles the termination and cleanup for a single session.
func stopAndCleanupSession(manager *state.Manager, session state.SessionState) error {
	var combinedErr error

	// --- Terminate Process ---
	process, err := os.FindProcess(session.PID)
	if err != nil {
		// Process likely doesn't exist anymore
		logging.Warnf("Could not find process with PID %d for session %s (already stopped?): %v", session.PID, session.SessionID, err)
		// Continue with cleanup
	} else {
		logging.Debugf("Sending SIGTERM to process PID %d for session %s", session.PID, session.SessionID)
		if err := process.Signal(syscall.SIGTERM); err != nil {
			logging.Warnf("Failed to send SIGTERM to PID %d: %v. Attempting SIGKILL.", session.PID, err)
			// Wait briefly before SIGKILL
			time.Sleep(500 * time.Millisecond)
			if killErr := process.Signal(syscall.SIGKILL); killErr != nil {
				logging.Errorf("Failed to send SIGKILL to PID %d: %v", session.PID, killErr)
				// Record error but continue cleanup
				if combinedErr == nil {
					combinedErr = fmt.Errorf("failed to kill process %d: %w", session.PID, killErr)
				}
			} else {
				logging.Debugf("Sent SIGKILL to process PID %d", session.PID)
			}
		} else {
			logging.Debugf("Sent SIGTERM successfully to PID %d. Waiting briefly...", session.PID)
			// Optionally wait a bit to see if it terminates gracefully
			// This part could be more sophisticated by checking if the process actually exited
			time.Sleep(1 * time.Second)
		}
	}

	// --- Remove Kubeconfig ---
	if session.KubeconfigPath != "" {
		logging.Debugf("Removing kubeconfig file: %s", session.KubeconfigPath)
		if err := os.Remove(session.KubeconfigPath); err != nil {
			if !os.IsNotExist(err) {
				logging.Errorf("Failed to remove kubeconfig file %s: %v", session.KubeconfigPath, err)
				if combinedErr == nil {
					combinedErr = fmt.Errorf("failed to remove kubeconfig %s: %w", session.KubeconfigPath, err)
				}
			} else {
				logging.Debugf("Kubeconfig file %s already removed.", session.KubeconfigPath)
			}
		} else {
			logging.Debugf("Successfully removed kubeconfig %s", session.KubeconfigPath)
		}
	} else {
		logging.Warnf("No kubeconfig path found in state for session %s, skipping removal.", session.SessionID)
	}

	// --- Remove Session from State ---
	// This happens implicitly if stopping all sessions via ClearAllSessions.
	// If stopping a single session, we remove it explicitly.
	if stopSessionID != "" { // Only remove explicitly if stopping a single ID
		logging.Debugf("Removing session %s from state file.", session.SessionID)
		if err := manager.RemoveSession(session.SessionID); err != nil {
			logging.Errorf("Failed to remove session %s from state: %v", session.SessionID, err)
			if combinedErr == nil {
				combinedErr = fmt.Errorf("failed to remove session %s from state: %w", session.SessionID, err)
			}
		}
	}

	if combinedErr == nil {
		logging.Infof("Successfully cleaned up session %s", session.SessionID)
	}

	return combinedErr
}
