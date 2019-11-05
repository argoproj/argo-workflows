package cron

import (
	"context"
	"github.com/argoproj/argo/pkg/client/clientset/versioned"
	"github.com/argoproj/argo/pkg/client/informers/externalversions"
	"github.com/argoproj/argo/pkg/client/informers/externalversions/workflow/v1alpha1"
	"github.com/robfig/cron"
	log "github.com/sirupsen/logrus"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"time"
)

// CronController is a controller for cron workflows
type CronController struct {
	cron           *cron.Cron
	namespace      string
	kubeclientset  kubernetes.Interface
	wfclientset    versioned.Interface
	wfcronInformer v1alpha1.CronWorkflowInformer
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
		kubeclientset: kubeclientset,
		wfclientset:   wfclientset,
		namespace:     namespace,
		cron:          cron.New(),
	}
}

func (cc *CronController) Run(ctx context.Context) {
	
	// Get outstanding CronWorkflows
	cc.wfcronInformer.Lister().CronWorkflows(cc.namespace).List(nil)

	cc.wfcronInformer = cc.newCronWorkflowInformer()
	cc.addCronWorkflowInformerHandler()

	go cc.wfcronInformer.Informer().Run(ctx.Done())

	<-ctx.Done()
}

func (cc *CronController) addCronWorkflowInformerHandler() {
	cc.wfcronInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			log.Infof("SIMON Adding object: %v", obj)
		},
		UpdateFunc: func(old, new interface{}) {
			log.Infof("SIMON Updating object: %v", new)
		},
		DeleteFunc: func(obj interface{}) {
			log.Infof("SIMON Deleting object: %v", obj)
		},
	})
}

func (cc *CronController) newCronWorkflowInformer() v1alpha1.CronWorkflowInformer {
	var informerFactory externalversions.SharedInformerFactory
	informerFactory = externalversions.NewSharedInformerFactory(cc.wfclientset, cronWorkflowResyncPeriod)
	return informerFactory.Argoproj().V1alpha1().CronWorkflows()
}
