package clustertemplate

import (
	"context"
	"fmt"

	"os"
	"path/filepath"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/argoproj/argo/cmd/argo/commands/client"
	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	cmdutil "github.com/argoproj/argo/util/cmd"
	"github.com/argoproj/argo/workflow/validate"
	"github.com/argoproj/argo/pkg/apiclient/clusterworkflowtemplate"
)

func NewLintCommand() *cobra.Command {
	var (
		strict bool
	)
	var command = &cobra.Command{
		Use:   "lint (DIRECTORY | FILE1 FILE2 FILE3...)",
		Short: "validate a file or directory of cluster workflow template manifests",
		Run: func(cmd *cobra.Command, args []string) {
			err := ServerSideLint(args, strict)
			if err != nil {
				log.Fatal(err)
			}
			fmt.Printf("ClusterWorkflowTemplate manifests validated\n")
		},
	}
	command.Flags().BoolVar(&strict, "strict", true, "perform strict workflow validation")
	return command
}

func ServerSideLint(args []string, strict bool) error {
	validateDir := cmdutil.MustIsDir(args[0])

	ctx, apiClient := client.NewAPIClient()
	serviceClient := apiClient.NewClusterWorkflowTemplateServiceClient()

	if validateDir {
		if len(args) > 1 {
			fmt.Printf("Validation of a single directory supported")
			os.Exit(1)
		}
		walkFunc := func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if info == nil || info.IsDir() {
				return nil
			}
			fileExt := filepath.Ext(info.Name())
			switch fileExt {
			case ".yaml", ".yml", ".json":
			default:
				return nil
			}
			cwfTmpls, err := validate.ParseCWfTmplFromFile(path, strict)
			if err != nil {
				log.Error(err)
			}
			for _, cwfTmpl := range cwfTmpls {
				err := ServerLintValidation(ctx, serviceClient, cwfTmpl)
				if err != nil {
					log.Error(err)
				}
			}
			return nil
		}
		return filepath.Walk(args[0], walkFunc)
	} else {
		for _, arg := range args {
			cwfTmpls, err := validate.ParseCWfTmplFromFile(arg, strict)
			if err != nil {
				log.Error(err)
			}
			for _, cwfTmpl := range cwfTmpls {
				err := ServerLintValidation(ctx, serviceClient, cwfTmpl)
				if err != nil {
					log.Error(err)
				}
			}
		}
	}
	return nil
}

func ServerLintValidation(ctx context.Context, client clusterworkflowtemplate.ClusterWorkflowTemplateServiceClient, cwfTmpl wfv1.ClusterWorkflowTemplate) error {
	cwfTmplReq := clusterworkflowtemplate.ClusterWorkflowTemplateLintRequest{
		Template:  &cwfTmpl,
	}
	_, err := client.LintClusterWorkflowTemplate(ctx, &cwfTmplReq)
	return err
}
