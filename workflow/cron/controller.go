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
	cronWorkflowResyncPeriod = 20 * time.Minute
	cronWorkflowWorkers      = 8
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

	wfInformer := util.NewWorkflowInformer(cc.dynamicInterface, cc.managedNamespace, cronWorkflowResyncPeriod, func(options *v1.ListOptions) {
		wfInformerListOptionsFunc(options, cc.instanceId)
	}, cache.Indexers{})

	cc.wfLister = util.NewWorkflowLister(wfInformer)

	cc.cron.Start()
	defer cc.cron.Stop()

	go cc.cronWfInformer.Informer().Run(ctx.Done())
	go wait.Until(cc.syncAll, 10 * time.Second, ctx.Done())

	for i := 0; i < cronWorkflowWorkers; i++ {
		go wait.Until(cc.runCronWorker, time.Second, ctx.Done())
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
	err = util.FromUnstructuredObj(un, cronWf)
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

	wfWasRun, err := cronWorkflowOperationCtx.runOutstandingWorkflows()
	if err != nil {
		logCtx.WithError(err).Error("could not run outstanding Workflow")
		return true
	} else if wfWasRun {
		// A workflow was run, so the cron workflow will be requeued. Return here to avoid duplicating work
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

func (cc *Controller) syncAll() {
	log.Info("Syncing all CronWorkflows")

	cronWorkflows := cc.cronWfInformer.Informer().GetStore().List()
	for _, obj := range cronWorkflows {

		un, ok := obj.(*unstructured.Unstructured)
		if !ok {
			log.Error("Unable to convert object to unstructured when syncing CronWorkflows")
			continue
		}
		cronWf := &v1alpha1.CronWorkflow{}
		err := util.FromUnstructuredObj(un, cronWf)
		if err != nil {
			log.WithError(err).Error("Unable to convert unstructured to CronWorkflow when syncing CronWorkflows")
			continue
		}

		cwoc := newCronWfOperationCtx(cronWf, cc.wfClientset, cc.wfLister, cc.metrics)

		err = cwoc.enforceHistoryLimit()
		if err != nil {
			log.WithError(err).Error("Error enforcing history limit")
			continue
		}
		err = cwoc.reconcileActiveWfs()
		if err != nil {
			log.WithError(err).Error("Error reconciling workflows")
			continue
		}

		cwoc.persistUpdate()
	}
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
