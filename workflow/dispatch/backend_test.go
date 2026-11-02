package dispatch

import (
	"errors"
	"net/http"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/require"
	apiv1 "k8s.io/api/core/v1"
	apierr "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/uuid"
	k8stesting "k8s.io/client-go/testing"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/pkg/client/clientset/versioned/fake"
	"github.com/argoproj/argo-workflows/v4/util/logging"
)

// backend is one implementation under the shared tests, with direct access
// to its storage for setting up and inspecting objects.
type backend struct {
	name       string
	store      store
	dispatcher Dispatcher
	consumer   func(Worker) Consumer
	// setFault makes calls matching op fail; nil clears it.
	setFault func(func(Op) error)
	// replaceResult deletes the result of result's name, unconditionally, and
	// creates result in its place. A fault may call it, to stage a race.
	replaceResult func(result *wfv1.WorkflowTaskResult) error
	// fake is set for the fake backend.
	fake *Fake
	// clientset is set for the Kubernetes backend.
	clientset *fake.Clientset
}

func newFakeBackend() *backend {
	f := NewFake()
	return &backend{
		name:       "fake",
		store:      f,
		dispatcher: f.Dispatcher(),
		consumer:   f.Consumer,
		setFault:   f.SetFault,
		replaceResult: func(result *wfv1.WorkflowTaskResult) error {
			f.DeleteResult(result.Namespace, result.Name)
			_, err := f.CreateResult(result)
			return err
		},
		fake: f,
	}
}

// newKubeBackend is the Kubernetes implementation over the generated fake
// clientset. Its tracker never sets UIDs or creation timestamps, ignores
// delete preconditions and returns a failed JSON patch test as a plain
// error, so a reactor adds what the API server does.
func newKubeBackend() *backend {
	clientset := fake.NewClientset()
	tracker := clientset.Tracker()
	var faultPtr atomic.Pointer[func(Op) error]
	// Reactors run under the clientset's lock, so each call is atomic.
	clientset.PrependReactor("*", "*", func(action k8stesting.Action) (bool, runtime.Object, error) {
		if fault := faultPtr.Load(); fault != nil {
			op := Op{Verb: action.GetVerb(), Resource: action.GetResource().Resource, Namespace: action.GetNamespace()}
			if named, ok := action.(interface{ GetName() string }); ok {
				op.Name = named.GetName()
			}
			if create, ok := action.(k8stesting.CreateAction); ok {
				m, _ := meta.Accessor(create.GetObject())
				op.Name = m.GetName()
			}
			if err := (*fault)(op); err != nil {
				return true, nil, err
			}
		}
		gvr := action.GetResource()
		switch a := action.(type) {
		case k8stesting.CreateAction:
			obj := a.GetObject().DeepCopyObject()
			m, err := meta.Accessor(obj)
			if err != nil {
				return true, nil, err
			}
			m.SetUID(uuid.NewUUID())
			m.SetCreationTimestamp(metav1.Now().Rfc3339Copy())
			if err := tracker.Create(gvr, obj, a.GetNamespace()); err != nil {
				return true, nil, err
			}
			return true, obj, nil
		case k8stesting.DeleteAction:
			if pre := a.GetDeleteOptions().Preconditions; pre != nil && pre.UID != nil {
				existing, err := tracker.Get(gvr, a.GetNamespace(), a.GetName())
				if err != nil {
					return true, nil, err
				}
				m, _ := meta.Accessor(existing)
				if m.GetUID() != *pre.UID {
					return true, nil, apierr.NewConflict(gvr.GroupResource(), a.GetName(),
						errors.New("Precondition failed: UID in precondition: "+string(*pre.UID)+", UID in object meta: "+string(m.GetUID())))
				}
			}
		case k8stesting.PatchAction:
			handled, obj, err := k8stesting.ObjectReaction(tracker)(action)
			var status apierr.APIStatus
			if err != nil && !errors.As(err, &status) {
				err = apierr.NewGenericServerResponse(http.StatusUnprocessableEntity, "patch", gvr.GroupResource(), a.GetName(), err.Error(), 0, false)
			}
			return handled, obj, err
		}
		return false, nil, nil
	})
	return &backend{
		name:       "kubernetes",
		store:      kubeStore{client: clientset},
		dispatcher: NewDispatcher(clientset),
		consumer: func(w Worker) Consumer {
			return NewConsumer(clientset, w, WatchSources{})
		},
		setFault: func(f func(Op) error) {
			if f == nil {
				faultPtr.Store(nil)
				return
			}
			faultPtr.Store(&f)
		},
		// A fault runs under the clientset's lock, so this goes to the tracker.
		replaceResult: func(result *wfv1.WorkflowTaskResult) error {
			gvr := wfv1.SchemeGroupVersion.WithResource(ResourceResults)
			if err := tracker.Delete(gvr, result.Namespace, result.Name); err != nil && !apierr.IsNotFound(err) {
				return err
			}
			obj := result.DeepCopy()
			obj.UID = uuid.NewUUID()
			obj.CreationTimestamp = metav1.Now().Rfc3339Copy()
			return tracker.Create(gvr, obj, result.Namespace)
		},
		clientset: clientset,
	}
}

// forEachBackend runs the test against every implementation.
func forEachBackend(t *testing.T, test func(t *testing.T, b *backend)) {
	t.Helper()
	for _, newBackend := range []func() *backend{newFakeBackend, newKubeBackend} {
		b := newBackend()
		t.Run(b.name, func(t *testing.T) { test(t, b) })
	}
}

const testNamespace = "argo"

func testTask(nodeID string) Task {
	return Task{
		Key:          Key{Namespace: testNamespace, NodeID: nodeID},
		WorkflowName: "my-wf",
		WorkflowUID:  "wf-uid",
		PoolHash:     "hash-1",
		Template: wfv1.Template{
			Name:      "main",
			Container: &apiv1.Container{Image: "busybox", Command: []string{"echo"}},
		},
		Pool:     wfv1.PoolSpec{Name: "my-pool", InitialReplicas: 1, ControllerVersion: "v4"},
		Dispatch: wfv1.TaskDispatch{TaskName: "my-wf-main-" + nodeID},
	}
}

func testWorker(name string) Worker {
	return Worker{Namespace: testNamespace, PoolHash: "hash-1", PodName: name, PodUID: types.UID(name + "-uid")}
}

func (b *backend) enqueue(t *testing.T, task Task) *wfv1.WorkflowTaskSet {
	t.Helper()
	enqueued, err := b.dispatcher.Enqueue(logging.TestContext(t.Context()), task)
	require.NoError(t, err)
	return enqueued.TaskSet
}

func (b *backend) taskSet(t *testing.T, key Key) (*wfv1.WorkflowTaskSet, bool) {
	t.Helper()
	ts, err := b.store.getTaskSet(t.Context(), key.Namespace, key.TaskSetName())
	if errors.Is(err, ErrNotFound) {
		return nil, false
	}
	require.NoError(t, err)
	return ts, true
}

func (b *backend) result(t *testing.T, key Key) (*wfv1.WorkflowTaskResult, bool) {
	t.Helper()
	r, err := b.store.getResult(t.Context(), key.Namespace, key.ResultName())
	if errors.Is(err, ErrNotFound) {
		return nil, false
	}
	require.NoError(t, err)
	return r, true
}

func (b *backend) candidate(t *testing.T, key Key) Candidate {
	t.Helper()
	ts, ok := b.taskSet(t, key)
	require.True(t, ok, "TaskSet %s exists", key.TaskSetName())
	c, ok := CandidateFromTaskSet(ts)
	require.True(t, ok)
	return c
}

func (b *backend) claim(t *testing.T, w Worker, key Key) Claimed {
	t.Helper()
	h, err := b.consumer(w).Claim(logging.TestContext(t.Context()), b.candidate(t, key))
	require.NoError(t, err)
	return h
}

// deleteResult deletes the result unconditionally, as garbage collection
// does when the workflow is deleted.
func (b *backend) deleteResult(t *testing.T, key Key) {
	t.Helper()
	r, ok := b.result(t, key)
	require.True(t, ok)
	require.NoError(t, b.store.deleteResult(t.Context(), key.Namespace, key.ResultName(), r.UID))
}

// deleteTaskSet deletes the TaskSet unconditionally.
func (b *backend) deleteTaskSet(t *testing.T, key Key) {
	t.Helper()
	ts, ok := b.taskSet(t, key)
	require.True(t, ok)
	require.NoError(t, b.store.deleteTaskSet(t.Context(), key.Namespace, key.TaskSetName(), ts.UID))
}
