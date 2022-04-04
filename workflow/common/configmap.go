package common

import (
	"fmt"

	apiv1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/cache"

	"github.com/argoproj/argo-workflows/v3/errors"
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
		if cmType := cm.Labels[LabelKeyConfigMapType]; cmType != LabelValueTypeConfigMapParameter {
			return "", fmt.Errorf(
				"ConfigMap '%s' needs to have the label %s: %s to load parameters",
				name, LabelKeyConfigMapType, LabelValueTypeConfigMapParameter)
		}
		cmValue, ok := cm.Data[key]
		if !ok {
			return "", errors.Errorf(errors.CodeNotFound, "ConfigMap '%s' does not have the key '%s'", name, key)
		}
		return cmValue, nil
	}
	return "", errors.Errorf(errors.CodeNotFound, "ConfigMap '%s' does not exist. Please make sure it has the label %s: %s to be detectable by the controller",
		name, LabelKeyConfigMapType, LabelValueTypeConfigMapParameter)
}
