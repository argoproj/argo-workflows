package common

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/argoproj/pkg/errors"
	log "github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	workflowpkg "github.com/argoproj/argo-workflows/v3/pkg/apiclient/workflow"
	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v3/util"
	"github.com/argoproj/argo-workflows/v3/workflow/packer"
)

func WatchWorkflow(ctx context.Context, serviceClient workflowpkg.WorkflowServiceClient, namespace string, workflow string, getArgs GetFlags) {
	req := &workflowpkg.WatchWorkflowsRequest{
		Namespace: namespace,
		ListOptions: &metav1.ListOptions{
			FieldSelector:   util.GenerateFieldSelectorFromWorkflowName(workflow),
			ResourceVersion: "0",
		},
	}
	stream, err := serviceClient.WatchWorkflows(ctx, req)
	errors.CheckError(err)

	wfChan := make(chan *wfv1.Workflow)
	go func() {
		for {
			event, err := stream.Recv()
			if err == io.EOF {
				log.Debug("Re-establishing workflow watch")
				stream, err = serviceClient.WatchWorkflows(ctx, req)
				errors.CheckError(err)
				continue
			}
			errors.CheckError(err)
			if event == nil {
				continue
			}
			wfChan <- event.Object
		}
	}()

	var wf *wfv1.Workflow
	ticker := time.NewTicker(time.Second)
	for {
		select {
		case newWf := <-wfChan:
			// If we get a new event, update our workflow
			if newWf == nil {
				return
			}
			wf = newWf
		case <-ticker.C:
			// If we don't, refresh the workflow screen every second
		case <-ctx.Done():
			// When the context gets canceled
			return
		}

		printWorkflowStatus(wf, getArgs)
		if wf != nil && !wf.Status.FinishedAt.IsZero() {
			return
		}
	}
}

func printWorkflowStatus(wf *wfv1.Workflow, getArgs GetFlags) {
	if wf == nil {
		return
	}
	err := packer.DecompressWorkflow(wf)
	errors.CheckError(err)
	print("\033[H\033[2J")
	print("\033[0;0H")
	fmt.Print(PrintWorkflowHelper(wf, getArgs))
}
