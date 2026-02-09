package clustertemplate

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/argoproj/argo-workflows/v3/cmd/argo/commands/client"
	"github.com/argoproj/argo-workflows/v3/cmd/argo/commands/common"
	"github.com/argoproj/argo-workflows/v3/cmd/argo/lint"
	wf "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow"
)

func NewLintCommand() *cobra.Command {
	var (
		strict bool
		output = common.EnumFlagValue{
			AllowedValues: []string{"pretty", "simple"},
			Value:         "pretty",
		}
	)

	command := &cobra.Command{
		Use:   "lint FILE...",
		Short: "validate files or directories of cluster workflow template manifests",
		Example: `# Lint a single cluster workflow template:
  argo cluster-template lint my-cluster-template.yaml

# Lint multiple files:
  argo cluster-template lint template1.yaml template2.yaml

# Lint all templates in a directory:
  argo cluster-template lint ./cluster-templates/

# Lint with simple output format:
  argo cluster-template lint -o simple my-cluster-template.yaml

# Lint without strict validation:
  argo cluster-template lint --strict=false my-cluster-template.yaml
`,

		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, apiClient, err := client.NewAPIClient(cmd.Context())
			if err != nil {
				return err
			}
			opts := lint.Options{
				Files:            args,
				DefaultNamespace: client.Namespace(ctx),
				Strict:           strict,
				Printer:          os.Stdout,
			}

			return lint.RunLint(ctx, apiClient, []string{wf.ClusterWorkflowTemplatePlural}, output.String(), false, opts)
		},
	}

	command.Flags().VarP(&output, "output", "o", "Linting results output format. "+output.Usage())
	command.Flags().BoolVar(&strict, "strict", true, "perform strict workflow validation")
	return command
}
