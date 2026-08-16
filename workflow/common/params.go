package common

import "maps"

// Parameters extends string map with useful methods.
type Parameters map[string]string

// Merge merges given parameters.
func (ps Parameters) Merge(args ...Parameters) Parameters {
	newParams := ps.DeepCopy()
	for _, params := range args {
		maps.Copy(newParams, params)
	}
	return newParams
}

// DeepCopy returns a new instance which has the same parameters as the receiver.
func (ps Parameters) DeepCopy() Parameters {
	newParams := make(Parameters)
	maps.Copy(newParams, ps)
	return newParams
}
