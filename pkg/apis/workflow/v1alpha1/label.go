package v1alpha1

// Labels is list of workflow labels
type LabelValues struct {
	Items []string `json:"items,omitempty" protobuf:"bytes,1,rep,name=items"`
}

// LabelKeys is list of keys
type LabelKeys struct {
	Items []string `json:"items,omitempty" protobuf:"bytes,1,rep,name=items"`
}
