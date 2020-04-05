package clustertemplate

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/argoproj/pkg/errors"

	"github.com/argoproj/argo/cmd/argo/commands/client"
	"github.com/argoproj/argo/pkg/apiclient/clusterworkflowtemplate"
	"github.com/argoproj/argo/workflow/validate"
)

func NewLintCommand() *cobra.Command {
	var (
		strict bool
	)
	var command = &cobra.Command{
		Use:   "lint FILE...",
		Short: "validate files or directories of cluster workflow template manifests",
		Run: func(cmd *cobra.Command, args []string) {
			ctx, apiClient := client.NewAPIClient()
			serviceClient := apiClient.NewClusterWorkflowTemplateServiceClient()

			lint := func(file string) error {
				cwfTmpls, err := validate.ParseCWfTmplFromFile(file, strict)
				if err != nil {
					return err
				}
				for _, cfwft := range cwfTmpls {
					_, err := serviceClient.LintClusterWorkflowTemplate(ctx, &clusterworkflowtemplate.ClusterWorkflowTemplateLintRequest{Template: &cfwft})
					if err != nil {
						return err
					}
				}
				fmt.Printf("%s is valid\n", file)
				return nil
			}

			for _, file := range args {
				stat, err := os.Stat(file)
				errors.CheckError(err)
				if stat.IsDir() {
					err := filepath.Walk(file, func(path string, info os.FileInfo, err error) error {
						fileExt := filepath.Ext(info.Name())
						switch fileExt {
						case ".yaml", ".yml", ".json":
						default:
							return nil
						}
						return lint(path)
					})
					errors.CheckError(err)
				} else {
					err := lint(file)
					errors.CheckError(err)
				}
			}
			fmt.Printf("Cluster Workflow Template manifests validated\n")
		},
	}
	command.Flags().BoolVar(&strict, "strict", true, "perform strict workflow validation")
	return command
}
