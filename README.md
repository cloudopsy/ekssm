# EKSSM - EKS SSM Proxy

EKSSM is a robust CLI tool that allows you to run Kubernetes CLI commands (like kubectl, helm, etc.) against an Amazon EKS cluster via an SSM-enabled instance. This enables secure access to EKS clusters without requiring direct network access to the Kubernetes API server.

## Features

- Securely connect to EKS clusters via SSM
- **Two Modes:**
  - **Run Mode:** Execute single commands via a temporary proxy session.
  - **Session Mode:** Manage a persistent background proxy session for multiple commands.
- Kubeconfig management (backup, temporary generation, restore) with automatic cleanup
- Support for all standard Kubernetes CLI commands (kubectl, helm, etc.)
- Proper signal handling and cleanup
- Detailed logging with `--debug` option

## Installation

**Using Make (Recommended):**

```bash
# Build the binary
make build

# Install the binary (requires GOPATH to be set)
make install
```

**Manual Build:**

```bash
# Build from source
git clone https://github.com/cloudopsy/ekssm.git
cd ekssm
go build -o ekssm ./cmd/ekssm

# Move to a directory in your PATH
mv ekssm /usr/local/bin/
```

## Usage

### Run Command (Temporary Session)

The `run` command starts a temporary SSM proxy, executes your command, and then stops the proxy and restores your original kubeconfig. Ideal for single commands or scripts.

```bash
# Basic kubectl usage:
ekssm run --instance-id <INSTANCE_ID> --cluster-name <CLUSTER_NAME> -- kubectl get pods

# With Helm:
ekssm run --instance-id <INSTANCE_ID> --cluster-name <CLUSTER_NAME> -- helm list

# With custom local port:
ekssm run --instance-id <INSTANCE_ID> --cluster-name <CLUSTER_NAME> --local-port 8443 -- kubectl get nodes

# Enable debug logging:
ekssm run --instance-id <INSTANCE_ID> --cluster-name <CLUSTER_NAME> --debug -- kubectl get pods -A
```

**Important:** The command and its arguments *must* follow the double dash (`--`).

### Session Commands (Persistent Session)

The `session` commands manage a persistent background SSM proxy session. This is useful when you need to run multiple commands against the cluster without the overhead of starting/stopping the proxy each time.

**Starting a Session:**

```bash
ekssm session start --instance-id <INSTANCE_ID> --cluster-name <CLUSTER_NAME> [--local-port <PORT>]
```

This command:
- Starts the SSM port forwarding in the background.
- Backs up your current default kubeconfig (e.g., `~/.kube/config`) to `~/.kube/config.ekssm-bak`.
- Writes a new default kubeconfig pointing to the local proxy (`localhost:<PORT>`).
- Saves session details (PID, etc.) to `~/.ekssm/session.json`.

After starting a session, you can use `kubectl`, `helm`, etc. directly, and they will automatically use the proxied connection.

**Stopping a Session:**

```bash
ekssm session stop
```

This command:
- Stops the background SSM proxy process.
- Restores your original kubeconfig from the backup.
- Deletes the session state file (`~/.ekssm/session.json`).

**Flags:**
- `--instance-id` (Required for `run`, `session start`): EC2 instance ID with SSM agent.
- `--cluster-name` (Required for `run`, `session start`): EKS cluster name.
- `--local-port` (Optional, default: `9443` for `run`, `session start`): Local port for the proxy.
- `--debug` (Optional, Global): Enable verbose debug logging.

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

EKSSM leverages AWS Systems Manager Session Manager's port forwarding capability.

**Run Mode:**
1.  Fetches EKS cluster info to get the API server endpoint.
2.  Starts an SSM port forwarding session (`AWS-StartPortForwardingSessionToRemoteHost`) from `localhost:<local-port>` to `<eks-endpoint>:443` via the specified EC2 instance.
3.  Waits for the local port to be available.
4.  Backs up the current kubeconfig (`~/.kube/config` or `$KUBECONFIG`) to `.ekssm-run-bak`.
5.  Writes a temporary kubeconfig pointing to `localhost:<local-port>`.
6.  Executes the user-provided command (e.g., `kubectl get pods`).
7.  Restores the original kubeconfig from the backup.
8.  Terminates the SSM session and stops the `session-manager-plugin` process.

**Session Mode:**
1.  **`start`**: Performs steps 1-3 similar to `run` mode.
2.  Backs up the current kubeconfig to `.ekssm-bak`.
3.  Writes a new kubeconfig pointing to `localhost:<local-port>`.
4.  Writes the process ID and session details to `~/.ekssm/session.json`.
5.  The SSM proxy continues running in the background.
6.  **`stop`**: Reads PID from `~/.ekssm/session.json`.
7.  Terminates the SSM session and process.
8.  Restores the original kubeconfig from `.ekssm-bak`.
9.  Deletes the state file.

The tool uses AWS Systems Manager's remote host port forwarding feature to securely connect to the EKS cluster's API server through an SSM-enabled instance, without requiring the instance to have direct network access to the cluster.

## Makefile Targets

Useful commands available via the Makefile:

- `make build`: Compile the `ekssm` binary into the `bin/` directory.
- `make install`: Build and copy the binary to `$GOPATH/bin`.
- `make test`: Run unit tests.
- `make lint`: Run the Go linter (`golangci-lint`). Requires linter to be installed.
- `make fmt`: Format Go code using `go fmt`.
- `make clean`: Remove the `bin/` directory.
- `make run ARGS="..."`: Build and run the tool locally, passing arguments via `ARGS` (e.g., `make run ARGS="session start --cluster-name test --instance-id i-123"`).

## License

MIT
