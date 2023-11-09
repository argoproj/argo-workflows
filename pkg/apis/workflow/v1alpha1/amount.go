package v1alpha1

import (
	"encoding/json"
	"strconv"
)

// Amount represent a numeric amount.
// +kubebuilder:validation:Type=number
type Amount struct {
	Value json.Number `json:"-" protobuf:"bytes,1,opt,name=value,casttype=encoding/json.Number"`
}

func (a *Amount) UnmarshalJSON(data []byte) error {
	a.Value = json.Number(data)
	return nil
}

func (a Amount) MarshalJSON() ([]byte, error) {
	return []byte(a.Value), nil
}

func (a Amount) OpenAPISchemaType() []string {
	return []string{"number"}
}

func (a Amount) OpenAPISchemaFormat() string {
	return ""
}

func (a *Amount) Float64() (float64, error) {
	return strconv.ParseFloat(string(a.Value), 64)
}
