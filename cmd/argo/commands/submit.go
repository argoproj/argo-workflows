package commands

import (
	"log"
	"os"
	"strings"

	"github.com/argoproj/pkg/errors"
	argoJson "github.com/argoproj/pkg/json"
	"github.com/spf13/cobra"
	"github.com/valyala/fasttemplate"
	"gopkg.in/yaml.v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/argoproj/argo/cmd/argo/commands/client"
	cronworkflowpkg "github.com/argoproj/argo/pkg/apiclient/cronworkflow"
	workflowpkg "github.com/argoproj/argo/pkg/apiclient/workflow"
	workflowtemplatepkg "github.com/argoproj/argo/pkg/apiclient/workflowtemplate"
	"github.com/argoproj/argo/pkg/apis/workflow"
	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo/workflow/common"
	"github.com/argoproj/argo/workflow/util"
)

// cliSubmitOpts holds submission options specific to CLI submission (e.g. controlling output)
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
		Use:   "submit [FILE... | --from `kind/name]",
		Short: "submit a workflow",
		Example: `# Submit multiple workflows from files:

  argo submit my-wf.yaml

# Submit a single workflow from an existing resource

  argo submit --from cronwf/my-cron-wf
`,
		Run: func(cmd *cobra.Command, args []string) {
			if cmd.Flag("priority").Changed {
				cliSubmitOpts.priority = &priority
			}

			if from != "" {
				if len(args) != 0 {
					cmd.HelpFunc()(cmd, args)
					os.Exit(1)
				}
				submitWorkflowFromResource(from, &submitOpts, &cliSubmitOpts)
			} else {
				submitWorkflowsFromFile(args, &submitOpts, &cliSubmitOpts)
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
	command.Flags().StringVar(&from, "from", "", "Submit from an existing `kind/name` E.g., --from=cronwf/hello-world-cwf")
	// Only complete files with appropriate extension.
	err := command.Flags().SetAnnotation("parameter-file", cobra.BashCompFilenameExt, []string{"json", "yaml", "yml"})
	if err != nil {
		log.Fatal(err)
	}
	return command
}

/*
func replaceParameters(workflows []wfv1.Workflow, submitOpts *util.SubmitOpts) ([]wfv1.Workflow, error) {
	if submitOpts.SubstituteParams {
		var _workflows []wfv1.Workflow
		for _, wf := range workflows {
			//localParams := make(map[string]string)
			globalParams := make(map[string]string)
			for _, param := range wf.Spec.Arguments.Parameters {
				globalParams["workflow.parameters."+param.Name] = *param.Value
			}
			var templates []wfv1.Template
			for _, tmpWf := range wf.Spec.Templates {
				tmplBytes, err := json.Marshal(tmpWf)
				if err != nil {
					return nil, err
				}
				fstTmpl := fasttemplate.New(string(tmplBytes), "{{", "}}")
				globalReplacedTmplStr, err := common.Replace(fstTmpl, globalParams, true)
				if err != nil {
					return nil, err
				}
				var template wfv1.Template
				err = json.Unmarshal([]byte(globalReplacedTmplStr), &template)
				if err != nil {
					return nil, err
				}
				templates = append(templates, template)
				fmt.Println("Here, Here, Here")
				fmt.Println(globalReplacedTmplStr)
			}
			wf.Spec.Templates = templates
			_workflows = append(_workflows, wf)
		}
		return _workflows, nil
	}
	return workflows, nil
}
*/
func replaceGlobalParameters(fileContents [][]byte) ([][]byte, error) {
	// 1
	var output [][]byte
	for _, body := range fileContents {
		// 2
		workflowRaw := make(map[interface{}]interface{})
		err := yaml.Unmarshal(body, &workflowRaw)
		if err != nil {
			return nil, err
		}
		// 3
		spec, _ := yaml.Marshal(workflowRaw["spec"])
		var wfSpec wfv1.WorkflowSpec
		yaml.Unmarshal(spec, &wfSpec)
		globalParams := make(map[string]string)
		for _, param := range wfSpec.Arguments.Parameters {
			globalParams["workflow.parameters."+param.Name] = *param.Value
		}
		fstTmpl := fasttemplate.New(string(body), `"{{`, `}}"`)
		globalReplacedTmplStr, err := common.Replace(fstTmpl, globalParams, true)
		output = append(output, []byte(globalReplacedTmplStr))
	}
	return output, nil
}

func submitWorkflowsFromFile(filePaths []string, submitOpts *util.SubmitOpts, cliOpts *cliSubmitOpts) {
	fileContents, err := util.ReadManifest(filePaths...)
	errors.CheckError(err)

	var workflows []wfv1.Workflow
	for _, body := range fileContents {
		wfs := unmarshalWorkflows(body, cliOpts.strict)
		workflows = append(workflows, wfs...)
	}
	/*
		if opts.SubstituteParams {
			fmt.Println("Print Start")
			localParams := make(map[string]string)
			globalParams := make(map[string]string)
			for _, param := range wf.Spec.Arguments.Parameters {
				globalParams["workflow.parameters."+param.Name] = *param.Value
			}
			for _, tmpWf := range wf.Spec.Templates {
				for _, param := range tmpWf.Arguments.Parameters {
					localParams["inputs.parameters."+param.Name] = *param.Value
				}
				fmt.Println("These are the global/workflow once:")
				fmt.Println(globalParams)
				fmt.Println("These are the local/input once:")
				fmt.Println(localParams)
				//fstTmpl := fasttemplate.New(string(tmplBytes), "{{", "}}")
				//globalReplacedTmplStr, err := Replace(fstTmpl, globalParams, true)
				fmt.Println()
				//processedTmpl, _ := common.ProcessArgs(&tmpWf, args, globalParams, localParams, false)
				//newTmpl := tmpWf.DeepCopy()
				//processedTmpl, _ := common.SubstituteParams(newTmpl, globalParams, localParams)
				//println("After:")
				//fmt.Println(*newTmpl)
				//fmt.Println(*processedTmpl)
			}
			fmt.Println("Print Done")
		}
	*/

	submitWorkflows(workflows, submitOpts, cliOpts)
}

func submitWorkflowFromResource(resourceIdentifier string, submitOpts *util.SubmitOpts, cliOpts *cliSubmitOpts) {

	parts := strings.SplitN(resourceIdentifier, "/", 2)
	if len(parts) != 2 {
		log.Fatalf("resource identifier '%s' is malformed. Should be `kind/name`, e.g. cronwf/hello-world-cwf", resourceIdentifier)
	}
	kind := parts[0]
	name := parts[1]

	ctx, apiClient := client.NewAPIClient()
	namespace := client.Namespace()

	var workflowToSubmit *wfv1.Workflow
	switch kind {
	case workflow.CronWorkflowKind, workflow.CronWorkflowSingular, workflow.CronWorkflowPlural, workflow.CronWorkflowShortName:
		serviceClient := apiClient.NewCronWorkflowServiceClient()
		cronWf, err := serviceClient.GetCronWorkflow(ctx, &cronworkflowpkg.GetCronWorkflowRequest{
			Name:      name,
			Namespace: namespace,
		})
		if err != nil {
			log.Fatalf("Unable to get cron workflow '%s': %s", name, err)
		}
		workflowToSubmit = common.ConvertCronWorkflowToWorkflow(cronWf)
	case workflow.WorkflowTemplateKind, workflow.WorkflowTemplateSingular, workflow.WorkflowTemplatePlural, workflow.WorkflowTemplateShortName:
		serviceClient := apiClient.NewWorkflowTemplateServiceClient()
		template, err := serviceClient.GetWorkflowTemplate(ctx, &workflowtemplatepkg.WorkflowTemplateGetRequest{
			Name:      name,
			Namespace: namespace,
		})
		if err != nil {
			log.Fatalf("Unable to get workflow template '%s': %s", name, err)
		}
		workflowToSubmit = common.ConvertWorkflowTemplateToWorkflow(template)
	default:
		log.Fatalf("Resource kind '%s' is not supported with --from", kind)
	}

	submitWorkflows([]wfv1.Workflow{*workflowToSubmit}, submitOpts, cliOpts)
}

func submitWorkflows(workflows []wfv1.Workflow, submitOpts *util.SubmitOpts, cliOpts *cliSubmitOpts) {

	ctx, apiClient := client.NewAPIClient()
	serviceClient := apiClient.NewWorkflowServiceClient()
	namespace := client.Namespace()

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
	}

	if len(workflows) == 0 {
		log.Println("No Workflow found in given files")
		os.Exit(1)
	}

	var workflowNames []string

	for _, wf := range workflows {
		if wf.Namespace == "" {
			// This is here to avoid passing an empty namespace when using --server-dry-run
			wf.Namespace = namespace
		}
		err := util.ApplySubmitOpts(&wf, submitOpts)
		errors.CheckError(err)
		wf.Spec.Priority = cliOpts.priority
		options := &metav1.CreateOptions{}
		if submitOpts.DryRun {
			options.DryRun = []string{"All"}
		}
		created, err := serviceClient.CreateWorkflow(ctx, &workflowpkg.WorkflowCreateRequest{
			Namespace:     wf.Namespace,
			Workflow:      &wf,
			InstanceID:    submitOpts.InstanceID,
			ServerDryRun:  submitOpts.ServerDryRun,
			CreateOptions: options,
		})
		if err != nil {
			log.Fatalf("Failed to submit workflow: %v", err)
		}
		printWorkflow(created, cliOpts.output, DefaultStatus)
		workflowNames = append(workflowNames, created.Name)
	}

	waitOrWatch(workflowNames, *cliOpts)
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
