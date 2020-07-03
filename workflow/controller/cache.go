package controller

import (
	"encoding/json"
	"regexp"
	"sync"

	apierr "k8s.io/apimachinery/pkg/api/errors"

	"github.com/argoproj/argo/errors"

	log "github.com/sirupsen/logrus"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
)

var keyRegex = regexp.MustCompile("^[a-zA-Z0-9][-a-zA-Z0-9]*$")

type MemoizationCache interface {
	Load(key string, configMapName string) (*wfv1.Outputs, error)
	Save(key string, nodeId string, value *wfv1.Outputs, configMapName string) error
}

type CacheEntry struct {
	NodeID  string       `json:"nodeID"`
	Outputs wfv1.Outputs `json:"outputs"`
}

type configMapCache struct {
	namespace  string
	kubeClient kubernetes.Interface
	locked     *sync.Mutex
}

func NewConfigMapCache(ns string, ki kubernetes.Interface) MemoizationCache {
	return &configMapCache{
		namespace:  ns,
		kubeClient: ki,
		locked:     &sync.Mutex{},
	}
}

func (c *configMapCache) Load(key string, configMapName string) (*wfv1.Outputs, error) {
	if !keyRegex.MatchString(key) {
		log.Errorf("Invalid cache key %s", key)
		return nil, errors.InternalError("Invalid cache key")
	}
	c.locked.Lock()
	defer c.locked.Unlock()
	cm, err := c.kubeClient.CoreV1().ConfigMaps(c.namespace).Get(configMapName, metav1.GetOptions{})
	if apierr.IsNotFound(err) {
		log.Infof("MemoizationCache miss: ConfigMap does not exist")
		return nil, nil
	}
	if err != nil {
		log.Infof("Error loading ConfigMap cache %s in namespace %s: %s", configMapName, c.namespace, err)
		return nil, err
	}
	log.Infof("ConfigMap cache %s loaded", configMapName)
	rawEntry, ok := cm.Data[key]
	if !ok || rawEntry == "" {
		log.Debugf("MemoizationCache miss: Entry for %s doesn't exist", key)
		return nil, nil
	}
	var entry CacheEntry
	err = json.Unmarshal([]byte(rawEntry), &entry)
	if err != nil {
		return nil, err
	}
	outputs := entry.Outputs
	log.Infof("ConfigMap cache %s hit for %s", configMapName, key)
	return &outputs, nil
}

func (c *configMapCache) Save(key string, nodeId string, value *wfv1.Outputs, configMapName string) error {
	if !keyRegex.MatchString(key) {
		log.Errorf("Invalid cache key %s", key)
		return errors.InternalError("Invalid cache key")
	}
	c.locked.Lock()
	defer c.locked.Unlock()
	log.Infof("Saving to cache %s...", key)
	cache, err := c.kubeClient.CoreV1().ConfigMaps(c.namespace).Get(configMapName, metav1.GetOptions{})
	if apierr.IsNotFound(err) || cache == nil {
		cache, err = c.kubeClient.CoreV1().ConfigMaps(c.namespace).Create(&apiv1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name: configMapName,
			},
		})
		if err != nil {
			log.Warnf("Error saving to cache: %s", err)
			return err
		}
	}

	newEntry := CacheEntry{
		NodeID:  nodeId,
		Outputs: *value,
	}

	entryJSON, err := json.Marshal(newEntry)
	if err != nil {
		return err
	}
	if cache.Data != nil {
		cache.Data[key] = string(entryJSON)
	} else {
		cache.Data = map[string]string{
			key: string(entryJSON),
		}
	}
	_, err = c.kubeClient.CoreV1().ConfigMaps(c.namespace).Update(cache)

	if err != nil {
		log.Infof("Error creating new cache entry for %s: %s", key, err)
		return err
	}
	return nil
}
