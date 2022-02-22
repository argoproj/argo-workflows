package event

import (
	"context"
	"sync"

	log "github.com/sirupsen/logrus"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	eventpkg "github.com/argoproj/argo-workflows/v3/pkg/apiclient/event"
	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v3/server/auth"
	"github.com/argoproj/argo-workflows/v3/server/event/dispatch"
	"github.com/argoproj/argo-workflows/v3/util/instanceid"
	"github.com/argoproj/argo-workflows/v3/workflow/events"
)

type Controller struct {
	instanceIDService    instanceid.Service
	eventRecorderManager events.EventRecorderManager
	// a channel for operations to be executed async on
	operationQueue chan dispatch.Operation
	workerCount    int
	asyncDispatch  bool
}

var _ eventpkg.EventServiceServer = &Controller{}

func NewController(instanceIDService instanceid.Service, eventRecorderManager events.EventRecorderManager, operationQueueSize, workerCount int, asyncDispatch bool) *Controller {
	log.WithFields(log.Fields{"workerCount": workerCount, "operationQueueSize": operationQueueSize, "asyncDispatch": asyncDispatch}).Info("Creating event controller")

	return &Controller{
		instanceIDService:    instanceIDService,
		eventRecorderManager: eventRecorderManager,
		//  so we can have `operationQueueSize` operations outstanding before we start putting back pressure on the senders
		operationQueue: make(chan dispatch.Operation, operationQueueSize),
		workerCount:    workerCount,
		asyncDispatch:  asyncDispatch,
	}
}

func (s *Controller) Run(stopCh <-chan struct{}) {
	// this `WaitGroup` allows us to wait for all events to dispatch before exiting
	wg := sync.WaitGroup{}

	for w := 0; w < s.workerCount; w++ {
		go func() {
			defer wg.Done()
			for operation := range s.operationQueue {
				_ = operation.Dispatch(context.Background())
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

	list, err := auth.GetWfClient(ctx).ArgoprojV1alpha1().WorkflowEventBindings(req.Namespace).List(ctx, options)
	if err != nil {
		return nil, err
	}

	operation, err := dispatch.NewOperation(ctx, s.instanceIDService, s.eventRecorderManager.Get(req.Namespace), list.Items, req.Namespace, req.Discriminator, req.Payload)
	if err != nil {
		return nil, err
	}

	if !s.asyncDispatch {
		if err := operation.Dispatch(ctx); err != nil {
			return nil, err
		}
		return &eventpkg.EventResponse{}, nil
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
	return auth.GetWfClient(ctx).ArgoprojV1alpha1().WorkflowEventBindings(in.Namespace).List(ctx, listOptions)
}
