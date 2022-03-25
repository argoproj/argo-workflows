package cron

import (
	"context"
	"fmt"
	"reflect"
	"time"

	"github.com/argoproj/pkg/sync"
	log "github.com/sirupsen/logrus"
	apiv1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/apimachinery/pkg/types"
	runtimeutil "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/dynamic/dynamicinformer"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"

	"github.com/argoproj/argo-workflows/v3/pkg/apis/workflow"
	"github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v3/pkg/client/clientset/versioned"
	"github.com/argoproj/argo-workflows/v3/util/env"
	"github.com/argoproj/argo-workflows/v3/workflow/common"
	"github.com/argoproj/argo-workflows/v3/workflow/events"
	"github.com/argoproj/argo-workflows/v3/workflow/metrics"
	"github.com/argoproj/argo-workflows/v3/workflow/util"
)

// Controller is a controller for cron workflows
type Controller struct {
	namespace            string
	managedNamespace     string
	instanceId           string
	cron                 *cronFacade
	keyLock              sync.KeyLock
	wfClientset          versioned.Interface
	wfLister             util.WorkflowLister
	cronWfInformer       informers.GenericInformer
	cronWfQueue          workqueue.RateLimitingInterface
	dynamicInterface     dynamic.Interface
	metrics              *metrics.Metrics
	eventRecorderManager events.EventRecorderManager
}

const (
	cronWorkflowResyncPeriod = 20 * time.Minute
	cronWorkflowWorkers      = 8
)

var (
	cronSyncPeriod = env.LookupEnvDurationOr("CRON_SYNC_PERIOD", 10*time.Second)
)

func init() {
	// this make sure we support timezones
	_, err := time.Parse(time.RFC822, "17 Oct 07 14:03 PST")
	if err != nil {
		log.Fatal(err)
	}
	log.WithField("cronSyncPeriod", cronSyncPeriod).Info("cron config")
}

func NewCronController(wfclientset versioned.Interface, dynamicInterface dynamic.Interface, namespace string, managedNamespace string, instanceId string, metrics *metrics.Metrics, eventRecorderManager events.EventRecorderManager) *Controller {
	return &Controller{
		wfClientset:          wfclientset,
		namespace:            namespace,
		managedNamespace:     managedNamespace,
		instanceId:           instanceId,
		cron:                 newCronFacade(),
		keyLock:              sync.NewKeyLock(),
		dynamicInterface:     dynamicInterface,
		cronWfQueue:          metrics.RateLimiterWithBusyWorkers(workqueue.DefaultControllerRateLimiter(), "cron_wf_queue"),
		metrics:              metrics,
		eventRecorderManager: eventRecorderManager,
	}
}

func (cc *Controller) Run(ctx context.Context) {
	defer runtimeutil.HandleCrash(runtimeutil.PanicHandlers...)
	defer cc.cronWfQueue.ShutDown()
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
	go wfInformer.Run(ctx.Done())

	cc.wfLister = util.NewWorkflowLister(wfInformer)

	cc.cron.Start()
	defer cc.cron.Stop()

	go cc.cronWfInformer.Informer().Run(ctx.Done())

	go wait.UntilWithContext(ctx, cc.syncAll, cronSyncPeriod)

	for i := 0; i < cronWorkflowWorkers; i++ {
		go wait.Until(cc.runCronWorker, time.Second, ctx.Done())
	}

	<-ctx.Done()
}

func (cc *Controller) runCronWorker() {
	ctx := context.TODO()
	for cc.processNextCronItem(ctx) {
	}
}

func (cc *Controller) processNextCronItem(ctx context.Context) bool {
	defer runtimeutil.HandleCrash(runtimeutil.PanicHandlers...)

	key, quit := cc.cronWfQueue.Get()
	if quit {
		return false
	}
	defer cc.cronWfQueue.Done(key)

	cc.keyLock.Lock(key.(string))
	defer cc.keyLock.Unlock(key.(string))

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

	cronWorkflowOperationCtx := newCronWfOperationCtx(cronWf, cc.wfClientset, cc.metrics)

	err = cronWorkflowOperationCtx.validateCronWorkflow()
	if err != nil {
		logCtx.WithError(err).Error("invalid cron workflow")
		return true
	}

	wfWasRun, err := cronWorkflowOperationCtx.runOutstandingWorkflows(ctx)
	if err != nil {
		logCtx.WithError(err).Error("could not run outstanding Workflow")
		return true
	} else if wfWasRun {
		// A workflow was run, so the cron workflow will be requeued. Return here to avoid duplicating work
		return true
	}

	// The job is currently scheduled, remove it and re add it.
	cc.cron.Delete(key.(string))

	lastScheduledTimeFunc, err := cc.cron.AddJob(key.(string), cronWf.Spec.GetScheduleString(), cronWorkflowOperationCtx)
	if err != nil {
		logCtx.WithError(err).Error("could not schedule CronWorkflow")
		return true
	}

	cronWorkflowOperationCtx.scheduledTimeFunc = lastScheduledTimeFunc

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

func (cc *Controller) syncAll(ctx context.Context) {
	defer runtimeutil.HandleCrash(runtimeutil.PanicHandlers...)

	log.Debug("Syncing all CronWorkflows")

	workflows, err := cc.wfLister.List()
	if err != nil {
		return
	}
	groupedWorkflows := groupWorkflows(workflows)

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

		err = cc.syncCronWorkflow(ctx, cronWf, groupedWorkflows[cronWf.UID])
		if err != nil {
			log.WithError(err).Error("Unable to sync CronWorkflow")
			continue
		}
	}
}

func (cc *Controller) syncCronWorkflow(ctx context.Context, cronWf *v1alpha1.CronWorkflow, workflows []v1alpha1.Workflow) error {
	key := cronWf.Namespace + "/" + cronWf.Name
	cc.keyLock.Lock(key)
	defer cc.keyLock.Unlock(key)

	cwoc := newCronWfOperationCtx(cronWf, cc.wfClientset, cc.metrics)
	err := cwoc.enforceHistoryLimit(ctx, workflows)
	if err != nil {
		return err
	}
	err = cwoc.reconcileActiveWfs(ctx, workflows)
	if err != nil {
		return err
	}

	return nil
}

func groupWorkflows(wfs []*v1alpha1.Workflow) map[types.UID][]v1alpha1.Workflow {
	cwfChildren := make(map[types.UID][]v1alpha1.Workflow)
	for _, wf := range wfs {
		owner := v1.GetControllerOf(wf)
		if owner == nil || owner.Kind != workflow.CronWorkflowKind {
			continue
		}
		cwfChildren[owner.UID] = append(cwfChildren[owner.UID], *wf)
	}
	return cwfChildren
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
