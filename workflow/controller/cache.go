package controller

import (
	"encoding/json"
	"regexp"

	apierr "k8s.io/apimachinery/pkg/api/errors"

	"github.com/argoproj/argo/errors"

	log "github.com/sirupsen/logrus"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
)

type MemoizationCache interface {
	Load(key string) (*wfv1.Outputs, error)
	Save(key string, nodeId string, value *wfv1.Outputs) error
}

type CacheEntry struct {
	ExpiresAt string       `json:"expiresAt"`
	NodeID    string       `json:"nodeID"`
	Outputs   wfv1.Outputs `json:"outputs"`
}

type cacheMutex bool

type CacheMutex interface {
	Lock()
	Unlock()
}

type configMapCache struct {
	namespace     string
	configMapName string
	kubeClient    kubernetes.Interface
	locked        cacheMutex
}

func NewConfigMapCache(cm string, ns string, ki kubernetes.Interface) MemoizationCache {
	return &configMapCache{
		configMapName: cm,
		namespace:     ns,
		kubeClient:    ki,
		locked:        false,
	}
}

func generateCacheKey(key string) (string, error) {
	log.Infof("Validating cache key %s", key)
	reg, err := regexp.Compile("[^-._a-zA-Z0-9]+")
	if err != nil {
		return "", err
	}
	s := reg.ReplaceAllString(key, "-")
	return s, nil
}

func (c *configMapCache) Load(key string) (*wfv1.Outputs, error) {
	if c.locked {
		log.Warnf("MemoizationCache miss: Cache locked")
		return nil, nil
	}
	c.locked.Lock()
	cm, err := c.kubeClient.CoreV1().ConfigMaps(c.namespace).Get(c.configMapName, metav1.GetOptions{})
	c.locked.Unlock()
	if apierr.IsNotFound(err) {
		log.Infof("MemoizationCache miss: ConfigMap does not exist")
		return nil, nil
	}
	if err != nil {
		log.Infof("Error loading ConfigMap cache %s in namespace %s: %s", c.configMapName, c.namespace, err)
		return nil, err
	}
	log.Infof("ConfigMap cache %s loaded", c.configMapName)
	key, err = generateCacheKey(key)
	if err != nil {
		return nil, err
	}
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
	log.Infof("ConfigMap cache %s hit for %s", c.configMapName, key)
	return &outputs, nil
}

func (c *configMapCache) Save(key string, nodeId string, value *wfv1.Outputs) error {
	if c.locked {
		log.Warnf("Could not save to cache")
		return errors.InternalError("Could not save to cache: Cache locked")
	}
	log.Infof("Saving to cache %s...", key)
	c.locked.Lock()
	cache, err := c.kubeClient.CoreV1().ConfigMaps(c.namespace).Get(c.configMapName, metav1.GetOptions{})
	c.locked.Unlock()
	if apierr.IsNotFound(err) {
		c.locked.Lock()
		_, err = c.kubeClient.CoreV1().ConfigMaps(c.namespace).Create(&apiv1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name: c.configMapName,
			},
		})
		c.locked.Unlock()
		if err != nil {
			log.Warnf("Error saving to cache: %s", err)
			return err
		}
	}

	newEntry := CacheEntry{
		ExpiresAt: "2020-06-18T17:11:05Z",
		NodeID:    nodeId,
		Outputs:   *value,
	}

	entryJSON, err := json.Marshal(newEntry)
	if err != nil {
		return err
	}
	key, err = generateCacheKey(key)
	if err != nil {
		return err
	}
	cache.Data[key] = string(entryJSON)
	c.locked.Lock()
	_, err = c.kubeClient.CoreV1().ConfigMaps(c.namespace).Update(cache)
	c.locked.Unlock()

	if err != nil {
		log.Infof("Error creating new cache entry for %s: %s", key, err)
		return err
	}
	return nil
}

func (m *cacheMutex) Lock() {
	_ = cacheMutex(true)
	//m = &b
}

func (m *cacheMutex) Unlock() {
	_ = cacheMutex(false)
	//m = &b
}
