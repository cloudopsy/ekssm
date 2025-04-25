// Package util provides internal utility functions for file operations, networking,
// and kubeconfig management specific to ekssm.
package util

import (
	"context"
	"fmt"
	"strings"

	"github.com/cloudopsy/ekssm/internal/logging"
	awsclient "github.com/cloudopsy/ekssm/pkg/aws"
)

// EKSClusterEndpoint fetches and validates an EKS cluster endpoint.
// Returns the endpoint hostname (without https://) and an error if the operation fails.
func EKSClusterEndpoint(ctx context.Context, clusterName string) (string, error) {
	logging.Debugf("Fetching endpoint for EKS cluster: %s", clusterName)
	
	awsClient, err := awsclient.NewClient(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to initialize AWS client: %w", err)
	}

	clusterOutput, err := awsClient.DescribeEKSCluster(ctx, clusterName)
	if err != nil {
		return "", fmt.Errorf("failed to describe EKS cluster: %w", err)
	}
	
	if clusterOutput.Cluster == nil || 
	   clusterOutput.Cluster.Endpoint == nil || 
	   *clusterOutput.Cluster.Endpoint == "" {
		return "", fmt.Errorf("invalid cluster information returned from EKS API")
	}

	eksEndpoint := *clusterOutput.Cluster.Endpoint
	logging.Debugf("EKS API server endpoint: %s", eksEndpoint)

	// Extract host from https://... endpoint
	eksHost := strings.TrimPrefix(eksEndpoint, "https://")
	return eksHost, nil
}