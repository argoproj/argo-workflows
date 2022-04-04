package config

import (
	"fmt"
	"math"
	"time"

	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
)

type ResourceRateLimit struct {
	Limit float64 `json:"limit"`
	Burst int     `json:"burst"`
}

// Config contain the configuration settings for the workflow controller
type Config struct {

	// NodeEvents configures how node events are emitted
	NodeEvents NodeEvents `json:"nodeEvents,omitempty"`

	// Executor holds container customizations for the executor to use when running pods
	Executor *apiv1.Container `json:"executor,omitempty"`

	// MainContainer holds container customization for the main container
	MainContainer *apiv1.Container `json:"mainContainer,omitempty"`

	// KubeConfig specifies a kube config file for the wait & init containers
	KubeConfig *KubeConfig `json:"kubeConfig,omitempty"`

	// ArtifactRepository contains the default location of an artifact repository for container artifacts
	ArtifactRepository wfv1.ArtifactRepository `json:"artifactRepository,omitempty"`

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

	// MetricsConfig specifies configuration for metrics emission. Metrics are enabled and emitted on localhost:9090/metrics
	// by default.
	MetricsConfig MetricsConfig `json:"metricsConfig,omitempty"`

	// TelemetryConfig specifies configuration for telemetry emission. Telemetry is enabled and emitted in the same endpoint
	// as metrics by default, but can be overridden using this config.
	TelemetryConfig MetricsConfig `json:"telemetryConfig,omitempty"`

	// Parallelism limits the max total parallel workflows that can execute at the same time
	Parallelism int `json:"parallelism,omitempty"`

	// NamespaceParallelism limits the max workflows that can execute at the same time in a namespace
	NamespaceParallelism int `json:"namespaceParallelism,omitempty"`

	// ResourceRateLimit limits the rate at which pods are created
	ResourceRateLimit *ResourceRateLimit `json:"resourceRateLimit,omitempty"`

	// Persistence contains the workflow persistence DB configuration
	Persistence *PersistConfig `json:"persistence,omitempty"`

	// Links to related apps.
	Links []*wfv1.Link `json:"links,omitempty"`

	// WorkflowDefaults are values that will apply to all Workflows from this controller, unless overridden on the Workflow-level
	WorkflowDefaults *wfv1.Workflow `json:"workflowDefaults,omitempty"`

	// PodSpecLogStrategy enables the logging of podspec on controller log.
	PodSpecLogStrategy PodSpecLogStrategy `json:"podSpecLogStrategy,omitempty"`

	// PodGCGracePeriodSeconds specifies the duration in seconds before a terminating pod is forcefully killed.
	// Value must be non-negative integer. A zero value indicates that the pod will be forcefully terminated immediately.
	// Defaults to the Kubernetes default of 30 seconds.
	PodGCGracePeriodSeconds *int64 `json:"podGCGracePeriodSeconds,omitempty"`

	// PodGCDeleteDelayDuration specifies the duration in seconds before the pods in the GC queue get deleted.
	// Value must be non-negative integer. A zero value indicates that the pods will be deleted immediately.
	// Defaults to 5 seconds.
	PodGCDeleteDelayDuration *metav1.Duration `json:"podGCDeleteDelayDuration,omitempty"`

	// WorkflowRestrictions restricts the controller to executing Workflows that meet certain restrictions
	WorkflowRestrictions *WorkflowRestrictions `json:"workflowRestrictions,omitempty"`

	// Adding configurable initial delay (for K8S clusters with mutating webhooks) to prevent workflow getting modified by MWC.
	InitialDelay metav1.Duration `json:"initialDelay,omitempty"`

	// The command/args for each image, needed when the command is not specified and the emissary executor is used.
	// https://argoproj.github.io/argo-workflows/workflow-executors/#emissary-emissary
	Images map[string]Image `json:"images,omitempty"`

	RetentionPolicy *RetentionPolicy `json:"retentionPolicy,omitempty"`

	// NavColor is an ui navigation bar background color
	NavColor string `json:"navColor,omitempty"`

	// SSO in settings for single-sign on
	SSO SSOConfig `json:"sso,omitempty"`
}

func (c Config) GetExecutor() *apiv1.Container {
	if c.Executor != nil {
		return c.Executor
	}
	return &apiv1.Container{}
}

func (c Config) GetResourceRateLimit() ResourceRateLimit {
	if c.ResourceRateLimit != nil {
		return *c.ResourceRateLimit
	}
	return ResourceRateLimit{
		Limit: math.MaxFloat32,
		Burst: math.MaxInt32,
	}
}

func (c Config) GetPodGCDeleteDelayDuration() time.Duration {
	if c.PodGCDeleteDelayDuration == nil {
		return 5 * time.Second
	}

	return c.PodGCDeleteDelayDuration.Duration
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
	// ArchivelabelSelector holds LabelSelector to determine workflow persistence.
	ArchiveLabelSelector *metav1.LabelSelector `json:"archiveLabelSelector,omitempty"`
	// in days
	ArchiveTTL     TTL               `json:"archiveTTL,omitempty"`
	ClusterName    string            `json:"clusterName,omitempty"`
	ConnectionPool *ConnectionPool   `json:"connectionPool,omitempty"`
	PostgreSQL     *PostgreSQLConfig `json:"postgresql,omitempty"`
	MySQL          *MySQLConfig      `json:"mysql,omitempty"`
	SkipMigration  bool              `json:"skipMigration,omitempty"`
}

func (c PersistConfig) GetArchiveLabelSelector() (labels.Selector, error) {
	if c.ArchiveLabelSelector == nil {
		return labels.Everything(), nil
	}
	return metav1.LabelSelectorAsSelector(c.ArchiveLabelSelector)
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

type DatabaseConfig struct {
	Host           string                  `json:"host"`
	Port           int                     `json:"port,omitempty"`
	Database       string                  `json:"database"`
	TableName      string                  `json:"tableName,omitempty"`
	UsernameSecret apiv1.SecretKeySelector `json:"userNameSecret,omitempty"`
	PasswordSecret apiv1.SecretKeySelector `json:"passwordSecret,omitempty"`
}

func (c DatabaseConfig) GetHostname() string {
	if c.Port == 0 {
		return c.Host
	}
	return fmt.Sprintf("%s:%v", c.Host, c.Port)
}

type PostgreSQLConfig struct {
	DatabaseConfig
	SSL     bool   `json:"ssl,omitempty"`
	SSLMode string `json:"sslMode,omitempty"`
}

type MySQLConfig struct {
	DatabaseConfig
	Options map[string]string `json:"options,omitempty"`
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
	Port int `json:"port,omitempty"`
	// IgnoreErrors is a flag that instructs prometheus to ignore metric emission errors
	IgnoreErrors bool `json:"ignoreErrors,omitempty"`
	// Secure is a flag that starts the metrics servers using TLS
	Secure *bool `json:"secure,omitempty"`
}

func (mc MetricsConfig) GetSecure(defaultValue bool) bool {
	if mc.Secure != nil {
		return *mc.Secure
	}
	return defaultValue
}

type WorkflowRestrictions struct {
	TemplateReferencing TemplateReferencing `json:"templateReferencing,omitempty"`
}

type TemplateReferencing string

const (
	TemplateReferencingStrict TemplateReferencing = "Strict"
	TemplateReferencingSecure TemplateReferencing = "Secure"
)

func (req *WorkflowRestrictions) MustUseReference() bool {
	if req == nil {
		return false
	}
	return req.TemplateReferencing == TemplateReferencingStrict || req.TemplateReferencing == TemplateReferencingSecure
}

func (req *WorkflowRestrictions) MustNotChangeSpec() bool {
	if req == nil {
		return false
	}
	return req.TemplateReferencing == TemplateReferencingSecure
}
