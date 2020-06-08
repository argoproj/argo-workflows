package v1alpha1

// ArtifactRepository represents a artifact repository in which a controller will store its artifacts
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
}

func (a ArtifactRepository) IsArchiveLogs() bool {
	return a.ArchiveLogs != nil && *a.ArchiveLogs
}

// S3ArtifactRepository defines the controller configuration for an S3 artifact repository
type S3ArtifactRepository struct {
	S3Bucket `json:",inline" protobuf:"bytes,1,opt,name=s3Bucket"`

	// KeyFormat is defines the format of how to store keys. Can reference workflow variables
	KeyFormat string `json:"keyFormat,omitempty" protobuf:"bytes,2,opt,name=keyFormat"`

	// KeyPrefix is prefix used as part of the bucket key in which the controller will store artifacts.
	// DEPRECATED. Use KeyFormat instead
	KeyPrefix string `json:"keyPrefix,omitempty" protobuf:"bytes,3,opt,name=keyPrefix"`
}

// OSSArtifactRepository defines the controller configuration for an OSS artifact repository
type OSSArtifactRepository struct {
	OSSBucket `json:",inline" protobuf:"bytes,1,opt,name=oSSBucket"`

	// KeyFormat is defines the format of how to store keys. Can reference workflow variables
	KeyFormat string `json:"keyFormat,omitempty" protobuf:"bytes,2,opt,name=keyFormat"`
}

// GCSArtifactRepository defines the controller configuration for a GCS artifact repository
type GCSArtifactRepository struct {
	GCSBucket `json:",inline" protobuf:"bytes,1,opt,name=gCSBucket"`

	// KeyFormat is defines the format of how to store keys. Can reference workflow variables
	KeyFormat string `json:"keyFormat,omitempty" protobuf:"bytes,2,opt,name=keyFormat"`
}

// ArtifactoryArtifactRepository defines the controller configuration for an artifactory artifact repository
type ArtifactoryArtifactRepository struct {
	ArtifactoryAuth `json:",inline" protobuf:"bytes,1,opt,name=artifactoryAuth"`
	// RepoURL is the url for artifactory repo.
	RepoURL string `json:"repoURL,omitempty" protobuf:"bytes,2,opt,name=repoURL"`
}

// HDFSArtifactRepository defines the controller configuration for an HDFS artifact repository
type HDFSArtifactRepository struct {
	HDFSConfig `json:",inline" protobuf:"bytes,1,opt,name=hDFSConfig"`

	// PathFormat is defines the format of path to store a file. Can reference workflow variables
	PathFormat string `json:"pathFormat,omitempty" protobuf:"bytes,2,opt,name=pathFormat"`

	// Force copies a file forcibly even if it exists (default: false)
	Force bool `json:"force,omitempty" protobuf:"varint,3,opt,name=force"`
}
