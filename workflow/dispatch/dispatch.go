// Package dispatch hands pooled tasks from the controller to hot worker pods
// and back.
//
// A pooled task is a WorkflowTaskSet holding one task, named after its node
// (TaskSetName). A worker claims it by creating the task's
// WorkflowTaskResult, named after the node ID: Create is atomic, so at most
// one worker holds a claim. Stale objects are fenced by UID: the result
// carries the UID of the TaskSet it claims, every delete carries the UID of
// the object it means to delete, and every write to a result tests that the
// result is still the writer's own. An object of the same name with another
// UID is never touched.
//
// The controller's side is Dispatcher, and a worker's side is Consumer and
// the Claimed handle it returns. NewDispatcher and NewConsumer implement them
// over the Kubernetes API; NewFake implements them in memory for tests.
package dispatch

import (
	"context"
	"errors"
	"slices"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/argoproj/argo-workflows/v4/pkg/apis/workflow"
	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/workflow/common"
)

// Errors returned by the operations, wrapped; test for them with errors.Is.
// An operation that finds its claim gone returns ErrClaimLost together with
// ErrNotFound or ErrStale, saying whether the result was deleted or replaced.
var (
	// ErrClaimLost means the claim isn't held as the caller recorded it:
	// another worker claimed the task first, or the claim's result was
	// deleted or replaced. A worker discards the task's work.
	ErrClaimLost = errors.New("claim lost")
	// ErrTaskGone means the TaskSet was deleted or replaced just after the
	// claim was created; the claim was withdrawn, unless the error also says
	// withdrawing it failed.
	ErrTaskGone = errors.New("task set gone")
	// ErrNotFound means the object doesn't exist.
	ErrNotFound = errors.New("not found")
	// ErrStale means an object of that name exists but isn't the one meant:
	// it has another UID or, for a claim, another worker's. It was left
	// untouched.
	ErrStale = errors.New("stale UID")
	// ErrFieldsDropped means the stored TaskSet lacks spec.pool or
	// spec.dispatch, typically because its CRD predates them.
	ErrFieldsDropped = errors.New("the WorkflowTaskSet was stored without spec.pool or spec.dispatch; install the current CRDs")
	// ErrNameTaken means a WorkflowTaskSet with the task's name exists that
	// isn't a pooled task, such as the agent TaskSet of a workflow with that
	// name.
	ErrNameTaken = errors.New("a WorkflowTaskSet that isn't a pooled task has the task's name")
	// ErrStaleTaskSet means a pooled WorkflowTaskSet that can't be adopted,
	// left by an earlier workflow of the same name or already deleted, still
	// holds the task's name, as while its deletion is held by a finalizer:
	// enqueue again.
	ErrStaleTaskSet = errors.New("an earlier pooled WorkflowTaskSet of the task's name is still being deleted")
	// ErrWrongPool means a worker was asked to claim a task of another pool
	// or namespace.
	ErrWrongPool = errors.New("task belongs to another pool")
	// ErrNotImplemented means a watch-backed operation was called on an
	// implementation that wasn't given its source.
	ErrNotImplemented = errors.New("not implemented")
)

// taskSetSuffix makes a pooled TaskSet's name from its node ID, so it can't
// collide with an agent TaskSet, which is named after its workflow, except
// for a workflow literally named that.
const taskSetSuffix = "-task"

// Key identifies a pooled task: its workflow's namespace and its node ID.
type Key struct {
	Namespace string
	NodeID    string
}

// TaskSetName is the name of the task's WorkflowTaskSet.
func (k Key) TaskSetName() string { return k.NodeID + taskSetSuffix }

// ResultName is the name of the task's WorkflowTaskResult, which is its claim.
func (k Key) ResultName() string { return k.NodeID }

// Task is what the controller enqueues for a pooled node.
type Task struct {
	Key
	WorkflowName string
	WorkflowUID  types.UID
	// OnExit marks a task run by an exit handler.
	OnExit bool
	// PoolHash is the pool's hash, computed once at dispatch.
	PoolHash string
	// Template is the task's merged template, stored under the node ID.
	Template wfv1.Template
	// Pool is the resolved pool spec. Its Name and InstanceID label the TaskSet.
	Pool     wfv1.PoolSpec
	Dispatch wfv1.TaskDispatch
}

// PoolLabels are the labels that identify the pool with hash and spec, on its
// TaskSets, its Deployment and its pods: the pool name and hash, and the
// controller instance ID only when there is one, as a controller without one
// selects them without the label.
func PoolLabels(spec *wfv1.PoolSpec, hash string) map[string]string {
	labels := map[string]string{
		common.LabelKeyPoolName: spec.Name,
		common.LabelKeyPoolHash: hash,
	}
	if spec.InstanceID != "" {
		labels[common.LabelKeyControllerInstanceID] = spec.InstanceID
	}
	return labels
}

// Enqueued is the TaskSet a task was enqueued as.
type Enqueued struct {
	TaskSet *wfv1.WorkflowTaskSet
	// Adopted is set when the TaskSet already existed, as when the status
	// write that recorded its UID was lost.
	Adopted bool
}

// OwnedByWorkflow reports whether the object's owner is the workflow with uid,
// rather than an earlier workflow of the same name.
func OwnedByWorkflow(m metav1.Object, uid types.UID) bool {
	return slices.ContainsFunc(m.GetOwnerReferences(), func(owner metav1.OwnerReference) bool {
		return owner.Kind == workflow.WorkflowKind && owner.UID == uid
	})
}

// TaskSetRef names one pooled TaskSet of a workflow by node ID and UID.
type TaskSetRef struct {
	NodeID string
	UID    types.UID
}

// TaskSetRefFromMeta returns the ref of a pooled TaskSet, or false if the
// object isn't one.
func TaskSetRefFromMeta(obj metav1.Object) (TaskSetRef, bool) {
	nodeID, ok := strings.CutSuffix(obj.GetName(), taskSetSuffix)
	if !ok || obj.GetLabels()[common.LabelKeyPoolHash] == "" {
		return TaskSetRef{}, false
	}
	return TaskSetRef{NodeID: nodeID, UID: obj.GetUID()}, true
}

// LedgerEntry is a pooled node as its ledger records it.
type LedgerEntry struct {
	Key
	Ledger wfv1.PooledTaskStatus
}

// ShutdownOutcome is what Shutdown did to one task.
type ShutdownOutcome struct {
	Key
	// Found is the claim Drop found on a task the ledger had as enqueued,
	// which Shutdown then interrupts; Err is set if that failed.
	Found *Claim
	// Finished is set when no claim holds the task any more, so the node can
	// finish now. Otherwise the task was interrupted and its worker has yet to
	// confirm it stopped.
	Finished bool
	// Err is set when the task couldn't be dropped or interrupted; try again.
	Err error
}

// Dispatcher is the controller's side.
type Dispatcher interface {
	// Enqueue creates the task's TaskSet or, if one of that name exists that
	// the task's workflow owns, adopts it. A pooled TaskSet of that name owned
	// by another workflow UID, or by none, was left by an earlier workflow of
	// the same name: it is deleted and the TaskSet created afresh, failing with
	// ErrStaleTaskSet while the old one is still there, as it does while one
	// already deleted is. The TaskSet is annotated with the version of the
	// controller that wrote its spec.pool. It fails with
	// ErrFieldsDropped if the stored TaskSet lacks spec.pool or spec.dispatch,
	// still returning the TaskSet so it can be cleaned up, and with
	// ErrNameTaken if the existing TaskSet isn't a pooled task.
	Enqueue(ctx context.Context, task Task) (Enqueued, error)
	// Drop ends a queued task: it deletes the TaskSet, then reads the result
	// live. A nil claim means the task was never claimed and the node can
	// finish; otherwise it was claimed and must be interrupted. A worker
	// creates its claim before it reads the TaskSet, so no claim can start
	// after Drop without Drop finding it.
	Drop(ctx context.Context, key Key, taskSetUID types.UID) (*Claim, error)
	// Interrupt ends a claimed task: it deletes the TaskSet, then labels the
	// claim's result for its worker to stop the task. It fails with
	// ErrClaimLost if the result has been deleted or replaced.
	Interrupt(ctx context.Context, key Key, taskSetUID, claimUID types.UID) error
	// Shutdown drops each enqueued task and interrupts each claimed or
	// interrupted one. Which tasks to spare is the caller's choice.
	Shutdown(ctx context.Context, tasks []LedgerEntry) []ShutdownOutcome
	// Ack deletes a finished node's TaskSet. A TaskSet already gone is fine;
	// one of the same name with another UID is left alone and reported as
	// ErrStale.
	Ack(ctx context.Context, key Key, taskSetUID types.UID) error
	// Completion deletes all of a workflow's pooled TaskSets, given those in
	// the cache and those the ledgers record (a TaskSet created this
	// reconcile may not be cached yet), one at a time. It returns the refs it
	// found gone: deleted, or already deleted or replaced. A ref without a UID
	// is skipped. Every other ref failed to delete, and err says why; while
	// err is set, their TaskSets may remain. Delete the workflow's results
	// only once it returns nil: a TaskSet left without its claim looks
	// claimable, and a worker would run its task again.
	Completion(ctx context.Context, namespace string, cached, ledgers []TaskSetRef) (gone []TaskSetRef, err error)
	// Discard deletes a result that isn't the claim the node's ledger
	// accepts, such as a claim on an older TaskSet of the node, which would
	// otherwise keep every worker from claiming the node's TaskSet. A result
	// already gone or replaced is fine.
	Discard(ctx context.Context, key Key, resultUID types.UID) error
	// Result reads the task's result live, bypassing any cache, as the
	// controller must before it records a claimant dead: a claim completed
	// just before its claimant went still counts, and one its claimant
	// deleted is a release. It fails with ErrNotFound if there is none.
	Result(ctx context.Context, key Key) (*wfv1.WorkflowTaskResult, error)
}

// Worker identifies a worker pod.
type Worker struct {
	Namespace string
	PoolHash  string
	PodName   string
	PodUID    types.UID
}

// Candidate is a TaskSet that looked claimable.
type Candidate struct {
	Key
	TaskSetUID   types.UID
	WorkflowName string
	WorkflowUID  types.UID
	// InstanceID is the TaskSet's controller instance ID, empty if it has none.
	InstanceID string
	PoolName   string
	PoolHash   string
	CreatedAt  metav1.Time
}

// CandidateFromTaskSet returns the candidate for a pooled TaskSet's metadata,
// or false if the object isn't one.
func CandidateFromTaskSet(obj metav1.Object) (Candidate, bool) {
	ref, ok := TaskSetRefFromMeta(obj)
	if !ok {
		return Candidate{}, false
	}
	// A TaskSet being deleted, as while a finalizer holds it, is gone to its
	// workflow, which may already have completed and deleted its results.
	if obj.GetDeletionTimestamp() != nil {
		return Candidate{}, false
	}
	labels := obj.GetLabels()
	c := Candidate{
		Key:          Key{Namespace: obj.GetNamespace(), NodeID: ref.NodeID},
		TaskSetUID:   ref.UID,
		WorkflowName: labels[common.LabelKeyWorkflow],
		InstanceID:   labels[common.LabelKeyControllerInstanceID],
		PoolName:     labels[common.LabelKeyPoolName],
		PoolHash:     labels[common.LabelKeyPoolHash],
		CreatedAt:    obj.GetCreationTimestamp(),
	}
	for _, owner := range obj.GetOwnerReferences() {
		if owner.Kind == workflow.WorkflowKind && owner.Name == c.WorkflowName {
			c.WorkflowUID = owner.UID
		}
	}
	if c.WorkflowName == "" || c.WorkflowUID == "" {
		return Candidate{}, false
	}
	return c, true
}

// Consumer is a worker's side.
type Consumer interface {
	// Candidates blocks until at least one of the worker's pool's TaskSets
	// has no result, then returns those, oldest first. They may be stale by
	// the time the caller sees them; Claim decides.
	Candidates(ctx context.Context) ([]Candidate, error)
	// Claim claims a candidate. It fails with ErrClaimLost if another worker
	// holds the claim. A claim its create made though the create failed, as
	// when the response was lost, is read back and kept. Once the claim is created it reads the TaskSet, and if
	// that fails it withdraws the claim and returns why: ErrTaskGone if the
	// TaskSet was deleted or replaced, otherwise the read's error. If the
	// withdrawal fails too, the claim may remain and the error says so.
	Claim(ctx context.Context, candidate Candidate) (Claimed, error)
	// FailOwn finalises as Error, with the message, every incomplete result
	// claimed by this worker pod, as on a restart before claiming again. It
	// returns the tasks it finalised.
	FailOwn(ctx context.Context, message string) ([]Key, error)
}

// StopReason is why a worker must stop a claimed task.
type StopReason string

const (
	// StopInterrupted means the controller ended the node: stop the task and
	// Complete the result, whose phase and outputs are then ignored.
	StopInterrupted StopReason = "interrupt"
	// StopLost means the claim's result was deleted or replaced: stop the
	// task and report nothing.
	StopLost StopReason = "lost"
)

// Claimed is a worker's handle on a task it claimed.
type Claimed interface {
	// Claim is the claim as created.
	Claim() Claim
	// TaskSet is the TaskSet as read when claiming.
	TaskSet() *wfv1.WorkflowTaskSet
	// Progress records the task's progress. It fails with ErrClaimLost if
	// the claim was lost.
	Progress(ctx context.Context, progress wfv1.Progress) error
	// Complete writes the task's phase, message, outputs and progress and
	// marks the result complete, in one write. It fails with ErrClaimLost if
	// the claim was lost, and never recreates the result.
	Complete(ctx context.Context, result wfv1.NodeResult) error
	// Release deletes the claim so the task can be claimed again. A result
	// already gone is fine; one with another UID is left alone and reported
	// as ErrClaimLost. Release first stops the claim's watch, so the watch
	// reports neither the release nor anything after it on Stopped; if
	// Stopped was already closed, it stays closed.
	Release(ctx context.Context) error
	// Stopped is closed when the worker must stop the task; StopReason then
	// says why.
	Stopped() <-chan struct{}
	// StopReason is the first reason the task had to stop, or empty.
	StopReason() StopReason
}

// Claim is a worker's claim on a task, as its result records it.
type Claim struct {
	Key
	// UID is the result's UID.
	UID           types.UID
	TaskSetUID    types.UID
	WorkerPodName string
	WorkerPodUID  types.UID
	// CreatedAt is the result's creation time, on the API server's clock.
	CreatedAt   metav1.Time
	Completed   bool
	Interrupted bool
}

// ClaimFromResult returns the claim a result records, or false if the result
// isn't a pooled task's claim.
func ClaimFromResult(obj metav1.Object) (Claim, bool) {
	labels := obj.GetLabels()
	taskSetUID := labels[common.LabelKeyTaskSetUID]
	if taskSetUID == "" {
		return Claim{}, false
	}
	return Claim{
		Key:           Key{Namespace: obj.GetNamespace(), NodeID: obj.GetName()},
		UID:           obj.GetUID(),
		TaskSetUID:    types.UID(taskSetUID),
		WorkerPodName: labels[common.LabelKeyWorkerPod],
		WorkerPodUID:  types.UID(labels[common.LabelKeyWorkerPodUID]),
		CreatedAt:     obj.GetCreationTimestamp(),
		Completed:     labels[common.LabelKeyReportOutputsCompleted] == "true",
		Interrupted:   labels[common.LabelKeyInterrupt] == "true",
	}, true
}

// HealthState is a pool's health.
type HealthState string

const (
	// HealthOK means the pool is serving.
	HealthOK HealthState = "OK"
	// HealthDegraded means the pool's pods can't be created or started; its
	// tasks stay queued.
	HealthDegraded HealthState = "Degraded"
	// HealthUnbuildable means the pool can't be built from its spec; its
	// tasks fail.
	HealthUnbuildable HealthState = "Unbuildable"
)

// Health is a pool's health and why.
type Health struct {
	State  HealthState
	Reason string
}

// PoolHealth reports pools' health. The controller's is nil until the pool
// controller provides one.
type PoolHealth interface {
	// Health reports the health of the pool with the hash in the namespace,
	// or false if nothing is known about it.
	Health(namespace, hash string) (Health, bool)
}
