package workflow

import (
	"context"
	"fmt"

	log "github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/argoproj/argo/persist/sqldb"
	workflowpkg "github.com/argoproj/argo/pkg/apiclient/workflow"
	"github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo/pkg/client/clientset/versioned"
	"github.com/argoproj/argo/workflow/packer"
)

type Watcher interface {
	WatchWorkflows(ctx context.Context, namespace string, opts *metav1.ListOptions, sender WatchEventSender) error
}

type WatchEventSender interface {
	Send(e *workflowpkg.WorkflowWatchEvent) error
}

func NewWatcher(wfClient versioned.Interface, offloadNodeStatusRepo sqldb.OffloadNodeStatusRepo) Watcher {
	return &watcher{wfClient, offloadNodeStatusRepo}
}

type watcher struct {
	wfClient              versioned.Interface
	offloadNodeStatusRepo sqldb.OffloadNodeStatusRepo
}

func (s *watcher) WatchWorkflows(ctx context.Context, namespace string, opts *metav1.ListOptions, sender WatchEventSender) error {
	wfClient := s.wfClient
	if opts == nil {
		opts = &metav1.ListOptions{}
	}
	watch, err := wfClient.ArgoprojV1alpha1().Workflows(namespace).Watch(*opts)
	if err != nil {
		return err
	}
	defer watch.Stop()

	log.Debug("Piping events to channel")

	for next := range watch.ResultChan() {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		gvk := next.Object.GetObjectKind().GroupVersionKind().String()
		logCtx := log.WithFields(log.Fields{"type": next.Type, "objectKind": gvk})
		logCtx.Debug("Received event")
		wf, ok := next.Object.(*v1alpha1.Workflow)
		if !ok {
			return fmt.Errorf("watch object was not a workflow %v", gvk)
		}
		err := packer.DecompressWorkflow(wf)
		if err != nil {
			return err
		}
		if wf.Status.IsOffloadNodeStatus() && s.offloadNodeStatusRepo.IsEnabled() {
			offloadedNodes, err := s.offloadNodeStatusRepo.Get(string(wf.UID), wf.GetOffloadNodeStatusVersion())
			if err != nil {
				return err
			}
			wf.Status.Nodes = offloadedNodes
		}
		logCtx.Debug("Sending event")
		err = sender.Send(&workflowpkg.WorkflowWatchEvent{Type: string(next.Type), Object: wf})
		if err != nil {
			return err
		}
	}

	return nil
}
