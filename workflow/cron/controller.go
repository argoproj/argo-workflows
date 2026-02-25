package cron

import (
	"context"
	"fmt"
	"reflect"
	"time"

	"github.com/argoproj/pkg/sync"
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

	"github.com/argoproj/argo-workflows/v4/pkg/apis/workflow"
	"github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/pkg/client/clientset/versioned"
	wfextvv1alpha1 "github.com/argoproj/argo-workflows/v4/pkg/client/informers/externalversions/workflow/v1alpha1"
	wfctx "github.com/argoproj/argo-workflows/v4/util/context"
	"github.com/argoproj/argo-workflows/v4/util/env"
	"github.com/argoproj/argo-workflows/v4/util/logging"
	"github.com/argoproj/argo-workflows/v4/workflow/common"
	"github.com/argoproj/argo-workflows/v4/workflow/events"
	"github.com/argoproj/argo-workflows/v4/workflow/metrics"
	"github.com/argoproj/argo-workflows/v4/workflow/util"
)

// Controller is a controller for cron workflows
type Controller struct {
	namespace            string
	managedNamespace     string
	instanceID           string
	cron                 *cronFacade
	keyLock              sync.KeyLock
	wfClientset          versioned.Interface
	wfLister             util.WorkflowLister
	cronWfInformer       informers.GenericInformer
	wftmplInformer       wfextvv1alpha1.WorkflowTemplateInformer
	cwftmplInformer      wfextvv1alpha1.ClusterWorkflowTemplateInformer
	wfDefaults           *v1alpha1.Workflow
	cronWfQueue          workqueue.TypedRateLimitingInterface[string]
	dynamicInterface     dynamic.Interface
	metrics              *metrics.Metrics
	eventRecorderManager events.EventRecorderManager
	cronWorkflowWorkers  int
	logger               logging.Logger
}

const (
	cronWorkflowResyncPeriod = 20 * time.Minute
)

var cronSyncPeriod time.Duration

func init() {
	// this make sure we support timezones
	_, err := time.Parse(time.RFC822, "17 Oct 07 14:03 PST")
	if err != nil {
		logging.InitLogger().WithFatal().WithError(err).Error(context.Background(), "failed to parse time")
	}
	cronSyncPeriod = env.LookupEnvDurationOr(logging.InitLoggerInContext(), "CRON_SYNC_PERIOD", 10*time.Second)
	logging.InitLogger().WithField("cronSyncPeriod", cronSyncPeriod).Info(context.Background(), "cron config")
}

// NewCronController creates a new cron controller
func NewCronController(ctx context.Context, wfclientset versioned.Interface, dynamicInterface dynamic.Interface, namespace string, managedNamespace string, instanceID string, metrics *metrics.Metrics,
	eventRecorderManager events.EventRecorderManager, cronWorkflowWorkers int, wftmplInformer wfextvv1alpha1.WorkflowTemplateInformer, cwftmplInformer wfextvv1alpha1.ClusterWorkflowTemplateInformer, wfDefaults *v1alpha1.Workflow,
) *Controller {
	ctx, logger := logging.RequireLoggerFromContext(ctx).WithField("component", "cron").InContext(ctx)

	return &Controller{
		wfClientset:          wfclientset,
		namespace:            namespace,
		managedNamespace:     managedNamespace,
		instanceID:           instanceID,
		cron:                 newCronFacade(),
		keyLock:              sync.NewKeyLock(),
		dynamicInterface:     dynamicInterface,
		cronWfQueue:          metrics.RateLimiterWithBusyWorkers(ctx, workqueue.DefaultTypedControllerRateLimiter[string](), "cron_wf_queue"),
		wfDefaults:           wfDefaults,
		metrics:              metrics,
		eventRecorderManager: eventRecorderManager,
		wftmplInformer:       wftmplInformer,
		cwftmplInformer:      cwftmplInformer,
		cronWorkflowWorkers:  cronWorkflowWorkers,
		logger:               logger,
	}
}

// Run start the cron controller
func (cc *Controller) Run(ctx context.Context) {
	defer runtimeutil.HandleCrashWithContext(ctx, runtimeutil.PanicHandlers...)
	defer cc.cronWfQueue.ShutDown()
	cc.logger.WithField("instanceID", cc.instanceID).Info(ctx, "Starting CronWorkflow controller")

	cc.cronWfInformer = dynamicinformer.NewFilteredDynamicSharedInformerFactory(cc.dynamicInterface, cronWorkflowResyncPeriod, cc.managedNamespace, func(options *v1.ListOptions) {
		cronWfInformerListOptionsFunc(options, cc.instanceID)
	}).ForResource(schema.GroupVersionResource{Group: workflow.Group, Version: workflow.Version, Resource: workflow.CronWorkflowPlural})
	err := cc.addCronWorkflowInformerHandler(ctx)
	if err != nil {
		cc.logger.WithFatal().Error(ctx, err.Error())
	}

	wfInformer := util.NewWorkflowInformer(ctx, cc.dynamicInterface, cc.managedNamespace, cronWorkflowResyncPeriod,
		func(options *v1.ListOptions) { wfInformerListOptionsFunc(options, cc.instanceID) },
		func(options *v1.ListOptions) { wfInformerListOptionsFunc(options, cc.instanceID) },
		cache.Indexers{})
	go wfInformer.Run(ctx.Done())

	cc.wfLister = util.NewWorkflowLister(ctx, wfInformer)

	cc.cron.Start()
	defer cc.cron.Stop()

	go cc.cronWfInformer.Informer().Run(ctx.Done())

	go wait.UntilWithContext(ctx, cc.syncAll, cronSyncPeriod)

	for i := 0; i < cc.cronWorkflowWorkers; i++ {
		go wait.UntilWithContext(ctx, cc.runCronWorker, time.Second)
	}

	<-ctx.Done()
}

func (cc *Controller) runCronWorker(ctx context.Context) {
	for cc.processNextCronItem(ctx) {
	}
}

func (cc *Controller) processNextCronItem(ctx context.Context) bool {
	defer runtimeutil.HandleCrashWithContext(ctx, runtimeutil.PanicHandlers...)

	key, quit := cc.cronWfQueue.Get()
	if quit {
		return false
	}
	defer cc.cronWfQueue.Done(key)

	cc.keyLock.Lock(key)
	defer cc.keyLock.Unlock(key)

	ctx, logger := cc.logger.WithField("cronWorkflow", key).InContext(ctx)
	logger.Info(ctx, "Processing cron workflow")

	obj, exists, err := cc.cronWfInformer.Informer().GetIndexer().GetByKey(key)
	if err != nil {
		logger.WithError(err).Error(ctx, fmt.Sprintf("Failed to get CronWorkflow '%s' from informer index", key))
		return true
	}
	if !exists {
		logger.Info(ctx, "Deleting cron workflow")
		cc.cron.Delete(key)
		return true
	}

	un, ok := obj.(*unstructured.Unstructured)
	if !ok {
		logger.WithField("type", reflect.TypeOf(obj).Name()).Error(ctx, "malformed cluster workflow template: expected *unstructured.Unstructured, got type")
		return true
	}
	cronWf := &v1alpha1.CronWorkflow{}
	err = util.FromUnstructuredObj(un, cronWf)
	if err != nil {
		cc.eventRecorderManager.Get(ctx, un.GetNamespace()).Event(un, apiv1.EventTypeWarning, "Malformed", err.Error())
		logger.WithError(err).Error(ctx, "malformed cron workflow: could not convert from unstructured")
		return true
	}
	ctx = wfctx.InjectObjectMeta(ctx, &cronWf.ObjectMeta)

	cronWorkflowOperationCtx := newCronWfOperationCtx(ctx, cronWf, cc.wfClientset, cc.metrics, cc.wftmplInformer, cc.cwftmplInformer, cc.wfDefaults)

	err = cronWorkflowOperationCtx.validateCronWorkflow(ctx)
	if err != nil {
		logger.WithError(err).Error(ctx, "invalid cron workflow")
		return true
	}

	wfWasRun, err := cronWorkflowOperationCtx.runOutstandingWorkflows(ctx)
	if err != nil {
		logger.WithError(err).Error(ctx, "could not run outstanding Workflow")
		return true
	} else if wfWasRun {
		// A workflow was run, so the cron workflow will be requeued. Return here to avoid duplicating work
		return true
	}

	// The job is currently scheduled, remove it and re add it.
	cc.cron.Delete(key)

	for _, schedule := range cronWf.Spec.GetSchedulesWithTimezone() {
		lastScheduledTimeFunc, err := cc.cron.AddJob(key, schedule, cronWorkflowOperationCtx)
		if err != nil {
			logger.WithError(err).Error(ctx, "could not schedule CronWorkflow")
			return true
		}
		cronWorkflowOperationCtx.scheduledTimeFunc = lastScheduledTimeFunc
	}

	logger.Info(ctx, "CronWorkflow added")

	return true
}

func (cc *Controller) addCronWorkflowInformerHandler(ctx context.Context) error {
	_, err := cc.cronWfInformer.Informer().AddEventHandler(
		cache.FilteringResourceEventHandler{
			FilterFunc: func(obj interface{}) bool {
				un, ok := obj.(*unstructured.Unstructured)
				if !ok {
					cc.logger.WithField("obj", obj).Warn(ctx, "Cron Workflow FilterFunc: is not an unstructured")
					return false
				}
				return !isCompleted(un)
			},
			Handler: cache.ResourceEventHandlerFuncs{
				AddFunc: func(obj interface{}) {
					key, err := cache.MetaNamespaceKeyFunc(obj)
					if err == nil {
						cc.cronWfQueue.Add(key)
					}
				},
				UpdateFunc: func(old, newObj interface{}) {
					key, err := cache.MetaNamespaceKeyFunc(newObj)
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
			},
		})
	if err != nil {
		return err
	}
	return nil
}

func isCompleted(wf v1.Object) bool {
	completed, ok := wf.GetLabels()[common.LabelKeyCronWorkflowCompleted]
	if !ok {
		return false
	}
	return completed == "true"
}

func (cc *Controller) syncAll(ctx context.Context) {
	defer runtimeutil.HandleCrashWithContext(ctx, runtimeutil.PanicHandlers...)

	cc.logger.Debug(ctx, "Syncing all CronWorkflows")

	workflows, err := cc.wfLister.List()
	if err != nil {
		return
	}
	groupedWorkflows := groupWorkflows(workflows)

	cronWorkflows := cc.cronWfInformer.Informer().GetStore().List()
	for _, obj := range cronWorkflows {
		un, ok := obj.(*unstructured.Unstructured)
		if !ok {
			cc.logger.Error(ctx, "Unable to convert object to unstructured when syncing CronWorkflows")
			continue
		}
		cronWf := &v1alpha1.CronWorkflow{}
		err := util.FromUnstructuredObj(un, cronWf)
		if err != nil {
			cc.logger.WithError(err).Error(ctx, "Unable to convert unstructured to CronWorkflow when syncing CronWorkflows")
			continue
		}

		err = cc.syncCronWorkflow(ctx, cronWf, groupedWorkflows[cronWf.UID])
		if err != nil {
			cc.logger.WithError(err).Error(ctx, "Unable to sync CronWorkflow")
			continue
		}
	}
}

func (cc *Controller) syncCronWorkflow(ctx context.Context, cronWf *v1alpha1.CronWorkflow, workflows []v1alpha1.Workflow) error {
	key := cronWf.Namespace + "/" + cronWf.Name
	cc.keyLock.Lock(key)
	defer cc.keyLock.Unlock(key)

	cwoc := newCronWfOperationCtx(ctx, cronWf, cc.wfClientset, cc.metrics, cc.wftmplInformer, cc.cwftmplInformer, cc.wfDefaults)
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

func cronWfInformerListOptionsFunc(options *v1.ListOptions, instanceID string) {
	options.FieldSelector = fields.Everything().String()
	labelSelector := labels.NewSelector().Add(util.InstanceIDRequirement(instanceID))
	options.LabelSelector = labelSelector.String()
}

func wfInformerListOptionsFunc(options *v1.ListOptions, instanceID string) {
	options.FieldSelector = fields.Everything().String()
	isCronWorkflowChildReq, err := labels.NewRequirement(common.LabelKeyCronWorkflow, selection.Exists, []string{})
	if err != nil {
		panic(err)
	}
	labelSelector := labels.NewSelector().Add(*isCronWorkflowChildReq)
	labelSelector = labelSelector.Add(util.InstanceIDRequirement(instanceID))
	options.LabelSelector = labelSelector.String()
}
