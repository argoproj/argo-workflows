package indexes

import (
	"github.com/argoproj/argo-workflows/v3/workflow/common"
	corev1 "k8s.io/api/core/v1"
)

func ConfigMapIndexFunc(obj interface{}) ([]string, error) {
	cm, ok := obj.(*corev1.ConfigMap)
	if !ok {
		return nil, nil
	}
	value, ok := cm.GetLabels()[common.LabelKeyConfigMapType]
	if !ok {
		return nil, nil
	}
	return []string{ConfigMapIndexValue(cm.GetNamespace(), value)}, nil
}

func ConfigMapIndexValue(namespace string, configMapType string) string {
	return namespace + "/" + configMapType
}
