package sync

import (
	"time"
)

type ChainThrottler []Throttler

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

var _ Throttler = ChainThrottler{}
