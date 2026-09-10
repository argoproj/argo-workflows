package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"

	argoErr "github.com/argoproj/argo-workflows/v4/errors"
	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/util/logging"
	"github.com/argoproj/argo-workflows/v4/workflow/sync"
)

const configMap = "@testdata/operator_concurrency/semaphore-configmap.yaml"

const wfWithSemaphore = "@testdata/operator_concurrency/wf-with-semaphore.yaml"

const ScriptWfWithSemaphore = "@testdata/operator_concurrency/script-wf-with-semaphore.yaml"

const ScriptWfWithSemaphoreDifferentNamespace = "@testdata/operator_concurrency/script-wf-with-semaphore-different-namespace.yaml"

const ResourceWfWithSemaphore = "@testdata/operator_concurrency/resource-wf-with-semaphore.yaml"

var workflowExistenceFunc = func(key string) bool {
	return true
}

func getSyncLimitFunc(_ context.Context, kube kubernetes.Interface) sync.GetSyncLimit {
	syncLimitConfig := func(ctx context.Context, lockName string) (int, error) {
		items := strings.Split(lockName, "/")
		if len(items) < 4 {
			return 0, argoErr.New(argoErr.CodeBadRequest, "Invalid Config Map Key")
		}

		configMap, err := kube.CoreV1().ConfigMaps(items[0]).Get(ctx, items[2], metav1.GetOptions{})
		if err != nil {
			return 0, err
		}

		value, found := configMap.Data[items[3]]

		if !found {
			return 0, argoErr.New(argoErr.CodeBadRequest, "Invalid Sync configuration Key")
		}
		return strconv.Atoi(value)
	}
	return syncLimitConfig
}

func TestSemaphoreTmplLevel(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx)
	defer cancel()
	controller.syncManager, _ = sync.NewLockManager(ctx, controller.kubeclientset, controller.namespace, nil, getSyncLimitFunc(ctx, controller.kubeclientset), func(key string) {
	}, workflowExistenceFunc, false)
	var cm apiv1.ConfigMap
	wfv1.MustUnmarshal(configMap, &cm)
	_, err := controller.kubeclientset.CoreV1().ConfigMaps("default").Create(ctx, &cm, metav1.CreateOptions{})
	require.NoError(t, err)

	t.Run("TmplLevelAcquireAndRelease", func(t *testing.T) {
		wf := wfv1.MustUnmarshalWorkflow(wfWithSemaphore)
		wf.Name = "one"
		wf, err := controller.wfclientset.ArgoprojV1alpha1().Workflows(wf.Namespace).Create(ctx, wf, metav1.CreateOptions{})
		require.NoError(t, err)
		woc := newWorkflowOperationCtx(ctx, wf, controller)

		// acquired the lock
		woc.operate(ctx)
		assert.NotNil(t, woc.wf.Status.Synchronization)
		assert.NotNil(t, woc.wf.Status.Synchronization.Semaphore)
		assert.Len(t, woc.wf.Status.Synchronization.Semaphore.Holding, 1)

		for _, node := range woc.wf.Status.Nodes {
			assert.Equal(t, wfv1.NodePending, node.Phase)
		}

		// Try to Acquire the lock, But lock is not available
		wfTwo := wf.DeepCopy()
		wfTwo.Name = "two"
		wfTwo, err = controller.wfclientset.ArgoprojV1alpha1().Workflows(wf.Namespace).Create(ctx, wfTwo, metav1.CreateOptions{})
		require.NoError(t, err)
		wocTwo := newWorkflowOperationCtx(ctx, wfTwo, controller)
		// Try Acquire the lock
		wocTwo.operate(ctx)

		// Check Node status
		_, err = wocTwo.podReconciliation(ctx)
		require.NoError(t, err)
		for _, node := range wocTwo.wf.Status.Nodes {
			assert.Equal(t, wfv1.NodePending, node.Phase)
		}

		// Updating Pod state
		makePodsPhase(ctx, woc, apiv1.PodFailed)

		// Release the lock
		woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
		woc.operate(ctx)
		assert.Nil(t, woc.wf.Status.Synchronization)

		// Try to acquired the lock
		wocTwo = newWorkflowOperationCtx(ctx, wocTwo.wf, controller)
		wocTwo.operate(ctx)
		assert.NotNil(t, wocTwo.wf.Status.Synchronization)
		assert.NotNil(t, wocTwo.wf.Status.Synchronization.Semaphore)
		assert.Len(t, wocTwo.wf.Status.Synchronization.Semaphore.Holding, 1)
	})
}

func TestSemaphoreScriptTmplLevel(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx)
	defer cancel()
	controller.syncManager, _ = sync.NewLockManager(ctx, controller.kubeclientset, controller.namespace, nil, getSyncLimitFunc(ctx, controller.kubeclientset), func(key string) {
	}, workflowExistenceFunc, false)
	var cm apiv1.ConfigMap
	wfv1.MustUnmarshal(configMap, &cm)
	_, err := controller.kubeclientset.CoreV1().ConfigMaps("default").Create(ctx, &cm, metav1.CreateOptions{})
	require.NoError(t, err)

	t.Run("ScriptTmplLevelAcquireAndRelease", func(t *testing.T) {
		wf := wfv1.MustUnmarshalWorkflow(ScriptWfWithSemaphore)
		wf.Name = "one"
		wf, err := controller.wfclientset.ArgoprojV1alpha1().Workflows(wf.Namespace).Create(ctx, wf, metav1.CreateOptions{})
		require.NoError(t, err)
		woc := newWorkflowOperationCtx(ctx, wf, controller)

		// acquired the lock
		woc.operate(ctx)
		assert.NotNil(t, woc.wf.Status.Synchronization)
		assert.NotNil(t, woc.wf.Status.Synchronization.Semaphore)
		assert.Len(t, woc.wf.Status.Synchronization.Semaphore.Holding, 1)

		for _, node := range woc.wf.Status.Nodes {
			assert.Equal(t, wfv1.NodePending, node.Phase)
		}

		// Try to Acquire the lock, But lock is not available
		wfTwo := wf.DeepCopy()
		wfTwo.Name = "two"
		wfTwo, err = controller.wfclientset.ArgoprojV1alpha1().Workflows(wf.Namespace).Create(ctx, wfTwo, metav1.CreateOptions{})
		require.NoError(t, err)
		wocTwo := newWorkflowOperationCtx(ctx, wfTwo, controller)
		// Try Acquire the lock
		wocTwo.operate(ctx)

		// Check Node status
		_, err = wocTwo.podReconciliation(ctx)
		require.NoError(t, err)
		for _, node := range wocTwo.wf.Status.Nodes {
			assert.Equal(t, wfv1.NodePending, node.Phase)
		}
		// Updating Pod state
		makePodsPhase(ctx, woc, apiv1.PodFailed)

		// Release the lock
		woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
		woc.operate(ctx)
		assert.Nil(t, woc.wf.Status.Synchronization)

		// Try to acquired the lock
		wocTwo = newWorkflowOperationCtx(ctx, wocTwo.wf, controller)
		wocTwo.operate(ctx)
		assert.NotNil(t, wocTwo.wf.Status.Synchronization)
		assert.NotNil(t, wocTwo.wf.Status.Synchronization.Semaphore)
		assert.Len(t, wocTwo.wf.Status.Synchronization.Semaphore.Holding, 1)
	})
}

func TestSemaphoreScriptConfigMapInDifferentNamespace(t *testing.T) {
	cancel, controller := newController(logging.TestContext(t.Context()))
	defer cancel()
	ctx := logging.TestContext(t.Context())
	controller.syncManager, _ = sync.NewLockManager(ctx, controller.kubeclientset, controller.namespace, nil, getSyncLimitFunc(ctx, controller.kubeclientset), func(key string) {
	}, workflowExistenceFunc, false)
	var cm apiv1.ConfigMap
	wfv1.MustUnmarshal(configMap, &cm)
	_, err := controller.kubeclientset.CoreV1().ConfigMaps("other").Create(ctx, &cm, metav1.CreateOptions{})
	require.NoError(t, err)

	t.Run("ScriptTmplLevelAcquireAndRelease", func(t *testing.T) {
		wf := wfv1.MustUnmarshalWorkflow(ScriptWfWithSemaphoreDifferentNamespace)
		wf.Name = "one"
		wf.Namespace = "namespace-one"
		wf, err := controller.wfclientset.ArgoprojV1alpha1().Workflows(wf.Namespace).Create(ctx, wf, metav1.CreateOptions{})
		require.NoError(t, err)
		woc := newWorkflowOperationCtx(ctx, wf, controller)

		// acquired the lock
		woc.operate(ctx)
		assert.NotNil(t, woc.wf.Status.Synchronization)
		assert.NotNil(t, woc.wf.Status.Synchronization.Semaphore)
		assert.Len(t, woc.wf.Status.Synchronization.Semaphore.Holding, 1)

		for _, node := range woc.wf.Status.Nodes {
			assert.Equal(t, wfv1.NodePending, node.Phase)
		}

		// Try to Acquire the lock, But lock is not available
		wfTwo := wf.DeepCopy()
		wfTwo.Name = "two"
		wfTwo.Namespace = "namespace-two"
		wfTwo, err = controller.wfclientset.ArgoprojV1alpha1().Workflows(wfTwo.Namespace).Create(ctx, wfTwo, metav1.CreateOptions{})
		require.NoError(t, err)
		wocTwo := newWorkflowOperationCtx(ctx, wfTwo, controller)
		// Try Acquire the lock
		wocTwo.operate(ctx)

		// Check Node status
		_, err = wocTwo.podReconciliation(ctx)
		require.NoError(t, err)
		for _, node := range wocTwo.wf.Status.Nodes {
			assert.Equal(t, wfv1.NodePending, node.Phase)
		}
		// Updating Pod state
		makePodsPhase(ctx, woc, apiv1.PodFailed)

		// Release the lock
		woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
		woc.operate(ctx)
		assert.Nil(t, woc.wf.Status.Synchronization)

		// Try to acquired the lock
		wocTwo = newWorkflowOperationCtx(ctx, wocTwo.wf, controller)
		wocTwo.operate(ctx)
		assert.NotNil(t, wocTwo.wf.Status.Synchronization)
		assert.NotNil(t, wocTwo.wf.Status.Synchronization.Semaphore)
		assert.Len(t, wocTwo.wf.Status.Synchronization.Semaphore.Holding, 1)
	})
}

func TestSemaphoreResourceTmplLevel(t *testing.T) {
	cancel, controller := newController(logging.TestContext(t.Context()))
	defer cancel()
	ctx := logging.TestContext(t.Context())
	controller.syncManager, _ = sync.NewLockManager(ctx, controller.kubeclientset, controller.namespace, nil, getSyncLimitFunc(ctx, controller.kubeclientset), func(key string) {
	}, workflowExistenceFunc, false)
	var cm apiv1.ConfigMap
	wfv1.MustUnmarshal(configMap, &cm)
	_, err := controller.kubeclientset.CoreV1().ConfigMaps("default").Create(ctx, &cm, metav1.CreateOptions{})
	require.NoError(t, err)

	t.Run("ResourceTmplLevelAcquireAndRelease", func(t *testing.T) {
		wf := wfv1.MustUnmarshalWorkflow(ResourceWfWithSemaphore)
		wf.Name = "one"
		wf, err := controller.wfclientset.ArgoprojV1alpha1().Workflows(wf.Namespace).Create(ctx, wf, metav1.CreateOptions{})
		require.NoError(t, err)
		woc := newWorkflowOperationCtx(ctx, wf, controller)

		// acquired the lock
		woc.operate(ctx)
		assert.NotNil(t, woc.wf.Status.Synchronization)
		assert.NotNil(t, woc.wf.Status.Synchronization.Semaphore)
		assert.Len(t, woc.wf.Status.Synchronization.Semaphore.Holding, 1)

		for _, node := range woc.wf.Status.Nodes {
			assert.Equal(t, wfv1.NodePending, node.Phase)
		}

		// Try to Acquire the lock, But lock is not available
		wfTwo := wf.DeepCopy()
		wfTwo.Name = "two"
		wfTwo, err = controller.wfclientset.ArgoprojV1alpha1().Workflows(wf.Namespace).Create(ctx, wfTwo, metav1.CreateOptions{})
		require.NoError(t, err)
		wocTwo := newWorkflowOperationCtx(ctx, wfTwo, controller)
		// Try Acquire the lock
		wocTwo.operate(ctx)

		// Check Node status
		_, err = wocTwo.podReconciliation(ctx)
		require.NoError(t, err)
		for _, node := range wocTwo.wf.Status.Nodes {
			assert.Equal(t, wfv1.NodePending, node.Phase)
		}

		// Updating Pod state
		makePodsPhase(ctx, woc, apiv1.PodFailed)

		// Release the lock
		woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
		woc.operate(ctx)
		assert.Nil(t, woc.wf.Status.Synchronization)

		// Try to acquired the lock
		wocTwo = newWorkflowOperationCtx(ctx, wocTwo.wf, controller)
		wocTwo.operate(ctx)
		assert.NotNil(t, wocTwo.wf.Status.Synchronization)
		assert.NotNil(t, wocTwo.wf.Status.Synchronization.Semaphore)
		assert.Len(t, wocTwo.wf.Status.Synchronization.Semaphore.Holding, 1)
	})
}

func TestSemaphoreWithOutConfigMap(t *testing.T) {
	cancel, controller := newController(logging.TestContext(t.Context()))
	defer cancel()

	ctx := logging.TestContext(t.Context())
	controller.syncManager, _ = sync.NewLockManager(ctx, controller.kubeclientset, controller.namespace, nil, getSyncLimitFunc(ctx, controller.kubeclientset), func(key string) {
	}, workflowExistenceFunc, false)

	t.Run("SemaphoreRefWithOutConfigMap", func(t *testing.T) {
		wf := wfv1.MustUnmarshalWorkflow(wfWithSemaphore)
		wf.Name = "one"
		wf, err := controller.wfclientset.ArgoprojV1alpha1().Workflows(wf.Namespace).Create(ctx, wf, metav1.CreateOptions{})
		require.NoError(t, err)
		woc := newWorkflowOperationCtx(ctx, wf, controller)
		_, err = woc.podReconciliation(ctx)
		require.NoError(t, err)
		for _, node := range woc.wf.Status.Nodes {
			assert.Equal(t, wfv1.NodePending, node.Phase)
		}
		// Acquire the lock
		woc.operate(ctx)
		assert.Nil(t, woc.wf.Status.Synchronization)
		for _, node := range woc.wf.Status.Nodes {
			assert.Equal(t, wfv1.NodeError, node.Phase)
		}
	})
}

const DAGWithMutex = "@testdata/operator_concurrency/dag-with-mutex.yaml"

func TestMutexInDAG(t *testing.T) {
	assert := assert.New(t)

	cancel, controller := newController(logging.TestContext(t.Context()))
	defer cancel()
	ctx := logging.TestContext(t.Context())
	controller.syncManager, _ = sync.NewLockManager(ctx, controller.kubeclientset, controller.namespace, nil, getSyncLimitFunc(ctx, controller.kubeclientset), func(key string) {
	}, workflowExistenceFunc, false)
	t.Run("MutexWithDAG", func(t *testing.T) {
		wf := wfv1.MustUnmarshalWorkflow(DAGWithMutex)
		wf, err := controller.wfclientset.ArgoprojV1alpha1().Workflows(wf.Namespace).Create(ctx, wf, metav1.CreateOptions{})
		require.NoError(t, err)
		woc := newWorkflowOperationCtx(ctx, wf, controller)
		woc.operate(ctx)
		for _, node := range woc.wf.Status.Nodes {
			if node.Name == "dag-mutex.A" {
				assert.Equal(wfv1.NodePending, node.Phase)
			}
		}
		assert.Equal(wfv1.WorkflowRunning, woc.wf.Status.Phase)
		makePodsPhase(ctx, woc, apiv1.PodSucceeded)

		woc1 := newWorkflowOperationCtx(ctx, woc.wf, controller)
		woc1.operate(ctx)
		for _, node := range woc1.wf.Status.Nodes {
			if node.Name == "dag-mutex.B" {
				assert.Nil(node.SynchronizationStatus)
				assert.Equal(wfv1.NodePending, node.Phase)
			}
		}
	})
}

const DAGWithInterpolatedMutex = "@testdata/operator_concurrency/dag-with-interpolated-mutex.yaml"

func TestMutexInDAGWithInterpolation(t *testing.T) {
	assert := assert.New(t)

	cancel, controller := newController(logging.TestContext(t.Context()))
	defer cancel()
	ctx := logging.TestContext(t.Context())
	controller.syncManager, _ = sync.NewLockManager(ctx, controller.kubeclientset, controller.namespace, nil, getSyncLimitFunc(ctx, controller.kubeclientset), func(key string) {
	}, workflowExistenceFunc, false)
	t.Run("InterpolatedMutexWithDAG", func(t *testing.T) {
		wf := wfv1.MustUnmarshalWorkflow(DAGWithInterpolatedMutex)
		wf, err := controller.wfclientset.ArgoprojV1alpha1().Workflows(wf.Namespace).Create(ctx, wf, metav1.CreateOptions{})
		require.NoError(t, err)
		woc := newWorkflowOperationCtx(ctx, wf, controller)
		woc.operate(ctx)
		for _, node := range woc.wf.Status.Nodes {
			if node.Name == "dag-mutex.A" {
				assert.Equal(wfv1.NodePending, node.Phase)
			}
		}
		assert.Equal(wfv1.WorkflowRunning, woc.wf.Status.Phase)
		makePodsPhase(ctx, woc, apiv1.PodSucceeded)

		woc1 := newWorkflowOperationCtx(ctx, woc.wf, controller)
		woc1.operate(ctx)
		for _, node := range woc1.wf.Status.Nodes {
			assert.NotEqual(wfv1.NodeError, node.Phase)
			if node.Name == "dag-mutex.B" {
				assert.Nil(node.SynchronizationStatus)
				assert.Equal(wfv1.NodePending, node.Phase)
			}
		}
	})
}

const RetryWfWithSemaphore = "@testdata/operator_concurrency/retry-wf-with-semaphore.yaml"

func TestSynchronizationWithRetry(t *testing.T) {
	assert := assert.New(t)
	cancel, controller := newController(logging.TestContext(t.Context()))
	defer cancel()
	ctx := logging.TestContext(t.Context())
	controller.syncManager, _ = sync.NewLockManager(ctx, controller.kubeclientset, controller.namespace, nil, getSyncLimitFunc(ctx, controller.kubeclientset), func(key string) {
	}, workflowExistenceFunc, false)
	var cm apiv1.ConfigMap
	wfv1.MustUnmarshal(configMap, &cm)
	_, err := controller.kubeclientset.CoreV1().ConfigMaps("default").Create(ctx, &cm, metav1.CreateOptions{})
	require.NoError(t, err)
	t.Run("WorkflowWithRetry", func(t *testing.T) {
		wf := wfv1.MustUnmarshalWorkflow(RetryWfWithSemaphore)
		wf, err := controller.wfclientset.ArgoprojV1alpha1().Workflows(wf.Namespace).Create(ctx, wf, metav1.CreateOptions{})
		require.NoError(t, err)
		woc := newWorkflowOperationCtx(ctx, wf, controller)
		woc.operate(ctx)
		for _, node := range woc.wf.Status.Nodes {
			if node.Name == "hello1" {
				assert.Equal(wfv1.NodePending, node.Phase)
			}
		}

		// Updating Pod state
		makePodsPhase(ctx, woc, apiv1.PodSucceeded)

		// Release the lock from hello1
		woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
		woc.operate(ctx)
		for _, node := range woc.wf.Status.Nodes {
			if node.Name == "hello1" {
				assert.Equal(wfv1.NodeSucceeded, node.Phase)
			}
			if node.Name == "hello2" {
				assert.Equal(wfv1.NodePending, node.Phase)
			}
		}
		// Updating Pod state
		makePodsPhase(ctx, woc, apiv1.PodSucceeded)

		// Release the lock  from hello2
		woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
		woc.operate(ctx)
		// Nobody is waiting for the lock
		assert.Nil(woc.wf.Status.Synchronization)
	})
}

const StepWithSync = "@testdata/operator_concurrency/step-with-sync.yaml"

const StepWithSyncStatus = "@testdata/operator_concurrency/step-with-sync-status.yaml"

func TestSynchronizationWithStep(t *testing.T) {
	assert := assert.New(t)
	cancel, controller := newController(logging.TestContext(t.Context()))
	defer cancel()
	ctx := logging.TestContext(t.Context())
	controller.syncManager, _ = sync.NewLockManager(ctx, controller.kubeclientset, controller.namespace, nil, getSyncLimitFunc(ctx, controller.kubeclientset), func(key string) {
	}, workflowExistenceFunc, false)
	var cm apiv1.ConfigMap
	wfv1.MustUnmarshal(configMap, &cm)
	_, err := controller.kubeclientset.CoreV1().ConfigMaps("default").Create(ctx, &cm, metav1.CreateOptions{})
	require.NoError(t, err)

	t.Run("StepWithSychronization", func(t *testing.T) {
		// First workflow Acquire the lock
		wf := wfv1.MustUnmarshalWorkflow(StepWithSync)
		wf, err := controller.wfclientset.ArgoprojV1alpha1().Workflows("default").Create(ctx, wf, metav1.CreateOptions{})
		require.NoError(t, err)
		woc := newWorkflowOperationCtx(ctx, wf, controller)
		woc.operate(ctx)
		assert.NotNil(woc.wf.Status.Synchronization)
		assert.NotNil(woc.wf.Status.Synchronization.Semaphore)
		assert.Len(woc.wf.Status.Synchronization.Semaphore.Holding, 1)

		// Second workflow try to acquire the lock and wait for lock
		wf1 := wfv1.MustUnmarshalWorkflow(StepWithSync)
		wf1.Name = "step2"
		wf1, err = controller.wfclientset.ArgoprojV1alpha1().Workflows("default").Create(ctx, wf1, metav1.CreateOptions{})
		require.NoError(t, err)
		woc1 := newWorkflowOperationCtx(ctx, wf1, controller)
		woc1.operate(ctx)
		assert.NotNil(woc1.wf.Status.Synchronization)
		assert.NotNil(woc1.wf.Status.Synchronization.Semaphore)
		assert.Nil(woc1.wf.Status.Synchronization.Semaphore.Holding)
		assert.Len(woc1.wf.Status.Synchronization.Semaphore.Waiting, 1)

		// Finished all StepGroup in step
		wf = wfv1.MustUnmarshalWorkflow(StepWithSyncStatus)
		woc = newWorkflowOperationCtx(ctx, wf, controller)
		woc.operate(ctx)
		assert.Nil(woc.wf.Status.Synchronization)

		// Second workflow acquire the lock
		woc1 = newWorkflowOperationCtx(ctx, woc1.wf, controller)
		woc1.operate(ctx)
		assert.NotNil(woc1.wf.Status.Synchronization)
		assert.NotNil(woc1.wf.Status.Synchronization.Semaphore)
		assert.NotNil(woc1.wf.Status.Synchronization.Semaphore.Holding)
		assert.Len(woc1.wf.Status.Synchronization.Semaphore.Holding, 1)
	})
}

const wfWithStepRetry = "@testdata/operator_concurrency/wf-with-step-retry.yaml"

func TestSynchronizationWithStepRetry(t *testing.T) {
	assert := assert.New(t)
	cancel, controller := newController(logging.TestContext(t.Context()))
	defer cancel()
	ctx := logging.TestContext(t.Context())
	controller.syncManager, _ = sync.NewLockManager(ctx, controller.kubeclientset, controller.namespace, nil, getSyncLimitFunc(ctx, controller.kubeclientset), func(key string) {
	}, workflowExistenceFunc, false)
	var cm apiv1.ConfigMap
	wfv1.MustUnmarshal(configMap, &cm)
	_, err := controller.kubeclientset.CoreV1().ConfigMaps("default").Create(ctx, &cm, metav1.CreateOptions{})
	require.NoError(t, err)

	t.Run("StepRetryWithSynchronization", func(t *testing.T) {
		// First workflow Acquire the lock
		wf := wfv1.MustUnmarshalWorkflow(wfWithStepRetry)
		wf, err := controller.wfclientset.ArgoprojV1alpha1().Workflows("default").Create(ctx, wf, metav1.CreateOptions{})
		require.NoError(t, err)
		woc := newWorkflowOperationCtx(ctx, wf, controller)
		woc.operate(ctx)
		for _, n := range woc.wf.Status.Nodes {
			if n.Name == "[0].step1(0)" {
				assert.Equal(wfv1.NodePending, n.Phase)
			}
		}
		// Updating Pod state
		makePodsPhase(ctx, woc, apiv1.PodRunning)

		woc.operate(ctx)
		for _, n := range woc.wf.Status.Nodes {
			if n.Name == "[0].step1(0)" {
				assert.Equal(wfv1.NodeRunning, n.Phase)
			}
		}
		makePodsPhase(ctx, woc, apiv1.PodFailed)
		woc.operate(ctx)
		for _, n := range woc.wf.Status.Nodes {
			if n.Name == "[0].step1(0)" {
				assert.Equal(wfv1.NodeFailed, n.Phase)
			}
			if n.Name == "[0].step1(1)" {
				assert.Equal(wfv1.NodePending, n.Phase)
			}
		}
	})
}

const pendingWfWithShutdownStrategy = "@testdata/operator_concurrency/pending-wf-with-shutdown-strategy.yaml"

func TestSynchronizationForPendingShuttingdownWfs(t *testing.T) {
	cancel, controller := newController(logging.TestContext(t.Context()))
	defer cancel()
	ctx := logging.TestContext(t.Context())
	controller.syncManager, _ = sync.NewLockManager(ctx, controller.kubeclientset, controller.namespace, nil, getSyncLimitFunc(ctx, controller.kubeclientset), func(key string) {
	}, workflowExistenceFunc, false)

	t.Run("PendingShuttingdownTerminatingWf", func(t *testing.T) {
		// Create and acquire the lock for the first workflow
		wf := wfv1.MustUnmarshalWorkflow(pendingWfWithShutdownStrategy)
		wf.Name = "one-terminating"
		wf.Spec.Synchronization.Mutexes[0].Name = "terminating-test"
		wf, err := controller.wfclientset.ArgoprojV1alpha1().Workflows(wf.Namespace).Create(ctx, wf, metav1.CreateOptions{})
		require.NoError(t, err)
		woc := newWorkflowOperationCtx(ctx, wf, controller)
		woc.operate(ctx)
		assert.NotNil(t, woc.wf.Status.Synchronization)
		assert.NotNil(t, woc.wf.Status.Synchronization.Mutex)
		assert.Len(t, woc.wf.Status.Synchronization.Mutex.Holding, 1)

		// Create the second workflow and try to acquire the lock, which should not be available.
		wfTwo := wf.DeepCopy()
		wfTwo.Name = "two-terminating"
		wfTwo, err = controller.wfclientset.ArgoprojV1alpha1().Workflows(wf.Namespace).Create(ctx, wfTwo, metav1.CreateOptions{})
		require.NoError(t, err)
		// This workflow should be pending since the first workflow still holds the lock.
		wocTwo := newWorkflowOperationCtx(ctx, wfTwo, controller)
		wocTwo.operate(ctx)
		assert.Equal(t, wfv1.WorkflowPending, wocTwo.wf.Status.Phase)

		// Shutdown the second workflow that's pending.
		patchObj := map[string]any{
			"spec": map[string]any{
				"shutdown": wfv1.ShutdownStrategyTerminate,
			},
		}
		patch, err := json.Marshal(patchObj)
		require.NoError(t, err)
		wfTwo, err = controller.wfclientset.ArgoprojV1alpha1().Workflows(wf.Namespace).Patch(ctx, wfTwo.Name, types.MergePatchType, patch, metav1.PatchOptions{})
		require.NoError(t, err)

		// The pending workflow that's being shutdown should have failed and released the lock.
		wocTwo = newWorkflowOperationCtx(ctx, wfTwo, controller)
		wocTwo.operate(ctx)
		assert.Equal(t, wfv1.WorkflowFailed, wocTwo.wf.Status.Phase)
		assert.Equal(t, "Stopped with strategy 'Terminate'", wocTwo.wf.Status.Message)
		assert.Nil(t, wocTwo.wf.Status.Synchronization)
		// The workflow never ran, so no nodes should have been created for it.
		assert.Empty(t, wocTwo.wf.Status.Nodes)

		// Release the lock from the first workflow.
		woc.wf.Status.Phase = wfv1.WorkflowSucceeded
		woc.operate(ctx)
		assert.Nil(t, woc.wf.Status.Synchronization)

		// The terminated workflow must also have been removed from the lock's
		// waiting queue, so a new workflow acquires the lock immediately.
		wfThree := wf.DeepCopy()
		wfThree.Name = "three-terminating"
		wfThree, err = controller.wfclientset.ArgoprojV1alpha1().Workflows(wf.Namespace).Create(ctx, wfThree, metav1.CreateOptions{})
		require.NoError(t, err)
		wocThree := newWorkflowOperationCtx(ctx, wfThree, controller)
		wocThree.operate(ctx)
		assert.Equal(t, wfv1.WorkflowRunning, wocThree.wf.Status.Phase)
		require.NotNil(t, wocThree.wf.Status.Synchronization)
		require.NotNil(t, wocThree.wf.Status.Synchronization.Mutex)
		assert.Len(t, wocThree.wf.Status.Synchronization.Mutex.Holding, 1)
	})

	t.Run("PendingShuttingdownStoppingWf", func(t *testing.T) {
		if githubActions, ok := os.LookupEnv(`GITHUB_ACTIONS`); ok && githubActions == "true" {
			t.Skip("This test regularly fails in Github Actions CI")
		}
		// Create and acquire the lock for the first workflow
		wf := wfv1.MustUnmarshalWorkflow(pendingWfWithShutdownStrategy)
		wf.Name = "one-stopping"
		wf.Spec.Synchronization.Mutexes[0].Name = "stopping-test"
		wf, err := controller.wfclientset.ArgoprojV1alpha1().Workflows(wf.Namespace).Create(ctx, wf, metav1.CreateOptions{})
		require.NoError(t, err)
		woc := newWorkflowOperationCtx(ctx, wf, controller)
		woc.operate(ctx)
		assert.NotNil(t, woc.wf.Status.Synchronization)
		assert.NotNil(t, woc.wf.Status.Synchronization.Mutex)
		assert.Len(t, woc.wf.Status.Synchronization.Mutex.Holding, 1)

		// Create the second workflow and try to acquire the lock, which should not be available.
		wfTwo := wf.DeepCopy()
		wfTwo.Name = "two-stopping"
		wfTwo, err = controller.wfclientset.ArgoprojV1alpha1().Workflows(wf.Namespace).Create(ctx, wfTwo, metav1.CreateOptions{})
		require.NoError(t, err)
		// This workflow should be pending since the first workflow still holds the lock.
		wocTwo := newWorkflowOperationCtx(ctx, wfTwo, controller)
		wocTwo.operate(ctx)
		assert.Equal(t, wfv1.WorkflowPending, wocTwo.wf.Status.Phase)

		// Shutdown the second workflow that's pending.
		patchObj := map[string]any{
			"spec": map[string]any{
				"shutdown": wfv1.ShutdownStrategyStop,
			},
		}
		patch, err := json.Marshal(patchObj)
		require.NoError(t, err)
		wfTwo, err = controller.wfclientset.ArgoprojV1alpha1().Workflows(wf.Namespace).Patch(ctx, wfTwo.Name, types.MergePatchType, patch, metav1.PatchOptions{})
		require.NoError(t, err)

		// The pending workflow that's being shutdown should still be pending and waiting to acquire the lock.
		wocTwo = newWorkflowOperationCtx(ctx, wfTwo, controller)
		wocTwo.operate(ctx)
		assert.Equal(t, wfv1.WorkflowPending, wocTwo.execWf.Status.Phase)
		assert.NotNil(t, wocTwo.wf.Status.Synchronization)
		assert.NotNil(t, wocTwo.wf.Status.Synchronization.Mutex)
		assert.Len(t, wocTwo.wf.Status.Synchronization.Mutex.Waiting, 1)

		// Mark the first workflow as succeeded
		woc.wf.Status.Phase = wfv1.WorkflowSucceeded
		woc.operate(ctx)
		assert.Nil(t, woc.wf.Status.Synchronization)
		// The pending workflow should now be running normally
		wocTwo.operate(ctx)
		assert.Equal(t, wfv1.WorkflowRunning, wocTwo.execWf.Status.Phase)
	})
}

func TestWorkflowMemoizationWithMutex(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow("@testdata/operator_concurrency/wf-memoization-with-mutex.yaml")
	wf.Name = "example-steps-simple-gas12"
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx, wf)
	defer cancel()

	woc := newWorkflowOperationCtx(ctx, wf, controller)
	woc.operate(ctx)

	holdingJobs := make(map[string]string)
	for _, node := range woc.wf.Status.Nodes {
		holdingJobs[fmt.Sprintf("%s/%s/%s", wf.Namespace, wf.Name, node.ID)] = node.DisplayName
	}

	// Check initial status: job-1 acquired the lock
	job1AcquiredLock := false
	if woc.wf.Status.Synchronization != nil && woc.wf.Status.Synchronization.Mutex != nil {
		for _, holding := range woc.wf.Status.Synchronization.Mutex.Holding {
			if holdingJobs[holding.Holder] == "job-1" {
				fmt.Println("acquired: ", holding.Holder)
				job1AcquiredLock = true
			}
		}
	}
	assert.True(t, job1AcquiredLock)

	// Make job-1's pod succeed
	makePodsPhase(ctx, woc, apiv1.PodSucceeded, func(pod *apiv1.Pod, _ *wfOperationCtx) {
		if pod.Name == "job-1" {
			pod.Status.Phase = apiv1.PodSucceeded
		}
	})
	woc.operate(ctx)

	// Check final status: both job-1 and job-2 succeeded, job-2 simply hit the cache
	for _, node := range woc.wf.Status.Nodes {
		switch node.DisplayName {
		case "job-1":
			assert.Equal(t, wfv1.NodeSucceeded, node.Phase)
			assert.False(t, node.MemoizationStatus.Hit)
		case "job-2":
			assert.Equal(t, wfv1.NodeSucceeded, node.Phase)
			assert.True(t, node.MemoizationStatus.Hit)
		}
	}
}

const bareWfWithTmplMutex = "@testdata/operator_concurrency/bare-wf-with-tmpl-mutex.yaml"

const stepsWfWithTmplMutex = "@testdata/operator_concurrency/steps-wf-with-tmpl-mutex.yaml"

const dagWfWithTmplMutex = "@testdata/operator_concurrency/dag-wf-with-tmpl-mutex.yaml"

// TestShutdownWaitingForTmplLevelLock ensures that shutting down a workflow
// whose node is still waiting to acquire a template-level synchronization lock
// fails that node immediately instead of silently ignoring the shutdown until
// the lock becomes available.
func TestShutdownWaitingForTmplLevelLock(t *testing.T) {
	tests := []struct {
		name       string
		waiterYAML string
		strategy   wfv1.ShutdownStrategy
	}{
		{"BareTerminate", bareWfWithTmplMutex, wfv1.ShutdownStrategyTerminate},
		{"StepsTerminate", stepsWfWithTmplMutex, wfv1.ShutdownStrategyTerminate},
		{"DAGTerminate", dagWfWithTmplMutex, wfv1.ShutdownStrategyTerminate},
		{"BareStop", bareWfWithTmplMutex, wfv1.ShutdownStrategyStop},
		{"StepsStop", stepsWfWithTmplMutex, wfv1.ShutdownStrategyStop},
		{"DAGStop", dagWfWithTmplMutex, wfv1.ShutdownStrategyStop},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := logging.TestContext(t.Context())
			cancel, controller := newController(ctx)
			defer cancel()
			controller.syncManager, _ = sync.NewLockManager(ctx, controller.kubeclientset, controller.namespace, nil, getSyncLimitFunc(ctx, controller.kubeclientset), func(key string) {
			}, workflowExistenceFunc, false)
			mutexName := "tmpl-shutdown-" + strings.ToLower(tt.name)

			// The holder acquires the mutex and keeps it for the duration of the test.
			holder := wfv1.MustUnmarshalWorkflow(bareWfWithTmplMutex)
			holder.Name = "holder-" + strings.ToLower(tt.name)
			holder.Spec.Templates[0].Synchronization.Mutexes[0].Name = mutexName
			holder, err := controller.wfclientset.ArgoprojV1alpha1().Workflows(holder.Namespace).Create(ctx, holder, metav1.CreateOptions{})
			require.NoError(t, err)
			wocHolder := newWorkflowOperationCtx(ctx, holder, controller)
			wocHolder.operate(ctx)
			require.NotNil(t, wocHolder.wf.Status.Synchronization)
			require.Len(t, wocHolder.wf.Status.Synchronization.Mutex.Holding, 1)

			// The waiter's node blocks waiting for the mutex.
			waiter := wfv1.MustUnmarshalWorkflow(tt.waiterYAML)
			waiter.Name = "waiter-" + strings.ToLower(tt.name)
			for i := range waiter.Spec.Templates {
				if waiter.Spec.Templates[i].Synchronization != nil {
					waiter.Spec.Templates[i].Synchronization.Mutexes[0].Name = mutexName
				}
			}
			waiter, err = controller.wfclientset.ArgoprojV1alpha1().Workflows(waiter.Namespace).Create(ctx, waiter, metav1.CreateOptions{})
			require.NoError(t, err)
			wocWaiter := newWorkflowOperationCtx(ctx, waiter, controller)
			wocWaiter.operate(ctx)
			waitingNode := findWaitingSyncNode(wocWaiter.wf)
			require.NotNil(t, waitingNode, "expected a node waiting for the lock")
			require.Equal(t, wfv1.NodePending, waitingNode.Phase)

			// Shut the waiter down while it is still waiting for the lock.
			patch, err := json.Marshal(map[string]any{"spec": map[string]any{"shutdown": tt.strategy}})
			require.NoError(t, err)
			// Persist the waiter's status from the first operate before patching.
			wf, err := controller.wfclientset.ArgoprojV1alpha1().Workflows(waiter.Namespace).Get(ctx, waiter.Name, metav1.GetOptions{})
			require.NoError(t, err)
			wf.Status = wocWaiter.wf.Status
			wf, err = controller.wfclientset.ArgoprojV1alpha1().Workflows(wf.Namespace).Update(ctx, wf, metav1.UpdateOptions{})
			require.NoError(t, err)
			wf, err = controller.wfclientset.ArgoprojV1alpha1().Workflows(wf.Namespace).Patch(ctx, wf.Name, types.MergePatchType, patch, metav1.PatchOptions{})
			require.NoError(t, err)

			wocWaiter = newWorkflowOperationCtx(ctx, wf, controller)
			wocWaiter.operate(ctx)

			message := fmt.Sprintf("Stopped with strategy '%s'", tt.strategy)
			node := findWaitingSyncNodeByID(wocWaiter.wf, waitingNode.ID)
			require.NotNil(t, node)
			assert.Equal(t, wfv1.NodeFailed, node.Phase)
			assert.Equal(t, message, node.Message)
			// No node may be left behind unfulfilled: the shutdown must
			// propagate to the whole tree.
			for _, n := range wocWaiter.wf.Status.Nodes {
				assert.NotEqual(t, wfv1.NodePending, n.Phase, "node %s left pending after shutdown", n.Name)
			}
			assert.Equal(t, wfv1.WorkflowFailed, wocWaiter.wf.Status.Phase)
		})
	}
}

func findWaitingSyncNode(wf *wfv1.Workflow) *wfv1.NodeStatus {
	for _, node := range wf.Status.Nodes {
		if node.SynchronizationStatus != nil && node.SynchronizationStatus.Waiting != "" {
			return &node
		}
	}
	return nil
}

func findWaitingSyncNodeByID(wf *wfv1.Workflow, id string) *wfv1.NodeStatus {
	for _, node := range wf.Status.Nodes {
		if node.ID == id {
			return &node
		}
	}
	return nil
}
