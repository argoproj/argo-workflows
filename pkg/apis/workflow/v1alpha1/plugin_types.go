package v1alpha1

import (
	"encoding/json"
	"fmt"
)

type PluginTemplate struct {
	Value json.RawMessage `json:"-" protobuf:"bytes,1,opt,name=value,casttype=encoding/json.RawMessage"`
}

func (a *PluginTemplate) UnmarshalJSON(data []byte) error {
	a.Value = data
	return nil
}

func (a PluginTemplate) MarshalJSON() ([]byte, error) {
	return a.Value, nil
}

func (m PluginTemplate) OpenAPISchemaType() []string { return nil }
func (m PluginTemplate) OpenAPISchemaFormat() string { return "" }

func (m PluginTemplate) UnmarshalTo(v interface{}) error {
	_, x, err := m.Get()
	if err != nil {
		return err
	}
	data, err := json.Marshal(x)
	if err != nil {
		return fmt.Errorf("failed to marshal: %w", err)
	}
	return json.Unmarshal(data, v)
}

func (m PluginTemplate) Get() (string, interface{}, error) {
	x := make(map[string]interface{})
	err := json.Unmarshal(m.Value, &x)
	if err != nil {
		return "", nil, nil
	}
	if len(x) > 1 {
		return "", nil, fmt.Errorf("invalid plugin template: ambiguous type")
	}
	for x, y := range x {
		return x, y, nil
	}
	return "", nil, fmt.Errorf("invalid plugin template: no type")
}
