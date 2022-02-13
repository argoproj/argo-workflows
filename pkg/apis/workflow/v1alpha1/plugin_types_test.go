package v1alpha1

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPlugin_UnmarshalJSON(t *testing.T) {
	t.Run("Invalid", func(t *testing.T) {
		p := Plugin{}
		assert.EqualError(t, p.UnmarshalJSON([]byte(`1`)), "json: cannot unmarshal number into Go value of type map[string]interface {}")
	})
	t.Run("NoKeys", func(t *testing.T) {
		p := Plugin{}
		assert.EqualError(t, p.UnmarshalJSON([]byte(`{}`)), "expected exactly one key, got 0")
	})
	t.Run("OneKey", func(t *testing.T) {
		p := Plugin{}
		assert.NoError(t, p.UnmarshalJSON([]byte(`{"foo":1}`)))
	})
}
