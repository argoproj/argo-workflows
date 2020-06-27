package controller

import (
	"encoding/json"
	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	log "github.com/sirupsen/logrus"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"regexp"
)

var sampleEntry = CacheEntry{
	ExpiresAt: "2020-06-18T17:11:05Z",
	NodeID: "memoize-abx4124-123129321123",
	Outputs: wfv1.Outputs{},
}

// TWO INTERFACES:
// Top level of abstraction: MemoizationCache for operator to interact with cache
// Lower level: Interface for interacting with K8S that can be substituted w mock for testing


type MemoizationCache interface {
	Load(key string) (*wfv1.Outputs, bool)
	Save(key string, value *wfv1.Outputs) bool
}

type CacheEntry struct {
	ExpiresAt string `json"expiresAt"`
	NodeID string `json"nodeID"`
	Outputs wfv1.Outputs `json"outputs"`
}

type configMapCache struct {
	namespace string
	configMapName string
	kubeClient kubernetes.Interface
}

func NewConfigMapCache(cm string, ns string, ki kubernetes.Interface) MemoizationCache {
	return &configMapCache{
		configMapName: cm,
		namespace: ns,
		kubeClient: ki,
	}
}

func validateCacheKey(key string) string {
	log.Infof("Validating cache key %s", key)
	reg, err := regexp.Compile("[^-._a-zA-Z0-9]+")
    if err != nil {
        log.Fatal(err)
    }
    s := reg.ReplaceAllString(key, "-")
    log.Info(s)
    return s
}

func (c *configMapCache) Load(key string) (*wfv1.Outputs, bool) {
	cm, err := c.kubeClient.CoreV1().ConfigMaps(c.namespace).Get(c.configMapName, metav1.GetOptions{})
	if err != nil {
		log.Infof("Error loading ConfigMap cache %s: %s", c.namespace, err)
		return nil, false
	}
	if cm == nil {
		log.Infof("MemoizationCache miss: ConfigMap does not exist")
		return nil, false
	}
	log.Infof("ConfigMap cache %s loaded", c.configMapName)
	key = validateCacheKey(key)
	rawEntry, ok := cm.Data[key];
	if !ok || rawEntry == "" {
		log.Infof("MemoizationCache miss: Entry for %s doesn't exist", key)
		return nil, false
	}
	var entry CacheEntry
	err = json.Unmarshal([]byte(rawEntry), &entry)
	if err != nil {
		panic(err)
	}
	outputs := entry.Outputs
	log.Infof("ConfigMap cache %s hit for %s", c.configMapName, key)
	return &outputs, true
}

func (c *configMapCache) Save(key string, value *wfv1.Outputs) bool {
	log.Infof("Saving to cache %s...", key)
	cm, err := c.kubeClient.CoreV1().ConfigMaps(c.namespace).Get(c.configMapName, metav1.GetOptions{})
	if err != nil {
		if cm == nil {
			return false
		}
		if len(cm.Data) == 0 {
			_, err = c.kubeClient.CoreV1().ConfigMaps(c.namespace).Create(&apiv1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name: c.configMapName,
				},
			},
			)
			if err != nil {
				log.Infof("Error saving to cache: %s", err)
				return false
			}
		}
	}
	sampleEntry.Outputs = *value
	entryJSON, err := json.Marshal(sampleEntry)
	key = validateCacheKey(key)
	opts := apiv1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name: c.configMapName,
		},
		Data: map[string]string{
			key: string(entryJSON),
		},
	}

	_, err = c.kubeClient.CoreV1().ConfigMaps(c.namespace).Update(&opts)

	if err != nil {
		log.Infof("Error creating new cache entry for %s: %s", key, err)
		return false
	}
	return true
}
