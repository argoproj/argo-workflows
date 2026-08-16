package util

import (
	"k8s.io/client-go/tools/cache"
)

type Indexer struct {
	byIndex map[string][]any
	byKey   map[string]any
}

var _ cache.Indexer = &Indexer{}

func NewIndexer() *Indexer {
	return &Indexer{make(map[string][]any), make(map[string]any)}
}

func (i Indexer) Add(any) error                              { panic("implement me") }
func (i Indexer) Update(any) error                           { panic("implement me") }
func (i Indexer) Delete(any) error                           { panic("implement me") }
func (i Indexer) List() []any                                { panic("implement me") }
func (i Indexer) ListKeys() []string                         { panic("implement me") }
func (i Indexer) Get(any) (item any, exists bool, err error) { panic("implement me") }
func (i Indexer) GetByKey(key string) (item any, exists bool, err error) {
	obj, ok := i.byKey[key]
	return obj, ok, nil
}
func (i Indexer) SetByKey(key string, item any)              { i.byKey[key] = item }
func (i Indexer) Replace([]any, string) error                { panic("implement me") }
func (i Indexer) Resync() error                              { panic("implement me") }
func (i Indexer) Index(string, any) ([]any, error)           { panic("implement me") }
func (i Indexer) IndexKeys(string, string) ([]string, error) { panic("implement me") }
func (i Indexer) ListIndexFuncValues(string) []string        { panic("implement me") }
func (i Indexer) SetByIndex(indexName, indexedValue string, objs ...any) {
	i.byIndex[indexName+"="+indexedValue] = objs
}

func (i Indexer) ByIndex(indexName, indexedValue string) ([]any, error) {
	return i.byIndex[indexName+"="+indexedValue], nil
}
func (i Indexer) GetIndexers() cache.Indexers      { panic("implement me") }
func (i Indexer) AddIndexers(cache.Indexers) error { panic("implement me") }
