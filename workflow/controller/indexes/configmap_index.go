package indexes

import (
	corev1 "k8s.io/api/core/v1"

	"github.com/argoproj/argo-workflows/v3/workflow/common"
)

func ConfigMapIndexFunc(obj interface{}) ([]string, error) {
	cm, ok := obj.(*corev1.ConfigMap)

	if !ok {
		return nil, nil
	}
	return []string{cm.GetLabels()[common.LabelKeyConfigMapType]}, nil
}
