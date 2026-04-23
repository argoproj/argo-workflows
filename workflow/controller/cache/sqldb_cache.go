package cache

import (
	"context"
	"encoding/json"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	memodb "github.com/argoproj/argo-workflows/v4/util/memo/db"
	"github.com/argoproj/argo-workflows/v4/util/sqldb"
)

type sqlDBCache struct {
	namespace    string
	name         string
	sessionProxy *sqldb.SessionProxy
	queries      *memodb.Queries
}

func newSQLDBCache(namespace, name string, sp *sqldb.SessionProxy, tableName string) (MemoizationCache, error) {
	queries, err := memodb.NewQueries(tableName, sp.DBType())
	if err != nil {
		return nil, err
	}
	return &sqlDBCache{
		namespace:    namespace,
		name:         name,
		sessionProxy: sp,
		queries:      queries,
	}, nil
}

func (c *sqlDBCache) Load(ctx context.Context, key string) (*Entry, error) {
	if !cacheKeyRegex.MatchString(key) {
		return nil, fmt.Errorf("invalid cache key: %s", key)
	}
	record, err := c.queries.Load(ctx, c.sessionProxy, c.namespace, c.name, key)
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

func (c *sqlDBCache) Save(ctx context.Context, key string, nodeID string, value *wfv1.Outputs, maxAgeSeconds int64) error {
	if !cacheKeyRegex.MatchString(key) {
		return fmt.Errorf("invalid cache key: %s", key)
	}
	return c.queries.Save(ctx, c.sessionProxy, c.namespace, c.name, key, nodeID, value, maxAgeSeconds)
}
