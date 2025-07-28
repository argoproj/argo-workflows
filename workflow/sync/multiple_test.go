package sync

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/utils/ptr"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
)

var wfTmpl = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: %s
  namespace: default
spec:
  entrypoint: whalesay
  synchronization:
%s
  templates:
  - name: whalesay
    container:
      image: docker/whalesay:latest
      command: [cowsay]
      args: ["hello world"]
`

func templatedWorkflow(name string, syncBlock string) *wfv1.Workflow {
	return wfv1.MustUnmarshalWorkflow(fmt.Sprintf(wfTmpl, name, syncBlock))
}

func TestMultipleMutexLock(t *testing.T) {
	ctx := context.Background()
	kube := fake.NewSimpleClientset()
	syncLimitFunc := GetSyncLimitFunc(kube)
	t.Run("MultipleMutex", func(t *testing.T) {
		syncManager := NewLockManager(syncLimitFunc, func(key string) {},
			WorkflowExistenceFunc)
		wfall := templatedWorkflow("all",
			`    mutexes:
      - name: one
      - name: two
      - name: three
`)
		wf1 := templatedWorkflow("one",
			`    mutexes:
      - name: one
`)
		wf2 := templatedWorkflow("two",
			`    mutexes:
      - name: two
`)
		wf3 := templatedWorkflow("three",
			`    mutexes:
      - name: three
`)
		// Acquire 1
		status, wfUpdate, msg, failedLockName, err := syncManager.TryAcquire(ctx, wf1, "", wf1.Spec.Synchronization)
		require.NoError(t, err)
		assert.Empty(t, msg)
		assert.Empty(t, failedLockName)
		assert.True(t, status)
		assert.True(t, wfUpdate)

		// Acquire 2
		status, wfUpdate, msg, failedLockName, err = syncManager.TryAcquire(ctx, wf2, "", wf2.Spec.Synchronization)
		require.NoError(t, err)
		assert.Empty(t, msg)
		assert.Empty(t, failedLockName)
		assert.True(t, status)
		assert.True(t, wfUpdate)

		// Acquire 3
		status, wfUpdate, msg, failedLockName, err = syncManager.TryAcquire(ctx, wf3, "", wf3.Spec.Synchronization)
		require.NoError(t, err)
		assert.Empty(t, msg)
		assert.Empty(t, failedLockName)
		assert.True(t, status)
		assert.True(t, wfUpdate)

		// Fail to acquire because one locked
		status, wfUpdate, msg, failedLockName, err = syncManager.TryAcquire(ctx, wfall, "", wfall.Spec.Synchronization)
		require.NoError(t, err)
		assert.NotEmpty(t, msg)
		assert.Equal(t, "default/Mutex/one", failedLockName)
		assert.False(t, status)
		assert.True(t, wfUpdate)

		syncManager.ReleaseAll(wf1)
		// Fail to acquire because two locked
		status, wfUpdate, msg, failedLockName, err = syncManager.TryAcquire(ctx, wfall, "", wfall.Spec.Synchronization)
		require.NoError(t, err)
		assert.NotEmpty(t, msg)
		assert.Equal(t, "default/Mutex/two", failedLockName)
		assert.False(t, status)
		assert.False(t, wfUpdate)

		syncManager.ReleaseAll(wf2)
		// Fail to acquire because three locked
		status, wfUpdate, msg, failedLockName, err = syncManager.TryAcquire(ctx, wfall, "", wfall.Spec.Synchronization)
		require.NoError(t, err)
		assert.NotEmpty(t, msg)
		assert.Equal(t, "default/Mutex/three", failedLockName)
		assert.False(t, status)
		assert.False(t, wfUpdate)

		syncManager.ReleaseAll(wf3)
		// Now lock
		status, wfUpdate, msg, failedLockName, err = syncManager.TryAcquire(ctx, wfall, "", wfall.Spec.Synchronization)
		require.NoError(t, err)
		assert.Empty(t, msg)
		assert.Empty(t, failedLockName)
		assert.True(t, status)
		assert.True(t, wfUpdate)
	})
	t.Run("MultipleMutexOrdering", func(t *testing.T) {
		syncManager := NewLockManager(syncLimitFunc, func(key string) {},
			WorkflowExistenceFunc)
		wfall := templatedWorkflow("all",
			`    mutexes:
      - name: one
      - name: two
      - name: three
`)
		// Old style single mutex
		wf1 := templatedWorkflow("one",
			`    mutex:
      name: one
`)
		wf2 := templatedWorkflow("two",
			`    mutexes:
      - name: two
`)
		// Acquire 1
		status, wfUpdate, msg, failedLockName, err := syncManager.TryAcquire(ctx, wf1, "", wf1.Spec.Synchronization)
		require.NoError(t, err)
		assert.Empty(t, msg)
		assert.Empty(t, failedLockName)
		assert.True(t, status)
		assert.True(t, wfUpdate)

		// Fail to acquire because one locked
		status, wfUpdate, msg, failedLockName, err = syncManager.TryAcquire(ctx, wfall, "", wfall.Spec.Synchronization)
		require.NoError(t, err)
		assert.NotEmpty(t, msg)
		assert.Equal(t, "default/Mutex/one", failedLockName)
		assert.False(t, status)
		assert.True(t, wfUpdate)

		// Attempt 2, but blocked by all
		status, wfUpdate, msg, failedLockName, err = syncManager.TryAcquire(ctx, wf2, "", wf2.Spec.Synchronization)
		require.NoError(t, err)
		assert.NotEmpty(t, msg)
		assert.Equal(t, "default/Mutex/two", failedLockName)
		assert.False(t, status)
		assert.False(t, wfUpdate)

		// Fail to acquire because one locked
		status, wfUpdate, msg, failedLockName, err = syncManager.TryAcquire(ctx, wfall, "", wfall.Spec.Synchronization)
		require.NoError(t, err)
		assert.NotEmpty(t, msg)
		assert.Equal(t, "default/Mutex/one", failedLockName)
		assert.False(t, status)
		assert.False(t, wfUpdate)

		syncManager.ReleaseAll(wf1)
		syncManager.ReleaseAll(wf2)

		// Now lock
		status, wfUpdate, msg, failedLockName, err = syncManager.TryAcquire(ctx, wfall, "", wfall.Spec.Synchronization)
		require.NoError(t, err)
		assert.Empty(t, msg)
		assert.Empty(t, failedLockName)
		assert.True(t, status)
		assert.True(t, wfUpdate)
	})
}

const multipleConfigMap = `
apiVersion: v1
kind: ConfigMap
metadata:
 name: my-config
data:
 double: "2"
`

func TestMutexAndSemaphore(t *testing.T) {
	kube := fake.NewSimpleClientset()
	var cm v1.ConfigMap
	wfv1.MustUnmarshal([]byte(multipleConfigMap), &cm)

	ctx := context.Background()
	_, err := kube.CoreV1().ConfigMaps("default").Create(ctx, &cm, metav1.CreateOptions{})
	require.NoError(t, err)

	syncLimitFunc := GetSyncLimitFunc(kube)
	t.Run("MutexSemaphore", func(t *testing.T) {
		syncManager := NewLockManager(syncLimitFunc, func(key string) {},
			WorkflowExistenceFunc)
		wfmands1 := templatedWorkflow("mands1",
			`    mutexes:
       - name: one
    semaphores:
       - configMapKeyRef:
           key: double
           name: my-config
`)
		wfmands1copy := wfmands1.DeepCopy()
		wfmands2 := templatedWorkflow("mands2",
			`    mutex:
       name: two
    semaphore:
       configMapKeyRef:
         key: double
         name: my-config
`)
		wf1 := templatedWorkflow("one",
			`    mutexes:
       - name: one
`)
		wf2 := templatedWorkflow("two",
			`    mutexes:
       - name: two
`)
		wfsem := templatedWorkflow("three",
			`    semaphores:
       - configMapKeyRef:
           key: double
           name: my-config
`)
		// Acquire sem + 1
		status, wfUpdate, msg, failedLockName, err := syncManager.TryAcquire(ctx, wfmands1, "", wfmands1.Spec.Synchronization)
		require.NoError(t, err)
		assert.Empty(t, msg)
		assert.Empty(t, failedLockName)
		assert.True(t, status)
		assert.True(t, wfUpdate)

		// Acquire sem + 2
		status, wfUpdate, msg, failedLockName, err = syncManager.TryAcquire(ctx, wfmands2, "", wfmands2.Spec.Synchronization)
		require.NoError(t, err)
		assert.Empty(t, msg)
		assert.Empty(t, failedLockName)
		assert.True(t, status)
		assert.True(t, wfUpdate)

		// Fail 1
		status, wfUpdate, msg, failedLockName, err = syncManager.TryAcquire(ctx, wf1, "", wf1.Spec.Synchronization)
		require.NoError(t, err)
		assert.NotEmpty(t, msg)
		assert.Equal(t, "default/Mutex/one", failedLockName)
		assert.False(t, status)
		assert.True(t, wfUpdate)

		// Fail 2
		status, wfUpdate, msg, failedLockName, err = syncManager.TryAcquire(ctx, wf2, "", wf2.Spec.Synchronization)
		require.NoError(t, err)
		assert.NotEmpty(t, msg)
		assert.Equal(t, "default/Mutex/two", failedLockName)
		assert.False(t, status)
		assert.True(t, wfUpdate)

		// Fail sem
		status, wfUpdate, msg, failedLockName, err = syncManager.TryAcquire(ctx, wfsem, "", wfsem.Spec.Synchronization)
		require.NoError(t, err)
		assert.NotEmpty(t, msg)
		assert.Equal(t, "default/ConfigMap/my-config/double", failedLockName)
		assert.False(t, status)
		assert.True(t, wfUpdate)

		// Release 1 and sem
		syncManager.ReleaseAll(wfmands1)

		// Succeed 1
		status, wfUpdate, msg, failedLockName, err = syncManager.TryAcquire(ctx, wf1, "", wf1.Spec.Synchronization)
		require.NoError(t, err)
		assert.Empty(t, msg)
		assert.Empty(t, failedLockName)
		assert.True(t, status)
		assert.True(t, wfUpdate)

		// Fail 2
		status, wfUpdate, msg, failedLockName, err = syncManager.TryAcquire(ctx, wf2, "", wf2.Spec.Synchronization)
		require.NoError(t, err)
		assert.NotEmpty(t, msg)
		assert.Equal(t, "default/Mutex/two", failedLockName)
		assert.False(t, status)
		assert.False(t, wfUpdate)

		// Succeed sem
		status, wfUpdate, msg, failedLockName, err = syncManager.TryAcquire(ctx, wfsem, "", wfsem.Spec.Synchronization)
		require.NoError(t, err)
		assert.Empty(t, msg)
		assert.Empty(t, failedLockName)
		assert.True(t, status)
		assert.True(t, wfUpdate)

		syncManager.ReleaseAll(wf1)
		syncManager.ReleaseAll(wfsem)

		// And reacquire in a sem+mutex wf
		status, wfUpdate, msg, failedLockName, err = syncManager.TryAcquire(ctx, wfmands1copy, "", wfmands1copy.Spec.Synchronization)
		require.NoError(t, err)
		assert.Empty(t, msg)
		assert.Empty(t, failedLockName)
		assert.True(t, status)
		assert.True(t, wfUpdate)

	})
}
func TestPriority(t *testing.T) {
	ctx := context.Background()
	kube := fake.NewSimpleClientset()
	syncLimitFunc := GetSyncLimitFunc(kube)
	t.Run("Priority", func(t *testing.T) {
		syncManager := NewLockManager(syncLimitFunc, func(key string) {},
			WorkflowExistenceFunc)
		wflow := templatedWorkflow("prioritylow",
			`    mutexes:
       - name: one
       - name: two
`)
		wfhigh := wflow.DeepCopy()
		wfhigh.Name = "priorityhigh"
		wfhigh.Spec.Priority = ptr.To(int32(5))
		wf1 := templatedWorkflow("one",
			`    mutexes:
       - name: two
`)
		// wf2 takes mutex two
		wf2 := templatedWorkflow("two",
			`    mutexes:
       - name: two
`)
		// Acquire 1 + 2 as low
		status, wfUpdate, msg, failedLockName, err := syncManager.TryAcquire(ctx, wflow, "", wflow.Spec.Synchronization)
		require.NoError(t, err)
		assert.Empty(t, msg)
		assert.Empty(t, failedLockName)
		assert.True(t, status)
		assert.True(t, wfUpdate)

		// Attempt to acquire 2, fail
		status, wfUpdate, msg, failedLockName, err = syncManager.TryAcquire(ctx, wf1, "", wf1.Spec.Synchronization)
		require.NoError(t, err)
		assert.NotEmpty(t, msg)
		assert.Equal(t, "default/Mutex/two", failedLockName)
		assert.False(t, status)
		assert.True(t, wfUpdate)

		// Attempt get 1 + 2 as high but fail
		status, wfUpdate, msg, failedLockName, err = syncManager.TryAcquire(ctx, wfhigh, "", wfhigh.Spec.Synchronization)
		require.NoError(t, err)
		assert.NotEmpty(t, msg)
		assert.Equal(t, "default/Mutex/one", failedLockName)
		assert.False(t, status)
		assert.True(t, wfUpdate)

		// Attempt to acquire 2 again as two, fail
		status, wfUpdate, msg, failedLockName, err = syncManager.TryAcquire(ctx, wf2, "", wf2.Spec.Synchronization)
		require.NoError(t, err)
		assert.NotEmpty(t, msg)
		assert.Equal(t, "default/Mutex/two", failedLockName)
		assert.False(t, status)
		assert.True(t, wfUpdate)

		// Release locks
		syncManager.ReleaseAll(wflow)

		// Attempt to acquire 2 again, but priority blocks
		status, wfUpdate, msg, failedLockName, err = syncManager.TryAcquire(ctx, wf1, "", wf1.Spec.Synchronization)
		require.NoError(t, err)
		assert.NotEmpty(t, msg)
		assert.Equal(t, "default/Mutex/two", failedLockName)
		assert.False(t, status)
		assert.False(t, wfUpdate)

		// Attempt get 1 + 2 as high and priority succeeds
		status, wfUpdate, msg, failedLockName, err = syncManager.TryAcquire(ctx, wfhigh, "", wfhigh.Spec.Synchronization)
		require.NoError(t, err)
		assert.Empty(t, msg)
		assert.Empty(t, failedLockName)
		assert.True(t, status)
		assert.True(t, wfUpdate)
	})
}

func TestDuplicates(t *testing.T) {
	ctx := context.Background()
	kube := fake.NewSimpleClientset()
	syncLimitFunc := GetSyncLimitFunc(kube)
	t.Run("Mutex", func(t *testing.T) {
		syncManager := NewLockManager(syncLimitFunc, func(key string) {},
			WorkflowExistenceFunc)
		wfdupmutex := templatedWorkflow("mutex",
			`    mutexes:
       - name: one
       - name: one
`)
		_, _, _, _, err := syncManager.TryAcquire(ctx, wfdupmutex, "", wfdupmutex.Spec.Synchronization)
		assert.Error(t, err)
	})
	t.Run("Semaphore", func(t *testing.T) {
		syncManager := NewLockManager(syncLimitFunc, func(key string) {},
			WorkflowExistenceFunc)
		wfdupsemaphore := templatedWorkflow("semaphore",
			`    semaphores:
       - configMapKeyRef:
           key: double
           name: my-config
       - configMapKeyRef:
           key: double
           name: my-config
`)
		_, _, _, _, err := syncManager.TryAcquire(ctx, wfdupsemaphore, "", wfdupsemaphore.Spec.Synchronization)
		assert.Error(t, err)
	})
}
