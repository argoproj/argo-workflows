package cache

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/client-go/tools/cache"
)

func Test_store(t *testing.T) {
	s := newStore()
	t.Run("Add", func(t *testing.T) {
		s.Add(cache.ExplicitKey("my-ns/my-name"))
		obj, exists, err := s.GetByKey("my-ns/my-name")
		assert.NoError(t, err)
		assert.True(t, exists)
		assert.Equal(t, cache.ExplicitKey("my-ns/my-name"), obj)
	})
	t.Run("GetKeyKey", func(t *testing.T) {
		_, exists, err := s.GetByKey("my-ns/not-found")
		assert.NoError(t, err)
		assert.False(t, exists)
		obj, exists, err := s.GetByKey("my-ns/my-name")
		assert.NoError(t, err)
		assert.True(t, exists)
		assert.Equal(t, cache.ExplicitKey("my-ns/my-name"), obj)
	})
	t.Run("ListKeys", func(t *testing.T) {
		keys := s.ListKeys()
		if assert.Len(t, keys, 1) {
			assert.Equal(t, "my-ns/my-name", keys[0])
		}
	})
	t.Run("Delete", func(t *testing.T) {
		s.Delete(cache.ExplicitKey("my-ns/my-name"))
		keys := s.ListKeys()
		assert.Len(t, keys, 0)
	})
}
