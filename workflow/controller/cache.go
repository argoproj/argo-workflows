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
// Top level of abstraction: Cache for operator to interact with cache
// Lower level: Interface for interacting with K8S that can be substituted w mock for testing


type Cache interface {
	Load(key string) (*wfv1.Outputs, bool)
	Save(key string, value *wfv1.Outputs) bool
}

type CacheEntry struct {
	ExpiresAt string `json"expiresAt"`
	NodeID string `json"nodeID"`
	Outputs wfv1.Outputs `json"outputs"`
}

type configMapCache struct {
	configMapName string
	configMapClient *configMapClient
}

type ConfigMapClient interface {
	Create(*apiv1.ConfigMap) (*apiv1.ConfigMap, error)
	Get(string) (*apiv1.ConfigMap, error)
	Update(*apiv1.ConfigMap) (*apiv1.ConfigMap, error)
}

type configMapClient struct {
	namespace string
	kubeClient kubernetes.Interface
}

func NewConfigMapCache(cm string, ns string, ki kubernetes.Interface) *configMapCache {
	cmc := configMapClient{
		namespace: ns,
		kubeClient: ki,
	}
	return &configMapCache{
		configMapName: cm,
		configMapClient: &cmc,
	}
}

func foo(bar ConfigMapClient) {
	return
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

func (c *configMapClient) Create(cm *apiv1.ConfigMap) (*apiv1.ConfigMap, error) {
	return c.kubeClient.CoreV1().ConfigMaps(c.namespace).Create(cm)
}

func (c *configMapClient) Get(name string) (*apiv1.ConfigMap, error) {
	cm, err := c.kubeClient.CoreV1().ConfigMaps(c.namespace).Get(name, metav1.GetOptions{})
	if err != nil {
		log.Infof("Error loading ConfigMap cache %s: %s", name, err)
		return nil, err
	}
	return cm, nil
}

func (c *configMapClient) Update(cm *apiv1.ConfigMap) (*apiv1.ConfigMap, error) {
	return c.kubeClient.CoreV1().ConfigMaps(c.namespace).Update(cm)
}

func (c *configMapCache) Load(key string) (*wfv1.Outputs, bool) {
	cm, err := c.configMapClient.Get(c.configMapName)
	if cm == nil {
		log.Infof("Cache miss: ConfigMap does not exist")
		return nil, false
	}
	log.Infof("ConfigMap cache %s loaded", c.configMapName)
	key = validateCacheKey(key)
	rawEntry, ok := cm.Data[key];
	if !ok || rawEntry == "" {
		log.Infof("Cache miss: Entry for %s doesn't exist", key)
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
	cm, err := c.configMapClient.Get(c.configMapName)
	if len(cm.Data) == 0 && err != nil {
		_, err = c.configMapClient.Create(&apiv1.ConfigMap{
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

	_, err = c.configMapClient.Update(&opts)

	if err != nil {
		log.Infof("Error creating new cache entry for %s: %s", key, err)
		return false
	}
	return true
}
