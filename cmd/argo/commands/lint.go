package commands

import (
	"fmt"
	"os"
	"strings"

	"github.com/argoproj/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/argoproj/argo-workflows/v3/cmd/argo/commands/client"
	"github.com/argoproj/argo-workflows/v3/cmd/argo/lint"
	"github.com/argoproj/argo-workflows/v3/pkg/apiclient"
)

func NewLintCommand() *cobra.Command {
	var (
		strict    bool
		lintKinds []string
		output    string
	)

	allKinds := []string{"workflow", "workflow-template", "cron-workflow", "cluster-workflow-template"}

	command := &cobra.Command{
		Use:   "lint FILE...",
		Short: "validate files or directories of manifests",
		Example: `# Lint all manifests in a specified directory:

  argo lint ./manifests

# Lint only manifests of kinds Workflow and CronWorkflow from stdin:

  cat manifests.yaml | argo lint --kinds=wf,cwf -
`,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) == 0 {
				cmd.HelpFunc()(cmd, args)
				os.Exit(1)
			}

			if len(lintKinds) == 0 || strings.Contains(strings.Join(lintKinds, ","), "all") {
				lintKinds = allKinds
			}
			RunLint(args, lintKinds, output, strict)
		},
	}

	command.Flags().StringSliceVar(&lintKinds, "kinds", []string{"all"}, fmt.Sprintf("Which kinds will be linted. Can be: %s", strings.Join(allKinds, "|")))
	command.Flags().StringVarP(&output, "output", "o", "pretty", "Linting results output format. One of: pretty|simple")
	command.Flags().BoolVar(&strict, "strict", true, "Perform strict workflow validation")

	return command
}

func getLintClients(client apiclient.Client, kinds []string) (lint.ServiceClients, error) {
	res := lint.ServiceClients{}
	for _, kind := range kinds {
		switch kind {
		case "workflow", "wf":
			res.WorkflowsClient = client.NewWorkflowServiceClient()
		case "workflow-template", "wft":
			res.WorkflowTemplatesClient = client.NewWorkflowTemplateServiceClient()
		case "cron-workflow", "cwf":
			res.CronWorkflowsClient = client.NewCronWorkflowServiceClient()
		case "cluster-workflow-template", "cwft":
			res.ClusterWorkflowTemplateClient = client.NewClusterWorkflowTemplateServiceClient()
		default:
			return res, fmt.Errorf("unknown kind: %s", kind)
		}
	}

	return res, nil
}

func RunLint(files, kinds []string, output string, strict bool) {
	ctx, apiClient := client.NewAPIClient()

	fmtr, err := lint.GetFormatter(output)
	errors.CheckError(err)

	clients, err := getLintClients(apiClient, kinds)
	errors.CheckError(err)

	res, err := lint.Lint(ctx, &lint.LintOptions{
		ServiceClients:   clients,
		Files:            files,
		Strict:           strict,
		DefaultNamespace: client.Namespace(),
		Formatter:        fmtr,
		Output:           os.Stdout,
	})
	errors.CheckError(err)

	if !res.Success {
		os.Exit(1)
	}
}
