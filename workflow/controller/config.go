package controller

import (
	"context"
	"fmt"
	"io/ioutil"

	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"

	"github.com/argoproj/argo/errors"
	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo/workflow/common"
	"github.com/argoproj/argo/workflow/metrics"
	"github.com/ghodss/yaml"
	log "github.com/sirupsen/logrus"
)

// WorkflowControllerConfig contain the configuration settings for the workflow controller
type WorkflowControllerConfig struct {
	// ExecutorImage is the image name of the executor to use when running pods
	ExecutorImage string `json:"executorImage,omitempty"`

	// ExecutorImagePullPolicy is the imagePullPolicy of the executor to use when running pods
	ExecutorImagePullPolicy string `json:"executorImagePullPolicy,omitempty"`

	// ExecutorResources specifies the resource requirements that will be used for the executor sidecar
	ExecutorResources *apiv1.ResourceRequirements `json:"executorResources,omitempty"`

	// KubeConfig specifies a kube config file for the wait & init containers
	KubeConfig *KubeConfig `json:"kubeConfig,omitempty"`

	// ContainerRuntimeExecutor specifies the container runtime interface to use, default is docker
	ContainerRuntimeExecutor string `json:"containerRuntimeExecutor,omitempty"`

	// KubeletPort is needed when using the kubelet containerRuntimeExecutor, default to 10250
	KubeletPort int `json:"kubeletPort,omitempty"`

	// KubeletInsecure disable the TLS verification of the kubelet containerRuntimeExecutor, default to false
	KubeletInsecure bool `json:"kubeletInsecure,omitempty"`

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

	MetricsConfig metrics.PrometheusConfig `json:"metricsConfig,omitempty"`

	TelemetryConfig metrics.PrometheusConfig `json:"telemetryConfig,omitempty"`

	// Parallelism limits the max total parallel workflows that can execute at the same time
	Parallelism int `json:"parallelism,omitempty"`
}

// KubeConfig is used for wait & init sidecar containers to communicate with a k8s apiserver by a outofcluster method,
// it is used when the workflow controller is in a different cluster with the workflow workloads
type KubeConfig struct {
	// SecretName of the kubeconfig secret
	// may not be empty if kuebConfig specified
	SecretName string `json:"secretName"`
	// SecretKey of the kubeconfig in the secret
	// may not be empty if kubeConfig specified
	SecretKey string `json:"secretKey"`
	// VolumeName of kubeconfig, default to 'kubeconfig'
	VolumeName string `json:"volumeName,omitempty"`
	// MountPath of the kubeconfig secret, default to '/kube/config'
	MountPath string `json:"mountPath,omitempty"`
}

// ArtifactRepository represents a artifact repository in which a controller will store its artifacts
type ArtifactRepository struct {
	// ArchiveLogs enables log archiving
	ArchiveLogs *bool `json:"archiveLogs,omitempty"`
	// S3 stores artifact in a S3-compliant object store
	S3 *S3ArtifactRepository `json:"s3,omitempty"`
	// Artifactory stores artifacts to JFrog Artifactory
	Artifactory *ArtifactoryArtifactRepository `json:"artifactory,omitempty"`
	// HDFS stores artifacts in HDFS
	HDFS *HDFSArtifactRepository `json:"hdfs,omitempty"`
}

// S3ArtifactRepository defines the controller configuration for an S3 artifact repository
type S3ArtifactRepository struct {
	wfv1.S3Bucket `json:",inline"`

	// KeyFormat is defines the format of how to store keys. Can reference workflow variables
	KeyFormat string `json:"keyFormat,omitempty"`

	// KeyPrefix is prefix used as part of the bucket key in which the controller will store artifacts.
	// DEPRECATED. Use KeyFormat instead
	KeyPrefix string `json:"keyPrefix,omitempty"`
}

// ArtifactoryArtifactRepository defines the controller configuration for an artifactory artifact repository
type ArtifactoryArtifactRepository struct {
	wfv1.ArtifactoryAuth `json:",inline"`
	// RepoURL is the url for artifactory repo.
	RepoURL string `json:"repoURL,omitempty"`
}

// HDFSArtifactRepository defines the controller configuration for an HDFS artifact repository
type HDFSArtifactRepository struct {
	wfv1.HDFSConfig `json:",inline"`

	// PathFormat is defines the format of path to store a file. Can reference workflow variables
	PathFormat string `json:"pathFormat,omitempty"`

	// Force copies a file forcibly even if it exists (default: false)
	Force bool `json:"force,omitempty"`
}

// ResyncConfig reloads the controller config from the configmap
func (wfc *WorkflowController) ResyncConfig() error {
	if wfc.configPath != "" {
		if e := wfc.loadConfigFile(wfc.configPath); e != nil {
			return errors.InternalWrapError(e)
		}
	} else if wfc.configMap != "" {
		cmClient := wfc.kubeclientset.CoreV1().ConfigMaps(wfc.namespace)
		cm, err := cmClient.Get(wfc.configMap, metav1.GetOptions{})
		if err != nil {
			return errors.InternalWrapError(err)
		}
		if err := wfc.updateConfig(cm); err != nil {
			return errors.InternalWrapError(err)
		}
	}
	if wfc.cliExecutorImage == "" && wfc.Config.ExecutorImage == "" {
		return errors.Errorf(errors.CodeBadRequest, "executorImage unspecified")
	}
	return nil
}

func (wfc *WorkflowController) updateConfig(cm *apiv1.ConfigMap) error {
	configStr, ok := cm.Data[common.WorkflowControllerConfigMapKey]
	if !ok {
		log.Warnf("ConfigMap '%s' does not have key '%s'", wfc.configMap, common.WorkflowControllerConfigMapKey)
		return nil
	}
	var config WorkflowControllerConfig
	err := yaml.Unmarshal([]byte(configStr), &config)
	if err != nil {
		return errors.InternalWrapError(err)
	}
	log.Printf("workflow controller configuration from %s:\n%s", wfc.configMap, configStr)
	wfc.Config = config
	return nil
}

func (wfc *WorkflowController) loadConfigFile(path string) error {
	buf, e := ioutil.ReadFile(path)
	if e != nil {
		return errors.InternalWrapError(e, "read config file failed")
	}
	var config WorkflowControllerConfig
	if e := yaml.Unmarshal(buf, &config); e != nil {
		return errors.InternalWrapError(e, "parse config file failed")
	}
	log.Printf("workflow controller configuration from %s:\n%s", path, string(buf))
	wfc.Config = config
	wfc.throttler.SetParallelism(config.Parallelism)
	return nil
}

// executorImage returns the image to use for the workflow executor
func (wfc *WorkflowController) executorImage() string {
	if wfc.cliExecutorImage != "" {
		return wfc.cliExecutorImage
	}
	return wfc.Config.ExecutorImage
}

// executorImagePullPolicy returns the imagePullPolicy to use for the workflow executor
func (wfc *WorkflowController) executorImagePullPolicy() apiv1.PullPolicy {
	var policy string
	if wfc.cliExecutorImagePullPolicy != "" {
		policy = wfc.cliExecutorImagePullPolicy
	} else {
		policy = wfc.Config.ExecutorImagePullPolicy
	}
	return apiv1.PullPolicy(policy)
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
				oldCM := old.(*apiv1.ConfigMap)
				newCM := new.(*apiv1.ConfigMap)
				if oldCM.ResourceVersion == newCM.ResourceVersion {
					return
				}
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
	c := wfc.kubeclientset.CoreV1().RESTClient()
	resource := "configmaps"
	name := wfc.configMap
	fieldSelector := fields.ParseSelectorOrDie(fmt.Sprintf("metadata.name=%s", name))

	listFunc := func(options metav1.ListOptions) (runtime.Object, error) {
		options.FieldSelector = fieldSelector.String()
		req := c.Get().
			Namespace(wfc.namespace).
			Resource(resource).
			VersionedParams(&options, metav1.ParameterCodec)
		return req.Do().Get()
	}
	watchFunc := func(options metav1.ListOptions) (watch.Interface, error) {
		options.Watch = true
		options.FieldSelector = fieldSelector.String()
		req := c.Get().
			Namespace(wfc.namespace).
			Resource(resource).
			VersionedParams(&options, metav1.ParameterCodec)
		return req.Watch()
	}
	return &cache.ListWatch{ListFunc: listFunc, WatchFunc: watchFunc}
}
