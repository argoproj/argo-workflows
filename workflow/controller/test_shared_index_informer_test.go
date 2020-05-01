package controller

import (
	"time"

	"k8s.io/client-go/tools/cache"
)

type testSharedIndexInformer struct {
}

var _ cache.SharedIndexInformer = &testSharedIndexInformer{}

func (t testSharedIndexInformer) AddEventHandler(cache.ResourceEventHandler) {
	panic("implement me")
}

func (t testSharedIndexInformer) AddEventHandlerWithResyncPeriod(cache.ResourceEventHandler, time.Duration) {
	panic("implement me")
}

func (t testSharedIndexInformer) GetStore() cache.Store {
	return &cache.FakeCustomStore{}
}

func (t testSharedIndexInformer) GetController() cache.Controller {
	panic("implement me")
}

func (t testSharedIndexInformer) Run(<-chan struct{}) {
	panic("implement me")
}

func (t testSharedIndexInformer) HasSynced() bool {
	panic("implement me")
}

func (t testSharedIndexInformer) LastSyncResourceVersion() string {
	panic("implement me")
}

func (t testSharedIndexInformer) AddIndexers(cache.Indexers) error {
	panic("implement me")
}

func (t testSharedIndexInformer) GetIndexer() cache.Indexer {
	panic("implement me")
}
