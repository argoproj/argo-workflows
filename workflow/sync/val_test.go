package sync

import "github.com/argoproj/argo/workflow/sync/queue"

type val struct {
	key      string
	priority int32
}

func (t *val) GetKey() string {
	return t.key
}

func (t *val) HigherPriorityThan(x interface{}) bool {
	return t.priority > x.(*val).priority
}

var _ queue.Keyed = &val{}
