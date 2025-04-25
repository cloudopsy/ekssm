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

func BackupKubeconfig(kubeconfigPath string) (string, error) {
	backupPath := kubeconfigPath + ".bak"
	logging.Debugf("Backing up kubeconfig from %s to %s", kubeconfigPath, backupPath)

	if _, err := os.Stat(kubeconfigPath); err == nil {
		if err := os.Rename(kubeconfigPath, backupPath); err != nil {
			return "", fmt.Errorf("failed to backup kubeconfig: %w", err)
		}
	}
	return backupPath, nil
}

func RestoreKubeconfig(kubeconfigPath, backupPath string) error {
	logging.Debugf("Attempting to restore kubeconfig state: target=%s, backup=%s", kubeconfigPath, backupPath)

	_, backupStatErr := os.Stat(backupPath)

	if backupStatErr == nil {
		if err := os.Rename(backupPath, kubeconfigPath); err != nil {
			return fmt.Errorf("failed to restore kubeconfig by renaming backup %s: %w", backupPath, err)
		}
		logging.Debugf("Successfully restored kubeconfig from backup %s", backupPath)
	} else {
		if os.IsNotExist(backupStatErr) {
			if removeErr := os.Remove(kubeconfigPath); removeErr != nil && !os.IsNotExist(removeErr) {
				return fmt.Errorf("backup %s missing and failed to remove temporary kubeconfig %s: %w", backupPath, kubeconfigPath, removeErr)
			}
		} else {
			return fmt.Errorf("failed to check backup kubeconfig file %s: %w", backupPath, backupStatErr)
		}
	}
	return nil
}

func WriteKubeconfig(path string, content string) error {
	logging.Debugf("Writing kubeconfig to %s", path)
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		return fmt.Errorf("failed to write kubeconfig: %w", err)
	}
	return nil
}
