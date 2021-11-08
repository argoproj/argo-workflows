package indexes

import (
	corev1 "k8s.io/api/core/v1"
)

const ConfigMapTypeLabel = "workflows.argoproj.io/configmap-type"

func ConfigMapIndexFunc(obj interface{}) ([]string, error) {
	cm, ok := obj.(*corev1.ConfigMap)
	if !ok {
		return nil, nil
	}
	value, ok := cm.GetLabels()[ConfigMapTypeLabel]
	if !ok {
		return nil, nil
	}
	return []string{ConfigMapIndexValue(cm.GetNamespace(), value)}, nil
}

func ConfigMapIndexValue(namespace string, configMapType string) string {
	return namespace + "/" + configMapType
}
