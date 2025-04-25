# EKSSM - EKS SSM Proxy

EKSSM is a robust CLI tool that allows you to run Kubernetes CLI commands (like kubectl, helm, etc.) against an Amazon EKS cluster via an SSM-enabled instance. This enables secure access to EKS clusters without requiring direct network access to the Kubernetes API server.

## Features

- Securely connect to EKS clusters via SSM
- Temporary kubeconfig management with automatic cleanup
- Support for all kubectl commands
- Proper signal handling and cleanup
- Detailed logging with debug option

## Installation

```bash
# Build from source
git clone https://github.com/cloudopsy/ekssm.git
cd ekssm
go build -o ekssm ./cmd/ekssm

# Move to a directory in your PATH
mv ekssm /usr/local/bin/
```

## Usage

```bash
# Basic usage with kubectl
ekssm --instance-id i-0123456789abcdef0 --cluster-name my-cluster -- kubectl get pods

# With Helm
ekssm --instance-id i-0123456789abcdef0 --cluster-name my-cluster -- helm list

# With custom local port
ekssm --instance-id i-0123456789abcdef0 --cluster-name my-cluster --local-port 8443 -- kubectl get nodes

# Enable debug logging
ekssm --instance-id i-0123456789abcdef0 --cluster-name my-cluster --debug -- kubectl get pods -A
```

## Requirements

- AWS CLI configured with access to the EKS and SSM services
- EC2 instance with SSM enabled and network access to the EKS API server
- kubectl installed locally
- `session-manager-plugin` installed on your local machine (see [AWS documentation](https://docs.aws.amazon.com/systems-manager/latest/userguide/session-manager-working-with-install-plugin.html))
- Proper IAM permissions for both EKS and SSM operations:
  - `ssm:StartSession` with the document `AWS-StartPortForwardingSessionToRemoteHost`
  - `ssm:TerminateSession`
  - `eks:DescribeCluster`
- The SSM agent on the bastion instance must be version 2.3.672.0 or later to support remote port forwarding

## Troubleshooting

### "Invalid Operation" Error

If you encounter an "Invalid Operation" error when starting the SSM session, check:

1. **IAM Permissions**: Ensure your IAM user/role has the proper permissions listed in the Requirements section.

2. **SSM Agent Version**: The instance must have an SSM agent version that supports remote port forwarding (2.3.672.0 or later).
   ```bash
   aws ssm describe-instance-information --instance-id i-0123456789abcdef0 --query "InstanceInformationList[0].AgentVersion"
   ```

3. **Network Connectivity**: The bastion instance must have network access to the EKS API server endpoint. Check:
   - Security groups allow outbound connections to port 443
   - The instance is in a VPC with proper routing to the EKS control plane
   - The EKS cluster's API server endpoint is accessible from the instance's subnet

4. **Validate Session Manager Plugin**: Ensure the session-manager-plugin is installed correctly:
   ```bash
   session-manager-plugin --version
   ```

## How It Works

1. Fetches the EKS cluster information to determine the cluster endpoint
2. Establishes an SSM port forwarding session to the remote EKS endpoint via the specified EC2 instance using AWS-StartPortForwardingSessionToRemoteHost
3. Creates a temporary kubeconfig that points to the local forwarded port
4. Executes the kubectl command using the temporary kubeconfig
5. Restores the original kubeconfig and cleans up resources

The tool uses AWS Systems Manager's remote host port forwarding feature to securely connect to the EKS cluster's API server through an SSM-enabled instance, without requiring the instance to have direct access to the cluster.

## License

MIT
