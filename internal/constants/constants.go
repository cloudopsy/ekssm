// Package constants provides static, well-known values used throughout the ekssm application.
package constants

// Backup file suffixes
const SessionBackupSuffix = ".ekssm-bak" 
const RunBackupSuffix = ".ekssm-run-bak"

// Network port constants 
const DefaultLocalPort = "9443" // Default local port for the SSM proxy
const EKSApiPort = "443"        // Standard HTTPS port used by EKS API server
