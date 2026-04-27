package db

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	upperdb "github.com/upper/db/v4"
)

func TestCacheRecordCond(t *testing.T) {
	record := &CacheRecord{
		Namespace: "my-ns",
		CacheName: "my-cache",
		CacheKey:  "my-key",
		NodeID:    "ignored",
	}

	assert.Equal(t, upperdb.Cond{
		colNamespace: "my-ns",
		colCacheName: "my-cache",
		colCacheKey:  "my-key",
	}, cacheRecordCond(record))
}

func TestCacheRecordUpdates(t *testing.T) {
	now := time.Unix(100, 0).UTC()
	expiresAt := time.Unix(200, 0).UTC()
	record := &CacheRecord{
		NodeID:    "node-1",
		Outputs:   `{"result":"ok"}`,
		CreatedAt: now,
		ExpiresAt: expiresAt,
	}

	assert.Equal(t, map[string]any{
		"node_id":    "node-1",
		"outputs":    `{"result":"ok"}`,
		"created_at": now,
		"expires_at": expiresAt,
	}, cacheRecordUpdates(record))
}

func TestIsDuplicateKeyError(t *testing.T) {
	assert.True(t, isDuplicateKeyError(errors.New("pq: duplicate key value violates unique constraint")))
	assert.True(t, isDuplicateKeyError(errors.New("Error 1062: Duplicate entry 'x' for key 'PRIMARY'")))
	assert.False(t, isDuplicateKeyError(errors.New("some other error")))
	assert.False(t, isDuplicateKeyError(nil))
}
