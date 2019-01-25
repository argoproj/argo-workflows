package commands

import (
	"bufio"
	"io/ioutil"
	"log"
	"net/http"
	"os"

	"github.com/argoproj/pkg/json"
	"github.com/spf13/cobra"

	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	cmdutil "github.com/argoproj/argo/util/cmd"
	"github.com/argoproj/argo/workflow/common"
	"github.com/argoproj/argo/workflow/util"
)

// cliSubmitOpts holds submition options specific to CLI submission (e.g. controlling output)
type cliSubmitOpts struct {
	output   string // --output
	wait     bool   // --wait
	watch    bool   // --watch
	strict   bool   // --strict
	priority *int32 // --priority
}

func NewSubmitCommand() *cobra.Command {
	var (
		submitOpts    util.SubmitOpts
		cliSubmitOpts cliSubmitOpts
		priority      int32
	)
	var command = &cobra.Command{
		Use:   "submit FILE1 FILE2...",
		Short: "submit a workflow",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) == 0 {
				cmd.HelpFunc()(cmd, args)
				os.Exit(1)
			}
			if cmd.Flag("priority").Changed {
				cliSubmitOpts.priority = &priority
			}

			SubmitWorkflows(args, &submitOpts, &cliSubmitOpts)
		},
	}
	command.Flags().StringVar(&submitOpts.Name, "name", "", "override metadata.name")
	command.Flags().StringVar(&submitOpts.GenerateName, "generate-name", "", "override metadata.generateName")
	command.Flags().StringVar(&submitOpts.Entrypoint, "entrypoint", "", "override entrypoint")
	command.Flags().StringArrayVarP(&submitOpts.Parameters, "parameter", "p", []string{}, "pass an input parameter")
	command.Flags().StringVarP(&submitOpts.ParameterFile, "parameter-file", "f", "", "pass a file containing all input parameters")
	command.Flags().StringVar(&submitOpts.ServiceAccount, "serviceaccount", "", "run all pods in the workflow using specified serviceaccount")
	command.Flags().StringVar(&submitOpts.InstanceID, "instanceid", "", "submit with a specific controller's instance id label")
	command.Flags().StringVarP(&cliSubmitOpts.output, "output", "o", "", "Output format. One of: name|json|yaml|wide")
	command.Flags().BoolVarP(&cliSubmitOpts.wait, "wait", "w", false, "wait for the workflow to complete")
	command.Flags().BoolVar(&cliSubmitOpts.watch, "watch", false, "watch the workflow until it completes")
	command.Flags().BoolVar(&cliSubmitOpts.strict, "strict", true, "perform strict workflow validation")
	command.Flags().Int32Var(&priority, "priority", 0, "workflow priority")
	return command
}

func SubmitWorkflows(filePaths []string, submitOpts *util.SubmitOpts, cliOpts *cliSubmitOpts) {
	if submitOpts == nil {
		submitOpts = &util.SubmitOpts{}
	}
	if cliOpts == nil {
		cliOpts = &cliSubmitOpts{}
	}
	InitWorkflowClient()
	var workflows []wfv1.Workflow
	if len(filePaths) == 1 && filePaths[0] == "-" {
		reader := bufio.NewReader(os.Stdin)
		body, err := ioutil.ReadAll(reader)
		if err != nil {
			log.Fatal(err)
		}
		workflows = unmarshalWorkflows(body, cliOpts.strict)
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
			wfs := unmarshalWorkflows(body, cliOpts.strict)
			workflows = append(workflows, wfs...)
		}
	}

	if cliOpts.watch {
		if len(workflows) > 1 {
			log.Fatalf("Cannot watch more than one workflow")
		}
		if cliOpts.wait {
			log.Fatalf("--wait cannot be combined with --watch")
		}
	}

	var workflowNames []string
	for _, wf := range workflows {
		wf.Spec.Priority = cliOpts.priority
		created, err := util.SubmitWorkflow(wfClient, &wf, submitOpts)
		if err != nil {
			log.Fatalf("Failed to submit workflow: %v", err)
		}
		printWorkflow(created, cliOpts.output)
		workflowNames = append(workflowNames, created.Name)
	}
	waitOrWatch(workflowNames, *cliOpts)
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

func waitOrWatch(workflowNames []string, cliSubmitOpts cliSubmitOpts) {
	if cliSubmitOpts.wait {
		WaitWorkflows(workflowNames, false, !(cliSubmitOpts.output == "" || cliSubmitOpts.output == "wide"))
	} else if cliSubmitOpts.watch {
		watchWorkflow(workflowNames[0])
	}
}
