package types

import (
	"fmt"

	"github.com/argoproj/argo-workflows/v3/workflow/common"
)

type Profiles map[ProfileKey]*Profile

func (ps Profiles) Primary() *Profile {
	return ps[common.PrimaryCluster()]
}

func (ps Profiles) Find(cluster string) (*Profile, error) {
	if f, ok := ps[cluster]; ok {
		return f, nil
	}
	return nil, fmt.Errorf("profile not found for cluster %q", cluster)
}

func (ps Profiles) Keys() []string {
	var keys []string
	for key := range ps {
		keys = append(keys, key)
	}
	return keys
}
