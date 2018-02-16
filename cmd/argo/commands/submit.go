package commands

import (
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"

	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	cmdutil "github.com/argoproj/argo/util/cmd"
	"github.com/argoproj/argo/workflow/common"
	"github.com/spf13/cobra"
)

func init() {
	RootCmd.AddCommand(submitCmd)
	submitCmd.Flags().StringVar(&submitArgs.entrypoint, "entrypoint", "", "override entrypoint")
	submitCmd.Flags().StringArrayVarP(&submitArgs.parameters, "parameter", "p", []string{}, "pass an input parameter")
	submitCmd.Flags().StringVarP(&submitArgs.output, "output", "o", "", "Output format. One of: name|json|yaml|wide")
	submitCmd.Flags().BoolVarP(&submitArgs.wait, "wait", "w", false, "wait for the workflow to complete")
	submitCmd.Flags().StringVar(&submitArgs.serviceAccount, "serviceaccount", "", "run all pods in the workflow using specified serviceaccount")
	submitCmd.Flags().StringVar(&submitArgs.instanceID, "instanceid", "", "label selector which limits the controller's watch to a specific instance")
}

type submitFlags struct {
	instanceID     string   // --instanceid
	entrypoint     string   // --entrypoint
	parameters     []string // --parameter
	output         string   // --output
	wait           bool     // --wait
	serviceAccount string   // --serviceaccount
}

var submitArgs submitFlags

var submitCmd = &cobra.Command{
	Use:   "submit FILE1 FILE2...",
	Short: "submit a workflow",
	Run:   SubmitWorkflows,
}

// SubmitWorkflows submits the the specified worfklow manifest files
func SubmitWorkflows(cmd *cobra.Command, args []string) {
	if len(args) == 0 {
		cmd.HelpFunc()(cmd, args)
		os.Exit(1)
	}
	InitWorkflowClient()

	var workflowNames []string
	for _, filePath := range args {
		var body []byte
		var err error
		if cmdutil.IsURL(filePath) {
			response, err := http.Get(filePath)
			if err != nil {
				log.Fatal(err)
			}
			body, err = ioutil.ReadAll(response.Body)
			_ = response.Body.Close()
			if err != nil {
				log.Fatal(err)
			}
		} else {
			body, err = ioutil.ReadFile(filePath)
			if err != nil {
				log.Fatal(err)
			}
		}
		workflows, err := splitYAMLFile(body)
		if err != nil {
			log.Fatalf("%s failed to parse: %v", filePath, err)
		}
		for _, wf := range workflows {
			wfName, err := submitWorkflow(&wf)
			if err != nil {
				log.Fatalf("Workflow manifest %s failed submission: %v", filePath, err)
			}

			workflowNames = append(workflowNames, wfName)
		}
	}

	if submitArgs.wait {
		wsp := NewWorkflowStatusPoller(wfClient, false, submitArgs.output == "json")
		wsp.WaitWorkflows(workflowNames)
	}
}

// submitWorkflow is a helper to validate and submit a single workflow and override the entrypoint/params supplied from command line
func submitWorkflow(wf *wfv1.Workflow) (string, error) {
	if submitArgs.entrypoint != "" {
		wf.Spec.Entrypoint = submitArgs.entrypoint
	}
	if submitArgs.serviceAccount != "" {
		wf.Spec.ServiceAccountName = submitArgs.serviceAccount
	}
	if submitArgs.instanceID != "" {
		labels := wf.GetLabels()
		if labels == nil {
			labels = make(map[string]string)
		}
		labels[common.LabelKeyControllerInstanceID] = submitArgs.instanceID
		wf.SetLabels(labels)
	}
	if len(submitArgs.parameters) > 0 {
		newParams := make([]wfv1.Parameter, 0)
		passedParams := make(map[string]bool)
		for _, paramStr := range submitArgs.parameters {
			parts := strings.SplitN(paramStr, "=", 2)
			if len(parts) == 1 {
				log.Fatalf("Expected parameter of the form: NAME=VALUE. Received: %s", paramStr)
			}
			param := wfv1.Parameter{
				Name:  parts[0],
				Value: &parts[1],
			}
			newParams = append(newParams, param)
			passedParams[param.Name] = true
		}
		for _, param := range wf.Spec.Arguments.Parameters {
			if _, ok := passedParams[param.Name]; ok {
				// this parameter was overridden via command line
				continue
			}
			newParams = append(newParams, param)
		}
		wf.Spec.Arguments.Parameters = newParams
	}
	err := common.ValidateWorkflow(wf)
	if err != nil {
		return "", err
	}
	created, err := wfClient.Create(wf)
	if err != nil {
		return "", err
	}
	printWorkflow(submitArgs.output, created)

	return created.Name, nil
}
