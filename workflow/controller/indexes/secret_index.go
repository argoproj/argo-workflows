package indexes

import (
	corev1 "k8s.io/api/core/v1"

	"github.com/argoproj/argo-workflows/v3/workflow/common"
)

func SecretIndexFunc(obj interface{}) ([]string, error) {
	secret, ok := obj.(*corev1.Secret)

	if !ok {
		return nil, nil
	}
	v, ok := secret.GetLabels()[common.LabelKeySecretType]
	if !ok {
		return nil, nil
	}
	return []string{v}, nil
}
