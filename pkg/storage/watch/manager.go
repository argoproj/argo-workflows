package watch

import (
	"context"
	"encoding/json"
	"sync"
	"sync/atomic"
	"time"

	"gorm.io/gorm"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"

	"github.com/argoproj/argo-workflows/v4/pkg/storage/models"
)

const (
	defaultBufferSize    = 100
	defaultCleanupTTL    = 5 * time.Minute
	defaultCleanupPeriod = 1 * time.Minute
)

// Manager manages active watchers and fan-out of events.
type Manager struct {
	db       *gorm.DB
	mu       sync.RWMutex
	watchers map[uint64]*watcher
	nextID   atomic.Uint64
}

// NewManager creates a new WatchManager.
func NewManager(db *gorm.DB) *Manager {
	return &Manager{
		db:       db,
		watchers: make(map[uint64]*watcher),
	}
}

// Watch starts watching for events matching the given kind and namespace.
// If resourceVersion > 0, it replays missed events from the watch_events table.
func (m *Manager) Watch(kind, namespace string, resourceVersion int64, scheme *runtime.Scheme) (watch.Interface, error) {
	id := m.nextID.Add(1)

	w := newWatcher(id, kind, namespace, defaultBufferSize, m.removeWatcher)

	// Replay missed events if a resource version was provided.
	if resourceVersion > 0 {
		var events []models.WatchEvent
		query := m.db.Where("kind = ? AND resource_version > ?", kind, resourceVersion).
			Order("resource_version ASC")
		if namespace != "" {
			query = query.Where("namespace = ?", namespace)
		}
		if err := query.Find(&events).Error; err != nil {
			return nil, err
		}
		for _, evt := range events {
			obj, err := deserializeEvent(evt.Data)
			if err != nil {
				continue
			}
			w.send(watch.Event{
				Type:   watch.EventType(evt.EventType),
				Object: obj,
			})
		}
	}

	m.mu.Lock()
	m.watchers[id] = w
	m.mu.Unlock()

	return w, nil
}

// Notify fans out an event to all matching watchers.
func (m *Manager) Notify(kind, namespace string, eventType watch.EventType, obj runtime.Object, rv int64) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	event := watch.Event{
		Type:   eventType,
		Object: obj,
	}

	for _, w := range m.watchers {
		if w.kind != kind {
			continue
		}
		if w.ns != "" && w.ns != namespace {
			continue
		}
		w.send(event)
	}
}

// StartCleanup starts a background goroutine that deletes old watch events.
func (m *Manager) StartCleanup(ctx context.Context) {
	go func() {
		ticker := time.NewTicker(defaultCleanupPeriod)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				cutoff := time.Now().Add(-defaultCleanupTTL)
				m.db.Where("created_at < ?", cutoff).Delete(&models.WatchEvent{})
			}
		}
	}()
}

func (m *Manager) removeWatcher(id uint64) {
	m.mu.Lock()
	if w, ok := m.watchers[id]; ok {
		close(w.ch)
		delete(m.watchers, id)
	}
	m.mu.Unlock()
}

func deserializeEvent(data string) (runtime.Object, error) {
	obj := &unstructured.Unstructured{}
	if err := json.Unmarshal([]byte(data), &obj.Object); err != nil {
		return nil, err
	}
	return obj, nil
}
