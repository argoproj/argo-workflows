package commands

import (
	"context"
	"fmt"
	"os"
	"sync"

	"github.com/argoproj/pkg/errors"
	"github.com/spf13/cobra"
	apierr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"

	"github.com/argoproj/argo/cmd/argo/commands/client"
	"github.com/argoproj/argo/cmd/server/workflow"

	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
)

func NewWaitCommand() *cobra.Command {
	var (
		ignoreNotFound bool
	)
	var command = &cobra.Command{
		Use:   "wait WORKFLOW1 WORKFLOW2..,",
		Short: "waits for a workflow to complete",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) == 0 {
				cmd.HelpFunc()(cmd, args)
				os.Exit(1)
			}

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
	var apiClient workflow.WorkflowServiceClient
	var ctx context.Context
	ns, _, _ := client.Config.Namespace()
	if client.ArgoServer != "" {
		conn := client.GetClientConn()
		defer conn.Close()
		apiClient, ctx = GetWFApiServerGRPCClient(conn)
	} else {
		InitWorkflowClient()
	}

	for _, workflowName := range workflowNames {
		wg.Add(1)
		if client.ArgoServer != "" {
			go func(name string) {
				if !apiServerWaitOnOne(apiClient, ctx, name, ns, ignoreNotFound, quiet) {
					wfSuccessStatus = false
				}
				wg.Done()
			}(workflowName)
		} else {
			go func(name string) {
				if !waitOnOne(name, ignoreNotFound, quiet) {
					wfSuccessStatus = false
				}
				wg.Done()
			}(workflowName)
		}
	}
	wg.Wait()

	if !wfSuccessStatus {
		os.Exit(1)
	}
}

func apiServerWaitOnOne(client workflow.WorkflowServiceClient, ctx context.Context, wfName string, namespace string, ignoreNotFound, quiet bool) bool {
	fieldSelector := fields.ParseSelectorOrDie(fmt.Sprintf("metadata.name=%s", wfName))
	wfReq := workflow.WatchWorkflowsRequest{
		Namespace: namespace,
		ListOptions: &metav1.ListOptions{
			FieldSelector: fieldSelector.String(),
		},
	}
	stream, err := client.WatchWorkflows(ctx, &wfReq)
	if err != nil {
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

func waitOnOne(workflowName string, ignoreNotFound, quiet bool) bool {
	fieldSelector := fields.ParseSelectorOrDie(fmt.Sprintf("metadata.name=%s", workflowName))
	opts := metav1.ListOptions{
		FieldSelector: fieldSelector.String(),
	}

	_, err := wfClient.Get(workflowName, metav1.GetOptions{})
	if err != nil {
		if apierr.IsNotFound(err) && ignoreNotFound {
			return true
		}
		errors.CheckError(err)
	}

	watchIf, err := wfClient.Watch(opts)
	errors.CheckError(err)
	defer watchIf.Stop()
	for {
		next := <-watchIf.ResultChan()
		wf, _ := next.Object.(*wfv1.Workflow)
		if wf == nil {
			watchIf.Stop()
			watchIf, err = wfClient.Watch(opts)
			errors.CheckError(err)
			continue
		}
		if !wf.Status.FinishedAt.IsZero() {
			if !quiet {
				fmt.Printf("%s %s at %v\n", workflowName, wf.Status.Phase, wf.Status.FinishedAt)
			}
			if wf.Status.Phase == wfv1.NodeFailed || wf.Status.Phase == wfv1.NodeError {
				return false
			}
			return true
		}
	}
}
