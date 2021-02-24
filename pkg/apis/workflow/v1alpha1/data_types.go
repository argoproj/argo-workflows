package v1alpha1

// Data is a data template
type Data struct {
	PodPolicy PodPolicy `json:"podPolicy,omitempty" protobuf:"bytes,1,opt,name=podPolicy,casttype=PodPolicy"`

	// Source sources external data into a data template
	Source DataSource `json:"source" protobuf:"bytes,2,opt,name=source"`

	// Transformation applies a set of transformations
	Transformation Transformation `json:"transformation" protobuf:"bytes,3,rep,name=transformation"`
}

type PodPolicy string

const (
	PodPolicyAlways PodPolicy = "Always"
)

func (d *Data) UsePod() bool {
	// If we're not using artifact paths, only use a pod if PodPolicy is set to Always
	if d.Source.ArtifactPaths == nil {
		return d.PodPolicy == PodPolicyAlways
	}
	return true
}

func (d *Data) GetArtifactIfAny() *Artifact {
	return &d.Source.ArtifactPaths.Artifact
}

type Transformation []TransformationStep

type TransformationStep struct {
	// Expression defines an expr expression to apply
	Expression string `json:"expression" protobuf:"bytes,1,opt,name=expression"`
}

// DataSource sources external data into a data template
type DataSource struct {
	// Raw is raw data
	Raw string `json:"raw,omitempty" protobuf:"bytes,1,opt,name=raw"`

	// ArtifactPaths is a data transformation that collects a list of artifact paths
	ArtifactPaths *ArtifactPaths `json:"artifactPaths,omitempty" protobuf:"bytes,2,opt,name=artifactPaths"`
}

// ArtifactPaths expands a step from a collection of artifacts
type ArtifactPaths struct {
	// Artifact is the artifact location from which to source the artifacts, it can be a directory
	Artifact `json:",inline" protobuf:"bytes,1,opt,name=artifact"`
}

type DataSourceProcessor interface {
	ProcessArtifactPaths(*ArtifactPaths) (interface{}, error)
}
