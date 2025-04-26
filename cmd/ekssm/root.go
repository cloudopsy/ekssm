package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	rootCmd = &cobra.Command{
		Use:   "ekssm",
		Short: "EKS SSM Proxy allows running commands against an EKS cluster via an SSM-enabled instance.",
		Long: `EKS SSM Proxy allows running commands against an EKS cluster via an SSM-enabled instance.
Primarily used for kubectl, but can support any command that can use the KUBECONFIG environment.

Use 'ekssm run --help' or 'ekssm session --help' for more details on subcommands.`,
	}

	debug bool
)

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().BoolVar(&debug, "debug", false, "Enable debug logging")
}
