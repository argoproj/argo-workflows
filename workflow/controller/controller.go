package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	wfv1 "github.com/argoproj/argo/api/workflow/v1alpha1"
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
	// namespace for config map
	ConfigMapNS string
	//WorkflowClient *workflowclient.WorkflowClient
	Config WorkflowControllerConfig

	restConfig *rest.Config
	restClient *rest.RESTClient
	scheme     *runtime.Scheme
	clientset  *kubernetes.Clientset
	wfUpdates  chan *wfv1.Workflow
	podUpdates chan *apiv1.Pod
}

type WorkflowControllerConfig struct {
	ExecutorImage      string             `json:"executorImage,omitempty"`
	ArtifactRepository ArtifactRepository `json:"artifactRepository,omitempty"`
	Namespace          string             `json:"namespace,omitempty"`
	MatchLabels        map[string]string  `json:"matchLabels,omitempty"`
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

	restClient, scheme, err := workflowclient.NewRESTClient(config)
	if err != nil {
		panic(err)
	}

	wfc := WorkflowController{
		restClient: restClient,
		restConfig: config,
		clientset:  clientset,
		scheme:     scheme,
		ConfigMap:  configMap,
		wfUpdates:  make(chan *wfv1.Workflow, 1024),
		podUpdates: make(chan *apiv1.Pod, 1024),
	}
	return &wfc
}

// Run starts an Workflow resource controller
func (wfc *WorkflowController) Run(ctx context.Context) error {

	log.Info("Watch Workflow controller config map updates")
	_, err := wfc.watchControllerConfigMap(ctx)
	if err != nil {
		log.Errorf("Failed to register watch for controller config map: %v", err)
		return err
	}

	log.Info("Watch Workflow objects")

	// Watch Workflow objects
	_, err = wfc.watchWorkflows(ctx)
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
	wfc.ConfigMapNS = cm.Namespace
	return wfc.updateConfig(cm)
}

func (wfc *WorkflowController) updateConfig(cm *apiv1.ConfigMap) error {
	configStr, ok := cm.Data[common.WorkflowControllerConfigMapKey]
	if !ok {
		return errors.Errorf(errors.CodeBadRequest, "ConfigMap '%s' does not have key '%s'", wfc.ConfigMap, common.WorkflowControllerConfigMapKey)
	}
	var config WorkflowControllerConfig
	err := yaml.Unmarshal([]byte(configStr), &config)
	if err != nil {
		return errors.InternalWrapError(err)
	}
	log.Printf("workflow controller configuration from %s:\n%s", wfc.ConfigMap, configStr)
	if config.ExecutorImage == "" {
		return errors.Errorf(errors.CodeBadRequest, "ConfigMap '%s' does not have executorImage", wfc.ConfigMap)
	}
	wfc.Config = config
	return nil
}

// addLabelSelectors adds label selectors from the workflow controller's config
func (wfc *WorkflowController) addLabelSelectors(req *rest.Request) *rest.Request {
	for label, labelVal := range wfc.Config.MatchLabels {
		req = req.Param("labelSelector", fmt.Sprintf("%s=%s", label, labelVal))
	}
	return req
}

func (wfc *WorkflowController) newWorkflowWatch() *cache.ListWatch {
	c := wfc.restClient
	resource := wfv1.CRDPlural
	namespace := wfc.Config.Namespace
	fieldSelector := fields.Everything()

	listFunc := func(options metav1.ListOptions) (runtime.Object, error) {
		options.FieldSelector = fieldSelector.String()
		req := c.Get().
			Namespace(namespace).
			Resource(resource).
			Param("labelSelector", fmt.Sprintf("%s notin (true)", common.LabelKeyCompleted)).
			VersionedParams(&options, metav1.ParameterCodec)
		req = wfc.addLabelSelectors(req)
		return req.Do().Get()
	}
	watchFunc := func(options metav1.ListOptions) (watch.Interface, error) {
		options.Watch = true
		options.FieldSelector = fieldSelector.String()
		req := c.Get().
			Namespace(namespace).
			Resource(resource).
			Param("labelSelector", fmt.Sprintf("%s notin (true)", common.LabelKeyCompleted)).
			VersionedParams(&options, metav1.ParameterCodec)
		req = wfc.addLabelSelectors(req)
		return req.Watch()
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

func (wfc *WorkflowController) watchControllerConfigMap(ctx context.Context) (cache.Controller, error) {
	source := wfc.newControllerConfigMapWatch()

	_, controller := cache.NewInformer(
		source,

		// The object type.
		&apiv1.ConfigMap{},

		// resyncPeriod
		// Every resyncPeriod, all resources in the cache will retrigger events.
		// Set to 0 to disable the resync.
		0,

		// Your custom resource event handlers.
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				cm := obj.(*apiv1.ConfigMap)
				log.Infof("Detected ConfigMap update. Updating the controller config.")
				err := wfc.updateConfig(cm)
				if err != nil {
					log.Errorf("Update of config failed due to: %v", err)
				}

			},
			UpdateFunc: func(old, new interface{}) {
				newCm := new.(*apiv1.ConfigMap)
				log.Infof("Detected ConfigMap update. Updating the controller config.")
				err := wfc.updateConfig(newCm)
				if err != nil {
					log.Errorf("Update of config failed due to: %v", err)
				}
			},
		})

	go controller.Run(ctx.Done())
	return controller, nil
}

func (wfc *WorkflowController) newControllerConfigMapWatch() *cache.ListWatch {
	c := wfc.clientset.Core().RESTClient()
	resource := "configmaps"
	name := wfc.ConfigMap
	namespace := wfc.ConfigMapNS

	listFunc := func(options metav1.ListOptions) (runtime.Object, error) {
		req := c.Get().
			Namespace(namespace).
			Resource(resource).
			Param("fieldSelector", fmt.Sprintf("metadata.name=%s", name)).
			VersionedParams(&options, metav1.ParameterCodec)
		return req.Do().Get()
	}
	watchFunc := func(options metav1.ListOptions) (watch.Interface, error) {
		options.Watch = true
		req := c.Get().
			Namespace(namespace).
			Resource(resource).
			Param("fieldSelector", fmt.Sprintf("metadata.name=%s", name)).
			VersionedParams(&options, metav1.ParameterCodec)
		return req.Watch()
	}
	return &cache.ListWatch{ListFunc: listFunc, WatchFunc: watchFunc}
}

func (wfc *WorkflowController) newWorkflowPodWatch() *cache.ListWatch {
	c := wfc.clientset.Core().RESTClient()
	resource := "pods"
	namespace := wfc.Config.Namespace
	fieldSelector := fields.Everything()

	listFunc := func(options metav1.ListOptions) (runtime.Object, error) {
		options.FieldSelector = fieldSelector.String()
		req := c.Get().
			Namespace(namespace).
			Resource(resource).
			Param("labelSelector", fmt.Sprintf("%s=false", common.LabelKeyCompleted)).
			VersionedParams(&options, metav1.ParameterCodec)
		req = wfc.addLabelSelectors(req)
		return req.Do().Get()
	}
	watchFunc := func(options metav1.ListOptions) (watch.Interface, error) {
		options.Watch = true
		options.FieldSelector = fieldSelector.String()
		req := c.Get().
			Namespace(namespace).
			Resource(resource).
			Param("labelSelector", fmt.Sprintf("%s=false", common.LabelKeyCompleted)).
			VersionedParams(&options, metav1.ParameterCodec)
		req = wfc.addLabelSelectors(req)
		return req.Watch()
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
// It is also responsible for unsetting the deamoned flag from a node status when it notices that a daemoned pod terminated.
func (wfc *WorkflowController) handlePodUpdate(pod *apiv1.Pod) {
	workflowName, ok := pod.Labels[common.LabelKeyWorkflow]
	if !ok {
		// Ignore pods unrelated to workflow (this shouldn't happen unless the watch is setup incorrectly)
		return
	}
	var newPhase wfv1.NodePhase
	var newDaemonStatus *bool
	var message string
	switch pod.Status.Phase {
	case apiv1.PodPending:
		return
	case apiv1.PodSucceeded:
		newPhase = wfv1.NodeSucceeded
		f := false
		newDaemonStatus = &f
	case apiv1.PodFailed:
		newPhase, newDaemonStatus, message = inferFailedReason(pod)
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
		newPhase = wfv1.NodeSucceeded
		t := true
		newDaemonStatus = &t
		log.Infof("Processing ready daemon pod: %v", pod.ObjectMeta.SelfLink)
	default:
		log.Infof("Unexpected pod phase for %s: %s", pod.ObjectMeta.Name, pod.Status.Phase)
		newPhase = wfv1.NodeError
	}

	wfClient := workflowclient.NewWorkflowClient(wfc.restClient, wfc.scheme, pod.ObjectMeta.Namespace)
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
	updateNeeded := applyUpdates(pod, &node, newPhase, newDaemonStatus, message)
	if !updateNeeded {
		log.Infof("No workflow updated needed for node %s", node)
	} else {
		wf.Status.Nodes[pod.Name] = node
		_, err = wfClient.UpdateWorkflow(wf)
		if err != nil {
			log.Errorf("Failed to update %s status: %+v", pod.Name, err)
			// if we fail to update the CRD state, we will need to rely on resync to catch up
			return
		}
		log.Infof("Updated %s", node)
	}

	if node.Completed() {
		// If we get here, we need to decide whether or not to set the 'completed=true' label on the pod,
		// which prevents the controller from seeing any pod updates for the rest of its existance.
		// We only add the label if the pod is *not* daemoned, because we still rely on this pod watch
		// for daemoned pods, in order to properly remove the daemoned status from the node when the pod
		// terminates.
		if !node.IsDaemoned() {
			err = common.AddPodLabel(wfc.clientset, pod.ObjectMeta.Name, pod.ObjectMeta.Namespace, common.LabelKeyCompleted, "true")
			if err != nil {
				log.Errorf("Failed to label completed pod %s: %+v", node, err)
				return
			}
			log.Infof("Set completed=true label to pod: %s", node)
		} else {
			log.Infof("Skipping completed labeling for daemoned pod: %s", node)
		}
	}
}

// inferFailedReason examines a Failed pod object to determine why it failed and return NodeStatus metadata
func inferFailedReason(pod *apiv1.Pod) (wfv1.NodePhase, *bool, string) {
	f := false
	if pod.Status.Message != "" {
		// Pod has a nice error message. Use that.
		return wfv1.NodeFailed, &f, pod.Status.Message
	}
	annotatedMsg := pod.Annotations[common.AnnotationKeyNodeMessage]
	// We only get one message to set for the overall node status.
	// If mutiple containers failed, in order of preference: init, main, wait, sidecars
	for _, ctr := range pod.Status.InitContainerStatuses {
		if ctr.State.Terminated == nil {
			// We should never get here
			log.Warnf("Pod %s phase was Failed but %s did not have terminated state", pod.ObjectMeta.Name, ctr.Name)
			continue
		}
		if ctr.State.Terminated.ExitCode == 0 {
			continue
		}
		errMsg := fmt.Sprintf("failed to load artifacts")
		for _, msg := range []string{annotatedMsg, ctr.State.Terminated.Message} {
			if msg != "" {
				errMsg += ": " + msg
				break
			}
		}
		// NOTE: we consider artifact load issues as Error instead of Failed
		return wfv1.NodeError, &f, errMsg
	}
	failMessages := make(map[string]string)
	for _, ctr := range pod.Status.ContainerStatuses {
		if ctr.State.Terminated == nil {
			// We should never get here
			log.Warnf("Pod %s phase was Failed but %s did not have terminated state", pod.ObjectMeta.Name, ctr.Name)
			continue
		}
		if ctr.State.Terminated.ExitCode == 0 {
			continue
		}
		if ctr.Name == common.WaitContainerName {
			errMsg := fmt.Sprintf("failed to save artifacts")
			for _, msg := range []string{annotatedMsg, ctr.State.Terminated.Message} {
				if msg != "" {
					errMsg += ": " + msg
					break
				}
			}
			failMessages[ctr.Name] = errMsg
		} else {
			if ctr.State.Terminated.Message != "" {
				failMessages[ctr.Name] = ctr.State.Terminated.Message
			} else {
				failMessages[ctr.Name] = fmt.Sprintf("failed with exit code %d", ctr.State.Terminated.ExitCode)
			}
		}
	}
	if failMsg, ok := failMessages[common.MainContainerName]; ok {
		return wfv1.NodeFailed, &f, failMsg
	}
	if failMsg, ok := failMessages[common.WaitContainerName]; ok {
		return wfv1.NodeError, &f, failMsg
	}

	// If we get here, both the main and wait container succeeded.
	// Identify the sidecar which failed and give proper message
	// NOTE: we may need to distinguish between the main container
	// succeeding and ignoring the sidecar statuses. This is because
	// executor may have had to forcefully terminate a sidecar
	// (kill -9), resulting in an non-zero exit code of a sidecar,
	// and overall pod status as failed. Or the sidecar is actually
	// *expected* to fail non-zero and should be ignored. Users may
	// want the option to consider a step failed only if the main
	// container failed. For now return the first failure.
	for _, failMsg := range failMessages {
		return wfv1.NodeFailed, &f, failMsg
	}
	return wfv1.NodeFailed, &f, fmt.Sprintf("pod failed for unknown reason")
}

// applyUpdates applies any new state information about a pod, to the current status of the workflow node
// returns whether or not any updates were necessary (resulting in a update to the workflow)
func applyUpdates(pod *apiv1.Pod, node *wfv1.NodeStatus, newPhase wfv1.NodePhase, newDaemonStatus *bool, message string) bool {
	// Check various fields of the pods to see if we need to update the workflow
	updateNeeded := false
	if node.Phase != newPhase {
		if node.Completed() {
			// Don't modify the phase if this node was already considered completed.
			// This might happen with daemoned steps which fail after they were daemoned
			log.Infof("Ignoring node %s status update %s -> %s", node, node.Phase, newPhase)
		} else {
			log.Infof("Updating node %s status %s -> %s", node, node.Phase, newPhase)
			updateNeeded = true
			node.Phase = newPhase
		}
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
		if (newDaemonStatus != nil && node.Daemoned == nil) || (newDaemonStatus == nil && node.Daemoned != nil) {
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
			node.Phase = wfv1.NodeError
		} else {
			node.Outputs = &outputs
		}
	}
	if message != "" && node.Message != message {
		log.Infof("Updating node %s message: %s", node, message)
		node.Message = message
	}
	if node.Completed() && node.FinishedAt.IsZero() {
		// TODO: rather than using time.Now(), we should use the latest container finishedAt
		// timestamp to get a more accurate finished timestamp, in the event the controller is
		// down or backlogged. But this would not work for daemoned containers.
		node.FinishedAt = metav1.Time{Time: time.Now().UTC()}
		updateNeeded = true
	}
	return updateNeeded
}
