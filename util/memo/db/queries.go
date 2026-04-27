package db

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/upper/db/v4"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/util/sqldb"
)

const (
	colNamespace = "namespace"
	colCacheName = "cache_name"
	colCacheKey  = "cache_key"
	colExpiresAt = "expires_at"
)

// CacheRecord is the database row for a single memoization cache entry.
type CacheRecord struct {
	Namespace string    `db:"namespace"`
	CacheName string    `db:"cache_name"`
	CacheKey  string    `db:"cache_key"`
	NodeID    string    `db:"node_id"`
	Outputs   string    `db:"outputs"` // JSON
	CreatedAt time.Time `db:"created_at"`
	ExpiresAt time.Time `db:"expires_at"`
}

// Queries provides database operations for the memoization cache table.
type Queries struct {
	tableName string
}

func NewQueries(tableName string) (*Queries, error) {
	if err := validateTableName(tableName); err != nil {
		return nil, err
	}
	return &Queries{tableName: tableName}, nil
}

// Load retrieves the outputs for the given cache key.
// Returns nil when the entry does not exist or has expired.
func (q *Queries) Load(ctx context.Context, sp *sqldb.SessionProxy, namespace, cacheName, cacheKey string) (*CacheRecord, error) {
	var r CacheRecord
	now := time.Now().UTC()
	err := sp.With(ctx, func(sess db.Session) error {
		return sess.SQL().
			SelectFrom(q.tableName).
			Where(db.Cond{colNamespace: namespace}).
			And(db.Cond{colCacheName: cacheName}).
			And(db.Cond{colCacheKey: cacheKey}).
			And(db.Cond{colExpiresAt + " >": now}).
			One(&r)
	})
	if err == db.ErrNoMoreRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &r, nil
}

// Prune deletes cache entries whose expires_at has elapsed. It is called
// periodically by the controller to bound the size of the configured memoization cache table.
func (q *Queries) Prune(ctx context.Context, sp *sqldb.SessionProxy) (int64, error) {
	now := time.Now().UTC()
	var n int64
	err := sp.With(ctx, func(sess db.Session) error {
		result, err := sess.SQL().
			DeleteFrom(q.tableName).
			Where(db.Cond{colExpiresAt + " <": now}).
			Exec()
		if err != nil {
			return err
		}
		n, err = result.RowsAffected()
		return err
	})
	return n, err
}

func (q *Queries) Save(ctx context.Context, sp *sqldb.SessionProxy, namespace, cacheName, cacheKey, nodeID string, outputs *wfv1.Outputs, maxAgeSeconds int64) error {
	outputsJSON, err := json.Marshal(outputs)
	if err != nil {
		return fmt.Errorf("unable to marshal memoization outputs: %w", err)
	}
	now := time.Now().UTC()
	expiresAt := now.Add(time.Duration(maxAgeSeconds) * time.Second)
	return sp.With(ctx, func(sess db.Session) error {
		return sess.TxContext(ctx, func(tx db.Session) error {
			_, err := tx.SQL().
				DeleteFrom(q.tableName).
				Where(db.Cond{colNamespace: namespace}).
				And(db.Cond{colCacheName: cacheName}).
				And(db.Cond{colCacheKey: cacheKey}).
				Exec()
			if err != nil {
				return err
			}
			_, err = tx.Collection(q.tableName).Insert(&CacheRecord{
				Namespace: namespace,
				CacheName: cacheName,
				CacheKey:  cacheKey,
				NodeID:    nodeID,
				Outputs:   string(outputsJSON),
				CreatedAt: now,
				ExpiresAt: expiresAt,
			})
			return err
		}, nil)
	})
}
