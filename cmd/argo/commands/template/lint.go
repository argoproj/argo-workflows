package template

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"

	"github.com/argoproj/argo/cmd/argo/commands/client"
	"github.com/argoproj/argo/cmd/server/workflowtemplate"
	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	cmdutil "github.com/argoproj/argo/util/cmd"
	"github.com/argoproj/argo/workflow/validate"
)

func NewLintCommand() *cobra.Command {
	var (
		strict bool
	)
	var command = &cobra.Command{
		Use:   "lint (DIRECTORY | FILE1 FILE2 FILE3...)",
		Short: "validate a file or directory of workflow template manifests",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) == 0 {
				cmd.HelpFunc()(cmd, args)
				os.Exit(1)
			}

			if client.ArgoServer != "" {
				conn := client.GetClientConn()
				defer conn.Close()
				err := ServerSideLint(args, conn, strict)
				if err != nil {
					log.Fatal(err)
				}
				fmt.Printf("WorkflowTemplate manifests validated\n")
			} else {
				wftmplGetter := &LazyWorkflowTemplateGetter{}
				validateDir := cmdutil.MustIsDir(args[0])
				if validateDir {
					if len(args) > 1 {
						fmt.Printf("Validation of a single directory supported")
						os.Exit(1)
					}
					fmt.Printf("Verifying all workflow template manifests in directory: %s\n", args[0])
					err := validate.LintWorkflowTemplateDir(wftmplGetter, args[0], strict)
					if err != nil {
						log.Fatal(err)
					}
					fmt.Printf("WorkflowTemplate manifests validated\n")
				} else {
					yamlFiles := make([]string, 0)
					for _, filePath := range args {
						if cmdutil.MustIsDir(filePath) {
							fmt.Printf("Validate against a list of files or a single directory, not both")
							os.Exit(1)
						}
						yamlFiles = append(yamlFiles, filePath)
					}
					for _, yamlFile := range yamlFiles {
						err := validate.LintWorkflowTemplateFile(wftmplGetter, yamlFile, strict)
						if err != nil {
							log.Fatal(err)
						}
					}
				}
				fmt.Printf("WorkflowTemplate manifests validated\n")
			}

		},
	}
	command.Flags().BoolVar(&strict, "strict", true, "perform strict workflow validation")
	return command
}

func ServerSideLint(args []string, conn *grpc.ClientConn, strict bool) error {
	validateDir := cmdutil.MustIsDir(args[0])
	grpcClient, ctx := GetWFtmplApiServerGRPCClient(conn)
	ns, _, _ := client.Config.Namespace()

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
			wfTmpls, err := validate.ParseWfTmplFromFile(path, strict)
			if err != nil {
				log.Error(err)
			}
			for _, wfTmpl := range wfTmpls {
				err := ServerLintValidation(ctx, grpcClient, wfTmpl, ns)
				if err != nil {
					log.Error(err)
				}
			}
			return nil
		}
		return filepath.Walk(args[0], walkFunc)
	} else {
		for _, arg := range args {
			wfTmpls, err := validate.ParseWfTmplFromFile(arg, strict)
			if err != nil {
				log.Error(err)
			}
			for _, wfTmpl := range wfTmpls {
				err := ServerLintValidation(ctx, grpcClient, wfTmpl, ns)
				if err != nil {
					log.Error(err)
				}
			}
		}
	}
	return nil
}

func ServerLintValidation(ctx context.Context, client workflowtemplate.WorkflowTemplateServiceClient, wfTmpl wfv1.WorkflowTemplate, ns string) error {
	wfTmplReq := workflowtemplate.WorkflowTemplateLintRequest{
		Namespace: ns,
		Template:  &wfTmpl,
	}
	_, err := client.LintWorkflowTemplate(ctx, &wfTmplReq)
	return err
}
