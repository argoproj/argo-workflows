package controller

import (
	"fmt"
)

type profiles map[profileKey]*profile

func (ps profiles) find(cluster string) (*profile, error) {
	if p, ok := ps[cluster]; ok {
		return p, nil
	}
	return nil, fmt.Errorf("profile not found for cluster %q", cluster)
}

func (ps profiles) run(done <-chan struct{}) {
	for _, p := range ps {
		p.run(done)
	}
}

func (ps profiles) hasSynced() bool {
	for _, p := range ps {
		if !p.hasSynced() {
			return false
		}
	}
	return true
}
