package common

// Parameters extends string map with useful methods.
type Parameters map[string]string

// Merge merges given parameters.
func (ps Parameters) Merge(args ...Parameters) Parameters {
	newParams := ps.DeepCopy()
	for _, params := range args {
		for k, v := range params {
			newParams[k] = v
		}
	}
	return newParams
}

// DeepCopy returns a new instance which has the same parameters as the receiver.
func (ps Parameters) DeepCopy() Parameters {
	newParams := make(Parameters)
	for k, v := range ps {
		newParams[k] = v
	}
	return newParams
}
