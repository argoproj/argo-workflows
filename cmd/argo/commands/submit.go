package commands

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/argoproj/pkg/errors"
	argoJson "github.com/argoproj/pkg/json"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/argoproj/argo-workflows/v3/cmd/argo/commands/client"
	common "github.com/argoproj/argo-workflows/v3/cmd/argo/commands/common"
	workflowpkg "github.com/argoproj/argo-workflows/v3/pkg/apiclient/workflow"
	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	wfcommon "github.com/argoproj/argo-workflows/v3/workflow/common"
	"github.com/argoproj/argo-workflows/v3/workflow/util"
)

func NewSubmitCommand() *cobra.Command {
	var (
		submitOpts     wfv1.SubmitOpts
		parametersFile string
		cliSubmitOpts  common.CliSubmitOpts
		priority       int32
		from           string
	)
	command := &cobra.Command{
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
		Run: func(cmd *cobra.Command, args []string) {
			if cmd.Flag("priority").Changed {
				cliSubmitOpts.Priority = &priority
			}

			if !cliSubmitOpts.Watch && len(cliSubmitOpts.GetArgs.Status) > 0 {
				log.Warn("--status should only be used with --watch")
			}

			if parametersFile != "" {
				err := util.ReadParametersFile(parametersFile, &submitOpts)
				errors.CheckError(err)
			}

			ctx, apiClient := client.NewAPIClient(cmd.Context())
			serviceClient := apiClient.NewWorkflowServiceClient()
			namespace := client.Namespace()
			if from != "" {
				if len(args) != 0 {
					cmd.HelpFunc()(cmd, args)
					os.Exit(1)
				}
				submitWorkflowFromResource(ctx, serviceClient, namespace, from, &submitOpts, &cliSubmitOpts)
			} else {
				submitWorkflowsFromFile(ctx, serviceClient, namespace, args, &submitOpts, &cliSubmitOpts)
			}
		},
	}
	util.PopulateSubmitOpts(command, &submitOpts, &parametersFile, true)
	command.Flags().StringVarP(&cliSubmitOpts.Output, "output", "o", "", "Output format. One of: name|json|yaml|wide")
	command.Flags().BoolVarP(&cliSubmitOpts.Wait, "wait", "w", false, "wait for the workflow to complete")
	command.Flags().BoolVar(&cliSubmitOpts.Watch, "watch", false, "watch the workflow until it completes")
	command.Flags().BoolVar(&cliSubmitOpts.Log, "log", false, "log the workflow until it completes")
	command.Flags().BoolVar(&cliSubmitOpts.Strict, "strict", true, "perform strict workflow validation")
	command.Flags().Int32Var(&priority, "priority", 0, "workflow priority")
	command.Flags().StringVar(&from, "from", "", "Submit from an existing `kind/name` E.g., --from=cronwf/hello-world-cwf")
	command.Flags().StringVar(&cliSubmitOpts.GetArgs.Status, "status", "", "Filter by status (Pending, Running, Succeeded, Skipped, Failed, Error). Should only be used with --watch.")
	command.Flags().StringVar(&cliSubmitOpts.GetArgs.NodeFieldSelectorString, "node-field-selector", "", "selector of node to display, eg: --node-field-selector phase=abc")
	command.Flags().StringVar(&cliSubmitOpts.ScheduledTime, "scheduled-time", "", "Override the workflow's scheduledTime parameter (useful for backfilling). The time must be RFC3339")

	// Only complete files with appropriate extension.
	err := command.Flags().SetAnnotation("parameter-file", cobra.BashCompFilenameExt, []string{"json", "yaml", "yml"})
	if err != nil {
		log.Fatal(err)
	}
	return command
}

func submitWorkflowsFromFile(ctx context.Context, serviceClient workflowpkg.WorkflowServiceClient, namespace string, filePaths []string, submitOpts *wfv1.SubmitOpts, cliOpts *common.CliSubmitOpts) {
	fileContents, err := util.ReadManifest(filePaths...)
	errors.CheckError(err)

	var workflows []wfv1.Workflow
	for _, body := range fileContents {
		wfs := unmarshalWorkflows(body, cliOpts.Strict)
		workflows = append(workflows, wfs...)
	}

	submitWorkflows(ctx, serviceClient, namespace, workflows, submitOpts, cliOpts)
}

func validateOptions(workflows []wfv1.Workflow, submitOpts *wfv1.SubmitOpts, cliOpts *common.CliSubmitOpts) {
	if cliOpts.Watch {
		if len(workflows) > 1 {
			log.Fatalf("Cannot watch more than one workflow")
		}
		if cliOpts.Wait {
			log.Fatalf("--wait cannot be combined with --watch")
		}
		if submitOpts.DryRun {
			log.Fatalf("--watch cannot be combined with --dry-run")
		}
		if submitOpts.ServerDryRun {
			log.Fatalf("--watch cannot be combined with --server-dry-run")
		}
	}

	if cliOpts.Wait {
		if submitOpts.DryRun {
			log.Fatalf("--wait cannot be combined with --dry-run")
		}
		if submitOpts.ServerDryRun {
			log.Fatalf("--wait cannot be combined with --server-dry-run")
		}
	}

	if submitOpts.DryRun {
		if cliOpts.Output == "" {
			log.Fatalf("--dry-run should have an output option")
		}
		if submitOpts.ServerDryRun {
			log.Fatalf("--dry-run cannot be combined with --server-dry-run")
		}
	}

	if submitOpts.ServerDryRun {
		if cliOpts.Output == "" {
			log.Fatalf("--server-dry-run should have an output option")
		}
	}
}

func submitWorkflowFromResource(ctx context.Context, serviceClient workflowpkg.WorkflowServiceClient, namespace string, resourceIdentifier string, submitOpts *wfv1.SubmitOpts, cliOpts *common.CliSubmitOpts) {
	parts := strings.SplitN(resourceIdentifier, "/", 2)
	if len(parts) != 2 {
		log.Fatalf("resource identifier '%s' is malformed. Should be `kind/name`, e.g. cronwf/hello-world-cwf", resourceIdentifier)
	}
	kind := parts[0]
	name := parts[1]

	tempwf := wfv1.Workflow{}

	validateOptions([]wfv1.Workflow{tempwf}, submitOpts, cliOpts)
	if cliOpts.ScheduledTime != "" {
		_, err := time.Parse(time.RFC3339, cliOpts.ScheduledTime)
		if err != nil {
			log.Fatalf("scheduled-time contains invalid time.RFC3339 format. (e.g.: `2006-01-02T15:04:05-07:00`)")
		}
		submitOpts.Annotations = fmt.Sprintf("%s=%s", wfcommon.AnnotationKeyCronWfScheduledTime, cliOpts.ScheduledTime)
	}

	created, err := serviceClient.SubmitWorkflow(ctx, &workflowpkg.WorkflowSubmitRequest{
		Namespace:     namespace,
		ResourceKind:  kind,
		ResourceName:  name,
		SubmitOptions: submitOpts,
	})
	if err != nil {
		log.Fatalf("Failed to submit workflow: %v", err)
	}

	printWorkflow(created, common.GetFlags{Output: cliOpts.Output})

	common.WaitWatchOrLog(ctx, serviceClient, namespace, []string{created.Name}, *cliOpts)
}

func submitWorkflows(ctx context.Context, serviceClient workflowpkg.WorkflowServiceClient, namespace string, workflows []wfv1.Workflow, submitOpts *wfv1.SubmitOpts, cliOpts *common.CliSubmitOpts) {
	validateOptions(workflows, submitOpts, cliOpts)

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
		wf.Spec.Priority = cliOpts.Priority
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
			log.Fatalf("Failed to submit workflow: %v", err)
		}

		printWorkflow(created, common.GetFlags{Output: cliOpts.Output, Status: cliOpts.GetArgs.Status})
		workflowNames = append(workflowNames, created.Name)
	}

	common.WaitWatchOrLog(ctx, serviceClient, namespace, workflowNames, *cliOpts)
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
	yamlWfs, err := wfcommon.SplitWorkflowYAMLFile(wfBytes, strict)
	if err == nil {
		return yamlWfs
	}
	log.Fatalf("Failed to parse workflow: %v", err)
	return nil
}
