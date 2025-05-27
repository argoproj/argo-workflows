package util

import (
	"context"

	"k8s.io/client-go/tools/cache"
)

type Indexer struct {
	byIndex map[string][]interface{}
	byKey   map[string]interface{}
}

var _ cache.Indexer = &Indexer{}

func NewIndexer() *Indexer {
	return &Indexer{make(map[string][]interface{}), make(map[string]interface{})}
}

func (i Indexer) Add(interface{}) error                                      { panic("implement me") }
func (i Indexer) Update(interface{}) error                                   { panic("implement me") }
func (i Indexer) Delete(interface{}) error                                   { panic("implement me") }
func (i Indexer) List() []interface{}                                        { panic("implement me") }
func (i Indexer) ListKeys() []string                                         { panic("implement me") }
func (i Indexer) Get(interface{}) (item interface{}, exists bool, err error) { panic("implement me") }
func (i Indexer) GetByKey(key string) (item interface{}, exists bool, err error) {
	obj, ok := i.byKey[key]
	return obj, ok, nil
}
func (i Indexer) SetByKey(key string, item interface{})            { i.byKey[key] = item }
func (i Indexer) Replace([]interface{}, string) error              { panic("implement me") }
func (i Indexer) Resync() error                                    { panic("implement me") }
func (i Indexer) Index(string, interface{}) ([]interface{}, error) { panic("implement me") }
func (i Indexer) IndexKeys(string, string) ([]string, error)       { panic("implement me") }
func (i Indexer) ListIndexFuncValues(string) []string              { panic("implement me") }
func (i Indexer) SetByIndex(indexName, indexedValue string, objs ...interface{}) {
	i.byIndex[indexName+"="+indexedValue] = objs
}

func (i Indexer) ByIndex(indexName, indexedValue string) ([]interface{}, error) {
	return i.byIndex[indexName+"="+indexedValue], nil
}
func (i Indexer) GetIndexers() cache.Indexers      { panic("implement me") }
func (i Indexer) AddIndexers(cache.Indexers) error { panic("implement me") }

func (s SharedIndexInformer) AddEventHandlerWithOptions(handler cache.ResourceEventHandler, options cache.HandlerOptions) (cache.ResourceEventHandlerRegistration, error) {
	panic("implement me")
}

func (s SharedIndexInformer) RunWithContext(ctx context.Context) {
	panic("implement me")
}

func (s SharedIndexInformer) SetWatchErrorHandlerWithContext(handler cache.WatchErrorHandlerWithContext) error {
	panic("implement me")
}
