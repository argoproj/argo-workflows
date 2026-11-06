package dispatch

import (
	"cmp"
	"context"
	"fmt"
	"maps"
	"slices"
	"sync"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/uuid"

	"github.com/argoproj/argo-workflows/v4/pkg/apis/workflow"
	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
)

// Resources the protocol calls, as an Op names them.
const (
	ResourceTaskSets = workflow.WorkflowTaskSetPlural
	ResourceResults  = workflow.WorkflowTaskResultPlural
)

// Op is one API call a Fake is about to make, for fault injection.
type Op struct {
	// Verb is create, get, list, patch or delete.
	Verb      string
	Resource  string
	Namespace string
	// Name is empty for list.
	Name string
}

// Fake is an in-memory implementation of every interface in this package,
// for tests. It keeps the API server's semantics the protocol relies on:
// create sets the UID and a creation timestamp and fails on a taken name,
// get, patch and delete fail on a missing name, deletes honour their UID
// precondition and patches their tests. Its consumers find candidates and
// see interrupts and lost claims as soon as they happen. It is safe for
// concurrent use.
type Fake struct {
	mu       sync.Mutex
	taskSets map[string]*fakeTaskSet
	results  map[string]*wfv1.WorkflowTaskResult
	health   map[string]Health
	watches  claimWatches
	seq      uint64
	// changed is closed and replaced on every change.
	changed chan struct{}
	fault   func(Op) error
	now     func() time.Time
}

type fakeTaskSet struct {
	obj *wfv1.WorkflowTaskSet
	seq uint64
}

// NewFake returns an empty Fake.
func NewFake() *Fake {
	return &Fake{
		taskSets: map[string]*fakeTaskSet{},
		results:  map[string]*wfv1.WorkflowTaskResult{},
		health:   map[string]Health{},
		watches:  claimWatches{},
		changed:  make(chan struct{}),
		now:      time.Now,
	}
}

// Dispatcher returns the controller's side.
func (f *Fake) Dispatcher() Dispatcher {
	return &dispatcher{store: f}
}

// Consumer returns a worker's side, with candidates and claim notifications.
func (f *Fake) Consumer(worker Worker) Consumer {
	return &consumer{
		store:      f,
		worker:     worker,
		candidates: fakeCandidates{fake: f, worker: worker},
		watcher:    fakeWatcher{fake: f},
	}
}

// Health implements PoolHealth.
func (f *Fake) Health(namespace, hash string) (Health, bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	h, ok := f.health[namespace+"/"+hash]
	return h, ok
}

// SetHealth sets a pool's health.
func (f *Fake) SetHealth(namespace, hash string, health Health) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.health[namespace+"/"+hash] = health
}

// SetFault makes every following call ask fault first: if it returns an
// error, the call fails with it and changes nothing. fault runs without the
// Fake's lock held, so it may change the Fake to stage a race. Nil clears it.
func (f *Fake) SetFault(fault func(Op) error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.fault = fault
}

// SetClock sets the clock creation timestamps come from.
func (f *Fake) SetClock(now func() time.Time) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.now = now
}

// TaskSet returns a copy of the named TaskSet.
func (f *Fake) TaskSet(namespace, name string) (*wfv1.WorkflowTaskSet, bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	ts, ok := f.taskSets[namespace+"/"+name]
	if !ok {
		return nil, false
	}
	return ts.obj.DeepCopy(), true
}

// TaskSets returns copies of the namespace's TaskSets, oldest first.
func (f *Fake) TaskSets(namespace string) []wfv1.WorkflowTaskSet {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []wfv1.WorkflowTaskSet
	for _, ts := range f.sortedTaskSets() {
		if ts.obj.Namespace == namespace {
			out = append(out, *ts.obj.DeepCopy())
		}
	}
	return out
}

// Result returns a copy of the named result.
func (f *Fake) Result(namespace, name string) (*wfv1.WorkflowTaskResult, bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	r, ok := f.results[namespace+"/"+name]
	if !ok {
		return nil, false
	}
	return r.DeepCopy(), true
}

// CreateTaskSet creates a TaskSet as another client might, such as the agent
// TaskSet of a workflow named like a pooled task, or a pooled TaskSet stored
// by an old CRD without spec.pool, and returns it as stored.
func (f *Fake) CreateTaskSet(taskSet *wfv1.WorkflowTaskSet) (*wfv1.WorkflowTaskSet, error) {
	return f.createTaskSet(context.Background(), taskSet)
}

// CreateResult creates a result as another client might, such as a claim on
// an older TaskSet of the node, and returns it as stored.
func (f *Fake) CreateResult(result *wfv1.WorkflowTaskResult) (*wfv1.WorkflowTaskResult, error) {
	return f.createResult(context.Background(), result)
}

// DeleteTaskSet deletes the named TaskSet without a precondition, as a user
// might, and says whether it existed.
func (f *Fake) DeleteTaskSet(namespace, name string) bool {
	f.mu.Lock()
	key := namespace + "/" + name
	_, ok := f.taskSets[key]
	delete(f.taskSets, key)
	notify := f.changedLocked("")
	f.mu.Unlock()
	notify()
	return ok
}

// DeleteResult deletes the named result without a precondition, as garbage
// collection does when its workflow is deleted, and says whether it existed.
func (f *Fake) DeleteResult(namespace, name string) bool {
	f.mu.Lock()
	key := namespace + "/" + name
	_, ok := f.results[key]
	delete(f.results, key)
	notify := f.changedLocked(key)
	f.mu.Unlock()
	notify()
	return ok
}

// sortedTaskSets returns the TaskSets oldest first. Call with the lock held.
func (f *Fake) sortedTaskSets() []*fakeTaskSet {
	return slices.SortedFunc(maps.Values(f.taskSets), func(a, b *fakeTaskSet) int { return cmp.Compare(a.seq, b.seq) })
}

// changedLocked wakes everything waiting for a change and, if resultKey is
// set, works out what that result's watches must be told. Call with the lock
// held, and call the returned func after releasing it.
func (f *Fake) changedLocked(resultKey string) func() {
	close(f.changed)
	f.changed = make(chan struct{})
	if resultKey == "" {
		return func() {}
	}
	result := f.results[resultKey]
	var calls []func()
	kept := f.watches[resultKey][:0]
	for _, w := range f.watches[resultKey] {
		switch {
		case result == nil || result.UID != w.claim.UID:
			calls = append(calls, func() { w.notify(StopLost) })
			continue
		case w.interruptedBy(result):
			calls = append(calls, func() { w.notify(StopInterrupted) })
		}
		kept = append(kept, w)
	}
	f.watches[resultKey] = kept
	return func() {
		for _, call := range calls {
			call()
		}
	}
}

// begin runs the fault hook, then takes the lock.
func (f *Fake) begin(op Op) error {
	f.mu.Lock()
	fault := f.fault
	f.mu.Unlock()
	if fault != nil {
		if err := fault(op); err != nil {
			return err
		}
	}
	f.mu.Lock()
	return nil
}

func (f *Fake) stamp(meta *metav1.ObjectMeta) {
	meta.UID = uuid.NewUUID()
	meta.CreationTimestamp = metav1.NewTime(f.now()).Rfc3339Copy()
}

func notFound(resource, namespace, name string) error {
	return fmt.Errorf("%s %s/%s: %w", resource, namespace, name, ErrNotFound)
}

func (f *Fake) createTaskSet(_ context.Context, taskSet *wfv1.WorkflowTaskSet) (*wfv1.WorkflowTaskSet, error) {
	if err := f.begin(Op{"create", ResourceTaskSets, taskSet.Namespace, taskSet.Name}); err != nil {
		return nil, err
	}
	key := taskSet.Namespace + "/" + taskSet.Name
	if _, ok := f.taskSets[key]; ok {
		f.mu.Unlock()
		return nil, fmt.Errorf("%s %s: %w", ResourceTaskSets, key, errAlreadyExists)
	}
	stored := taskSet.DeepCopy()
	f.stamp(&stored.ObjectMeta)
	f.seq++
	f.taskSets[key] = &fakeTaskSet{obj: stored, seq: f.seq}
	notify := f.changedLocked("")
	f.mu.Unlock()
	notify()
	return stored.DeepCopy(), nil
}

func (f *Fake) getTaskSet(_ context.Context, namespace, name string) (*wfv1.WorkflowTaskSet, error) {
	if err := f.begin(Op{"get", ResourceTaskSets, namespace, name}); err != nil {
		return nil, err
	}
	defer f.mu.Unlock()
	ts, ok := f.taskSets[namespace+"/"+name]
	if !ok {
		return nil, notFound(ResourceTaskSets, namespace, name)
	}
	return ts.obj.DeepCopy(), nil
}

func (f *Fake) deleteTaskSet(_ context.Context, namespace, name string, uid types.UID) error {
	if err := f.begin(Op{"delete", ResourceTaskSets, namespace, name}); err != nil {
		return err
	}
	key := namespace + "/" + name
	ts, ok := f.taskSets[key]
	switch {
	case !ok:
		f.mu.Unlock()
		return notFound(ResourceTaskSets, namespace, name)
	case ts.obj.UID != uid:
		f.mu.Unlock()
		return fmt.Errorf("%s %s has UID %s, not %s: %w", ResourceTaskSets, key, ts.obj.UID, uid, ErrStale)
	}
	delete(f.taskSets, key)
	notify := f.changedLocked("")
	f.mu.Unlock()
	notify()
	return nil
}

func (f *Fake) createResult(_ context.Context, result *wfv1.WorkflowTaskResult) (*wfv1.WorkflowTaskResult, error) {
	if err := f.begin(Op{"create", ResourceResults, result.Namespace, result.Name}); err != nil {
		return nil, err
	}
	key := result.Namespace + "/" + result.Name
	if _, ok := f.results[key]; ok {
		f.mu.Unlock()
		return nil, fmt.Errorf("%s %s: %w", ResourceResults, key, errAlreadyExists)
	}
	stored := result.DeepCopy()
	f.stamp(&stored.ObjectMeta)
	f.results[key] = stored
	notify := f.changedLocked(key)
	f.mu.Unlock()
	notify()
	return stored.DeepCopy(), nil
}

func (f *Fake) getResult(_ context.Context, namespace, name string) (*wfv1.WorkflowTaskResult, error) {
	if err := f.begin(Op{"get", ResourceResults, namespace, name}); err != nil {
		return nil, err
	}
	defer f.mu.Unlock()
	r, ok := f.results[namespace+"/"+name]
	if !ok {
		return nil, notFound(ResourceResults, namespace, name)
	}
	return r.DeepCopy(), nil
}

func (f *Fake) listResults(_ context.Context, namespace string, selector labels.Selector) ([]wfv1.WorkflowTaskResult, error) {
	if err := f.begin(Op{"list", ResourceResults, namespace, ""}); err != nil {
		return nil, err
	}
	defer f.mu.Unlock()
	var out []wfv1.WorkflowTaskResult
	for _, r := range f.results {
		if r.Namespace == namespace && selector.Matches(labels.Set(r.Labels)) {
			out = append(out, *r.DeepCopy())
		}
	}
	slices.SortFunc(out, func(a, b wfv1.WorkflowTaskResult) int { return cmp.Compare(a.Name, b.Name) })
	return out, nil
}

func (f *Fake) patchResult(_ context.Context, namespace, name string, patch resultPatch) error {
	if err := f.begin(Op{"patch", ResourceResults, namespace, name}); err != nil {
		return err
	}
	key := namespace + "/" + name
	r, ok := f.results[key]
	switch {
	case !ok:
		f.mu.Unlock()
		return notFound(ResourceResults, namespace, name)
	case !patch.owns(r):
		f.mu.Unlock()
		return fmt.Errorf("%s %s isn't UID %s of worker pod %q: %w", ResourceResults, key, patch.uid, patch.workerPodUID, ErrStale)
	}
	patch.apply(r)
	notify := f.changedLocked(key)
	f.mu.Unlock()
	notify()
	return nil
}

func (f *Fake) deleteResult(_ context.Context, namespace, name string, uid types.UID) error {
	if err := f.begin(Op{"delete", ResourceResults, namespace, name}); err != nil {
		return err
	}
	key := namespace + "/" + name
	r, ok := f.results[key]
	switch {
	case !ok:
		f.mu.Unlock()
		return notFound(ResourceResults, namespace, name)
	case r.UID != uid:
		f.mu.Unlock()
		return fmt.Errorf("%s %s has UID %s, not %s: %w", ResourceResults, key, r.UID, uid, ErrStale)
	}
	delete(f.results, key)
	notify := f.changedLocked(key)
	f.mu.Unlock()
	notify()
	return nil
}

type fakeCandidates struct {
	fake   *Fake
	worker Worker
}

func (c fakeCandidates) Candidates(ctx context.Context) ([]Candidate, error) {
	f := c.fake
	next := func() <-chan struct{} {
		f.mu.Lock()
		defer f.mu.Unlock()
		return f.changed
	}
	return waitForCandidates(ctx, next, func() []Candidate {
		f.mu.Lock()
		defer f.mu.Unlock()
		hasResult := func(key string) bool {
			_, ok := f.results[key]
			return ok
		}
		var found []Candidate
		for _, ts := range f.sortedTaskSets() {
			if candidate, ok := c.worker.candidate(ts.obj, hasResult); ok {
				found = append(found, candidate)
			}
		}
		return found
	})
}

type fakeWatcher struct {
	fake *Fake
}

func (w fakeWatcher) Watch(_ context.Context, claim Claim, notify func(StopReason)) func() {
	f := w.fake
	key := claim.Namespace + "/" + claim.ResultName()
	watch := &claimWatch{claim: claim, notify: notify}
	f.mu.Lock()
	f.watches.add(key, watch)
	// Report anything that happened before the watch started.
	pending := f.changedLocked(key)
	f.mu.Unlock()
	pending()
	return func() {
		f.mu.Lock()
		defer f.mu.Unlock()
		f.watches.remove(key, watch)
	}
}
