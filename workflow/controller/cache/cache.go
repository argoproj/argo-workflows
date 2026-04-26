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
	"github.com/argoproj/argo-workflows/v4/util/sqldb"
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
	caches       map[string]MemoizationCache
	kubeclient   kubernetes.Interface
	lock         sync.RWMutex
	sessionProxy *sqldb.SessionProxy
	tableName    string
	// sqlEnabled indicates that SQL caching was explicitly configured by the operator, even if
	// the session proxy is currently unavailable. When true and sessionProxy is nil, GetCache
	// returns nil rather than silently falling back to ConfigMap-based caching.
	sqlEnabled bool
}

type Factory interface {
	GetCache(ctx context.Context, ct Type, namespace, name string) MemoizationCache
	// SetSessionProxy configures the factory to use database-backed caching with the given
	// session proxy and table name. Calling this clears any previously created cache instances
	// so they are recreated against the SQL backend.
	SetSessionProxy(sp *sqldb.SessionProxy, tableName string)
	// ClearSessionProxy removes any SQL backend configuration. If sqlEnabled is true, GetCache
	// returns nil rather than silently falling back to ConfigMap-based caching (e.g. after a
	// transient DB failure). If sqlEnabled is false, GetCache falls back to ConfigMap caching.
	ClearSessionProxy(sqlEnabled bool)
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
	// When a database session proxy is configured, SQL-backed caching is used instead.
	ConfigMapCache Type = "ConfigMapCache"
)

// SetSessionProxy configures the factory to use a SQL backend, clearing any previously
// cached instances so they are recreated against the new backend.
func (cf *cacheFactory) SetSessionProxy(sp *sqldb.SessionProxy, tableName string) {
	cf.lock.Lock()
	defer cf.lock.Unlock()
	cf.sessionProxy = sp
	cf.tableName = tableName
	cf.sqlEnabled = true
	cf.caches = make(map[string]MemoizationCache)
}

// ClearSessionProxy removes the SQL backend. When sqlEnabled is true (DB configured but
// temporarily unavailable), GetCache returns nil. When false (no DB configured), GetCache
// falls back to ConfigMap-based caching.
func (cf *cacheFactory) ClearSessionProxy(sqlEnabled bool) {
	cf.lock.Lock()
	defer cf.lock.Unlock()
	cf.sessionProxy = nil
	cf.tableName = ""
	cf.sqlEnabled = sqlEnabled
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
		switch {
		case cf.sessionProxy != nil:
			var err error
			c, err = newSQLDBCache(namespace, name, cf.sessionProxy, cf.tableName)
			if err != nil {
				logger.WithFields(logging.Fields{"cacheName": name, "workflowNamespace": namespace}).WithError(err).Error(ctx, "Failed to create SQL memoization cache")
				return nil
			}
		case cf.sqlEnabled:
			// SQL was explicitly configured but is currently unavailable. Return nil so callers
			// can skip caching rather than silently redirecting to a ConfigMap backend.
			logger.WithFields(logging.Fields{"cacheName": name, "workflowNamespace": namespace}).Warn(ctx, "SQL memoization cache requested but SQL backend is unavailable; skipping cache")
			return nil
		default:
			c = NewConfigMapCache(namespace, cf.kubeclient, name)
		}
		cf.caches[idx] = c
		return c
	default:
		return nil
	}
}
