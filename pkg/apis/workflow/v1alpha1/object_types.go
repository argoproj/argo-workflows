package v1alpha1

import (
	"encoding/json"
)

// Object represents a JSON object value.
// +kubebuilder:validation:Type=object
type Object struct {
	Value json.RawMessage `json:"-" protobuf:"bytes,1,opt,name=value,casttype=encoding/json.RawMessage"`
}

func (i *Object) UnmarshalJSON(value []byte) error {
	return i.Value.UnmarshalJSON(value)
}

func (i Object) MarshalJSON() ([]byte, error) {
	return i.Value.MarshalJSON()
}

func (i Object) OpenAPISchemaType() []string {
	return []string{"object"}
}

func (i Object) OpenAPISchemaFormat() string { return "" }
