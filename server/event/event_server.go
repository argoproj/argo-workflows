package event

import (
	"context"
	"fmt"
	"sync"

	log "github.com/sirupsen/logrus"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/watch"

	eventpkg "github.com/argoproj/argo/pkg/apiclient/event"
	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo/server/auth"
	"github.com/argoproj/argo/server/event/dispatch"
	"github.com/argoproj/argo/util/instanceid"
	"github.com/argoproj/argo/workflow/events"
)

type Controller struct {
	instanceIDService    instanceid.Service
	eventRecorderManager events.EventRecorderManager
	// a channel for operations to be executed async on
	operationQueue chan dispatch.Operation
	workerCount    int
}

var _ eventpkg.EventServiceServer = &Controller{}

func NewController(instanceIDService instanceid.Service, eventRecorderManager events.EventRecorderManager, operationQueueSize, workerCount int) *Controller {
	log.WithFields(log.Fields{"workerCount": workerCount, "operationQueueSize": operationQueueSize}).Info("Creating event controller")

	return &Controller{
		instanceIDService:    instanceIDService,
		eventRecorderManager: eventRecorderManager,
		//  so we can have `operationQueueSize` operations outstanding before we start putting back pressure on the senders
		operationQueue: make(chan dispatch.Operation, operationQueueSize),
		workerCount:    workerCount,
	}
}

func (s *Controller) Run(stopCh <-chan struct{}) {

	// this `WaitGroup` allows us to wait for all events to dispatch before exiting
	wg := sync.WaitGroup{}

	for w := 0; w < s.workerCount; w++ {
		go func() {
			defer wg.Done()
			for operation := range s.operationQueue {
				operation.Dispatch()
			}
		}()
		wg.Add(1)
	}

	<-stopCh

	// stop accepting new events
	close(s.operationQueue)

	log.WithFields(log.Fields{"operations": len(s.operationQueue)}).Info("Waiting until all remaining events are processed")

	// no more new events, process the existing events
	wg.Wait()
}

func (s *Controller) ReceiveEvent(ctx context.Context, req *eventpkg.EventRequest) (*eventpkg.EventResponse, error) {

	options := metav1.ListOptions{}
	s.instanceIDService.With(&options)

	list, err := auth.GetWfClient(ctx).ArgoprojV1alpha1().WorkflowEventBindings(req.Namespace).List(options)
	if err != nil {
		return nil, err
	}

	operation, err := dispatch.NewOperation(ctx, s.instanceIDService, s.eventRecorderManager.Get(req.Namespace), list.Items, req.Namespace, req.Discriminator, req.Payload)
	if err != nil {
		return nil, err
	}

	select {
	case s.operationQueue <- *operation:
		return &eventpkg.EventResponse{}, nil
	default:
		return nil, apierrors.NewServiceUnavailable("operation queue full")
	}
}

func (s *Controller) ListWorkflowEventBindings(ctx context.Context, in *eventpkg.ListWorkflowEventBindingsRequest) (*wfv1.WorkflowEventBindingList, error) {
	listOptions := metav1.ListOptions{}
	if in.ListOptions != nil {
		listOptions = *in.ListOptions
	}
	return auth.GetWfClient(ctx).ArgoprojV1alpha1().WorkflowEventBindings(in.Namespace).List(listOptions)
}

func (s *Controller) WatchWorkflowEventBindings(in *eventpkg.ListWorkflowEventBindingsRequest, srv eventpkg.EventService_WatchWorkflowEventBindingsServer) error {
	ctx := srv.Context()
	listOptions := metav1.ListOptions{}
	if in.ListOptions != nil {
		listOptions = *in.ListOptions
	}
	workflowEventBindings := auth.GetWfClient(ctx).ArgoprojV1alpha1().WorkflowEventBindings(in.Namespace)
	watcher, err := watch.NewRetryWatcher(listOptions.ResourceVersion, workflowEventBindings)
	if err != nil {
		return err
	}
	for {
		select {
		case <-ctx.Done():
			return nil
		case event, ok := <-watcher.ResultChan():
			if !ok {
				return fmt.Errorf("failed to read event")
			}
			obj, ok := event.Object.(*wfv1.WorkflowEventBinding)
			if !ok {
				return apierrors.FromObject(event.Object)
			}
			err := srv.Send(&eventpkg.WorkflowEventBindingWatchEvent{Type: string(event.Type), Object: obj})
			if err != nil {
				return err
			}
		}
	}
}
