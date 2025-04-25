package kubectl

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/cloudopsy/ekssm/internal/logging"
)

// ExecuteCommand runs the specified command with the KUBECONFIG environment variable set.
func ExecuteCommand(args []string, kubeconfigPath string) error {
	if len(args) == 0 {
		return fmt.Errorf("no command arguments provided")
	}
	if kubeconfigPath == "" {
		return fmt.Errorf("kubeconfig path must be provided to ExecuteCommand")
	}

	cmdName := args[0]
	var cmdArgs []string
	if len(args) > 1 {
		cmdArgs = args[1:]
	}

	cmdStr := strings.Join(args, " ")
	logging.Debugf("Executing command: %s with KUBECONFIG=%s", cmdStr, kubeconfigPath)

	cmd := exec.Command(cmdName, cmdArgs...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	// Set the KUBECONFIG environment variable for the command
	cmd.Env = append(os.Environ(), fmt.Sprintf("KUBECONFIG=%s", kubeconfigPath))

	return cmd.Run()
}
