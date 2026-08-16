package sync

import (
	"context"
	"strings"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/util/logging"
	wfmetrics "github.com/argoproj/argo-workflows/v4/workflow/metrics"
)

// syncMetrics is the subset of the metrics recorder the sync Manager needs to emit the
// locks-taken counter. The current-state gauges (locks_held / locks_pending) are pulled
// separately via Manager.LockMetrics at scrape time, so they are not part of this interface.
type syncMetrics interface {
	RecordLockTaken(ctx context.Context, lockType, storage, name, namespace string)
}

const (
	storageConfigMap = "configmap"
	storageDatabase  = "database"
)

// parseLockKey splits an encoded lock key into the parts needed for metric labels. Unlike
// DecodeLockName it does not log or validate, so it is safe to call on the metrics scrape path.
func parseLockKey(key string) (namespace, name, storage string, ok bool) {
	items := strings.SplitN(key, "/", 3)
	if len(items) < 3 {
		return "", "", "", false
	}
	namespace = items[0]
	// items[2] is the remainder after namespace/kind. For a ConfigMap semaphore it is
	// "resourceName/key" and both parts together identify the specific lock: a single ConfigMap can
	// hold multiple independent semaphores keyed by "key", so the key must be kept to avoid distinct
	// locks collapsing into one metric series. For mutexes and database locks it is the resource name.
	name = items[2]
	switch lockKind(items[1]) {
	case lockKindConfigMap, lockKindMutex:
		storage = storageConfigMap
	case lockKindDatabase:
		storage = storageDatabase
	default:
		return "", "", "", false
	}
	return namespace, name, storage, true
}

// syncTypeLabel maps a SynchronizationType to its metric label value.
func syncTypeLabel(t wfv1.SynchronizationType) string {
	if t == wfv1.SynchronizationTypeMutex {
		return string(lockTypeMutex)
	}
	return string(lockTypeSemaphore)
}

// parseDBStateName decodes a database state-table lock name (as produced by databaseSemaphore's
// longDBKey, e.g. "sem/<namespace>/<resource>" or "mtx/<namespace>/<resource>") into metric labels.
func parseDBStateName(dbName string) (lockType, name, namespace string, ok bool) {
	prefix, rest, found := strings.Cut(dbName, "/")
	if !found {
		return "", "", "", false
	}
	switch prefix {
	case "mtx":
		lockType = string(lockTypeMutex)
	case "sem":
		lockType = string(lockTypeSemaphore)
	default:
		return "", "", "", false
	}
	namespace, name, found = strings.Cut(rest, "/")
	if !found {
		return "", "", "", false
	}
	return lockType, name, namespace, true
}

// LockMetrics returns a point-in-time snapshot of the locks this controller currently participates
// in, for the observable locks_held / locks_pending gauges. It is called at metric scrape time.
//
// In-memory locks (ConfigMap semaphores and Mutexes) are read directly from memory. Database-backed
// locks are read with a single controller-scoped aggregate query rather than one query per lock, so
// each controller reports only its own contribution; `sum by (lock_name)` across controllers yields
// the true global picture without double-counting.
func (sm *Manager) LockMetrics(ctx context.Context) []wfmetrics.LockGaugeSample {
	// The observable-gauge callback is invoked by the metrics scrape with a bare context that has no
	// logger. Downstream code (e.g. the database session) calls RequireLoggerFromContext and would
	// panic, so attach the Manager's logger before doing any work.
	ctx = logging.WithLogger(ctx, sm.log)
	samples := sm.inMemoryLockSamples(ctx)
	return append(samples, sm.databaseLockSamples(ctx)...)
}

// inMemoryLockSamples snapshots ConfigMap and Mutex (in-memory) locks from syncLockMap. Database
// locks are skipped here; they are reported by databaseLockSamples. Poisoned locks are skipped.
func (sm *Manager) inMemoryLockSamples(ctx context.Context) []wfmetrics.LockGaugeSample {
	sm.lock.RLock()
	defer sm.lock.RUnlock()

	samples := make([]wfmetrics.LockGaugeSample, 0, len(sm.syncLockMap))
	for key, lock := range sm.syncLockMap {
		if _, poisoned := lock.(*poisonedLock); poisoned {
			continue
		}
		namespace, name, storage, ok := parseLockKey(key)
		if !ok || storage == storageDatabase {
			continue
		}
		lockType := string(lockTypeSemaphore)
		if storage == storageConfigMap && strings.SplitN(key, "/", 3)[1] == string(lockKindMutex) {
			lockType = string(lockTypeMutex)
		}
		holders, err := lock.getCurrentHolders(ctx)
		if err != nil {
			sm.log.WithField("lockKey", key).WithError(err).Debug(ctx, "could not read lock holders for metrics")
			continue
		}
		pending, err := lock.getCurrentPending(ctx)
		if err != nil {
			sm.log.WithField("lockKey", key).WithError(err).Debug(ctx, "could not read lock pending for metrics")
			continue
		}
		samples = append(samples, wfmetrics.LockGaugeSample{
			Type:      lockType,
			Storage:   storage,
			Name:      name,
			Namespace: namespace,
			Held:      int64(len(holders)),
			Pending:   int64(len(pending)),
		})
	}
	return samples
}

// databaseLockSamples reports this controller's contribution to database-backed locks using a single
// aggregate query. Returns nil when no database is configured.
func (sm *Manager) databaseLockSamples(ctx context.Context) []wfmetrics.LockGaugeSample {
	if sm.dbInfo.SessionProxy == nil {
		return nil
	}
	counts, err := sm.queries.GetStateCountsByController(ctx, sm.dbInfo.Config.ControllerName)
	if err != nil {
		sm.log.WithError(err).Debug(ctx, "could not read database lock counts for metrics")
		return nil
	}
	byLock := make(map[string]*wfmetrics.LockGaugeSample, len(counts))
	for _, c := range counts {
		lockType, name, namespace, ok := parseDBStateName(c.Name)
		if !ok {
			continue
		}
		s := byLock[c.Name]
		if s == nil {
			s = &wfmetrics.LockGaugeSample{Type: lockType, Storage: storageDatabase, Name: name, Namespace: namespace}
			byLock[c.Name] = s
		}
		if c.Held {
			s.Held += c.Count
		} else {
			s.Pending += c.Count
		}
	}
	samples := make([]wfmetrics.LockGaugeSample, 0, len(byLock))
	for _, s := range byLock {
		samples = append(samples, *s)
	}
	return samples
}
