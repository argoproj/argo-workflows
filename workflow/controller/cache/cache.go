package cache

import (
	"encoding/json"
	"fmt"
	"regexp"
	"sync"

	apierr "k8s.io/apimachinery/pkg/api/errors"

	log "github.com/sirupsen/logrus"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
)

var cacheKeyRegex = regexp.MustCompile("^[a-zA-Z0-9][-a-zA-Z0-9]*$")

type MemoizationCache interface {
	Load(key string) (*CacheEntry, error)
	Save(key string, nodeId string, value *wfv1.Outputs) error
}

type CacheEntry struct {
	NodeID  string        `json:"nodeID"`
	Outputs *wfv1.Outputs `json:"outputs"`
}

type cacheFactory struct {
	caches     map[string]*MemoizationCache
	kubeclient kubernetes.Interface
	namespace  string
}

type CacheFactory interface {
	GetCache(ct CacheType, name string) *MemoizationCache
}

func NewCacheFactory(ki kubernetes.Interface, ns string) CacheFactory {
	return &cacheFactory{
		make(map[string]*MemoizationCache),
		ki,
		ns,
	}
}

type CacheType string

const (
	// Only config maps are currently supported for caching
	ConfigMapCache CacheType = "ConfigMapCache"
)

// Returns a cache if it exists and creates it otherwise
func (cf *cacheFactory) GetCache(ct CacheType, name string) *MemoizationCache {
	idx := string(ct) + "." + name
	if c := cf.caches[idx]; c != nil {
		return c
	}
	switch ct {
	case ConfigMapCache:
		c := NewConfigMapCache(cf.namespace, cf.kubeclient, name)
		cf.caches[idx] = &c
		return &c
	default:
		return nil
	}
}

// ConfigMap cache

type configMapCache struct {
	namespace  string
	name       string
	kubeClient kubernetes.Interface
	locked     sync.RWMutex
}

func NewConfigMapCache(ns string, ki kubernetes.Interface, n string) MemoizationCache {
	return &configMapCache{
		namespace:  ns,
		name:       n,
		kubeClient: ki,
		locked:     sync.RWMutex{},
	}
}

func (c *configMapCache) logError(err error, fields log.Fields, message string) {
	log.WithFields(log.Fields{"namespace": c.namespace, "name": c.name}).WithFields(fields).WithError(err).Debug(message)
}

func (c *configMapCache) logInfo(fields log.Fields, message string) {
	log.WithFields(log.Fields{"namespace": c.namespace, "name": c.name}).WithFields(fields).Info(message)
}

func (c *configMapCache) Load(key string) (*CacheEntry, error) {
	if !cacheKeyRegex.MatchString(key) {
		return nil, fmt.Errorf("Invalid cache key %s", key)
	}
	c.locked.Lock()
	defer c.locked.Unlock()
	cm, err := c.kubeClient.CoreV1().ConfigMaps(c.namespace).Get(c.name, metav1.GetOptions{})
	if apierr.IsNotFound(err) {
		c.logError(err, log.Fields{}, "config map cache miss: config map does not exist")
		return nil, nil
	}
	if err != nil {
		c.logError(err, log.Fields{}, "Error loading config map cache")
		return nil, fmt.Errorf("could not load config map cache: %w", err)
	}
	c.logInfo(log.Fields{}, "config map cache loaded")
	rawEntry, ok := cm.Data[key]
	if !ok || rawEntry == "" {
		c.logInfo(log.Fields{}, "config map cache miss: entry does not exist")
		return nil, nil
	}
	var entry CacheEntry
	err = json.Unmarshal([]byte(rawEntry), &entry)
	if err != nil {
		return nil, fmt.Errorf("malformed cache entry: could not unmarshal JSON; unable to parse: %w", err)
	}
	outputs := entry.Outputs
	c.logInfo(log.Fields{"key": key}, "config map cache hit")
	return &CacheEntry{
		Outputs: outputs,
	}, nil
}

func (c *configMapCache) Save(key string, nodeId string, value *wfv1.Outputs) error {
	if !cacheKeyRegex.MatchString(key) {
		err := fmt.Errorf("Invalid cache key")
		c.logError(err, log.Fields{"key": key}, "Invalid cache key")
		return err
	}
	c.locked.Lock()
	defer c.locked.Unlock()
	c.logInfo(log.Fields{"key": key, "nodeId": nodeId}, "Saving ConfigMap cache entry")
	cache, err := c.kubeClient.CoreV1().ConfigMaps(c.namespace).Get(c.name, metav1.GetOptions{})
	if apierr.IsNotFound(err) || cache == nil {
		cache, err = c.kubeClient.CoreV1().ConfigMaps(c.namespace).Create(&apiv1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name: c.name,
			},
		})
		if err != nil {
			c.logError(err, log.Fields{"key": key, "nodeId": nodeId}, "Error saving to ConfigMap cache")
			return fmt.Errorf("could not save to config map cache: %w", err)
		}
	}

	newEntry := CacheEntry{
		NodeID:  nodeId,
		Outputs: value,
	}

	entryJSON, err := json.Marshal(newEntry)
	if err != nil {
		c.logError(err, log.Fields{"key": key, "nodeId": nodeId}, "Unable to marshal cache entry")
		return fmt.Errorf("unable to marshal cache entry: %w", err)
	}

	if cache.Data == nil {
		cache.Data = make(map[string]string)
	}
	cache.Data[key] = string(entryJSON)

	_, err = c.kubeClient.CoreV1().ConfigMaps(c.namespace).Update(cache)
	if err != nil {
		c.logError(err, log.Fields{"key": key, "nodeId": nodeId}, "Kubernetes error creating new cache entry")
		return fmt.Errorf("error creating cache entry: %w", err)
	}
	return nil
}
