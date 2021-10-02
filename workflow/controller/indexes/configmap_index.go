package indexes

import (
	corev1 "k8s.io/api/core/v1"
)

const (
	// LabelKeyConfigMapType is the label key for the type of configmap.
	LabelKeyConfigMapType = "workflows.argoproj.io/configmap-type"
	// LabelValueCacheTypeConfigMap is a key for configmaps that are memoization cache.
	LabelValueCacheTypeConfigMap = "Cache"
	// LabelValueParameterTypeConfigMap is a key for configmaps that contains parameter values.
	LabelValueParameterTypeConfigMap = "Parameter"
)

func ConfigMapIndexFunc(obj interface{}) ([]string, error) {
	cm, ok := obj.(*corev1.ConfigMap)

	if !ok {
		return nil, nil
	}
	return []string{cm.GetLabels()[LabelKeyConfigMapType]}, nil
}
