package util

import (
	"fmt"

	apiv1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/cache"

	"github.com/argoproj/argo-workflows/v3/workflow/common"
)

// GetConfigMapValue retrieves a configmap value
func GetConfigMapValue(configMapInformer cache.SharedIndexInformer, namespace, name, key string) (string, error) {
	obj, exists, err := configMapInformer.GetIndexer().GetByKey(namespace + "/" + name)
	if err != nil {
		return "", err
	}
	if exists {
		cm, ok := obj.(*apiv1.ConfigMap)
		if !ok {
			return "", fmt.Errorf("unable to convert object %s to configmap when syncing ConfigMaps", name)
		}
		if cmType := cm.Labels[common.LabelKeyConfigMapType]; cmType != common.LabelValueTypeConfigMapParameter {
			return "", fmt.Errorf(
				"ConfigMap '%s' needs to have the label %s: %s for parameters loading",
				name, common.LabelKeyConfigMapType, common.LabelValueTypeConfigMapParameter)
		}
		cmValue, ok := cm.Data[key]
		if !ok {
			return "", fmt.Errorf("ConfigMap '%s' does not have the key '%s'", name, key)
		}
		return cmValue, nil
	}
	return "", fmt.Errorf("ConfigMap '%s' does not exist", name)
}
