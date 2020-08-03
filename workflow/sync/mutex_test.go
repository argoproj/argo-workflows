package sync

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"

	fakewfclientset "github.com/argoproj/argo/pkg/client/clientset/versioned/fake"
)

var mutexWf = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: synchronization-wf-level-
spec:
  entrypoint: whalesay
  synchronization:
    mutex:
      name: test
  templates:
  - name: whalesay
    container:
      image: docker/whalesay:latest
      command: [cowsay]
      args: ["hello world"]
`

var mutexwfstatus = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  creationTimestamp: "2020-07-31T20:35:41Z"
  generateName: synchronization-wf-level-
  generation: 5
  labels:
    workflows.argoproj.io/phase: Running
  name: synchronization-wf-level-xxs94
  namespace: default
  resourceVersion: "347429"
  selfLink: /apis/argoproj.io/v1alpha1/namespaces/default/workflows/synchronization-wf-level-xxs94
  uid: fad73006-e1f3-4234-b04b-38c0bf79c5c1
spec:
  arguments: {}
  entrypoint: whalesay
  synchronization:
    mutex:
      name: test
  templates:
  - arguments: {}
    container:
      args:
      - hello world
      command:
      - cowsay
      image: docker/whalesay:latest
      name: ""
      resources: {}
    inputs: {}
    metadata: {}
    name: whalesay
    outputs: {}
status:
  finishedAt: null
  nodes:
    synchronization-wf-level-xxs94:
      displayName: synchronization-wf-level-xxs94
      finishedAt: null
      hostNodeName: docker-desktop
      id: synchronization-wf-level-xxs94
      message: ContainerCreating
      name: synchronization-wf-level-xxs94
      phase: Pending
      startedAt: "2020-07-31T20:35:49Z"
      templateName: whalesay
      templateScope: local/synchronization-wf-level-xxs94
      type: Pod
  phase: Running
  startedAt: "2020-07-31T20:35:49Z"
  synchronization:
    mutex:
      holding:
      - holder: synchronization-wf-level-xxs94
        mutex: default/mutex/test
`

func TestMutexLock(t *testing.T) {
	kube := fake.NewSimpleClientset()
	syncLimitFunc := GetSyncLimitFunc(kube)
	t.Run("InitializeSynchronization", func(t *testing.T) {
		concurrenyMgr := NewLockManager(syncLimitFunc, func(key string) {
		})
		wf := unmarshalWF(mutexwfstatus)
		wfclientset := fakewfclientset.NewSimpleClientset(wf)

		wfList, err := wfclientset.ArgoprojV1alpha1().Workflows("default").List(metav1.ListOptions{})
		assert.NoError(t, err)
		concurrenyMgr.Initialize(wfList)
		assert.Equal(t, 1, len(concurrenyMgr.syncLockMap))
	})
	t.Run("WfLevelMutexAcquireAndRelease", func(t *testing.T) {
		var nextKey string
		concurrenyMgr := NewLockManager(syncLimitFunc, func(key string) {
			nextKey = key
		})
		wf := unmarshalWF(mutexWf)
		wf1 := wf.DeepCopy()
		wf2 := wf.DeepCopy()
		wf3 := wf.DeepCopy()
		status, wfUpdate, msg, err := concurrenyMgr.TryAcquire(wf, "", 0, time.Now(), wf.Spec.Synchronization)
		assert.NoError(t, err)
		assert.Empty(t, msg)
		assert.True(t, status)
		assert.True(t, wfUpdate)
		assert.NotNil(t, wf.Status.Synchronization)
		assert.NotNil(t, wf.Status.Synchronization.Mutex)
		assert.NotNil(t, wf.Status.Synchronization.Mutex.Holding)
		assert.Equal(t, wf.Name, wf.Status.Synchronization.Mutex.Holding[0].Holder)

		// Try to acquire again
		status, wfUpdate, msg, err = concurrenyMgr.TryAcquire(wf, "", 0, time.Now(), wf.Spec.Synchronization)
		assert.NoError(t, err)
		assert.True(t, status)
		assert.Empty(t, msg)
		assert.False(t, wfUpdate)

		wf1.Name = "two"
		status, wfUpdate, msg, err = concurrenyMgr.TryAcquire(wf1, "", 0, time.Now(), wf1.Spec.Synchronization)
		assert.NoError(t, err)
		assert.NotEmpty(t, msg)
		assert.False(t, status)
		assert.True(t, wfUpdate)

		wf2.Name = "three"
		holderKey2 := getHolderKey(wf2, "")
		status, wfUpdate, msg, err = concurrenyMgr.TryAcquire(wf2, "", 5, time.Now(), wf2.Spec.Synchronization)
		assert.NoError(t, err)
		assert.NotEmpty(t, msg)
		assert.False(t, status)
		assert.True(t, wfUpdate)

		wf3.Name = "four"
		status, wfUpdate, msg, err = concurrenyMgr.TryAcquire(wf3, "", 0, time.Now(), wf3.Spec.Synchronization)
		assert.NoError(t, err)
		assert.NotEmpty(t, msg)
		assert.False(t, status)
		assert.True(t, wfUpdate)

		concurrenyMgr.Release(wf, "", wf.Namespace, wf.Spec.Synchronization)
		assert.Equal(t, holderKey2, nextKey)
		assert.NotNil(t, wf.Status.Synchronization)
		assert.Equal(t, 0, len(wf.Status.Synchronization.Mutex.Holding))

		// Low priority workflow try to acquire the lock
		status, wfUpdate, msg, err = concurrenyMgr.TryAcquire(wf1, "", 0, time.Now(), wf1.Spec.Synchronization)
		assert.NoError(t, err)
		assert.NotEmpty(t, msg)
		assert.False(t, status)
		assert.True(t, wfUpdate)

		// High Priority workflow acquires the lock
		status, wfUpdate, msg, err = concurrenyMgr.TryAcquire(wf2, "", 5, time.Now(), wf2.Spec.Synchronization)
		assert.NoError(t, err)
		assert.Empty(t, msg)
		assert.True(t, status)
		assert.True(t, wfUpdate)
		assert.NotNil(t, wf2.Status.Synchronization)
		assert.NotNil(t, wf2.Status.Synchronization.Mutex)
		assert.Equal(t, wf2.Name, wf2.Status.Synchronization.Mutex.Holding[0].Holder)
		concurrenyMgr.ReleaseAll(wf2)
		assert.Nil(t, wf2.Status.Synchronization)
	})

}

func TestMutexTmplLevel(t *testing.T) {
	kube := fake.NewSimpleClientset()

	syncLimitFunc := GetSyncLimitFunc(kube)
	t.Run("TemplateLevelAcquireAndRelease", func(t *testing.T) {
		//var nextKey string
		concurrenyMgr := NewLockManager(syncLimitFunc, func(key string) {
			//nextKey = key
		})
		wf := unmarshalWF(wfWithTmplSemaphore)
		tmpl := wf.Spec.Templates[2]

		status, wfUpdate, msg, err := concurrenyMgr.TryAcquire(wf, "semaphore-tmpl-level-xjvln-3448864205", 0, time.Now(), tmpl.Synchronization)
		assert.NoError(t, err)
		assert.Empty(t, msg)
		assert.True(t, status)
		assert.True(t, wfUpdate)
		assert.NotNil(t, wf.Status.Synchronization)
		assert.NotNil(t, wf.Status.Synchronization.Semaphore)
		assert.Equal(t, "semaphore-tmpl-level-xjvln-3448864205", wf.Status.Synchronization.Semaphore.Holding[0].Holders[0])

		// Try to acquire again
		status, wfUpdate, msg, err = concurrenyMgr.TryAcquire(wf, "semaphore-tmpl-level-xjvln-3448864205", 0, time.Now(), tmpl.Synchronization)
		assert.NoError(t, err)
		assert.True(t, status)
		assert.False(t, wfUpdate)
		assert.Empty(t, msg)

		status, wfUpdate, msg, err = concurrenyMgr.TryAcquire(wf, "semaphore-tmpl-level-xjvln-1607747183", 0, time.Now(), tmpl.Synchronization)
		assert.NoError(t, err)
		assert.NotEmpty(t, msg)
		assert.True(t, wfUpdate)
		assert.False(t, status)

		concurrenyMgr.Release(wf, "semaphore-tmpl-level-xjvln-3448864205", wf.Namespace, tmpl.Synchronization)
		assert.NotNil(t, wf.Status.Synchronization)
		assert.NotNil(t, wf.Status.Synchronization.Semaphore)
		assert.Empty(t, wf.Status.Synchronization.Semaphore.Holding[0].Holders)

		status, wfUpdate, msg, err = concurrenyMgr.TryAcquire(wf, "semaphore-tmpl-level-xjvln-1607747183", 0, time.Now(), tmpl.Synchronization)
		assert.NoError(t, err)
		assert.Empty(t, msg)
		assert.True(t, status)
		assert.True(t, wfUpdate)
		assert.NotNil(t, wf.Status.Synchronization)
		assert.NotNil(t, wf.Status.Synchronization.Semaphore)
		assert.Equal(t, "semaphore-tmpl-level-xjvln-1607747183", wf.Status.Synchronization.Semaphore.Holding[0].Holders[0])

	})
}