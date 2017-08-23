package cache

import (
	"testing"
	"time"
)

var newInMemoryStore = func(_ *testing.T, defaultExpiration time.Duration) CacheStore {
	return NewInMemoryStore(defaultExpiration)
}

// Test typical cache interactions
func TestInMemoryCache_TypicalGetSet(t *testing.T) {
	typicalGetSet(t, newInMemoryStore)
}

func TestInMemoryCache_IncrDecr(t *testing.T) {
	incrDecr(t, newInMemoryStore)
}

func TestInMemoryCache_Expiration(t *testing.T) {
	expiration(t, newInMemoryStore)
}

func TestInMemoryCache_EmptyCache(t *testing.T) {
	emptyCache(t, newInMemoryStore)
}

func TestInMemoryCache_Replace(t *testing.T) {
	testReplace(t, newInMemoryStore)
}

func TestInMemoryCache_Add(t *testing.T) {
	testAdd(t, newInMemoryStore)
}
