package cron

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"strings"
	"time"

	"github.com/ghodss/yaml"
	cron "github.com/robfig/cron/v3"
	"github.com/spf13/cobra"

	"github.com/argoproj/pkg/rand"

	"github.com/argoproj/argo/cmd/argo/commands/client"
	"github.com/argoproj/argo/pkg/apiclient/cronworkflow"
	"github.com/argoproj/argo/pkg/apiclient/workflow"
	"github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo/workflow/common"
)

type backfillOpts struct {
	cronWfName string
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

			cliOps.cronWfName = args[0]
			return backfillCronWorkflow(args[0], cliOps)
		},
	}
	command.Flags().StringVar(&cliOps.name, "name", "", "Backfill name")
	command.Flags().StringVar(&cliOps.startDate, "start", "", "Start date")
	command.Flags().StringVar(&cliOps.endDate, "end", "", "End Date")
	command.Flags().BoolVar(&cliOps.parallel, "parallel", false, "Enabled all backfile workflows run parallel")
	command.Flags().StringVar(&cliOps.argName, "argname", "cronScheduleTime", "Schedule time argument name for workflow")
	command.Flags().StringVar(&cliOps.dateFormat, "format", time.RFC822, "Date format for Schedule time value")

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
		cliOps.endDate = endTime.Format(time.RFC822)
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
	paramArg := `{{inputs.parameters.backfillscheduletime}}`
	wf.GenerateName = cronWF.Name + "-backfill-" + strings.ToLower(cliOps.name) + "-"
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
	if err != nil {
		return err
	}
	yamlbyte, err := yaml.JSONToYAML(wfJsonByte)
	if err != nil {
		return err
	}
	wfYamlStr := "apiVersion: argoproj.io/v1alpha1 \n" + string(yamlbyte)
	if len(scheList) > 0 {
		return CreateMonitorWf(ctx, wfYamlStr, client.Namespace(), scheList, wfClient, cliOps)
	}
	return nil
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
                              "name": "backfillscheduletime",
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
                     "name": "backfillscheduletime"
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

func CreateMonitorWf(ctx context.Context, wf, namespace string, scheTime []string, wfClient workflow.WorkflowServiceClient, cliOps backfillOpts) error {
	const maxWfCount = 1000
	var monitorWfObj v1alpha1.Workflow
	err := json.Unmarshal([]byte(backfillWf), &monitorWfObj)
	if err != nil {
		return err
	}
	TotalScheCount := len(scheTime)
	iterCount := int(float64(len(scheTime)/maxWfCount)) + 1
	startIdx := 0
	var endIdx int
	var wfNames []string
	for i := 0; i < iterCount; i++ {
		tmpl := monitorWfObj.GetTemplateByName("create-workflow")
		if (TotalScheCount - i*maxWfCount) < maxWfCount {
			endIdx = TotalScheCount
		} else {
			endIdx = startIdx + maxWfCount
		}
		scheTimeByte, err := json.Marshal(scheTime[startIdx:endIdx])
		startIdx = endIdx
		if err != nil {

			return err
		}
		tmpl.Resource.Manifest = fmt.Sprint(wf)
		stepTmpl := monitorWfObj.GetTemplateByName("main")
		stepTmpl.Steps[0].Steps[0].WithParam = string(scheTimeByte)
		c, err := wfClient.CreateWorkflow(ctx, &workflow.WorkflowCreateRequest{Namespace: namespace, Workflow: &monitorWfObj})
		if err != nil {
			return err
		}
		wfNames = append(wfNames, c.Name)
	}
	printBackFillOutput(wfNames, len(scheTime), cliOps)
	return nil
}

func printBackFillOutput(wfNames []string, totalSches int, cliOps backfillOpts) {
	fmt.Printf("Created %s Backfill task for Cronworkflow %s \n", cliOps.name, cliOps.cronWfName)
	fmt.Printf("==================================================\n")
	fmt.Printf("Backfill Period :\n")
	fmt.Printf("Start Time : %s \n", cliOps.startDate)
	fmt.Printf("End Time : %s \n", cliOps.endDate)
	fmt.Printf("Total Backfill Schedule: %d \n", totalSches)
	fmt.Printf("==================================================\n")
	fmt.Printf("Backfill Workflows: \n")
	fmt.Printf("   NAMESPACE\t WORKFLOW: \n")
	namespace := client.Namespace()
	for idx, wfName := range wfNames {
		fmt.Printf("%d. %s \t %s \n", idx+1, namespace, wfName)
	}
}
