package main

import (
	"fmt"
	"os"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/cloudopsy/ekssm/internal/constants"
	"github.com/cloudopsy/ekssm/internal/logging"
	"github.com/cloudopsy/ekssm/internal/state"
	"github.com/cloudopsy/ekssm/internal/util"
)

var sessionStopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stops the running ekssm proxy session",
	Long:  `Stops the background SSM port forwarding session, kills the associated process, restores the original kubeconfig file, and removes the session state file.`,
	RunE:  stopSession,
}

func init() {
	sessionCmd.AddCommand(sessionStopCmd)
}

func stopSession(cmd *cobra.Command, args []string) error {
	logging.SetDebug(debug)

	logging.Info("Attempting to stop ekssm session...")

	currentState, err := state.ReadState()
	if err != nil {
		return fmt.Errorf("failed to read session state: %w", err)
	}

	if currentState == nil {
		fmt.Println("No active ekssm session found.")
		kubeconfigPath := util.GetKubeconfigPath()
		backupPath := kubeconfigPath + constants.SessionBackupSuffix
		if _, err := os.Stat(backupPath); err == nil {
			logging.Warnf("Found kubeconfig backup file %s but no active session state.", backupPath)
			fmt.Printf("A kubeconfig backup file exists at '%s'. You might need to restore it manually if the session terminated unexpectedly.\n", backupPath)
		}
		return nil
	}

	logging.Infof("Found active session: PID=%d, Cluster=%s", currentState.PID, currentState.ClusterName)

	var firstErr error

	process, err := os.FindProcess(currentState.PID)
	if err != nil {
		logging.Warnf("Could not find process with PID %d: %v. It might have already stopped.", currentState.PID, err)
	} else {
		if err := process.Signal(os.Interrupt); err != nil {
			logging.Warnf("Failed to send Interrupt signal to process %d: %v. Trying SIGTERM...", currentState.PID, err)
			if termErr := process.Signal(syscall.SIGTERM); termErr != nil {
				logging.Errorf("Failed to send SIGTERM signal to process %d: %v. Manual cleanup might be required.", currentState.PID, termErr)
				firstErr = fmt.Errorf("failed to terminate process %d: %w", currentState.PID, termErr)
			} else {
				logging.Infof("Successfully sent SIGTERM to process %d.", currentState.PID)
			}
		} else {
			logging.Infof("Successfully sent Interrupt signal to process %d.", currentState.PID)
		}
	}

	kubeconfigPath := util.GetKubeconfigPath()
	backupPath := kubeconfigPath + constants.SessionBackupSuffix
	if err := util.RestoreKubeconfig(kubeconfigPath, backupPath); err != nil {
		logging.Errorf("Failed to restore kubeconfig: %v", err)
		if firstErr == nil {
			firstErr = fmt.Errorf("kubeconfig restore failed: %w", err)
		}
	} else {
		logging.Info("Successfully restored kubeconfig.")
	}

	if err := state.ClearState(); err != nil {
		logging.Errorf("Failed to clear session state file: %v", err)
		if firstErr == nil {
			firstErr = fmt.Errorf("clearing session state failed: %w", err)
		}
	} else {
		logging.Info("Successfully cleared session state.")
	}

	if firstErr != nil {
		fmt.Printf("ekssm session stop completed with errors. Please check logs.\nError: %v\n", firstErr)
		return firstErr
	}

	fmt.Println("ekssm session stopped successfully.")
	return nil
}
