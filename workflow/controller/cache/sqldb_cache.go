package cache

import (
	"context"
	"encoding/json"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	memodb "github.com/argoproj/argo-workflows/v4/util/memo/db"
)

type sqlDBCache struct {
	namespace string
	name      string
	queries   memodb.MemoizationDB
}

func newSQLDBCache(namespace, name string, queries memodb.MemoizationDB) MemoizationCache {
	return &sqlDBCache{
		namespace: namespace,
		name:      name,
		queries:   queries,
	}
}

func (c *sqlDBCache) Load(ctx context.Context, key string) (*Entry, error) {
	if !cacheKeyRegex.MatchString(key) {
		return nil, fmt.Errorf("invalid cache key: %s", key)
	}
	record, err := c.queries.Load(ctx, c.namespace, c.name, key)
	if err != nil {
		return nil, fmt.Errorf("memoization db load failed: %w", err)
	}
	if record == nil {
		return nil, nil
	}
	var outputs wfv1.Outputs
	if err := json.Unmarshal([]byte(record.Outputs), &outputs); err != nil {
		return nil, fmt.Errorf("malformed memoization db entry: could not unmarshal outputs JSON: %w", err)
	}
	return &Entry{
		NodeID:            record.NodeID,
		Outputs:           &outputs,
		CreationTimestamp: metav1.Time{Time: record.CreatedAt},
		LastHitTimestamp:  metav1.Time{Time: record.CreatedAt},
	}, nil
}

func (c *sqlDBCache) Save(ctx context.Context, key string, nodeID string, value *wfv1.Outputs, maxAge string) error {
	if !cacheKeyRegex.MatchString(key) {
		return fmt.Errorf("invalid cache key: %s", key)
	}
	maxAgeSeconds, err := ResolveMaxAgeSeconds(maxAge)
	if err != nil {
		return err
	}
	return c.queries.Save(ctx, c.namespace, c.name, key, nodeID, value, maxAgeSeconds)
}
