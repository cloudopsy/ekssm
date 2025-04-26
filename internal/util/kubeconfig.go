package util

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/cloudopsy/ekssm/internal/logging"
)

func GetKubeconfigPath() string {
	return filepath.Join(os.Getenv("HOME"), ".kube", "config")
}

func KubeconfigBasePath() string {
	return filepath.Join(os.Getenv("HOME"), ".ekssm", "kubeconfigs")
}

func KubeconfigPathForSession(clusterName, sessionID string) string {
	clusterDir := filepath.Join(KubeconfigBasePath(), clusterName)
	return filepath.Join(clusterDir, fmt.Sprintf("%s.yaml", sessionID))
}

func KubeconfigPathForRun(clusterName string) string {
	clusterDir := filepath.Join(KubeconfigBasePath(), clusterName)
	return filepath.Join(clusterDir, "run-temp.yaml")
}

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
