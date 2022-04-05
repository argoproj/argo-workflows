package common

import (
	"context"
	"fmt"
	"io"

	"github.com/argoproj/pkg/errors"
	corev1 "k8s.io/api/core/v1"

	workflowpkg "github.com/argoproj/argo-workflows/v3/pkg/apiclient/workflow"
)

func LogWorkflow(ctx context.Context, serviceClient workflowpkg.WorkflowServiceClient, namespace, workflow, podName, grep, selector string, logOptions *corev1.PodLogOptions) {
	// logs
	stream, err := serviceClient.WorkflowLogs(ctx, &workflowpkg.WorkflowLogRequest{
		Name:       workflow,
		Namespace:  namespace,
		PodName:    podName,
		LogOptions: logOptions,
		Selector:   selector,
		Grep:       grep,
	})
	errors.CheckError(err)

	// loop on log lines
	for {
		event, err := stream.Recv()
		if err == io.EOF {
			return
		}
		errors.CheckError(err)
		fmt.Println(ansiFormat(fmt.Sprintf("%s: %s", event.PodName, event.Content), ansiColorCode(event.PodName)))
	}
}
