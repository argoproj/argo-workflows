package v1alpha1

import (
	"fmt"
	"path"
	"strings"
)

const (
	// DefaultArchivePattern is the default pattern when storing artifacts in an archive repository
	DefaultArchivePattern = "{{workflow.name}}/{{pod.name}}"
)

// ArtifactRepository represents an artifact repository in which a controller will store its artifacts
type ArtifactRepository struct {
	// ArchiveLogs enables log archiving
	ArchiveLogs *bool `json:"archiveLogs,omitempty" protobuf:"varint,1,opt,name=archiveLogs"`
	// S3 stores artifact in a S3-compliant object store
	S3 *S3ArtifactRepository `json:"s3,omitempty" protobuf:"bytes,2,opt,name=s3"`
	// Artifactory stores artifacts to JFrog Artifactory
	Artifactory *ArtifactoryArtifactRepository `json:"artifactory,omitempty" protobuf:"bytes,3,opt,name=artifactory"`
	// HDFS stores artifacts in HDFS
	HDFS *HDFSArtifactRepository `json:"hdfs,omitempty" protobuf:"bytes,4,opt,name=hdfs"`
	// OSS stores artifact in a OSS-compliant object store
	OSS *OSSArtifactRepository `json:"oss,omitempty" protobuf:"bytes,5,opt,name=oss"`
	// GCS stores artifact in a GCS object store
	GCS *GCSArtifactRepository `json:"gcs,omitempty" protobuf:"bytes,6,opt,name=gcs"`
	// Azure stores artifact in an Azure Storage account
	Azure *AzureArtifactRepository `json:"azure,omitempty" protobuf:"bytes,7,opt,name=azure"`
	// Plugin stores artifact in a plugin-specific artifact repository
	Plugin *PluginArtifactRepository `json:"plugin,omitempty" protobuf:"bytes,8,opt,name=plugin"`
}

func (a *ArtifactRepository) IsArchiveLogs() bool {
	return a != nil && a.ArchiveLogs != nil && *a.ArchiveLogs
}

type ArtifactRepositoryType interface {
	IntoArtifactLocation(l *ArtifactLocation)
}

func (a *ArtifactRepository) Get() ArtifactRepositoryType {
	if a == nil {
		return nil
	} else if a.Artifactory != nil {
		return a.Artifactory
	} else if a.Azure != nil {
		return a.Azure
	} else if a.GCS != nil {
		return a.GCS
	} else if a.HDFS != nil {
		return a.HDFS
	} else if a.OSS != nil {
		return a.OSS
	} else if a.Plugin != nil {
		return a.Plugin
	} else if a.S3 != nil {
		return a.S3
	}
	return nil
}

// ToArtifactLocation returns the artifact location set with default template key:
// key = `{{workflow.name}}/{{pod.name}}`
func (a *ArtifactRepository) ToArtifactLocation() *ArtifactLocation {
	if a == nil {
		return nil
	}
	l := &ArtifactLocation{ArchiveLogs: a.ArchiveLogs}
	v := a.Get()
	if v != nil {
		v.IntoArtifactLocation(l)
	}
	return l
}

// S3ArtifactRepository defines the controller configuration for an S3 artifact repository
type S3ArtifactRepository struct {
	S3Bucket `json:",inline" protobuf:"bytes,1,opt,name=s3Bucket"`

	// KeyFormat defines the format of how to store keys and can reference workflow variables.
	KeyFormat string `json:"keyFormat,omitempty" protobuf:"bytes,2,opt,name=keyFormat"`

	// KeyPrefix is prefix used as part of the bucket key in which the controller will store artifacts.
	// DEPRECATED. Use KeyFormat instead
	KeyPrefix string `json:"keyPrefix,omitempty" protobuf:"bytes,3,opt,name=keyPrefix"`
}

func (r *S3ArtifactRepository) IntoArtifactLocation(l *ArtifactLocation) {
	k := r.KeyFormat
	if k == "" {
		k = path.Join(r.KeyPrefix, DefaultArchivePattern)
	}
	l.S3 = &S3Artifact{S3Bucket: r.S3Bucket, Key: k}
}

// OSSArtifactRepository defines the controller configuration for an OSS artifact repository
type OSSArtifactRepository struct {
	OSSBucket `json:",inline" protobuf:"bytes,1,opt,name=oSSBucket"`

	// KeyFormat defines the format of how to store keys and can reference workflow variables.
	KeyFormat string `json:"keyFormat,omitempty" protobuf:"bytes,2,opt,name=keyFormat"`
}

func (r *OSSArtifactRepository) IntoArtifactLocation(l *ArtifactLocation) {
	k := r.KeyFormat
	if k == "" {
		k = DefaultArchivePattern
	}
	l.OSS = &OSSArtifact{OSSBucket: r.OSSBucket, Key: k}
}

// GCSArtifactRepository defines the controller configuration for a GCS artifact repository
type GCSArtifactRepository struct {
	GCSBucket `json:",inline" protobuf:"bytes,1,opt,name=gCSBucket"`

	// KeyFormat defines the format of how to store keys and can reference workflow variables.
	KeyFormat string `json:"keyFormat,omitempty" protobuf:"bytes,2,opt,name=keyFormat"`
}

func (r *GCSArtifactRepository) IntoArtifactLocation(l *ArtifactLocation) {
	k := r.KeyFormat
	if k == "" {
		k = DefaultArchivePattern
	}
	l.GCS = &GCSArtifact{GCSBucket: r.GCSBucket, Key: k}
}

// ArtifactoryArtifactRepository defines the controller configuration for an artifactory artifact repository
type ArtifactoryArtifactRepository struct {
	ArtifactoryAuth `json:",inline" protobuf:"bytes,1,opt,name=artifactoryAuth"`
	// RepoURL is the url for artifactory repo.
	RepoURL string `json:"repoURL,omitempty" protobuf:"bytes,2,opt,name=repoURL"`
	// KeyFormat defines the format of how to store keys and can reference workflow variables.
	KeyFormat string `json:"keyFormat,omitempty" protobuf:"bytes,3,opt,name=keyFormat"`
}

func (r *ArtifactoryArtifactRepository) IntoArtifactLocation(l *ArtifactLocation) {
	url := r.RepoURL
	if !strings.HasSuffix(url, "/") {
		url = url + "/"
	}
	k := r.KeyFormat
	if k == "" {
		k = DefaultArchivePattern
	}
	l.Artifactory = &ArtifactoryArtifact{ArtifactoryAuth: r.ArtifactoryAuth, URL: fmt.Sprintf("%s%s", url, k)}
}

// AzureArtifactRepository defines the controller configuration for an Azure Blob Storage artifact repository
type AzureArtifactRepository struct {
	AzureBlobContainer `json:",inline" protobuf:"bytes,1,opt,name=blobContainer"`

	// BlobNameFormat is defines the format of how to store blob names. Can reference workflow variables
	BlobNameFormat string `json:"blobNameFormat,omitempty" protobuf:"bytes,2,opt,name=blobNameFormat"`
}

func (r *AzureArtifactRepository) IntoArtifactLocation(l *ArtifactLocation) {
	k := r.BlobNameFormat
	if k == "" {
		k = DefaultArchivePattern
	}
	l.Azure = &AzureArtifact{AzureBlobContainer: r.AzureBlobContainer, Blob: k}
}

// HDFSArtifactRepository defines the controller configuration for an HDFS artifact repository
type HDFSArtifactRepository struct {
	HDFSConfig `json:",inline" protobuf:"bytes,1,opt,name=hDFSConfig"`

	// PathFormat is defines the format of path to store a file. Can reference workflow variables
	PathFormat string `json:"pathFormat,omitempty" protobuf:"bytes,2,opt,name=pathFormat"`

	// Force copies a file forcibly even if it exists
	Force bool `json:"force,omitempty" protobuf:"varint,3,opt,name=force"`
}

func (r *HDFSArtifactRepository) IntoArtifactLocation(l *ArtifactLocation) {
	p := r.PathFormat
	if p == "" {
		p = DefaultArchivePattern
	}
	l.HDFS = &HDFSArtifact{HDFSConfig: r.HDFSConfig, Path: p, Force: r.Force}
}

// PluginArtifactRepository defines the controller configuration for a plugin artifact repository
type PluginArtifactRepository struct {
	Name          ArtifactPluginName `json:"name" protobuf:"bytes,1,opt,name=name"`
	KeyFormat     string             `json:"keyFormat,omitempty" protobuf:"bytes,2,opt,name=keyFormat"`
	Configuration string             `json:"configuration" protobuf:"bytes,3,opt,name=configuration"`
}

func (r *PluginArtifactRepository) IntoArtifactLocation(l *ArtifactLocation) {
	k := r.KeyFormat
	if k == "" {
		k = DefaultArchivePattern
	}
	l.Plugin = &PluginArtifact{Name: r.Name, Configuration: r.Configuration, Key: k}
}
