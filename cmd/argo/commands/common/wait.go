package common

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"sync"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	workflowpkg "github.com/argoproj/argo-workflows/v4/pkg/apiclient/workflow"
	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/util"
	"github.com/argoproj/argo-workflows/v4/util/logging"
)

// waitWorkflows waits for the given workflowNames.
func WaitWorkflows(ctx context.Context, serviceClient workflowpkg.WorkflowServiceClient, namespace string, workflowNames []string, ignoreNotFound, quiet bool) {
	var wg sync.WaitGroup
	wfSuccessStatus := true

	for _, name := range workflowNames {
		wg.Go(func() {
			if ok, _ := waitOnOne(ctx, serviceClient, name, namespace, ignoreNotFound, quiet); !ok {
				wfSuccessStatus = false
			}
		})
	}
	wg.Wait()

	if !wfSuccessStatus {
		os.Exit(1)
	}
}

func waitOnOne(ctx context.Context, serviceClient workflowpkg.WorkflowServiceClient, wfName, namespace string, ignoreNotFound, quiet bool) (bool, error) {
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
		return false, nil
	}
	for {
		event, err := stream.Recv()
		if errors.Is(err, io.EOF) {
			logger := logging.RequireLoggerFromContext(ctx)
			logger.Debug(ctx, "Re-establishing workflow watch")
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
