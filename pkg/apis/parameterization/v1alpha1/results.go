package v1alpha1

// ActionResultDefinition declares a public read model, not an execution engine.
// It is frozen with ActionDefinition and resolved at the bound native call site.
type ActionResultDefinition struct {
	// +kubebuilder:validation:MinLength=1
	Name                     string `json:"name"`
	ActionResultTemplateSpec `json:",inline"`
}

// ActionResultTemplateSpec is a reusable public result contract. Source output
// names are mapped to the selected action's outputs by ActionResultBinding.
type ActionResultTemplateSpec struct {
	Title  string             `json:"title"`
	Source ActionResultSource `json:"source"`
	// Schema validates the normalized document before any public view receives it.
	// +kubebuilder:validation:Type=object
	// +kubebuilder:pruning:PreserveUnknownFields
	Schema JSONValue `json:"schema"`
	// Mappings project JSON Pointers into named top-level result fields.
	// With no mappings the source document is validated without transformation.
	// +kubebuilder:validation:MaxItems=64
	Mappings []ResultFieldMapping `json:"mappings,omitempty"`
	// +kubebuilder:validation:MinItems=1
	// +kubebuilder:validation:MaxItems=16
	Views []ResultView `json:"views"`
}

// ActionResultBinding replaces the action's default results at one call site.
// References resolve in the Workflow namespace and freeze at preview/submit.
type ActionResultBinding struct {
	Name           string `json:"name"`
	ResultTemplate string `json:"resultTemplate"`
	// OutputMappings maps actual action outputs (from) to source slots (to).
	// Unmapped slots retain their names; unknown or duplicate slots are rejected.
	// +kubebuilder:validation:MaxItems=3
	OutputMappings []ActionParameterMapping `json:"outputMappings,omitempty"`
	// InputMappings binds preallocated result identity slots to native inputs.
	// +kubebuilder:validation:MaxItems=2
	InputMappings []ActionParameterMapping `json:"inputMappings,omitempty"`
}

// ActionResultSource selects a frozen native output, or an exact object produced
// by that invocation. Refresh only reads: the producing controller owns polling.
// +kubebuilder:validation:XValidation:rule="has(self.parameter) != has(self.resource)",message="exactly one result source is required"
type ActionResultSource struct {
	Parameter string                `json:"parameter,omitempty"`
	Resource  *ResultResourceSource `json:"resource,omitempty"`
	// +kubebuilder:validation:Enum=json;text
	Format string `json:"format"`
}

type ResultResourceSource struct {
	Group    string `json:"group,omitempty"`
	Version  string `json:"version"`
	Resource string `json:"resource"`
	// The resource is always in the Workflow namespace. Both identity fields
	// come from exactly one native output pair or immutable invocation input
	// pair. Input identities permit observing a running or failed invocation.
	// Both modes still require the exact UID and Workflow owner reference.
	NameOutput     string `json:"nameOutput,omitempty"`
	UIDOutput      string `json:"uidOutput,omitempty"`
	NameInput      string `json:"nameInput,omitempty"`
	UIDInput       string `json:"uidInput,omitempty"`
	DocumentPath   string `json:"documentPath"`
	ObservedAtPath string `json:"observedAtPath,omitempty"`
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=86400
	MaxAgeSeconds int32 `json:"maxAgeSeconds,omitempty"`
}

type ResultFieldMapping struct {
	From string `json:"from"`
	To   string `json:"to"`
}

type ResultView struct {
	Title string `json:"title"`
	// +kubebuilder:validation:Enum=summary;table;tree;text
	Type string `json:"type"`
	Path string `json:"path,omitempty"`
	// +kubebuilder:validation:MaxItems=32
	Columns      []ResultColumn `json:"columns,omitempty"`
	ChildrenPath string         `json:"childrenPath,omitempty"`
}

type ResultColumn struct {
	Title string `json:"title"`
	Path  string `json:"path"`
	// +kubebuilder:validation:Enum=text;link;status
	Format string `json:"format,omitempty"`
}
