package util

import (
	"k8s.io/client-go/tools/cache"
)

type Indexer struct {
	objs map[string][]interface{}
}

var _ cache.Indexer = &Indexer{}

func (i Indexer) Add(interface{}) error {
	panic("implement me")
}

func (i Indexer) Update(interface{}) error {
	panic("implement me")
}

func (i Indexer) Delete(interface{}) error {
	panic("implement me")
}

func (i Indexer) List() []interface{} {
	panic("implement me")
}

func (i Indexer) ListKeys() []string {
	panic("implement me")
}

func (i Indexer) Get(interface{}) (item interface{}, exists bool, err error) {
	panic("implement me")
}

func (i Indexer) GetByKey(string) (item interface{}, exists bool, err error) {
	panic("implement me")
}

func (i Indexer) Replace([]interface{}, string) error {
	panic("implement me")
}

func (i Indexer) Resync() error {
	panic("implement me")
}

func (i Indexer) Index(string, interface{}) ([]interface{}, error) {
	panic("implement me")
}

func (i Indexer) IndexKeys(string, string) ([]string, error) {
	panic("implement me")
}

func (i Indexer) ListIndexFuncValues(string) []string {
	panic("implement me")
}

func (i Indexer) SetByIndex(indexName, indexedValue string, objs ...interface{}) {
	i.objs[indexName+"="+indexedValue] = objs
}

func (i Indexer) ByIndex(indexName, indexedValue string) ([]interface{}, error) {
	return i.objs[indexName+"="+indexedValue], nil
}

func (i Indexer) GetIndexers() cache.Indexers {
	panic("implement me")
}

func (i Indexer) AddIndexers(cache.Indexers) error {
	panic("implement me")
}
