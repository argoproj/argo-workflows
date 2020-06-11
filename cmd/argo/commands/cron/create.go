package cron

import (
	"fmt"
	"log"
	"os"

	"github.com/argoproj/pkg/json"
	"github.com/spf13/cobra"

	"github.com/argoproj/argo/cmd/argo/commands/client"
	cronworkflowpkg "github.com/argoproj/argo/pkg/apiclient/cronworkflow"

	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo/workflow/common"
	"github.com/argoproj/argo/workflow/util"
)

type cliCreateOpts struct {
	output string // --output
	strict bool   // --strict
}

func NewCreateCommand() *cobra.Command {
	var (
		cliCreateOpts cliCreateOpts
	)
	var command = &cobra.Command{
		Use:   "create FILE1 FILE2...",
		Short: "create a cron workflow",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) == 0 {
				cmd.HelpFunc()(cmd, args)
				os.Exit(1)
			}

			CreateCronWorkflows(args, &cliCreateOpts)
		},
	}
	command.Flags().StringVarP(&cliCreateOpts.output, "output", "o", "", "Output format. One of: name|json|yaml|wide")
	command.Flags().BoolVar(&cliCreateOpts.strict, "strict", true, "perform strict workflow validation")
	return command
}

func CreateCronWorkflows(filePaths []string, cliOpts *cliCreateOpts) {

	ctx, apiClient := client.NewAPIClient()
	serviceClient := apiClient.NewCronWorkflowServiceClient()
	namespace := client.Namespace()

	fileContents, err := util.ReadManifest(filePaths...)
	if err != nil {
		log.Fatal(err)
	}

	var cronWorkflows []wfv1.CronWorkflow
	for _, body := range fileContents {
		cronWfs := unmarshalCronWorkflows(body, cliOpts.strict)
		cronWorkflows = append(cronWorkflows, cronWfs...)
	}

	if len(cronWorkflows) == 0 {
		log.Println("No CronWorkflows found in given files")
		os.Exit(1)
	}

	for _, cronWf := range cronWorkflows {
		created, err := serviceClient.CreateCronWorkflow(ctx, &cronworkflowpkg.CreateCronWorkflowRequest{
			Namespace:    namespace,
			CronWorkflow: &cronWf,
		})
		if err != nil {
			log.Fatalf("Failed to create workflow template: %v", err)
		}
		fmt.Print(getCronWorkflowGet(created))
	}
}

// unmarshalCronWorkflows unmarshals the input bytes as either json or yaml
func unmarshalCronWorkflows(wfBytes []byte, strict bool) []wfv1.CronWorkflow {
	var cronWf wfv1.CronWorkflow
	var jsonOpts []json.JSONOpt
	if strict {
		jsonOpts = append(jsonOpts, json.DisallowUnknownFields)
	}
	err := json.Unmarshal(wfBytes, &cronWf, jsonOpts...)
	if err == nil {
		return []wfv1.CronWorkflow{cronWf}
	}
	yamlWfs, err := common.SplitCronWorkflowYAMLFile(wfBytes, strict)
	if err == nil {
		return yamlWfs
	}
	log.Fatalf("Failed to parse workflow template: %v", err)
	return nil
}
