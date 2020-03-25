package commands

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strconv"
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
	cmdutil "github.com/argoproj/argo/util/cmd"
	"github.com/argoproj/argo/workflow/common"
	"github.com/argoproj/argo/workflow/util"
)

// cliSubmitOpts holds submission options specific to CLI submission (e.g. controlling output)
type cliSubmitOpts struct {
	output           string // --output
	wait             bool   // --wait
	watch            bool   // --watch
	strict           bool   // --strict
	priority         *int32 // --priority
	SubstituteParams bool   // --substitute-params
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

func submitWorkflowsFromFile(filePaths []string, submitOpts *util.SubmitOpts, cliOpts *cliSubmitOpts) {
	fileContents, err := util.ReadManifest(filePaths...)
	errors.CheckError(err)
	if cliOpts.SubstituteParams {
		fileContents, err = replaceGlobalParameters(fileContents, submitOpts, cliOpts)
	}
	var workflows []wfv1.Workflow
	for _, body := range fileContents {
		wfs := unmarshalWorkflows(body, cliOpts.strict)
		workflows = append(workflows, wfs...)
	}
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

	// Need to Marshal in order to do the parameter replace
	fileContent, err := yaml.Marshal(workflowToSubmit)
	if err != nil {
		log.Fatalf("Unable to get marshale workflow: %s", err)
	}
	if cliOpts.SubstituteParams {
		fileContents := [][]byte{fileContent}
		fileContents, err = replaceGlobalParameters(fileContents, submitOpts, cliOpts)
		var workflows []wfv1.Workflow
		for _, body := range fileContents {
			wfs := unmarshalWorkflows(body, cliOpts.strict)
			workflows = append(workflows, wfs...)
		}
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

func parseParameters(opts *util.SubmitOpts) ([]wfv1.Parameter, error) {
	newParams := make([]wfv1.Parameter, 0)
	if len(opts.Parameters) > 0 || opts.ParameterFile != "" {
		passedParams := make(map[string]bool)
		for _, paramStr := range opts.Parameters {
			parts := strings.SplitN(paramStr, "=", 2)
			if len(parts) == 1 {
				return nil, fmt.Errorf("Expected parameter of the form: NAME=VALUE. Received: %s", paramStr)
			}
			param := wfv1.Parameter{
				Name:  parts[0],
				Value: &parts[1],
			}
			newParams = append(newParams, param)
			passedParams[param.Name] = true
		}
		if opts.ParameterFile != "" {
			var body []byte
			var err error
			if cmdutil.IsURL(opts.ParameterFile) {
				body, err = util.ReadFromUrl(opts.ParameterFile)
				if err != nil {
					return nil, err
				}
			} else {
				body, err = ioutil.ReadFile(opts.ParameterFile)
				if err != nil {
					return nil, err
				}
			}
			yamlParams := make(map[string]string)
			err = yaml.Unmarshal(body, &yamlParams)
			if err != nil {
				return nil, err
			}

			for k, v := range yamlParams {
				value, err := strconv.Unquote(string(v))
				if err != nil {
					value = string(v)
				}
				param := wfv1.Parameter{
					Name:  k,
					Value: &value,
				}
				if _, ok := passedParams[param.Name]; ok {
					continue
				}
				newParams = append(newParams, param)
				passedParams[param.Name] = true
			}
		}
	}
	return newParams, nil
}

func replaceGlobalParameters(fileContents [][]byte, submitOpts *util.SubmitOpts, cliOpts *cliSubmitOpts) ([][]byte, error) {
	var output [][]byte
	for _, body := range fileContents {
		workflowRaw := make(map[interface{}]interface{})
		err := yaml.Unmarshal(body, &workflowRaw)
		if err != nil {
			return nil, err
		}
		spec, ok := workflowRaw["spec"].(map[interface{}]interface{})
		if !ok {
			log.Fatalf("Problem with the spec defintion for the workflow: '%s'", workflowRaw["spec"])
		}
		args, err := yaml.Marshal(spec["arguments"])
		if err != nil {
			return nil, err
		}
		var argSpec wfv1.Arguments
		err = yaml.Unmarshal(args, &argSpec)
		if err != nil {
			return nil, err
		}
		globalParams := make(map[string]string)
		for _, param := range argSpec.Parameters {
			globalParams["workflow.parameters."+param.Name] = *param.Value
		}
		newParams, err := parseParameters(submitOpts)
		if err != nil {
			return nil, err
		}
		for _, param := range newParams {
			globalParams["workflow.parameters."+param.Name] = *param.Value
		}
		fstTmpl := fasttemplate.New(string(body), `"{{`, `}}"`)
		globalReplacedTmplStr, err := common.Replace(fstTmpl, globalParams, true)
		if err != nil {
			return nil, err
		}
		output = append(output, []byte(globalReplacedTmplStr))
	}
	return output, nil
}
