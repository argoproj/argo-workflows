package config

import (
	"fmt"
	"strings"

	apiv1 "k8s.io/api/core/v1"
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
	return &controller{
		namespace:     namespace,
		configMap:     name,
		kubeclientset: kubeclientset,
	}
}

func (cc *controller) updateConfig(cm *apiv1.ConfigMap, onChange func(config Config) error) error {
	c, err := parseConfigMap(cm)
	if err != nil {
		return err
	}
	log.Infof("workflow controller configuration from %s", cc.configMap)
	return onChange(c)
}

func parseConfigMap(cm *apiv1.ConfigMap) (Config, error) {
	// The key in the configmap to retrieve workflow configuration from.
	// Content encoding is expected to be YAML.
	config, ok := cm.Data["config"]
	var c Config
	// if we don't have /config field, then we iterate over the data and build up a new YAML block to use instead
	if !ok {
		config = ""
		for name, value := range cm.Data {
			if strings.Contains(value, "\n") {
				config = config + name + ":\n" + strings.Trim(value, "\n") + "\n"
			} else {
				config = config + name + ": " + value + "\n"
			}
		}
	}
	return c, yaml.Unmarshal([]byte(config), &c)
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
	if err != nil {
		return Config{}, err
	}
	return parseConfigMap(cm)
}
