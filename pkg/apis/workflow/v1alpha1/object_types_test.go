package v1alpha1

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestObject_Get(t *testing.T) {
	t.Run("Nil", func(t *testing.T) {
		var o *Object
		v, err := o.Get("")
		assert.NoError(t, err)
		assert.Nil(t, v)
	})
	t.Run("Missing", func(t *testing.T) {
		var o = Object{Value: json.RawMessage("{}")}
		v, err := o.Get("")
		assert.NoError(t, err)
		assert.Nil(t, v)
	})
	t.Run("Found", func(t *testing.T) {
		var o = Object{Value: json.RawMessage(`{"hello": 1}`)}
		v, err := o.Get("hello")
		assert.NoError(t, err)
		assert.Equal(t, float64(1), v)
	})
}
