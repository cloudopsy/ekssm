package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/cloudopsy/ekssm/internal/constants"
	"github.com/cloudopsy/ekssm/internal/logging"
	"github.com/cloudopsy/ekssm/internal/util"
	awsclient "github.com/cloudopsy/ekssm/pkg/aws"
	"github.com/cloudopsy/ekssm/pkg/kubectl"
	"github.com/cloudopsy/ekssm/pkg/proxy"
)

type runOptions struct {
	ClusterName string
	InstanceID  string
	LocalPort   string
}

var runOpts runOptions

var runCmd = &cobra.Command{
	Use:   "run [flags] -- <command> [args...]",
	Short: "Run a command with temporary EKS access via SSM proxy",
	Long: `Establishes an SSM port forwarding session, generates a temporary kubeconfig, 
and executes the specified command with the KUBECONFIG environment variable set. 
The session and kubeconfig are automatically cleaned up when the command finishes.

Example: ekssm run --cluster-name my-cluster --instance-id i-12345 -- kubectl get nodes`,
	Args: cobra.MinimumNArgs(1), // Ensures there's at least a command after '--'
	RunE: func(cmd *cobra.Command, args []string) error {
		debug, _ := cmd.Flags().GetBool("debug")
		logging.SetDebug(debug)

		if len(args) == 0 {
			return fmt.Errorf("no command provided after --")
		}
		// Command and args are now in 'args'
		logging.Debugf("Command to execute: %s", strings.Join(args, " "))

		if runOpts.ClusterName == "" || runOpts.InstanceID == "" {
			return fmt.Errorf("--cluster-name and --instance-id are required")
		}

		ctx := context.Background()

		awsClient, err := awsclient.NewClient(ctx)
		if err != nil {
			return fmt.Errorf("failed to initialize AWS client: %w", err)
		}

		clusterOutput, err := awsClient.DescribeEKSCluster(ctx, runOpts.ClusterName)
		if err != nil {
			return fmt.Errorf("failed to describe EKS cluster: %w", err)
		}
		if clusterOutput.Cluster == nil || clusterOutput.Cluster.Endpoint == nil || *clusterOutput.Cluster.Endpoint == "" {
			return fmt.Errorf("invalid cluster information returned from EKS API")
		}

		eksEndpoint := *clusterOutput.Cluster.Endpoint
		logging.Debugf("EKS API server endpoint: %s", eksEndpoint)

		// Extract host from https://... endpoint
		eksHost := strings.TrimPrefix(eksEndpoint, "https://")
		logging.Debugf("Using remote host: %s for port forwarding", eksHost)

		ssmProxy := proxy.NewSSMProxy(runOpts.InstanceID, runOpts.LocalPort, eksHost, constants.EKSApiPort)

		// Channel for proxy errors and signals
		signalCh := make(chan os.Signal, 1)
		signal.Notify(signalCh, os.Interrupt, syscall.SIGTERM)
		proxyErrChan := make(chan error, 1)

		// Start proxy in background
		go func() {
			logging.Debug("Starting SSM proxy session in background...")
			if _, err := ssmProxy.StartBackground(); err != nil {
				proxyErrChan <- fmt.Errorf("failed to start SSM proxy: %w", err)
			} else {
				proxyErrChan <- nil // Signal success
			}
		}()

		// Ensure proxy is stopped eventually
		defer func() {
			logging.Debug("Stopping SSM proxy session...")
			if err := ssmProxy.Stop(); err != nil {
				logging.Warnf("Failed to stop SSM proxy cleanly: %v", err)
			}
		}()

		// Generate path for the temporary kubeconfig for this run
		kubeconfigPath := util.KubeconfigPathForRun(runOpts.ClusterName)

		// Ensure temporary kubeconfig is cleaned up
		defer func() {
			logging.Debugf("Removing temporary kubeconfig: %s", kubeconfigPath)
			if err := os.Remove(kubeconfigPath); err != nil {
				if !os.IsNotExist(err) {
					logging.Warnf("Failed to remove temporary kubeconfig %s: %v", kubeconfigPath, err)
				}
			}
		}()

		// Generate kubeconfig content
		endpoint := fmt.Sprintf("https://localhost:%s", runOpts.LocalPort)
		kubeconfigContent := kubectl.GenerateKubeconfig(runOpts.ClusterName, endpoint)

		// Write the temporary kubeconfig
		if err := util.WriteKubeconfig(kubeconfigPath, kubeconfigContent); err != nil {
			return fmt.Errorf("failed to write temporary kubeconfig: %w", err)
		}
		logging.Debugf("Temporary kubeconfig written to %s", kubeconfigPath)

		// Wait for proxy to start or signal
		select {
		case err := <-proxyErrChan:
			if err != nil {
				return err // Proxy failed to start
			}
			logging.Debug("SSM proxy started successfully.")
		case sig := <-signalCh:
			logging.Infof("Received signal: %v. Shutting down...", sig)
			return fmt.Errorf("operation cancelled by signal %v", sig)
		}

		// Execute the user's command with the temporary kubeconfig
		logging.Debugf("Executing command: %v with KUBECONFIG=%s", args, kubeconfigPath)
		if err := kubectl.ExecuteCommand(args, kubeconfigPath); err != nil {
			// Return the error from the user's command
			// The defer statements will handle cleanup
			return err
		}

		// Command finished successfully, cleanup will happen via defers
		logging.Debugf("Command finished successfully.")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(runCmd)

	runCmd.Flags().StringVar(&runOpts.ClusterName, "cluster-name", "", "Name of the EKS cluster (required)")
	runCmd.Flags().StringVar(&runOpts.InstanceID, "instance-id", "", "EC2 instance ID of the bastion host (required)")
	runCmd.Flags().StringVar(&runOpts.LocalPort, "local-port", constants.DefaultLocalPort, "Local port for forwarding EKS API access")

	// Mark required flags
	if err := runCmd.MarkFlagRequired("cluster-name"); err != nil {
		fmt.Fprintf(os.Stderr, "Error marking cluster-name flag required: %v\n", err)
		os.Exit(1)
	}
	if err := runCmd.MarkFlagRequired("instance-id"); err != nil {
		fmt.Fprintf(os.Stderr, "Error marking instance-id flag required: %v\n", err)
		os.Exit(1)
	}
}
