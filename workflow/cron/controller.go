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
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	"math/rand"
	"sync"
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
	cronLock       sync.Mutex
}

const (
	cronWorkflowResyncPeriod = 20 * time.Minute
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
		wfQueue: workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter()),
		cronWfQueue: workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter()),
	}
}

func (cc *Controller) Run(ctx context.Context) {
	defer cc.cronWfQueue.ShutDown()
	log.Infof("Starting CronWorkflow controller")

	cc.cronWfInformer = cc.newCronWorkflowInformer()
	cc.addCronWorkflowInformerHandler()

	cc.wfInformer = util.NewWorkflowInformer(cc.restConfig, "", cronWorkflowResyncPeriod, wfInformerListOptionsFunc)
	log.Infof("SIMON CRON WFINF: %v", cc.wfInformer)
	cc.addWorkflowInformerHandler()

	cc.cron.Start()
	defer cc.cron.Stop()

	go cc.cronWfInformer.Informer().Run(ctx.Done())
	go cc.wfInformer.Run(ctx.Done())

	for i := 0; i < 8; i++ {
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

	cronWfIf := cc.wfClientset.ArgoprojV1alpha1().CronWorkflows(cc.namespace)
	cronWorkflowOperationCtx, err := newCronWfOperationCtx(cronWf, cc.wfClientset, cronWfIf)
	if err != nil {
		log.Error(err)
		return false
	}

	err = cronWorkflowOperationCtx.runOutstandingWorkflows()
	if err != nil {
		log.Errorf("could not run outstanding Workflow: %w", err)
		return false
	}

	// The job is currently scheduled, remove it and re add it.
	if entryId, ok := cc.nameEntryIDMap[key.(string)]; ok {
		cc.cron.Remove(entryId)
		delete(cc.nameEntryIDMap, key.(string))
	}

	entryId, err := cc.cron.AddJob(cronWf.Options.Schedule, cronWorkflowOperationCtx)
	if err != nil {
		log.Errorf("could not schedule CronWorkflow: %w", err)
		return false
	}
	cc.nameEntryIDMap[key.(string)] = entryId

	log.Infof("CronWorkflow %s added", key.(string))

	return false
}

func (cc *Controller) newCronWorkflowInformer() extv1alpha1.CronWorkflowInformer {
	var informerFactory externalversions.SharedInformerFactory
	informerFactory = externalversions.NewSharedInformerFactory(cc.wfClientset, cronWorkflowResyncPeriod)
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
			},
			UpdateFunc: func(old, new interface{}) {
			},
			DeleteFunc: func(obj interface{}) {
			},
		},
	)
}

const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

func RandStringBytes(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = letterBytes[rand.Intn(len(letterBytes))]
	}
	return string(b)
}

// The equivalent function for Workflows lives in util.go. If necessary, this could be moved there
func fromUnstructured(obj *unstructured.Unstructured) (*v1alpha1.CronWorkflow, error) {
	var cronWf v1alpha1.CronWorkflow
	err := runtime.DefaultUnstructuredConverter.FromUnstructured(obj.Object, &cronWf)
	return &cronWf, err
}

func wfInformerListOptionsFunc(options *v1.ListOptions) {
	options.FieldSelector = fields.Everything().String()
	incompleteReq, err := labels.NewRequirement(common.LabelKeyCompleted, selection.NotIn, []string{"true"})
	if err != nil {
		panic(err)
	}
	isCronWorkflowChildReq, err := labels.NewRequirement(common.LabelCronWorkflowParent, selection.Exists, []string{})
	if err != nil {
		panic(err)
	}
	labelSelector := labels.NewSelector().Add(*incompleteReq, *isCronWorkflowChildReq)
	options.LabelSelector = labelSelector.String()
}
