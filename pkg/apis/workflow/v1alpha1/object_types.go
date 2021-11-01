package v1alpha1

import (
	"encoding/json"
)

// +kubebuilder:validation:Type=object
type Object struct {
	Value json.RawMessage `json:"-"`
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

func (i *Object) AsMap() (map[string]interface{}, error) {
	if i == nil {
		return nil, nil
	}
	resp := map[string]interface{}{}
	return resp, json.Unmarshal(i.Value, &resp)
}

func (i *Object) Get(s string) (interface{}, error) {
	m, err := i.AsMap()
	if err != nil {
		return nil, err
	}
	if v, ok := m[s]; ok {
		return v, nil
	}
	return nil, nil
}
