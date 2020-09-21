package cron

import (
	"fmt"
	"log"

	"github.com/argoproj/pkg/json"
	"github.com/spf13/cobra"

	"github.com/argoproj/argo/cmd/argo/commands/client"
	cronworkflowpkg "github.com/argoproj/argo/pkg/apiclient/cronworkflow"

	cmdcommon "github.com/argoproj/argo/cmd/argo/commands/common"
	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo/workflow/common"
	"github.com/argoproj/argo/workflow/util"
)

type cliCreateOpts struct {
	output   string // --output
	schedule string // --schedule
	strict   bool   // --strict
}

func NewCreateCommand() *cobra.Command {
	var (
		cliCreateOpts cliCreateOpts
		submitOpts    wfv1.SubmitOpts
	)
	var command = &cobra.Command{
		Use:   "create FILE1 FILE2...",
		Short: "create a cron workflow",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				cmd.HelpFunc()(cmd, args)
				return cmdcommon.MissingArgumentsError
			}

			return CreateCronWorkflows(args, &cliCreateOpts, &submitOpts)
		},
	}

	util.PopulateSubmitOpts(command, &submitOpts, false)
	command.Flags().StringVarP(&cliCreateOpts.output, "output", "o", "", "Output format. One of: name|json|yaml|wide")
	command.Flags().BoolVar(&cliCreateOpts.strict, "strict", true, "perform strict workflow validation")
	command.Flags().StringVar(&cliCreateOpts.schedule, "schedule", "", "override cron workflow schedule")
	return command
}

func CreateCronWorkflows(filePaths []string, cliOpts *cliCreateOpts, submitOpts *wfv1.SubmitOpts) error {

	ctx, apiClient := cmdcommon.CreateNewAPIClientFunc()
	serviceClient := apiClient.NewCronWorkflowServiceClient()

	fileContents, err := util.ReadManifest(filePaths...)
	if err != nil {
		return err
	}

	var cronWorkflows []wfv1.CronWorkflow
	for _, body := range fileContents {
		cronWfs := unmarshalCronWorkflows(body, cliOpts.strict)
		cronWorkflows = append(cronWorkflows, cronWfs...)
	}

	if len(cronWorkflows) == 0 {
		log.Println("No CronWorkflows found in given files")
		return nil
	}

	for _, cronWf := range cronWorkflows {

		if cliOpts.schedule != "" {
			cronWf.Spec.Schedule = cliOpts.schedule
		}

		newWf := wfv1.Workflow{Spec: cronWf.Spec.WorkflowSpec}
		err := util.ApplySubmitOpts(&newWf, submitOpts)
		if err != nil {
			return err
		}
		cronWf.Spec.WorkflowSpec = newWf.Spec
		if cronWf.Namespace == "" {
			cronWf.Namespace = client.Namespace()
		}
		created, err := serviceClient.CreateCronWorkflow(ctx, &cronworkflowpkg.CreateCronWorkflowRequest{
			Namespace:    cronWf.Namespace,
			CronWorkflow: &cronWf,
		})
		if err != nil {
			return fmt.Errorf("Failed to create workflow template: %v", err)
		}
		fmt.Print(getCronWorkflowGet(created))
	}
	return nil
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
