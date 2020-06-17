package config

import (
	apiv1 "k8s.io/api/core/v1"

	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo/server/auth/sso"
)

var emptyConfig = Config{}

// Config contain the configuration settings for the workflow controller
type Config struct {
	// SSO in settings for single-sign on
	SSO sso.Config `json:"sso,omitempty"`

	// NodeEvents configures how node events are omitted
	NodeEvents NodeEvents `json:"nodeEvents,omitempty"`

	// ExecutorImage is the image name of the executor to use when running pods
	// DEPRECATED: use --executor-image flag to workflow-controller instead
	ExecutorImage string `json:"executorImage,omitempty"`

	// ExecutorImagePullPolicy is the imagePullPolicy of the executor to use when running pods
	// DEPRECATED: use `executor.imagePullPolicy` in configmap instead
	ExecutorImagePullPolicy string `json:"executorImagePullPolicy,omitempty"`

	// Executor holds container customizations for the executor to use when running pods
	Executor *apiv1.Container `json:"executor,omitempty"`

	// ExecutorResources specifies the resource requirements that will be used for the executor sidecar
	// DEPRECATED: use `executor.resources` in configmap instead
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
	ArtifactRepository wfv1.ArtifactRepository `json:"artifactRepository,omitempty"`

	// Namespace is a label selector filter to limit the controller's watch to a specific namespace
	// DEPRECATED: support will be remove in a future release
	Namespace string `json:"namespace,omitempty"`

	// InstanceID is a label selector to limit the controller's watch to a specific instance. It
	// contains an arbitrary value that is carried forward into its pod labels, under the key
	// workflows.argoproj.io/controller-instanceid, for the purposes of workflow segregation. This
	// enables a controller to only receive workflow and pod events that it is interested about,
	// in order to support multiple controllers in a single cluster, and ultimately allows the
	// controller itself to be bundled as part of a higher level application. If omitted, the
	// controller watches workflows and pods that *are not* labeled with an instance id.
	InstanceID string `json:"instanceID,omitempty"`

	// MetricsConfig specifies configuration for metrics emission. Metrics are enabled and emitted on localhost:9090/metrics
	// by default.
	MetricsConfig MetricsConfig `json:"metricsConfig,omitempty"`

	// TelemetryConfig specifies configuration for telemetry emission. Telemetry is enabled and emitted in the same endpoint
	// as metrics by default, but can be overridden using this config.
	TelemetryConfig MetricsConfig `json:"telemetryConfig,omitempty"`

	// Parallelism limits the max total parallel workflows that can execute at the same time
	Parallelism int `json:"parallelism,omitempty"`

	// Persistence contains the workflow persistence DB configuration
	Persistence *PersistConfig `json:"persistence,omitempty"`

	// Links to related apps.
	Links []*wfv1.Link `json:"links,omitempty"`

	// Config customized Docker Sock path
	DockerSockPath string `json:"dockerSockPath,omitempty"`

	// WorkflowDefaults are values that will apply to all Workflows from this controller, unless overridden on the Workflow-level
	WorkflowDefaults *wfv1.Workflow `json:"workflowDefaults,omitempty"`

	// PodSpecLogStrategy enable the logging of podspec on controller log.
	PodSpecLogStrategy PodSpecLogStrategy `json:"podSpecLogStrategy,omitempty"`
}

// PodSpecLogStrategy contains the configuration for logging the pod spec in controller log for debugging purpose
type PodSpecLogStrategy struct {
	FailedPod bool `json:"failedPod,omitempty"`
	AllPods   bool `json:"allPods,omitempty"`
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

type PersistConfig struct {
	NodeStatusOffload bool `json:"nodeStatusOffLoad,omitempty"`
	// Archive workflows to persistence.
	Archive bool `json:"archive,omitempty"`
	// in days
	ArchiveTTL     TTL               `json:"archiveTTL,omitempty"`
	ClusterName    string            `json:"clusterName,omitempty"`
	ConnectionPool *ConnectionPool   `json:"connectionPool,omitempty"`
	PostgreSQL     *PostgreSQLConfig `json:"postgresql,omitempty"`
	MySQL          *MySQLConfig      `json:"mysql,omitempty"`
}

func (c PersistConfig) GetClusterName() string {
	if c.ClusterName != "" {
		return c.ClusterName
	}
	return "default"
}

type ConnectionPool struct {
	MaxIdleConns    int `json:"maxIdleConns,omitempty"`
	MaxOpenConns    int `json:"maxOpenConns,omitempty"`
	ConnMaxLifetime TTL `json:"connMaxLifetime,omitempty"`
}
type PostgreSQLConfig struct {
	Host           string                  `json:"host"`
	Port           string                  `json:"port"`
	Database       string                  `json:"database"`
	TableName      string                  `json:"tableName,omitempty"`
	UsernameSecret apiv1.SecretKeySelector `json:"userNameSecret,omitempty"`
	PasswordSecret apiv1.SecretKeySelector `json:"passwordSecret,omitempty"`
	SSL            bool                    `json:"ssl,omitempty"`
	SSLMode        string                  `json:"sslMode,omitempty"`
}

type MySQLConfig struct {
	Host           string                  `json:"host"`
	Port           string                  `json:"port"`
	Database       string                  `json:"database"`
	TableName      string                  `json:"tableName,omitempty"`
	Options        map[string]string       `json:"options,omitempty"`
	UsernameSecret apiv1.SecretKeySelector `json:"userNameSecret,omitempty"`
	PasswordSecret apiv1.SecretKeySelector `json:"passwordSecret,omitempty"`
}

// MetricsConfig defines a config for a metrics server
type MetricsConfig struct {
	// Enabled controls metric emission. Default is true, set "enabled: false" to turn off
	Enabled *bool `json:"enabled,omitempty"`
	// DisableLegacy turns off legacy metrics
	// DEPRECATED: Legacy metrics are now removed, this field is ignored
	DisableLegacy bool `json:"disableLegacy,omitempty"`
	// MetricsTTL sets how often custom metrics are cleared from memory
	MetricsTTL TTL `json:"metricsTTL,omitempty"`
	// Path is the path where metrics are emitted. Must start with a "/". Default is "/metrics"
	Path string `json:"path,omitempty"`
	// Port is the port where metrics are emitted. Default is "9090"
	Port string `json:"port,omitempty"`
	// IgnoreErrors is a flag that instructs prometheus to ignore metric emission errors
	IgnoreErrors bool `json:"ignoreErrors,omitempty"`
}
