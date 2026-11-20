package db

import (
	"testing"

	"github.com/stretchr/testify/assert"
	upperdb "github.com/upper/db/v4"
)

func TestCacheRecordCond(t *testing.T) {
	record := &CacheRecord{Namespace: "my-ns", CacheName: "my-cache", CacheKey: "my-key", NodeID: "ignored"}
	assert.Equal(t, upperdb.Cond{colNamespace: "my-ns", colCacheName: "my-cache", colCacheKey: "my-key"}, cacheRecordCond(record))
}
