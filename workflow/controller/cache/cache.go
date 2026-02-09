package cache

import (
	"context"
	"regexp"
	"sync"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
)

var cacheKeyRegex = regexp.MustCompile("^[a-zA-Z0-9][-a-zA-Z0-9]*$")

type MemoizationCache interface {
	Load(ctx context.Context, key string) (*Entry, error)
	Save(ctx context.Context, key string, nodeID string, value *wfv1.Outputs) error
}

type Entry struct {
	NodeID            string        `json:"nodeID"`
	Outputs           *wfv1.Outputs `json:"outputs"`
	CreationTimestamp metav1.Time   `json:"creationTimestamp"`
	LastHitTimestamp  metav1.Time   `json:"lastHitTimestamp"`
}

func (e *Entry) Hit() bool {
	return e != nil && e.NodeID != ""
}

func (e *Entry) GetOutputs() *wfv1.Outputs {
	if e == nil {
		return nil
	}
	return e.Outputs
}

func (e *Entry) GetOutputsWithMaxAge(maxAge time.Duration) (*wfv1.Outputs, bool) {
	if e == nil {
		return nil, false
	}
	if time.Since(e.CreationTimestamp.Time) > maxAge {
		// Outputs have expired
		return nil, false
	}
	return e.Outputs, true
}

type cacheFactory struct {
	caches     map[string]MemoizationCache
	kubeclient kubernetes.Interface
	namespace  string
	lock       sync.RWMutex
}

type Factory interface {
	GetCache(ct Type, name string) MemoizationCache
}

func NewCacheFactory(ki kubernetes.Interface, ns string) Factory {
	return &cacheFactory{
		make(map[string]MemoizationCache),
		ki,
		ns,
		sync.RWMutex{},
	}
}

type Type string

const (
	// Only config maps are currently supported for caching
	ConfigMapCache Type = "ConfigMapCache"
)

// Returns a cache if it exists and creates it otherwise
func (cf *cacheFactory) GetCache(ct Type, name string) MemoizationCache {
	cf.lock.RLock()

	idx := string(ct) + "." + name
	if c := cf.caches[idx]; c != nil {
		cf.lock.RUnlock()
		return c
	}
	cf.lock.RUnlock()

	cf.lock.Lock()
	defer cf.lock.Unlock()

	if c := cf.caches[idx]; c != nil {
		return c
	}

	switch ct {
	case ConfigMapCache:
		c := NewConfigMapCache(cf.namespace, cf.kubeclient, name)
		cf.caches[idx] = c
		return c
	default:
		return nil
	}
}
