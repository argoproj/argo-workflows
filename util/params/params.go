package params

// Params extends string map with useful methods.
type Params map[string]string

// Merge merges given parameteres.
func (ps Params) Merge(args ...Params) Params {
	newParams := ps.DeepCopy()
	for _, params := range args {
		for k, v := range params {
			newParams[k] = v
		}
	}
	return newParams
}

// DeepCopy returns a new instance which has the same parameters as the receiver.
func (ps Params) DeepCopy() Params {
	newParams := make(Params)
	for k, v := range ps {
		newParams[k] = v
	}
	return newParams
}
