package sync

import (
	"time"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
)

type ChainThrottler []Throttler

func (c ChainThrottler) Init(wfs []wfv1.Workflow) error {
	for _, t := range c {
		if err := t.Init(wfs); err != nil {
			return err
		}
	}
	return nil
}

func (c ChainThrottler) Add(key Key, priority int32, creationTime time.Time) {
	for _, t := range c {
		t.Add(key, priority, creationTime)
	}
}

func (c ChainThrottler) Admit(key Key) bool {
	for _, t := range c {
		if !t.Admit(key) {
			return false
		}
	}
	return true
}

func (c ChainThrottler) Remove(key Key) {
	for _, t := range c {
		t.Remove(key)
	}
}

func (c ChainThrottler) RemoveParallelismLimit(key Key) {
	for _, t := range c {
		t.RemoveParallelismLimit(key)
	}
}

var _ Throttler = ChainThrottler{}
