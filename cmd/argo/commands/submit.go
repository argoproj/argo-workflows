package commands

import (
	"context"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	apimachineryversion "k8s.io/apimachinery/pkg/version"

	"github.com/argoproj/argo/cmd/argo/commands/client"
	"github.com/argoproj/argo/cmd/argo/commands/cron"
	"github.com/argoproj/argo/cmd/argo/commands/template"
	"github.com/argoproj/argo/pkg/apis/workflow"
	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	apiwf "github.com/argoproj/argo/server/workflow"
	apiUtil "github.com/argoproj/argo/util/api"
	"github.com/argoproj/argo/workflow/common"
	"github.com/argoproj/argo/workflow/util"
	"github.com/argoproj/pkg/errors"
	argoJson "github.com/argoproj/pkg/json"
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
		from          string
	)
	var command = &cobra.Command{
		Use:   "submit FILE1 FILE2...",
		Short: "submit a workflow",
		Run: func(cmd *cobra.Command, args []string) {
			if cmd.Flag("priority").Changed {
				cliSubmitOpts.priority = &priority
			}

			if from != "" {
				SubmitWorkflowFromResource(from, &submitOpts, &cliSubmitOpts)
			} else {
				if len(args) == 0 {
					cmd.HelpFunc()(cmd, args)
					os.Exit(1)
				}
				SubmitWorkflowsFromFile(args, &submitOpts, &cliSubmitOpts)
			}
		},
	}
	command.Flags().StringVar(&submitOpts.Name, "name", "", "override metadata.name")
	command.Flags().StringVar(&submitOpts.GenerateName, "generate-name", "", "override metadata.generateName")
	command.Flags().StringVar(&submitOpts.Entrypoint, "entrypoint", "", "override entrypoint")
	command.Flags().StringArrayVarP(&submitOpts.Parameters, "parameter", "p", []string{}, "pass an input parameter")
	command.Flags().StringVar(&submitOpts.ServiceAccount, "serviceaccount", "", "run all pods in the workflow using specified serviceaccount")
	command.Flags().StringVar(&submitOpts.InstanceID, "instanceid", "", "submit with a specific controller's instance id label")
	command.Flags().BoolVar(&submitOpts.DryRun, "dry-run", false, "modify the workflow on the client-side without creating it")
	command.Flags().BoolVar(&submitOpts.ServerDryRun, "server-dry-run", false, "send request to server with dry-run flag which will modify the workflow without creating it")
	command.Flags().StringVarP(&cliSubmitOpts.output, "output", "o", "", "Output format. One of: name|json|yaml|wide")
	command.Flags().BoolVarP(&cliSubmitOpts.wait, "wait", "w", false, "wait for the workflow to complete")
	command.Flags().BoolVar(&cliSubmitOpts.watch, "watch", false, "watch the workflow until it completes")
	command.Flags().BoolVar(&cliSubmitOpts.strict, "strict", true, "perform strict workflow validation")
	command.Flags().Int32Var(&priority, "priority", 0, "workflow priority")
	command.Flags().StringVarP(&submitOpts.ParameterFile, "parameter-file", "f", "", "pass a file containing all input parameters")
	command.Flags().StringVarP(&submitOpts.Labels, "labels", "l", "", "Comma separated labels to apply to the workflow. Will override previous values.")
	command.Flags().StringVar(&from, "from", "", "Submit from a WorkflowTempalte or CronWorkflow. E.g., --from=CronWorkflow/hello-world-cwf")
	// Only complete files with appropriate extension.
	err := command.Flags().SetAnnotation("parameter-file", cobra.BashCompFilenameExt, []string{"json", "yaml", "yml"})
	if err != nil {
		log.Fatal(err)
	}
	return command
}

func SubmitWorkflowsFromFile(filePaths []string, submitOpts *util.SubmitOpts, cliOpts *cliSubmitOpts) {
	fileContents, err := util.ReadManifest(filePaths...)
	if err != nil {
		log.Fatal(err)
	}

	var workflows []wfv1.Workflow
	for _, body := range fileContents {
		wfs := unmarshalWorkflows(body, cliOpts.strict)
		workflows = append(workflows, wfs...)
	}

	submitWorkflows(workflows, submitOpts, cliOpts)
}

func SubmitWorkflowFromResource(resourceIdentifier string, submitOpts *util.SubmitOpts, cliOpts *cliSubmitOpts) {

	resIdSplit := strings.Split(resourceIdentifier, "/")
	if len(resIdSplit) != 2 {
		log.Fatalf("resource identifier '%s' is malformed. Expected is KIND/NAME, e.g. CronWorkflow/hello-world-cwf", resourceIdentifier)
	}

	var workflowToSubmit *wfv1.Workflow
	switch resIdSplit[0] {
	case workflow.CronWorkflowKind:
		cwfIf := cron.InitCronWorkflowClient()
		cwf, err := cwfIf.Get(resIdSplit[1], v1.GetOptions{})
		if err != nil {
			log.Fatalf("Unable to get CronWorkflow '%s': %s", resIdSplit[1], err)
		}
		workflowToSubmit, err = common.ConvertCronWorkflowToWorkflow(cwf)
		if err != nil {
			log.Fatalf("Unable to create Workflow from CronWorkflow '%s': %s", resIdSplit[1], err)
		}
	case workflow.WorkflowTemplateKind:
		if submitOpts.Entrypoint == "" {
			log.Fatalf("When submitting a Workflow from a WorkflowTemplate an entrypoint must be passed with --entrypoint")
		}
		wftmplIf := template.InitWorkflowTemplateClient()
		wfTmpl, err := wftmplIf.Get(resIdSplit[1], v1.GetOptions{})
		if err != nil {
			log.Fatalf("Unable to get WorkflowTemplate '%s'", resIdSplit[1])
		}
		workflowToSubmit, err = common.ConvertWorkflowTemplateToWorkflow(wfTmpl, submitOpts.Entrypoint)
		if err != nil {
			log.Fatalf("Unable to create Workflow from WorkflowTemplate '%s': %s", resIdSplit[1], err)
		}
	default:
		log.Fatalf("Resource Kind '%s' is not supported with --from", resIdSplit[0])
	}

	submitWorkflows([]wfv1.Workflow{*workflowToSubmit}, submitOpts, cliOpts)
}

func submitWorkflows(workflows []wfv1.Workflow, submitOpts *util.SubmitOpts, cliOpts *cliSubmitOpts) {
	if submitOpts == nil {
		submitOpts = &util.SubmitOpts{}
	}
	if cliOpts == nil {
		cliOpts = &cliSubmitOpts{}
	}
	defaultWFClient := InitWorkflowClient()
	var defaultNS string

	defaultNS, _, err := client.Config.Namespace()
	if err != nil {
		log.Fatal(err)
	}

	if cliOpts.watch {
		if len(workflows) > 1 {
			log.Fatalf("Cannot watch more than one workflow")
		}
		if cliOpts.wait {
			log.Fatalf("--wait cannot be combined with --watch")
		}
		if submitOpts.DryRun {
			log.Fatalf("--watch cannot be combined with --dry-run")
		}
		if submitOpts.ServerDryRun {
			log.Fatalf("--watch cannot be combined with --server-dry-run")
		}
	}

	if cliOpts.wait {
		if submitOpts.DryRun {
			log.Fatalf("--wait cannot be combined with --dry-run")
		}
		if submitOpts.ServerDryRun {
			log.Fatalf("--wait cannot be combined with --server-dry-run")
		}
	}

	if submitOpts.DryRun {
		if cliOpts.output == "" {
			log.Fatalf("--dry-run should have an output option")
		}
		if submitOpts.ServerDryRun {
			log.Fatalf("--dry-run cannot be combined with --server-dry-run")
		}
	}

	if submitOpts.ServerDryRun {
		if cliOpts.output == "" {
			log.Fatalf("--server-dry-run should have an output option")
		}
		serverVersion, err := wfClientset.Discovery().ServerVersion()
		if err != nil {
			log.Fatalf("Unexpected error while getting the server's api version")
		}
		isCompatible, err := checkServerVersionForDryRun(serverVersion)
		if err != nil {
			log.Fatalf("Unexpected error while checking the server's api version compatibility with --server-dry-run")
		}
		if !isCompatible {
			log.Fatalf("--server-dry-run is not available for server api versions older than v1.12")
		}
	}

	if len(workflows) == 0 {
		log.Println("No Workflow found in given files")
		os.Exit(1)
	}

	var workflowNames []string
	var created *wfv1.Workflow
	var apiGRPCClient apiwf.WorkflowServiceClient
	var ctx context.Context
	if client.ArgoServer != "" {
		conn := client.GetClientConn()
		defer conn.Close()
		apiGRPCClient, ctx = GetWFApiServerGRPCClient(conn)
		errors.CheckError(err)
	}

	for _, wf := range workflows {
		if wf.Namespace == "" {
			// This is here to avoid passing an empty namespace when using --server-dry-run
			wf.Namespace = defaultNS
		}
		if client.ArgoServer != "" {
			err = util.ApplySubmitOpts(&wf, submitOpts)
			errors.CheckError(err)
			created, err = apiUtil.SubmitWorkflowToAPIServer(apiGRPCClient, ctx, &wf, submitOpts.ServerDryRun)
			errors.CheckError(err)
		} else {
			wf.Spec.Priority = cliOpts.priority
			wfClient := defaultWFClient
			if wf.Namespace != defaultNS {
				wfClient = InitWorkflowClient(wf.Namespace)
			} else {
				// This is here to avoid passing an empty namespace when using --server-dry-run
				namespace, _, err := client.Config.Namespace()
				if err != nil {
					log.Fatal(err)
				}
				wf.Namespace = namespace
			}
			created, err = util.SubmitWorkflow(wfClient, wfClientset, namespace, &wf, submitOpts)
		}
		if err != nil {
			log.Fatalf("Failed to submit workflow: %v", err)
		}
		printWorkflow(created, cliOpts.output, DefaultStatus)
		workflowNames = append(workflowNames, created.Name)
	}

	waitOrWatch(workflowNames, *cliOpts)
}

// Checks whether the server has support for the dry-run option
func checkServerVersionForDryRun(serverVersion *apimachineryversion.Info) (bool, error) {
	majorVersion, err := strconv.Atoi(serverVersion.Major)
	if err != nil {
		return false, err
	}
	minorVersion, err := strconv.Atoi(serverVersion.Minor)
	if err != nil {
		return false, err
	}
	if majorVersion < 1 {
		return false, nil
	} else if majorVersion == 1 && minorVersion < 12 {
		return false, nil
	}
	return true, nil
}

// unmarshalWorkflows unmarshals the input bytes as either json or yaml
func unmarshalWorkflows(wfBytes []byte, strict bool) []wfv1.Workflow {
	var wf wfv1.Workflow
	var jsonOpts []argoJson.JSONOpt
	if strict {
		jsonOpts = append(jsonOpts, argoJson.DisallowUnknownFields)
	}
	err := argoJson.Unmarshal(wfBytes, &wf, jsonOpts...)
	if err == nil {
		return []wfv1.Workflow{wf}
	}
	yamlWfs, err := common.SplitWorkflowYAMLFile(wfBytes, strict)
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
