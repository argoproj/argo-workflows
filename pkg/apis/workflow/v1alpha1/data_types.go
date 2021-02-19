package v1alpha1

// Data is a data template
type Data struct {
	PodPolicy PodPolicy `json:"podPolicy,omitempty"`

	// Source sources external data into a data template
	Source *DataSource `json:"source,omitempty"`

	// Transformation applies a set of transformations
	Transformation *Transformation `json:"transformation"`
}

type PodPolicy string

const (
	PodPolicyAlways PodPolicy = "Always"
)

func (d *Data) UsePod() bool {
	// If we're not using artifact paths, only use a pod if PodPolicy is set to Always
	if d.Source == nil || d.Source.ArtifactPaths == nil {
		return d.PodPolicy == PodPolicyAlways
	}
	return true
}

func (d *Data) GetArtifactIfAny() *Artifact {
	if d.Source != nil && d.Source.ArtifactPaths != nil {
		return &d.Source.ArtifactPaths.Artifact
	}
	return nil
}

type Transformation []TransformationStep

type TransformationStep struct {
	// Expression defines an expr expression to apply
	Expression string `json:"expression"`
}

// DataSource sources external data into a data template
type DataSource struct {
	// Raw is raw data
	Raw string `json:"raw,omitempty"`

	// ArtifactPaths is a data transformation that collects a list of artifact paths
	ArtifactPaths *ArtifactPaths `json:"artifactPaths,omitempty"`
}

// ArtifactPaths expands a step from a collection of artifacts
type ArtifactPaths struct {
	// Artifact is the artifact location from which to source the artifacts, it can be a directory
	Artifact `json:",inline"`
}

type SourceProcessor interface {
	ProcessArtifactPaths(*ArtifactPaths) (interface{}, error)
}
