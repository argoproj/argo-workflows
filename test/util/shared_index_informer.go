package util

import (
	"time"

	"k8s.io/client-go/tools/cache"
)

type SharedIndexInformer struct {
	Indexer *Indexer
}

var _ cache.SharedIndexInformer = &SharedIndexInformer{}

func NewSharedIndexInformer() *SharedIndexInformer {
	return &SharedIndexInformer{&Indexer{objs: make(map[string][]interface{})}}
}

func (s *SharedIndexInformer) AddEventHandler(cache.ResourceEventHandler) {}
func (s *SharedIndexInformer) AddEventHandlerWithResyncPeriod(cache.ResourceEventHandler, time.Duration) {
}
func (s *SharedIndexInformer) GetStore() cache.Store            { panic("implement me") }
func (s *SharedIndexInformer) GetController() cache.Controller  { panic("implement me") }
func (s *SharedIndexInformer) Run(<-chan struct{})              {}
func (s *SharedIndexInformer) HasSynced() bool                  { return true }
func (s *SharedIndexInformer) LastSyncResourceVersion() string  { return "" }
func (s *SharedIndexInformer) AddIndexers(cache.Indexers) error { return nil }
func (s *SharedIndexInformer) GetIndexer() cache.Indexer        { return s.Indexer }
