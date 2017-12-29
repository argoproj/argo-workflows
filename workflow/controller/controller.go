package controller

import (
	"context"
	"fmt"
	"os"
	goruntime "runtime"
	"time"

	"github.com/argoproj/argo"
	wfv1 "github.com/argoproj/argo/api/workflow/v1alpha1"
	"github.com/argoproj/argo/errors"
	workflowclient "github.com/argoproj/argo/workflow/client"
	"github.com/argoproj/argo/workflow/common"
	"github.com/ghodss/yaml"
	log "github.com/sirupsen/logrus"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
)

type WorkflowController struct {
	// ConfigMap is the name of the config map in which to derive configuration of the controller from
	ConfigMap string
	// namespace for config map
	ConfigMapNS string
	Config      WorkflowControllerConfig

	restConfig *rest.Config
	restClient rest.Interface
	scheme     *runtime.Scheme
	clientset  kubernetes.Interface

	// datastructures to support the processing of workflows and workflow pods
	wfInformer  cache.SharedIndexInformer
	podInformer cache.SharedIndexInformer
	wfQueue     workqueue.RateLimitingInterface
	podQueue    workqueue.RateLimitingInterface
}

// WorkflowControllerConfig contain the configuration settings for the workflow controller
type WorkflowControllerConfig struct {
	// ExecutorImage is the image name of the executor to use when running pods
	ExecutorImage string `json:"executorImage,omitempty"`

	// ArtifactRepository contains the default location of an artifact repository for container artifacts
	ArtifactRepository ArtifactRepository `json:"artifactRepository,omitempty"`

	// Namespace is a label selector filter to limit the controller's watch to a specific namespace
	Namespace string `json:"namespace,omitempty"`

	// InstanceID is a label selector to limit the controller's watch to a specific instance. It
	// contains an arbitrary value that is carried forward into its pod labels, under the key
	// workflows.argoproj.io/controller-instanceid, for the purposes of workflow segregation. This
	// enables a controller to only receive workflow and pod events that it is interested about,
	// in order to support multiple controllers in a single cluster, and ultimately allows the
	// controller itself to be bundled as part of a higher level application. If omitted, the
	// controller watches workflows and pods that *are not* labeled with an instance id.
	InstanceID string `json:"instanceID,omitempty"`

	MatchLabels map[string]string `json:"matchLabels,omitempty"`
}

const (
	workflowResyncPeriod = 20 * time.Minute
	podResyncPeriod      = 30 * time.Minute
)

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
		wfQueue:    workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter()),
		podQueue:   workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter()),
	}
	return &wfc
}

// Run starts an Workflow resource controller
func (wfc *WorkflowController) Run(ctx context.Context) {
	defer wfc.wfQueue.ShutDown()
	defer wfc.podQueue.ShutDown()

	log.Infof("Workflow Controller (version: %s) starting", argo.GetVersion())
	log.Info("Watch Workflow controller config map updates")
	_, err := wfc.watchControllerConfigMap(ctx)
	if err != nil {
		log.Errorf("Failed to register watch for controller config map: %v", err)
		return
	}

	wfc.wfInformer = wfc.newWorkflowInformer()
	wfc.podInformer = wfc.newPodInformer()
	go wfc.wfInformer.Run(ctx.Done())
	go wfc.podInformer.Run(ctx.Done())

	// Wait for all involved caches to be synced, before processing items from the queue is started
	for _, informer := range []cache.SharedIndexInformer{wfc.wfInformer, wfc.podInformer} {
		if !cache.WaitForCacheSync(ctx.Done(), informer.HasSynced) {
			log.Error("Timed out waiting for caches to sync")
			return
		}
	}

	wfc.StartStatsTicker(5 * time.Minute)
	for i := 0; i < 4; i++ {
		go wait.Until(wfc.runWorker, time.Second, ctx.Done())
	}
	for i := 0; i < 8; i++ {
		go wait.Until(wfc.podWorker, time.Second, ctx.Done())
	}
	<-ctx.Done()
}

func (wfc *WorkflowController) runWorker() {
	for wfc.processNextItem() {
	}
}

// processNextItem is the worker logic for handling workflow updates
func (wfc *WorkflowController) processNextItem() bool {
	key, quit := wfc.wfQueue.Get()
	if quit {
		return false
	}
	defer wfc.wfQueue.Done(key)

	obj, exists, err := wfc.wfInformer.GetIndexer().GetByKey(key.(string))
	if err != nil {
		log.Errorf("Failed to get workflow '%s' from informer index: %+v", key, err)
		return true
	}
	if !exists {
		// This happens after a workflow was labeled with completed=true
		// or was deleted, but the work queue still had an entry for it.
		return true
	}
	wf, ok := obj.(*wfv1.Workflow)
	if !ok {
		log.Warnf("Key '%s' in index is not a workflow", key)
		return true
	}
	wfc.operateWorkflow(wf)
	// TODO: operateWorkflow should return error if it was unable to operate properly
	// so we can requeue the work for a later time
	// See: https://github.com/kubernetes/client-go/blob/master/examples/workqueue/main.go
	//c.handleErr(err, key)
	return true
}

func (wfc *WorkflowController) podWorker() {
	for wfc.processNextPodItem() {
	}
}

// processNextPodItem is the worker logic for handling pod updates.
// For pods updates, this simply means to "wake up" the workflow by
// adding the corresponding workflow key into the workflow workqueue.
func (wfc *WorkflowController) processNextPodItem() bool {
	key, quit := wfc.podQueue.Get()
	if quit {
		return false
	}
	defer wfc.podQueue.Done(key)

	obj, exists, err := wfc.podInformer.GetIndexer().GetByKey(key.(string))
	if err != nil {
		log.Errorf("Failed to get pod '%s' from informer index: %+v", key, err)
		return true
	}
	if !exists {
		// we can get here if pod was queued into the pod workqueue,
		// but it was either deleted or labeled completed by the time
		// we dequeued it.
		return true
	}
	pod, ok := obj.(*apiv1.Pod)
	if !ok {
		log.Warnf("Key '%s' in index is not a pod", key)
		return true
	}
	if pod.Labels == nil {
		log.Warnf("Pod '%s' did not have labels", key)
		return true
	}
	workflowName, ok := pod.Labels[common.LabelKeyWorkflow]
	if !ok {
		// Ignore pods unrelated to workflow (this shouldn't happen unless the watch is setup incorrectly)
		log.Warnf("watch returned pod unrelated to any workflow: %s", pod.ObjectMeta.Name)
		return true
	}
	// TODO: currently we reawaken the workflow on *any* pod updates.
	// But this could be be much improved to become smarter by only
	// requeue the workflow when there are changes that we care about.
	wfc.wfQueue.Add(pod.ObjectMeta.Namespace + "/" + workflowName)
	return true
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

// instanceIDRequirement returns the label requirement to filter against a controller instance (or not)
func (wfc *WorkflowController) instanceIDRequirement() labels.Requirement {
	var instanceIDReq *labels.Requirement
	var err error
	if wfc.Config.InstanceID != "" {
		instanceIDReq, err = labels.NewRequirement(common.LabelKeyControllerInstanceID, selection.Equals, []string{wfc.Config.InstanceID})
	} else {
		instanceIDReq, err = labels.NewRequirement(common.LabelKeyControllerInstanceID, selection.DoesNotExist, nil)
	}
	if err != nil {
		panic(err)
	}
	return *instanceIDReq
}

func (wfc *WorkflowController) newWorkflowWatch() *cache.ListWatch {
	c := wfc.restClient
	resource := wfv1.CRDPlural
	namespace := wfc.Config.Namespace
	fieldSelector := fields.Everything()
	// completed notin (true)
	incompleteReq, err := labels.NewRequirement(common.LabelKeyCompleted, selection.NotIn, []string{"true"})
	if err != nil {
		panic(err)
	}
	labelSelector := labels.NewSelector().
		Add(*incompleteReq).
		Add(wfc.instanceIDRequirement())

	listFunc := func(options metav1.ListOptions) (runtime.Object, error) {
		options.FieldSelector = fieldSelector.String()
		options.LabelSelector = labelSelector.String()
		req := c.Get().
			Namespace(namespace).
			Resource(resource).
			VersionedParams(&options, metav1.ParameterCodec)
		return req.Do().Get()
	}
	watchFunc := func(options metav1.ListOptions) (watch.Interface, error) {
		options.Watch = true
		options.FieldSelector = fieldSelector.String()
		options.LabelSelector = labelSelector.String()
		req := c.Get().
			Namespace(namespace).
			Resource(resource).
			VersionedParams(&options, metav1.ParameterCodec)
		return req.Watch()
	}
	return &cache.ListWatch{ListFunc: listFunc, WatchFunc: watchFunc}
}

func (wfc *WorkflowController) newWorkflowInformer() cache.SharedIndexInformer {
	source := wfc.newWorkflowWatch()
	informer := cache.NewSharedIndexInformer(source, &wfv1.Workflow{}, workflowResyncPeriod, cache.Indexers{})
	informer.AddEventHandler(
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				key, err := cache.MetaNamespaceKeyFunc(obj)
				if err == nil {
					wfc.wfQueue.Add(key)
				}
			},
			UpdateFunc: func(old, new interface{}) {
				key, err := cache.MetaNamespaceKeyFunc(new)
				if err == nil {
					wfc.wfQueue.Add(key)
				}
			},
			DeleteFunc: func(obj interface{}) {
				// IndexerInformer uses a delta queue, therefore for deletes we have to use this
				// key function.
				key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
				if err == nil {
					wfc.wfQueue.Add(key)
				}
			},
		},
	)
	return informer
}

func (wfc *WorkflowController) watchControllerConfigMap(ctx context.Context) (cache.Controller, error) {
	source := wfc.newControllerConfigMapWatch()
	_, controller := cache.NewInformer(
		source,
		&apiv1.ConfigMap{},
		0,
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				if cm, ok := obj.(*apiv1.ConfigMap); ok {
					log.Infof("Detected ConfigMap update. Updating the controller config.")
					err := wfc.updateConfig(cm)
					if err != nil {
						log.Errorf("Update of config failed due to: %v", err)
					}
				}
			},
			UpdateFunc: func(old, new interface{}) {
				if newCm, ok := new.(*apiv1.ConfigMap); ok {
					log.Infof("Detected ConfigMap update. Updating the controller config.")
					err := wfc.updateConfig(newCm)
					if err != nil {
						log.Errorf("Update of config failed due to: %v", err)
					}
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
	fieldSelector := fields.ParseSelectorOrDie(fmt.Sprintf("metadata.name=%s", name))

	listFunc := func(options metav1.ListOptions) (runtime.Object, error) {
		options.FieldSelector = fieldSelector.String()
		req := c.Get().
			Namespace(namespace).
			Resource(resource).
			VersionedParams(&options, metav1.ParameterCodec)
		return req.Do().Get()
	}
	watchFunc := func(options metav1.ListOptions) (watch.Interface, error) {
		options.Watch = true
		options.FieldSelector = fieldSelector.String()
		req := c.Get().
			Namespace(namespace).
			Resource(resource).
			VersionedParams(&options, metav1.ParameterCodec)
		return req.Watch()
	}
	return &cache.ListWatch{ListFunc: listFunc, WatchFunc: watchFunc}
}

func (wfc *WorkflowController) newWorkflowPodWatch() *cache.ListWatch {
	c := wfc.clientset.Core().RESTClient()
	resource := "pods"
	namespace := wfc.Config.Namespace
	fieldSelector := fields.ParseSelectorOrDie("status.phase!=Pending")
	// completed=false
	incompleteReq, _ := labels.NewRequirement(common.LabelKeyCompleted, selection.Equals, []string{"false"})
	labelSelector := labels.NewSelector().
		Add(*incompleteReq).
		Add(wfc.instanceIDRequirement())

	listFunc := func(options metav1.ListOptions) (runtime.Object, error) {
		options.FieldSelector = fieldSelector.String()
		options.LabelSelector = labelSelector.String()
		req := c.Get().
			Namespace(namespace).
			Resource(resource).
			VersionedParams(&options, metav1.ParameterCodec)
		return req.Do().Get()
	}
	watchFunc := func(options metav1.ListOptions) (watch.Interface, error) {
		options.Watch = true
		options.FieldSelector = fieldSelector.String()
		options.LabelSelector = labelSelector.String()
		req := c.Get().
			Namespace(namespace).
			Resource(resource).
			VersionedParams(&options, metav1.ParameterCodec)
		return req.Watch()
	}
	return &cache.ListWatch{ListFunc: listFunc, WatchFunc: watchFunc}
}

func (wfc *WorkflowController) newPodInformer() cache.SharedIndexInformer {
	source := wfc.newWorkflowPodWatch()
	informer := cache.NewSharedIndexInformer(source, &apiv1.Pod{}, podResyncPeriod, cache.Indexers{})
	informer.AddEventHandler(
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				key, err := cache.MetaNamespaceKeyFunc(obj)
				if err == nil {
					wfc.podQueue.Add(key)
				}
			},
			UpdateFunc: func(old, new interface{}) {
				key, err := cache.MetaNamespaceKeyFunc(new)
				if err == nil {
					wfc.podQueue.Add(key)
				}
			},
			DeleteFunc: func(obj interface{}) {
				// IndexerInformer uses a delta queue, therefore for deletes we have to use this
				// key function.
				key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
				if err == nil {
					wfc.podQueue.Add(key)
				}
			},
		},
	)
	return informer
}

// StartStatsTicker starts a goroutine which dumps stats at a specified interval
func (wfc *WorkflowController) StartStatsTicker(d time.Duration) {
	ticker := time.NewTicker(d)
	go func() {
		for {
			<-ticker.C
			var m goruntime.MemStats
			goruntime.ReadMemStats(&m)
			log.Infof("Alloc=%v TotalAlloc=%v Sys=%v NumGC=%v Goroutines=%d",
				m.Alloc/1024, m.TotalAlloc/1024, m.Sys/1024, m.NumGC, goruntime.NumGoroutine())
		}
	}()
}
