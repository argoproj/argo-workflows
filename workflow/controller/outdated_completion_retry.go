package controller

import (
	"context"
	gosync "sync"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// outdatedCompletionRetries re-enqueues a completion-triggered reconcile that was
// skipped as outdated as soon as the workflow informer catches up, instead of
// after DEFAULT_REQUEUE_TIME. Both maps are sets of workflow keys.
type outdatedCompletionRetries struct {
	mutex gosync.Mutex
	// enqueuedByCompletion holds keys enqueued by a node completion and not yet
	// picked up by a worker.
	enqueuedByCompletion map[string]struct{}
	// skippedAsOutdated holds keys whose completion-triggered reconcile was skipped
	// as outdated and not yet retried.
	skippedAsOutdated map[string]struct{}
}

func (c *outdatedCompletionRetries) markEnqueuedByCompletion(key string) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	if c.enqueuedByCompletion == nil {
		c.enqueuedByCompletion = make(map[string]struct{})
	}
	c.enqueuedByCompletion[key] = struct{}{}
}

// startReconcile is called when a worker picks up a key. It reports whether a
// node completion is waiting on this reconcile, because a completion enqueued
// the key or an earlier reconcile of it was skipped as outdated, and clears both
// records. If this reconcile is skipped as outdated too, the caller marks the
// key again, so a routine reconcile in between does not lose the retry.
func (c *outdatedCompletionRetries) startReconcile(key string) bool {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	_, enqueued := c.enqueuedByCompletion[key]
	_, skipped := c.skippedAsOutdated[key]
	delete(c.enqueuedByCompletion, key)
	delete(c.skippedAsOutdated, key)
	return enqueued || skipped
}

func (c *outdatedCompletionRetries) markSkippedAsOutdated(key string) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	if c.skippedAsOutdated == nil {
		c.skippedAsOutdated = make(map[string]struct{})
	}
	c.skippedAsOutdated[key] = struct{}{}
}

// clearSkippedAsOutdated unmarks a key and reports whether it was marked, so
// that only one caller re-enqueues it.
func (c *outdatedCompletionRetries) clearSkippedAsOutdated(key string) bool {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	_, marked := c.skippedAsOutdated[key]
	delete(c.skippedAsOutdated, key)
	return marked
}

// forget drops a key whose workflow was deleted.
func (c *outdatedCompletionRetries) forget(key string) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	delete(c.enqueuedByCompletion, key)
	delete(c.skippedAsOutdated, key)
}

// skipOutdatedCompletion marks a key as skipped as outdated, then retries it in
// case the informer update arrived while the worker was checking.
func (wfc *WorkflowController) skipOutdatedCompletion(ctx context.Context, key string) {
	wfc.outdatedCompletionRetries.markSkippedAsOutdated(key)
	obj, ok := wfc.getWorkflowByKey(ctx, key)
	if !ok {
		return
	}
	if wf, ok := obj.(metav1.Object); ok {
		wfc.retryOutdatedCompletion(ctx, key, wf)
	}
}

// retryOutdatedCompletion re-enqueues a key after minCompletionDelay if the
// key was skipped as outdated and wf is no longer outdated. Returns true if it
// re-enqueued the key.
func (wfc *WorkflowController) retryOutdatedCompletion(ctx context.Context, key string, wf metav1.Object) bool {
	if wfc.minCompletionRequeueDelay == nil {
		return false
	}
	if outdated, _ := wfc.isOutdated(ctx, wf); outdated {
		return false
	}
	if !wfc.outdatedCompletionRetries.clearSkippedAsOutdated(key) {
		return false
	}
	// Mark it again so that a further lag is caught up the same way.
	wfc.outdatedCompletionRetries.markEnqueuedByCompletion(key)
	wfc.wfQueue.AddAfter(key, wfc.minCompletionDelay())
	return true
}

// enqueueWorkflowUpdate enqueues a workflow the informer has just updated,
// retrying a skipped completion if there is one and rate-limited otherwise.
func (wfc *WorkflowController) enqueueWorkflowUpdate(ctx context.Context, key string, wf metav1.Object) {
	if !wfc.retryOutdatedCompletion(ctx, key, wf) {
		wfc.wfQueue.AddRateLimited(key)
	}
}
