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

type sessionStopOptions struct {
	SessionID string
}

var stopOpts sessionStopOptions

var sessionStopCmd = &cobra.Command{
	Use:   "stop [--session-id <id>]",
	Short: "Stop background SSM proxy session(s)",
	Long: `Terminates running SSM proxy process(es) identified by the session state file(s).
Removes the generated kubeconfig file(s) for the session(s).

If --session-id is provided, only that specific session is stopped.
If no --session-id is provided, all active sessions are stopped.`,
	RunE: stopSession,
}

func stopSession(cmd *cobra.Command, args []string) error {
	debug, _ := cmd.Flags().GetBool("debug")
	logging.SetDebug(debug)

	stateManager, err := state.NewManager()
	if err != nil {
		return fmt.Errorf("failed to initialize state manager: %w", err)
	}

	if stopOpts.SessionID != "" {
		logging.Infof("Attempting to stop session with ID: %s", stopOpts.SessionID)
		session, err := stateManager.GetSession(stopOpts.SessionID)
		if err != nil {
			logging.Errorf("Failed to find session with ID '%s': %v", stopOpts.SessionID, err)
			return err
		}
		return stopAndCleanupSession(stateManager, *session, true)
	}

	logging.Info("Attempting to stop all active sessions...")
	allSessions, err := stateManager.GetAllSessions()
	if err != nil {
		return fmt.Errorf("failed to load session states: %w", err)
	}

	if len(allSessions) == 0 {
		logging.Info("No active sessions found to stop.")
		if clearErr := stateManager.ClearAllSessions(); clearErr != nil {
			logging.Warnf("Failed to ensure state file is cleared: %v", clearErr)
		}
		return nil
	}

	stoppedCount := 0
	var firstErr error

	for id, session := range allSessions {
		logging.Infof("Stopping session %s (PID: %d)...", id, session.PID)
		err := stopAndCleanupSession(stateManager, session, false)
		if err != nil {
			logging.Errorf("Failed to fully stop session %s: %v", id, err)
			if firstErr == nil {
				firstErr = err
			}
		} else {
			stoppedCount++
		}
	}

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

func stopAndCleanupSession(manager *state.Manager, session state.SessionState, removeFromState bool) error {
	var combinedErr error

	process, err := os.FindProcess(session.PID)
	if err != nil {
		logging.Warnf("Could not find process with PID %d for session %s (already stopped?): %v", session.PID, session.SessionID, err)
	} else {
		logging.Debugf("Sending SIGTERM to process PID %d for session %s", session.PID, session.SessionID)
		if err := process.Signal(syscall.SIGTERM); err != nil {
			logging.Warnf("Failed to send SIGTERM to PID %d: %v. Attempting SIGKILL.", session.PID, err)
			time.Sleep(500 * time.Millisecond)
			if killErr := process.Signal(syscall.SIGKILL); killErr != nil {
				logging.Errorf("Failed to send SIGKILL to PID %d: %v", session.PID, killErr)
				// Although lint flags this, we only want to store the first error encountered
				// during the stop attempt for this specific session.
				if combinedErr == nil { //nolint:nilness
					combinedErr = fmt.Errorf("failed to kill process %d: %w", session.PID, killErr)
				}
			} else {
				logging.Debugf("Sent SIGKILL to process PID %d", session.PID)
			}
		} else {
			logging.Debugf("Sent SIGTERM successfully to PID %d. Waiting briefly...", session.PID)
			time.Sleep(1 * time.Second)
		}
	}

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

	if removeFromState {
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

func init() {
	sessionCmd.AddCommand(sessionStopCmd)
	sessionStopCmd.Flags().StringVar(&stopOpts.SessionID, "session-id", "", "Optional ID of the specific session to stop.")
}
