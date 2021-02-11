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
	if d.Source == nil {
		return d.PodPolicy == PodPolicyAlways
	}
	return true
}

func (d *Data) GetArtifactIfAny() *Artifact {
	if d.Source != nil && d.Source.WithArtifactPaths != nil {
		return &d.Source.WithArtifactPaths.Artifact
	}
	return nil
}

type Transformation []TransformationStep

type TransformationStep struct {
	// Filter is the strategy in how to filter files
	Filter *Filter `json:"filter,omitempty"`

	// Map is the strategy in how to map files
	Map *MapTransform `json:"map"`

	// Group is the strategy in how to aggregate files
	Group *Group `json:"group,omitempty"`
}

// DataSource sources external data into a data template
type DataSource struct {
	// WithArtifactPaths is a data transformation that collects a list of artifact paths
	WithArtifactPaths *WithArtifactPaths `json:"withArtifactPaths,omitempty"`
}

// WithArtifactPaths expands a step from a collection of artifacts
type WithArtifactPaths struct {
	// Artifact is the artifact location from which to source the artifacts, it can be a directory
	Artifact `json:",inline"`
}

// Filter is the strategy in how to aggregate files
type Filter struct {
	// Regex applies a regex filter to all files in a directory
	Regex string `json:"regex,omitempty"`
}

// Group is the strategy in how to aggregate files
type Group struct {
	// Regex applies a regex and aggregates based on a capture group
	Regex string `json:"regex"`
	// Batch groups into batches of specified size
	Batch int `json:"batch"`
}

// Map is the strategy in how to map files
type MapTransform struct {
	Replace *Replace `json:"replace"`
}

type Replace struct {
	Old string `json:"old"`
	New string `json:"new"`
}
