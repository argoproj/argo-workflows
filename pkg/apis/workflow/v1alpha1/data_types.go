package v1alpha1

// Data is a data template
type Data struct {
	// Source sources external data into a data template
	Source DataSource `json:"source" protobuf:"bytes,1,opt,name=source"`

	// Transformation applies a set of transformations
	Transformation Transformation `json:"transformation" protobuf:"bytes,2,rep,name=transformation"`
}

func (ds *DataSource) GetArtifactIfNeeded() (*Artifact, bool) {
	if ds.ArtifactPaths != nil {
		return &ds.ArtifactPaths.Artifact, true
	}
	return nil, false
}

type Transformation []TransformationStep

type TransformationStep struct {
	// Expression defines an expr expression to apply
	Expression string `json:"expression" protobuf:"bytes,1,opt,name=expression"`
}

// DataSource sources external data into a data template
type DataSource struct {
	// ArtifactPaths is a data transformation that collects a list of artifact paths
	ArtifactPaths *ArtifactPaths `json:"artifactPaths,omitempty" protobuf:"bytes,1,opt,name=artifactPaths"`
}

// ArtifactPaths expands a step from a collection of artifacts
type ArtifactPaths struct {
	// Artifact is the artifact location from which to source the artifacts, it can be a directory
	Artifact `json:",inline" protobuf:"bytes,1,opt,name=artifact"`
}

type DataSourceProcessor interface {
	ProcessArtifactPaths(*ArtifactPaths) (interface{}, error)
}
