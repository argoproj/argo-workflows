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

func WatchWorkflow(ctx context.Context, serviceClient workflowpkg.WorkflowServiceClient, namespace string, workflow string, getArgs GetFlags) error {
	req := &workflowpkg.WatchWorkflowsRequest{
		Namespace: namespace,
		ListOptions: &metav1.ListOptions{
			FieldSelector:   util.GenerateFieldSelectorFromWorkflowName(workflow),
			ResourceVersion: "0",
		},
	}
	stream, err := serviceClient.WatchWorkflows(ctx, req)
	if err != nil {
		return err
	}

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
				return nil
			}
			wf = newWf
		case <-ticker.C:
			// If we don't, refresh the workflow screen every second
		case <-ctx.Done():
			// When the context gets canceled
			return nil
		}

		err := printWorkflowStatus(wf, getArgs)
		if err != nil {
			return err
		}
		if wf != nil && !wf.Status.FinishedAt.IsZero() {
			return nil
		}
	}
}

func printWorkflowStatus(wf *wfv1.Workflow, getArgs GetFlags) error {
	if wf == nil {
		return nil
	}
	if err := packer.DecompressWorkflow(wf); err != nil {
		return err
	}
	print("\033[H\033[2J")
	print("\033[0;0H")
	fmt.Print(PrintWorkflowHelper(wf, getArgs))
	return nil
}
