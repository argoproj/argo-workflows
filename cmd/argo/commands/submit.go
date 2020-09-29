package commands

import (
	"context"
	"fmt"
	"log"
	"strings"

	argoJson "github.com/argoproj/pkg/json"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/argoproj/argo/cmd/argo/commands/client"
	cmdcommon "github.com/argoproj/argo/cmd/argo/commands/common"
	workflowpkg "github.com/argoproj/argo/pkg/apiclient/workflow"
	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo/workflow/common"
	"github.com/argoproj/argo/workflow/util"
)

// cliSubmitOpts holds submission options specific to CLI submission (e.g. controlling output)
type cliSubmitOpts struct {
	output   string // --output
	wait     bool   // --wait
	watch    bool   // --watch
	log      bool   // --log
	strict   bool   // --strict
	priority *int32 // --priority
	getArgs  getFlags
}

func NewSubmitCommand() *cobra.Command {
	var (
		submitOpts    wfv1.SubmitOpts
		cliSubmitOpts cliSubmitOpts
		priority      int32
		from          string
	)
	var command = &cobra.Command{
		Use:   "submit [FILE... | --from `kind/name]",
		Short: "submit a workflow",
		Example: `# Submit multiple workflows from files:

  argo submit my-wf.yaml

# Submit and wait for completion:

  argo submit --wait my-wf.yaml

# Submit and watch until completion:

  argo submit --watch my-wf.yaml

# Submit and tail logs until completion:

  argo submit --log my-wf.yaml

# Submit a single workflow from an existing resource

  argo submit --from cronwf/my-cron-wf
`,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if cmd.Flag("priority").Changed {
				cliSubmitOpts.priority = &priority
			}

			if !cliSubmitOpts.watch && len(cliSubmitOpts.getArgs.status) > 0 {
				logrus.Warn("--status should only be used with --watch")
			}

			ctx, apiClient := cmdcommon.CreateNewAPIClientFunc()
			serviceClient := apiClient.NewWorkflowServiceClient()
			namespace := client.Namespace()
			if from != "" {
				if len(args) != 0 {
					cmd.HelpFunc()(cmd, args)
					return cmdcommon.MissingArgumentsError
				}
				return submitWorkflowFromResource(ctx, serviceClient, namespace, from, &submitOpts, &cliSubmitOpts)
			} else {
				return submitWorkflowsFromFile(ctx, serviceClient, namespace, args, &submitOpts, &cliSubmitOpts)
			}
		},
	}
	util.PopulateSubmitOpts(command, &submitOpts, true)
	command.Flags().StringVarP(&cliSubmitOpts.output, "output", "o", "", "Output format. One of: name|json|yaml|wide")
	command.Flags().BoolVarP(&cliSubmitOpts.wait, "wait", "w", false, "wait for the workflow to complete")
	command.Flags().BoolVar(&cliSubmitOpts.watch, "watch", false, "watch the workflow until it completes")
	command.Flags().BoolVar(&cliSubmitOpts.log, "log", false, "log the workflow until it completes")
	command.Flags().BoolVar(&cliSubmitOpts.strict, "strict", true, "perform strict workflow validation")
	command.Flags().Int32Var(&priority, "priority", 0, "workflow priority")
	command.Flags().StringVar(&from, "from", "", "Submit from an existing `kind/name` E.g., --from=cronwf/hello-world-cwf")
	command.Flags().StringVar(&cliSubmitOpts.getArgs.status, "status", "", "Filter by status (Pending, Running, Succeeded, Skipped, Failed, Error). Should only be used with --watch.")
	command.Flags().StringVar(&cliSubmitOpts.getArgs.nodeFieldSelectorString, "node-field-selector", "", "selector of node to display, eg: --node-field-selector phase=abc")
	// Only complete files with appropriate extension.
	err := command.Flags().SetAnnotation("parameter-file", cobra.BashCompFilenameExt, []string{"json", "yaml", "yml"})
	if err != nil {
		log.Fatal(err)
	}
	return command
}

func submitWorkflowsFromFile(ctx context.Context, serviceClient workflowpkg.WorkflowServiceClient, namespace string, filePaths []string, submitOpts *wfv1.SubmitOpts, cliOpts *cliSubmitOpts) error {
	fileContents, err := util.ReadManifest(filePaths...)
	if err != nil {
		return err
	}

	var workflows []wfv1.Workflow
	for _, body := range fileContents {
		wfs, err := unmarshalWorkflows(body, cliOpts.strict)
		if err != nil {
			return err
		}
		workflows = append(workflows, wfs...)
	}

	return submitWorkflows(ctx, serviceClient, namespace, workflows, submitOpts, cliOpts)
}

func validateOptions(workflows []wfv1.Workflow, submitOpts *wfv1.SubmitOpts, cliOpts *cliSubmitOpts) error {
	if cliOpts.watch {
		if len(workflows) > 1 {
			return fmt.Errorf("cannot watch more than one workflow")
		}
		if cliOpts.wait {
			return fmt.Errorf("--wait cannot be combined with --watch")
		}
		if submitOpts.DryRun {
			return fmt.Errorf("--watch cannot be combined with --dry-run")
		}
		if submitOpts.ServerDryRun {
			return fmt.Errorf("--watch cannot be combined with --server-dry-run")
		}
	}

	if cliOpts.wait {
		if submitOpts.DryRun {
			return fmt.Errorf("--wait cannot be combined with --dry-run")
		}
		if submitOpts.ServerDryRun {
			return fmt.Errorf("--wait cannot be combined with --server-dry-run")
		}
	}

	if submitOpts.DryRun {
		if cliOpts.output == "" {
			return fmt.Errorf("--dry-run should have an output option")
		}
		if submitOpts.ServerDryRun {
			return fmt.Errorf("--dry-run cannot be combined with --server-dry-run")
		}
	}

	if submitOpts.ServerDryRun {
		if cliOpts.output == "" {
			return fmt.Errorf("--server-dry-run should have an output option")
		}
	}
	return nil
}

func submitWorkflowFromResource(ctx context.Context, serviceClient workflowpkg.WorkflowServiceClient, namespace string, resourceIdentifier string, submitOpts *wfv1.SubmitOpts, cliOpts *cliSubmitOpts) error {

	parts := strings.SplitN(resourceIdentifier, "/", 2)
	if len(parts) != 2 {
		return fmt.Errorf("resource identifier '%s' is malformed. Should be `kind/name`, e.g. cronwf/hello-world-cwf", resourceIdentifier)
	}
	kind := parts[0]
	name := parts[1]

	tempwf := wfv1.Workflow{}

	err := validateOptions([]wfv1.Workflow{tempwf}, submitOpts, cliOpts)
	if err != nil {
		return err
	}
	created, err := serviceClient.SubmitWorkflow(ctx, &workflowpkg.WorkflowSubmitRequest{
		Namespace:     namespace,
		ResourceKind:  kind,
		ResourceName:  name,
		SubmitOptions: submitOpts,
	})
	if err != nil {
		return fmt.Errorf("failed to submit workflow: %v", err)
	}

	err = printWorkflow(created, getFlags{output: cliOpts.output})
	if err != nil {
		return err
	}
	return waitWatchOrLog(ctx, serviceClient, namespace, []string{created.Name}, *cliOpts)
}

func submitWorkflows(ctx context.Context, serviceClient workflowpkg.WorkflowServiceClient, namespace string, workflows []wfv1.Workflow, submitOpts *wfv1.SubmitOpts, cliOpts *cliSubmitOpts) error {

	err := validateOptions(workflows, submitOpts, cliOpts)
	if err != nil {
		return err
	}

	if len(workflows) == 0 {
		log.Println("No Workflow found in given files")
		return nil
	}

	var workflowNames []string

	for _, wf := range workflows {
		if wf.Namespace == "" {
			// This is here to avoid passing an empty namespace when using --server-dry-run
			wf.Namespace = namespace
		}
		err := util.ApplySubmitOpts(&wf, submitOpts)
		if err != nil {
			return err
		}
		wf.Spec.Priority = cliOpts.priority
		options := &metav1.CreateOptions{}
		if submitOpts.DryRun {
			options.DryRun = []string{"All"}
		}
		created, err := serviceClient.CreateWorkflow(ctx, &workflowpkg.WorkflowCreateRequest{
			Namespace:     wf.Namespace,
			Workflow:      &wf,
			ServerDryRun:  submitOpts.ServerDryRun,
			CreateOptions: options,
		})
		if err != nil {
			return fmt.Errorf("failed to submit workflow: %v", err)
		}

		err = printWorkflow(created, getFlags{output: cliOpts.output, status: cliOpts.getArgs.status})
		if err != nil {
			return err
		}
		workflowNames = append(workflowNames, created.Name)
	}

	return waitWatchOrLog(ctx, serviceClient, namespace, workflowNames, *cliOpts)
}

// unmarshalWorkflows unmarshals the input bytes as either json or yaml
func unmarshalWorkflows(wfBytes []byte, strict bool) ([]wfv1.Workflow, error) {
	var wf wfv1.Workflow
	var jsonOpts []argoJson.JSONOpt
	if strict {
		jsonOpts = append(jsonOpts, argoJson.DisallowUnknownFields)
	}
	err := argoJson.Unmarshal(wfBytes, &wf, jsonOpts...)
	if err == nil {
		return []wfv1.Workflow{wf}, nil
	}
	yamlWfs, err := common.SplitWorkflowYAMLFile(wfBytes, strict)
	if err != nil {
		return nil, fmt.Errorf("failed to parse workflow: %v", err)
	}
	return yamlWfs, nil
}

func waitWatchOrLog(ctx context.Context, serviceClient workflowpkg.WorkflowServiceClient, namespace string, workflowNames []string, cliSubmitOpts cliSubmitOpts) error {
	if cliSubmitOpts.log {
		for _, workflow := range workflowNames {
			err := logWorkflow(ctx, serviceClient, namespace, workflow, "", &corev1.PodLogOptions{
				Container: "main",
				Follow:    true,
				Previous:  false,
			})
			if err != nil {
				return err
			}
		}
	}
	if cliSubmitOpts.wait {
		return waitWorkflows(ctx, serviceClient, namespace, workflowNames, false, !(cliSubmitOpts.output == "" || cliSubmitOpts.output == "wide"))
	} else if cliSubmitOpts.watch {
		for _, workflow := range workflowNames {
			return watchWorkflow(ctx, serviceClient, namespace, workflow, cliSubmitOpts.getArgs)
		}
	}
	return nil
}
