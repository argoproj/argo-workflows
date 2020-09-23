package cron

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"time"

	"github.com/ghodss/yaml"
	cron "github.com/robfig/cron/v3"
	"github.com/spf13/cobra"

	"github.com/argoproj/argo/cmd/argo/commands/client"
	"github.com/argoproj/argo/pkg/apiclient/cronworkflow"
	"github.com/argoproj/argo/pkg/apiclient/workflow"
	"github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo/workflow/common"
	"github.com/argoproj/pkg/rand"
)

type backfillOpts struct {
	name       string
	startDate  string
	endDate    string
	parallel   bool
	argName    string
	dateFormat string
}

func NewBackfillCommand() *cobra.Command {
	var (
		cliOps backfillOpts
	)
	var command = &cobra.Command{
		Use:   "backfill cronwf",
		Short: "create a cron backfill",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				cmd.HelpFunc()(cmd, args)
				os.Exit(0)
			}
			if cliOps.name == "" {
				cliOps.name = rand.RandString(5)
			}
			return backfillCronWorkflow(args[0], cliOps)
		},
	}
	command.Flags().StringVar(&cliOps.name, "name", "", "Backfill name")
	command.Flags().StringVar(&cliOps.startDate, "start", "", "Start date")
	command.Flags().StringVar(&cliOps.endDate, "end", "", "End Date")
	command.Flags().BoolVar(&cliOps.parallel, "parallel", false, "")
	command.Flags().StringVar(&cliOps.argName, "argname", "cronscheduletime", "Schedule time argument name")
	command.Flags().StringVar(&cliOps.dateFormat, "format", time.RFC822, "Date format for workflow argument value")

	return command
}

func backfillCronWorkflow(cronWFName string, cliOps backfillOpts) error {
	if cliOps.startDate == "" {
		return fmt.Errorf("Start Date should be empty")
	}
	startTime, err := time.Parse(cliOps.dateFormat, cliOps.startDate)
	if err != nil {
		return err
	}
	var endTime time.Time
	if cliOps.endDate != "" {
		endTime, err = time.Parse(cliOps.dateFormat, cliOps.endDate)
		if err != nil {
			return err
		}
	} else {
		endTime = time.Now()
	}

	ctx, apiClient := client.NewAPIClient()
	cronClient := apiClient.NewCronWorkflowServiceClient()
	wfClient := apiClient.NewWorkflowServiceClient()
	req := cronworkflow.GetCronWorkflowRequest{
		Name:      cronWFName,
		Namespace: client.Namespace(),
	}
	cronWF, err := cronClient.GetCronWorkflow(ctx, &req)
	if err != nil {
		return err
	}
	cronTab, err := cron.ParseStandard(cronWF.Spec.Schedule)
	if err != nil {
		return err
	}
	scheTime := startTime
	priority := int32(math.MaxInt32)
	var scheList []string
	wf := common.ConvertCronWorkflowToWorkflow(cronWF)
	paramArg := "{{input.parameters.scheduletime}}"
	wf.GenerateName = cronWF.Name + "-backfill-" + cliOps.name + "-"
	param := v1alpha1.Parameter{
		Name:  cliOps.argName,
		Value: &paramArg,
	}
	if !cliOps.parallel {
		wf.Spec.Priority = &priority
		wf.Spec.Synchronization = &v1alpha1.Synchronization{
			Mutex: &v1alpha1.Mutex{Name: cliOps.name},
		}
	}
	wf.Spec.Arguments.Parameters = append(wf.Spec.Arguments.Parameters, param)
	for {
		scheTime = cronTab.Next(scheTime)
		timeStr := scheTime.String()
		scheList = append(scheList, timeStr)
		if endTime.Before(scheTime) {
			break
		}
	}
	wfJsonByte, err := json.Marshal(wf)
	yamlbyte, err := yaml.JSONToYAML(wfJsonByte)
	if err != nil {
		return err
	}
	wfYamlStr := "apiVersion: argoproj.io/v1alpha1 \n" + string(yamlbyte)
	return CreateMonitorWf(wfYamlStr, client.Namespace(), scheList, wfClient, ctx)
}

var backfillWf = `{
   "apiVersion": "argoproj.io/v1alpha1",
   "kind": "Workflow",
   "metadata": {
      "generateName": "backfill-wf-"
   },
   "spec": {
      "entrypoint": "main",
      "templates": [
         {
            "name": "main",
            "steps": [
               [
                  {
                     "name": "create-workflow",
                     "template": "create-workflow",
                     "arguments": {
                        "parameters": [
                           {
                              "name": "cronscheduletime",
                              "value": "{{item}}"
                           }
                        ],
                        "withParam": "{{workflows.parameters.cronscheduletime}}"
                     }
                  }
               ]
            ]
         },
         {
            "name": "create-workflow",
            "inputs": {
               "parameters": [
                  {
                     "name": "cronscheduletime"
                  }
               ]
            },
            "resource": {
               "successCondition": "status.phase == successed",
               "action": "create"
            }
         }
      ]
   }
}
`

func CreateMonitorWf(wf, namespace string, scheTime []string, wfClient workflow.WorkflowServiceClient, ctx context.Context) error {
	const max_wf_count = 1000
	var monitorWfObj v1alpha1.Workflow
	err := json.Unmarshal([]byte(backfillWf), &monitorWfObj)
	if err != nil {
		return err
	}
	fmt.Println(len(scheTime))
	count := len(scheTime) / max_wf_count
	startIdx := 0
	for i := 0; i < count; i++ {

		tmpl := monitorWfObj.GetTemplateByName("create-workflow")
		scheTimeByte, err := json.Marshal(scheTime[startIdx : startIdx+max_wf_count])
		if err != nil {
			return err
		}
		tmpl.Resource.Manifest = wf
		stepTmpl := monitorWfObj.GetTemplateByName("main")
		stepTmpl.Steps[0].Steps[0].WithParam = string(scheTimeByte)
		c, err := wfClient.CreateWorkflow(ctx, &workflow.WorkflowCreateRequest{Namespace: namespace, Workflow: &monitorWfObj})
		fmt.Println(c.Namespace + "/" + c.Name)
	}
	return nil
}
