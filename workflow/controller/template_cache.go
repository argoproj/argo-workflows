package controller

import (
	"encoding/json"
	"slices"
	"sync"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
)

// defaultTemplateCacheMaxBytes bounds the total size of cached template sets,
// evicted FIFO when exceeded (~20 large workflows on a 10Gi-limit controller).
const defaultTemplateCacheMaxBytes = 256 << 20

type cacheEntry struct {
	version    string
	templates  []wfv1.Template
	storedSize int64
}

// templateCache caches offloaded template sets per workflow UID, keyed by (uid, sha-256 of the
// offload marker). A hit proves the database rows are byte-identical to the cached templates, so
// re-hydration can skip the fetch; re-offloading changes the hash and causes a miss, so staleness is
// impossible. FIFO-evicted once the total stored size exceeds maxBytes.
type templateCache struct {
	mu       sync.RWMutex
	byUID    map[string]cacheEntry
	order    []string // FIFO eviction order
	maxBytes int64
	bytes    int64
}

// deepCopyTemplates returns an independent copy of a template set. The cache must
// not share template bodies with callers in either direction: get and put are value
// boundaries.
func deepCopyTemplates(templates []wfv1.Template) []wfv1.Template {
	if templates == nil {
		return nil
	}
	out := make([]wfv1.Template, len(templates))
	for i := range templates {
		templates[i].DeepCopyInto(&out[i])
	}
	return out
}

func newTemplateCache(maxBytes int64) *templateCache {
	return &templateCache{
		byUID:    make(map[string]cacheEntry),
		maxBytes: maxBytes,
	}
}

func (c *templateCache) get(uid, version string) ([]wfv1.Template, bool) {
	if c == nil {
		return nil, false
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	e, ok := c.byUID[uid]
	if !ok || e.version != version {
		return nil, false
	}
	return deepCopyTemplates(e.templates), true
}

func (c *templateCache) put(uid, version string, templates []wfv1.Template) {
	if c == nil {
		return
	}
	stored, err := json.Marshal(templates)
	if err != nil {
		// Cannot size the entry; skip caching rather than guessing. The next
		// hydration simply re-fetches from the database.
		return
	}
	entry := cacheEntry{version: version, templates: deepCopyTemplates(templates), storedSize: int64(len(stored))}

	c.mu.Lock()
	defer c.mu.Unlock()
	// First insert only: re-putting a cached UID must not grow the FIFO order (steady-state leak).
	if old, ok := c.byUID[uid]; ok {
		c.bytes -= old.storedSize
	} else {
		c.order = append(c.order, uid)
	}
	c.byUID[uid] = entry
	c.bytes += entry.storedSize
	for c.bytes > c.maxBytes && len(c.order) > 0 {
		oldest := c.order[0]
		c.order = c.order[1:]
		if e, ok := c.byUID[oldest]; ok {
			c.bytes -= e.storedSize
			delete(c.byUID, oldest)
		}
	}
}

func (c *templateCache) evict(uid string) {
	if c == nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if e, ok := c.byUID[uid]; ok {
		c.bytes -= e.storedSize
		delete(c.byUID, uid)
	}
	c.order = slices.DeleteFunc(c.order, func(u string) bool { return u == uid })
}

func (c *templateCache) size() int64 {
	if c == nil {
		return 0
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.bytes
}
