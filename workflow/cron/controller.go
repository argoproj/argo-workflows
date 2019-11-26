package cron

import (
	"context"
	"github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo/pkg/client/clientset/versioned"
	"github.com/argoproj/argo/pkg/client/informers/externalversions"
	extv1alpha1 "github.com/argoproj/argo/pkg/client/informers/externalversions/workflow/v1alpha1"
	"github.com/argoproj/argo/workflow/common"
	"github.com/argoproj/argo/workflow/util"
	"github.com/robfig/cron"
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
	"time"
)

// Controller is a controller for cron workflows
type Controller struct {
	namespace      string
	cron           *cron.Cron
	nameEntryIDMap map[string]cron.EntryID
	wfClientset    versioned.Interface
	wfInformer     cache.SharedIndexInformer
	wfQueue        workqueue.RateLimitingInterface
	cronWfInformer extv1alpha1.CronWorkflowInformer
	cronWfQueue    workqueue.RateLimitingInterface
	restConfig     *rest.Config
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
) *Controller {
	return &Controller{
		wfClientset:    wfclientset,
		namespace:      namespace,
		cron:           cron.New(),
		restConfig:     restConfig,
		nameEntryIDMap: make(map[string]cron.EntryID),
		wfQueue:        workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter()),
		cronWfQueue:    workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter()),
	}
}

func (cc *Controller) Run(ctx context.Context) {
	defer cc.cronWfQueue.ShutDown()
	defer cc.wfQueue.ShutDown()
	log.Infof("Starting CronWorkflow controller")

	cc.cronWfInformer = cc.newCronWorkflowInformer()
	cc.addCronWorkflowInformerHandler()

	cc.wfInformer = util.NewWorkflowInformer(cc.restConfig, "", cronWorkflowResyncPeriod, wfInformerListOptionsFunc)
	cc.addWorkflowInformerHandler()

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
		log.Errorf("Failed to get CronWorkflow '%s' from informer index: %+v", key, err)
		return true
	}
	if !exists {
		if entryId, ok := cc.nameEntryIDMap[key.(string)]; ok {
			log.Infof("Deleting '%s'", key)
			cc.cron.Remove(entryId)
			delete(cc.nameEntryIDMap, key.(string))
		}
		return true
	}

	// The workflow informer receives unstructured objects to deal with the possibility of invalid
	// workflow manifests that are unable to unmarshal to workflow objects
	cronWf, ok := obj.(*v1alpha1.CronWorkflow)
	if !ok {
		log.Warnf("Key '%s' in index is not a CronWorkflow", key)
		return true
	}

	cronWfIf := cc.wfClientset.ArgoprojV1alpha1().CronWorkflows(cronWf.Namespace)
	cronWorkflowOperationCtx, err := newCronWfOperationCtx(cronWf, cc.wfClientset, cronWfIf)
	if err != nil {
		log.Error(err)
		return true
	}

	err = cronWorkflowOperationCtx.runOutstandingWorkflows()
	if err != nil {
		log.Errorf("could not run outstanding Workflow: %s", err)
		return true
	}

	// The job is currently scheduled, remove it and re add it.
	if entryId, ok := cc.nameEntryIDMap[key.(string)]; ok {
		cc.cron.Remove(entryId)
		delete(cc.nameEntryIDMap, key.(string))
	}

	entryId, err := cc.cron.AddJob(cronWf.Spec.Schedule, cronWorkflowOperationCtx)
	if err != nil {
		log.Errorf("could not schedule CronWorkflow: %s", err)
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
		log.Errorf("Failed to get Workflow '%s' from informer index: %+v", key, err)
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
	if entryId, ok := cc.nameEntryIDMap[nameEntryIdMapKey]; ok {
		woc, ok = cc.cron.Entry(entryId).Job.(*cronWfOperationCtx)
		if !ok {
			log.Errorf("Parent CronWorkflow '%s' is malformed", nameEntryIdMapKey)
			return true
		}
	} else {
		log.Errorf("Parent CronWorkflow '%s' no longer exists", nameEntryIdMapKey)
		return true
	}

	// If the workflow is completed or was deleted, remove it from Active Workflows
	if wf.Status.Completed() || !wfExists {
		log.Warnf("Workflow '%s' from CronWorkflow '%s' completed", wf.Name, woc.cronWf.Name)
		woc.removeActiveWf(wf)
	}
	return true
}

func (cc *Controller) newCronWorkflowInformer() extv1alpha1.CronWorkflowInformer {
	informerFactory := externalversions.NewSharedInformerFactory(cc.wfClientset, cronWorkflowResyncPeriod)
	return informerFactory.Argoproj().V1alpha1().CronWorkflows()
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

func wfInformerListOptionsFunc(options *v1.ListOptions) {
	options.FieldSelector = fields.Everything().String()
	isCronWorkflowChildReq, err := labels.NewRequirement(common.LabelCronWorkflow, selection.Equals, []string{"true"})
	if err != nil {
		panic(err)
	}
	labelSelector := labels.NewSelector().Add(*isCronWorkflowChildReq)
	options.LabelSelector = labelSelector.String()
}
