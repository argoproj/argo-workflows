package event

import (
	"context"
	"time"

	log "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/cache"

	eventpkg "github.com/argoproj/argo/pkg/apiclient/event"
	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo/pkg/client/clientset/versioned"
	"github.com/argoproj/argo/pkg/client/informers/externalversions/workflow/v1alpha1"
	"github.com/argoproj/argo/server/auth"
	"github.com/argoproj/argo/util/instanceid"
	"github.com/argoproj/argo/workflow/common"
	"github.com/argoproj/argo/workflow/hydrator"
)

type Controller struct {
	workflowInformer  cache.SharedIndexInformer
	templateInformer  cache.SharedIndexInformer
	hydrator          hydrator.Interface
	instanceIDService instanceid.Service
	workflows         map[corev1.ObjectReference]bool
	templates         map[corev1.ObjectReference]bool
	pipeline          chan operation
}

func (s *Controller) Run(stopCh <-chan struct{}) {
	s.workflowInformer.AddEventHandler(cache.FilteringResourceEventHandler{
		// we're only interested in incomplete workflows that have running suspend nodes
		FilterFunc: func(obj interface{}) bool {
			wf, ok := obj.(*wfv1.Workflow)
			logCtx := log.WithFields(log.Fields{"namespace": wf.Namespace, "workflow": wf.Name})
			if !ok || wf.GetLabels()[common.LabelKeyCompleted] == "true" || s.instanceIDService.Validate(wf) != nil {
				logCtx.Debug("ignoring workflow for events: not a workflow, complete workflow, or invalid instance ID")
				return false
			}
			err := s.hydrator.Hydrate(wf)
			if err != nil {
				logCtx.WithError(err).Error("failed to hydrate workflow")
				return false
			}
			for _, node := range wf.Status.Nodes {
				if node.Type == wfv1.NodeTypeSuspend {
					if t := wf.GetTemplateByName(node.TemplateName); t != nil && t.Suspend != nil && t.Suspend.Event != nil {
						logCtx.Debug("considering workflow for events")
						return true
					}
				}
			}
			logCtx.Debug("ignoring workflow for events: no suspend nodes consuming events")
			return false
		},
		Handler: cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				wf := obj.(*wfv1.Workflow)
				log.WithFields(log.Fields{"namespace": wf.Namespace, "workflow": wf.Name}).Debug("adding workflow for event consideration")
				s.workflows[corev1.ObjectReference{Namespace: wf.Namespace, Name: wf.Name}] = true
			},
			DeleteFunc: func(obj interface{}) {
				wf := obj.(*wfv1.Workflow)
				log.WithFields(log.Fields{"namespace": wf.Namespace, "workflow": wf.Name}).Debug("deleting workflow from event consideration")
				delete(s.workflows, corev1.ObjectReference{Namespace: wf.Namespace, Name: wf.Name})
			},
		},
	})
	s.templateInformer.AddEventHandler(cache.FilteringResourceEventHandler{
		// we're only interested it templates that have event expressions
		FilterFunc: func(obj interface{}) bool {
			tmpl, ok := obj.(*wfv1.WorkflowTemplate)
			logCtx := log.WithFields(log.Fields{"namespace": tmpl.Namespace, "template": tmpl.Name})
			if !ok || s.instanceIDService.Validate(tmpl) != nil || tmpl.Spec.Event == nil {
				logCtx.Debug("ignoring workflow template for events")
				return false
			}
			logCtx.Debug("considering workflow template for events")
			return true
		},
		Handler: cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				tmpl := obj.(*wfv1.WorkflowTemplate)
				log.WithFields(log.Fields{"namespace": tmpl.Namespace, "template": tmpl.Name}).Debug("adding workflow template to event consideration")
				s.templates[corev1.ObjectReference{Namespace: tmpl.Namespace, Name: tmpl.Name}] = true
			},
			DeleteFunc: func(obj interface{}) {
				tmpl := obj.(*wfv1.WorkflowTemplate)
				log.WithFields(log.Fields{"namespace": tmpl.Namespace, "template": tmpl.Name}).Debug("deleting workflow template from event consideration")
				delete(s.templates, corev1.ObjectReference{Namespace: tmpl.Namespace, Name: tmpl.Name})
			},
		},
	})

	go s.workflowInformer.Run(stopCh)
	go s.templateInformer.Run(stopCh)

	for {
		select {
		case operation := <-s.pipeline:
			operation.Execute()
		case <-stopCh:
			return
		}
	}
}

func (s *Controller) ReceiveEvent(ctx context.Context, req *eventpkg.EventRequest) (*eventpkg.EventResponse, error) {
	s.pipeline <- operation{
		client:    auth.GetWfClient(ctx),
		hydrator:  s.hydrator,
		namespace: req.Namespace,
		event:     req.Event,
		metadata:  metaData(ctx),
		workflows: s.workflows,
		templates: s.templates,
	}
	return &eventpkg.EventResponse{}, nil
}

var _ eventpkg.EventServiceServer = &Controller{}

func NewController(client *versioned.Clientset, namespace string, instanceService instanceid.Service, hydrator hydrator.Interface) *Controller {
	return &Controller{
		workflowInformer:  v1alpha1.NewWorkflowInformer(client, namespace, 20*time.Second, cache.Indexers{}),
		templateInformer:  v1alpha1.NewWorkflowTemplateInformer(client, namespace, 20*time.Second, cache.Indexers{}),
		workflows:         make(map[corev1.ObjectReference]bool),
		templates:         make(map[corev1.ObjectReference]bool),
		instanceIDService: instanceService,
		hydrator:          hydrator,
		pipeline:          make(chan operation, 64),
	}
}
