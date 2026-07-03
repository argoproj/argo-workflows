package commands

import "github.com/spf13/cobra"

// NewResourceAgentCommand is
func NewResourceAgentCommand() *cobra.Command {
	cmd := cobra.Command{
		Use:          "resource-agent",
		SilenceUsage: true,
	}
	cmd.AddCommand(NewResourceAgentCommand())
	return &cmd
}

// NewResourceAgentMainCommand is
func NewResourceAgentMainCommand() *cobra.Command {
	return &cobra.Command{
		Use: "main",
	}
}
