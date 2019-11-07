package cron

import (
	"context"
	"fmt"
	"github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo/pkg/client/clientset/versioned"
	"github.com/argoproj/argo/pkg/client/informers/externalversions"
	extv1alpha1 "github.com/argoproj/argo/pkg/client/informers/externalversions/workflow/v1alpha1"
	"github.com/pkg/errors"
	"github.com/robfig/cron"
	log "github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"sync"
	"time"
)

// CronController is a controller for cron workflows
type CronController struct {
	namespace      string
	cron           *cron.Cron
	nameEntryIDMap map[string]cron.EntryID
	kubeclientset  kubernetes.Interface
	wfclientset    versioned.Interface
	wfcronInformer extv1alpha1.CronWorkflowInformer
	cronLock       sync.Mutex
}

const (
	cronWorkflowResyncPeriod = 20 * time.Minute
)

func NewCronController(
	kubeclientset kubernetes.Interface,
	wfclientset versioned.Interface,
	namespace string,
) *CronController {
	return &CronController{
		kubeclientset:  kubeclientset,
		wfclientset:    wfclientset,
		namespace:      namespace,
		cron:           cron.New(),
		nameEntryIDMap: make(map[string]cron.EntryID),
	}
}

func (cc *CronController) Run(ctx context.Context) {
	log.Infof("Starting CronWorkflow controller")

	cc.wfcronInformer = cc.newCronWorkflowInformer()
	cc.addCronWorkflowInformerHandler()

	// Get outstanding CronWorkflows
	err := cc.parseOutstandingCronWorkflows()
	if err != nil {
		log.Fatalf("Error: %s", err)
	}

	cc.cron.Start()
	defer cc.cron.Stop()

	go cc.wfcronInformer.Informer().Run(ctx.Done())

	<-ctx.Done()
}

func (cc *CronController) parseOutstandingCronWorkflows() error {
	log.Infof("Parsing outstanding CronWorkflows")

	cronWorkflows, err := cc.wfcronInformer.Lister().CronWorkflows(cc.namespace).List(labels.Everything())
	if err != nil {
		return errors.Wrap(err, "Error parsing existing CronWorkflow")
	}

	for _, cronWorkflow := range cronWorkflows {
		err := cc.startCronWorkflow(cronWorkflow)
		if err != nil {
			return errors.Wrap(err, "Error parsing existing CronWorkflow")
		}
	}
	return nil
}

func (cc *CronController) startCronWorkflow(cronWorkflow *v1alpha1.CronWorkflow) error {
	cc.cronLock.Lock()
	defer cc.cronLock.Unlock()

	log.Infof("Parsing CronWorkflow %s", cronWorkflow.Name)

	if entryId, ok := cc.nameEntryIDMap[cronWorkflow.Name]; ok {
		// The job is currently scheduled, remove it and re add it.
		cc.cron.Remove(entryId)
		delete(cc.nameEntryIDMap, cronWorkflow.Name)
	}
	// TODO: this is mostly a place holder. This is most likely not how/where we will be running the workflows
	entryId, err := cc.cron.AddFunc(cronWorkflow.Options.Schedule, func() { log.Infof("Would have run %s", cronWorkflow.Name) })
	if err != nil {
		return errors.Wrap(err, "Unable to add CronWorkflow")
	}
	cc.nameEntryIDMap[cronWorkflow.Name] = entryId

	log.Infof("CronWorkflow %s added", cronWorkflow.Name)

	log.Infof("SIMON Entries %v", cc.cron.Entries())
	log.Infof("SIMON Entry next %s", cc.cron.Entry(entryId).Next)

	return nil
}

func (cc *CronController) stopCronWorkflow(cronWorkflow *v1alpha1.CronWorkflow) error {
	cc.cronLock.Lock()
	defer cc.cronLock.Unlock()

	entryId, ok := cc.nameEntryIDMap[cronWorkflow.Name]
	if !ok {
		return fmt.Errorf("unable to remove workflow: workflow %s does not exist", cronWorkflow.Name)
	}

	cc.cron.Remove(entryId)
	delete(cc.nameEntryIDMap, cronWorkflow.Name)

	log.Infof("CronWorkflow %s removed", cronWorkflow.Name)
	return nil


}

func (cc *CronController) addCronWorkflowInformerHandler() {
	cc.wfcronInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			log.Infof("SIMON Adding object: %v", obj)
			cronWf, err := convertToWorkflow(obj)
			if err != nil {
				log.Error(err)
			}
			err = cc.startCronWorkflow(cronWf)
			if err != nil {
				log.Errorf("Error starting CronWorkflow %s: %s", cronWf.Name, err)
				return
			}
		},
		UpdateFunc: func(old, new interface{}) {
			log.Infof("SIMON Updating object: %v", new)
			cronWf, err := convertToWorkflow(new)
			if err != nil {
				log.Error(err)
			}
			err = cc.startCronWorkflow(cronWf)
			if err != nil {
				log.Errorf("Error starting CronWorkflow %s: %s", cronWf.Name, err)
				return
			}
		},
		DeleteFunc: func(obj interface{}) {
			log.Infof("SIMON Deleting object: %v", obj)
			cronWf, err := convertToWorkflow(obj)
			if err != nil {
				log.Error(err)
			}
			err = cc.stopCronWorkflow(cronWf)
			if err != nil {
				log.Errorf("Error stopping CronWorkflow %s: %s", cronWf.Name, err)
				return
			}
		},
	})
}

func (cc *CronController) newCronWorkflowInformer() extv1alpha1.CronWorkflowInformer {
	var informerFactory externalversions.SharedInformerFactory
	informerFactory = externalversions.NewSharedInformerFactory(cc.wfclientset, cronWorkflowResyncPeriod)
	return informerFactory.Argoproj().V1alpha1().CronWorkflows()
}

func convertToWorkflow(obj interface{}) (*v1alpha1.CronWorkflow, error) {
	cronWf, ok := obj.(*v1alpha1.CronWorkflow)
	if !ok {
		return nil, fmt.Errorf("error casting object")
	}
	return cronWf, nil
}
