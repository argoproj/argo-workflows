package util

import (
	"fmt"
	apiv1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/cache"
)

// GetConfigMapValue retrieves a configmap value
func GetConfigMapValue(configMapInformer cache.SharedIndexInformer, name, key string) (string, error) {
	obj, exists, err := configMapInformer.GetIndexer().GetByKey(name)
	if err != nil {
		return "", err
	}
	if exists {
		cm, ok := obj.(*apiv1.ConfigMap)
		if !ok {
			return "", fmt.Errorf("unable to convert object %s to configmap when syncing ConfigMaps", name)
		}
		cmValue, ok := cm.Data[name]
		if !ok {
			return "", fmt.Errorf("ConfigMap '%s' does not have the key '%s'", name, key)
		}
		return cmValue, nil
	}
	return "", nil
}
