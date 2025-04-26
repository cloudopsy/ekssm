# EKSSM - EKS SSM Proxy

EKSSM is a robust CLI tool that allows you to run Kubernetes CLI commands (like kubectl, helm, etc.) against an Amazon EKS cluster via an SSM-enabled instance. This enables secure access to EKS clusters without requiring direct network access to the Kubernetes API server.

## Features

- Securely connect to EKS clusters via SSM
- **Two Modes:**
  - **Run Mode:** Execute single commands via a temporary proxy session and dedicated kubeconfig.
  - **Session Mode:** Manage multiple, persistent background proxy sessions, each with its own dedicated kubeconfig.
- **Multi-Session Support:** Run and manage concurrent sessions to the same or different clusters.
- **Dedicated Kubeconfig Files:** Each session (and `run` command) uses a separate kubeconfig file stored in `$HOME/.ekssm/kubeconfigs/`, leaving your default `~/.kube/config` untouched.
- **Dynamic Port Allocation:** `session start` automatically finds an available local port, preventing conflicts (can be overridden with `--local-port`).
- **Session Management Commands:**
  - `session start`: Begin a new background session.
  - `session stop`: Stop a specific session by ID or all sessions.
  - `session list`: View details of all active sessions.
  - `session switch`: Get the command to point `KUBECONFIG` to a specific session's file.
- **Shell Integration:** Optional shell hooks to automatically set environment variables in your current shell.
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

The `run` command starts a temporary SSM proxy, executes your command using a temporary kubeconfig, and then stops the proxy and cleans up the temporary file. Ideal for single commands or scripts without interfering with active sessions.

```bash
# Basic kubectl usage:
ekssm run --instance-id <INSTANCE_ID> --cluster-name <CLUSTER_NAME> -- kubectl get pods

# With Helm:
ekssm run --instance-id <INSTANCE_ID> --cluster-name <CLUSTER_NAME> -- helm list

# With custom local port (rarely needed):
ekssm run --instance-id <INSTANCE_ID> --cluster-name <CLUSTER_NAME> --local-port 8443 -- kubectl get nodes

# Enable debug logging:
ekssm run --instance-id <INSTANCE_ID> --cluster-name <CLUSTER_NAME> --debug -- kubectl get pods -A
```

**Important:** The command and its arguments *must* follow the double dash (`--`). The `run` command sets the `KUBECONFIG` environment variable internally only for the child process running the command.

### Session Commands (Persistent Sessions)

The `session` commands manage persistent background SSM proxy sessions, each with its own dedicated kubeconfig. This is useful when you need to run multiple commands against one or more clusters.

**Starting a Session:**

```bash
# Start a session with dynamic port allocation
ekssm session start --instance-id <INSTANCE_ID> --cluster-name <CLUSTER_NAME>

# Start a session specifying a local port (if needed)
ekssm session start --instance-id <INSTANCE_ID> --cluster-name <CLUSTER_NAME> --local-port <PORT>
```

This command:
- Starts the SSM port forwarding in the background using a dynamically allocated (or specified) local port.
- Generates a unique Session ID.
- Creates a dedicated kubeconfig file at `$HOME/.ekssm/kubeconfigs/<cluster-name>/<session-id>.yaml`.
- Saves session details (PID, Port, Kubeconfig Path, etc.) to `$HOME/.ekssm/session.json`.
- Prints the `export KUBECONFIG=...` command needed to use the session.

**Listing Active Sessions:**

```bash
ekssm session list
```
Displays a table of all active sessions, including their IDs, cluster names, PIDs, ports, and kubeconfig paths.

**Switching KUBECONFIG for a Session:**

```bash
# Get the export command for a specific session
ekssm session switch <SESSION_ID>

# Example: Use the output in your shell
export KUBECONFIG=$(ekssm session switch <SESSION_ID>)
# or copy-paste the output: export KUBECONFIG='/path/to/session.yaml'

# If you've set up shell integration (see Shell Integration section below), you can use:
ekssm session switch <SESSION_ID>  # This will set KUBECONFIG automatically
```
This command prints the `export KUBECONFIG=...` command pointing to the specified session's kubeconfig file. **It does not execute the command itself** unless you've set up shell integration.

**Stopping Sessions:**

```bash
# Stop a specific session by ID
ekssm session stop --session-id <SESSION_ID>

# Stop ALL active sessions
ekssm session stop
```

This command:
- Stops the specified background SSM proxy process(es).
- Removes the dedicated kubeconfig file(s).
- Removes the session entry(ies) from the state file (`$HOME/.ekssm/session.json`).

### Flags

- `--instance-id` (Required for `run`, `session start`): EC2 instance ID with SSM agent.
- `--cluster-name` (Required for `run`, `session start`): EKS cluster name.
- `--local-port` (Optional for `run`, `session start`): Specific local port for the proxy. If omitted or "0", a dynamic port is allocated.
- `--session-id` (Optional for `session stop`): Specific session ID to stop. If omitted, all sessions are stopped.
- `--debug` (Optional, Global): Enable verbose debug logging.

## Shell Integration

EKSSM can be integrated with your shell to automatically set environment variables (like `KUBECONFIG`) in your current shell session. This allows commands like `ekssm session switch` to directly modify your shell environment without requiring you to manually export the variables.

To set up shell integration:

1. Add the following line to your shell configuration file (`.bashrc`, `.zshrc`, etc.):

```bash
# For bash or zsh
eval "$(ekssm shell bash)"  # or replace bash with zsh
```

2. Restart your shell or source the configuration file:

```bash
source ~/.bashrc  # or .zshrc, etc.
```

3. Now you can directly use the commands without manual exports:

```bash
# This will automatically set KUBECONFIG in your current shell
ekssm session switch <SESSION_ID>
```

The shell integration works by overriding the `ekssm` command with a shell function that intercepts certain commands and applies their output to the current shell environment.

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
1. Fetches EKS cluster info to get the API server endpoint.
2. Starts an SSM port forwarding session (`AWS-StartPortForwardingSessionToRemoteHost`) from `localhost:<local-port>` to `<eks-endpoint>:443` via the specified EC2 instance.
3. Waits for the local port to be available.
4. Generates a temporary kubeconfig file at `$HOME/.ekssm/kubeconfigs/<cluster-name>/run-temp.yaml` pointing to `localhost:<local-port>`.
5. Executes the user-provided command (e.g., `kubectl get pods`) with the `KUBECONFIG` environment variable set to the temporary file's path.
6. Terminates the SSM session and stops the `session-manager-plugin` process.
7. Removes the temporary kubeconfig file.

**Session Mode:**
1. **`start`**:
   - Fetches EKS cluster info.
   - Determines the local port (dynamic or user-specified).
   - Starts the SSM port forwarding session in the background.
   - Generates a unique Session ID.
   - Writes a dedicated kubeconfig file to `$HOME/.ekssm/kubeconfigs/<cluster-name>/<session-id>.yaml` pointing to `localhost:<local-port>`.
   - Writes the process ID and session details (including Kubeconfig path) to `$HOME/.ekssm/session.json`.
2. **`list`**: Reads `$HOME/.ekssm/session.json` and displays active sessions.
3. **`switch <id>`**: Reads `$HOME/.ekssm/session.json`, finds the session by ID, and prints the `export KUBECONFIG=...` command using the stored path.
4. **`stop [--session-id <id>]`**:
   - Reads session(s) from `$HOME/.ekssm/session.json`.
   - Terminates the SSM session process(es) by PID.
   - Removes the dedicated kubeconfig file(s).
   - Removes the session entry(ies) from the state file.

The tool uses AWS Systems Manager's remote host port forwarding feature to securely connect to the EKS cluster's API server through an SSM-enabled instance, without requiring the instance to have direct network access to the cluster.

## License

MIT
