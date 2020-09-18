package cron

import (
	"encoding/json"
	"fmt"
	"github.com/argoproj/argo/cmd/argo/commands/client"
	"github.com/argoproj/argo/pkg/apiclient/cronworkflow"
	"github.com/argoproj/argo/pkg/apiclient/workflow"
	"github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo/workflow/common"
	"github.com/argoproj/pkg/rand"
	"github.com/go-yaml/yaml"
	cron "github.com/robfig/cron/v3"
	"github.com/spf13/cobra"
	"math"
	"os"
	"time"
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
	command.Flags().StringVar(&cliOps.name, "name", "","Backfill name")
	command.Flags().StringVar(&cliOps.startDate, "start", "","Start date")
	command.Flags().StringVar(&cliOps.endDate, "end", "","End Date")
	command.Flags().BoolVar(&cliOps.parallel, "parallel", false,"")
	command.Flags().StringVar(&cliOps.argName, "argname","cronscheduletime","Schedule time argument name")
	command.Flags().StringVar(&cliOps.dateFormat, "format",time.RFC822,"Date format for workflow argument value")

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
	var jobWf []string
	for {
		scheTime = cronTab.Next(scheTime)
		timeStr := scheTime.String()
		wf := common.ConvertCronWorkflowToWorkflow(cronWF)
		wf.GenerateName = cronWF.Name + "-backfill-"+cliOps.name+"-"
		param := v1alpha1.Parameter{
			Name:       cliOps.argName,
			Value:       &timeStr,
		}
		if !cliOps.parallel {
			wf.Spec.Priority = &priority
			wf.Spec.Synchronization = &v1alpha1.Synchronization{
				Mutex:     &v1alpha1.Mutex{Name: cliOps.name},
			}
		}
		wf.Spec.Arguments.Parameters = append(wf.Spec.Arguments.Parameters, param)

		created, err := wfClient.CreateWorkflow(ctx, &workflow.WorkflowCreateRequest{
			Namespace: client.Namespace(),
			Workflow:  wf,
		})
		jobWf = append(jobWf, created.Name)
		if err != nil {
			fmt.Println(err)
		}
		priority--
		if endTime.Before(scheTime) {
			return nil
		}
	}
}

var monitorWf=`
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
   name: monitor-wf
spec:
  entrypoint: monitor
  templates:
  - name: monitor
    steps:
    - - name: watch-wf
        template: watch-wf

  - name: watch-wf
    inputs:
      parameters:  
      - name: workflowname
    container:
      image: alpine:latest
      command: [sh, -c]
      source: |

`

func constructParam(jobWf []string, tmpl *v1alpha1.Template)  error {
	jsonbyte, err := json.Marshal(jobWf)
	if err != nil {
		return err
	}
	item := "{{item}}"
	tmpl.Steps[0].Steps[0].Arguments = v1alpha1.Arguments{
		Parameters:[]v1alpha1.Parameter{v1alpha1.Parameter{
			Name:  "workflowname",
			Value: &item,
		},
		},
	}
	tmpl.Steps[0].Steps[0].WithParam = string(jsonbyte)
	return nil
}

func CreateMonitorWf(jobWf []string, wfClient workflow.WorkflowServiceClient) error {
	var monitorWfObj v1alpha1.Workflow
	err := yaml.Unmarshal([]byte(monitorWf), &monitorWfObj)
	if err != nil {
		return err
	}
	if len(jobWf) == 0 {
		return nil
	}
	if len(jobWf) == 1 {
		err = constructParam(jobWf,monitorWfObj.GetTemplateByName("watch-wf"))

	}
	if err != nil {
		return err
	}

	monitorWfSize := 100
	noSlice := len(jobWf)/monitorWfSize
	startIdx := 0
	endIdx := len(jobWf)

	for i :=1;i<=noSlice; i++{
		splitSlice := jobWf[startIdx: i* monitorWfSize]

	}
}

