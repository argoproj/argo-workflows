package v1alpha1

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestInt64OrString(t *testing.T) {
	t.Run("String", func(t *testing.T) {
		n := Int64OrStringPtr("my-string")
		data, err := n.MarshalJSON()
		if assert.NoError(t, err) {
			assert.Equal(t, `"my-string"`, string(data), "string value has quotes")
		}
		i := Int64OrStringPtr("")
		err = i.UnmarshalJSON([]byte(`"my-string"`))
		if assert.NoError(t, err) {
			assert.Equal(t, Int64OrStringPtr("my-string"), i)
		}
		assert.Equal(t, "my-string", i.String(), "string value does not have quotes")
	})
	t.Run("LargeNumber", func(t *testing.T) {
		n := ParseInt64OrString(881217801864)
		data, err := n.MarshalJSON()
		if assert.NoError(t, err) {
			assert.Equal(t, "881217801864", string(data))
		}
		i := Int64OrStringPtr("")
		err = i.UnmarshalJSON([]byte("881217801864"))
		if assert.NoError(t, err) {
			assert.Equal(t, Int64OrStringPtr("881217801864"), i)
		}
		assert.Equal(t, "881217801864", i.String())
	})
}
