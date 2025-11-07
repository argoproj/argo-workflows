package v1alpha1

import (
	"encoding/json"
	"fmt"
)

// Plugin is an Object with exactly one key
type Plugin struct {
	// +kubebuilder:pruning:PreserveUnknownFields
	Object `json:",inline" protobuf:"bytes,1,opt,name=object"`
}

// UnmarshalJSON unmarshalls the Plugin from JSON, and also validates that it is a map exactly one key
func (p *Plugin) UnmarshalJSON(value []byte) error {
	if err := p.Object.UnmarshalJSON(value); err != nil {
		return err
	}
	// by validating the structure in UnmarshallJSON, we prevent bad data entering the system at the point of
	// parsing, which means we do not need validate
	m := map[string]interface{}{}
	if err := json.Unmarshal(p.Value, &m); err != nil {
		return err
	}
	numKeys := len(m)
	if numKeys != 1 {
		return fmt.Errorf("expected exactly one key, got %d", numKeys)
	}
	return nil
}
