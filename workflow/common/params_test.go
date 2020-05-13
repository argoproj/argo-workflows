package common

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestParamsMerge ensures Merge of Parameters works correctly.
func TestParamsMerge(t *testing.T) {
	params := Parameters{"foo": "1"}
	newParams := params.Merge(Parameters{"foo": "2", "bar": "1"}, Parameters{"wow": "1"})
	assert.Equal(t, Parameters{"foo": "2", "bar": "1", "wow": "1"}, newParams)
	assert.NotSame(t, &params, &newParams)
}

// TestParamsClone ensures Clone of Parameters works correctly.
func TestParamsClone(t *testing.T) {
	params := Parameters{"foo": "1"}
	newParams := params.DeepCopy()
	assert.Equal(t, params, newParams)
	assert.NotSame(t, &params, &newParams)
}
