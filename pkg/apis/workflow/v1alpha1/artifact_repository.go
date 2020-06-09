package v1alpha1

import (
	"fmt"
	"path"

	"github.com/argoproj/argo/errors"
)

const (
	// the default pattern when storing artifacts in an archive repository
	defaultArchivePattern = "{{workflow.name}}/{{pod.name}}"
)

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

func (a *ArtifactRepository) getType() ArtifactType {
	if a == nil {
		return None
	} else if a.Artifactory != nil {
		return Artifactory
	} else if a.GCS != nil {
		return GCS
	} else if a.HDFS != nil {
		return HDFS
	} else if a.OSS != nil {
		return OSS
	} else if a.S3 != nil {
		return S3
	}
	return None
}

// artifact location is defaulted using the following formula:
// <worflow_name>/<pod_name>/<artifact_name>.tgz
// (e.g. myworkflowartifacts/argo-wf-fhljp/argo-wf-fhljp-123291312382/src.tgz)
func (a *ArtifactRepository) AsArtifactLocation() (*ArtifactLocation, error) {
	if a == nil {
		return nil, nil
	}
	l := &ArtifactLocation{ArchiveLogs: a.ArchiveLogs}
	switch a.getType() {
	case Artifactory:
		repoURL := ""
		if a.Artifactory.RepoURL != "" {
			repoURL = a.Artifactory.RepoURL + "/"
		}
		artURL := fmt.Sprintf("%s%s", repoURL, defaultArchivePattern)
		l.Artifactory = &ArtifactoryArtifact{ArtifactoryAuth: a.Artifactory.ArtifactoryAuth, URL: artURL}
	case GCS:
		key := a.GCS.KeyFormat
		if key == "" {
			key = defaultArchivePattern
		}
		l.GCS = &GCSArtifact{GCSBucket: a.GCS.GCSBucket, Key: key}
	case HDFS:
		// TODO - every other branch takes `defaultArtifactPattern` into account - why does this one not do that? looks like a bug to me
		l.HDFS = &HDFSArtifact{HDFSConfig: a.HDFS.HDFSConfig, Path: a.HDFS.PathFormat, Force: a.HDFS.Force}
	case OSS:
		key := a.OSS.KeyFormat
		// NOTE: we use unresolved variables, will get substituted later
		if key == "" {
			key = path.Join(a.OSS.KeyFormat, defaultArchivePattern)
		}
		l.OSS = &OSSArtifact{OSSBucket: a.OSS.OSSBucket, Key: key}
	case S3:
		key := a.S3.KeyFormat
		// NOTE: we use unresolved variables, will get substituted later
		if key == "" {
			key = path.Join(a.S3.KeyPrefix, defaultArchivePattern)
		}
		l.S3 = &S3Artifact{S3Bucket: a.S3.S3Bucket, Key: key}
	default:
		return nil, errors.Errorf(errors.CodeBadRequest, "controller is not configured with a default archive location")
	}
	return l, nil
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
