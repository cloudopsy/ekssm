package kubectl_test

import (
	"strings"
	"testing"

	"github.com/cloudopsy/ekssm/pkg/kubectl"
)

func TestGenerateKubeconfig(t *testing.T) {
	// Arrange
	expectedClusterName := "test-cluster"
	expectedEndpoint := "https://localhost:9443"

	// Act
	kubeconfig := kubectl.GenerateKubeconfig(expectedClusterName, expectedEndpoint)

	// Assert
	if !strings.Contains(kubeconfig, expectedClusterName) {
		t.Errorf("Expected kubeconfig to contain cluster name %s", expectedClusterName)
	}

	if !strings.Contains(kubeconfig, expectedEndpoint) {
		t.Errorf("Expected kubeconfig to contain endpoint %s", expectedEndpoint)
	}
}
