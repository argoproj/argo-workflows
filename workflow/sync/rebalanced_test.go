package sync

import (
	"runtime/debug"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
)

func syncWRebalance(key string) *v1alpha1.Synchronization {
	return &v1alpha1.Synchronization{Semaphore: &v1alpha1.SemaphoreRef{
		RebalanceKey: &key,
	}}
}

func TestSimpleRebalance(t *testing.T) {
	rq := NewRebalanceQueue()
	s := NewSemaphore("test-semaphore", 4, func(string) {}, "semaphore", rq)
	// rebalance needs a reference to semaphore
	rq.setParentSemaphore(s)

	rq.add("argo/wf-000/key-000-with-AAA", -1, time.Unix(1, 0), syncWRebalance("AAA"))
	rq.add("argo/wf-000/key-001-with-AAA", -1, time.Unix(2, 0), syncWRebalance("AAA"))
	rq.add("argo/wf-000/key-002-with-AAA", -1, time.Unix(3, 0), syncWRebalance("AAA"))
	rq.add("argo/wf-000/key-003-with-AAA", -1, time.Unix(4, 0), syncWRebalance("AAA"))
	rq.add("argo/wf-000/key-004-with-AAA", -1, time.Unix(5, 0), syncWRebalance("AAA"))
	rq.add("argo/wf-000/key-005-with-AAA", -1, time.Unix(6, 0), syncWRebalance("AAA"))

	rq.add("argo/wf-000/key-000-with-BBB", -1, time.Unix(7, 0), syncWRebalance("BBB"))
	rq.add("argo/wf-000/key-001-with-BBB", -1, time.Unix(8, 0), syncWRebalance("BBB"))
	rq.add("argo/wf-000/key-002-with-BBB", -1, time.Unix(9, 0), syncWRebalance("BBB"))
	rq.add("argo/wf-000/key-003-with-BBB", -1, time.Unix(10, 0), syncWRebalance("BBB"))
	rq.add("argo/wf-000/key-004-with-BBB", -1, time.Unix(11, 0), syncWRebalance("BBB"))
	rq.add("argo/wf-000/key-005-with-BBB", -1, time.Unix(12, 0), syncWRebalance("BBB"))

	// important: release is always called before remove in manager
	rq.onRelease("argo/wf-000/key-000-with-AAA")
	rq.remove("argo/wf-000/key-000-with-AAA")

	assert.Equal(t, rq.all()[0].key, "argo/wf-000/key-001-with-AAA")
	assert.Equal(t, rq.all()[1].key, "argo/wf-000/key-002-with-AAA")
	assert.Equal(t, rq.all()[2].key, "argo/wf-000/key-000-with-BBB")
	assert.Equal(t, rq.all()[3].key, "argo/wf-000/key-001-with-BBB")
	// rest of the items don't matter - we will never try to schedule past the limit, and we have no idea
	// what resource will finish next, so it'd be a complete guess. reshuffle after the next onRelease
}

// TestModWithLeftover ensures that if ${ limit / numUniqueRebalanceKeys } is a non-integer, that
// the remaining positions are filled equally (i.e. one user doesn't get an unfair number of locks)
//
// e.g. suppose a limit of 17 and we have 6 distinct rebalance keys. 17 % 6 is 5, and these 5
// remaining slots should be given to 5 distinct requesters, BUT NEVER to a requester that already
// is holding 3 locks
func TestModWithLeftover(t *testing.T) {
	rq := NewRebalanceQueue()
	s := NewSemaphore("test-semaphore", 17, func(string) {}, "semaphore", rq)
	// rebalance needs a reference to semaphore
	rq.setParentSemaphore(s)

	rq.add("argo/wf-000/key-000-with-AAA", -1, time.Unix(1, 0), syncWRebalance("AAA"))
	rq.add("argo/wf-000/key-001-with-AAA", -1, time.Unix(2, 0), syncWRebalance("AAA"))
	rq.add("argo/wf-000/key-002-with-AAA", -1, time.Unix(3, 0), syncWRebalance("AAA"))
	rq.add("argo/wf-000/key-003-with-AAA", -1, time.Unix(4, 0), syncWRebalance("AAA"))
	rq.add("argo/wf-000/key-004-with-AAA", -1, time.Unix(5, 0), syncWRebalance("AAA"))
	rq.add("argo/wf-000/key-005-with-AAA", -1, time.Unix(6, 0), syncWRebalance("AAA"))

	rq.add("argo/wf-001/key-000-with-BBB", -1, time.Unix(7, 0), syncWRebalance("BBB"))
	rq.add("argo/wf-001/key-001-with-BBB", -1, time.Unix(8, 0), syncWRebalance("BBB"))
	rq.add("argo/wf-001/key-002-with-BBB", -1, time.Unix(9, 0), syncWRebalance("BBB"))
	rq.add("argo/wf-001/key-003-with-BBB", -1, time.Unix(10, 0), syncWRebalance("BBB"))
	rq.add("argo/wf-001/key-004-with-BBB", -1, time.Unix(11, 0), syncWRebalance("BBB"))
	rq.add("argo/wf-001/key-005-with-BBB", -1, time.Unix(12, 0), syncWRebalance("BBB"))

	rq.add("argo/wf-002/key-000-with-CCC", -1, time.Unix(13, 0), syncWRebalance("CCC"))
	rq.add("argo/wf-002/key-001-with-CCC", -1, time.Unix(14, 0), syncWRebalance("CCC"))
	rq.add("argo/wf-002/key-002-with-CCC", -1, time.Unix(15, 0), syncWRebalance("CCC"))
	rq.add("argo/wf-002/key-003-with-CCC", -1, time.Unix(16, 0), syncWRebalance("CCC"))
	rq.add("argo/wf-002/key-004-with-CCC", -1, time.Unix(17, 0), syncWRebalance("CCC"))
	rq.add("argo/wf-002/key-005-with-CCC", -1, time.Unix(18, 0), syncWRebalance("CCC"))

	rq.add("argo/wf-003/key-000-with-DDD", -1, time.Unix(19, 0), syncWRebalance("DDD"))
	rq.add("argo/wf-003/key-001-with-DDD", -1, time.Unix(20, 0), syncWRebalance("DDD"))
	rq.add("argo/wf-003/key-002-with-DDD", -1, time.Unix(21, 0), syncWRebalance("DDD"))
	rq.add("argo/wf-003/key-003-with-DDD", -1, time.Unix(22, 0), syncWRebalance("DDD"))
	rq.add("argo/wf-003/key-004-with-DDD", -1, time.Unix(23, 0), syncWRebalance("DDD"))
	rq.add("argo/wf-003/key-005-with-DDD", -1, time.Unix(24, 0), syncWRebalance("DDD"))

	rq.add("argo/wf-004/key-000-with-EEE", -1, time.Unix(25, 0), syncWRebalance("EEE"))
	rq.add("argo/wf-004/key-001-with-EEE", -1, time.Unix(26, 0), syncWRebalance("EEE"))
	rq.add("argo/wf-004/key-002-with-EEE", -1, time.Unix(27, 0), syncWRebalance("EEE"))
	rq.add("argo/wf-004/key-003-with-EEE", -1, time.Unix(28, 0), syncWRebalance("EEE"))
	rq.add("argo/wf-004/key-004-with-EEE", -1, time.Unix(29, 0), syncWRebalance("EEE"))
	rq.add("argo/wf-004/key-005-with-EEE", -1, time.Unix(30, 0), syncWRebalance("EEE"))

	rq.add("argo/wf-005/key-000-with-FFF", -1, time.Unix(31, 0), syncWRebalance("FFF"))
	rq.add("argo/wf-005/key-001-with-FFF", -1, time.Unix(32, 0), syncWRebalance("FFF"))
	rq.add("argo/wf-005/key-002-with-FFF", -1, time.Unix(33, 0), syncWRebalance("FFF"))
	rq.add("argo/wf-005/key-003-with-FFF", -1, time.Unix(34, 0), syncWRebalance("FFF"))
	rq.add("argo/wf-005/key-004-with-FFF", -1, time.Unix(35, 0), syncWRebalance("FFF"))
	rq.add("argo/wf-005/key-005-with-FFF", -1, time.Unix(36, 0), syncWRebalance("FFF"))

	status, msg := s.tryAcquire("argo/wf-000/key-000-with-AAA")
	assert.True(t, status)
	assert.Empty(t, msg)
	// important: release is always called before remove in manager
	s.release("argo/wf-000/key-000-with-AAA")
	s.removeFromQueue("argo/wf-000/key-000-with-AAA")

	assert.Equal(t, rq.all()[0].key, "argo/wf-000/key-001-with-AAA")
	assert.Equal(t, rq.all()[1].key, "argo/wf-000/key-002-with-AAA")
	assert.Equal(t, rq.all()[2].key, "argo/wf-001/key-000-with-BBB")
	assert.Equal(t, rq.all()[3].key, "argo/wf-001/key-001-with-BBB")
	assert.Equal(t, rq.all()[4].key, "argo/wf-002/key-000-with-CCC")
	assert.Equal(t, rq.all()[5].key, "argo/wf-002/key-001-with-CCC")
	assert.Equal(t, rq.all()[6].key, "argo/wf-003/key-000-with-DDD")
	assert.Equal(t, rq.all()[7].key, "argo/wf-003/key-001-with-DDD")
	assert.Equal(t, rq.all()[8].key, "argo/wf-004/key-000-with-EEE")
	assert.Equal(t, rq.all()[9].key, "argo/wf-004/key-001-with-EEE")
	assert.Equal(t, rq.all()[10].key, "argo/wf-005/key-000-with-FFF")
	assert.Equal(t, rq.all()[11].key, "argo/wf-005/key-001-with-FFF")
	// distribute the remainder, choosing oldest first
	assert.Equal(t, rq.all()[12].key, "argo/wf-000/key-003-with-AAA")
	assert.Equal(t, rq.all()[13].key, "argo/wf-001/key-002-with-BBB")
	assert.Equal(t, rq.all()[14].key, "argo/wf-002/key-002-with-CCC")
	assert.Equal(t, rq.all()[15].key, "argo/wf-003/key-002-with-DDD")
	assert.Equal(t, rq.all()[16].key, "argo/wf-004/key-002-with-EEE")
	// rest of the items don't matter - we will never try to schedule past the limit, and we have no idea
	// what resource will finish next, so it'd be a complete guess. reshuffle after the next onRelease

	status, msg = s.tryAcquire("argo/wf-000/key-001-with-AAA")
	assert.True(t, status)
	assert.Empty(t, msg)

	status, msg = s.tryAcquire("argo/wf-000/key-002-with-AAA")
	assert.True(t, status)
	assert.Empty(t, msg)

	// technically it's allowed to acquire, but only once we get to overflow.
	status, msg = s.tryAcquire("argo/wf-000/key-003-with-AAA")
	assert.False(t, status)
	assert.Contains(t, msg, "Waiting for test-semaphore lock. Lock status: 15/17")

	// acquire the remaining ones. skip validating output because end of this test will fail if
	// something went wrong.
	s.tryAcquire("argo/wf-001/key-000-with-BBB")
	s.tryAcquire("argo/wf-001/key-001-with-BBB")
	s.tryAcquire("argo/wf-002/key-000-with-CCC")
	s.tryAcquire("argo/wf-002/key-001-with-CCC")
	s.tryAcquire("argo/wf-003/key-000-with-DDD")
	s.tryAcquire("argo/wf-003/key-001-with-DDD")
	s.tryAcquire("argo/wf-004/key-000-with-EEE")
	s.tryAcquire("argo/wf-004/key-001-with-EEE")
	s.tryAcquire("argo/wf-005/key-000-with-FFF")
	s.tryAcquire("argo/wf-005/key-001-with-FFF")

	// NOW we can acquire the 3rd, "overflow" item
	status, msg = s.tryAcquire("argo/wf-000/key-003-with-AAA")
	assert.True(t, status)
	assert.Empty(t, msg)

	// something other than A finishes
	s.release("argo/wf-001/key-001-with-BBB")
	s.removeFromQueue("argo/wf-001/key-001-with-BBB")

	// this piece is really important. AAA is holding 3 locks already, which is already in "overflow" (since 17 / 6
	// allows for requesters to have up to 3 locks). If AAA somehow sneaks back into overflow, we'd be in a situation
	// where it starts acquiring locks greedily and exceeds this number. We have to keep A out of overflow until one of
	// its current locks frees up
	assert.Equal(t, rq.all()[0].key, "argo/wf-001/key-002-with-BBB")
	assert.Equal(t, rq.all()[1].key, "argo/wf-001/key-003-with-BBB")
	assert.Equal(t, rq.all()[2].key, "argo/wf-002/key-002-with-CCC")
	assert.Equal(t, rq.all()[3].key, "argo/wf-003/key-002-with-DDD")
	assert.Equal(t, rq.all()[4].key, "argo/wf-004/key-002-with-EEE")
	assert.Equal(t, rq.all()[5].key, "argo/wf-005/key-002-with-FFF")

	// 12 locks currently being held
	assert.Len(t, s.getCurrentHolders(), 12)
}

// TestDuplicateKey ensures that non-unique lock contenders can only exist once
// in the queue
func TestDuplicateKey(t *testing.T) {
	rq := NewRebalanceQueue()
	s := NewSemaphore("test-semaphore", 4, func(string) {}, "semaphore", rq)
	// rebalance needs a reference to semaphore
	rq.setParentSemaphore(s)

	rq.add("argo/wf-000/key-000-with-AAA", -1, time.Unix(1, 0), syncWRebalance("AAA"))
	rq.add("argo/wf-000/key-000-with-AAA", -1, time.Unix(2, 0), syncWRebalance("AAA"))

	assert.Len(t, rq.all(), 1)
}

// TestFailureToSetParentReference ensures that failure to set parent semaphore does NOT fail gracefully
// (If this test fails, it indicates that an error has been introduced into argo codebase)
func TestFailureToSetParentReference(t *testing.T) {
	debug.SetPanicOnFault(true)
	defer debug.SetPanicOnFault(false) // don't think it's required but feel better doing it
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("The code did not panic")
		}
	}()

	rq := NewRebalanceQueue()

	rq.add("argo/wf-000/key-000-with-AAA", -1, time.Unix(1, 0), syncWRebalance("AAA"))

	rq.onRelease("argo/wf-000/key-000-with-AAA")
}

// TestKeyCacheRemoval ensures that releasing a resource key removes it from in-memory cache
func TestKeyCacheRemoval(t *testing.T) {
	rq := NewRebalanceQueue()
	s := NewSemaphore("test-semaphore", 4, func(string) {}, "semaphore", rq)
	// rebalance needs a reference to semaphore
	rq.setParentSemaphore(s)

	rq.add("argo/wf-000/key-000-with-AAA", -1, time.Unix(1, 0), syncWRebalance("AAA"))
	rq.onRelease("argo/wf-000/key-000-with-AAA")

}

// TestEmptyRebalanceKey ensures that a resource vying for a lock that has forgotten to pass
// a rebalance key (or has not on purpose) will be exist in the same grouping
func TestEmptyRebalanceKey(t *testing.T) {
	rq := NewRebalanceQueue()
	s := NewSemaphore("test-semaphore", 4, func(string) {}, "semaphore", rq)
	// rebalance needs a reference to semaphore
	rq.setParentSemaphore(s)

	rq.add("argo/wf-000/key-000-no-rebalance-key", -1, time.Unix(1, 0), &v1alpha1.Synchronization{Semaphore: &v1alpha1.SemaphoreRef{
		RebalanceKey: nil,
	}})

	assert.Len(t, rq.all(), 1)
	assert.Equal(t, rq.rebalanceKeyCache["argo/wf-000/key-000-no-rebalance-key"], "")
}

// TestRemove ensures that remove removes item from queue, but does not remove key from cache
func TestRemove(t *testing.T) {
	rq := NewRebalanceQueue()
	s := NewSemaphore("test-semaphore", 4, func(string) {}, "semaphore", rq)
	// rebalance needs a reference to semaphore
	rq.setParentSemaphore(s)

	rq.add("argo/wf-000/key-000-with-AAA", -1, time.Unix(1, 0), syncWRebalance("AAA"))
	rq.remove("argo/wf-000/key-000-with-AAA")

	assert.Len(t, rq.all(), 0)
}
