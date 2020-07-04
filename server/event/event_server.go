package event

import (
	"context"
	"time"

	log "github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/client-go/tools/cache"

	eventpkg "github.com/argoproj/argo/pkg/apiclient/event"
	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo/pkg/client/clientset/versioned"
	"github.com/argoproj/argo/pkg/client/informers/externalversions/workflow/v1alpha1"
	"github.com/argoproj/argo/server/event/dispatch"
	"github.com/argoproj/argo/server/event/keys"
	"github.com/argoproj/argo/util/instanceid"
	"github.com/argoproj/argo/workflow/common"
	"github.com/argoproj/argo/workflow/hydrator"
	"github.com/argoproj/argo/workflow/util"
)

type Controller struct {
	// use of shared informers allows us to avoid dealing with errors in `ReceiveEvent`
	workflowInformer         cache.SharedIndexInformer
	workflowTemplateInformer cache.SharedIndexInformer
	hydrator                 hydrator.Interface
	instanceIDService        instanceid.Service
	// the meta namespace keys of any workflow that can accept events
	workflowKeys *keys.Keys
	// the meta namespace keys of any workflow template that can accept events
	templateKeys *keys.Keys
	// a channel for operations to be executed async on
	operationPipeline chan dispatch.Operation
}

var _ eventpkg.EventServiceServer = &Controller{}

func NewController(client *versioned.Clientset, namespace string, instanceService instanceid.Service, hydrator hydrator.Interface) *Controller {
	return &Controller{
		workflowInformer: v1alpha1.NewFilteredWorkflowInformer(client, namespace, 20*time.Second, cache.Indexers{}, func(options *metav1.ListOptions) {
			incomplete, _ := labels.NewRequirement(common.LabelKeyCompleted, selection.NotEquals, []string{"true"})
			options.LabelSelector = labels.NewSelector().
				Add(*incomplete).
				Add(util.InstanceIDRequirement(instanceService.InstanceID())).
				String()
		}),
		workflowTemplateInformer: v1alpha1.NewFilteredWorkflowTemplateInformer(client, namespace, 20*time.Second, cache.Indexers{}, func(options *metav1.ListOptions) {
			options.LabelSelector = labels.NewSelector().
				Add(util.InstanceIDRequirement(instanceService.InstanceID())).
				String()
		}),
		workflowKeys:      keys.New(),
		templateKeys:      keys.New(),
		instanceIDService: instanceService,
		hydrator:          hydrator,
		// 64 length - so we can have 64 operations outstanding before we start putting back pressure on the senders
		operationPipeline: make(chan dispatch.Operation, 64),
	}
}

func (s *Controller) Run(stopCh <-chan struct{}) {
	s.workflowInformer.AddEventHandler(cache.FilteringResourceEventHandler{
		// we're only interested in incomplete workflows that have running suspend nodes
		FilterFunc: func(obj interface{}) bool {
			wf, ok := obj.(*wfv1.Workflow)
			// don't expect ok to be false here, but better to check rather than panic
			if !ok {
				return false
			}
			err := s.hydrator.Hydrate(wf)
			if err != nil {
				log.WithFields(log.Fields{"namespace": wf.Namespace, "workflow": wf.Name}).WithError(err).Error("failed to hydrate workflow")
				return false
			}
			for _, node := range wf.Status.Nodes {
				if node.Type == wfv1.NodeTypeSuspend {
					// don't expect the template to be nil here, but better to check than panic
					if t := wf.GetTemplateByName(node.TemplateName); t != nil && t.Suspend != nil && t.Suspend.Event != nil {
						return true
					}
				}
			}
			return false
		},
		Handler: cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				s.workflowKeys.Add(obj)
			},
			DeleteFunc: func(obj interface{}) {
				s.workflowKeys.Remove(obj)
			},
		},
	})
	s.workflowTemplateInformer.AddEventHandler(cache.FilteringResourceEventHandler{
		// we're only interested it templates that have event expressions
		FilterFunc: func(obj interface{}) bool {
			tmpl, ok := obj.(*wfv1.WorkflowTemplate)
			return ok && s.instanceIDService.Validate(tmpl) == nil && tmpl.Spec.Event != nil
		},
		Handler: cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				s.templateKeys.Add(obj)
			},
			DeleteFunc: func(obj interface{}) {
				s.templateKeys.Remove(obj)
			},
		},
	})

	go s.workflowInformer.Run(stopCh)
	go s.workflowTemplateInformer.Run(stopCh)

	for _, informer := range []cache.SharedIndexInformer{s.workflowInformer, s.workflowTemplateInformer} {
		if !cache.WaitForCacheSync(stopCh, informer.HasSynced) {
			log.Error("Timed out waiting for event caches to sync")
			return
		}
	}

	for {
		select {
		case <-stopCh:
			// process all outstanding operations, so we don't lose any operations
			for len(s.operationPipeline) > 0 {
				operation := <-s.operationPipeline
				operation.Execute()
			}
			return
		case operation := <-s.operationPipeline:
			operation.Execute()
		}
	}
}

func (s *Controller) ReceiveEvent(ctx context.Context, req *eventpkg.EventRequest) (*eventpkg.EventResponse, error) {
	s.operationPipeline <- dispatch.New(ctx, s.hydrator, s.workflowKeys, s.templateKeys, req.Namespace, req.Event)
	log.Infof("ALEX %v", len(s.workflowTemplateInformer.GetStore().List()))
	return &eventpkg.EventResponse{}, nil
}
