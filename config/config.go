package config

import (
	"fmt"
	"math"
	"path"
	"time"

	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v3/workflow/common"
)

var EmptyConfigFunc = func() interface{} { return &Config{} }

type ResourceRateLimit struct {
	Limit float64 `json:"limit"`
	Burst int     `json:"burst"`
}

// Config contain the configuration settings for the workflow controller
type Config struct {

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

	// MainContainer holds container customization for the main container
	MainContainer *apiv1.Container `json:"mainContainer,omitempty"`

	// KubeConfig specifies a kube config file for the wait & init containers
	KubeConfig *KubeConfig `json:"kubeConfig,omitempty"`

	// ContainerRuntimeExecutor specifies the container runtime interface to use, default is docker
	ContainerRuntimeExecutor string `json:"containerRuntimeExecutor,omitempty"`

	ContainerRuntimeExecutors ContainerRuntimeExecutors `json:"containerRuntimeExecutors,omitempty"`

	// KubeletPort is needed when using the kubelet containerRuntimeExecutor, default to 10250
	KubeletPort int `json:"kubeletPort,omitempty"`

	// KubeletInsecure disable the TLS verification of the kubelet containerRuntimeExecutor, default to false
	KubeletInsecure bool `json:"kubeletInsecure,omitempty"`

	// ArtifactRepository contains the default location of an artifact repository for container artifacts
	ArtifactRepository ArtifactRepository `json:"artifactRepository,omitempty"`

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

	// NamespaceParallelism limits the max workflows that can execute at the same time in a namespace
	NamespaceParallelism int `json:"namespaceParallelism,omitempty"`

	// ResourceRateLimit limits the rate at which pods are created
	ResourceRateLimit *ResourceRateLimit `json:"resourceRateLimit,omitempty"`

	// Persistence contains the workflow persistence DB configuration
	Persistence *PersistConfig `json:"persistence,omitempty"`

	// Links to related apps.
	Links []*wfv1.Link `json:"links,omitempty"`

	// Config customized Docker Sock path
	DockerSockPath string `json:"dockerSockPath,omitempty"`

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
}

func (c Config) GetContainerRuntimeExecutor(labels labels.Labels) (string, error) {
	name, err := c.ContainerRuntimeExecutors.Select(labels)
	if err != nil {
		return "", err
	}
	if name != "" {
		return name, nil
	}
	return c.ContainerRuntimeExecutor, nil
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

// ArtifactRepository represents an artifact repository in which a controller will store its artifacts
type ArtifactRepository struct {
	// ArchiveLogs enables log archiving
	ArchiveLogs *bool `json:"archiveLogs,omitempty"`
	// S3 stores artifact in a S3-compliant object store
	S3 *S3ArtifactRepository `json:"s3,omitempty"`
	// Artifactory stores artifacts to JFrog Artifactory
	Artifactory *ArtifactoryArtifactRepository `json:"artifactory,omitempty"`
	// HDFS stores artifacts in HDFS
	HDFS *HDFSArtifactRepository `json:"hdfs,omitempty"`
	// OSS stores artifact in a OSS-compliant object store
	OSS *OSSArtifactRepository `json:"oss,omitempty"`
	// GCS stores artifact in a GCS object store
	GCS *GCSArtifactRepository `json:"gcs,omitempty"`
}

func (a *ArtifactRepository) IsArchiveLogs() bool {
	return a != nil && a.ArchiveLogs != nil && *a.ArchiveLogs
}

type ArtifactRepositoryType interface {
	IntoArtifactLocation(l *wfv1.ArtifactLocation)
}

func (a *ArtifactRepository) Get() ArtifactRepositoryType {
	if a == nil {
		return nil
	} else if a.Artifactory != nil {
		return a.Artifactory
	} else if a.GCS != nil {
		return a.GCS
	} else if a.HDFS != nil {
		return a.HDFS
	} else if a.OSS != nil {
		return a.OSS
	} else if a.S3 != nil {
		return a.S3
	}
	return nil
}

// ToArtifactLocation returns the artifact location set with default template key:
// key = `{{workflow.name}}/{{pod.name}}`
func (a *ArtifactRepository) ToArtifactLocation() *wfv1.ArtifactLocation {
	if a == nil {
		return nil
	}
	l := &wfv1.ArtifactLocation{ArchiveLogs: a.ArchiveLogs}
	v := a.Get()
	if v != nil {
		v.IntoArtifactLocation(l)
	}
	return l
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

// S3ArtifactRepository defines the controller configuration for an S3 artifact repository
type S3ArtifactRepository struct {
	wfv1.S3Bucket `json:",inline"`

	// KeyFormat is defines the format of how to store keys. Can reference workflow variables
	KeyFormat string `json:"keyFormat,omitempty"`

	// KeyPrefix is prefix used as part of the bucket key in which the controller will store artifacts.
	// DEPRECATED. Use KeyFormat instead
	KeyPrefix string `json:"keyPrefix,omitempty"`
}

func (r *S3ArtifactRepository) IntoArtifactLocation(l *wfv1.ArtifactLocation) {
	k := r.KeyFormat
	if k == "" {
		k = path.Join(r.KeyPrefix, common.DefaultArchivePattern)
	}
	l.S3 = &wfv1.S3Artifact{S3Bucket: r.S3Bucket, Key: k}
}

// OSSArtifactRepository defines the controller configuration for an OSS artifact repository
type OSSArtifactRepository struct {
	wfv1.OSSBucket `json:",inline"`

	// KeyFormat is defines the format of how to store keys. Can reference workflow variables
	KeyFormat string `json:"keyFormat,omitempty"`
}

func (r *OSSArtifactRepository) IntoArtifactLocation(l *wfv1.ArtifactLocation) {
	k := r.KeyFormat
	if k == "" {
		k = common.DefaultArchivePattern
	}
	l.OSS = &wfv1.OSSArtifact{OSSBucket: r.OSSBucket, Key: k}
}

// GCSArtifactRepository defines the controller configuration for a GCS artifact repository
type GCSArtifactRepository struct {
	wfv1.GCSBucket `json:",inline"`

	// KeyFormat is defines the format of how to store keys. Can reference workflow variables
	KeyFormat string `json:"keyFormat,omitempty"`
}

func (r *GCSArtifactRepository) IntoArtifactLocation(l *wfv1.ArtifactLocation) {
	k := r.KeyFormat
	if k == "" {
		k = common.DefaultArchivePattern
	}
	l.GCS = &wfv1.GCSArtifact{GCSBucket: r.GCSBucket, Key: k}
}

// ArtifactoryArtifactRepository defines the controller configuration for an artifactory artifact repository
type ArtifactoryArtifactRepository struct {
	wfv1.ArtifactoryAuth `json:",inline"`
	// RepoURL is the url for artifactory repo.
	RepoURL string `json:"repoURL,omitempty"`
}

func (r *ArtifactoryArtifactRepository) IntoArtifactLocation(l *wfv1.ArtifactLocation) {
	u := ""
	if r.RepoURL != "" {
		u = r.RepoURL + "/"
	}
	u = fmt.Sprintf("%s%s", u, common.DefaultArchivePattern)
	l.Artifactory = &wfv1.ArtifactoryArtifact{ArtifactoryAuth: r.ArtifactoryAuth, URL: u}
}

// HDFSArtifactRepository defines the controller configuration for an HDFS artifact repository
type HDFSArtifactRepository struct {
	wfv1.HDFSConfig `json:",inline"`

	// PathFormat is defines the format of path to store a file. Can reference workflow variables
	PathFormat string `json:"pathFormat,omitempty"`

	// Force copies a file forcibly even if it exists (default: false)
	Force bool `json:"force,omitempty"`
}

func (r *HDFSArtifactRepository) IntoArtifactLocation(l *wfv1.ArtifactLocation) {
	p := r.PathFormat
	if p == "" {
		p = common.DefaultArchivePattern
	}
	l.HDFS = &wfv1.HDFSArtifact{HDFSConfig: r.HDFSConfig, Path: p, Force: r.Force}
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
