package commands

import (
	"fmt"
	"os"
	"path/filepath"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/argoproj/pkg/errors"

	"github.com/argoproj/argo/cmd/argo/commands/client"
	workflowpkg "github.com/argoproj/argo/pkg/apiclient/workflow"
	"github.com/argoproj/argo/workflow/validate"
)

func NewLintCommand() *cobra.Command {
	var (
		strict bool
	)
	var command = &cobra.Command{
		Use:   "lint FILE...",
		Short: "validate files or directories of workflow manifests",
		Run: func(cmd *cobra.Command, args []string) {
			ctx, apiClient := client.NewAPIClient()
			serviceClient := apiClient.NewWorkflowServiceClient()
			namespace := client.Namespace()

			lint := func(file string) error {
				wfs, err := validate.ParseWfFromFile(file, strict)
				if err != nil {
					return err
				}
				for _, wf := range wfs {
					_, err := serviceClient.LintWorkflow(ctx, &workflowpkg.WorkflowLintRequest{Namespace: namespace, Workflow: &wf})
					if err != nil {
						return err
					}
				}
				fmt.Printf("%s is valid\n", file)
				return nil
			}

			var failed bool
			for _, file := range args {
				stat, err := os.Stat(file)
				errors.CheckError(err)
				if stat.IsDir() {
					err := filepath.Walk(file, func(path string, info os.FileInfo, err error) error {
						if err != nil {
							return err
						}
						fileExt := filepath.Ext(info.Name())
						switch fileExt {
						case ".yaml", ".yml", ".json":
						default:
							return nil
						}
						err = lint(path)
						if err != nil {
							log.Warnf("%s is invalid: %v", path, err)
							failed = true
						}
						return nil
					})
					errors.CheckError(err)
				} else {
					err := lint(file)
					if err != nil {
						log.Warnf("%s is invalid: %v", file, err)
						failed = true
					}
				}
			}
			if failed {
				fmt.Printf("One or more manifests invalid\n")
			}
			fmt.Printf("Workflow manifests validated\n")
		},
	}
	command.Flags().BoolVar(&strict, "strict", true, "perform strict workflow validation")
	return command
}
