package config

import (
	"context"
	"fmt"
	"strings"

	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/yaml"
)

type Controller interface {
	Get(context.Context) (*Config, error)
	GetNamespace() string
	GetName() string
}

type controller struct {
	namespace string
	// name of the config map
	configMap     string
	kubeclientset kubernetes.Interface
}

func NewController(namespace, name string, kubeclientset kubernetes.Interface) Controller {
	return &controller{
		namespace:     namespace,
		configMap:     name,
		kubeclientset: kubeclientset,
	}
}

func parseConfigMap(cm *apiv1.ConfigMap, config *Config) error {
	// The key in the configmap to retrieve workflow configuration from.
	// Content encoding is expected to be YAML.
	rawConfig, ok := cm.Data["config"]
	if ok && len(cm.Data) != 1 {
		return fmt.Errorf("if you have an item in your config map named 'config', you must only have one item")
	}
	if !ok {
		for name, value := range cm.Data {
			if strings.Contains(value, "\n") {
				// this mucky code indents with two spaces
				rawConfig = rawConfig + name + ":\n  " + strings.Join(strings.Split(strings.Trim(value, "\n"), "\n"), "\n  ") + "\n"
			} else {
				rawConfig = rawConfig + name + ": " + value + "\n"
			}
		}
	}
	err := yaml.UnmarshalStrict([]byte(rawConfig), config)
	return err
}

func (cc *controller) Get(ctx context.Context) (*Config, error) {
	config := &Config{}
	cmClient := cc.kubeclientset.CoreV1().ConfigMaps(cc.namespace)
	cm, err := cmClient.Get(ctx, cc.configMap, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	return config, parseConfigMap(cm, config)
}

func (cc *controller) GetNamespace() string {
	return cc.namespace
}

func (cc *controller) GetName() string {
	return cc.configMap
}
