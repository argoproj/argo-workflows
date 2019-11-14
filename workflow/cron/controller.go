package cron

import (
	"context"
	"fmt"
	"github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo/pkg/client/clientset/versioned"
	"github.com/argoproj/argo/pkg/client/informers/externalversions"
	extv1alpha1 "github.com/argoproj/argo/pkg/client/informers/externalversions/workflow/v1alpha1"
	"github.com/argoproj/argo/workflow/common"
	"github.com/argoproj/argo/workflow/util"
	"github.com/robfig/cron"
	log "github.com/sirupsen/logrus"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
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
	wfCronInformer extv1alpha1.CronWorkflowInformer
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
	}
}

func (cc *Controller) Run(ctx context.Context) {
	log.Infof("Starting CronWorkflow controller")

	cc.wfCronInformer = cc.newCronWorkflowInformer()
	cc.addCronWorkflowInformerHandler()

	cc.wfInformer = util.NewWorkflowInformer(cc.restConfig, "", cronWorkflowResyncPeriod, wfInformerListOptionsFunc)
	cc.addWorkflowInformerHandler()

	cc.cron.Start()
	defer cc.cron.Stop()

	go cc.wfCronInformer.Informer().Run(ctx.Done())

	<-ctx.Done()
}

func (cc *Controller) startCronWorkflow(cronWorkflow *v1alpha1.CronWorkflow) error {
	cc.cronLock.Lock()
	defer cc.cronLock.Unlock()

	log.Infof("Parsing CronWorkflow %s", cronWorkflow.Name)

	if entryId, ok := cc.nameEntryIDMap[cronWorkflow.Name]; ok {
		// The job is currently scheduled, remove it and re add it.
		cc.cron.Remove(entryId)
		delete(cc.nameEntryIDMap, cronWorkflow.Name)
	}

	cronWfIf := cc.wfClientset.ArgoprojV1alpha1().CronWorkflows(cc.namespace)
	cronWorkflowJob, err := NewCronWorkflowWrapper(cronWorkflow, cc.wfClientset, cronWfIf)
	if err != nil {
		return err
	}
	cronSchedule, err := cron.ParseStandard(cronWorkflow.Options.Schedule)
	if err != nil {
		return fmt.Errorf("could not parse schedule '%s': %w", cronWorkflow.Options.Schedule, err)
	}

	runWorkflowIfMissed(cronWorkflow, cronSchedule, cronWorkflowJob)

	entryId := cc.cron.Schedule(cronSchedule, cronWorkflowJob)
	cc.nameEntryIDMap[cronWorkflow.Name] = entryId

	log.Infof("CronWorkflow %s added", cronWorkflow.Name)
	return nil
}

func runWorkflowIfMissed(cronWorkflow *v1alpha1.CronWorkflow, cronSchedule cron.Schedule, cronWorkflowJob *CronWorkflowWrapper) {
	// If this CronWorkflow has been run before, check if we have missed any scheduled executions
	if cronWorkflow.Status.LastScheduledTime != nil {
		now := time.Now()
		var missedExecutionTime time.Time
		nextScheduledRunTime := cronSchedule.Next(cronWorkflow.Status.LastScheduledTime.Time)
		// Workflow should have ran
		for nextScheduledRunTime.Before(now) {
			missedExecutionTime = nextScheduledRunTime
			nextScheduledRunTime = cronSchedule.Next(missedExecutionTime)
		}
		// We missed the latest execution time
		if !missedExecutionTime.IsZero() {
			// If StartingDeadlineSeconds is not set, or we are still within the deadline window, run the Workflow
			if cronWorkflow.Options.StartingDeadlineSeconds == nil || now.Before(missedExecutionTime.Add(time.Duration(*cronWorkflow.Options.StartingDeadlineSeconds)*time.Second)) {
				log.Infof("%s missed an execution at %s and is within StartingDeadline", cronWorkflow.Name, missedExecutionTime.Format("Mon Jan _2 15:04:05 2006"))
				cronWorkflowJob.Run()
			}
		}
	}
}

func (cc *Controller) stopCronWorkflow(cronWorkflowName string) error {
	cc.cronLock.Lock()
	defer cc.cronLock.Unlock()

	entryId, ok := cc.nameEntryIDMap[cronWorkflowName]
	if !ok {
		return fmt.Errorf("unable to remove workflow: workflow %s does not exist", cronWorkflowName)
	}

	cc.cron.Remove(entryId)
	delete(cc.nameEntryIDMap, cronWorkflowName)

	log.Infof("CronWorkflow %s removed", cronWorkflowName)
	return nil
}

func (cc *Controller) addCronWorkflowInformerHandler() {
	cc.wfCronInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			cronWf, err := convertToWorkflow(obj)
			if err != nil {
				log.Error(err)
				return
			}
			err = cc.startCronWorkflow(cronWf)
			if err != nil {
				log.Errorf("Error starting CronWorkflow %s: %s", cronWf.Name, err)
			}
		},
		UpdateFunc: func(old, new interface{}) {
			cronWf, err := convertToWorkflow(new)
			if err != nil {
				log.Error(err)
				return
			}
			err = cc.startCronWorkflow(cronWf)
			if err != nil {
				log.Errorf("Error starting CronWorkflow %s: %s", cronWf.Name, err)
			}
		},
		DeleteFunc: func(obj interface{}) {
			log.Infof("SIMON Deleting object: %v", obj)
			cronWf, err := convertToWorkflow(obj)
			if err != nil {
				log.Error(err)
				return
			}
			err = cc.stopCronWorkflow(cronWf.Name)
			if err != nil {
				log.Errorf("Error stopping CronWorkflow %s: %s", cronWf.Name, err)
			}
		},
	})
}

func (cc *Controller) addWorkflowInformerHandler() {
	log.Infof("SIMON adding informer")
	cc.wfInformer.AddEventHandler(
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				log.Infof("SIMON FOUND A WF: %s", obj.(v1alpha1.Workflow).Name)
			},
			UpdateFunc: func(old, new interface{}) {
				log.Infof("SIMON FOUND U WF: %s", new.(v1alpha1.Workflow).Name)

			},
			DeleteFunc: func(obj interface{}) {
				log.Infof("SIMON FOUND D WF: %s", obj.(v1alpha1.Workflow).Name)
			},
		},
	)
}

func (cc *Controller) newCronWorkflowInformer() extv1alpha1.CronWorkflowInformer {
	var informerFactory externalversions.SharedInformerFactory
	informerFactory = externalversions.NewSharedInformerFactory(cc.wfClientset, cronWorkflowResyncPeriod)
	return informerFactory.Argoproj().V1alpha1().CronWorkflows()
}

func convertToWorkflow(obj interface{}) (*v1alpha1.CronWorkflow, error) {
	cronWf, ok := obj.(*v1alpha1.CronWorkflow)
	if !ok {
		return nil, fmt.Errorf("error casting object")
	}
	return cronWf, nil
}

func wfInformerListOptionsFunc(options *v1.ListOptions) {
	options.FieldSelector = fields.Everything().String()
	incompleteReq, err := labels.NewRequirement(common.LabelKeyCompleted, selection.NotIn, []string{"true"})
	if err != nil {
		panic(err)
	}
	labelSelector := labels.NewSelector().Add(*incompleteReq)
	options.LabelSelector = labelSelector.String()
}
