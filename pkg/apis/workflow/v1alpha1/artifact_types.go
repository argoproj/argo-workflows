package v1alpha1

import (
	"fmt"
	"net/url"
	"path"
	"reflect"

	apiv1 "k8s.io/api/core/v1"
)

type Artifacts []Artifact

func (a Artifacts) GetArtifactByName(name string) *Artifact {
	for _, art := range a {
		if art.Name == name {
			return &art
		}
	}
	return nil
}

// Artifact indicates an artifact to place at a specified path
type Artifact struct {
	// name of the artifact. must be unique within a template's inputs/outputs.
	Name string `json:"name" protobuf:"bytes,1,opt,name=name"`

	// Path is the container path to the artifact
	Path string `json:"path,omitempty" protobuf:"bytes,2,opt,name=path"`

	// mode bits to use on this file, must be a value between 0 and 0777
	// set when loading input artifacts.
	Mode *int32 `json:"mode,omitempty" protobuf:"varint,3,opt,name=mode"`

	// From allows an artifact to reference an artifact from a previous step
	From string `json:"from,omitempty" protobuf:"bytes,4,opt,name=from"`

	// ArtifactLocation contains the location of the artifact
	ArtifactLocation `json:",inline" protobuf:"bytes,5,opt,name=artifactLocation"`

	// GlobalName exports an output artifact to the global scope, making it available as
	// '{{workflow.outputs.artifacts.XXXX}} and in workflow.status.outputs.artifacts
	GlobalName string `json:"globalName,omitempty" protobuf:"bytes,6,opt,name=globalName"`

	// Archive controls how the artifact will be saved to the artifact repository.
	Archive *ArchiveStrategy `json:"archive,omitempty" protobuf:"bytes,7,opt,name=archive"`

	// Make Artifacts optional, if Artifacts doesn't generate or exist
	Optional bool `json:"optional,omitempty" protobuf:"varint,8,opt,name=optional"`

	// SubPath allows an artifact to be sourced from a subpath within the specified source
	SubPath string `json:"subPath,omitempty" protobuf:"bytes,9,opt,name=subPath"`

	// If mode is set, apply the permission recursively into the artifact if it is a folder
	RecurseMode bool `json:"recurseMode,omitempty" protobuf:"varint,10,opt,name=recurseMode"`
}

// ArtifactLocation describes a location for a single or multiple artifacts.
// It is used as single artifact in the context of inputs/outputs (e.g. outputs.artifacts.artname).
// It is also used to describe the location of multiple artifacts such as the archive location
// of a single workflow step, which the executor will use as a default location to store its files.
type ArtifactLocation struct {
	// ArchiveLogs indicates if the container logs should be archived
	ArchiveLogs *bool `json:"archiveLogs,omitempty" protobuf:"varint,1,opt,name=archiveLogs"`

	// S3 contains S3 artifact location details
	S3 *S3Artifact `json:"s3,omitempty" protobuf:"bytes,2,opt,name=s3"`

	// Git contains git artifact location details
	Git *GitArtifact `json:"git,omitempty" protobuf:"bytes,3,opt,name=git"`

	// HTTP contains HTTP artifact location details
	HTTP *HTTPArtifact `json:"http,omitempty" protobuf:"bytes,4,opt,name=http"`

	// Artifactory contains artifactory artifact location details
	Artifactory *ArtifactoryArtifact `json:"artifactory,omitempty" protobuf:"bytes,5,opt,name=artifactory"`

	// HDFS contains HDFS artifact location details
	HDFS *HDFSArtifact `json:"hdfs,omitempty" protobuf:"bytes,6,opt,name=hdfs"`

	// Raw contains raw artifact location details
	Raw *RawArtifact `json:"raw,omitempty" protobuf:"bytes,7,opt,name=raw"`

	// OSS contains OSS artifact location details
	OSS *OSSArtifact `json:"oss,omitempty" protobuf:"bytes,8,opt,name=oss"`

	// GCS contains GCS artifact location details
	GCS *GCSArtifact `json:"gcs,omitempty" protobuf:"bytes,9,opt,name=gcs"`
}

func (a *ArtifactLocation) Get() ArtifactLocationType {
	if a == nil {
		return nil
	} else if a.Artifactory != nil {
		return a.Artifactory
	} else if a.Git != nil {
		return a.Git
	} else if a.GCS != nil {
		return a.GCS
	} else if a.HDFS != nil {
		return a.HDFS
	} else if a.HTTP != nil {
		return a.HTTP
	} else if a.OSS != nil {
		return a.OSS
	} else if a.Raw != nil {
		return a.Raw
	} else if a.S3 != nil {
		return a.S3
	}
	return nil
}

// SetType sets the type of the artifact to type the argument.
// Any existing value is deleted.
func (a *ArtifactLocation) SetType(x ArtifactLocationType) error {
	switch v := x.(type) {
	case *ArtifactoryArtifact:
		a.Artifactory = &ArtifactoryArtifact{}
	case *GCSArtifact:
		a.GCS = &GCSArtifact{}
	case *HDFSArtifact:
		a.HDFS = &HDFSArtifact{}
	case *HTTPArtifact:
		a.HTTP = &HTTPArtifact{}
	case *OSSArtifact:
		a.OSS = &OSSArtifact{}
	case *RawArtifact:
		a.Raw = &RawArtifact{}
	case *S3Artifact:
		a.S3 = &S3Artifact{}
	default:
		return fmt.Errorf("set type not supported for type: %v", reflect.TypeOf(v))
	}
	return nil
}

func (a *ArtifactLocation) HasLocationOrKey() bool {
	return a.HasLocation() || a.HasKey()
}

// HasKey returns whether or not an artifact has a key. They may or may not also HasLocation.
func (a *ArtifactLocation) HasKey() bool {
	key, _ := a.GetKey()
	return key != ""
}

// set the key to a new value, use path.Join to combine items
func (a *ArtifactLocation) SetKey(key string) error {
	v := a.Get()
	if v == nil {
		return keyUnsupportedErr
	}
	return v.SetKey(key)
}

func (a *ArtifactLocation) AppendToKey(x string) error {
	key, err := a.GetKey()
	if err != nil {
		return err
	}
	return a.SetKey(path.Join(key, x))
}

// Relocate copies all location info from the parameter, except the key.
// But only if it does not have a location already.
func (a *ArtifactLocation) Relocate(l *ArtifactLocation) error {
	if a.HasLocation() {
		return nil
	}
	if l == nil {
		return fmt.Errorf("template artifact location not set")
	}
	key, err := a.GetKey()
	if err != nil {
		return err
	}
	*a = *l.DeepCopy()
	return a.SetKey(key)
}

// HasLocation whether or not an artifact has a *full* location defined
// An artifact that has a location implicitly has a key (i.e. HasKey() == true).
func (a *ArtifactLocation) HasLocation() bool {
	v := a.Get()
	return v != nil && v.HasLocation()
}

func (a *ArtifactLocation) IsArchiveLogs() bool {
	return a != nil && a.ArchiveLogs != nil && *a.ArchiveLogs
}

func (a *ArtifactLocation) GetKey() (string, error) {
	v := a.Get()
	if v == nil {
		return "", keyUnsupportedErr
	}
	return v.GetKey()
}

// +protobuf.options.(gogoproto.goproto_stringer)=false
type ArtifactRepositoryRef struct {
	// The name of the config map. Defaults to "artifact-repositories".
	ConfigMap string `json:"configMap,omitempty" protobuf:"bytes,1,opt,name=configMap"`
	// The config map key. Defaults to the value of the "workflows.argoproj.io/default-artifact-repository" annotation.
	Key string `json:"key,omitempty" protobuf:"bytes,2,opt,name=key"`
}

func (r *ArtifactRepositoryRef) GetConfigMapOr(configMap string) string {
	if r == nil || r.ConfigMap == "" {
		return configMap
	}
	return r.ConfigMap
}

func (r *ArtifactRepositoryRef) GetKeyOr(key string) string {
	if r == nil || r.Key == "" {
		return key
	}
	return r.Key
}

func (r *ArtifactRepositoryRef) String() string {
	if r == nil {
		return "nil"
	}
	return fmt.Sprintf("%s#%s", r.ConfigMap, r.Key)
}

var DefaultArtifactRepositoryRefStatus = &ArtifactRepositoryRefStatus{Default: true}

// +protobuf.options.(gogoproto.goproto_stringer)=false
type ArtifactRepositoryRefStatus struct {
	ArtifactRepositoryRef `json:",inline" protobuf:"bytes,1,opt,name=artifactRepositoryRef"`
	// The namespace of the config map. Defaults to the workflow's namespace, or the controller's namespace (if found).
	Namespace string `json:"namespace,omitempty" protobuf:"bytes,2,opt,name=namespace"`
	// If this ref represents the default artifact repository, rather than a config map.
	Default bool `json:"default,omitempty" protobuf:"varint,3,opt,name=default"`
}

func (r *ArtifactRepositoryRefStatus) String() string {
	if r == nil {
		return "nil"
	}
	if r.Default {
		return "default-artifact-repository"
	}
	return fmt.Sprintf("%s/%s", r.Namespace, r.ArtifactRepositoryRef.String())
}

// S3Bucket contains the access information required for interfacing with an S3 bucket
type S3Bucket struct {
	// Endpoint is the hostname of the bucket endpoint
	Endpoint string `json:"endpoint,omitempty" protobuf:"bytes,1,opt,name=endpoint"`

	// Bucket is the name of the bucket
	Bucket string `json:"bucket,omitempty" protobuf:"bytes,2,opt,name=bucket"`

	// Region contains the optional bucket region
	Region string `json:"region,omitempty" protobuf:"bytes,3,opt,name=region"`

	// Insecure will connect to the service with TLS
	Insecure *bool `json:"insecure,omitempty" protobuf:"varint,4,opt,name=insecure"`

	// AccessKeySecret is the secret selector to the bucket's access key
	AccessKeySecret *apiv1.SecretKeySelector `json:"accessKeySecret,omitempty" protobuf:"bytes,5,opt,name=accessKeySecret"`

	// SecretKeySecret is the secret selector to the bucket's secret key
	SecretKeySecret *apiv1.SecretKeySelector `json:"secretKeySecret,omitempty" protobuf:"bytes,6,opt,name=secretKeySecret"`

	// RoleARN is the Amazon Resource Name (ARN) of the role to assume.
	RoleARN string `json:"roleARN,omitempty" protobuf:"bytes,7,opt,name=roleARN"`

	// UseSDKCreds tells the driver to figure out credentials based on sdk defaults.
	UseSDKCreds bool `json:"useSDKCreds,omitempty" protobuf:"varint,8,opt,name=useSDKCreds"`

	// CreateBucketIfNotPresent tells the driver to attempt to create the S3 bucket for output artifacts, if it doesn't exist
	CreateBucketIfNotPresent *CreateS3BucketOptions `json:"createBucketIfNotPresent,omitempty" protobuf:"bytes,9,opt,name=createBucketIfNotPresent"`
}

// CreateS3BucketOptions options used to determine automatic automatic bucket-creation process
type CreateS3BucketOptions struct {
	// ObjectLocking Enable object locking
	ObjectLocking bool `json:"objectLocking,omitempty" protobuf:"varint,3,opt,name=objectLocking"`
}

// S3Artifact is the location of an S3 artifact
type S3Artifact struct {
	S3Bucket `json:",inline" protobuf:"bytes,1,opt,name=s3Bucket"`

	// Key is the key in the bucket where the artifact resides
	Key string `json:"key,omitempty" protobuf:"bytes,2,opt,name=key"`
}

func (s *S3Artifact) GetKey() (string, error) {
	return s.Key, nil
}

func (s *S3Artifact) SetKey(key string) error {
	s.Key = key
	return nil
}

func (s *S3Artifact) HasLocation() bool {
	return s != nil && s.Endpoint != "" && s.Bucket != "" && s.Key != ""
}

// GitArtifact is the location of an git artifact
type GitArtifact struct {
	// Repo is the git repository
	Repo string `json:"repo" protobuf:"bytes,1,opt,name=repo"`

	// Revision is the git commit, tag, branch to checkout
	Revision string `json:"revision,omitempty" protobuf:"bytes,2,opt,name=revision"`

	// Depth specifies clones/fetches should be shallow and include the given
	// number of commits from the branch tip
	Depth *uint64 `json:"depth,omitempty" protobuf:"bytes,3,opt,name=depth"`

	// Fetch specifies a number of refs that should be fetched before checkout
	Fetch []string `json:"fetch,omitempty" protobuf:"bytes,4,rep,name=fetch"`

	// UsernameSecret is the secret selector to the repository username
	UsernameSecret *apiv1.SecretKeySelector `json:"usernameSecret,omitempty" protobuf:"bytes,5,opt,name=usernameSecret"`

	// PasswordSecret is the secret selector to the repository password
	PasswordSecret *apiv1.SecretKeySelector `json:"passwordSecret,omitempty" protobuf:"bytes,6,opt,name=passwordSecret"`

	// SSHPrivateKeySecret is the secret selector to the repository ssh private key
	SSHPrivateKeySecret *apiv1.SecretKeySelector `json:"sshPrivateKeySecret,omitempty" protobuf:"bytes,7,opt,name=sshPrivateKeySecret"`

	// InsecureIgnoreHostKey disables SSH strict host key checking during git clone
	InsecureIgnoreHostKey bool `json:"insecureIgnoreHostKey,omitempty" protobuf:"varint,8,opt,name=insecureIgnoreHostKey"`
}

func (g *GitArtifact) HasLocation() bool {
	return g != nil && g.Repo != ""
}

func (g *GitArtifact) GetKey() (string, error) {
	return "", keyUnsupportedErr
}

func (g *GitArtifact) SetKey(string) error {
	return keyUnsupportedErr
}

func (g *GitArtifact) GetDepth() int {
	if g == nil || g.Depth == nil {
		return 0
	}
	return int(*g.Depth)
}

// ArtifactoryAuth describes the secret selectors required for authenticating to artifactory
type ArtifactoryAuth struct {
	// UsernameSecret is the secret selector to the repository username
	UsernameSecret *apiv1.SecretKeySelector `json:"usernameSecret,omitempty" protobuf:"bytes,1,opt,name=usernameSecret"`

	// PasswordSecret is the secret selector to the repository password
	PasswordSecret *apiv1.SecretKeySelector `json:"passwordSecret,omitempty" protobuf:"bytes,2,opt,name=passwordSecret"`
}

// ArtifactoryArtifact is the location of an artifactory artifact
type ArtifactoryArtifact struct {
	// URL of the artifact
	URL             string `json:"url" protobuf:"bytes,1,opt,name=url"`
	ArtifactoryAuth `json:",inline" protobuf:"bytes,2,opt,name=artifactoryAuth"`
}

//func (a *ArtifactoryArtifact) String() string {
//	return a.URL
//}
func (a *ArtifactoryArtifact) GetKey() (string, error) {
	u, err := url.Parse(a.URL)
	if err != nil {
		return "", err
	}
	return u.Path, nil
}

func (a *ArtifactoryArtifact) SetKey(key string) error {
	u, err := url.Parse(a.URL)
	if err != nil {
		return err
	}
	u.Path = key
	a.URL = u.String()
	return nil
}

func (a *ArtifactoryArtifact) HasLocation() bool {
	return a != nil && a.URL != ""
}

// HDFSArtifact is the location of an HDFS artifact
type HDFSArtifact struct {
	HDFSConfig `json:",inline" protobuf:"bytes,1,opt,name=hDFSConfig"`

	// Path is a file path in HDFS
	Path string `json:"path" protobuf:"bytes,2,opt,name=path"`

	// Force copies a file forcibly even if it exists (default: false)
	Force bool `json:"force,omitempty" protobuf:"varint,3,opt,name=force"`
}

func (h *HDFSArtifact) GetKey() (string, error) {
	return h.Path, nil
}

func (g *HDFSArtifact) SetKey(key string) error {
	g.Path = key
	return nil
}

func (h *HDFSArtifact) HasLocation() bool {
	return h != nil && len(h.Addresses) > 0
}

// HDFSConfig is configurations for HDFS
type HDFSConfig struct {
	HDFSKrbConfig `json:",inline" protobuf:"bytes,1,opt,name=hDFSKrbConfig"`

	// Addresses is accessible addresses of HDFS name nodes
	Addresses []string `json:"addresses,omitempty" protobuf:"bytes,2,rep,name=addresses"`

	// HDFSUser is the user to access HDFS file system.
	// It is ignored if either ccache or keytab is used.
	HDFSUser string `json:"hdfsUser,omitempty" protobuf:"bytes,3,opt,name=hdfsUser"`
}

// HDFSKrbConfig is auth configurations for Kerberos
type HDFSKrbConfig struct {
	// KrbCCacheSecret is the secret selector for Kerberos ccache
	// Either ccache or keytab can be set to use Kerberos.
	KrbCCacheSecret *apiv1.SecretKeySelector `json:"krbCCacheSecret,omitempty" protobuf:"bytes,1,opt,name=krbCCacheSecret"`

	// KrbKeytabSecret is the secret selector for Kerberos keytab
	// Either ccache or keytab can be set to use Kerberos.
	KrbKeytabSecret *apiv1.SecretKeySelector `json:"krbKeytabSecret,omitempty" protobuf:"bytes,2,opt,name=krbKeytabSecret"`

	// KrbUsername is the Kerberos username used with Kerberos keytab
	// It must be set if keytab is used.
	KrbUsername string `json:"krbUsername,omitempty" protobuf:"bytes,3,opt,name=krbUsername"`

	// KrbRealm is the Kerberos realm used with Kerberos keytab
	// It must be set if keytab is used.
	KrbRealm string `json:"krbRealm,omitempty" protobuf:"bytes,4,opt,name=krbRealm"`

	// KrbConfig is the configmap selector for Kerberos config as string
	// It must be set if either ccache or keytab is used.
	KrbConfigConfigMap *apiv1.ConfigMapKeySelector `json:"krbConfigConfigMap,omitempty" protobuf:"bytes,5,opt,name=krbConfigConfigMap"`

	// KrbServicePrincipalName is the principal name of Kerberos service
	// It must be set if either ccache or keytab is used.
	KrbServicePrincipalName string `json:"krbServicePrincipalName,omitempty" protobuf:"bytes,6,opt,name=krbServicePrincipalName"`
}

// RawArtifact allows raw string content to be placed as an artifact in a container
type RawArtifact struct {
	// Data is the string contents of the artifact
	Data string `json:"data" protobuf:"bytes,1,opt,name=data"`
}

func (r *RawArtifact) GetKey() (string, error) {
	return "", keyUnsupportedErr
}

func (r *RawArtifact) SetKey(string) error {
	return keyUnsupportedErr
}

func (r *RawArtifact) HasLocation() bool {
	return r != nil
}

// Header indicate a key-value request header to be used when fetching artifacts over HTTP
type Header struct {
	// Name is the header name
	Name string `json:"name" protobuf:"bytes,1,opt,name=name"`

	// Value is the literal value to use for the header
	Value string `json:"value" protobuf:"bytes,2,opt,name=value"`
}

// HTTPArtifact allows an file served on HTTP to be placed as an input artifact in a container
type HTTPArtifact struct {
	// URL of the artifact
	URL string `json:"url" protobuf:"bytes,1,opt,name=url"`

	// Headers are an optional list of headers to send with HTTP requests for artifacts
	Headers []Header `json:"headers,omitempty" protobuf:"bytes,2,opt,name=headers"`
}

func (h *HTTPArtifact) GetKey() (string, error) {
	u, err := url.Parse(h.URL)
	if err != nil {
		return "", err
	}
	return u.Path, nil
}

func (g *HTTPArtifact) SetKey(key string) error {
	u, err := url.Parse(g.URL)
	if err != nil {
		return err
	}
	u.Path = key
	g.URL = u.String()
	return nil
}

func (h *HTTPArtifact) HasLocation() bool {
	return h != nil && h.URL != ""
}

// GCSBucket contains the access information for interfacring with a GCS bucket
type GCSBucket struct {
	// Bucket is the name of the bucket
	Bucket string `json:"bucket,omitempty" protobuf:"bytes,1,opt,name=bucket"`

	// ServiceAccountKeySecret is the secret selector to the bucket's service account key
	ServiceAccountKeySecret *apiv1.SecretKeySelector `json:"serviceAccountKeySecret,omitempty" protobuf:"bytes,2,opt,name=serviceAccountKeySecret"`
}

// GCSArtifact is the location of a GCS artifact
type GCSArtifact struct {
	GCSBucket `json:",inline" protobuf:"bytes,1,opt,name=gCSBucket"`

	// Key is the path in the bucket where the artifact resides
	Key string `json:"key" protobuf:"bytes,2,opt,name=key"`
}

func (g *GCSArtifact) GetKey() (string, error) {
	return g.Key, nil
}

func (g *GCSArtifact) SetKey(key string) error {
	g.Key = key
	return nil
}

func (g *GCSArtifact) HasLocation() bool {
	return g != nil && g.Bucket != "" && g.Key != ""
}

// OSSBucket contains the access information required for interfacing with an Alibaba Cloud OSS bucket
type OSSBucket struct {
	// Endpoint is the hostname of the bucket endpoint
	Endpoint string `json:"endpoint,omitempty" protobuf:"bytes,1,opt,name=endpoint"`

	// Bucket is the name of the bucket
	Bucket string `json:"bucket,omitempty" protobuf:"bytes,2,opt,name=bucket"`

	// AccessKeySecret is the secret selector to the bucket's access key
	AccessKeySecret *apiv1.SecretKeySelector `json:"accessKeySecret,omitempty" protobuf:"bytes,3,opt,name=accessKeySecret"`

	// SecretKeySecret is the secret selector to the bucket's secret key
	SecretKeySecret *apiv1.SecretKeySelector `json:"secretKeySecret,omitempty" protobuf:"bytes,4,opt,name=secretKeySecret"`

	// CreateBucketIfNotPresent tells the driver to attempt to create the OSS bucket for output artifacts, if it doesn't exist
	CreateBucketIfNotPresent bool `json:"createBucketIfNotPresent,omitempty" protobuf:"varint,5,opt,name=createBucketIfNotPresent"`
}

// OSSArtifact is the location of an Alibaba Cloud OSS artifact
type OSSArtifact struct {
	OSSBucket `json:",inline" protobuf:"bytes,1,opt,name=oSSBucket"`

	// Key is the path in the bucket where the artifact resides
	Key string `json:"key" protobuf:"bytes,2,opt,name=key"`
}

func (o *OSSArtifact) GetKey() (string, error) {
	return o.Key, nil
}

func (o *OSSArtifact) SetKey(key string) error {
	o.Key = key
	return nil
}

func (o *OSSArtifact) HasLocation() bool {
	return o != nil && o.Bucket != "" && o.Endpoint != "" && o.Key != ""
}

func (a *Artifact) GetArchive() *ArchiveStrategy {
	if a == nil || a.Archive == nil {
		return &ArchiveStrategy{}
	}
	return a.Archive
}
