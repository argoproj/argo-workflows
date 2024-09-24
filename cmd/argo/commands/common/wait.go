package common

import (
	"context"
	"fmt"
	"io"
	"os"
	"sync"

	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	workflowpkg "github.com/argoproj/argo-workflows/v3/pkg/apiclient/workflow"
	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v3/util"
)

// waitWorkflows waits for the given workflowNames.
func WaitWorkflows(ctx context.Context, serviceClient workflowpkg.WorkflowServiceClient, namespace string, workflowNames []string, ignoreNotFound, quiet bool) {
	var wg sync.WaitGroup
	wfSuccessStatus := true

	for _, name := range workflowNames {
		wg.Add(1)
		go func(name string) {
			ok, err := waitOnOne(serviceClient, ctx, name, namespace, ignoreNotFound, quiet)
			if !ok || err != nil {
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

func waitOnOne(serviceClient workflowpkg.WorkflowServiceClient, ctx context.Context, wfName, namespace string, ignoreNotFound, quiet bool) (bool, error) {
	req := &workflowpkg.WatchWorkflowsRequest{
		Namespace: namespace,
		ListOptions: &metav1.ListOptions{
			FieldSelector:   util.GenerateFieldSelectorFromWorkflowName(wfName),
			ResourceVersion: "0",
		},
	}
	stream, err := serviceClient.WatchWorkflows(ctx, req)
	if err != nil {
		if status.Code(err) == codes.NotFound && ignoreNotFound {
			return true, nil
		}
		if err != nil {
			return false, err
		}
		return false, nil
	}
	for {
		event, err := stream.Recv()
		if err == io.EOF {
			log.Debug("Re-establishing workflow watch")
			stream, err = serviceClient.WatchWorkflows(ctx, req)
			if err != nil {
				return false, err
			}
			continue
		}
		if err != nil {
			return false, err
		}
		if event == nil {
			continue
		}
		wf := event.Object
		if wf != nil && !wf.Status.FinishedAt.IsZero() {
			if !quiet {
				fmt.Printf("%s %s at %v\n", wfName, wf.Status.Phase, wf.Status.FinishedAt)
			}
			if wf.Status.Phase == wfv1.WorkflowFailed || wf.Status.Phase == wfv1.WorkflowError {
				return false, nil
			}
			return true, nil
		}
	}
}
