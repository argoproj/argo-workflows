package v1alpha1

// Event can trigger this workflow template.
type Event struct {
	// Expression (https://github.com/antonmedv/expr) that we must must match the event. E.g. `payload.message == "test"`
	// +kubebuilder:validation:MinLength=4
	Expression string `json:"expression" protobuf:"bytes,1,opt,name=expression"`

	// Parameters extracted from the event and then set as arguments to the workflow created.
	Parameters []Parameter `json:"parameters,omitempty" protobuf:"bytes,2,rep,name=parameters"`
}

type Events []Event
