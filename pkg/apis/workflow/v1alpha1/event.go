package v1alpha1

// Event can trigger this workflow template.
type Event struct {
	// Expression (https://github.com/antonmedv/expr) that we must must match the event. E.g. `payload.message == "test"`
	// +kubebuilder:validation:MinLength=4
	Expression string `json:"expression" protobuf:"bytes,1,opt,name=expression"`

	// Parameters extracted from the event and then set as arguments to the workflow created.
	Parameters []EventParameter `json:"parameters,omitempty" protobuf:"bytes,2,rep,name=parameters"`
}

// EventParameter is a parameter extracted from the event.
// +patchStrategy=merge
// +patchMergeKey=name
type EventParameter struct {
	// Name of the parameter
	Name string `json:"name" protobuf:"bytes,1,opt,name=name"`

	// Expression (https://github.com/antonmedv/expr) that is evaluated against the event to get the value of the parameter. E.g. `payload.message`
	// +kubebuilder:validation:MinLength=4
	Expression string `json:"expression" protobuf:"bytes,2,opt,name=expression"`
}
