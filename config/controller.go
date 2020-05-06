package config

import (
	"fmt"
	"strings"

	apiv1 "k8s.io/api/core/v1"
	apierr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"

	log "github.com/sirupsen/logrus"
	"sigs.k8s.io/yaml"
)

type Controller interface {
	Run(stopCh <-chan struct{}, onChange func(config Config) error)
	Get() (Config, error)
}

type controller struct {
	namespace string
	// name of the config map
	configMap     string
	kubeclientset kubernetes.Interface
}

func NewController(namespace, name string, kubeclientset kubernetes.Interface) Controller {
	log.WithField("name", name).Info("config map")
	return &controller{
		namespace:     namespace,
		configMap:     name,
		kubeclientset: kubeclientset,
	}
}

func (cc *controller) updateConfig(cm *apiv1.ConfigMap, onChange func(config Config) error) error {
	config, err := parseConfigMap(cm)
	if err != nil {
		return err
	}
	return onChange(config)
}

func parseConfigMap(cm *apiv1.ConfigMap) (Config, error) {
	if cm == nil {
		return emptyConfig, nil
	}
	// The key in the configmap to retrieve workflow configuration from.
	// Content encoding is expected to be YAML.
	var config Config
	rawConfig, ok := cm.Data["config"]
	if ok && len(cm.Data) != 1 {
		return emptyConfig, fmt.Errorf("if you have an item in your config map named 'config', you must only have one item")
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
	err := yaml.Unmarshal([]byte(rawConfig), &config)
	if err != nil {
		return emptyConfig, err
	}
	return config, nil
}

func (cc *controller) Run(stopCh <-chan struct{}, onChange func(config Config) error) {
	restClient := cc.kubeclientset.CoreV1().RESTClient()
	resource := "configmaps"
	fieldSelector := fields.ParseSelectorOrDie(fmt.Sprintf("metadata.name=%s", cc.configMap))
	listFunc := func(options metav1.ListOptions) (runtime.Object, error) {
		options.FieldSelector = fieldSelector.String()
		req := restClient.Get().
			Namespace(cc.namespace).
			Resource(resource).
			VersionedParams(&options, metav1.ParameterCodec)
		return req.Do().Get()
	}
	watchFunc := func(options metav1.ListOptions) (watch.Interface, error) {
		options.Watch = true
		options.FieldSelector = fieldSelector.String()
		req := restClient.Get().
			Namespace(cc.namespace).
			Resource(resource).
			VersionedParams(&options, metav1.ParameterCodec)
		return req.Watch()
	}
	source := &cache.ListWatch{ListFunc: listFunc, WatchFunc: watchFunc}
	_, controller := cache.NewInformer(
		source,
		&apiv1.ConfigMap{},
		0,
		cache.ResourceEventHandlerFuncs{
			UpdateFunc: func(old, new interface{}) {
				oldCM := old.(*apiv1.ConfigMap)
				newCM := new.(*apiv1.ConfigMap)
				if oldCM.ResourceVersion == newCM.ResourceVersion {
					return
				}
				if newCm, ok := new.(*apiv1.ConfigMap); ok {
					log.Infof("Detected ConfigMap update.")
					err := cc.updateConfig(newCm, onChange)
					if err != nil {
						log.Errorf("Update of config failed due to: %v", err)
					}
				}
			},
		})
	controller.Run(stopCh)
	log.Info("Watching config map updates")
}

func (cc *controller) Get() (Config, error) {
	cmClient := cc.kubeclientset.CoreV1().ConfigMaps(cc.namespace)
	cm, err := cmClient.Get(cc.configMap, metav1.GetOptions{})
	if err != nil && !apierr.IsNotFound(err) {
		return emptyConfig, err
	}
	return parseConfigMap(cm)
}
