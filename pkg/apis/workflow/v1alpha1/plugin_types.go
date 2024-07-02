package v1alpha1

import (
	"encoding/json"
	"fmt"
)

// Plugin is an Object with exactly one key
type Plugin struct {
	Object `json:",inline" protobuf:"bytes,1,opt,name=object"`
}

// UnmarshalJSON unmarshalls the Plugin from JSON, and also validates that it is a map exactly one key
func (p *Plugin) UnmarshalJSON(value []byte) error {
	if err := p.Object.UnmarshalJSON(value); err != nil {
		return err
	}
	// by validating the structure in UnmarshallJSON, we prevent bad data entering the system at the point of
	// parsing, which means we do not need validate
	_, err := p.mapValue()
	return err
}

// Name returns the user-specified plugin name
func (p *Plugin) Name() (string, error) {
	if p.Object.Value == nil {
		return "", fmt.Errorf("plugin value is empty")
	}
	mapValue, err := p.mapValue()
	if err != nil {
		return "", err
	}
	for key := range mapValue {
		return key, nil
	}
	return "", nil
}

// mapValue transforms the plugin value to a map of exactly one key, return err if failed
func (p *Plugin) mapValue() (map[string]interface{}, error) {
	mapValue := map[string]interface{}{}
	if err := json.Unmarshal(p.Object.Value, &mapValue); err != nil {
		return nil, err
	}
	numKeys := len(mapValue)
	if numKeys != 1 {
		return nil, fmt.Errorf("expected exactly one key, got %d", numKeys)
	}
	return mapValue, nil
}
