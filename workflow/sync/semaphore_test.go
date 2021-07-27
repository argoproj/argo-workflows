package sync

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsSameWorkflowNodeKeys(t *testing.T) {
	wfkey1 := "default/wf-1"
	wfkey2 := "default/wf-2"
	nodeWf1key1 := "default/wf-1/node-1"
	nodeWf1key2 := "default/wf-1/node-2"
	nodeWf2key1 := "default/wf-2/node-1"
	nodeWf2key2 := "default/wf-2/node-2"
	assert.True(t, isSameWorkflowNodeKeys(nodeWf1key1, nodeWf1key2))
	assert.True(t, isSameWorkflowNodeKeys(wfkey1, wfkey1))
	assert.False(t, isSameWorkflowNodeKeys(nodeWf1key1, nodeWf2key1))
	assert.False(t, isSameWorkflowNodeKeys(wfkey1, wfkey2))
	assert.True(t, isSameWorkflowNodeKeys(nodeWf2key1, nodeWf2key2))
}
