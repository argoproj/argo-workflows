package v1alpha1

type ArtifactType int

const (
	None ArtifactType = iota
	Artifactory
	Git
	GCS
	HDFS
	HTTP
	OSS
	Raw
	S3
)
