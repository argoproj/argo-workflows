package params

type Params map[string]string

func (ps Params) Merge(args ...Params) Params {
	newParams := ps.Clone()
	for _, params := range args {
		for k, v := range params {
			newParams[k] = v
		}
	}
	return newParams
}

func (ps Params) Clone() Params {
	newParams := make(Params)
	for k, v := range ps {
		newParams[k] = v
	}
	return newParams
}
