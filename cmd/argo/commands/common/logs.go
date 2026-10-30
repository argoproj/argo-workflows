package common

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"slices"

	corev1 "k8s.io/api/core/v1"

	workflowpkg "github.com/argoproj/argo-workflows/v4/pkg/apiclient/workflow"
	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/util/logging"
	"github.com/argoproj/argo-workflows/v4/workflow/util"
)

func LogWorkflow(ctx context.Context, serviceClient workflowpkg.WorkflowServiceClient, namespace, workflow, podName, grep, selector string, logOptions *corev1.PodLogOptions) error {
	// logs
	stream, err := serviceClient.WorkflowLogs(ctx, &workflowpkg.WorkflowLogRequest{
		Name:       workflow,
		Namespace:  namespace,
		PodName:    podName,
		LogOptions: logOptions,
		Selector:   selector,
		Grep:       grep,
	})
	if err != nil {
		return err
	}

	// loop on log lines
	for {
		event, recvErr := stream.Recv()
		if errors.Is(recvErr, io.EOF) {
			break
		}
		if recvErr != nil {
			return recvErr
		}
		fmt.Println(ansiFormat(fmt.Sprintf("%s: %s", event.PodName, event.Content), ansiColorCode(event.PodName)))
	}

	// A pooled task is in no pod the stream reads, whether picked by name or
	// by selector, so say where its logs are instead.
	wf, err := serviceClient.GetWorkflow(ctx, &workflowpkg.WorkflowGetRequest{
		Name:      workflow,
		Namespace: namespace,
		Fields:    "metadata.name,metadata.namespace,status.nodes",
	})
	// The stream itself succeeded, so a failed lookup (the workflow was
	// deleted, or the caller may not get it) only loses the notes; a POD that
	// named only pooled tasks then can't be told from one that matched nothing.
	if err != nil {
		logging.RequireLoggerFromContext(ctx).WithError(err).Debug(ctx, "Could not look up pooled tasks for their log notes")
		return nil
	}
	notes := pooledLogNotes(wf, podName, logOptions.Container)
	for _, note := range notes {
		_, _ = fmt.Fprintln(os.Stderr, note)
	}
	// A task or worker pod name never also names a workflow pod, so a POD
	// that matched pooled nodes matched nothing the stream could show.
	if podName != "" && len(notes) > 0 {
		return fmt.Errorf("%s names only pooled tasks, which have no live logs", podName)
	}
	return nil
}

// pooledLogNotes says where to find the logs of the workflow's pooled nodes
// that podName, if set, names by task name or worker pod. A pooled task runs
// on a worker pod shared with other tasks and carrying no workflow label, so
// the log stream never includes it; only its archived logs are available.
func pooledLogNotes(wf *wfv1.Workflow, podName, container string) []string {
	var nodes []wfv1.NodeStatus
	for _, node := range wf.Status.Nodes {
		task := node.PooledTask
		if task == nil || (podName != "" && podName != task.TaskName && podName != task.WorkerPodName) {
			continue
		}
		nodes = append(nodes, node)
	}
	slices.SortFunc(nodes, func(a, b wfv1.NodeStatus) int {
		return cmp.Or(a.StartedAt.Compare(b.StartedAt.Time), cmp.Compare(a.ID, b.ID))
	})
	artifactName := container + "-logs"
	notes := make([]string, 0, len(nodes))
	for _, node := range nodes {
		name, _ := util.PooledTaskPodName(node.PooledTask, node.Fulfilled())
		var where string
		switch {
		case node.Outputs.GetArtifactByName(artifactName) != nil:
			where = fmt.Sprintf("fetch its archived logs with `argo cp -n %s %s . --node-id %s --artifact-name %s`", wf.Namespace, wf.Name, node.ID, artifactName)
		case !node.Fulfilled():
			where = "its logs are archived when it finishes, if log archiving is enabled"
		default:
			where = "it has no archived logs"
		}
		notes = append(notes, fmt.Sprintf("%s: %s is a pooled task with no live logs; %s", name, node.DisplayName, where))
	}
	return notes
}
