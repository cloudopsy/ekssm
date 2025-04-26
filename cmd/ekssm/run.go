package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/cloudopsy/ekssm/internal/constants"
	"github.com/cloudopsy/ekssm/internal/logging"
	"github.com/cloudopsy/ekssm/internal/util"
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
	Args: cobra.MinimumNArgs(1),
	RunE: runCommand,
}

func runCommand(cmd *cobra.Command, args []string) error {
	debug, _ := cmd.Flags().GetBool("debug")
	logging.SetDebug(debug)

	if len(args) == 0 {
		return fmt.Errorf("no command provided after --")
	}

	logging.Debugf("Command to execute: %s", strings.Join(args, " "))

	if runOpts.ClusterName == "" || runOpts.InstanceID == "" {
		return fmt.Errorf("--cluster-name and --instance-id are required")
	}

	ctx, cancelCtx := util.SignalContext()
	defer cancelCtx()

	eksHost, err := util.EKSClusterEndpoint(ctx, runOpts.ClusterName)
	if err != nil {
		return err
	}

	localPort := runOpts.LocalPort
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

	ssmProxy := proxy.NewSSMProxy(runOpts.InstanceID, localPort, eksHost, constants.EKSApiPort)

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

	kubeconfigPath := util.KubeconfigPathForRun(runOpts.ClusterName)

	defer func() {
		logging.Debugf("Removing temporary kubeconfig: %s", kubeconfigPath)
		if err := os.Remove(kubeconfigPath); err != nil {
			if !os.IsNotExist(err) {
				logging.Warnf("Failed to remove temporary kubeconfig %s: %v", kubeconfigPath, err)
			}
		}
	}()

	endpoint := fmt.Sprintf("https://localhost:%s", localPort)
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
	case <-ctx.Done():
		logging.Info("Operation canceled.")
		return fmt.Errorf("operation cancelled by signal")
	}

	logging.Debugf("Executing command: %v with KUBECONFIG=%s", args, kubeconfigPath)
	if err := kubectl.ExecuteCommand(args, kubeconfigPath); err != nil {
		return err
	}

	logging.Debugf("Command finished successfully.")
	return nil
}

func init() {
	rootCmd.AddCommand(runCmd)

	runCmd.Flags().StringVar(&runOpts.ClusterName, "cluster-name", "", "Name of the EKS cluster (required)")
	runCmd.Flags().StringVar(&runOpts.InstanceID, "instance-id", "", "EC2 instance ID of the bastion host (required)")
	runCmd.Flags().StringVar(&runOpts.LocalPort, "local-port", "", "Local port for forwarding EKS API access (default: dynamically allocated)")

	for _, flag := range []string{"cluster-name", "instance-id"} {
		if err := runCmd.MarkFlagRequired(flag); err != nil {
			fmt.Fprintf(os.Stderr, "Error marking %s flag required: %v\n", flag, err)
			os.Exit(1)
		}
	}
}
