package v1alpha1

// Event can trigger this workflow template.
type Event struct {
	// Expression (https://github.com/antonmedv/expr) that we must must match the event. E.g. `event.type == "test"`
	// +kubebuilder:validation:MinLength=4
	Expression string `json:"expression" protobuf:"bytes,1,opt,name=expression"`

	// Parameters to set as arguments to workflows created.
	Parameters []EventParameter `json:"parameters,omitempty" protobuf:"bytes,2,rep,name=parameters"`
}

// EventParameter is a parameter extracted from the event.
// +patchStrategy=merge
// +patchMergeKey=name
type EventParameter struct {
	// Name of the parameters
	Name string `json:"name" protobuf:"bytes,1,opt,name=name"`

	// Expression (https://github.com/antonmedv/expr) that is evaluted against the event. E.g. `event.type`
	Expression string `json:"expression" protobuf:"bytes,2,opt,name=expression"`
}
