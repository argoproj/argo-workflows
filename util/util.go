package util

import (
	log "github.com/sirupsen/logrus"
	"io/ioutil"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"os"

	"github.com/argoproj/argo/errors"
	"github.com/argoproj/argo/util/retry"
	"k8s.io/client-go/rest"
)

type Closer interface {
	Close() error
}

// Close is a convenience function to close a object that has a Close() method, ignoring any errors
// Used to satisfy errcheck lint
func Close(c Closer) {
	_ = c.Close()
}

// GetSecrets retrieves a secret value and memoizes the result
func GetSecrets(clientSet kubernetes.Interface, namespace, name, key string) ([]byte, error) {

	secretsIf := clientSet.CoreV1().Secrets(namespace)
	var secret *apiv1.Secret
	var err error
	_ = wait.ExponentialBackoff(retry.DefaultRetry, func() (bool, error) {
		secret, err = secretsIf.Get(name, metav1.GetOptions{})
		if err != nil {
			log.Warnf("Failed to get secret '%s': %v", name, err)
			if !retry.IsRetryableKubeAPIError(err) {
				return false, err
			}
			return false, nil
		}
		return true, nil
	})
	if err != nil {
		return []byte{}, errors.InternalWrapError(err)
	}
	val, ok := secret.Data[key]
	if !ok {
		return []byte{}, errors.Errorf(errors.CodeBadRequest, "secret '%s' does not have the key '%s'", name, key)
	}
	return val, nil
}

// Write the Terminate message in pod spec
func WriteTeriminateMessage(message string) {
	err := ioutil.WriteFile("/dev/termination-log", []byte(message), 0644)
	if err != nil {
		panic(err)
	}
}

func GetClientConfig() clientcmd.ClientConfig {
	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	loadingRules.DefaultClientConfig = &clientcmd.DefaultClientConfig
	overrides := clientcmd.ConfigOverrides{}
	//kflags := clientcmd.RecommendedConfigOverrideFlags("")
	return clientcmd.NewInteractiveDeferredLoadingClientConfig(loadingRules, &overrides, os.Stdin)
}

func InitKubeClient() *rest.Config {

	var err error
	var clientConfig clientcmd.ClientConfig

	clientConfig = GetClientConfig()
	config, err := clientConfig.ClientConfig()
	if err != nil {
		panic(err.Error())
	}

	return config
}