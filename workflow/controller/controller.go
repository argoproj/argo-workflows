package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	wfv1 "github.com/argoproj/argo/api/workflow/v1"
	"github.com/argoproj/argo/errors"
	workflowclient "github.com/argoproj/argo/workflow/client"
	"github.com/argoproj/argo/workflow/common"
	"github.com/ghodss/yaml"
	log "github.com/sirupsen/logrus"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
)

type WorkflowController struct {
	// ConfigMap is the name of the config map in which to derive configuration of the controller from
	ConfigMap string
	//WorkflowClient *workflowclient.WorkflowClient
	Config WorkflowControllerConfig

	restConfig *rest.Config
	restClient *rest.RESTClient
	clientset  *kubernetes.Clientset
	wfUpdates  chan *wfv1.Workflow
	podUpdates chan *apiv1.Pod
}

type WorkflowControllerConfig struct {
	ExecutorImage      string               `json:"executorImage,omitempty"`
	ArtifactRepository ArtifactRepository   `json:"artifactRepository,omitempty"`
	Namespace          string               `json:"namespace,omitempty"`
	Selector           metav1.LabelSelector `json:"selector,omitempty"`
}

// ArtifactRepository represents a artifact repository in which a controller will store its artifacts
type ArtifactRepository struct {
	S3 *S3ArtifactRepository `json:"s3,omitempty"`
	// Future artifact repository support here
}
type S3ArtifactRepository struct {
	wfv1.S3Bucket `json:",inline,squash"`

	// KeyPrefix is prefix used as part of the bucket key in which the controller will store artifacts.
	KeyPrefix string `json:"keyPrefix,omitempty"`
}

// NewWorkflowController instantiates a new WorkflowController
func NewWorkflowController(config *rest.Config, configMap string) *WorkflowController {
	// make a new config for our extension's API group, using the first config as a baseline
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err)
	}

	restClient, _, err := workflowclient.NewRESTClient(config)
	if err != nil {
		panic(err)
	}

	wfc := WorkflowController{
		restClient: restClient,
		restConfig: config,
		clientset:  clientset,
		ConfigMap:  configMap,
		wfUpdates:  make(chan *wfv1.Workflow, 1024),
		podUpdates: make(chan *apiv1.Pod, 1024),
	}
	return &wfc
}

// Run starts an Workflow resource controller
func (wfc *WorkflowController) Run(ctx context.Context) error {
	log.Info("Watch Workflow objects")

	// Watch Workflow objects
	_, err := wfc.watchWorkflows(ctx)
	if err != nil {
		log.Errorf("Failed to register watch for Workflow resource: %v", err)
		return err
	}

	// Watch pods related to workflows
	_, err = wfc.watchWorkflowPods(ctx)
	if err != nil {
		log.Errorf("Failed to register watch for Workflow resource: %v", err)
		return err
	}

	for {
		select {
		case wf := <-wfc.wfUpdates:
			log.Infof("Processing wf: %v", wf.ObjectMeta.SelfLink)
			wfc.operateWorkflow(wf)
		case pod := <-wfc.podUpdates:
			wfc.handlePodUpdate(pod)
		}
	}

	<-ctx.Done()
	return ctx.Err()
}

// ResyncConfig reloads the controller config from the configmap
func (wfc *WorkflowController) ResyncConfig() error {
	namespace, _ := os.LookupEnv(common.EnvVarNamespace)
	if namespace == "" {
		namespace = common.DefaultControllerNamespace
	}
	cmClient := wfc.clientset.CoreV1().ConfigMaps(namespace)
	cm, err := cmClient.Get(wfc.ConfigMap, metav1.GetOptions{})
	if err != nil {
		return errors.InternalWrapError(err)
	}
	configStr, ok := cm.Data[common.WorkflowControllerConfigMapKey]
	if !ok {
		return errors.Errorf(errors.CodeBadRequest, "ConfigMap '%s' does not have key '%s'", wfc.ConfigMap, common.WorkflowControllerConfigMapKey)
	}
	var config WorkflowControllerConfig
	err = yaml.Unmarshal([]byte(configStr), &config)
	if err != nil {
		return errors.InternalWrapError(err)
	}
	log.Printf("workflow controller configuration from %s:\n%s", wfc.ConfigMap, configStr)
	if wfc.Config.ExecutorImage == "" {
		wfc.Config.ExecutorImage = common.DefaultExecutorImage
	}
	wfc.Config = config
	return nil
}

func (wfc *WorkflowController) newWorkflowWatch() *cache.ListWatch {
	c := wfc.restClient
	resource := wfv1.CRDPlural
	namespace := wfc.Config.Namespace
	fieldSelector := fields.Everything()

	listFunc := func(options metav1.ListOptions) (runtime.Object, error) {
		options.FieldSelector = fieldSelector.String()
		return c.Get().
			Namespace(namespace).
			Resource(resource).
			VersionedParams(&options, metav1.ParameterCodec).
			Do().
			Get()
	}
	watchFunc := func(options metav1.ListOptions) (watch.Interface, error) {
		options.Watch = true
		options.FieldSelector = fieldSelector.String()
		return c.Get().
			Namespace(namespace).
			Resource(resource).
			VersionedParams(&options, metav1.ParameterCodec).
			Watch()
	}
	return &cache.ListWatch{ListFunc: listFunc, WatchFunc: watchFunc}
}

func (wfc *WorkflowController) watchWorkflows(ctx context.Context) (cache.Controller, error) {
	source := wfc.newWorkflowWatch()

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
				//log.Infof("WF Add %s", wf.ObjectMeta.SelfLink)
				wfc.wfUpdates <- wf
			},
			UpdateFunc: func(old, new interface{}) {
				//oldWf := old.(*wfv1.Workflow)
				newWf := new.(*wfv1.Workflow)
				//log.Infof("WF Update %s", newWf.ObjectMeta.SelfLink)
				wfc.wfUpdates <- newWf
			},
			DeleteFunc: func(obj interface{}) {
				wf := obj.(*wfv1.Workflow)
				//log.Infof("WF Delete %s", wf.ObjectMeta.SelfLink)
				wfc.wfUpdates <- wf
			},
		})

	go controller.Run(ctx.Done())
	return controller, nil
}

func (wfc *WorkflowController) newWorkflowPodWatch() *cache.ListWatch {
	c := wfc.clientset.Core().RESTClient()
	resource := "pods"
	namespace := wfc.Config.Namespace
	fieldSelector := fields.Everything()

	listFunc := func(options metav1.ListOptions) (runtime.Object, error) {
		options.FieldSelector = fieldSelector.String()
		return c.Get().
			Namespace(namespace).
			Resource(resource).
			Param("labelSelector", fmt.Sprintf("%s=true", common.LabelKeyArgoWorkflow)).
			VersionedParams(&options, metav1.ParameterCodec).
			Do().
			Get()
	}
	watchFunc := func(options metav1.ListOptions) (watch.Interface, error) {
		options.Watch = true
		options.FieldSelector = fieldSelector.String()
		return c.Get().
			Namespace(namespace).
			Resource(resource).
			Param("labelSelector", fmt.Sprintf("%s=true", common.LabelKeyArgoWorkflow)).
			VersionedParams(&options, metav1.ParameterCodec).
			Watch()
	}
	return &cache.ListWatch{ListFunc: listFunc, WatchFunc: watchFunc}
}

func (wfc *WorkflowController) watchWorkflowPods(ctx context.Context) (cache.Controller, error) {
	source := wfc.newWorkflowPodWatch()

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
				//log.Infof("Pod Added %s", pod.ObjectMeta.SelfLink)
				wfc.podUpdates <- pod
			},
			UpdateFunc: func(old, new interface{}) {
				//oldPod := old.(*apiv1.Pod)
				newPod := new.(*apiv1.Pod)
				//log.Infof("Pod Updated %s", newPod.ObjectMeta.SelfLink)
				wfc.podUpdates <- newPod
			},
			DeleteFunc: func(obj interface{}) {
				pod := obj.(*apiv1.Pod)
				//log.Infof("Pod Deleted %s", pod.ObjectMeta.SelfLink)
				wfc.podUpdates <- pod
			},
		})

	go controller.Run(ctx.Done())
	return controller, nil
}

// handlePodUpdate receives an update from a pod, and updates the status of the node in the workflow object accordingly
func (wfc *WorkflowController) handlePodUpdate(pod *apiv1.Pod) {
	workflowName, ok := pod.Labels[common.LabelKeyWorkflow]
	if !ok {
		// Ignore pods unrelated to workflow (this shouldn't happen unless the watch is setup incorrectly)
		return
	}
	var newStatus string
	var newDaemonStatus *bool
	switch pod.Status.Phase {
	case apiv1.PodPending:
		return
	case apiv1.PodSucceeded:
		newStatus = wfv1.NodeStatusSucceeded
		f := false
		newDaemonStatus = &f
	case apiv1.PodFailed:
		// TODO: we may need to distinguish between the main container suceeding and ignoring the sidekick
		// statuses. This is because executor may have had to forcibly kill a sidekick resulting in an
		// overall pod status as failed, but we really only care about the main container status.
		newStatus = wfv1.NodeStatusFailed
		f := false
		newDaemonStatus = &f
	case apiv1.PodRunning:
		tmplStr, ok := pod.Annotations[common.AnnotationKeyTemplate]
		if !ok {
			log.Warnf("%s missing template annotation", pod.ObjectMeta.Name)
			return
		}
		var tmpl wfv1.Template
		err := json.Unmarshal([]byte(tmplStr), &tmpl)
		if err != nil {
			log.Warnf("%s template annotation unreadable: %v", pod.ObjectMeta.Name, err)
			return
		}
		if tmpl.Daemon == nil || !*tmpl.Daemon {
			// incidental state change of a running pod. No need to inspect further
			return
		}
		// pod is running and template is marked daemon. check if everything is ready
		for _, ctrStatus := range pod.Status.ContainerStatuses {
			if !ctrStatus.Ready {
				return
			}
		}
		// proceed to mark node status as succeeded (and daemoned)
		newStatus = wfv1.NodeStatusSucceeded
		t := true
		newDaemonStatus = &t
		log.Infof("Processing ready daemon pod: %v", pod.ObjectMeta.SelfLink)
	default:
		log.Infof("Unexpected pod phase for %s: %s", pod.ObjectMeta.Name, pod.Status.Phase)
		newStatus = wfv1.NodeStatusError
	}

	wfClient := workflowclient.NewWorkflowClient(wfc.restClient, pod.ObjectMeta.Namespace)
	wf, err := wfClient.GetWorkflow(workflowName)
	if err != nil {
		log.Warnf("Failed to find workflow %s %+v", workflowName, err)
		return
	}
	node, ok := wf.Status.Nodes[pod.Name]
	if !ok {
		log.Warnf("pod %s unassociated with workflow %s", pod.Name, workflowName)
		return
	}
	updateNeeded := applyUpdates(pod, &node, newStatus, newDaemonStatus)
	if !updateNeeded {
		log.Infof("No workflow updated needed for node %s", node)
		return
	}
	//addOutputs(pod, &node)
	wf.Status.Nodes[pod.Name] = node
	_, err = wfClient.UpdateWorkflow(wf)
	if err != nil {
		log.Errorf("Failed to update %s status: %+v", pod.Name, err)
		// if we fail to update the CRD state, we will need to rely on resync to catch up
		return
	}
	log.Infof("Updated %s", node)
}

// applyUpdates applies any new state information about a pod, to the current status of the workflow node
// returns whether or not any updates were necessary (resulting in a update to the workflow)
func applyUpdates(pod *apiv1.Pod, node *wfv1.NodeStatus, newStatus string, newDaemonStatus *bool) bool {
	// Check various fields of the pods to see if we need to update the workflow
	updateNeeded := false
	if node.Status != newStatus {
		log.Infof("Updating node %s status %s -> %s", node, node.Status, newStatus)
		updateNeeded = true
		node.Status = newStatus
	}
	if pod.Status.PodIP != node.PodIP {
		log.Infof("Updating node %s IP %s -> %s", node, node.PodIP, pod.Status.PodIP)
		updateNeeded = true
		node.PodIP = pod.Status.PodIP
	}
	if newDaemonStatus != nil {
		if *newDaemonStatus == false {
			// if the daemon status switched to false, we prefer to just unset daemoned status field
			// (as opposed to setting it to false)
			newDaemonStatus = nil
		}
		if newDaemonStatus != nil && node.Daemoned == nil || newDaemonStatus == nil && node.Daemoned != nil {
			log.Infof("Setting node %v daemoned: %v -> %v", node, node.Daemoned, newDaemonStatus)
			node.Daemoned = newDaemonStatus
			updateNeeded = true
		}
	}
	outputStr, ok := pod.Annotations[common.AnnotationKeyOutputs]
	if ok && node.Outputs == nil {
		log.Infof("Setting node %v outputs", node)
		updateNeeded = true
		var outputs wfv1.Outputs
		err := json.Unmarshal([]byte(outputStr), &outputs)
		if err != nil {
			log.Errorf("Failed to unmarshal %s outputs from pod annotation: %v", pod.Name, err)
			node.Status = wfv1.NodeStatusError
		} else {
			node.Outputs = &outputs
		}
	}
	return updateNeeded
}
