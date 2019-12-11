package v1alpha1

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNodes_FindByDisplayName(t *testing.T) {
	assert.Nil(t, Nodes{}.FindByDisplayName(""))
	assert.NotNil(t, Nodes{"": NodeStatus{DisplayName:"foo"}}.FindByDisplayName("foo"))
}
