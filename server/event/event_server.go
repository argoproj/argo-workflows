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
	// use of shared informers allows us to avoid dealing with errors in `ReceiveEvent`
	workflowTemplateController cache.Controller
	workflowTemplateKeyLister  cache.KeyLister
	// a channel for operations to be executed async on
	operationPipeline chan dispatch.Operation
	workerCount       int
}

var _ eventpkg.EventServiceServer = &Controller{}

func NewController(client *versioned.Clientset, namespace string, instanceIDService instanceid.Service, pipelineSize, workerCount int) *Controller {
	restClient := client.ArgoprojV1alpha1().RESTClient()
	idRequirement := util.InstanceIDRequirement(instanceIDService.InstanceID())

	workflowTemplateController, workflowTemplateKeyLister := eventcache.NewFilterUsingKeyController(restClient, namespace, labels.NewSelector().Add(idRequirement), "workflowtemplates", &wfv1.WorkflowTemplate{}, func(d cache.Delta) bool {
		return d.Object.(*wfv1.WorkflowTemplate).Spec.Event != nil
	})

	return &Controller{
		workflowTemplateController: workflowTemplateController,
		workflowTemplateKeyLister:  workflowTemplateKeyLister,
		//  so we can have N operations outstanding before we start putting back pressure on the senders
		operationPipeline: make(chan dispatch.Operation, pipelineSize),
		workerCount:       workerCount,
	}
}

func (s *Controller) Run(stopCh <-chan struct{}) {

	go s.workflowTemplateController.Run(stopCh)

	for _, c := range []cache.Controller{s.workflowTemplateController} {
		err := wait.PollUntil(1*time.Second, func() (done bool, err error) { return c.HasSynced(), nil }, stopCh)
		if err != nil {
			log.WithError(err).Error("failed to sync controller")
		}
	}

	// this block of code waits for all events to be processed
	wg := sync.WaitGroup{}

	for w := 0; w <= s.workerCount; w++ {
		go func() {
			defer wg.Done()
			for operation := range s.operationPipeline {
				operation.Execute()
			}
		}()
		wg.Add(1)
	}

	<-stopCh

	// stop accepting new events
	close(s.operationPipeline)

	// no more new events, process the existing events
	wg.Wait()
}

func (s *Controller) ReceiveEvent(ctx context.Context, req *eventpkg.EventRequest) (*eventpkg.EventResponse, error) {
	select {
	case s.operationPipeline <- dispatch.NewOperation(ctx, s.workflowTemplateKeyLister, req.Event):
		return &eventpkg.EventResponse{}, nil
	default:
		return nil, errors.NewServiceUnavailable("operation pipeline full")
	}
}
