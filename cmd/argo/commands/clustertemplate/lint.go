package clustertemplate

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	cmdcommon "github.com/argoproj/argo/cmd/argo/commands/common"
	"github.com/argoproj/argo/pkg/apiclient/clusterworkflowtemplate"
	"github.com/argoproj/argo/workflow/validate"
)

func NewLintCommand() *cobra.Command {
	var (
		strict bool
	)
	var command = &cobra.Command{
		Use:          "lint FILE...",
		Short:        "validate files or directories of cluster workflow template manifests",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, apiClient := cmdcommon.CreateNewAPIClientFunc()
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
				if err != nil {
					return err
				}
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
					if err != nil {
						return err
					}
				} else {
					err := lint(file)
					if err != nil {
						return err
					}
				}
			}
			fmt.Printf("Cluster Workflow Template manifests validated\n")
			return nil
		},
	}
	command.Flags().BoolVar(&strict, "strict", true, "perform strict workflow validation")
	return command
}
