package cache

import (
	"context"
	"regexp"
	"sync"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	memodb "github.com/argoproj/argo-workflows/v4/util/memo/db"
)

var cacheKeyRegex = regexp.MustCompile("^[a-zA-Z0-9][-a-zA-Z0-9]*$")

type MemoizationCache interface {
	Load(ctx context.Context, key string) (*Entry, error)
	// Save stores the outputs of a completed memoized node. ConfigMap-backed caches ignore maxAge.
	// SQL-backed caches record an expiry only when maxAge is explicitly set.
	Save(ctx context.Context, key string, nodeID string, value *wfv1.Outputs, maxAge string) error
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
	queries    memodb.MemoizationDB
}

type Factory interface {
	GetCache(name string) MemoizationCache
	// SetQueries selects SQL-backed caching, or ConfigMap-backed caching when nil.
	SetQueries(q memodb.MemoizationDB)
}

func NewCacheFactory(ki kubernetes.Interface, namespace string) Factory {
	return &cacheFactory{
		caches:     make(map[string]MemoizationCache),
		kubeclient: ki,
		namespace:  namespace,
	}
}

// SetQueries configures the factory's memoization backend, clearing any previously
// cached instances so they are recreated against the new backend. A nil MemoizationDB
// selects ConfigMap-backed caching; a non-nil MemoizationDB selects SQL-backed
// memoization semantics, even if the DB implementation is disabled/no-op.
func (cf *cacheFactory) SetQueries(q memodb.MemoizationDB) {
	cf.lock.Lock()
	defer cf.lock.Unlock()
	cf.queries = q
	cf.caches = make(map[string]MemoizationCache)
}

// GetCache returns a cache in the controller's namespace, shared by its workflows.
func (cf *cacheFactory) GetCache(name string) MemoizationCache {
	cf.lock.RLock()
	c := cf.caches[name]
	cf.lock.RUnlock()
	if c != nil {
		return c
	}

	cf.lock.Lock()
	defer cf.lock.Unlock()
	c = cf.caches[name]
	if c != nil {
		return c
	}

	if cf.queries != nil {
		c = newSQLDBCache(cf.namespace, name, cf.getQueries)
	} else {
		c = NewConfigMapCache(cf.namespace, cf.kubeclient, name)
	}
	cf.caches[name] = c
	return c
}

func (cf *cacheFactory) getQueries() memodb.MemoizationDB {
	cf.lock.RLock()
	defer cf.lock.RUnlock()
	if cf.queries == nil {
		return memodb.NullMemoizationDB
	}
	return cf.queries
}
