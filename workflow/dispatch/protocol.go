package dispatch

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"slices"
	"sync"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"

	"github.com/argoproj/argo-workflows/v4/pkg/apis/workflow"
	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/util/logging"
	"github.com/argoproj/argo-workflows/v4/workflow/common"
)

// errAlreadyExists is a store's Create finding the name taken.
var errAlreadyExists = errors.New("already exists")

// store is what the protocol runs over: each method is one API call. Errors
// are ErrNotFound, errAlreadyExists, ErrStale (a delete's UID precondition or
// a patch's tests failed) or anything else, which is worth retrying.
type store interface {
	createTaskSet(ctx context.Context, taskSet *wfv1.WorkflowTaskSet) (*wfv1.WorkflowTaskSet, error)
	getTaskSet(ctx context.Context, namespace, name string) (*wfv1.WorkflowTaskSet, error)
	deleteTaskSet(ctx context.Context, namespace, name string, uid types.UID) error
	createResult(ctx context.Context, result *wfv1.WorkflowTaskResult) (*wfv1.WorkflowTaskResult, error)
	getResult(ctx context.Context, namespace, name string) (*wfv1.WorkflowTaskResult, error)
	listResults(ctx context.Context, namespace string, selector labels.Selector) ([]wfv1.WorkflowTaskResult, error)
	patchResult(ctx context.Context, namespace, name string, patch resultPatch) error
	deleteResult(ctx context.Context, namespace, name string, uid types.UID) error
}

// resultPatch is one write to a result. It applies only while the result has
// uid and, when workerPodUID is set, that worker-pod-uid label; it then sets
// the labels and whichever fields are non-nil.
type resultPatch struct {
	uid          types.UID
	workerPodUID types.UID
	labels       map[string]string
	phase        *wfv1.NodePhase
	message      *string
	outputs      *wfv1.Outputs
	progress     *wfv1.Progress
}

// owns says whether the patch's tests pass on result.
func (p resultPatch) owns(result *wfv1.WorkflowTaskResult) bool {
	return result.UID == p.uid &&
		(p.workerPodUID == "" || result.Labels[common.LabelKeyWorkerPodUID] == string(p.workerPodUID))
}

func (p resultPatch) apply(result *wfv1.WorkflowTaskResult) {
	if result.Labels == nil {
		result.Labels = map[string]string{}
	}
	maps.Copy(result.Labels, p.labels)
	if p.phase != nil {
		result.Phase = *p.phase
	}
	if p.message != nil {
		result.Message = *p.message
	}
	if p.outputs != nil {
		result.Outputs = p.outputs.DeepCopy()
	}
	if p.progress != nil {
		result.Progress = *p.progress
	}
}

// completion is the patch that finalises a claim with result.
func completion(claim Claim, result wfv1.NodeResult) resultPatch {
	p := resultPatch{
		uid:          claim.UID,
		workerPodUID: claim.WorkerPodUID,
		labels:       map[string]string{common.LabelKeyReportOutputsCompleted: "true"},
		phase:        &result.Phase,
		message:      &result.Message,
		outputs:      result.Outputs,
	}
	if result.Progress != "" {
		p.progress = &result.Progress
	}
	return p
}

// gone says whether a store error means the object meant no longer exists:
// it was deleted or replaced.
func gone(err error) bool {
	return errors.Is(err, ErrNotFound) || errors.Is(err, ErrStale)
}

type dispatcher struct {
	store store
}

func newTaskSet(task Task) *wfv1.WorkflowTaskSet {
	labels := PoolLabels(&task.Pool, task.PoolHash)
	labels[common.LabelKeyWorkflow] = task.WorkflowName
	if task.OnExit {
		labels[common.LabelKeyOnExit] = "true"
	}
	return &wfv1.WorkflowTaskSet{
		TypeMeta: metav1.TypeMeta{
			APIVersion: workflow.APIVersion,
			Kind:       workflow.WorkflowTaskSetKind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace:       task.Namespace,
			Name:            task.TaskSetName(),
			Labels:          labels,
			Annotations:     map[string]string{common.AnnotationKeyPoolControllerVersion: task.Pool.ControllerVersion},
			OwnerReferences: []metav1.OwnerReference{common.WorkflowOwnerReference(task.WorkflowName, task.WorkflowUID)},
		},
		Spec: wfv1.WorkflowTaskSetSpec{
			Tasks:    map[string]wfv1.Template{task.NodeID: task.Template},
			Pool:     task.Pool.DeepCopy(),
			Dispatch: task.Dispatch.DeepCopy(),
		},
	}
}

func (d *dispatcher) Enqueue(ctx context.Context, task Task) (Enqueued, error) {
	enqueued, err := d.enqueue(ctx, task)
	if err != nil {
		err = fmt.Errorf("enqueueing WorkflowTaskSet %s/%s: %w", task.Namespace, task.TaskSetName(), err)
	}
	return enqueued, err
}

func (d *dispatcher) enqueue(ctx context.Context, task Task) (Enqueued, error) {
	name := task.TaskSetName()
	taskSet, err := d.store.createTaskSet(ctx, newTaskSet(task))
	adopted := errors.Is(err, errAlreadyExists)
	if adopted {
		taskSet, err = d.store.getTaskSet(ctx, task.Namespace, name)
	}
	if err != nil {
		return Enqueued{}, err
	}
	if _, ok := TaskSetRefFromMeta(taskSet); !ok {
		return Enqueued{}, ErrNameTaken
	}
	if adopted && taskSet.DeletionTimestamp != nil {
		// Deleted, as when the controller replaced it, and held by a
		// finalizer: no worker claims it, so it can't be adopted.
		return Enqueued{}, ErrStaleTaskSet
	}
	if adopted && !OwnedByWorkflow(taskSet, task.WorkflowUID) {
		// Only one workflow holds a name at a time, so a pooled TaskSet that
		// another workflow UID owns, or none does after an orphan deletion,
		// was left by an earlier workflow of the same name. Adopting it
		// would let that workflow's claim through the fence; replace it.
		logging.RequireLoggerFromContext(ctx).
			WithFields(logging.Fields{"namespace": task.Namespace, "taskSet": name, "uid": taskSet.UID}).
			Info(ctx, "Replacing a WorkflowTaskSet left by an earlier workflow")
		if err = d.removeTaskSet(ctx, task.Key, taskSet.UID); err != nil {
			return Enqueued{}, err
		}
		taskSet, err = d.store.createTaskSet(ctx, newTaskSet(task))
		if errors.Is(err, errAlreadyExists) {
			return Enqueued{}, ErrStaleTaskSet
		}
		if err != nil {
			return Enqueued{}, err
		}
		adopted = false
	}
	enqueued := Enqueued{TaskSet: taskSet, Adopted: adopted}
	if taskSet.Spec.Pool == nil || taskSet.Spec.Dispatch == nil {
		return enqueued, ErrFieldsDropped
	}
	if adopted {
		logging.RequireLoggerFromContext(ctx).
			WithFields(logging.Fields{"namespace": task.Namespace, "taskSet": name, "uid": taskSet.UID}).
			Info(ctx, "Adopted existing WorkflowTaskSet")
	}
	return enqueued, nil
}

// removeTaskSet deletes the TaskSet with uid. Like a successful delete, a
// TaskSet already gone or replaced means it no longer exists.
func (d *dispatcher) removeTaskSet(ctx context.Context, key Key, uid types.UID) error {
	err := d.store.deleteTaskSet(ctx, key.Namespace, key.TaskSetName(), uid)
	if err == nil || gone(err) {
		return nil
	}
	return fmt.Errorf("deleting WorkflowTaskSet %s/%s: %w", key.Namespace, key.TaskSetName(), err)
}

func (d *dispatcher) Drop(ctx context.Context, key Key, taskSetUID types.UID) (*Claim, error) {
	if err := d.removeTaskSet(ctx, key, taskSetUID); err != nil {
		return nil, err
	}
	result, err := d.store.getResult(ctx, key.Namespace, key.ResultName())
	if errors.Is(err, ErrNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("getting WorkflowTaskResult %s/%s: %w", key.Namespace, key.ResultName(), err)
	}
	claim, ok := ClaimFromResult(result)
	if !ok || claim.TaskSetUID != taskSetUID {
		// Not a claim on this TaskSet.
		return nil, nil
	}
	return &claim, nil
}

func (d *dispatcher) Interrupt(ctx context.Context, key Key, taskSetUID, claimUID types.UID) error {
	if err := d.removeTaskSet(ctx, key, taskSetUID); err != nil {
		return err
	}
	err := d.store.patchResult(ctx, key.Namespace, key.ResultName(), resultPatch{
		uid:    claimUID,
		labels: map[string]string{common.LabelKeyInterrupt: "true"},
	})
	if gone(err) {
		return fmt.Errorf("interrupting WorkflowTaskResult %s/%s: %w: %w", key.Namespace, key.ResultName(), ErrClaimLost, err)
	}
	if err != nil {
		return fmt.Errorf("interrupting WorkflowTaskResult %s/%s: %w", key.Namespace, key.ResultName(), err)
	}
	return nil
}

func (d *dispatcher) Shutdown(ctx context.Context, tasks []LedgerEntry) []ShutdownOutcome {
	outcomes := make([]ShutdownOutcome, 0, len(tasks))
	for _, task := range tasks {
		outcome := ShutdownOutcome{Key: task.Key}
		claimUID := task.Ledger.ClaimUID
		switch state := task.Ledger.State(); state {
		case wfv1.PooledTaskEnqueued:
			found, err := d.Drop(ctx, task.Key, task.Ledger.TaskSetUID)
			if err != nil || found == nil {
				outcome.Finished, outcome.Err = err == nil, err
				outcomes = append(outcomes, outcome)
				continue
			}
			outcome.Found, claimUID = found, found.UID
		case wfv1.PooledTaskClaimed, wfv1.PooledTaskInterrupted:
		default:
			outcome.Err = fmt.Errorf("shutting down %s/%s: unknown pooled task state %q", task.Namespace, task.NodeID, state)
			outcomes = append(outcomes, outcome)
			continue
		}
		err := d.Interrupt(ctx, task.Key, task.Ledger.TaskSetUID, claimUID)
		switch {
		case errors.Is(err, ErrClaimLost):
			outcome.Finished = true
		case err != nil:
			outcome.Err = err
		}
		outcomes = append(outcomes, outcome)
	}
	return outcomes
}

func (d *dispatcher) Ack(ctx context.Context, key Key, taskSetUID types.UID) error {
	err := d.store.deleteTaskSet(ctx, key.Namespace, key.TaskSetName(), taskSetUID)
	if err == nil || errors.Is(err, ErrNotFound) {
		return nil
	}
	return fmt.Errorf("acknowledging WorkflowTaskSet %s/%s: %w", key.Namespace, key.TaskSetName(), err)
}

func (d *dispatcher) Completion(ctx context.Context, namespace string, cached, ledgers []TaskSetRef) ([]TaskSetRef, error) {
	// Each name can be known by several UIDs, the cache's stale and the
	// ledger's fresh, or the reverse. Every ref is of a TaskSet created before
	// this call, and no two objects hold a name at once, so once one of a
	// name's UIDs is deleted, or nothing has the name, all its refs are gone.
	// A UID another object has replaced is gone too; that object isn't one
	// the caller knows, and is left alone. A ref without a UID can't be
	// deleted safely, so it is skipped.
	var names []string
	refs := map[string][]TaskSetRef{}
	for _, ref := range slices.Concat(cached, ledgers) {
		if ref.UID == "" {
			continue
		}
		name := Key{Namespace: namespace, NodeID: ref.NodeID}.TaskSetName()
		if _, ok := refs[name]; !ok {
			names = append(names, name)
		}
		if !slices.Contains(refs[name], ref) {
			refs[name] = append(refs[name], ref)
		}
	}
	var gone []TaskSetRef
	var errs []error
	for _, name := range names {
		var replaced []TaskSetRef
		var failed []error
		cleared := false
		for _, ref := range refs[name] {
			err := d.store.deleteTaskSet(ctx, namespace, name, ref.UID)
			if err == nil || errors.Is(err, ErrNotFound) {
				cleared = true
				break
			}
			if errors.Is(err, ErrStale) {
				replaced = append(replaced, ref)
			} else {
				failed = append(failed, fmt.Errorf("deleting WorkflowTaskSet %s/%s: %w", namespace, name, err))
			}
		}
		if cleared {
			gone = append(gone, refs[name]...)
			continue
		}
		gone = append(gone, replaced...)
		errs = append(errs, failed...)
	}
	return gone, errors.Join(errs...)
}

func (d *dispatcher) Discard(ctx context.Context, key Key, resultUID types.UID) error {
	err := d.store.deleteResult(ctx, key.Namespace, key.ResultName(), resultUID)
	if err == nil || gone(err) {
		return nil
	}
	return fmt.Errorf("discarding WorkflowTaskResult %s/%s: %w", key.Namespace, key.ResultName(), err)
}

func (d *dispatcher) Result(ctx context.Context, key Key) (*wfv1.WorkflowTaskResult, error) {
	result, err := d.store.getResult(ctx, key.Namespace, key.ResultName())
	if err != nil {
		return nil, fmt.Errorf("getting WorkflowTaskResult %s/%s: %w", key.Namespace, key.ResultName(), err)
	}
	return result, nil
}

// CandidateSource finds a worker's candidates.
type CandidateSource interface {
	Candidates(ctx context.Context) ([]Candidate, error)
}

// ClaimWatcher tells a claim when its result is labelled for interrupt, or
// deleted or replaced. Watch calls notify, possibly more than once, until
// the returned cancel is called.
type ClaimWatcher interface {
	Watch(ctx context.Context, claim Claim, notify func(StopReason)) (cancel func())
}

type consumer struct {
	store      store
	worker     Worker
	candidates CandidateSource
	watcher    ClaimWatcher
}

func (c *consumer) Candidates(ctx context.Context) ([]Candidate, error) {
	if c.candidates == nil {
		return nil, fmt.Errorf("finding candidates: %w", ErrNotImplemented)
	}
	return c.candidates.Candidates(ctx)
}

// newResult is the claim on candidate. Its TaskSet owns it as well as its
// workflow, so when the workflow is deleted, garbage collection deletes the
// claim only once the TaskSet is gone: a TaskSet whose claim went first
// would look claimable, and a worker would run its task again.
func (c *consumer) newResult(candidate Candidate) *wfv1.WorkflowTaskResult {
	result := common.NewTaskResult(candidate.Namespace, candidate.ResultName(), candidate.WorkflowName, candidate.WorkflowUID, candidate.InstanceID)
	result.OwnerReferences = append(result.OwnerReferences, common.TaskSetOwnerReference(candidate.TaskSetName(), candidate.TaskSetUID))
	maps.Copy(result.Labels, map[string]string{
		common.LabelKeyPoolName:     candidate.PoolName,
		common.LabelKeyPoolHash:     candidate.PoolHash,
		common.LabelKeyTaskSetUID:   string(candidate.TaskSetUID),
		common.LabelKeyWorkerPod:    c.worker.PodName,
		common.LabelKeyWorkerPodUID: string(c.worker.PodUID),
	})
	return result
}

func (c *consumer) Claim(ctx context.Context, candidate Candidate) (Claimed, error) {
	ns, name := candidate.Namespace, candidate.ResultName()
	if ns != c.worker.Namespace || candidate.PoolHash != c.worker.PoolHash {
		return nil, fmt.Errorf("claiming %s/%s from pool %s: %w", ns, name, candidate.PoolHash, ErrWrongPool)
	}
	result, err := c.store.createResult(ctx, c.newResult(candidate))
	if err != nil && !errors.Is(err, errAlreadyExists) {
		// A name already taken is definite: this create made nothing. Reading
		// back would take a claim this worker already holds from an earlier
		// Claim on a stale candidate, and run its task twice.
		result, err = c.readBack(ctx, candidate, err)
	}
	if errors.Is(err, errAlreadyExists) {
		return nil, fmt.Errorf("claiming %s/%s: %w", ns, name, ErrClaimLost)
	}
	if err != nil {
		return nil, fmt.Errorf("claiming %s/%s: %w", ns, name, err)
	}
	claim, _ := ClaimFromResult(result)
	taskSet, err := c.store.getTaskSet(ctx, ns, candidate.TaskSetName())
	switch {
	case err != nil:
	case taskSet.UID != candidate.TaskSetUID:
		err = fmt.Errorf("WorkflowTaskSet %s/%s has UID %s, not %s: %w", ns, candidate.TaskSetName(), taskSet.UID, candidate.TaskSetUID, ErrStale)
	case taskSet.DeletionTimestamp != nil:
		// Gone to its workflow, which counts a TaskSet as deleted once its
		// deletion is accepted, though a finalizer may hold it for a while.
		err = fmt.Errorf("WorkflowTaskSet %s/%s is being deleted: %w", ns, candidate.TaskSetName(), ErrNotFound)
	}
	if err != nil {
		// The claim is withdrawn whether the TaskSet is gone or can't be read:
		// nothing has started, so the task can be claimed again.
		cause := err
		if gone(err) {
			cause = ErrTaskGone
		}
		if delErr := c.store.deleteResult(ctx, ns, name, claim.UID); delErr != nil && !gone(delErr) {
			return nil, fmt.Errorf("claiming %s/%s: %w; withdrawing the claim: %w", ns, name, cause, delErr)
		}
		logging.RequireLoggerFromContext(ctx).
			WithFields(logging.Fields{"namespace": ns, "result": name}).
			WithError(err).
			Info(ctx, "Withdrew claim on a WorkflowTaskSet that is gone or unreadable")
		return nil, fmt.Errorf("claiming %s/%s: %w", ns, name, cause)
	}
	h := &claimed{store: c.store, claim: claim, taskSet: taskSet, stop: newStopSignal(), cancel: func() {}}
	if c.watcher != nil {
		// The watch outlives this call, until the claim is completed, released or lost.
		h.cancel = sync.OnceFunc(c.watcher.Watch(context.WithoutCancel(ctx), claim, h.stop.stop))
	}
	return h, nil
}

// readBack returns the claim on candidate that a create failing with err
// (other than errAlreadyExists) may still have made, as when the create's
// response was lost. Such a
// claim, left alone, would never run: the TaskSet has a result, so no worker
// sees it as a candidate again, and this worker's pod looks alive to the
// controller. A claim is this create's only if it carries this worker pod's
// UID and the candidate's TaskSet UID and isn't complete; any other result
// means the claim is lost, and none, or a failed read, returns err.
func (c *consumer) readBack(ctx context.Context, candidate Candidate, err error) (*wfv1.WorkflowTaskResult, error) {
	result, getErr := c.store.getResult(ctx, candidate.Namespace, candidate.ResultName())
	if getErr != nil {
		return nil, err
	}
	claim, ok := ClaimFromResult(result)
	if !ok || claim.WorkerPodUID != c.worker.PodUID || claim.TaskSetUID != candidate.TaskSetUID || claim.Completed {
		return nil, errAlreadyExists
	}
	logging.RequireLoggerFromContext(ctx).
		WithFields(logging.Fields{"namespace": candidate.Namespace, "result": candidate.ResultName()}).
		WithError(err).
		Info(ctx, "Kept the claim a failed create made")
	return result, nil
}

func (c *consumer) FailOwn(ctx context.Context, message string) ([]Key, error) {
	selector := labels.SelectorFromSet(labels.Set{
		common.LabelKeyWorkerPodUID:           string(c.worker.PodUID),
		common.LabelKeyReportOutputsCompleted: "false",
	})
	results, err := c.store.listResults(ctx, c.worker.Namespace, selector)
	if err != nil {
		return nil, fmt.Errorf("listing own WorkflowTaskResults: %w", err)
	}
	var failed []Key
	var errs []error
	for i := range results {
		claim, ok := ClaimFromResult(&results[i])
		if !ok {
			continue
		}
		err := c.store.patchResult(ctx, claim.Namespace, claim.ResultName(), completion(claim, wfv1.NodeResult{
			Phase:   wfv1.NodeError,
			Message: message,
		}))
		switch {
		case err == nil:
			failed = append(failed, claim.Key)
		case gone(err):
			// No longer ours.
		default:
			errs = append(errs, fmt.Errorf("failing WorkflowTaskResult %s/%s: %w", claim.Namespace, claim.ResultName(), err))
		}
	}
	return failed, errors.Join(errs...)
}

type claimed struct {
	store   store
	claim   Claim
	taskSet *wfv1.WorkflowTaskSet
	stop    *stopSignal
	cancel  func()
}

func (h *claimed) Claim() Claim                   { return h.claim }
func (h *claimed) TaskSet() *wfv1.WorkflowTaskSet { return h.taskSet }
func (h *claimed) Stopped() <-chan struct{}       { return h.stop.done }
func (h *claimed) StopReason() StopReason         { return h.stop.reason() }

// write applies a patch fenced by the claim, and on finding the claim gone
// tells the task to stop.
func (h *claimed) write(ctx context.Context, what string, patch resultPatch) error {
	err := h.store.patchResult(ctx, h.claim.Namespace, h.claim.ResultName(), patch)
	if gone(err) {
		h.stop.stop(StopLost)
		h.cancel()
		return fmt.Errorf("%s %s/%s: %w: %w", what, h.claim.Namespace, h.claim.ResultName(), ErrClaimLost, err)
	}
	if err != nil {
		return fmt.Errorf("%s %s/%s: %w", what, h.claim.Namespace, h.claim.ResultName(), err)
	}
	return nil
}

func (h *claimed) Progress(ctx context.Context, progress wfv1.Progress) error {
	return h.write(ctx, "reporting progress to", resultPatch{
		uid:          h.claim.UID,
		workerPodUID: h.claim.WorkerPodUID,
		progress:     &progress,
	})
}

func (h *claimed) Complete(ctx context.Context, result wfv1.NodeResult) error {
	if err := h.write(ctx, "completing", completion(h.claim, result)); err != nil {
		return err
	}
	h.cancel()
	return nil
}

func (h *claimed) Release(ctx context.Context) error {
	// Stop watching first, so the release isn't reported as a lost claim.
	h.cancel()
	err := h.store.deleteResult(ctx, h.claim.Namespace, h.claim.ResultName(), h.claim.UID)
	if errors.Is(err, ErrStale) {
		return fmt.Errorf("releasing %s/%s: %w: %w", h.claim.Namespace, h.claim.ResultName(), ErrClaimLost, err)
	}
	if err != nil && !errors.Is(err, ErrNotFound) {
		return fmt.Errorf("releasing %s/%s: %w", h.claim.Namespace, h.claim.ResultName(), err)
	}
	return nil
}

// stopSignal records the first reason a claimed task must stop.
type stopSignal struct {
	once  sync.Once
	done  chan struct{}
	mu    sync.Mutex
	first StopReason
}

func newStopSignal() *stopSignal {
	return &stopSignal{done: make(chan struct{})}
}

func (s *stopSignal) stop(reason StopReason) {
	s.once.Do(func() {
		s.mu.Lock()
		s.first = reason
		s.mu.Unlock()
		close(s.done)
	})
}

func (s *stopSignal) reason() StopReason {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.first
}
