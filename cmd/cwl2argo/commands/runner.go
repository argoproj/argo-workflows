package commands

import (
	"errors"

	"github.com/spf13/cobra"
)

// Cobra command to transpile
func NewRunnerCommand() *cobra.Command {

	command := cobra.Command{
		Use:   "runner",
		Short: "run cwl file on argo",
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) != 2 {
				return errors.New("runner only accepts two arguments <WORKFLOW.cwl> and <INPUTS.(yml|json|cwl)>")
			}
			return nil
		},
		Run: func(cmd *cobra.Command, args []string) {
			panic("Runner has not been implemented yet")
		},
	}

	return &command
}
