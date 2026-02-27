package v1alpha1

// LabelValues is a list of workflow labels.
type LabelValues struct {
	Items []string `json:"items,omitempty" protobuf:"bytes,1,opt,name=items"`
}

// LabelKeys is list of keys
type LabelKeys struct {
	Items []string `json:"items,omitempty" protobuf:"bytes,1,opt,name=items"`
}
