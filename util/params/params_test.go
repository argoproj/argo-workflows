package params

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestParamsMerge ensures Merge of Params works correctly.
func TestParamsMerge(t *testing.T) {
	params := Params{"foo": "1"}
	newParams := params.Merge(Params{"foo": "2", "bar": "1"}, Params{"wow": "1"})
	assert.Equal(t, Params{"foo": "2", "bar": "1", "wow": "1"}, newParams)
	assert.NotSame(t, &params, &newParams)
}

// TestParamsClone ensures Clone of Params works correctly.
func TestParamsClone(t *testing.T) {
	params := Params{"foo": "1"}
	newParams := params.DeepCopy()
	assert.Equal(t, params, newParams)
	assert.NotSame(t, &params, &newParams)
}
