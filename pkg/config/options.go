package config

import (
	"fmt"
	"os"
)

type Options struct {
	InstanceID  string
	ClusterName string
	LocalPort   string
	CommandArgs []string
	Debug       bool
}

func (o *Options) Validate() error {
	if o.InstanceID == "" {
		return fmt.Errorf("instance ID is required")
	}
	
	if o.ClusterName == "" {
		return fmt.Errorf("cluster name is required")
	}
	
	if len(o.CommandArgs) == 0 {
		return fmt.Errorf("command arguments after -- are required")
	}
	
	return nil
}

func (o *Options) PrintUsage() {
	fmt.Fprintf(os.Stderr, `Usage: ekssm [options] -- command [args...]

Establishes an SSM port forwarding session to an EKS cluster via a bastion host,
updates kubeconfig to use the local proxy, and then executes the specified command.

Options:
  --instance-id string   EC2 instance ID of the bastion host (required)
  --cluster-name string  Name of the EKS cluster (required)
  --local-port string    Local port for forwarding EKS API access (default "9443")
  --debug                Enable debug logging
  --help                 Show this help message

Examples:
  ekssm --instance-id i-0123... --cluster-name my-cluster -- kubectl get pods
  ekssm --instance-id i-0123... --cluster-name my-cluster -- helm list -n monitoring
  ekssm --instance-id i-0123... --cluster-name my-cluster --local-port 8443 -- kubectl version
`)
}
