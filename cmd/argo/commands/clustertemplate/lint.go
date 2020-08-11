package clustertemplate

import (
	"github.com/spf13/cobra"

	"github.com/argoproj/argo/cmd/argo/commands/client"
	"github.com/argoproj/argo/workflow/lint"
)

func NewLintCommand() *cobra.Command {
	var (
		strict bool
	)
	var command = &cobra.Command{
		Use:   "lint FILE...",
		Short: "Lint files or directories of manifests",
		Example: `
# Lint one or more files:

argo lint file.yaml file.json

# Lint a directory:

argo lint examples/

# Lint one or more files:

argo lint file.yaml file.json

# Lint from stdin:

argo lint /dev/stdin < file.yaml
`,
		Run: func(cmd *cobra.Command, args []string) {
			ctx, apiClient := client.NewAPIClient()
			lint.Lint(ctx, apiClient, "", args, strict)
		},
	}
	command.Flags().BoolVar(&strict, "strict", true, "perform strict validation")
	return command
}
