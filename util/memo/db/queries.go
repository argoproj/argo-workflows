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
	dbType    sqldb.DBType
}

func NewQueries(tableName string, dbType sqldb.DBType) (*Queries, error) {
	if !validTableName.MatchString(tableName) {
		return nil, fmt.Errorf("invalid table name %q: must match [A-Za-z0-9_]+", tableName)
	}
	return &Queries{tableName: tableName, dbType: dbType}, nil
}

// Load retrieves the outputs for the given cache key.
// Returns nil when the entry does not exist.
func (q *Queries) Load(ctx context.Context, sp *sqldb.SessionProxy, namespace, cacheName, cacheKey string) (*CacheRecord, error) {
	var r CacheRecord
	var found bool
	err := sp.With(ctx, func(sess db.Session) error {
		// Use raw SQL to avoid upper/db ORM timestamp scanning issues with
		// "timestamp without timezone" columns (the ORM may not populate time.Time fields).
		var query string
		switch q.dbType {
		case sqldb.Postgres:
			query = fmt.Sprintf(`SELECT namespace, cache_name, cache_key, node_id, outputs, created_at, expires_at FROM %s WHERE namespace = $1 AND cache_name = $2 AND cache_key = $3`, q.tableName)
		case sqldb.MySQL:
			query = fmt.Sprintf("SELECT namespace, cache_name, cache_key, node_id, outputs, created_at, expires_at FROM %s WHERE namespace = ? AND cache_name = ? AND cache_key = ?", q.tableName)
		default:
			return fmt.Errorf("unsupported database type: %s", q.dbType)
		}
		rows, err := sess.SQL().QueryContext(ctx, query, namespace, cacheName, cacheKey)
		if err != nil {
			return err
		}
		defer rows.Close()
		if !rows.Next() {
			return rows.Err()
		}
		found = true
		return rows.Scan(&r.Namespace, &r.CacheName, &r.CacheKey, &r.NodeID, &r.Outputs, &r.CreatedAt, &r.ExpiresAt)
	})
	if err != nil || !found {
		return nil, err
	}
	return &r, nil
}

// Prune deletes cache entries whose expires_at has elapsed. It is called
// periodically by the controller to bound the size of the memoization_cache table.
func (q *Queries) Prune(ctx context.Context, sp *sqldb.SessionProxy) (int64, error) {
	now := time.Now().UTC()
	var n int64
	err := sp.With(ctx, func(sess db.Session) error {
		var query string
		switch q.dbType {
		case sqldb.Postgres:
			query = fmt.Sprintf(`DELETE FROM %s WHERE expires_at < $1`, q.tableName)
		case sqldb.MySQL:
			query = fmt.Sprintf("DELETE FROM %s WHERE expires_at < ?", q.tableName)
		default:
			return fmt.Errorf("unsupported database type: %s", q.dbType)
		}
		result, err := sess.SQL().ExecContext(ctx, query, now)
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
	outputsStr := string(outputsJSON)
	now := time.Now().UTC()
	expiresAt := now.Add(time.Duration(maxAgeSeconds) * time.Second)
	return sp.With(ctx, func(sess db.Session) error {
		switch q.dbType {
		case sqldb.Postgres:
			_, err := sess.SQL().ExecContext(ctx,
				fmt.Sprintf(`INSERT INTO %s (namespace, cache_name, cache_key, node_id, outputs, created_at, expires_at)
VALUES ($1, $2, $3, $4, $5, $6, $7)
ON CONFLICT (namespace, cache_name, cache_key) DO UPDATE SET node_id = $4, outputs = $5, expires_at = $7`, q.tableName),
				namespace, cacheName, cacheKey, nodeID, outputsStr, now, expiresAt)
			return err
		case sqldb.MySQL:
			_, err := sess.SQL().ExecContext(ctx,
				fmt.Sprintf("INSERT INTO %s (namespace, cache_name, cache_key, node_id, outputs, created_at, expires_at) VALUES (?, ?, ?, ?, ?, ?, ?) ON DUPLICATE KEY UPDATE node_id = ?, outputs = ?, expires_at = ?", q.tableName),
				namespace, cacheName, cacheKey, nodeID, outputsStr, now, expiresAt, nodeID, outputsStr, expiresAt)
			return err
		default:
			return fmt.Errorf("unsupported database type: %s", q.dbType)
		}
	})
}
