// Package util provides internal utility functions for file operations, networking,
// and kubeconfig management specific to ekssm.
package util

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/cloudopsy/ekssm/internal/logging"
)

// GetKubeconfigPath returns the path to the default kubeconfig file.
func GetKubeconfigPath() string {
	return filepath.Join(os.Getenv("HOME"), ".kube", "config")
}

// KubeconfigBasePath returns the base path for ekssm-managed kubeconfig files.
func KubeconfigBasePath() string {
	return filepath.Join(os.Getenv("HOME"), ".ekssm", "kubeconfigs")
}

// KubeconfigPathForSession returns the path for a session-specific kubeconfig file.
func KubeconfigPathForSession(clusterName, sessionID string) string {
	clusterDir := filepath.Join(KubeconfigBasePath(), clusterName)
	return filepath.Join(clusterDir, fmt.Sprintf("%s.yaml", sessionID))
}

// KubeconfigPathForRun returns the path for a temporary kubeconfig file used during a run.
func KubeconfigPathForRun(clusterName string) string {
	clusterDir := filepath.Join(KubeconfigBasePath(), clusterName)
	return filepath.Join(clusterDir, "run-temp.yaml")
}

// WriteKubeconfig writes a kubeconfig file to the specified path, creating parent
// directories if needed.
func WriteKubeconfig(path string, content string) error {
	logging.Debugf("Writing kubeconfig to %s", path)
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0750); err != nil {
		return fmt.Errorf("failed to create kubeconfig directory %s: %w", dir, err)
	}
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		return fmt.Errorf("failed to write kubeconfig: %w", err)
	}
	return nil
}
