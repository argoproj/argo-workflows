package controller

import (
	"context"
	"fmt"

	wfv1 "github.com/argoproj/argo/api/workflow/v1"
	workflowclient "github.com/argoproj/argo/workflow/client"
	apiv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
)

type WorkflowController struct {
	clientset      *kubernetes.Clientset
	WorkflowClient *workflowclient.WorkflowClient
	WorkflowScheme *runtime.Scheme
	podCl          corev1.PodInterface
	podClient      *rest.RESTClient
	wfUpdates      chan *wfv1.Workflow
	podUpdates     chan *apiv1.Pod
}

// NewWorkflowController instantiates a new WorkflowController
func NewWorkflowController(config *rest.Config) *WorkflowController {
	// make a new config for our extension's API group, using the first config as a baseline

	wfClient, wfScheme, err := workflowclient.NewClient(config)
	if err != nil {
		panic(err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err)
	}

	wfc := WorkflowController{
		clientset:      clientset,
		WorkflowClient: wfClient,
		WorkflowScheme: wfScheme,
		podCl:          clientset.CoreV1().Pods(apiv1.NamespaceDefault),
		//podClient:      newPodClient(config),
		wfUpdates:  make(chan *wfv1.Workflow),
		podUpdates: make(chan *apiv1.Pod),
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

	// Watch pods related to workflows
	_, err = wfc.watchWorkflowPods(ctx)
	if err != nil {
		fmt.Printf("Failed to register watch for Workflow resource: %v\n", err)
		return err
	}

	for {
		select {
		case wf := <-wfc.wfUpdates:
			fmt.Printf("Processing wf: %v\n", wf.ObjectMeta.SelfLink)
			wfc.operateWorkflow(wf)
		case pod := <-wfc.podUpdates:
			if pod.Status.Phase != "Succeeded" && pod.Status.Phase != "Failed" {
				continue
			}
			fmt.Printf("Processing completed pod: %v\n", pod.ObjectMeta.SelfLink)
			workflowName, ok := pod.Labels["workflow"]
			if !ok {
				continue
			}
			wf, err := wfc.WorkflowClient.GetWorkflow(workflowName)
			if err != nil {
				fmt.Printf("Failed to find workflow %s %+v\n", workflowName, err)
				continue
			}
			node, ok := wf.Status.Nodes[pod.Name]
			if !ok {
				fmt.Printf("pod %s unassociated with workflow %s", pod.Name, workflowName)
				continue
			}
			if string(pod.Status.Phase) == node.Status {
				fmt.Printf("pod %s already marked %s\n", pod.Name, node.Status)
				continue
			}
			fmt.Printf("Updating pod %s status %s -> %s\n", pod.Name, node.Status, pod.Status.Phase)
			node.Status = string(pod.Status.Phase)
			wf.Status.Nodes[pod.Name] = node
			_, err = wfc.WorkflowClient.UpdateWorkflow(wf)
			if err != nil {
				fmt.Printf("Failed to update %s status: %+v\n", pod.Name, err)
			}
			fmt.Printf("Updated %v\n", wf.Status.Nodes)
		}
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
			AddFunc: func(obj interface{}) {
				wf := obj.(*wfv1.Workflow)
				fmt.Printf("[CONTROLLER] WF Add %s\n", wf.ObjectMeta.SelfLink)
				wfc.wfUpdates <- wf
			},
			UpdateFunc: func(old, new interface{}) {
				//oldWf := old.(*wfv1.Workflow)
				newWf := new.(*wfv1.Workflow)
				fmt.Printf("[CONTROLLER] WF Update %s\n\n", newWf.ObjectMeta.SelfLink)
				wfc.wfUpdates <- newWf
			},
			DeleteFunc: func(obj interface{}) {
				wf := obj.(*wfv1.Workflow)
				fmt.Printf("[CONTROLLER] WF Delete %s\n", wf.ObjectMeta.SelfLink)
				wfc.wfUpdates <- wf
			},
		})

	go controller.Run(ctx.Done())
	return controller, nil
}

func (wfc *WorkflowController) watchWorkflowPods(ctx context.Context) (cache.Controller, error) {
	source := cache.NewListWatchFromClient(
		wfc.clientset.Core().RESTClient(),
		"pods",
		apiv1.NamespaceDefault,
		fields.Everything(),
	)

	_, controller := cache.NewInformer(
		source,

		// The object type.
		&apiv1.Pod{},

		// resyncPeriod
		// Every resyncPeriod, all resources in the cache will retrigger events.
		// Set to 0 to disable the resync.
		0,

		// Your custom resource event handlers.
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				pod := obj.(*apiv1.Pod)
				fmt.Printf("[CONTROLLER] Pod Added%s\n", pod.ObjectMeta.SelfLink)
				wfc.podUpdates <- pod
			},
			UpdateFunc: func(old, new interface{}) {
				//oldPod := old.(*apiv1.Pod)
				newPod := new.(*apiv1.Pod)
				fmt.Printf("[CONTROLLER] Pod Updated %s\n", newPod.ObjectMeta.SelfLink)
				wfc.podUpdates <- newPod
			},
			DeleteFunc: func(obj interface{}) {
				pod := obj.(*apiv1.Pod)
				fmt.Printf("[CONTROLLER] Pod Deleted%s\n", pod.ObjectMeta.SelfLink)
				wfc.podUpdates <- pod
			},
		})

	go controller.Run(ctx.Done())
	return controller, nil
}
