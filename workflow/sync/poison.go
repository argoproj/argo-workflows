package sync

import (
	"context"
	"fmt"
	"time"
)

// poisonedLock is a sentinel lock installed into the Manager's syncLockMap when,
// during Initialize, the controller cannot re-establish a holder that a Running
// workflow's status claims to hold.
//
// The soundness invariant is: if a Workflow's status records that it is holding
// a lock, the in-memory lock map must reflect that hold after Initialize.
// Otherwise a racing workflow's TryAcquire would find the lock absent, create a
// fresh one, and acquire a lock that is - per persisted state - already held.
// For a mutex that means two workflows running concurrently under the same
// mutex.
//
// Rather than silently dropping the holder (the previous behaviour), we install
// this lock, which refuses every acquire and reports a poisoned-state message.
// That message surfaces on the waiting node's synchronization status, marking
// the node/workflow as blocked by a poisoned lock so an operator can intervene.
//
// The poison is in-memory only and is cleared on the next controller restart,
// at which point Initialize re-evaluates: if the offending workflow is no longer
// Running the lock is recreated clean; if it is still Running and still
// unresolvable, it is poisoned again.
type poisonedLock struct {
	name   string
	reason string
}

var _ semaphore = &poisonedLock{}

func newPoisonedLock(name, reason string) *poisonedLock {
	return &poisonedLock{name: name, reason: reason}
}

func (p *poisonedLock) message() string {
	return fmt.Sprintf("lock %s is in a poisoned state: %s; manual intervention required", p.name, p.reason)
}

func (p *poisonedLock) acquire(_ context.Context, _ string, _ *transaction) (bool, error) {
	return false, nil
}

// reacquire is a no-op: a poisoned lock refuses all holds until restart. It
// returns nil because the poison already protects the recorded hold; failing
// the holding workflow on top of that would punish it for an unrelated
// holder's poisoning.
func (p *poisonedLock) reacquire(_ context.Context, _ string, _ *transaction) error {
	return nil
}

func (p *poisonedLock) checkAcquire(_ context.Context, _ string, _ *transaction) (bool, bool, string) {
	return false, false, p.message()
}

func (p *poisonedLock) tryAcquire(_ context.Context, _ string, _ *transaction) (bool, string, error) {
	return false, p.message(), nil
}

func (p *poisonedLock) release(_ context.Context, _ string) bool { return false }

func (p *poisonedLock) getName() string { return p.name }

func (p *poisonedLock) addToQueue(_ context.Context, _ string, _ int32, _ time.Time) error {
	return nil
}

func (p *poisonedLock) removeFromQueue(_ context.Context, _ string) error { return nil }

func (p *poisonedLock) getCurrentHolders(_ context.Context) ([]string, error) { return nil, nil }

func (p *poisonedLock) getCurrentPending(_ context.Context) ([]string, error) { return nil, nil }

func (p *poisonedLock) getLimit(_ context.Context) int { return 0 }

func (p *poisonedLock) probeWaiting(_ context.Context) {}

// lock returns true so that tryAcquireImpl proceeds to checkAcquire, which
// returns the poisoned-state message rather than a generic "failed to lock()".
func (p *poisonedLock) lock(_ context.Context) bool { return true }

func (p *poisonedLock) unlock(_ context.Context) {}
