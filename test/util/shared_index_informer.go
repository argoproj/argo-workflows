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
	return &SharedIndexInformer{Indexer: NewIndexer()}
}

func (s *SharedIndexInformer) AddEventHandler(cache.ResourceEventHandler) {}
func (s *SharedIndexInformer) AddEventHandlerWithResyncPeriod(cache.ResourceEventHandler, time.Duration) {
}
func (s *SharedIndexInformer) GetStore() cache.Store                                      { return s.Indexer }
func (s *SharedIndexInformer) GetController() cache.Controller                            { panic("implement me") }
func (s *SharedIndexInformer) Run(<-chan struct{})                                        {}
func (s *SharedIndexInformer) HasSynced() bool                                            { return true }
func (s *SharedIndexInformer) LastSyncResourceVersion() string                            { return "" }
func (s *SharedIndexInformer) AddIndexers(cache.Indexers) error                           { return nil }
func (s *SharedIndexInformer) GetIndexer() cache.Indexer                                  { return s.Indexer }
func (s *SharedIndexInformer) SetWatchErrorHandler(handler cache.WatchErrorHandler) error { return nil }
func (s *SharedIndexInformer) SetTransform(handler cache.TransformFunc) error             { return nil }
