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
	return []string{cm.GetLabels()[ConfigMapTypeLabel]}, nil
}
