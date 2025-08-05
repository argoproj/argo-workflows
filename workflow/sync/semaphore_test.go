package sync

import (
	"testing"
	"time"

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

func TestTryAcquire(t *testing.T) {
	nextWorkflow := func(key string) {
	}

	s := NewSemaphore("foo", 2, nextWorkflow, "semaphore")
	now := time.Now()
	s.addToQueue("default/wf-01", 0, now)
	s.addToQueue("default/wf-02", 0, now.Add(time.Second))
	s.addToQueue("default/wf-03", 0, now.Add(2*time.Second))
	s.addToQueue("default/wf-04", 0, now.Add(3*time.Second))

	// verify only the first in line is allowed to acquired the semaphore
	var acquired bool
	acquired, _ = s.tryAcquire("default/wf-04")
	assert.False(t, acquired)
	acquired, _ = s.tryAcquire("default/wf-03")
	assert.False(t, acquired)
	acquired, _ = s.tryAcquire("default/wf-02")
	assert.False(t, acquired)
	acquired, _ = s.tryAcquire("default/wf-01")
	assert.True(t, acquired)
	// now that wf-01 obtained it, wf-02 can
	acquired, _ = s.tryAcquire("default/wf-02")
	assert.True(t, acquired)
	acquired, _ = s.tryAcquire("default/wf-03")
	assert.False(t, acquired)
	acquired, _ = s.tryAcquire("default/wf-04")
	assert.False(t, acquired)
}

// TestNotifyWaiters ensures we notify the correct waiters after acquiring and releasing a semaphore
func TestNotifyWaitersAcquire(t *testing.T) {
	notified := make(map[string]bool)
	nextWorkflow := func(key string) {
		notified[key] = true
	}

	s := NewSemaphore("foo", 3, nextWorkflow, "semaphore")
	now := time.Now()
	s.addToQueue("default/wf-04", 0, now.Add(3*time.Second))
	s.addToQueue("default/wf-02", 0, now.Add(time.Second))
	s.addToQueue("default/wf-01", 0, now)
	s.addToQueue("default/wf-05", 0, now.Add(4*time.Second))
	s.addToQueue("default/wf-03", 0, now.Add(2*time.Second))

	acquired, _ := s.tryAcquire("default/wf-01")
	assert.True(t, acquired)

	assert.Len(t, notified, 2)
	assert.True(t, notified["default/wf-02"])
	assert.True(t, notified["default/wf-03"])
	assert.False(t, notified["default/wf-04"])
	assert.False(t, notified["default/wf-05"])

	notified = make(map[string]bool)
	released := s.release("default/wf-01")
	assert.True(t, released)

	assert.Len(t, notified, 3)
	assert.True(t, notified["default/wf-02"])
	assert.True(t, notified["default/wf-03"])
	assert.True(t, notified["default/wf-04"])
	assert.False(t, notified["default/wf-05"])
}

// TestNotifyWorkflowFromTemplateSemaphore verifies we enqueue a proper workflow key when using a semaphore template
func TestNotifyWorkflowFromTemplateSemaphore(t *testing.T) {
	notified := make(map[string]bool)
	nextWorkflow := func(key string) {
		notified[key] = true
	}

	s := NewSemaphore("foo", 2, nextWorkflow, "semaphore")
	now := time.Now()
	s.addToQueue("default/wf-01/nodeid-123", 0, now)
	s.addToQueue("default/wf-02/nodeid-456", 0, now.Add(time.Second))

	acquired, _ := s.tryAcquire("default/wf-01/nodeid-123")
	assert.True(t, acquired)

	assert.Len(t, notified, 1)
	assert.True(t, notified["default/wf-02"])
}
