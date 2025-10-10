package config

import (
	"fmt"
	"math"
	"net/url"
	"time"

	metricsdk "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
)

type ResourceRateLimit struct {
	// Limit is the maximum rate at which pods can be created
	Limit float64 `json:"limit"`
	// Burst allows temporary spikes above the limit
	Burst int `json:"burst"`
}

// Config contains the root of the configuration settings for the workflow controller
// as read from the ConfigMap called workflow-controller-configmap
type Config struct {
	// NodeEvents configures how node events are emitted
	NodeEvents NodeEvents `json:"nodeEvents,omitempty"`

	// WorkflowEvents configures how workflow events are emitted
	WorkflowEvents WorkflowEvents `json:"workflowEvents,omitempty"`

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

	// Columns are custom columns that will be exposed in the Workflow List View.
	Columns []*wfv1.Column `json:"columns,omitempty"`

	// WorkflowDefaults are values that will apply to all Workflows from this controller, unless overridden on the Workflow-level
	WorkflowDefaults *wfv1.Workflow `json:"workflowDefaults,omitempty"`

	// PodSpecLogStrategy enables the logging of podspec on controller log.
	PodSpecLogStrategy PodSpecLogStrategy `json:"podSpecLogStrategy,omitempty"`

	// PodGCGracePeriodSeconds specifies the duration in seconds before a terminating pod is forcefully killed.
	// Value must be non-negative integer. A zero value indicates that the pod will be forcefully terminated immediately.
	// Defaults to the Kubernetes default of 30 seconds.
	PodGCGracePeriodSeconds *int64 `json:"podGCGracePeriodSeconds,omitempty"`

	// PodGCDeleteDelayDuration specifies the duration before pods in the GC queue get deleted.
	// Value must be non-negative. A zero value indicates that the pods will be deleted immediately.
	// Defaults to 5 seconds.
	PodGCDeleteDelayDuration *metav1.Duration `json:"podGCDeleteDelayDuration,omitempty"`

	// WorkflowRestrictions restricts the controller to executing Workflows that meet certain restrictions
	WorkflowRestrictions *WorkflowRestrictions `json:"workflowRestrictions,omitempty"`

	// Adds configurable initial delay (for K8S clusters with mutating webhooks) to prevent workflow getting modified by MWC.
	InitialDelay metav1.Duration `json:"initialDelay,omitempty"`

	// The command/args for each image, needed when the command is not specified and the emissary executor is used.
	// https://argo-workflows.readthedocs.io/en/latest/workflow-executors/#emissary-emissary
	Images map[string]Image `json:"images,omitempty"`

	// Workflow retention by number of workflows
	RetentionPolicy *RetentionPolicy `json:"retentionPolicy,omitempty"`

	// NavColor is an ui navigation bar background color
	NavColor string `json:"navColor,omitempty"`

	// SSO in settings for single-sign on
	SSO SSOConfig `json:"sso,omitempty"`

	// Synchronization via databases config
	Synchronization *SyncConfig `json:"synchronization,omitempty"`
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

func (c Config) ValidateProtocol(inputProtocol string, allowedProtocol []string) error {
	for _, protocol := range allowedProtocol {
		if inputProtocol == protocol {
			return nil
		}
	}
	return fmt.Errorf("protocol %s is not allowed", inputProtocol)
}

func (c *Config) Sanitize(allowedProtocol []string) error {
	links := c.Links

	for _, link := range links {
		// We only validate user-supplied URL but not encode/decode it
		// see 2.4.2 on https://www.ietf.org/rfc/rfc2396.txt
		u, err := url.Parse(link.URL)
		if err != nil {
			return err
		}
		err = c.ValidateProtocol(u.Scheme, allowedProtocol)
		if err != nil {
			return err
		}
	}
	return nil
}

// PodSpecLogStrategy contains the configuration for logging the pod spec in controller log for debugging purpose
type PodSpecLogStrategy struct {
	FailedPod bool `json:"failedPod,omitempty"`
	AllPods   bool `json:"allPods,omitempty"`
}

// KubeConfig is used for wait & init sidecar containers to communicate with a k8s apiserver by an out-of-cluster method;
// it is used when the workflow controller is in a different cluster from the workflow workloads
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

// DBConfig contains database configuration settings
type DBConfig struct {
	// PostgreSQL configuration for PostgreSQL database, don't use MySQL at the same time
	PostgreSQL *PostgreSQLConfig `json:"postgresql,omitempty"`
	// MySQL configuration for MySQL database, don't use PostgreSQL at the same time
	MySQL *MySQLConfig `json:"mysql,omitempty"`
	// Pooled connection settings for all types of database connections
	ConnectionPool *ConnectionPool `json:"connectionPool,omitempty"`
}

// PersistConfig contains workflow persistence configuration
type PersistConfig struct {
	DBConfig
	// NodeStatusOffload saves node status only to the persistence DB to avoid the 1MB limit in etcd
	NodeStatusOffload bool `json:"nodeStatusOffLoad,omitempty"`
	// Archive completed and Workflows to persistence so you can access them after they're
	// removed from kubernetes
	Archive bool `json:"archive,omitempty"`
	// ArchiveLabelSelector holds LabelSelector to determine which Workflows to archive
	ArchiveLabelSelector *metav1.LabelSelector `json:"archiveLabelSelector,omitempty"`
	// ArchiveTTL is the time to live for archived Workflows
	ArchiveTTL TTL `json:"archiveTTL,omitempty"`
	// ClusterName is the name of the cluster (or technically controller) for the persistence database
	ClusterName string `json:"clusterName,omitempty"`
	// SkipMigration skips database migration even if needed
	SkipMigration bool `json:"skipMigration,omitempty"`
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

// SyncConfig contains synchronization configuration for database locks (semaphores and mutexes)
type SyncConfig struct {
	DBConfig
	// EnableAPI enables the database synchronization API
	EnableAPI bool `json:"enableAPI,omitempty"`
	// ControllerName sets a unique name for this controller instance
	ControllerName string `json:"controllerName"`
	// SkipMigration skips database migration if needed
	SkipMigration bool `json:"skipMigration,omitempty"`
	// LimitTableName customizes the table name for semaphore limits, if not set, the default value is "sync_limit"
	LimitTableName string `json:"limitTableName,omitempty"`
	// StateTableName customizes the table name for current lock state, if not set, the default value is "sync_state"
	StateTableName string `json:"stateTableName,omitempty"`
	// ControllerTableName customizes the table name for controller heartbeats, if not set, the default value is "sync_controller"
	ControllerTableName string `json:"controllerTableName,omitempty"`
	// LockTableName customizes the table name for lock coordination data, if not set, the default value is "sync_lock"
	LockTableName string `json:"lockTableName,omitempty"`
	// PollSeconds specifies how often to check for lock changes, if not set, the default value is 5 seconds
	PollSeconds *int `json:"pollSeconds,omitempty"`
	// HeartbeatSeconds specifies how often to update controller heartbeat, if not set, the default value is 60 seconds
	HeartbeatSeconds *int `json:"heartbeatSeconds,omitempty"`
	// InactiveControllerSeconds specifies when to consider a controller dead, if not set, the default value is 300 seconds
	InactiveControllerSeconds *int `json:"inactiveControllerSeconds,omitempty"`
	// SemaphoreLimitCacheSeconds specifies the duration in seconds before the workflow controller will re-fetch the limit
	// for a semaphore from its associated data source. Defaults to 0 seconds (re-fetch every time the semaphore is checked).
	SemaphoreLimitCacheSeconds *int64 `json:"semaphoreLimitCacheSeconds,omitempty"`
}

// ConnectionPool contains database connection pool settings
type ConnectionPool struct {
	// MaxIdleConns sets the maximum number of idle connections in the pool
	MaxIdleConns int `json:"maxIdleConns,omitempty"`
	// MaxOpenConns sets the maximum number of open connections to the database
	MaxOpenConns int `json:"maxOpenConns,omitempty"`
	// ConnMaxLifetime sets the maximum amount of time a connection may be reused
	ConnMaxLifetime TTL `json:"connMaxLifetime,omitempty"`
}

// DatabaseConfig contains common database connection settings
type DatabaseConfig struct {
	// Host is the database server hostname
	Host string `json:"host"`
	// Port is the database server port
	Port int `json:"port,omitempty"`
	// Database is the name of the database to connect to
	Database string `json:"database"`
	// TableName is the name of the table to use, must be set
	TableName string `json:"tableName,omitempty"`
	// UsernameSecret references a secret containing the database username
	UsernameSecret apiv1.SecretKeySelector `json:"userNameSecret,omitempty"`
	// PasswordSecret references a secret containing the database password
	PasswordSecret apiv1.SecretKeySelector `json:"passwordSecret,omitempty"`
}

func (c DatabaseConfig) GetHostname() string {
	if c.Port == 0 {
		return c.Host
	}
	return fmt.Sprintf("%s:%v", c.Host, c.Port)
}

// PostgreSQLConfig contains PostgreSQL-specific database configuration
type PostgreSQLConfig struct {
	DatabaseConfig
	// SSL enables SSL connection to the database
	SSL bool `json:"ssl,omitempty"`
	// SSLMode specifies the SSL mode (disable, require, verify-ca, verify-full)
	SSLMode string `json:"sslMode,omitempty"`
}

// MySQLConfig contains MySQL-specific database configuration
type MySQLConfig struct {
	DatabaseConfig
	// Options contains additional MySQL connection options
	Options map[string]string `json:"options,omitempty"`
}

// MetricModifier are modifiers for an individual named metric to change their behaviour
type MetricModifier struct {
	// Disabled disables the emission of this metric completely
	Disabled bool `json:"disabled,omitempty"`
	// DisabledAttributes lists labels for this metric to remove that attributes to save on cardinality
	DisabledAttributes []string `json:"disabledAttributes"`
	// HistogramBuckets allow configuring of the buckets used in a histogram
	// Has no effect on non-histogram buckets
	HistogramBuckets []float64 `json:"histogramBuckets,omitempty"`
}

// MetricsTemporality defines the temporality of OpenTelemetry metrics
type MetricsTemporality string

const (
	// MetricsTemporalityCumulative indicates cumulative temporality
	MetricsTemporalityCumulative MetricsTemporality = "Cumulative"
	// MetricsTemporalityDelta indicates delta temporality
	MetricsTemporalityDelta MetricsTemporality = "Delta"
)

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
	// Secure is a flag that starts the metrics servers using TLS, defaults to true
	Secure *bool `json:"secure,omitempty"`
	// Modifiers configure metrics by name
	Modifiers map[string]MetricModifier `json:"modifiers,omitempty"`
	// Temporality of the OpenTelemetry metrics.
	// Enum of Cumulative or Delta, defaulting to Cumulative.
	// No effect on Prometheus metrics, which are always Cumulative.
	Temporality MetricsTemporality `json:"temporality,omitempty"`
}

func (mc *MetricsConfig) GetSecure(defaultValue bool) bool {
	if mc.Secure != nil {
		return *mc.Secure
	}
	return defaultValue
}

func (mc *MetricsConfig) GetTemporality() metricsdk.TemporalitySelector {
	switch mc.Temporality {
	case MetricsTemporalityCumulative:
		return func(metricsdk.InstrumentKind) metricdata.Temporality {
			return metricdata.CumulativeTemporality
		}
	case MetricsTemporalityDelta:
		return func(metricsdk.InstrumentKind) metricdata.Temporality {
			return metricdata.DeltaTemporality
		}
	default:
		return metricsdk.DefaultTemporalitySelector
	}
}

// WorkflowRestrictions contains restrictions for workflow execution
type WorkflowRestrictions struct {
	// TemplateReferencing controls how templates can be referenced
	TemplateReferencing TemplateReferencing `json:"templateReferencing,omitempty"`
}

// TemplateReferencing defines how templates can be referenced in workflows
type TemplateReferencing string

const (
	// TemplateReferencingStrict requires templates to be referenced, not embedded
	TemplateReferencingStrict TemplateReferencing = "Strict"
	// TemplateReferencingSecure requires templates to be referenced and prevents spec changes
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
