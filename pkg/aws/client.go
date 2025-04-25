// Package aws provides client utilities for interacting with AWS services
// such as EKS and SSM.
package aws

import (
	"context"
	"fmt"
	
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/eks"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	
	"github.com/cloudopsy/ekssm/internal/logging"
)

// Client holds AWS service clients and region information.
type Client struct {
   EKS    *eks.Client
   SSM    *ssm.Client
   Region string
}

// NewClient creates a new AWS client with EKS and SSM service clients.
func NewClient(ctx context.Context) (*Client, error) {
	logging.Debug("Initializing AWS client")
	
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, err
	}
	
   return &Client{
       EKS:    eks.NewFromConfig(cfg),
       SSM:    ssm.NewFromConfig(cfg),
       Region: cfg.Region,
   }, nil
}

// DescribeEKSCluster retrieves information about the specified EKS cluster.
func (c *Client) DescribeEKSCluster(ctx context.Context, clusterName string) (*eks.DescribeClusterOutput, error) {
	if clusterName == "" {
		return nil, fmt.Errorf("cluster name is required")
	}
	
	logging.Debugf("Fetching information for EKS cluster %s", clusterName)
	
	output, err := c.EKS.DescribeCluster(ctx, &eks.DescribeClusterInput{
		Name: aws.String(clusterName),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to describe EKS cluster %s: %w", clusterName, err)
	}
	
	if output.Cluster == nil {
		return nil, fmt.Errorf("received nil cluster data from EKS API")
	}
	
	if output.Cluster.Endpoint == nil || *output.Cluster.Endpoint == "" {
		return nil, fmt.Errorf("received empty API server endpoint from EKS API")
	}
	
	if output.Cluster.CertificateAuthority == nil || output.Cluster.CertificateAuthority.Data == nil {
		return nil, fmt.Errorf("received empty certificate authority data from EKS API")
	}
	
	return output, nil
}
