package event

import (
	"context"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/cache"

	eventpkg "github.com/argoproj/argo/pkg/apiclient/event"
	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo/pkg/client/clientset/versioned"
	eventcache "github.com/argoproj/argo/server/event/cache"
	"github.com/argoproj/argo/server/event/dispatch"
	"github.com/argoproj/argo/util/instanceid"
	"github.com/argoproj/argo/workflow/util"
)

type Controller struct {
	instanceIDService instanceid.Service
	// use of shared informers allows us to avoid dealing with errors in `ReceiveEvent`
	workflowTemplateController cache.Controller
	workflowTemplateKeyLister  cache.KeyLister
	// a channel for operations to be executed async on
	operationQueue chan dispatch.Operation
	workerCount    int
}

var _ eventpkg.EventServiceServer = &Controller{}

func NewController(client versioned.Interface, namespace string, instanceIDService instanceid.Service, operationQueueSize, workerCount int) *Controller {
	restClient := client.ArgoprojV1alpha1().RESTClient()
	instanceIDRequirement := util.InstanceIDRequirement(instanceIDService.InstanceID())

	workflowTemplateController, workflowTemplateKeyLister := eventcache.NewFilterUsingKeyController(restClient, namespace, labels.NewSelector().Add(instanceIDRequirement), "workflowtemplates", &wfv1.WorkflowTemplate{}, func(d cache.Delta) bool {
		// no `ok` check here because a panic would indicate a bug
		return len(d.Object.(*wfv1.WorkflowTemplate).Spec.Events) > 0
	})

	log.WithFields(log.Fields{"workerCount": workerCount, "operationQueueSize": operationQueueSize}).Info("Creating event controller")

	return &Controller{
		instanceIDService:          instanceIDService,
		workflowTemplateController: workflowTemplateController,
		workflowTemplateKeyLister:  workflowTemplateKeyLister,
		//  so we can have `operationQueueSize` operations outstanding before we start putting back pressure on the senders
		operationQueue: make(chan dispatch.Operation, operationQueueSize),
		workerCount:    workerCount,
	}
}

func (s *Controller) Run(stopCh <-chan struct{}) {

	go s.workflowTemplateController.Run(stopCh)

	for _, c := range []cache.Controller{s.workflowTemplateController} {
		// 30s wait to avoid start up which is unable to dispatch events
		err := wait.PollUntil(30*time.Second, func() (done bool, err error) { return c.HasSynced(), nil }, stopCh)
		if err != nil {
			log.WithError(err).Error("Failed to sync controller")
		}
	}

	log.WithFields(log.Fields{"workflowTemplateCount": len(s.workflowTemplateKeyLister.ListKeys())}).Info("Running event controller")

	s.processEvents(stopCh)
}

func (s *Controller) processEvents(stopCh <-chan struct{}) {
	// this `WaitGroup` allows us to wait for all events to dispatch before exiting
	wg := sync.WaitGroup{}

	for w := 0; w < s.workerCount; w++ {
		go func() {
			defer wg.Done()
			for operation := range s.operationQueue {
				operation.Execute()
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
	select {
	case s.operationQueue <- dispatch.NewOperation(ctx, s.instanceIDService, s.workflowTemplateKeyLister, req.Discriminator, req.Payload):
		return &eventpkg.EventResponse{}, nil
	default:
		return nil, errors.NewServiceUnavailable("operation pipeline full")
	}
}
