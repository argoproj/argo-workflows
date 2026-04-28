package cache

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"sync"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/util/logging"
	memodb "github.com/argoproj/argo-workflows/v4/util/memo/db"
)

var cacheKeyRegex = regexp.MustCompile("^[a-zA-Z0-9][-a-zA-Z0-9]*$")

// defaultMaxAgeSeconds is 30 days in seconds, used when maxAge is not specified on the template.
const defaultMaxAgeSeconds int64 = 30 * 24 * 60 * 60

// resolvedDefaultMaxAge caches the DEFAULT_MAX_AGE env var so it is only read once.
var resolvedDefaultMaxAge struct {
	once sync.Once
	secs int64
	err  error
}

// ResolveMaxAgeSeconds converts a template's maxAge duration string to seconds for SQL-backed
// memoization cache entries. If maxAge is empty, it falls back to the DEFAULT_MAX_AGE env var
// (a Go duration string like "720h"), then to 30 days. Returns an error only if the duration
// string is malformed.
func ResolveMaxAgeSeconds(maxAge string) (int64, error) {
	if maxAge == "" {
		resolvedDefaultMaxAge.once.Do(func() {
			envVal := os.Getenv("DEFAULT_MAX_AGE")
			if envVal == "" {
				resolvedDefaultMaxAge.secs = defaultMaxAgeSeconds
				return
			}
			// Try parsing as a Go duration first (e.g. "720h")
			if d, err := time.ParseDuration(envVal); err == nil {
				resolvedDefaultMaxAge.secs = int64(d.Seconds())
				return
			}
			// Fall back to parsing as raw seconds (e.g. "2592000")
			if secs, err := strconv.ParseInt(envVal, 10, 64); err == nil {
				resolvedDefaultMaxAge.secs = secs
				return
			}
			resolvedDefaultMaxAge.err = fmt.Errorf("invalid DEFAULT_MAX_AGE value %q: must be a Go duration (e.g. 720h) or integer seconds", envVal)
		})
		return resolvedDefaultMaxAge.secs, resolvedDefaultMaxAge.err
	}
	d, err := time.ParseDuration(maxAge)
	if err != nil {
		return 0, fmt.Errorf("invalid maxAge %q: %w", maxAge, err)
	}
	return int64(d.Seconds()), nil
}

type MemoizationCache interface {
	Load(ctx context.Context, key string) (*Entry, error)
	// Save stores the outputs of a completed memoized node. ConfigMap-backed caches ignore maxAge.
	// SQL-backed caches use maxAge, or DEFAULT_MAX_AGE when maxAge is empty, to compute expires_at.
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
	lock       sync.RWMutex
	queries    memodb.MemoizationDB
}

type Factory interface {
	GetCache(ctx context.Context, ct Type, namespace, name string) MemoizationCache
	// SetQueries configures the factory to use database-backed caching with the given
	// MemoizationDB. Calling this clears any previously created cache instances
	// so they are recreated against the SQL backend.
	SetQueries(q memodb.MemoizationDB)
}

func NewCacheFactory(ki kubernetes.Interface) Factory {
	return &cacheFactory{
		caches:     make(map[string]MemoizationCache),
		kubeclient: ki,
	}
}

type Type string

const (
	// ConfigMapCache is a cache type identifier used as a key prefix in the cache map.
	// When a MemoizationDB is configured, SQL-backed memoization semantics are used instead.
	ConfigMapCache Type = "ConfigMapCache"
)

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

// GetCache returns a cache scoped to the given workflow namespace if it exists and creates it
// otherwise.
func (cf *cacheFactory) GetCache(ctx context.Context, ct Type, namespace, name string) MemoizationCache {
	logger := logging.RequireLoggerFromContext(ctx)
	if namespace == "" {
		logger.WithField("cacheName", name).Error(ctx, "Workflow namespace is required to resolve memoization cache")
		return nil
	}

	cf.lock.RLock()

	idx := string(ct) + "." + namespace + "." + name
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
		var c MemoizationCache
		if cf.queries != nil {
			c = newSQLDBCache(namespace, name, func() memodb.MemoizationDB { return cf.queries }, &cf.lock)
		} else {
			c = NewConfigMapCache(namespace, cf.kubeclient, name)
		}
		cf.caches[idx] = c
		return c
	default:
		return nil
	}
}
