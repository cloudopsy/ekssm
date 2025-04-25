package util_test

import (
	"os"
	"path/filepath"
	"testing"
	
	"github.com/cloudopsy/ekssm/internal/util"
)

func TestWriteAndRestoreKubeconfig(t *testing.T) {
	// Create a temp dir for tests
	tmpDir, err := os.MkdirTemp("", "ekssm-test-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)
	
	// Set up test paths
	testKubeconfigPath := filepath.Join(tmpDir, "config")
	testBackupPath := testKubeconfigPath + ".bak"
	
	// Test content
	originalContent := "original kubeconfig"
	newContent := "new kubeconfig"
	
	// Write original content
	if err := os.WriteFile(testKubeconfigPath, []byte(originalContent), 0600); err != nil {
		t.Fatal(err)
	}
	
	// Test backup
	backupPath, err := util.BackupKubeconfig(testKubeconfigPath)
	if err != nil {
		t.Fatalf("BackupKubeconfig failed: %v", err)
	}
	
	if backupPath != testBackupPath {
		t.Errorf("Expected backup path %s, got %s", testBackupPath, backupPath)
	}
	
	// Check if original is gone and moved to backup
	if _, err := os.Stat(testKubeconfigPath); !os.IsNotExist(err) {
		t.Errorf("Original file should not exist after backup")
	}
	
	backupContent, err := os.ReadFile(testBackupPath)
	if err != nil {
		t.Fatal(err)
	}
	
	if string(backupContent) != originalContent {
		t.Errorf("Backup content doesn't match original")
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
	
	// Test restore
	if err := util.RestoreKubeconfig(testKubeconfigPath, testBackupPath); err != nil {
		t.Fatalf("RestoreKubeconfig failed: %v", err)
	}
	
	restoredContent, err := os.ReadFile(testKubeconfigPath)
	if err != nil {
		t.Fatal(err)
	}
	
	if string(restoredContent) != originalContent {
		t.Errorf("Restored content doesn't match original")
	}
	
	// Check if backup is gone
	if _, err := os.Stat(testBackupPath); !os.IsNotExist(err) {
		t.Errorf("Backup file should not exist after restore")
	}
}
