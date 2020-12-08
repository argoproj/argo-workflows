package util

import (
	"fmt"

	log "github.com/sirupsen/logrus"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"

	errorsutil "github.com/argoproj/argo/util/errors"
	"github.com/argoproj/argo/util/retry"
)

// GetConfigMaps retrieves a configmap value and memoizes the result
func GetConfigMaps(clientSet kubernetes.Interface, namespace, name, key string) (string, error) {

	configMapsIf := clientSet.CoreV1().ConfigMaps(namespace)
	var configMap *apiv1.ConfigMap
	var err error

	err = wait.ExponentialBackoff(retry.DefaultRetry, func() (bool, error) {
		configMap, err = configMapsIf.Get(name, metav1.GetOptions{})
		if err != nil {
			log.Warnf("Failed to get config map '%s': %v", name, err)
			if !errorsutil.IsTransientErr(err) {
				return false, err
			}
			return false, nil
		}
		return true, nil
	})

	if err != nil {
		return "", fmt.Errorf("Config map not found: %w", err)
	}

	val, ok := configMap.Data[key]
	if !ok {
		return "", fmt.Errorf("Config map '%s' does not have the key '%s'", name, key)
	}
	return val, nil
}
