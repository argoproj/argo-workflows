package event

import (
	"context"
	"sync"

	"google.golang.org/grpc/codes"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	eventpkg "github.com/argoproj/argo-workflows/v4/pkg/apiclient/event"
	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/server/auth"
	"github.com/argoproj/argo-workflows/v4/server/event/dispatch"
	"github.com/argoproj/argo-workflows/v4/util/instanceid"
	"github.com/argoproj/argo-workflows/v4/util/logging"
	"github.com/argoproj/argo-workflows/v4/workflow/events"

	sutils "github.com/argoproj/argo-workflows/v4/server/utils"
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

func NewController(ctx context.Context, instanceIDService instanceid.Service, eventRecorderManager events.EventRecorderManager, operationQueueSize, workerCount int, asyncDispatch bool) *Controller {
	logger := logging.RequireLoggerFromContext(ctx)
	logger.WithFields(logging.Fields{"workerCount": workerCount, "operationQueueSize": operationQueueSize, "asyncDispatch": asyncDispatch}).Info(ctx, "Creating event controller")

	return &Controller{
		instanceIDService:    instanceIDService,
		eventRecorderManager: eventRecorderManager,
		//  so we can have `operationQueueSize` operations outstanding before we start putting back pressure on the senders
		operationQueue: make(chan dispatch.Operation, operationQueueSize),
		workerCount:    workerCount,
		asyncDispatch:  asyncDispatch,
	}
}

//nolint:contextcheck
func (s *Controller) Run(ctx context.Context, stopCh <-chan struct{}) {
	// this `WaitGroup` allows us to wait for all events to dispatch before exiting
	wg := sync.WaitGroup{}
	logger := logging.RequireLoggerFromContext(ctx)

	for w := 0; w < s.workerCount; w++ {
		wg.Go(func() {
			for operation := range s.operationQueue {
				ctx := operation.Context()
				_ = operation.Dispatch(ctx)
			}
		})
	}

	<-stopCh

	// stop accepting new events
	close(s.operationQueue)

	logger.WithFields(logging.Fields{"operations": len(s.operationQueue)}).Info(ctx, "Waiting until all remaining events are processed")

	// no more new events, process the existing events
	wg.Wait()
}

func (s *Controller) ReceiveEvent(ctx context.Context, req *eventpkg.EventRequest) (*eventpkg.EventResponse, error) {
	options := metav1.ListOptions{}
	s.instanceIDService.With(&options)

	list, err := auth.GetWfClient(ctx).ArgoprojV1alpha1().WorkflowEventBindings(req.Namespace).List(ctx, options)
	if err != nil {
		return nil, sutils.ToStatusError(err, codes.Internal)
	}

	operation, err := dispatch.NewOperation(ctx, s.instanceIDService, s.eventRecorderManager.Get(ctx, req.Namespace), list.Items, req.Namespace, req.Discriminator, req.Payload)
	if err != nil {
		return nil, sutils.ToStatusError(err, codes.Internal)
	}

	if !s.asyncDispatch {
		if err := operation.Dispatch(ctx); err != nil {
			return nil, sutils.ToStatusError(err, codes.Internal)
		}
		return &eventpkg.EventResponse{}, nil
	}

	select {
	case s.operationQueue <- *operation:
		return &eventpkg.EventResponse{}, nil
	default:
		return nil, sutils.ToStatusError(apierrors.NewServiceUnavailable("operation queue full"), codes.ResourceExhausted)
	}
}

func (s *Controller) ListWorkflowEventBindings(ctx context.Context, in *eventpkg.ListWorkflowEventBindingsRequest) (*wfv1.WorkflowEventBindingList, error) {
	listOptions := metav1.ListOptions{}
	if in.ListOptions != nil {
		listOptions = *in.ListOptions
	}
	eventBindings, err := auth.GetWfClient(ctx).ArgoprojV1alpha1().WorkflowEventBindings(in.Namespace).List(ctx, listOptions)
	if err != nil {
		return nil, sutils.ToStatusError(err, codes.Internal)
	}
	return eventBindings, nil
}
