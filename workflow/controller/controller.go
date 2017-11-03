package controller

import (
	"context"
	"encoding/json"

	"github.com/argoproj/argo"
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
	"k8s.io/client-go/kubernetes"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
)

type WorkflowController struct {
	// ConfigMap is the name of the config map in which to derive configuration of the controller from
	ConfigMap      string
	WorkflowClient *workflowclient.WorkflowClient
	WorkflowScheme *runtime.Scheme
	Config         WorkflowControllerConfig

	clientset  *kubernetes.Clientset
	podCl      corev1.PodInterface
	wfUpdates  chan *wfv1.Workflow
	podUpdates chan *apiv1.Pod
}

type WorkflowControllerConfig struct {
	ExecutorImage      string             `json:"executorImage,omitempty"`
	ArtifactRepository ArtifactRepository `json:"artifactRepository,omitempty"`
}

// ArtifactRepository represents a artifact repository in which a controller will store its artifacts
type ArtifactRepository struct {
	S3 *S3ArtifactRepository `json:"s3,omitempty"`
	// Future artifact repository support here
}
type S3ArtifactRepository struct {
	wfv1.S3Bucket `json:",inline"`

	// KeyPrefix is prefix used as part of the bucket key in which the controller will store artifacts.
	KeyPrefix string `json:"keyPrefix,omitempty"`
}

// NewWorkflowController instantiates a new WorkflowController
func NewWorkflowController(config *rest.Config, configMap string) *WorkflowController {
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
		ConfigMap:      configMap,
		podCl:          clientset.CoreV1().Pods(apiv1.NamespaceDefault),
		wfUpdates:      make(chan *wfv1.Workflow),
		podUpdates:     make(chan *apiv1.Pod),
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
	cmClient := wfc.clientset.CoreV1().ConfigMaps(apiv1.NamespaceDefault)
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
	if config.ArtifactRepository.S3 != nil {
		err = wfc.validateS3Repository(*config.ArtifactRepository.S3)
		if err != nil {
			return err
		}
	}
	wfc.Config = config
	if wfc.Config.ExecutorImage == "" {
		wfc.Config.ExecutorImage = "argoproj/argoexec:" + argo.Version
	}
	return nil
}

func (wfc *WorkflowController) validateS3Repository(s3repo S3ArtifactRepository) error {
	secClient := wfc.clientset.CoreV1().Secrets(apiv1.NamespaceDefault)
	for _, secSelector := range []apiv1.SecretKeySelector{s3repo.AccessKeySecret, s3repo.SecretKeySecret} {
		s3bucketSecret, err := secClient.Get(secSelector.Name, metav1.GetOptions{})
		if err != nil {
			return errors.InternalWrapError(err)
		}
		secBytes := s3bucketSecret.Data[secSelector.Key]
		if len(secBytes) == 0 {
			return errors.Errorf(errors.CodeBadRequest, "secret '%s' key '%s' empty", secSelector.LocalObjectReference, secSelector.Key)
		}
	}
	return nil
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
				log.Infof("WF Add %s", wf.ObjectMeta.SelfLink)
				wfc.wfUpdates <- wf
			},
			UpdateFunc: func(old, new interface{}) {
				//oldWf := old.(*wfv1.Workflow)
				newWf := new.(*wfv1.Workflow)
				log.Infof("WF Update %s", newWf.ObjectMeta.SelfLink)
				wfc.wfUpdates <- newWf
			},
			DeleteFunc: func(obj interface{}) {
				wf := obj.(*wfv1.Workflow)
				log.Infof("WF Delete %s", wf.ObjectMeta.SelfLink)
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
				log.Infof("Pod Added %s", pod.ObjectMeta.SelfLink)
				wfc.podUpdates <- pod
			},
			UpdateFunc: func(old, new interface{}) {
				//oldPod := old.(*apiv1.Pod)
				newPod := new.(*apiv1.Pod)
				log.Infof("Pod Updated %s", newPod.ObjectMeta.SelfLink)
				wfc.podUpdates <- newPod
			},
			DeleteFunc: func(obj interface{}) {
				pod := obj.(*apiv1.Pod)
				log.Infof("Pod Deleted %s", pod.ObjectMeta.SelfLink)
				wfc.podUpdates <- pod
			},
		})

	go controller.Run(ctx.Done())
	return controller, nil
}

func (wfc *WorkflowController) handlePodUpdate(pod *apiv1.Pod) {
	if pod.Status.Phase != apiv1.PodSucceeded && pod.Status.Phase != apiv1.PodFailed {
		// Ignore pod updates for running pods
		return
	}
	workflowName, ok := pod.Labels[common.LabelKeyWorkflow]
	if !ok {
		return
	}
	log.Infof("Processing completed pod: %v", pod.ObjectMeta.SelfLink)
	wf, err := wfc.WorkflowClient.GetWorkflow(workflowName)
	if err != nil {
		log.Warnf("Failed to find workflow %s %+v", workflowName, err)
		return
	}
	node, ok := wf.Status.Nodes[pod.Name]
	if !ok {
		log.Warnf("pod %s unassociated with workflow %s", pod.Name, workflowName)
		return
	}
	if node.Completed() {
		log.Infof("node %v already marked completed (%s)", node, node.Status)
		return
	}
	var newStatus string
	switch pod.Status.Phase {
	case apiv1.PodSucceeded:
		newStatus = wfv1.NodeStatusSucceeded
	case apiv1.PodFailed:
		newStatus = wfv1.NodeStatusFailed
	default:
		newStatus = wfv1.NodeStatusError
	}
	outputStr, ok := pod.Annotations[common.AnnotationKeyOutputs]
	if ok {
		var outputs wfv1.Outputs
		err = json.Unmarshal([]byte(outputStr), &outputs)
		if err != nil {
			log.Errorf("Failed to unmarshal %s outputs from pod annotation: %v", pod.Name, err)
			newStatus = wfv1.NodeStatusError
		} else {
			node.Outputs = &outputs
		}
	}
	log.Infof("Updating node %s status %s -> %s", node, node.Status, newStatus)
	node.Status = newStatus
	wf.Status.Nodes[pod.Name] = node
	_, err = wfc.WorkflowClient.UpdateWorkflow(wf)
	if err != nil {
		log.Errorf("Failed to update %s status: %+v", pod.Name, err)
		// if we fail to update the CRD state, we will need to rely on resync to catch up
		return
	}
	log.Infof("Updated %v", node)
}
