package config_test

import (
	"testing"

	"github.com/cloudopsy/ekssm/pkg/config"
	"github.com/stretchr/testify/assert"
)

func TestOptionsValidate(t *testing.T) {
	tests := []struct {
		name        string
		options     config.Options
		expectError bool
	}{
		{
			name: "Valid options with kubectl",
			options: config.Options{
				InstanceID:  "i-12345",
				ClusterName: "test-cluster",
				LocalPort:   "9443",
				CommandArgs: []string{"kubectl", "get", "pods"}, // Include command
			},
			expectError: false,
		},
		{
			name: "Valid options with helm",
			options: config.Options{
				InstanceID:  "i-12345",
				ClusterName: "test-cluster",
				LocalPort:   "9443",
				CommandArgs: []string{"helm", "list", "-n", "monitoring"}, // Include command
			},
			expectError: false,
		},
		{
			name: "Missing instance ID",
			options: config.Options{
				ClusterName: "test-cluster",
				LocalPort:   "9443",
				CommandArgs: []string{"kubectl", "get", "pods"}, // Include command
			},
			expectError: true,
		},
		{
			name: "Missing cluster name",
			options: config.Options{
				InstanceID:  "i-12345",
				LocalPort:   "9443",
				CommandArgs: []string{"kubectl", "get", "pods"}, // Include command
			},
			expectError: true,
		},
		{
			name: "Missing command args",
			options: config.Options{
				InstanceID:  "i-12345",
				ClusterName: "test-cluster",
				LocalPort:   "9443",
				CommandArgs: []string{}, // Check empty args
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.options.Validate()
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
