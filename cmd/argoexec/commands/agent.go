package commands

import (
	"context"

	"github.com/spf13/cobra"
)

func NewAgentCommand() *cobra.Command {
	command := cobra.Command{
		Use:          "agent",
		SilenceUsage: true, // this prevents confusing usage message being printed on error
		RunE: func(cmd *cobra.Command, args []string) error {
			return initExecutor().Agent(context.Background())
		},
	}
	return &command
}
