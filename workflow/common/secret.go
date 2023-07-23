package common

import (
	"fmt"

	apiv1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/cache"

	"github.com/argoproj/argo-workflows/v3/errors"
)

// GetSecretValue retrieves a secret value
func GetSecretValue(secretInformer cache.SharedIndexInformer, namespace, name, key string) (string, error) {
	obj, exists, err := secretInformer.GetIndexer().GetByKey(namespace + "/" + name)
	if err != nil {
		return "", err
	}
	if exists {
		secret, ok := obj.(*apiv1.Secret)

		if !ok {
			return "", fmt.Errorf("unable to convert object %s to configmap when syncing Secrets", name)
		}
		if secretType := secret.Labels[LabelKeySecretType]; secretType != LabelValueTypeSecretParameter {
			return "", fmt.Errorf(
				"Secret '%s' needs to have the label %s: %s to load parameters",
				name, LabelKeySecretType, LabelValueTypeSecretParameter)
		}
		secretValueBytes, ok := secret.Data[key]
		secretValue := string(secretValueBytes)

		if !ok {
			return "", errors.Errorf(errors.CodeNotFound, "Secret '%s' does not have the key '%s'", name, key)
		}
		return secretValue, nil
	}
	return "", errors.Errorf(errors.CodeNotFound, "Secret '%s' does not exist. Please make sure it has the label %s: %s to be detectable by the controller",
		name, LabelKeySecretType, LabelValueTypeSecretParameter)
}
