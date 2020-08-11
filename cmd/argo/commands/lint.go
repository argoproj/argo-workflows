package commands

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/argoproj/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

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

			lint := func(file string) error {
				wfs, err := validate.ParseWfFromFile(file, strict)
				if err != nil {
					return err
				}
				if len(wfs) == 0 {
					return fmt.Errorf("there was nothing to validate")
				}
				for _, wf := range wfs {
					if wf.Namespace == "" {
						wf.Namespace = client.Namespace()
					}
					_, err := serviceClient.LintWorkflow(ctx, &workflowpkg.WorkflowLintRequest{Namespace: wf.Namespace, Workflow: &wf})
					if err != nil {
						return err
					}
				}
				fmt.Printf("%s is valid\n", file)
				return nil
			}

			invalid := false
			for _, file := range args {
				stat, err := os.Stat(file)
				errors.CheckError(err)
				if stat.IsDir() {
					_ = filepath.Walk(file, func(path string, info os.FileInfo, err error) error {
						// If there was an error with the walk, return
						if err != nil {
							return err
						}

						fileExt := filepath.Ext(info.Name())
						switch fileExt {
						case ".yaml", ".yml", ".json":
						default:
							return nil
						}

						if err := lint(path); err != nil {
							log.Errorf("Error in file %s: %s", path, err)
							invalid = true
						}
						return nil
					})
				} else {
					if err := lint(file); err != nil {
						log.Errorf("Error in file %s: %s", file, err)
						invalid = true
					}
				}
			}
			if invalid {
				log.Fatalf("Errors encountered in validation")
			}
			fmt.Printf("Workflow manifests validated\n")
		},
	}
	command.Flags().BoolVar(&strict, "strict", true, "perform strict workflow validation")
	return command
}
