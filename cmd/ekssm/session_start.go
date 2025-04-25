// Package main implements the command-line interface for ekssm.
package main

import (
	"context"
	"fmt"
	"os"
	"syscall"
	"time"

	"github.com/google/uuid"
	"github.com/spf13/cobra"

	"github.com/cloudopsy/ekssm/internal/constants"
	"github.com/cloudopsy/ekssm/internal/logging"
	"github.com/cloudopsy/ekssm/internal/state"
	"github.com/cloudopsy/ekssm/internal/util"
	"github.com/cloudopsy/ekssm/pkg/kubectl"
	"github.com/cloudopsy/ekssm/pkg/proxy"
)

// startOptions contains all the command line options for the session start command
var startOpts struct {
	ClusterName string
	InstanceID  string
	LocalPort   string // Optional, leave empty or "0" for dynamic port allocation
}

// sessionStartCmd represents the session start command
var sessionStartCmd = &cobra.Command{
	Use:   "start",
	Short: "Start a background SSM proxy session for EKS access",
	Long: `Starts an SSM port forwarding session in the background to the specified EKS cluster endpoint via an EC2 instance.
It automatically finds an available local port unless one is specified with --local-port.
It generates a dedicated kubeconfig file for this session and saves the session details.
Multiple sessions can be started concurrently.`,
	RunE: startSession,
}

// startSession handles starting a background SSM proxy session
func startSession(cmd *cobra.Command, args []string) error {
	debug, _ := cmd.Flags().GetBool("debug")
	logging.SetDebug(debug)
	logging.Info("Starting new ekssm session...")

	if startOpts.ClusterName == "" || startOpts.InstanceID == "" {
		return fmt.Errorf("--cluster-name and --instance-id are required")
	}

	// Initialize state manager
	stateManager, err := state.NewManager()
	if err != nil {
		return fmt.Errorf("failed to initialize state manager: %w", err)
	}

	// Create a context that can be canceled on signal
	ctx, cancelCtx := context.WithCancel(context.Background())
	defer cancelCtx()

	// Get the EKS cluster endpoint
	eksHost, err := util.EKSClusterEndpoint(ctx, startOpts.ClusterName)
	if err != nil {
		return err
	}

	// Determine local port
	localPort := startOpts.LocalPort
	if localPort == "" || localPort == "0" {
		logging.Debug("No local port specified or set to 0, finding an available port...")
		foundPort, err := util.FindAvailablePort()
		if err != nil {
			return fmt.Errorf("failed to find an available local port: %w", err)
		}
		localPort = foundPort
		logging.Infof("Using dynamically allocated local port: %s", localPort)
	} else {
		logging.Infof("Using user-specified local port: %s", localPort)
	}

	ssmProxy := proxy.NewSSMProxy(startOpts.InstanceID, localPort, eksHost, constants.EKSApiPort)

	// Generate a unique session ID
	sessionID := uuid.New().String()
	logging.Debugf("Generated Session ID: %s", sessionID)

	// Generate path for the session-specific kubeconfig
	kubeconfigPath := util.KubeconfigPathForSession(startOpts.ClusterName, sessionID)
	logging.Debugf("Session kubeconfig path: %s", kubeconfigPath)

	// Generate kubeconfig content using the determined localPort
	endpoint := fmt.Sprintf("https://localhost:%s", localPort)
	kubeconfigContent := kubectl.GenerateKubeconfig(startOpts.ClusterName, endpoint)

	// Write the session-specific kubeconfig
	if err := util.WriteKubeconfig(kubeconfigPath, kubeconfigContent); err != nil {
		return fmt.Errorf("failed to write session kubeconfig to %s: %w", kubeconfigPath, err)
	}
	logging.Debugf("Session kubeconfig written successfully.")

	// Start the proxy in the background
	pid, err := ssmProxy.StartBackground()
	if err != nil {
		// Attempt cleanup if proxy fails to start
		_ = os.Remove(kubeconfigPath)
		return fmt.Errorf("failed to start SSM proxy: %w", err)
	}
	logging.Infof("SSM proxy started successfully in background (PID: %d)", pid)

	// Save the session state using the determined localPort
	newState := state.SessionState{
		PID:            pid,
		SessionID:      sessionID,
		ClusterName:    startOpts.ClusterName,
		InstanceID:     startOpts.InstanceID,
		LocalPort:      localPort,
		KubeconfigPath: kubeconfigPath,
	}

	if err := stateManager.AddSession(newState); err != nil {
		// Attempt to kill the orphaned proxy process if state saving fails
		logging.Errorf("Failed to save session state: %v. Attempting to terminate proxy process PID %d...", err, pid)
		process, findErr := os.FindProcess(pid)
		if findErr == nil {
			if killErr := process.Signal(syscall.SIGTERM); killErr != nil {
				logging.Warnf("Failed to send SIGTERM to proxy process PID %d: %v", pid, killErr)
			}
		} else {
			logging.Warnf("Could not find proxy process PID %d to terminate: %v", pid, findErr)
		}
		// Also attempt cleanup of kubeconfig file
		_ = os.Remove(kubeconfigPath)
		return fmt.Errorf("failed to save session state after starting proxy: %w", err)
	}

	// Print success information
	printSessionInfo(pid, sessionID, startOpts.ClusterName, localPort, eksHost, startOpts.InstanceID, kubeconfigPath)

	// Set up signal handling for cleaner shutdown
	cleanup := func() {
		logging.Warnf("Attempting cleanup for session %s...", sessionID)
		if stopErr := ssmProxy.Stop(); stopErr != nil {
			logging.Warnf("Error stopping proxy during signal cleanup: %v", stopErr)
		}
		if removeErr := os.Remove(kubeconfigPath); removeErr != nil && !os.IsNotExist(removeErr) {
			logging.Warnf("Error removing kubeconfig during signal cleanup: %v", removeErr)
		}
		_ = stateManager.RemoveSession(sessionID)
	}

	// Optional: Wait a short moment to let the proxy establish fully
	time.Sleep(1 * time.Second)
	
	// Setup custom signal handling for this long-running command
	util.HandleSignalCustom(cleanup)
	
	return nil
}

// printSessionInfo prints information about the started session
func printSessionInfo(pid int, sessionID, clusterName, localPort, eksHost, instanceID, kubeconfigPath string) {
	fmt.Println("Successfully started ekssm session in background.")
	fmt.Printf("  PID: %d\n", pid)
	fmt.Printf("  SessionID: %s\n", sessionID)
	fmt.Printf("  Cluster: %s\n", clusterName)
	fmt.Printf("  Proxy: localhost:%s -> %s:%s (via %s)\n", localPort, eksHost, constants.EKSApiPort, instanceID)
	fmt.Printf("  Session Kubeconfig: %s\n\n", kubeconfigPath)
	fmt.Println("To use this session, export the KUBECONFIG environment variable:")
	fmt.Printf("  export KUBECONFIG='%s'\n\n", kubeconfigPath)
	fmt.Println("Use 'ekssm session list' to see all sessions.")
	fmt.Println("Use 'ekssm session switch <id>' to get the export command for a session.")
	fmt.Println("Run 'ekssm session stop --session-id <id>' or 'ekssm session stop' to terminate sessions.")
}

func init() {
	sessionCmd.AddCommand(sessionStartCmd)

	// Add command flags
	sessionStartCmd.Flags().StringVar(&startOpts.ClusterName, "cluster-name", "", "Name of the EKS cluster (required)")
	sessionStartCmd.Flags().StringVar(&startOpts.InstanceID, "instance-id", "", "EC2 instance ID of the bastion host (required)")
	sessionStartCmd.Flags().StringVar(&startOpts.LocalPort, "local-port", "", "Local port for forwarding EKS API access (default: dynamically allocated)")

	// Mark required flags
	for _, flag := range []string{"cluster-name", "instance-id"} {
		if err := sessionStartCmd.MarkFlagRequired(flag); err != nil {
			fmt.Fprintf(os.Stderr, "Error marking %s flag required: %v\n", flag, err)
			os.Exit(1)
		}
	}
}
