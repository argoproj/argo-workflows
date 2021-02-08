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
	return []byte(a.Value), nil
}

func (m PluginTemplate) OpenAPISchemaType() []string { return nil }
func (m PluginTemplate) OpenAPISchemaFormat() string { return "" }

var ErrPluginTemplateNotRequiredType = fmt.Errorf("plugin template not required type") // sentinel error

func (m PluginTemplate) UnmarshalTo(requiredType string, v interface{}) error {
	ty, err := m.GetType()
	if err != nil {
		return err
	}
	if ty != requiredType {
		return ErrPluginTemplateNotRequiredType
	}
	x := make(map[string]interface{})
	err = json.Unmarshal(m.Value, &x)
	if err != nil {
		return fmt.Errorf("failed to unmarshal %q: %w", requiredType, err)
	}
	data, err := json.Marshal(x[requiredType])
	if err != nil {
		return fmt.Errorf("failed to marshal %q: %w", requiredType, err)
	}
	return json.Unmarshal(data, v)
}

func (m PluginTemplate) GetType() (string, error) {
	x := make(map[string]interface{})
	err := json.Unmarshal(m.Value, &x)
	if err != nil {
		return "", nil
	}
	if len(x) > 1 {
		return "", fmt.Errorf("invalid plugin template: ambiguous type")
	}
	for x := range x {
		return x, nil
	}
	return "", fmt.Errorf("invalid plugin template: no type")
}
