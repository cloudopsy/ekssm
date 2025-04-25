package util_test

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/cloudopsy/ekssm/internal/util"
)

func TestKubeconfigBasePath(t *testing.T) {
	homeDir, err := os.UserHomeDir()
	require.NoError(t, err)
	expectedPath := filepath.Join(homeDir, ".ekssm", "kubeconfigs")
	assert.Equal(t, expectedPath, util.KubeconfigBasePath())
}

func TestKubeconfigPathForSession(t *testing.T) {
	homeDir, err := os.UserHomeDir()
	require.NoError(t, err)
	basePath := filepath.Join(homeDir, ".ekssm", "kubeconfigs")
	clusterName := "test-cluster"
	sessionID := "test-session-123"
	expectedPath := filepath.Join(basePath, clusterName, fmt.Sprintf("%s.yaml", sessionID))
	assert.Equal(t, expectedPath, util.KubeconfigPathForSession(clusterName, sessionID))
}

func TestKubeconfigPathForRun(t *testing.T) {
	homeDir, err := os.UserHomeDir()
	require.NoError(t, err)
	basePath := filepath.Join(homeDir, ".ekssm", "kubeconfigs")
	clusterName := "another-cluster"
	expectedPath := filepath.Join(basePath, clusterName, "run-temp.yaml")
	assert.Equal(t, expectedPath, util.KubeconfigPathForRun(clusterName))
}

func TestWriteKubeconfig(t *testing.T) {
	tempDir := t.TempDir()
	clusterName := "write-test-cluster"
	sessionID := "write-session-abc"
	kubeconfigPath := filepath.Join(tempDir, clusterName, fmt.Sprintf("%s.yaml", sessionID))
	content := "apiVersion: v1\nkind: Config"

	err := util.WriteKubeconfig(kubeconfigPath, content)
	require.NoError(t, err)

	readBytes, err := os.ReadFile(kubeconfigPath)
	require.NoError(t, err)
	assert.Equal(t, content, string(readBytes))

	// Test overwriting
	newContent := "apiVersion: v1\nkind: Config\nclusters:\n- name: test"
	err = util.WriteKubeconfig(kubeconfigPath, newContent)
	require.NoError(t, err)

	readBytes, err = os.ReadFile(kubeconfigPath)
	require.NoError(t, err)
	assert.Equal(t, newContent, string(readBytes))
}

func TestWriteAndRestoreKubeconfig(t *testing.T) { // Renaming this test might be good later, as it no longer restores.
	// Create a temp dir for tests
	tmpDir, err := os.MkdirTemp("", "ekssm-test-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	testKubeconfigPath := filepath.Join(tmpDir, "test-config.yaml")

	originalContent := "original kubeconfig content"
	newContent := "new kubeconfig content"

	// Create original file to simulate existing state
	if err := os.WriteFile(testKubeconfigPath, []byte(originalContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Write new content
	if err := util.WriteKubeconfig(testKubeconfigPath, newContent); err != nil {
		t.Fatalf("WriteKubeconfig failed: %v", err)
	}

	newFileContent, err := os.ReadFile(testKubeconfigPath)
	if err != nil {
		t.Fatal(err)
	}

	if string(newFileContent) != newContent {
		t.Errorf("New file content doesn't match expected")
	}
}
