package commands

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"

	apiServer "github.com/argoproj/argo/cmd/server/workflow"
	"github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	cmdutil "github.com/argoproj/argo/util/cmd"
	"github.com/argoproj/argo/workflow/validate"
	"github.com/argoproj/argo/cmd/argo/commands/client"
)

func NewLintCommand() *cobra.Command {
	var (
		strict     bool
		serverHost string
	)
	var command = &cobra.Command{
		Use:   "lint (DIRECTORY | FILE1 FILE2 FILE3...)",
		Short: "validate a file or directory of workflow manifests",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) == 0 {
				cmd.HelpFunc()(cmd, args)
				os.Exit(1)
			}

			var err error
			wftmplGetter := &LazyWorkflowTemplateGetter{}
			validateDir := cmdutil.MustIsDir(args[0])

			conn, err := GetServerConn(serverHost)
			if err != nil {
				panic(err)
			}
			if conn != nil {
				defer conn.Close()
				err = ServerSideLint(args[0], conn, strict)
				if err != nil {
					return
				}
			} else {
				if validateDir {
					if len(args) > 1 {
						fmt.Printf("Validation of a single directory supported")
						os.Exit(1)
					}
					fmt.Printf("Verifying all workflow manifests in directory: %s\n", args[0])
					err = validate.LintWorkflowDir(wftmplGetter, args[0], strict)
					if err != nil {
						return
					}
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
						err = validate.LintWorkflowFile(wftmplGetter, yamlFile, strict)
						if err != nil {
							break
						}
					}
				}
			}
			if err != nil {
				log.Fatal(err)
			}
			fmt.Printf("Workflow manifests validated\n")
		},
	}
	command.Flags().BoolVar(&strict, "strict", true, "perform strict workflow validatation")
	command.Flags().StringVar(&serverHost, "server", "", "API Server host and port")
	return command
}

func ServerSideLint(arg string, conn *grpc.ClientConn, strict bool) error {
	validateDir := cmdutil.MustIsDir(arg)
	grpcClient, ctx := GetApiServerGRPCClient(conn)
	ns, _, _ := client.Config.Namespace()
	var wfs [] v1alpha1.Workflow
	var err error

	if validateDir {
		walkFunc := func(path string, info os.FileInfo, err error) error {
			if info == nil || info.IsDir() {
				return nil
			}
			fileExt := filepath.Ext(info.Name())
			switch fileExt {
			case ".yaml", ".yml", ".json":
			default:
				return nil
			}
			wfs, err1 := validate.ParseWfFromFile(path, strict)
			if err1 != nil {
				return err1
			}
			for _, wf := range wfs {
				err = ServerLintValidation(ctx, grpcClient, wf, ns)
				if err != nil {
					log.Errorf("Validation Error in %s :%v", path, err)
				}
			}
			return err
		}
		return filepath.Walk(arg, walkFunc)
	} else {
		wfs, err = validate.ParseWfFromFile(arg, strict)
	}
	if err != nil {
		log.Error(err)
		return err
	}
	for _, wf := range wfs {
		err = ServerLintValidation(ctx, grpcClient, wf, ns)
		if err != nil {
			log.Error(err)
		}
	}
	return err
}

func ServerLintValidation(ctx context.Context, client apiServer.WorkflowServiceClient, wf v1alpha1.Workflow, ns string) error {
	wfReq := apiServer.WorkflowCreateRequest{
		Namespace: ns,
		Workflow:  &wf,
	}
	_, err := client.LintWorkflow(ctx, &wfReq)
	return err
}
