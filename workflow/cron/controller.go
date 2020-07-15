package cron

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/robfig/cron/v3"
	log "github.com/sirupsen/logrus"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"

	"github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo/pkg/client/clientset/versioned"
	"github.com/argoproj/argo/pkg/client/informers/externalversions"
	extv1alpha1 "github.com/argoproj/argo/pkg/client/informers/externalversions/workflow/v1alpha1"
	"github.com/argoproj/argo/workflow/common"
	"github.com/argoproj/argo/workflow/metrics"
	"github.com/argoproj/argo/workflow/util"
)

// Controller is a controller for cron workflows
type Controller struct {
	namespace          string
	managedNamespace   string
	instanceId         string
	cron               *cron.Cron
	nameEntryIDMap     map[string]cron.EntryID
	nameEntryIDMapLock *sync.Mutex
	wfClientset        versioned.Interface
	wfInformer         cache.SharedIndexInformer
	wfLister           util.WorkflowLister
	wfQueue            workqueue.RateLimitingInterface
	cronWfInformer     extv1alpha1.CronWorkflowInformer
	cronWfQueue        workqueue.RateLimitingInterface
	restConfig         *rest.Config
	metrics            *metrics.Metrics
}

const (
	cronWorkflowResyncPeriod    = 20 * time.Minute
	cronWorkflowWorkers         = 2
	cronWorkflowWorkflowWorkers = 2
)

func NewCronController(
	wfclientset versioned.Interface,
	restConfig *rest.Config,
	namespace string,
	managedNamespace string,
	instanceId string,
	metrics *metrics.Metrics,
) *Controller {
	return &Controller{
		wfClientset:        wfclientset,
		namespace:          namespace,
		managedNamespace:   managedNamespace,
		instanceId:         instanceId,
		cron:               cron.New(),
		restConfig:         restConfig,
		nameEntryIDMap:     make(map[string]cron.EntryID),
		nameEntryIDMapLock: &sync.Mutex{},
		wfQueue:            workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "wf_cron_queue"),
		cronWfQueue:        workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "cron_wf_queue"),
		metrics:            metrics,
	}
}

func (cc *Controller) Run(ctx context.Context) {
	defer cc.cronWfQueue.ShutDown()
	defer cc.wfQueue.ShutDown()
	log.Infof("Starting CronWorkflow controller")
	if cc.instanceId != "" {
		log.Infof("...with InstanceID: %s", cc.instanceId)
	}

	cc.cronWfInformer = externalversions.NewSharedInformerFactoryWithOptions(cc.wfClientset, cronWorkflowResyncPeriod, externalversions.WithNamespace(cc.managedNamespace),
		externalversions.WithTweakListOptions(func(options *v1.ListOptions) {
			cronWfInformerListOptionsFunc(options, cc.instanceId)
		})).Argoproj().V1alpha1().CronWorkflows()
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
	log.Infof("Processing %s", key)

	obj, exists, err := cc.cronWfInformer.Informer().GetIndexer().GetByKey(key.(string))
	if err != nil {
		log.WithError(err).Error(fmt.Sprintf("Failed to get CronWorkflow '%s' from informer index", key))
		return true
	}
	cc.nameEntryIDMapLock.Lock()
	defer cc.nameEntryIDMapLock.Unlock()
	if !exists {
		if entryId, ok := cc.nameEntryIDMap[key.(string)]; ok {
			log.Infof("Deleting '%s'", key)
			cc.cron.Remove(entryId)
			delete(cc.nameEntryIDMap, key.(string))
		}
		return true
	}

	cronWf, ok := obj.(*v1alpha1.CronWorkflow)
	if !ok {
		log.Warnf("Key '%s' in index is not a CronWorkflow", key)
		return true
	}

	cronWorkflowOperationCtx := newCronWfOperationCtx(cronWf, cc.wfClientset, cc.wfLister, cc.metrics)

	err = cronWorkflowOperationCtx.validateCronWorkflow()
	if err != nil {
		log.WithError(err).Error("invalid cron workflow")
		return true
	}

	err = cronWorkflowOperationCtx.runOutstandingWorkflows()
	if err != nil {
		log.WithError(err).Error("could not run outstanding Workflow")
		return true
	}

	// The job is currently scheduled, remove it and re add it.
	if entryId, ok := cc.nameEntryIDMap[key.(string)]; ok {
		cc.cron.Remove(entryId)
		delete(cc.nameEntryIDMap, key.(string))
	}

	cronSchedule := cronWf.Spec.Schedule
	if cronWf.Spec.Timezone != "" {
		cronSchedule = "CRON_TZ=" + cronWf.Spec.Timezone + " " + cronSchedule
	}

	entryId, err := cc.cron.AddJob(cronSchedule, cronWorkflowOperationCtx)
	if err != nil {
		log.WithError(err).Error("could not schedule CronWorkflow")
		return true
	}
	cc.nameEntryIDMap[key.(string)] = entryId

	log.Infof("CronWorkflow %s added", key.(string))

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
	var woc *cronWfOperationCtx
	cc.nameEntryIDMapLock.Lock()
	defer cc.nameEntryIDMapLock.Unlock()
	if entryId, ok := cc.nameEntryIDMap[nameEntryIdMapKey]; ok {
		woc, ok = cc.cron.Entry(entryId).Job.(*cronWfOperationCtx)
		if !ok {
			log.Errorf("Parent CronWorkflow '%s' is malformed", nameEntryIdMapKey)
			return true
		}
	} else {
		log.Warnf("Parent CronWorkflow '%s' no longer exists", nameEntryIdMapKey)
		return true
	}

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
				key, err := cache.MetaNamespaceKeyFunc(obj)
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
