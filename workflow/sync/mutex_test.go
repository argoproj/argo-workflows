package sync

import (
	"testing"

	"github.com/argoproj/argo-workflows/v3/util/logging"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/utils/ptr"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	fakewfclientset "github.com/argoproj/argo-workflows/v3/pkg/client/clientset/versioned/fake"
)

var mutexWf = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: one
  namespace: default
spec:
  entrypoint: whalesay
  synchronization:
    mutexes:
      - name: test
  templates:
  - name: whalesay
    container:
      image: docker/whalesay:latest
      command: [cowsay]
      args: ["hello world"]
`

var mutexWfNamespaced = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: one
  namespace: default
spec:
  entrypoint: whalesay
  synchronization:
    mutexes:
      - namespace: other
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
  entrypoint: whalesay
  synchronization:
    mutexes:
      - name: test
  templates:
  - container:
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
        mutex: default/Mutex/test
`

func TestMutexLock(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	kube := fake.NewSimpleClientset()
	syncLimitFunc := GetSyncLimitFunc(kube)
	t.Run("InitializeSynchronization", func(t *testing.T) {
		syncManager := NewLockManager(ctx, kube, "", nil, syncLimitFunc, func(key string) {
		}, WorkflowExistenceFunc)
		wf := wfv1.MustUnmarshalWorkflow(mutexwfstatus)
		wfclientset := fakewfclientset.NewSimpleClientset(wf)

		wfList, err := wfclientset.ArgoprojV1alpha1().Workflows("default").List(ctx, metav1.ListOptions{})
		require.NoError(t, err)
		syncManager.Initialize(ctx, wfList.Items)
		assert.Len(t, syncManager.syncLockMap, 1)
	})
	t.Run("WfLevelMutexAcquireAndRelease", func(t *testing.T) {
		var nextWorkflow string
		syncManager := NewLockManager(ctx, kube, "", nil, syncLimitFunc, func(key string) {
			nextWorkflow = key
		}, WorkflowExistenceFunc)
		wf := wfv1.MustUnmarshalWorkflow(mutexWf)
		wf1 := wf.DeepCopy()
		wf2 := wf.DeepCopy()
		wf3 := wf.DeepCopy()
		status, wfUpdate, msg, failedLockName, err := syncManager.TryAcquire(ctx, wf, "", wf.Spec.Synchronization)
		require.NoError(t, err)
		assert.Empty(t, msg)
		assert.Empty(t, failedLockName)
		assert.True(t, status)
		assert.True(t, wfUpdate)
		require.NotNil(t, wf.Status.Synchronization)
		require.NotNil(t, wf.Status.Synchronization.Mutex)
		require.NotNil(t, wf.Status.Synchronization.Mutex.Holding)
		assert.Equal(t, getHolderKey(wf, ""), wf.Status.Synchronization.Mutex.Holding[0].Holder)

		// Try to acquire again
		status, wfUpdate, msg, failedLockName, err = syncManager.TryAcquire(ctx, wf, "", wf.Spec.Synchronization)
		require.NoError(t, err)
		assert.True(t, status)
		assert.Empty(t, msg)
		assert.Empty(t, failedLockName)
		assert.False(t, wfUpdate)

		wf1.Name = "two"
		status, wfUpdate, msg, failedLockName, err = syncManager.TryAcquire(ctx, wf1, "", wf1.Spec.Synchronization)
		require.NoError(t, err)
		assert.NotEmpty(t, msg)
		assert.Equal(t, "default/Mutex/test", failedLockName)
		assert.False(t, status)
		assert.True(t, wfUpdate)

		wf2.Name = "three"
		wf2.Spec.Priority = ptr.To(int32(5))
		holderKey2 := getHolderKey(wf2, "")
		status, wfUpdate, msg, failedLockName, err = syncManager.TryAcquire(ctx, wf2, "", wf2.Spec.Synchronization)
		require.NoError(t, err)
		assert.NotEmpty(t, msg)
		assert.Equal(t, "default/Mutex/test", failedLockName)
		assert.False(t, status)
		assert.True(t, wfUpdate)

		wf3.Name = "four"
		status, wfUpdate, msg, failedLockName, err = syncManager.TryAcquire(ctx, wf3, "", wf3.Spec.Synchronization)
		require.NoError(t, err)
		assert.NotEmpty(t, msg)
		assert.Equal(t, "default/Mutex/test", failedLockName)
		assert.False(t, status)
		assert.True(t, wfUpdate)

		syncManager.Release(ctx, wf, "", wf.Spec.Synchronization)
		assert.Equal(t, holderKey2, nextWorkflow)
		require.NotNil(t, wf.Status.Synchronization)
		assert.Empty(t, wf.Status.Synchronization.Mutex.Holding)

		// Low priority workflow try to acquire the lock
		status, wfUpdate, msg, failedLockName, err = syncManager.TryAcquire(ctx, wf1, "", wf1.Spec.Synchronization)
		require.NoError(t, err)
		assert.NotEmpty(t, msg)
		assert.Equal(t, "default/Mutex/test", failedLockName)
		assert.False(t, status)
		assert.False(t, wfUpdate)

		// High Priority workflow acquires the lock
		status, wfUpdate, msg, failedLockName, err = syncManager.TryAcquire(ctx, wf2, "", wf2.Spec.Synchronization)
		require.NoError(t, err)
		assert.Empty(t, msg)
		assert.Empty(t, failedLockName)
		assert.True(t, status)
		assert.True(t, wfUpdate)
		require.NotNil(t, wf2.Status.Synchronization)
		require.NotNil(t, wf2.Status.Synchronization.Mutex)
		assert.Equal(t, getHolderKey(wf2, ""), wf2.Status.Synchronization.Mutex.Holding[0].Holder)
		syncManager.ReleaseAll(ctx, wf2)
		assert.Nil(t, wf2.Status.Synchronization)
	})

	t.Run("WfLevelMutexOthernamespace", func(t *testing.T) {
		var nextWorkflow string
		syncManager := NewLockManager(ctx, kube, "", nil, syncLimitFunc, func(key string) {
			nextWorkflow = key
		}, WorkflowExistenceFunc)
		wf := wfv1.MustUnmarshalWorkflow(mutexWfNamespaced)
		wf1 := wf.DeepCopy()
		wf2 := wf.DeepCopy()
		wf3 := wf.DeepCopy()
		status, wfUpdate, msg, failedLockName, err := syncManager.TryAcquire(ctx, wf, "", wf.Spec.Synchronization)
		require.NoError(t, err)
		assert.Empty(t, msg)
		assert.Empty(t, failedLockName)
		assert.True(t, status)
		assert.True(t, wfUpdate)
		require.NotNil(t, wf.Status.Synchronization)
		require.NotNil(t, wf.Status.Synchronization.Mutex)
		require.NotNil(t, wf.Status.Synchronization.Mutex.Holding)
		expected := getHolderKey(wf, "")
		assert.Equal(t, expected, wf.Status.Synchronization.Mutex.Holding[0].Holder)

		// Try to acquire again
		status, wfUpdate, msg, failedLockName, err = syncManager.TryAcquire(ctx, wf, "", wf.Spec.Synchronization)
		require.NoError(t, err)
		assert.True(t, status)
		assert.Empty(t, failedLockName)
		assert.Empty(t, msg)
		assert.False(t, wfUpdate)

		wf1.Name = "two"
		wf1.Namespace = "two"
		status, wfUpdate, msg, failedLockName, err = syncManager.TryAcquire(ctx, wf1, "", wf1.Spec.Synchronization)
		require.NoError(t, err)
		assert.NotEmpty(t, msg)
		assert.Equal(t, "other/Mutex/test", failedLockName)
		assert.False(t, status)
		assert.True(t, wfUpdate)

		wf2.Name = "three"
		wf2.Namespace = "three"
		wf2.Spec.Priority = ptr.To(int32(5))
		holderKey2 := getHolderKey(wf2, "")
		status, wfUpdate, msg, failedLockName, err = syncManager.TryAcquire(ctx, wf2, "", wf2.Spec.Synchronization)
		require.NoError(t, err)
		assert.NotEmpty(t, msg)
		assert.Equal(t, "other/Mutex/test", failedLockName)
		assert.False(t, status)
		assert.True(t, wfUpdate)

		wf3.Name = "four"
		wf3.Namespace = "four"
		status, wfUpdate, msg, failedLockName, err = syncManager.TryAcquire(ctx, wf3, "", wf3.Spec.Synchronization)
		require.NoError(t, err)
		assert.NotEmpty(t, msg)
		assert.Equal(t, "other/Mutex/test", failedLockName)
		assert.False(t, status)
		assert.True(t, wfUpdate)

		syncManager.Release(ctx, wf, "", wf.Spec.Synchronization)
		assert.Equal(t, holderKey2, nextWorkflow)
		require.NotNil(t, wf.Status.Synchronization)
		assert.Empty(t, wf.Status.Synchronization.Mutex.Holding)

		// Low priority workflow try to acquire the lock
		status, wfUpdate, msg, failedLockName, err = syncManager.TryAcquire(ctx, wf1, "", wf1.Spec.Synchronization)
		require.NoError(t, err)
		assert.NotEmpty(t, msg)
		assert.Equal(t, "other/Mutex/test", failedLockName)
		assert.False(t, status)
		assert.False(t, wfUpdate)

		// High Priority workflow acquires the lock
		status, wfUpdate, msg, failedLockName, err = syncManager.TryAcquire(ctx, wf2, "", wf2.Spec.Synchronization)
		require.NoError(t, err)
		assert.Empty(t, msg)
		assert.Empty(t, failedLockName)
		assert.True(t, status)
		assert.True(t, wfUpdate)
		require.NotNil(t, wf2.Status.Synchronization)
		require.NotNil(t, wf2.Status.Synchronization.Mutex)
		expected = getHolderKey(wf2, "")
		assert.Equal(t, expected, wf2.Status.Synchronization.Mutex.Holding[0].Holder)
		syncManager.ReleaseAll(ctx, wf2)
		assert.Nil(t, wf2.Status.Synchronization)
	})
}

var mutexWfWithTmplLevel = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: synchronization-tmpl-level-mutex-vjcdk
  namespace: default
spec:
  entrypoint: synchronization-tmpl-level-mutex-example
  templates:
  - name: synchronization-tmpl-level-mutex-example
    steps:
    - - arguments:
          parameters:
          - name: seconds
            value: '{{item}}'
        name: synchronization-acquire-lock
        template: acquire-lock
        withParam: '["1","2","3"]'
  - container:
      args:
      - sleep 20; echo acquired lock
      command:
      - sh
      - -c
      image: alpine:3.23
      name: ""
    name: acquire-lock
    synchronization:
      mutexes:
        - name: welcome
status:
  finishedAt: null
  nodes:
    synchronization-tmpl-level-mutex-vjcdk:
      children:
      - synchronization-tmpl-level-mutex-vjcdk-1320763997
      displayName: synchronization-tmpl-level-mutex-vjcdk
      finishedAt: null
      id: synchronization-tmpl-level-mutex-vjcdk
      name: synchronization-tmpl-level-mutex-vjcdk
      phase: Pending
      startedAt: "2020-08-03T04:13:26Z"
      templateName: synchronization-tmpl-level-mutex-example
      templateScope: local/synchronization-tmpl-level-mutex-vjcdk
      type: Steps
    synchronization-tmpl-level-mutex-vjcdk-1320763997:
      boundaryID: synchronization-tmpl-level-mutex-vjcdk
      children:
      - synchronization-tmpl-level-mutex-vjcdk-3941195474
      - synchronization-tmpl-level-mutex-vjcdk-1432992664
      - synchronization-tmpl-level-mutex-vjcdk-2216915482
      displayName: '[0]'
      finishedAt: null
      id: synchronization-tmpl-level-mutex-vjcdk-1320763997
      name: synchronization-tmpl-level-mutex-vjcdk[0]
      phase: Pending
      startedAt: "2020-08-03T04:13:26Z"
      templateName: synchronization-tmpl-level-mutex-example
      templateScope: local/synchronization-tmpl-level-mutex-vjcdk
      type: StepGroup
    synchronization-tmpl-level-mutex-vjcdk-1432992664:
      boundaryID: synchronization-tmpl-level-mutex-vjcdk
      displayName: synchronization-acquire-lock(1:2)
      finishedAt: null
      id: synchronization-tmpl-level-mutex-vjcdk-1432992664
      message: 'Waiting for argo/mutex/welcome lock. Lock status: 0/1 '
      name: synchronization-tmpl-level-mutex-vjcdk[0].synchronization-acquire-lock(1:2)
      phase: Pending
      startedAt: "2020-08-03T04:13:26Z"
      templateName: acquire-lock
      templateScope: local/synchronization-tmpl-level-mutex-vjcdk
      type: Pod
    synchronization-tmpl-level-mutex-vjcdk-2216915482:
      boundaryID: synchronization-tmpl-level-mutex-vjcdk
      displayName: synchronization-acquire-lock(2:3)
      finishedAt: null
      id: synchronization-tmpl-level-mutex-vjcdk-2216915482
      message: 'Waiting for argo/mutex/welcome lock. Lock status: 0/1 '
      name: synchronization-tmpl-level-mutex-vjcdk[0].synchronization-acquire-lock(2:3)
      phase: Pending
      startedAt: "2020-08-03T04:13:26Z"
      templateName: acquire-lock
      templateScope: local/synchronization-tmpl-level-mutex-vjcdk
      type: Pod
    synchronization-tmpl-level-mutex-vjcdk-3941195474:
      boundaryID: synchronization-tmpl-level-mutex-vjcdk
      displayName: synchronization-acquire-lock(0:1)
      finishedAt: null
      id: synchronization-tmpl-level-mutex-vjcdk-3941195474
      name: synchronization-tmpl-level-mutex-vjcdk[0].synchronization-acquire-lock(0:1)
      phase: Pending
      startedAt: "2020-08-03T04:13:26Z"
      templateName: acquire-lock
      templateScope: local/synchronization-tmpl-level-mutex-vjcdk
      type: Pod
  phase: Running
  startedAt: "2020-08-03T04:13:26Z"
`

func TestMutexTmplLevel(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	kube := fake.NewSimpleClientset()

	syncLimitFunc := GetSyncLimitFunc(kube)
	t.Run("TemplateLevelAcquireAndRelease", func(t *testing.T) {
		// var nextKey string
		syncManager := NewLockManager(ctx, kube, "", nil, syncLimitFunc, func(key string) {
			// nextKey = key
		}, WorkflowExistenceFunc)
		wf := wfv1.MustUnmarshalWorkflow(mutexWfWithTmplLevel)
		tmpl := wf.Spec.Templates[1]

		status, wfUpdate, msg, failedLockName, err := syncManager.TryAcquire(ctx, wf, "synchronization-tmpl-level-mutex-vjcdk-3941195474", tmpl.Synchronization)
		require.NoError(t, err)
		assert.Empty(t, msg)
		assert.Empty(t, failedLockName)
		assert.True(t, status)
		assert.True(t, wfUpdate)
		require.NotNil(t, wf.Status.Synchronization)
		require.NotNil(t, wf.Status.Synchronization.Mutex)
		expected := getHolderKey(wf, "synchronization-tmpl-level-mutex-vjcdk-3941195474")
		assert.Equal(t, expected, wf.Status.Synchronization.Mutex.Holding[0].Holder)

		// Try to acquire again
		status, wfUpdate, msg, failedLockName, err = syncManager.TryAcquire(ctx, wf, "synchronization-tmpl-level-mutex-vjcdk-2216915482", tmpl.Synchronization)
		require.NoError(t, err)
		assert.True(t, wfUpdate)
		assert.Equal(t, "default/Mutex/welcome", failedLockName)
		assert.False(t, status)
		assert.NotEmpty(t, msg)

		status, wfUpdate, msg, failedLockName, err = syncManager.TryAcquire(ctx, wf, "synchronization-tmpl-level-mutex-vjcdk-1432992664", tmpl.Synchronization)
		require.NoError(t, err)
		assert.NotEmpty(t, msg)
		assert.Equal(t, "default/Mutex/welcome", failedLockName)
		assert.False(t, wfUpdate)
		assert.False(t, status)

		expected = getHolderKey(wf, "synchronization-tmpl-level-mutex-vjcdk-3941195474")
		assert.Equal(t, expected, wf.Status.Synchronization.Mutex.Holding[0].Holder)
		syncManager.Release(ctx, wf, "synchronization-tmpl-level-mutex-vjcdk-3941195474", tmpl.Synchronization)
		require.NotNil(t, wf.Status.Synchronization)
		require.NotNil(t, wf.Status.Synchronization.Mutex)
		assert.Empty(t, wf.Status.Synchronization.Mutex.Holding)

		status, wfUpdate, msg, failedLockName, err = syncManager.TryAcquire(ctx, wf, "synchronization-tmpl-level-mutex-vjcdk-2216915482", tmpl.Synchronization)
		require.NoError(t, err)
		assert.Empty(t, msg)
		assert.Empty(t, failedLockName)
		assert.True(t, status)
		assert.True(t, wfUpdate)
		require.NotNil(t, wf.Status.Synchronization)
		require.NotNil(t, wf.Status.Synchronization.Mutex)
		expected = getHolderKey(wf, "synchronization-tmpl-level-mutex-vjcdk-2216915482")
		assert.Equal(t, expected, wf.Status.Synchronization.Mutex.Holding[0].Holder)

		assert.NotEqual(t, "synchronization-tmpl-level-mutex-vjcdk-3941195474", wf.Status.Synchronization.Mutex.Holding[0].Holder)
		syncManager.Release(ctx, wf, "synchronization-tmpl-level-mutex-vjcdk-3941195474", tmpl.Synchronization)
		require.NotNil(t, wf.Status.Synchronization)
		require.NotNil(t, wf.Status.Synchronization.Mutex)
		assert.NotEmpty(t, wf.Status.Synchronization.Mutex.Holding)
	})
}
