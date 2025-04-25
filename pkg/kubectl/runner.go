package kubectl

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	
	"github.com/cloudopsy/ekssm/internal/logging"
)

func ExecuteCommand(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("no command arguments provided")
	}
	
	cmdName := args[0]
	var cmdArgs []string
	if len(args) > 1 {
		cmdArgs = args[1:]
	}
	
	cmdStr := strings.Join(args, " ")
	logging.Debugf("Executing command: %s", cmdStr)
	
	cmd := exec.Command(cmdName, cmdArgs...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	
	return cmd.Run()
}
