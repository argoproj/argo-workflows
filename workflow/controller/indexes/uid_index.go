package indexes

import (
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/client-go/tools/cache"
)

var MetaUIDFunc cache.IndexFunc = func(obj interface{}) ([]string, error) {
	v, err := meta.Accessor(obj)
	if err != nil {
		return nil, nil
	}
	return []string{string(v.GetUID())}, nil
}
