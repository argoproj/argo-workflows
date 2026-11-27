package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"k8s.io/apimachinery/pkg/util/wait"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	argoerr "github.com/argoproj/argo-workflows/v4/util/errors"
	memodb "github.com/argoproj/argo-workflows/v4/util/memo/db"
	"github.com/argoproj/argo-workflows/v4/util/sqldb"
)

type sqlDBCache struct {
	namespace   string
	name        string
	withQueries func(func(memodb.MemoizationDB) error) error
}

func newSQLDBCache(namespace, name string, withQueries func(func(memodb.MemoizationDB) error) error) MemoizationCache {
	return &sqlDBCache{
		namespace:   namespace,
		name:        name,
		withQueries: withQueries,
	}
}

func (c *sqlDBCache) Load(ctx context.Context, key string) (*Entry, error) {
	if err := validateCacheKey(key); err != nil {
		return nil, err
	}
	var record *memodb.CacheRecord
	err := retryMemoization(ctx, func() error {
		return c.withQueries(func(queries memodb.MemoizationDB) error {
			var loadErr error
			record, loadErr = queries.Load(ctx, c.namespace, c.name, key)
			return loadErr
		})
	})
	if err != nil {
		return nil, fmt.Errorf("memoization db load failed: %w", err)
	}
	if record == nil {
		return nil, nil
	}
	var outputs *wfv1.Outputs
	if err := json.Unmarshal([]byte(record.Outputs), &outputs); err != nil {
		return nil, fmt.Errorf("malformed memoization db entry: could not unmarshal outputs JSON: %w", err)
	}
	return &Entry{
		NodeID:            record.NodeID,
		Outputs:           outputs,
		CreationTimestamp: metav1.Time{Time: record.CreatedAt},
		LastHitTimestamp:  metav1.Time{Time: record.CreatedAt},
	}, nil
}

func (c *sqlDBCache) Save(ctx context.Context, key string, nodeID string, value *wfv1.Outputs, maxAge string) error {
	if err := validateCacheKey(key); err != nil {
		return err
	}
	return retryMemoization(ctx, func() error {
		return c.withQueries(func(queries memodb.MemoizationDB) error {
			return queries.Save(ctx, c.namespace, c.name, key, nodeID, value, maxAge)
		})
	})
}

func retryMemoization(ctx context.Context, operation func() error) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	var lastErr error
	err := wait.ExponentialBackoffWithContext(ctx, wait.Backoff{Duration: time.Second, Factor: 2, Jitter: 0.1, Steps: 5, Cap: 30 * time.Second}, func(context.Context) (bool, error) {
		lastErr = operation()
		if sqldb.IsTransientError(lastErr) || argoerr.IsTransientErr(ctx, lastErr) {
			return false, nil
		}
		return true, lastErr
	})
	if wait.Interrupted(err) && ctx.Err() == nil && lastErr != nil {
		return lastErr
	}
	return err
}
