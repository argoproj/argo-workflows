package cron

import (
	"context"
	"fmt"
	"reflect"
	"time"

	log "github.com/sirupsen/logrus"
	apiv1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/dynamic/dynamicinformer"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"

	"github.com/argoproj/argo/pkg/apis/workflow"
	"github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo/pkg/client/clientset/versioned"
	"github.com/argoproj/argo/workflow/common"
	"github.com/argoproj/argo/workflow/events"
	"github.com/argoproj/argo/workflow/metrics"
	"github.com/argoproj/argo/workflow/util"
)

// Controller is a controller for cron workflows
type Controller struct {
	namespace            string
	managedNamespace     string
	instanceId           string
	cron                 *cronFacade
	wfClientset          versioned.Interface
	wfInformer           cache.SharedIndexInformer
	wfLister             util.WorkflowLister
	wfQueue              workqueue.RateLimitingInterface
	cronWfInformer       informers.GenericInformer
	cronWfQueue          workqueue.RateLimitingInterface
	restConfig           *rest.Config
	dynamicInterface     dynamic.Interface
	metrics              *metrics.Metrics
	eventRecorderManager events.EventRecorderManager
}

const (
	cronWorkflowResyncPeriod    = 20 * time.Minute
	cronWorkflowWorkers         = 8
	cronWorkflowWorkflowWorkers = 8
)

func NewCronController(wfclientset versioned.Interface, restConfig *rest.Config, dynamicInterface dynamic.Interface, namespace string, managedNamespace string, instanceId string, metrics *metrics.Metrics, eventRecorderManager events.EventRecorderManager) *Controller {
	return &Controller{
		wfClientset:          wfclientset,
		namespace:            namespace,
		managedNamespace:     managedNamespace,
		instanceId:           instanceId,
		cron:                 newCronFacade(),
		restConfig:           restConfig,
		dynamicInterface:     dynamicInterface,
		wfQueue:              workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "wf_cron_queue"),
		cronWfQueue:          workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "cron_wf_queue"),
		metrics:              metrics,
		eventRecorderManager: eventRecorderManager,
	}
}

func (cc *Controller) Run(ctx context.Context) {
	defer cc.cronWfQueue.ShutDown()
	defer cc.wfQueue.ShutDown()
	log.Infof("Starting CronWorkflow controller")
	if cc.instanceId != "" {
		log.Infof("...with InstanceID: %s", cc.instanceId)
	}

	cc.cronWfInformer = dynamicinformer.NewFilteredDynamicSharedInformerFactory(cc.dynamicInterface, cronWorkflowResyncPeriod, cc.managedNamespace, func(options *v1.ListOptions) {
		cronWfInformerListOptionsFunc(options, cc.instanceId)
	}).ForResource(schema.GroupVersionResource{Group: workflow.Group, Version: workflow.Version, Resource: workflow.CronWorkflowPlural})
	cc.addCronWorkflowInformerHandler()

	cc.wfInformer = util.NewWorkflowInformer(cc.restConfig, cc.managedNamespace, cronWorkflowResyncPeriod, func(options *v1.ListOptions) {
		wfInformerListOptionsFunc(options, cc.instanceId)
	})
	cc.addWorkflowInformerHandler()

	cc.wfLister = util.NewWorkflowLister(cc.wfInformer)

	cc.cron.Start()
	defer cc.cron.Stop()

	go cc.cronWfInformer.Informer().Run(ctx.Done())
	go cc.wfInformer.Run(ctx.Done())

	for i := 0; i < cronWorkflowWorkers; i++ {
		go wait.Until(cc.runCronWorker, time.Second, ctx.Done())
	}

	for i := 0; i < cronWorkflowWorkflowWorkers; i++ {
		go wait.Until(cc.runWorkflowWorker, time.Second, ctx.Done())
	}

	<-ctx.Done()
}

func (cc *Controller) runCronWorker() {
	for cc.processNextCronItem() {
	}
}

func (cc *Controller) processNextCronItem() bool {
	key, quit := cc.cronWfQueue.Get()
	if quit {
		return false
	}
	defer cc.cronWfQueue.Done(key)
	logCtx := log.WithField("cronWorkflow", key)
	logCtx.Infof("Processing %s", key)

	obj, exists, err := cc.cronWfInformer.Informer().GetIndexer().GetByKey(key.(string))
	if err != nil {
		logCtx.WithError(err).Error(fmt.Sprintf("Failed to get CronWorkflow '%s' from informer index", key))
		return true
	}
	if !exists {
		logCtx.Infof("Deleting '%s'", key)
		cc.cron.Delete(key.(string))
		return true
	}

	un, ok := obj.(*unstructured.Unstructured)
	if !ok {
		logCtx.Errorf("malformed cluster workflow template: expected *unstructured.Unstructured, got %s", reflect.TypeOf(obj).Name())
		return true
	}
	cronWf := &v1alpha1.CronWorkflow{}
	err = runtime.DefaultUnstructuredConverter.FromUnstructured(un.Object, cronWf)
	if err != nil {
		cc.eventRecorderManager.Get(un.GetNamespace()).Event(un, apiv1.EventTypeWarning, "Malformed", err.Error())
		logCtx.WithError(err).Error("malformed cron workflow: could not convert from unstructured")
		return true
	}

	cronWorkflowOperationCtx := newCronWfOperationCtx(cronWf, cc.wfClientset, cc.wfLister, cc.metrics)

	err = cronWorkflowOperationCtx.validateCronWorkflow()
	if err != nil {
		logCtx.WithError(err).Error("invalid cron workflow")
		return true
	}

	err = cronWorkflowOperationCtx.runOutstandingWorkflows()
	if err != nil {
		logCtx.WithError(err).Error("could not run outstanding Workflow")
		return true
	}

	// The job is currently scheduled, remove it and re add it.
	cc.cron.Delete(key.(string))

	cronSchedule := cronWf.Spec.Schedule
	if cronWf.Spec.Timezone != "" {
		cronSchedule = "CRON_TZ=" + cronWf.Spec.Timezone + " " + cronSchedule
	}

	err = cc.cron.AddJob(key.(string), cronSchedule, cronWorkflowOperationCtx)
	if err != nil {
		logCtx.WithError(err).Error("could not schedule CronWorkflow")
		return true
	}

	logCtx.Infof("CronWorkflow %s added", key.(string))

	return true
}

func (cc *Controller) runWorkflowWorker() {
	for cc.processNextWorkflowItem() {
	}
}

func (cc *Controller) processNextWorkflowItem() bool {
	key, quit := cc.wfQueue.Get()
	if quit {
		return false
	}
	defer cc.wfQueue.Done(key)

	obj, wfExists, err := cc.wfInformer.GetIndexer().GetByKey(key.(string))
	if err != nil {
		log.WithError(err).Error(fmt.Sprintf("Failed to get Workflow '%s' from informer index", key))
		return true
	}

	// Check if the workflow no longer exists. If the workflow was deleted while it was an active workflow of a cron
	// workflow, the cron workflow will reconcile this fact on its own next time it is processed.
	if !wfExists {
		log.Warnf("Workflow '%s' no longer exists", key)
		return true
	}

	// The workflow informer receives unstructured objects to deal with the possibility of invalid
	// workflow manifests that are unable to unmarshal to workflow objects
	un, ok := obj.(*unstructured.Unstructured)
	if !ok {
		log.Warnf("Key '%s' in index is not an unstructured", key)
		return true
	}

	wf, err := util.FromUnstructured(un)
	if err != nil {
		log.Warnf("Failed to unmarshal key '%s' to workflow object: %v", key, err)
		return true
	}

	if wf.OwnerReferences == nil || len(wf.OwnerReferences) != 1 {
		log.Warnf("Workflow '%s' stemming from CronWorkflow is malformed", wf.Name)
		return true
	}

	// Workflows are run in the same namespace as CronWorkflow
	nameEntryIdMapKey := wf.Namespace + "/" + wf.OwnerReferences[0].Name
	woc, err := cc.cron.Load(nameEntryIdMapKey)
	if err != nil {
		log.Warnf("Parent CronWorkflow '%s' is bad: %v", nameEntryIdMapKey, err)
		return true
	}

	defer woc.persistUpdate()

	// If the workflow is completed or was deleted, remove it from Active Workflows
	if wf.Status.Fulfilled() || !wfExists {
		log.Warnf("Workflow '%s' from CronWorkflow '%s' completed", wf.Name, woc.cronWf.Name)
		woc.removeActiveWf(wf)
	}

	woc.enforceHistoryLimit()
	return true
}

func (cc *Controller) addCronWorkflowInformerHandler() {
	cc.cronWfInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			key, err := cache.MetaNamespaceKeyFunc(obj)
			if err == nil {
				cc.cronWfQueue.Add(key)
			}
		},
		UpdateFunc: func(old, new interface{}) {
			key, err := cache.MetaNamespaceKeyFunc(new)
			if err == nil {
				cc.cronWfQueue.Add(key)
			}
		},
		DeleteFunc: func(obj interface{}) {
			key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
			if err == nil {
				cc.cronWfQueue.Add(key)
			}
		},
	})
}

func (cc *Controller) addWorkflowInformerHandler() {
	cc.wfInformer.AddEventHandler(
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				key, err := cache.MetaNamespaceKeyFunc(obj)
				if err == nil {
					cc.wfQueue.Add(key)
				}
			},
			UpdateFunc: func(old, new interface{}) {
				key, err := cache.MetaNamespaceKeyFunc(new)
				if err == nil {
					cc.wfQueue.Add(key)
				}
			},
			DeleteFunc: func(obj interface{}) {
				key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
				if err == nil {
					cc.wfQueue.Add(key)
				}
			},
		},
	)
}

func cronWfInformerListOptionsFunc(options *v1.ListOptions, instanceId string) {
	options.FieldSelector = fields.Everything().String()
	labelSelector := labels.NewSelector().Add(util.InstanceIDRequirement(instanceId))
	options.LabelSelector = labelSelector.String()
}

func wfInformerListOptionsFunc(options *v1.ListOptions, instanceId string) {
	options.FieldSelector = fields.Everything().String()
	isCronWorkflowChildReq, err := labels.NewRequirement(common.LabelKeyCronWorkflow, selection.Exists, []string{})
	if err != nil {
		panic(err)
	}
	labelSelector := labels.NewSelector().Add(*isCronWorkflowChildReq)
	labelSelector = labelSelector.Add(util.InstanceIDRequirement(instanceId))
	options.LabelSelector = labelSelector.String()
}
