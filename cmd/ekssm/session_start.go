package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/cloudopsy/ekssm/internal/constants"
	"github.com/cloudopsy/ekssm/internal/logging"
	"github.com/cloudopsy/ekssm/internal/state"
	"github.com/cloudopsy/ekssm/internal/util"
	"github.com/cloudopsy/ekssm/pkg/aws"
	"github.com/cloudopsy/ekssm/pkg/kubectl"
	"github.com/cloudopsy/ekssm/pkg/proxy"
)

var (
	startClusterName string
	startInstanceID  string
	startLocalPort   string
)

var sessionStartCmd = &cobra.Command{
	Use:   "start",
	Short: "Start a persistent SSM proxy session in the background",
	Long:  `Starts an SSM port forwarding session to the EKS cluster via the specified instance. This session runs in the background, and a kubeconfig file is generated. Use 'ekssm session stop' to terminate it.`,
	RunE:  startSession,
}

func init() {
	sessionCmd.AddCommand(sessionStartCmd)

	sessionStartCmd.Flags().StringVar(&startClusterName, "cluster-name", "", "Name of the EKS cluster (required)")
	sessionStartCmd.Flags().StringVar(&startInstanceID, "instance-id", "", "EC2 instance ID of the bastion host (required)")
	sessionStartCmd.Flags().StringVar(&startLocalPort, "local-port", "9443", "Local port for forwarding EKS API access")

	sessionStartCmd.MarkFlagRequired("cluster-name")
	sessionStartCmd.MarkFlagRequired("instance-id")
}

func startSession(cmd *cobra.Command, args []string) error {
	logging.SetDebug(debug)

	currentState, err := state.ReadState()
	if err != nil {
		return fmt.Errorf("failed to read session state: %w", err)
	}
	if currentState != nil {
		return fmt.Errorf("an active session is already running (PID: %d, Cluster: %s). Use 'ekssm session stop' first", currentState.PID, currentState.ClusterName)
	}

	logging.Info("Starting new ekssm session...")

	ctx := context.Background()
	awsClient, err := aws.NewClient(ctx)
	if err != nil {
		return fmt.Errorf("failed to initialize AWS client: %w", err)
	}
	logging.Debugf("Retrieving information for EKS cluster: %s", startClusterName)
	clusterOutput, err := awsClient.DescribeEKSCluster(ctx, startClusterName)
	if err != nil {
		return fmt.Errorf("failed to get EKS cluster info for '%s': %w", startClusterName, err)
	}
	if clusterOutput.Cluster == nil || clusterOutput.Cluster.Endpoint == nil {
		return fmt.Errorf("invalid cluster information returned from EKS API for '%s'", startClusterName)
	}
	eksEndpoint := *clusterOutput.Cluster.Endpoint
	eksHost := strings.TrimPrefix(eksEndpoint, "https://")
	logging.Debugf("Using EKS endpoint host: %s", eksHost)

	ssmProxy := proxy.NewSSMProxy(startInstanceID, startLocalPort, eksHost, "443")
	pid, err := ssmProxy.StartBackground()
	if err != nil {
		return fmt.Errorf("failed to start SSM proxy: %w", err)
	}
	logging.Infof("SSM proxy started successfully (PID: %d, SessionID: %s)", pid, ssmProxy.SessionID)

	kubeconfigPath := util.GetKubeconfigPath()
	backupPath := kubeconfigPath + constants.SessionBackupSuffix

	logging.Debugf("Backing up existing kubeconfig '%s' to '%s'", kubeconfigPath, backupPath)
	if err := util.CopyFile(kubeconfigPath, backupPath); err != nil {
		logging.Errorf("Failed to backup kubeconfig: %v. Attempting to stop proxy...", err)
		_ = ssmProxy.Stop()
		return fmt.Errorf("failed to backup kubeconfig from %s to %s: %w", kubeconfigPath, backupPath, err)
	}

	localEndpoint := fmt.Sprintf("https://localhost:%s", startLocalPort)
	kubeconfigContent := kubectl.GenerateKubeconfig(startClusterName, localEndpoint)

	logging.Debugf("Writing new kubeconfig for session to '%s'", kubeconfigPath)
	if err := util.WriteKubeconfig(kubeconfigPath, kubeconfigContent); err != nil {
		logging.Errorf("Failed to write session kubeconfig: %v. Restoring backup and stopping proxy...", err)
		_ = util.RestoreKubeconfig(kubeconfigPath, backupPath)
		_ = ssmProxy.Stop()
		return fmt.Errorf("failed to write kubeconfig to %s: %w", kubeconfigPath, err)
	}

	newSessionState := &state.SessionState{
		PID:         pid,
		SessionID:   ssmProxy.SessionID,
		ClusterName: startClusterName,
		InstanceID:  startInstanceID,
		LocalPort:   startLocalPort,
	}

	if err := state.WriteState(newSessionState); err != nil {
		logging.Errorf("CRITICAL: Failed to write session state file: %v. Kubeconfig may be inconsistent! Restoring backup and stopping proxy...", err)
		_ = util.RestoreKubeconfig(kubeconfigPath, backupPath)
		_ = ssmProxy.Stop()
		_ = state.ClearState()
		return fmt.Errorf("failed to write session state file: %w", err)
	}

	fmt.Printf("Successfully started ekssm session in background.\n")
	fmt.Printf("  PID: %d\n", pid)
	fmt.Printf("  Cluster: %s\n", startClusterName)
	fmt.Printf("  Proxy: localhost:%s -> %s:%s (via %s)\n", startLocalPort, eksHost, "443", startInstanceID)
	fmt.Printf("Your kubeconfig at '%s' has been updated.\n", kubeconfigPath)
	fmt.Printf("Run 'ekssm session stop' to terminate the session and restore your original kubeconfig.\n")

	return nil
}
