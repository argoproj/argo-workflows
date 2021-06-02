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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/argoproj/argo-workflows/v3/cmd/argo/commands/client"
	workflowpkg "github.com/argoproj/argo-workflows/v3/pkg/apiclient/workflow"
	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v3/workflow/common"
	"github.com/argoproj/argo-workflows/v3/workflow/util"
)

// cliSubmitOpts holds submission options specific to CLI submission (e.g. controlling output)
type cliSubmitOpts struct {
	output        string // --output
	wait          bool   // --wait
	watch         bool   // --watch
	verify        bool   // --verify
	log           bool   // --log
	strict        bool   // --strict
	priority      *int32 // --priority
	getArgs       getFlags
	scheduledTime string // --scheduled-time
}

func NewSubmitCommand() *cobra.Command {
	var (
		submitOpts    wfv1.SubmitOpts
		cliSubmitOpts cliSubmitOpts
		priority      int32
		from          string
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
				cliSubmitOpts.priority = &priority
			}

			if !cliSubmitOpts.watch && len(cliSubmitOpts.getArgs.status) > 0 {
				log.Warn("--status should only be used with --watch")
			}

			ctx, apiClient := client.NewAPIClient()
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
	util.PopulateSubmitOpts(command, &submitOpts, true)
	command.Flags().StringVarP(&cliSubmitOpts.output, "output", "o", "", "Output format. One of: name|json|yaml|wide")
	command.Flags().BoolVarP(&cliSubmitOpts.wait, "wait", "w", false, "wait for the workflow to complete")
	command.Flags().BoolVar(&cliSubmitOpts.watch, "watch", false, "watch the workflow until it completes")
	command.Flags().BoolVar(&cliSubmitOpts.verify, "verify", false, "verify completed workflows by running the Python code in the workflows.argoproj.io/verify.py annotation")
	errors.CheckError(command.Flags().MarkHidden("verify"))
	command.Flags().BoolVar(&cliSubmitOpts.log, "log", false, "log the workflow until it completes")
	command.Flags().BoolVar(&cliSubmitOpts.strict, "strict", true, "perform strict workflow validation")
	command.Flags().Int32Var(&priority, "priority", 0, "workflow priority")
	command.Flags().StringVar(&from, "from", "", "Submit from an existing `kind/name` E.g., --from=cronwf/hello-world-cwf")
	command.Flags().StringVar(&cliSubmitOpts.getArgs.status, "status", "", "Filter by status (Pending, Running, Succeeded, Skipped, Failed, Error). Should only be used with --watch.")
	command.Flags().StringVar(&cliSubmitOpts.getArgs.nodeFieldSelectorString, "node-field-selector", "", "selector of node to display, eg: --node-field-selector phase=abc")
	command.Flags().StringVar(&cliSubmitOpts.scheduledTime, "scheduled-time", "", "Override the workflow's scheduledTime parameter (useful for backfilling). The time must be RFC3339")

	// Only complete files with appropriate extension.
	err := command.Flags().SetAnnotation("parameter-file", cobra.BashCompFilenameExt, []string{"json", "yaml", "yml"})
	if err != nil {
		log.Fatal(err)
	}
	return command
}

func submitWorkflowsFromFile(ctx context.Context, serviceClient workflowpkg.WorkflowServiceClient, namespace string, filePaths []string, submitOpts *wfv1.SubmitOpts, cliOpts *cliSubmitOpts) {
	fileContents, err := util.ReadManifest(filePaths...)
	errors.CheckError(err)

	var workflows []wfv1.Workflow
	for _, body := range fileContents {
		wfs := unmarshalWorkflows(body, cliOpts.strict)
		workflows = append(workflows, wfs...)
	}

	submitWorkflows(ctx, serviceClient, namespace, workflows, submitOpts, cliOpts)
}

func validateOptions(workflows []wfv1.Workflow, submitOpts *wfv1.SubmitOpts, cliOpts *cliSubmitOpts) {
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
}

func submitWorkflowFromResource(ctx context.Context, serviceClient workflowpkg.WorkflowServiceClient, namespace string, resourceIdentifier string, submitOpts *wfv1.SubmitOpts, cliOpts *cliSubmitOpts) {
	parts := strings.SplitN(resourceIdentifier, "/", 2)
	if len(parts) != 2 {
		log.Fatalf("resource identifier '%s' is malformed. Should be `kind/name`, e.g. cronwf/hello-world-cwf", resourceIdentifier)
	}
	kind := parts[0]
	name := parts[1]

	tempwf := wfv1.Workflow{}

	validateOptions([]wfv1.Workflow{tempwf}, submitOpts, cliOpts)
	if cliOpts.scheduledTime != "" {
		_, err := time.Parse(time.RFC3339, cliOpts.scheduledTime)
		if err != nil {
			log.Fatalf("scheduled-time contains invalid time.RFC3339 format. (e.g.: `2006-01-02T15:04:05-07:00`)")
		}
		submitOpts.Annotations = fmt.Sprintf("%s=%s", common.AnnotationKeyCronWfScheduledTime, cliOpts.scheduledTime)
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

	printWorkflow(created, getFlags{output: cliOpts.output})

	waitWatchOrLog(ctx, serviceClient, namespace, []string{created.Name}, *cliOpts)
}

func submitWorkflows(ctx context.Context, serviceClient workflowpkg.WorkflowServiceClient, namespace string, workflows []wfv1.Workflow, submitOpts *wfv1.SubmitOpts, cliOpts *cliSubmitOpts) {
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
			log.Fatalf("Failed to submit workflow: %v", err)
		}

		printWorkflow(created, getFlags{output: cliOpts.output, status: cliOpts.getArgs.status})
		workflowNames = append(workflowNames, created.Name)
	}

	waitWatchOrLog(ctx, serviceClient, namespace, workflowNames, *cliOpts)
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

func waitWatchOrLog(ctx context.Context, serviceClient workflowpkg.WorkflowServiceClient, namespace string, workflowNames []string, cliSubmitOpts cliSubmitOpts) {
	if cliSubmitOpts.log {
		for _, workflow := range workflowNames {
			logWorkflow(ctx, serviceClient, namespace, workflow, "", "", &corev1.PodLogOptions{
				Container: common.MainContainerName,
				Follow:    true,
				Previous:  false,
			})
		}
	}
	if cliSubmitOpts.wait {
		waitWorkflows(ctx, serviceClient, namespace, workflowNames, false, !(cliSubmitOpts.output == "" || cliSubmitOpts.output == "wide"))
	} else if cliSubmitOpts.watch {
		for _, workflow := range workflowNames {
			watchWorkflow(ctx, serviceClient, namespace, workflow, cliSubmitOpts.getArgs)
		}
	}
	if cliSubmitOpts.verify {
		verifyWorkflows(ctx, serviceClient, namespace, workflowNames)
	}
}
