package config

import (
	"fmt"
	"strings"

	log "github.com/sirupsen/logrus"
	apiv1 "k8s.io/api/core/v1"
	apierr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"sigs.k8s.io/yaml"

	"github.com/argoproj/argo/util/slice"
)

type Controller interface {
	Run(stopCh <-chan struct{})
	Get(namespace, name string, emptyConfigFunc func() interface{}) (interface{}, error)
	RegisterObserver(observer ConfigObserver)
}

type ConfigObserver interface {
	Update(config interface{}) error
	GetConfigKeyFilter() []string
	GetEmptyConfigFunc() func() interface{}
	EnableConfigParse() bool
	GetName() string
}

type controller struct {
	namespace     string
	kubeclientset kubernetes.Interface
	observerList  []ConfigObserver
}

func NewController(namespace string, kubeclientset kubernetes.Interface) Controller {
	log.WithField("namespace", namespace).Info("config map")
	return &controller{
		namespace:     namespace,
		kubeclientset: kubeclientset,
	}
}

func (cc *controller) RegisterObserver(observer ConfigObserver) {
	cc.observerList = append(cc.observerList, observer)
}

func (cc *controller) updateConfig(cm *apiv1.ConfigMap) {
	for _, observer := range cc.observerList {
		log.Infof("updating configmap change to %s Observer", observer.GetName())
		nameFilter := observer.GetConfigKeyFilter()
		if nameFilter != nil && len(nameFilter) > 0 && !slice.ContainsString(nameFilter, fmt.Sprintf("%s/%s", cm.Namespace, cm.Name)) {
			continue
		}
		if observer.EnableConfigParse() {
			config, err := cc.parseConfigMap(cm, observer.GetEmptyConfigFunc())
			if err != nil {
				log.Errorf("parse configmap failed due to: %v", err)
				continue
			}
			err = observer.Update(config)
			if err != nil {
				log.Errorf("update of config failed due to: %v", err)
				continue
			}
		} else {
			err := observer.Update(cm)
			if err != nil {
				log.Errorf("update of config failed due to: %v", err)
				continue
			}
		}
	}
}

func (cc *controller) parseConfigMap(cm *apiv1.ConfigMap, emptyConfigFunc func() interface{}) (interface{}, error) {
	config := emptyConfigFunc()
	if cm == nil {
		return config, nil
	}
	// The key in the configmap to retrieve workflow configuration from.
	// Content encoding is expected to be YAML.
	rawConfig, ok := cm.Data["config"]
	if ok && len(cm.Data) != 1 {
		return config, fmt.Errorf("if you have an item in your config map named 'config', you must only have one item")
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
	err := yaml.Unmarshal([]byte(rawConfig), config)
	return config, err
}

func (cc *controller) Run(stopCh <-chan struct{}) {
	restClient := cc.kubeclientset.CoreV1().RESTClient()
	resource := "configmaps"
	//fieldSelector := fields.ParseSelectorOrDie(fmt.Sprintf("metadata.name=%s", cc.configMap))
	listFunc := func(options metav1.ListOptions) (runtime.Object, error) {
		//options.FieldSelector = fieldSelector.String()
		req := restClient.Get().
			Namespace(cc.namespace).
			Resource(resource).
			VersionedParams(&options, metav1.ParameterCodec)
		return req.Do().Get()
	}
	watchFunc := func(options metav1.ListOptions) (watch.Interface, error) {
		options.Watch = true
		//options.FieldSelector = fieldSelector.String()
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
					cc.updateConfig(newCm)
				}
			},
		})
	controller.Run(stopCh)
	log.Info("Watching config map updates")
}

func (cc *controller) Get(namespace, name string, emptyConfigFunc func() interface{}) (interface{}, error) {
	cmClient := cc.kubeclientset.CoreV1().ConfigMaps(namespace)
	cm, err := cmClient.Get(name, metav1.GetOptions{})
	if err != nil && !apierr.IsNotFound(err) {
		return emptyConfigFunc(), err
	}
	return cc.parseConfigMap(cm, emptyConfigFunc)
}
