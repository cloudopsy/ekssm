package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

var shellCmd = &cobra.Command{
	Use:   "shell [shell]",
	Short: "Generate shell integration code",
	Long: `Generates shell integration code that allows ekssm to modify environment variables
in the parent shell.

Supported shells: bash, zsh

Usage:
  # For bash/zsh (add to ~/.bashrc or ~/.zshrc):
  eval "$(ekssm shell bash)"

This enables automatic setting of KUBECONFIG when using 'ekssm session switch'.`,
	Args: cobra.ExactValidArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		shellType := args[0]

		switch shellType {
		case "bash", "zsh":
			fmt.Print(`
# ekssm shell integration
ekssm() {
  # Parse first argument
  local cmd="$1"
  
  # Handle session commands specially
  if [ "$cmd" = "session" ]; then
    local subcmd="$2"
    
    # Handle switch command
    if [ "$subcmd" = "switch" ] && [ -n "$3" ]; then
      local kubeconfig_cmd=$(command ekssm session switch "$3")
      local exit_code=$?
      
      if [ $exit_code -eq 0 ]; then
        eval "$kubeconfig_cmd"
        echo "KUBECONFIG environment variable set for session $3"
      else
        echo "$kubeconfig_cmd"
      fi
      return $exit_code
      
    # Handle start command
    elif [ "$subcmd" = "start" ]; then
      # Remove the first two arguments to pass the rest to the command
      shift 2
      command ekssm session start "$@"
      local exit_code=$?
      
      if [ $exit_code -eq 0 ]; then
        local session_id=$(command ekssm session list | grep "Latest session created:" | awk '{print $NF}')
        if [ -n "$session_id" ]; then
          local kubeconfig_cmd=$(command ekssm session switch "$session_id")
          eval "$kubeconfig_cmd"
          echo "KUBECONFIG environment variable automatically set for new session $session_id"
        fi
      fi
      return $exit_code
    
    # Handle list command
    elif [ "$subcmd" = "list" ]; then
      shift
      command ekssm session "$@"
      return $?
    
    # Handle stop command
    elif [ "$subcmd" = "stop" ]; then
      shift
      command ekssm session stop "$@"
      local exit_code=$?
      if [ $exit_code -eq 0 ]; then
        # Unset KUBECONFIG when stopping sessions
        unset KUBECONFIG
        echo "KUBECONFIG environment variable unset"
      fi
      return $exit_code
    
    # Handle any other session subcommands for future compatibility
    else
      shift
      command ekssm session "$@"
      return $?
    fi
  
  # For all other commands, pass through directly
  else
    command ekssm "$@"
    return $?
  fi
}
`)
		default:
			return fmt.Errorf("unsupported shell type: %s (supported: bash, zsh)", shellType)
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(shellCmd)
}
