package v1alpha1

// Data is a data template
type Data struct {
	// Source sources external data into a data template
	Source *DataSource `json:"source,omitempty"`

	// Transformation applies a set of transformations
	Transformation Transformation `json:"transformation"`
}

func (d *Data) NeedsPod() bool {
	return d.Source != nil
}

func (d *Data) GetArtifactIfAny() *Artifact {
	if d.Source != nil && d.Source.WithArtifactPaths != nil {
		return &d.Source.WithArtifactPaths.Artifact
	}
	return nil
}

type Transformation []DataStep

type DataStep struct {
	// Name is the name of the data step
	Name string `json:"name,omitempty"`

	// Filter is the strategy in how to filter files
	Filter *Filter `json:"filter,omitempty"`

	// Aggregator is the strategy in how to aggregate files
	Aggregator *Aggregator `json:"aggregator,omitempty"`
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

// Aggregator is the strategy in how to aggregate files
type Aggregator struct {
	// Regex applies a regex and aggregates based on a capture group
	Regex string `json:"regex"`
	// Batch groups into batches of specified size
	Batch int `json:"batch"`
}
