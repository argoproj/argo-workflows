package controller

import (
	"context"
	"fmt"

	wfv1 "github.com/argoproj/argo/api/workflow/v1"
	workflowclient "github.com/argoproj/argo/workflow/client"
	apiv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
)

type WorkflowController struct {
	WorkflowClient *workflowclient.WorkflowClient
	WorkflowScheme *runtime.Scheme
	podCl          corev1.PodInterface
}

// NewWorkflowController instantiates a new WorkflowController
func NewWorkflowController(config *rest.Config) *WorkflowController {
	// make a new config for our extension's API group, using the first config as a baseline
	wfClient, wfScheme, err := workflowclient.NewClient(config)
	if err != nil {
		panic(err)
	}

	k8client, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err)
	}

	wfc := WorkflowController{
		WorkflowClient: wfClient,
		WorkflowScheme: wfScheme,
		podCl:          k8client.CoreV1().Pods(apiv1.NamespaceDefault),
	}
	return &wfc
}

// Run starts an Workflow resource controller
func (wfc *WorkflowController) Run(ctx context.Context) error {
	fmt.Print("Watch Workflow objects\n")

	// Watch Workflow objects
	_, err := wfc.watchWorkflows(ctx)
	if err != nil {
		fmt.Printf("Failed to register watch for Workflow resource: %v\n", err)
		return err
	}

	<-ctx.Done()
	return ctx.Err()
}

func (wfc *WorkflowController) watchWorkflows(ctx context.Context) (cache.Controller, error) {
	source := wfc.WorkflowClient.NewListWatch()

	_, controller := cache.NewInformer(
		source,

		// The object type.
		&wfv1.Workflow{},

		// resyncPeriod
		// Every resyncPeriod, all resources in the cache will retrigger events.
		// Set to 0 to disable the resync.
		0,

		// Your custom resource event handlers.
		cache.ResourceEventHandlerFuncs{
			AddFunc:    wfc.onAdd,
			UpdateFunc: wfc.onUpdate,
			DeleteFunc: wfc.onDelete,
		})

	go controller.Run(ctx.Done())
	return controller, nil
}

func (wfc *WorkflowController) onAdd(obj interface{}) {
	wf := obj.(*wfv1.Workflow)
	fmt.Printf("[CONTROLLER] OnAdd %s\n", wf.ObjectMeta.SelfLink)
	go wfc.simulateRun(wfc.WorkflowClient, wf)
}

func (wfc *WorkflowController) onDelete(obj interface{}) {
	wf := obj.(*wfv1.Workflow)
	fmt.Printf("[CONTROLLER] Delete %s\n", wf.ObjectMeta.SelfLink)
}

func (wfc *WorkflowController) onUpdate(old, new interface{}) {
	oldWf := old.(*wfv1.Workflow)
	newWf := new.(*wfv1.Workflow)
	fmt.Printf("[CONTROLLER] Update\nOld: %v \nNew: %v\n", oldWf, newWf)
}
