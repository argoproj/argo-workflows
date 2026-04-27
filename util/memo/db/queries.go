package db

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
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

// MemoizationDB is the interface for database-backed memoization cache operations.
type MemoizationDB interface {
	Load(ctx context.Context, namespace, cacheName, cacheKey string) (*CacheRecord, error)
	Save(ctx context.Context, namespace, cacheName, cacheKey, nodeID string, outputs *wfv1.Outputs, maxAgeSeconds int64) error
	Prune(ctx context.Context) (int64, error)
	IsEnabled() bool
}

// NullMemoizationDB is a no-op implementation used when database memoization is disabled.
var NullMemoizationDB MemoizationDB = &nullMemoizationDB{}

type nullMemoizationDB struct{}

func (n *nullMemoizationDB) Load(context.Context, string, string, string) (*CacheRecord, error) {
	return nil, nil
}

func (n *nullMemoizationDB) Save(context.Context, string, string, string, string, *wfv1.Outputs, int64) error {
	return nil
}

func (n *nullMemoizationDB) Prune(context.Context) (int64, error) {
	return 0, nil
}

func (n *nullMemoizationDB) IsEnabled() bool {
	return false
}

var _ MemoizationDB = &queries{}

// queries provides database operations for the memoization cache table.
type queries struct {
	tableName    string
	sessionProxy *sqldb.SessionProxy
}

func NewQueries(tableName string, sessionProxy *sqldb.SessionProxy) (MemoizationDB, error) {
	if err := validateTableName(tableName); err != nil {
		return nil, err
	}
	return &queries{tableName: tableName, sessionProxy: sessionProxy}, nil
}

func (q *queries) IsEnabled() bool {
	return true
}

func cacheRecordCond(record *CacheRecord) db.Cond {
	return db.Cond{
		colNamespace: record.Namespace,
		colCacheName: record.CacheName,
		colCacheKey:  record.CacheKey,
	}
}

func cacheRecordUpdates(record *CacheRecord) map[string]any {
	return map[string]any{
		"node_id":    record.NodeID,
		"outputs":    record.Outputs,
		"created_at": record.CreatedAt,
		"expires_at": record.ExpiresAt,
	}
}

func isDuplicateKeyError(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), "duplicate key") || strings.Contains(err.Error(), "Duplicate entry")
}

func saveRecord(sess db.Session, tableName string, record *CacheRecord) error {
	collection := sess.Collection(tableName)
	_, err := collection.Insert(record)
	if err == nil {
		return nil
	}
	if !isDuplicateKeyError(err) {
		return err
	}
	return collection.Find(cacheRecordCond(record)).Update(cacheRecordUpdates(record))
}

// Load retrieves the outputs for the given cache key.
// Returns nil when the entry does not exist or has expired.
func (q *queries) Load(ctx context.Context, namespace, cacheName, cacheKey string) (*CacheRecord, error) {
	var r CacheRecord
	now := time.Now().UTC()
	err := q.sessionProxy.With(ctx, func(sess db.Session) error {
		return sess.SQL().
			SelectFrom(q.tableName).
			Where(db.Cond{colNamespace: namespace}).
			And(db.Cond{colCacheName: cacheName}).
			And(db.Cond{colCacheKey: cacheKey}).
			And(db.Cond{colExpiresAt + " >": now}).
			One(&r)
	})
	if errors.Is(err, db.ErrNoMoreRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &r, nil
}

// Prune deletes cache entries whose expires_at has elapsed. It is called
// periodically by the controller to bound the size of the configured memoization cache table.
func (q *queries) Prune(ctx context.Context) (int64, error) {
	now := time.Now().UTC()
	var n int64
	err := q.sessionProxy.With(ctx, func(sess db.Session) error {
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

func (q *queries) Save(ctx context.Context, namespace, cacheName, cacheKey, nodeID string, outputs *wfv1.Outputs, maxAgeSeconds int64) error {
	outputsJSON, err := json.Marshal(outputs)
	if err != nil {
		return fmt.Errorf("unable to marshal memoization outputs: %w", err)
	}
	now := time.Now().UTC()
	expiresAt := now.Add(time.Duration(maxAgeSeconds) * time.Second)
	record := &CacheRecord{
		Namespace: namespace,
		CacheName: cacheName,
		CacheKey:  cacheKey,
		NodeID:    nodeID,
		Outputs:   string(outputsJSON),
		CreatedAt: now,
		ExpiresAt: expiresAt,
	}
	return q.sessionProxy.With(ctx, func(sess db.Session) error {
		return saveRecord(sess, q.tableName, record)
	})
}
