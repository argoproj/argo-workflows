package event

import (
	"context"
	"sync"

	log "github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	eventpkg "github.com/argoproj/argo/pkg/apiclient/event"
	"github.com/argoproj/argo/server/auth"
	"github.com/argoproj/argo/server/event/dispatch"
	"github.com/argoproj/argo/util/instanceid"
)

type Controller struct {
	instanceIDService instanceid.Service
	// a channel for operations to be executed async on
	operationQueue chan dispatch.Operation
	workerCount    int
}

var _ eventpkg.EventServiceServer = &Controller{}

func NewController(instanceIDService instanceid.Service, operationQueueSize, workerCount int) *Controller {
	log.WithFields(log.Fields{"workerCount": workerCount, "operationQueueSize": operationQueueSize}).Info("Creating event controller")

	return &Controller{
		instanceIDService: instanceIDService,
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

	operation, err := dispatch.NewOperation(ctx, s.instanceIDService, list.Items, req.Namespace, req.Discriminator, req.Payload)
	if err != nil {
		return nil, err
	}

	select {
	case s.operationQueue <- *operation:
		return &eventpkg.EventResponse{}, nil
	default:
		return nil, errors.NewServiceUnavailable("operation queue full")
	}
}
