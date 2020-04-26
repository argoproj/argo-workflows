package commands

import (
	"context"
	"fmt"
	"os"
	"sync"

	"github.com/argoproj/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	apierr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"

	"github.com/argoproj/argo/cmd/argo/commands/client"
	workflowpkg "github.com/argoproj/argo/pkg/apiclient/workflow"
	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
)

func NewWaitCommand() *cobra.Command {
	var (
		ignoreNotFound bool
	)
	var command = &cobra.Command{
		Use:   "wait [WORKFLOW...]",
		Short: "waits for workflows to complete",
		Run: func(cmd *cobra.Command, args []string) {
			WaitWorkflows(args, ignoreNotFound, false)
		},
	}
	command.Flags().BoolVar(&ignoreNotFound, "ignore-not-found", false, "Ignore the wait if the workflow is not found")
	return command
}

// WaitWorkflows waits for the given workflowNames.
func WaitWorkflows(workflowNames []string, ignoreNotFound, quiet bool) {
	var wg sync.WaitGroup
	wfSuccessStatus := true

	ctx, apiClient := client.NewAPIClient()
	serviceClient, err := apiClient.NewWorkflowServiceClient()
	errors.CheckError(err)
	namespace := client.Namespace()

	for _, name := range workflowNames {
		wg.Add(1)
		go func(name string) {
			if !waitOnOne(serviceClient, ctx, name, namespace, ignoreNotFound, quiet) {
				wfSuccessStatus = false
			}
			wg.Done()
		}(name)

	}
	wg.Wait()

	if !wfSuccessStatus {
		os.Exit(1)
	}
}

func waitOnOne(serviceClient workflowpkg.WorkflowServiceClient, ctx context.Context, wfName, namespace string, ignoreNotFound, quiet bool) bool {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	req := &workflowpkg.WatchWorkflowsRequest{
		Namespace: namespace,
		ListOptions: &metav1.ListOptions{
			FieldSelector: fields.ParseSelectorOrDie(fmt.Sprintf("metadata.name=%s", wfName)).String(),
		},
	}
	stream, err := serviceClient.WatchWorkflows(ctx, req)
	if err != nil {
		if apierr.IsNotFound(err) && ignoreNotFound {
			return true
		}
		errors.CheckError(err)
		return false
	}
	for {
		event, err := stream.Recv()
		if err != nil {
			errors.CheckError(err)
			break
		}
		wf := event.Object
		if wf == nil {
			log.Debug("Re-establishing workflow watch")
			stream, err = serviceClient.WatchWorkflows(ctx, req)
			if err != nil {
				errors.CheckError(err)
				return false
			}
			continue
		}
		if !wf.Status.FinishedAt.IsZero() {
			if !quiet {
				fmt.Printf("%s %s at %v\n", wfName, wf.Status.Phase, wf.Status.FinishedAt)
			}
			if wf.Status.Phase == wfv1.NodeFailed || wf.Status.Phase == wfv1.NodeError {
				return false
			}
			return true
		}
	}
	return true
}
