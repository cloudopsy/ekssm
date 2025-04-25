package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/cloudopsy/ekssm/internal/constants"
	"github.com/cloudopsy/ekssm/internal/logging"
	"github.com/cloudopsy/ekssm/internal/util"
	awsclient "github.com/cloudopsy/ekssm/pkg/aws"
	"github.com/cloudopsy/ekssm/pkg/kubectl"
	"github.com/cloudopsy/ekssm/pkg/proxy"
	"github.com/spf13/cobra"
)

type runOptions struct {
	ClusterName string
	InstanceID  string
	LocalPort   string
}

var runOpts runOptions

var runCmd = &cobra.Command{
	Use:   "run [flags] -- command [args]...",
	Short: "Runs a command against EKS via SSM proxy (starts/stops proxy per command)",
	Long: `Runs a command (typically kubectl) against the specified EKS cluster by\n` +
		`temporarily starting an SSM port forwarding session to the cluster endpoint.\n` +
		`The proxy session is stopped automatically after the command completes.\n\n` +
		`The command and its arguments must be specified after a double dash (--).\n\n` +
		`Example: ekssm run --cluster-name my-cluster --instance-id i-123 -- kubectl get nodes`,
	Args: cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		debug, _ := cmd.Flags().GetBool("debug")
		logging.SetDebug(debug)

		if len(args) == 0 {
			return fmt.Errorf("no command provided after --")
		}
		command := args[0]
		commandArgs := args[1:]

		logging.Debugf("Command to execute: %s %s", command, strings.Join(commandArgs, " "))

		if runOpts.ClusterName == "" || runOpts.InstanceID == "" {
			return fmt.Errorf("--cluster-name and --instance-id are required")
		}

		ctx := context.Background()

		awsClient, err := awsclient.NewClient(ctx)
		if err != nil {
			return fmt.Errorf("failed to initialize AWS client: %w", err)
		}

		logging.Debugf("Retrieving information for EKS cluster: %s", runOpts.ClusterName)
		clusterOutput, err := awsClient.DescribeEKSCluster(ctx, runOpts.ClusterName)
		if err != nil {
			return fmt.Errorf("failed to get EKS cluster info: %w", err)
		}

		if clusterOutput.Cluster == nil || clusterOutput.Cluster.Endpoint == nil {
			return fmt.Errorf("invalid cluster information returned from EKS API")
		}

		eksEndpoint := *clusterOutput.Cluster.Endpoint
		logging.Debugf("EKS API server endpoint: %s", eksEndpoint)

		eksHost := eksEndpoint
		if len(eksHost) > 8 && eksHost[:8] == "https://" {
			eksHost = eksHost[8:]
		}
		logging.Debugf("Using remote host: %s for port forwarding", eksHost)

		ssmProxy := proxy.NewSSMProxy(runOpts.InstanceID, runOpts.LocalPort, eksHost, "443")

		signalCh := make(chan os.Signal, 1)
		signal.Notify(signalCh, os.Interrupt, syscall.SIGTERM)

		proxyErrChan := make(chan error, 1)
		go func() {
			logging.Debug("Starting SSM proxy session in background...")
			if _, err := ssmProxy.StartBackground(); err != nil {
				proxyErrChan <- fmt.Errorf("failed to start SSM proxy: %w", err)
			} else {
				proxyErrChan <- nil
			}
		}()

		defer func() {
			logging.Debug("Stopping SSM proxy session...")
			if err := ssmProxy.Stop(); err != nil {
				logging.Warnf("Failed to stop SSM proxy cleanly: %v", err)
			}
		}()

		kubeconfigPath := util.GetKubeconfigPath()
		backupPath := kubeconfigPath + constants.RunBackupSuffix
		var kubeconfigRestored bool

		defer func() {
			if !kubeconfigRestored {
				logging.Debugf("Restoring original kubeconfig from %s", backupPath)
				if err := util.RestoreKubeconfig(kubeconfigPath, backupPath); err != nil {
					logging.Errorf("CRITICAL: Failed to restore original kubeconfig from backup %s: %v", backupPath, err)
					fmt.Fprintf(os.Stderr, "\nCRITICAL: Failed to restore original kubeconfig from %s. Please restore manually!\n", backupPath)
				}
			}
		}()

		logging.Debugf("Backing up existing kubeconfig %s to %s", kubeconfigPath, backupPath)
		copyErr := util.CopyFile(kubeconfigPath, backupPath)
		if copyErr != nil {
			return fmt.Errorf("failed to backup kubeconfig from %s to %s: %w", kubeconfigPath, backupPath, copyErr)
		}

		endpoint := fmt.Sprintf("https://localhost:%s", runOpts.LocalPort)
		kubeconfigContent := kubectl.GenerateKubeconfig(runOpts.ClusterName, endpoint)

		if err := util.WriteKubeconfig(kubeconfigPath, kubeconfigContent); err != nil {
			return fmt.Errorf("failed to write temporary kubeconfig: %w", err)
		}
		logging.Debugf("Temporary kubeconfig written to %s", kubeconfigPath)

		select {
		case err := <-proxyErrChan:
			if err != nil {
				return err
			}
			logging.Debug("SSM proxy started successfully.")
		case sig := <-signalCh:
			logging.Infof("Received signal: %v. Shutting down...", sig)
			return fmt.Errorf("operation cancelled by signal %v", sig)
		}

		logging.Debugf("Executing command: %v", args)
		if err := kubectl.ExecuteCommand(args); err != nil {
			return err
		}

		logging.Debugf("Command finished. Restoring original kubeconfig from %s", backupPath)
		if err := util.RestoreKubeconfig(kubeconfigPath, backupPath); err != nil {
			logging.Errorf("CRITICAL: Failed to restore original kubeconfig from backup %s: %v", backupPath, err)
			fmt.Fprintf(os.Stderr, "\nCRITICAL: Failed to restore original kubeconfig from %s. Please restore manually!\n", backupPath)
			return fmt.Errorf("failed to restore original kubeconfig: %w", err)
		} else {
			kubeconfigRestored = true
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(runCmd)

	runCmd.Flags().StringVar(&runOpts.ClusterName, "cluster-name", "", "Name of the EKS cluster (required)")
	runCmd.Flags().StringVar(&runOpts.InstanceID, "instance-id", "", "EC2 instance ID of the bastion host (required)")
	runCmd.Flags().StringVar(&runOpts.LocalPort, "local-port", "9443", "Local port for forwarding EKS API access")

	if err := runCmd.MarkFlagRequired("cluster-name"); err != nil {
		fmt.Fprintf(os.Stderr, "Error marking cluster-name flag required: %v\n", err)
		os.Exit(1)
	}
	if err := runCmd.MarkFlagRequired("instance-id"); err != nil {
		fmt.Fprintf(os.Stderr, "Error marking instance-id flag required: %v\n", err)
		os.Exit(1)
	}
}
