package kubectl_test

import (
	"os"
	"os/exec"
	"testing"
	
	"github.com/cloudopsy/ekssm/pkg/kubectl"
)

func TestExecuteCommand(t *testing.T) {
	// Skip if running in CI or if 'echo' command not available
	if os.Getenv("CI") != "" {
		t.Skip("Skipping test in CI environment")
	}
	
	// Test with echo command
	err := kubectl.ExecuteCommand([]string{"echo", "test"})
	if err != nil {
		t.Errorf("ExecuteCommand failed with echo: %v", err)
	}
	
	// Test with a command that doesn't exist
	err = kubectl.ExecuteCommand([]string{"command-that-does-not-exist"})
	if err == nil {
		t.Error("ExecuteCommand should have failed with non-existent command")
	}
	
	// Test with an empty command list
	err = kubectl.ExecuteCommand([]string{})
	if err == nil {
		t.Error("ExecuteCommand should have failed with empty arguments")
	}
	
	// Test with kubectl command (only if kubectl is available)
	if _, err := exec.LookPath("kubectl"); err == nil {
		err := kubectl.ExecuteCommand([]string{"kubectl", "version", "--client"})
		if err != nil {
			t.Errorf("ExecuteCommand failed with kubectl: %v", err)
		}
	}
}