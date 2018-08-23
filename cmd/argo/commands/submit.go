package commands

import (
	"bufio"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/argoproj/pkg/json"
	"github.com/spf13/cobra"

	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	cmdutil "github.com/argoproj/argo/util/cmd"
	"github.com/argoproj/argo/workflow/common"
	"github.com/argoproj/argo/workflow/validate"
	"github.com/ghodss/yaml"
)

type submitFlags struct {
	name           string   // --name
	generateName   string   // --generate-name
	instanceID     string   // --instanceid
	entrypoint     string   // --entrypoint
	parameters     []string // --parameter
	parameterFile  string   // --parameter-file
	output         string   // --output
	wait           bool     // --wait
	strict         bool     // --strict
	serviceAccount string   // --serviceaccount
}

func NewSubmitCommand() *cobra.Command {
	var (
		submitArgs submitFlags
	)
	var command = &cobra.Command{
		Use:   "submit FILE1 FILE2...",
		Short: "submit a workflow",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) == 0 {
				cmd.HelpFunc()(cmd, args)
				os.Exit(1)
			}
			SubmitWorkflows(args, &submitArgs)
		},
	}
	command.Flags().StringVar(&submitArgs.name, "name", "", "override metadata.name")
	command.Flags().StringVar(&submitArgs.generateName, "generate-name", "", "override metadata.generateName")
	command.Flags().StringVar(&submitArgs.entrypoint, "entrypoint", "", "override entrypoint")
	command.Flags().StringArrayVarP(&submitArgs.parameters, "parameter", "p", []string{}, "pass an input parameter")
	command.Flags().StringVarP(&submitArgs.parameterFile, "parameter-file", "f", "", "pass a file containing all input parameters")
	command.Flags().StringVarP(&submitArgs.output, "output", "o", "", "Output format. One of: name|json|yaml|wide")
	command.Flags().BoolVarP(&submitArgs.wait, "wait", "w", false, "wait for the workflow to complete")
	command.Flags().BoolVar(&submitArgs.strict, "strict", true, "perform strict workflow validation")
	command.Flags().StringVar(&submitArgs.serviceAccount, "serviceaccount", "", "run all pods in the workflow using specified serviceaccount")
	command.Flags().StringVar(&submitArgs.instanceID, "instanceid", "", "submit with a specific controller's instance id label")
	return command
}

func SubmitWorkflows(filePaths []string, submitArgs *submitFlags) {
	InitWorkflowClient()
	var workflows []wfv1.Workflow
	if len(filePaths) == 1 && filePaths[0] == "-" {
		reader := bufio.NewReader(os.Stdin)
		body, err := ioutil.ReadAll(reader)
		if err != nil {
			log.Fatal(err)
		}
		workflows = unmarshalWorkflows(body, submitArgs.strict)
	} else {
		for _, filePath := range filePaths {
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
			wfs := unmarshalWorkflows(body, submitArgs.strict)
			workflows = append(workflows, wfs...)
		}
	}

	var workflowNames []string
	for _, wf := range workflows {
		wfName, err := submitWorkflow(&wf, submitArgs)
		if err != nil {
			log.Fatalf("Failed to submit workflow: %v", err)
		}
		workflowNames = append(workflowNames, wfName)
	}
	if submitArgs.wait {
		wsp := NewWorkflowStatusPoller(wfClient, false, submitArgs.output == "json")
		wsp.WaitWorkflows(workflowNames)
	}
}

// unmarshalWorkflows unmarshals the input bytes as either json or yaml
func unmarshalWorkflows(wfBytes []byte, strict bool) []wfv1.Workflow {
	var wf wfv1.Workflow
	var jsonOpts []json.JSONOpt
	if strict {
		jsonOpts = append(jsonOpts, json.DisallowUnknownFields)
	}
	err := json.Unmarshal(wfBytes, &wf, jsonOpts...)
	if err == nil {
		return []wfv1.Workflow{wf}
	}
	yamlWfs, err := common.SplitYAMLFile(wfBytes, strict)
	if err == nil {
		return yamlWfs
	}
	log.Fatalf("Failed to parse workflow: %v", err)
	return nil
}

// submitWorkflow is a helper to validate and submit a single workflow and override the entrypoint/params supplied from command line
func submitWorkflow(wf *wfv1.Workflow, submitArgs *submitFlags) (string, error) {
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
	if len(submitArgs.parameters) > 0 || submitArgs.parameterFile != "" {
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

		// Add parameters from a parameter-file, if one was provided
		if submitArgs.parameterFile != "" {
			var body []byte
			var err error
			if cmdutil.IsURL(submitArgs.parameterFile) {
				response, err := http.Get(submitArgs.parameterFile)
				if err != nil {
					log.Fatal(err)
				}
				body, err = ioutil.ReadAll(response.Body)
				_ = response.Body.Close()
				if err != nil {
					log.Fatal(err)
				}
			} else {
				body, err = ioutil.ReadFile(submitArgs.parameterFile)
				if err != nil {
					log.Fatal(err)
				}
			}

			yamlParams := map[string]string{}
			err = yaml.Unmarshal(body, &yamlParams)
			if err != nil {
				log.Fatal(err)
			}

			for k, v := range yamlParams {
				param := wfv1.Parameter{
					Name:  k,
					Value: &v,
				}
				if _, ok := passedParams[param.Name]; ok {
					// this parameter was overridden via command line
					continue
				}
				newParams = append(newParams, param)
				passedParams[param.Name] = true
			}
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
	if submitArgs.generateName != "" {
		wf.ObjectMeta.GenerateName = submitArgs.generateName
	}
	if submitArgs.name != "" {
		wf.ObjectMeta.Name = submitArgs.name
	}
	err := validate.ValidateWorkflow(wf)
	if err != nil {
		return "", err
	}
	created, err := wfClient.Create(wf)
	if err != nil {
		return "", err
	}
	printWorkflow(created, submitArgs.output)
	return created.Name, nil
}
