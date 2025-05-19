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

	argoErr "github.com/argoproj/argo-workflows/v3/errors"
	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v3/workflow/sync"
)

const configMap = `
apiVersion: v1
kind: ConfigMap
metadata:
  name: my-config
data:
  workflow: "2"
  template: "1"
  step: "1"
`

const wfWithSemaphore = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: hello-world
  namespace: default
spec:
  entrypoint: whalesay
  templates:
    -
      synchronization:
        semaphores:
          - configMapKeyRef:
              key: template
              name: my-config
      container:
        args:
          - "hello world"
        command:
          - cowsay
        image: "docker/whalesay:latest"
      name: whalesay
`

const ScriptWfWithSemaphore = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: script-wf
  namespace: default
spec:
  entrypoint: scriptTmpl
  templates:
  - name: scriptTmpl
    synchronization:
      semaphores:
        - configMapKeyRef:
            key: template
            name: my-config
    script:
      image: python:alpine3.6
      command: ["python"]
      # fail with a 66% probability
      source: |
        import random;
        import sys;
        exit_code = random.choice([0, 1, 1]);
        sys.exit(exit_code)
`

const ScriptWfWithSemaphoreDifferentNamespace = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: script-wf
  namespace: default
spec:
  entrypoint: scriptTmpl
  templates:
  - name: scriptTmpl
    synchronization:
      semaphores:
        - namespace: other
          configMapKeyRef:
            key: template
            name: my-config
    script:
      image: python:alpine3.6
      command: ["python"]
      # fail with a 66% probability
      source: |
        import random;
        import sys;
        exit_code = random.choice([0, 1, 1]);
        sys.exit(exit_code)
`

const ResourceWfWithSemaphore = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: resource-wf
  namespace: default
spec:
  entrypoint: resourceTmpl
  templates:
  - name: resourceTmpl
    synchronization:
      semaphores:
        - configMapKeyRef:
            key: template
            name: my-config
    resource:
      action: create
      manifest: |
        apiVersion: v1
        kind: ConfigMap
        metadata:
          name: workflow-controller-configmap1
`

var workflowExistenceFunc = func(key string) bool {
	return true
}

func getSyncLimitFunc(ctx context.Context, kube kubernetes.Interface) func(string) (int, error) {
	syncLimitConfig := func(lockName string) (int, error) {
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
	cancel, controller := newController()
	defer cancel()
	ctx := context.Background()
	controller.syncManager = sync.NewLockManager(ctx, controller.kubeclientset, controller.namespace, nil, getSyncLimitFunc(ctx, controller.kubeclientset), func(key string) {
	}, workflowExistenceFunc)
	var cm apiv1.ConfigMap
	wfv1.MustUnmarshal([]byte(configMap), &cm)
	_, err := controller.kubeclientset.CoreV1().ConfigMaps("default").Create(ctx, &cm, metav1.CreateOptions{})
	require.NoError(t, err)

	t.Run("TmplLevelAcquireAndRelease", func(t *testing.T) {
		wf := wfv1.MustUnmarshalWorkflow(wfWithSemaphore)
		wf.Name = "one"
		wf, err := controller.wfclientset.ArgoprojV1alpha1().Workflows(wf.Namespace).Create(ctx, wf, metav1.CreateOptions{})
		require.NoError(t, err)
		woc := newWorkflowOperationCtx(wf, controller)

		// acquired the lock
		woc.operate(ctx)
		assert.NotNil(t, woc.wf.Status.Synchronization)
		assert.NotNil(t, woc.wf.Status.Synchronization.Semaphore)
		assert.Len(t, woc.wf.Status.Synchronization.Semaphore.Holding, 1)

		for _, node := range woc.wf.Status.Nodes {
			assert.Equal(t, wfv1.NodePending, node.Phase)
		}

		// Try to Acquire the lock, But lock is not available
		wf_Two := wf.DeepCopy()
		wf_Two.Name = "two"
		wf_Two, err = controller.wfclientset.ArgoprojV1alpha1().Workflows(wf.Namespace).Create(ctx, wf_Two, metav1.CreateOptions{})
		require.NoError(t, err)
		woc_two := newWorkflowOperationCtx(wf_Two, controller)
		// Try Acquire the lock
		woc_two.operate(ctx)

		// Check Node status
		err, _ = woc_two.podReconciliation(ctx)
		require.NoError(t, err)
		for _, node := range woc_two.wf.Status.Nodes {
			assert.Equal(t, wfv1.NodePending, node.Phase)
		}

		// Updating Pod state
		makePodsPhase(ctx, woc, apiv1.PodFailed)

		// Release the lock
		woc = newWorkflowOperationCtx(woc.wf, controller)
		woc.operate(ctx)
		assert.Nil(t, woc.wf.Status.Synchronization)

		// Try to acquired the lock
		woc_two = newWorkflowOperationCtx(woc_two.wf, controller)
		woc_two.operate(ctx)
		assert.NotNil(t, woc_two.wf.Status.Synchronization)
		assert.NotNil(t, woc_two.wf.Status.Synchronization.Semaphore)
		assert.Len(t, woc_two.wf.Status.Synchronization.Semaphore.Holding, 1)
	})
}

func TestSemaphoreScriptTmplLevel(t *testing.T) {
	cancel, controller := newController()
	defer cancel()
	ctx := context.Background()
	controller.syncManager = sync.NewLockManager(ctx, controller.kubeclientset, controller.namespace, nil, getSyncLimitFunc(ctx, controller.kubeclientset), func(key string) {
	}, workflowExistenceFunc)
	var cm apiv1.ConfigMap
	wfv1.MustUnmarshal([]byte(configMap), &cm)
	_, err := controller.kubeclientset.CoreV1().ConfigMaps("default").Create(ctx, &cm, metav1.CreateOptions{})
	require.NoError(t, err)

	t.Run("ScriptTmplLevelAcquireAndRelease", func(t *testing.T) {
		wf := wfv1.MustUnmarshalWorkflow(ScriptWfWithSemaphore)
		wf.Name = "one"
		wf, err := controller.wfclientset.ArgoprojV1alpha1().Workflows(wf.Namespace).Create(ctx, wf, metav1.CreateOptions{})
		require.NoError(t, err)
		woc := newWorkflowOperationCtx(wf, controller)

		// acquired the lock
		woc.operate(ctx)
		assert.NotNil(t, woc.wf.Status.Synchronization)
		assert.NotNil(t, woc.wf.Status.Synchronization.Semaphore)
		assert.Len(t, woc.wf.Status.Synchronization.Semaphore.Holding, 1)

		for _, node := range woc.wf.Status.Nodes {
			assert.Equal(t, wfv1.NodePending, node.Phase)
		}

		// Try to Acquire the lock, But lock is not available
		wf_Two := wf.DeepCopy()
		wf_Two.Name = "two"
		wf_Two, err = controller.wfclientset.ArgoprojV1alpha1().Workflows(wf.Namespace).Create(ctx, wf_Two, metav1.CreateOptions{})
		require.NoError(t, err)
		woc_two := newWorkflowOperationCtx(wf_Two, controller)
		// Try Acquire the lock
		woc_two.operate(ctx)

		// Check Node status
		err, _ = woc_two.podReconciliation(ctx)
		require.NoError(t, err)
		for _, node := range woc_two.wf.Status.Nodes {
			assert.Equal(t, wfv1.NodePending, node.Phase)
		}
		// Updating Pod state
		makePodsPhase(ctx, woc, apiv1.PodFailed)

		// Release the lock
		woc = newWorkflowOperationCtx(woc.wf, controller)
		woc.operate(ctx)
		assert.Nil(t, woc.wf.Status.Synchronization)

		// Try to acquired the lock
		woc_two = newWorkflowOperationCtx(woc_two.wf, controller)
		woc_two.operate(ctx)
		assert.NotNil(t, woc_two.wf.Status.Synchronization)
		assert.NotNil(t, woc_two.wf.Status.Synchronization.Semaphore)
		assert.Len(t, woc_two.wf.Status.Synchronization.Semaphore.Holding, 1)
	})
}

func TestSemaphoreScriptConfigMapInDifferentNamespace(t *testing.T) {
	cancel, controller := newController()
	defer cancel()
	ctx := context.Background()
	controller.syncManager = sync.NewLockManager(ctx, controller.kubeclientset, controller.namespace, nil, getSyncLimitFunc(ctx, controller.kubeclientset), func(key string) {
	}, workflowExistenceFunc)
	var cm apiv1.ConfigMap
	wfv1.MustUnmarshal([]byte(configMap), &cm)
	_, err := controller.kubeclientset.CoreV1().ConfigMaps("other").Create(ctx, &cm, metav1.CreateOptions{})
	require.NoError(t, err)

	t.Run("ScriptTmplLevelAcquireAndRelease", func(t *testing.T) {
		wf := wfv1.MustUnmarshalWorkflow(ScriptWfWithSemaphoreDifferentNamespace)
		wf.Name = "one"
		wf.Namespace = "namespace-one"
		wf, err := controller.wfclientset.ArgoprojV1alpha1().Workflows(wf.Namespace).Create(ctx, wf, metav1.CreateOptions{})
		require.NoError(t, err)
		woc := newWorkflowOperationCtx(wf, controller)

		// acquired the lock
		woc.operate(ctx)
		assert.NotNil(t, woc.wf.Status.Synchronization)
		assert.NotNil(t, woc.wf.Status.Synchronization.Semaphore)
		assert.Len(t, woc.wf.Status.Synchronization.Semaphore.Holding, 1)

		for _, node := range woc.wf.Status.Nodes {
			assert.Equal(t, wfv1.NodePending, node.Phase)
		}

		// Try to Acquire the lock, But lock is not available
		wf_Two := wf.DeepCopy()
		wf_Two.Name = "two"
		wf_Two.Namespace = "namespace-two"
		wf_Two, err = controller.wfclientset.ArgoprojV1alpha1().Workflows(wf_Two.Namespace).Create(ctx, wf_Two, metav1.CreateOptions{})
		require.NoError(t, err)
		woc_two := newWorkflowOperationCtx(wf_Two, controller)
		// Try Acquire the lock
		woc_two.operate(ctx)

		// Check Node status
		err, _ = woc_two.podReconciliation(ctx)
		require.NoError(t, err)
		for _, node := range woc_two.wf.Status.Nodes {
			assert.Equal(t, wfv1.NodePending, node.Phase)
		}
		// Updating Pod state
		makePodsPhase(ctx, woc, apiv1.PodFailed)

		// Release the lock
		woc = newWorkflowOperationCtx(woc.wf, controller)
		woc.operate(ctx)
		assert.Nil(t, woc.wf.Status.Synchronization)

		// Try to acquired the lock
		woc_two = newWorkflowOperationCtx(woc_two.wf, controller)
		woc_two.operate(ctx)
		assert.NotNil(t, woc_two.wf.Status.Synchronization)
		assert.NotNil(t, woc_two.wf.Status.Synchronization.Semaphore)
		assert.Len(t, woc_two.wf.Status.Synchronization.Semaphore.Holding, 1)
	})
}

func TestSemaphoreResourceTmplLevel(t *testing.T) {
	cancel, controller := newController()
	defer cancel()
	ctx := context.Background()
	controller.syncManager = sync.NewLockManager(ctx, controller.kubeclientset, controller.namespace, nil, getSyncLimitFunc(ctx, controller.kubeclientset), func(key string) {
	}, workflowExistenceFunc)
	var cm apiv1.ConfigMap
	wfv1.MustUnmarshal([]byte(configMap), &cm)
	_, err := controller.kubeclientset.CoreV1().ConfigMaps("default").Create(ctx, &cm, metav1.CreateOptions{})
	require.NoError(t, err)

	t.Run("ResourceTmplLevelAcquireAndRelease", func(t *testing.T) {
		wf := wfv1.MustUnmarshalWorkflow(ResourceWfWithSemaphore)
		wf.Name = "one"
		wf, err := controller.wfclientset.ArgoprojV1alpha1().Workflows(wf.Namespace).Create(ctx, wf, metav1.CreateOptions{})
		require.NoError(t, err)
		woc := newWorkflowOperationCtx(wf, controller)

		// acquired the lock
		woc.operate(ctx)
		assert.NotNil(t, woc.wf.Status.Synchronization)
		assert.NotNil(t, woc.wf.Status.Synchronization.Semaphore)
		assert.Len(t, woc.wf.Status.Synchronization.Semaphore.Holding, 1)

		for _, node := range woc.wf.Status.Nodes {
			assert.Equal(t, wfv1.NodePending, node.Phase)
		}

		// Try to Acquire the lock, But lock is not available
		wf_Two := wf.DeepCopy()
		wf_Two.Name = "two"
		wf_Two, err = controller.wfclientset.ArgoprojV1alpha1().Workflows(wf.Namespace).Create(ctx, wf_Two, metav1.CreateOptions{})
		require.NoError(t, err)
		woc_two := newWorkflowOperationCtx(wf_Two, controller)
		// Try Acquire the lock
		woc_two.operate(ctx)

		// Check Node status
		err, _ = woc_two.podReconciliation(ctx)
		require.NoError(t, err)
		for _, node := range woc_two.wf.Status.Nodes {
			assert.Equal(t, wfv1.NodePending, node.Phase)
		}

		// Updating Pod state
		makePodsPhase(ctx, woc, apiv1.PodFailed)

		// Release the lock
		woc = newWorkflowOperationCtx(woc.wf, controller)
		woc.operate(ctx)
		assert.Nil(t, woc.wf.Status.Synchronization)

		// Try to acquired the lock
		woc_two = newWorkflowOperationCtx(woc_two.wf, controller)
		woc_two.operate(ctx)
		assert.NotNil(t, woc_two.wf.Status.Synchronization)
		assert.NotNil(t, woc_two.wf.Status.Synchronization.Semaphore)
		assert.Len(t, woc_two.wf.Status.Synchronization.Semaphore.Holding, 1)
	})
}

func TestSemaphoreWithOutConfigMap(t *testing.T) {
	cancel, controller := newController()
	defer cancel()

	ctx := context.Background()
	controller.syncManager = sync.NewLockManager(ctx, controller.kubeclientset, controller.namespace, nil, getSyncLimitFunc(ctx, controller.kubeclientset), func(key string) {
	}, workflowExistenceFunc)

	t.Run("SemaphoreRefWithOutConfigMap", func(t *testing.T) {
		wf := wfv1.MustUnmarshalWorkflow(wfWithSemaphore)
		wf.Name = "one"
		wf, err := controller.wfclientset.ArgoprojV1alpha1().Workflows(wf.Namespace).Create(ctx, wf, metav1.CreateOptions{})
		require.NoError(t, err)
		woc := newWorkflowOperationCtx(wf, controller)
		err, _ = woc.podReconciliation(ctx)
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

var DAGWithMutex = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
 name: dag-mutex
 namespace: default
spec:
 entrypoint: diamond
 templates:
 - name: diamond
   dag:
     tasks:
     - name: A
       template: mutex
     - name: B
       depends: A
       template: mutex

 - name: mutex
   synchronization:
     mutexes:
       - name: welcome
   container:
     image: alpine:3.7
     command: [sh, -c, "exit 0"]
`

func TestMutexInDAG(t *testing.T) {
	assert := assert.New(t)

	cancel, controller := newController()
	defer cancel()
	ctx := context.Background()
	controller.syncManager = sync.NewLockManager(ctx, controller.kubeclientset, controller.namespace, nil, getSyncLimitFunc(ctx, controller.kubeclientset), func(key string) {
	}, workflowExistenceFunc)
	t.Run("MutexWithDAG", func(t *testing.T) {
		wf := wfv1.MustUnmarshalWorkflow(DAGWithMutex)
		wf, err := controller.wfclientset.ArgoprojV1alpha1().Workflows(wf.Namespace).Create(ctx, wf, metav1.CreateOptions{})
		require.NoError(t, err)
		woc := newWorkflowOperationCtx(wf, controller)
		woc.operate(ctx)
		for _, node := range woc.wf.Status.Nodes {
			if node.Name == "dag-mutex.A" {
				assert.Equal(wfv1.NodePending, node.Phase)
			}
		}
		assert.Equal(wfv1.WorkflowRunning, woc.wf.Status.Phase)
		makePodsPhase(ctx, woc, apiv1.PodSucceeded)

		woc1 := newWorkflowOperationCtx(woc.wf, controller)
		woc1.operate(ctx)
		for _, node := range woc1.wf.Status.Nodes {
			if node.Name == "dag-mutex.B" {
				assert.Nil(node.SynchronizationStatus)
				assert.Equal(wfv1.NodePending, node.Phase)
			}
		}
	})
}

var DAGWithInterpolatedMutex = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
 name: dag-mutex
 namespace: default
spec:
 entrypoint: diamond
 templates:
 - name: diamond
   dag:
     tasks:
     - name: A
       template: mutex
       arguments:
         parameters:
         - name: message
           value: foo/bar

     - name: B
       depends: A
       template: mutex
       arguments:
         parameters:
         - name: message
           value: foo/bar

 - name: mutex
   synchronization:
     mutexes:
       - name: '{{=sprig.replace("/", "-", inputs.parameters.message)}}'
   inputs:
     parameters:
     - name: message
   container:
     image: alpine:3.7
     command: [sh, -c, "echo {{inputs.parameters.message}}"]
`

func TestMutexInDAGWithInterpolation(t *testing.T) {
	assert := assert.New(t)

	cancel, controller := newController()
	defer cancel()
	ctx := context.Background()
	controller.syncManager = sync.NewLockManager(ctx, controller.kubeclientset, controller.namespace, nil, getSyncLimitFunc(ctx, controller.kubeclientset), func(key string) {
	}, workflowExistenceFunc)
	t.Run("InterpolatedMutexWithDAG", func(t *testing.T) {
		wf := wfv1.MustUnmarshalWorkflow(DAGWithInterpolatedMutex)
		wf, err := controller.wfclientset.ArgoprojV1alpha1().Workflows(wf.Namespace).Create(ctx, wf, metav1.CreateOptions{})
		require.NoError(t, err)
		woc := newWorkflowOperationCtx(wf, controller)
		woc.operate(ctx)
		for _, node := range woc.wf.Status.Nodes {
			if node.Name == "dag-mutex.A" {
				assert.Equal(wfv1.NodePending, node.Phase)
			}
		}
		assert.Equal(wfv1.WorkflowRunning, woc.wf.Status.Phase)
		makePodsPhase(ctx, woc, apiv1.PodSucceeded)

		woc1 := newWorkflowOperationCtx(woc.wf, controller)
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

const RetryWfWithSemaphore = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: script-wf
  namespace: default
spec:
  entrypoint: step1
  retryStrategy:
    limit: 10
  templates:
    - name: step1
      steps:
        - - name: hello1
            template: whalesay
        - - name: hello2
            template: whalesay
    - name: whalesay
      synchronization:
        semaphores:
          - configMapKeyRef:
              key: template
              name: my-config
      container:
        args:
          - "hello world"
        command:
          - cowsay
        image: "docker/whalesay:latest"
`

func TestSynchronizationWithRetry(t *testing.T) {
	assert := assert.New(t)
	cancel, controller := newController()
	defer cancel()
	ctx := context.Background()
	controller.syncManager = sync.NewLockManager(ctx, controller.kubeclientset, controller.namespace, nil, getSyncLimitFunc(ctx, controller.kubeclientset), func(key string) {
	}, workflowExistenceFunc)
	var cm apiv1.ConfigMap
	wfv1.MustUnmarshal([]byte(configMap), &cm)
	_, err := controller.kubeclientset.CoreV1().ConfigMaps("default").Create(ctx, &cm, metav1.CreateOptions{})
	require.NoError(t, err)
	t.Run("WorkflowWithRetry", func(t *testing.T) {
		wf := wfv1.MustUnmarshalWorkflow(RetryWfWithSemaphore)
		wf, err := controller.wfclientset.ArgoprojV1alpha1().Workflows(wf.Namespace).Create(ctx, wf, metav1.CreateOptions{})
		require.NoError(t, err)
		woc := newWorkflowOperationCtx(wf, controller)
		woc.operate(ctx)
		for _, node := range woc.wf.Status.Nodes {
			if node.Name == "hello1" {
				assert.Equal(wfv1.NodePending, node.Phase)
			}
		}

		// Updating Pod state
		makePodsPhase(ctx, woc, apiv1.PodSucceeded)

		// Release the lock from hello1
		woc = newWorkflowOperationCtx(woc.wf, controller)
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
		woc = newWorkflowOperationCtx(woc.wf, controller)
		woc.operate(ctx)
		// Nobody is waiting for the lock
		assert.Nil(woc.wf.Status.Synchronization)
	})
}

const StepWithSync = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: steps-jklcl
  namespace: default
spec:
  entrypoint: hello-hello-hello
  templates:
  -
    name: hello-hello-hello
    steps:
    - - arguments:
          parameters:
          - name: message
            value: hello1
        name: hello1
        template: whalesay
    synchronization:
      semaphores:
        - configMapKeyRef:
            key: step
            name: my-config
  -
    container:
      args:
      - '{{inputs.parameters.message}}'
      command:
      - cowsay
      image: docker/whalesay
    inputs:
      parameters:
      - name: message
    name: whalesay
`

const StepWithSyncStatus = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: steps-jklcl
  namespace: default
spec:
  entrypoint: hello-hello-hello
  templates:
  - inputs: {}
    name: hello-hello-hello
    steps:
    - - arguments:
          parameters:
          - name: message
            value: hello1
        name: hello1
        template: whalesay
    synchronization:
      semaphores:
        - configMapKeyRef:
            key: step
            name: my-config
  - container:
      args:
      - '{{inputs.parameters.message}}'
      command:
      - cowsay
      image: docker/whalesay
      resources: {}
    inputs:
      parameters:
      - name: message
    name: whalesay
status:
  artifactRepositoryRef:
    configMap: artifact-repositories
    key: default-v1
    namespace: argo
  conditions:
  - status: "False"
    type: PodRunning
  - status: "True"
    type: Completed
  finishedAt: "2021-02-11T19:46:55Z"
  nodes:
    steps-jklcl:
      children:
      - steps-jklcl-3895081407
      displayName: steps-jklcl
      finishedAt: "2021-02-11T19:46:55Z"
      id: steps-jklcl
      name: steps-jklcl
      outboundNodes:
      - steps-jklcl-969694128
      phase: Running
      progress: 1/1
      resourcesDuration:
        cpu: 7
        memory: 4
      startedAt: "2021-02-11T19:46:33Z"
      templateName: hello-hello-hello
      templateScope: local/steps-jklcl
      type: Steps
    steps-jklcl-969694128:
      boundaryID: steps-jklcl
      displayName: hello1
      finishedAt: "2021-02-11T19:46:44Z"
      id: steps-jklcl-969694128
      inputs:
        parameters:
        - name: message
          value: hello1
      name: steps-jklcl[0].hello1
      outputs:
        artifacts:
        - archiveLogs: true
          name: main-logs
          s3:
            accessKeySecret:
              key: accesskey
              name: my-minio-cred
            bucket: my-bucket
            endpoint: minio:9000
            insecure: true
            key: steps-jklcl/steps-jklcl-969694128/main.log
            secretKeySecret:
              key: secretkey
              name: my-minio-cred
        exitCode: "0"
      phase: Succeeded
      progress: 1/1
      resourcesDuration:
        cpu: 7
        memory: 4
      startedAt: "2021-02-11T19:46:33Z"
      templateName: whalesay
      templateScope: local/steps-jklcl
      type: Pod
    steps-jklcl-3895081407:
      boundaryID: steps-jklcl
      children:
      - steps-jklcl-969694128
      displayName: '[0]'
      finishedAt: "2021-02-11T19:46:55Z"
      id: steps-jklcl-3895081407
      name: steps-jklcl[0]
      phase: Succeeded
      progress: 1/1
      resourcesDuration:
        cpu: 7
        memory: 4
      startedAt: "2021-02-11T19:46:33Z"
      templateScope: local/steps-jklcl
      type: StepGroup
  phase: Succeeded
  progress: 1/1
  resourcesDuration:
    cpu: 7
    memory: 4
  startedAt: "2021-02-11T19:46:33Z"

`

func TestSynchronizationWithStep(t *testing.T) {
	assert := assert.New(t)
	cancel, controller := newController()
	defer cancel()
	ctx := context.Background()
	controller.syncManager = sync.NewLockManager(ctx, controller.kubeclientset, controller.namespace, nil, getSyncLimitFunc(ctx, controller.kubeclientset), func(key string) {
	}, workflowExistenceFunc)
	var cm apiv1.ConfigMap
	wfv1.MustUnmarshal([]byte(configMap), &cm)
	_, err := controller.kubeclientset.CoreV1().ConfigMaps("default").Create(ctx, &cm, metav1.CreateOptions{})
	require.NoError(t, err)

	t.Run("StepWithSychronization", func(t *testing.T) {
		// First workflow Acquire the lock
		wf := wfv1.MustUnmarshalWorkflow(StepWithSync)
		wf, err := controller.wfclientset.ArgoprojV1alpha1().Workflows("default").Create(ctx, wf, metav1.CreateOptions{})
		require.NoError(t, err)
		woc := newWorkflowOperationCtx(wf, controller)
		woc.operate(ctx)
		assert.NotNil(woc.wf.Status.Synchronization)
		assert.NotNil(woc.wf.Status.Synchronization.Semaphore)
		assert.Len(woc.wf.Status.Synchronization.Semaphore.Holding, 1)

		// Second workflow try to acquire the lock and wait for lock
		wf1 := wfv1.MustUnmarshalWorkflow(StepWithSync)
		wf1.Name = "step2"
		wf1, err = controller.wfclientset.ArgoprojV1alpha1().Workflows("default").Create(ctx, wf1, metav1.CreateOptions{})
		require.NoError(t, err)
		woc1 := newWorkflowOperationCtx(wf1, controller)
		woc1.operate(ctx)
		assert.NotNil(woc1.wf.Status.Synchronization)
		assert.NotNil(woc1.wf.Status.Synchronization.Semaphore)
		assert.Nil(woc1.wf.Status.Synchronization.Semaphore.Holding)
		assert.Len(woc1.wf.Status.Synchronization.Semaphore.Waiting, 1)

		// Finished all StepGroup in step
		wf = wfv1.MustUnmarshalWorkflow(StepWithSyncStatus)
		woc = newWorkflowOperationCtx(wf, controller)
		woc.operate(ctx)
		assert.Nil(woc.wf.Status.Synchronization)

		// Second workflow acquire the lock
		woc1 = newWorkflowOperationCtx(woc1.wf, controller)
		woc1.operate(ctx)
		assert.NotNil(woc1.wf.Status.Synchronization)
		assert.NotNil(woc1.wf.Status.Synchronization.Semaphore)
		assert.NotNil(woc1.wf.Status.Synchronization.Semaphore.Holding)
		assert.Len(woc1.wf.Status.Synchronization.Semaphore.Holding, 1)
	})
}

const wfWithStepRetry = `apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: my-workflow-
spec:
  entrypoint: step-entry
  templates:
    - name: step-entry
      steps:
      - - name: step1
          template: sleep

    - name: sleep
      retryStrategy:
        limit: 5
        retryPolicy: Always
      synchronization:
        semaphores:
          - configMapKeyRef:
              name: my-config
              key: template
      container:
        image: alpine:3.6
        command: [sh, -c]
        args: ["sleep 300"]`

func TestSynchronizationWithStepRetry(t *testing.T) {
	assert := assert.New(t)
	cancel, controller := newController()
	defer cancel()
	ctx := context.Background()
	controller.syncManager = sync.NewLockManager(ctx, controller.kubeclientset, controller.namespace, nil, getSyncLimitFunc(ctx, controller.kubeclientset), func(key string) {
	}, workflowExistenceFunc)
	var cm apiv1.ConfigMap
	wfv1.MustUnmarshal([]byte(configMap), &cm)
	_, err := controller.kubeclientset.CoreV1().ConfigMaps("default").Create(ctx, &cm, metav1.CreateOptions{})
	require.NoError(t, err)

	t.Run("StepRetryWithSynchronization", func(t *testing.T) {
		// First workflow Acquire the lock
		wf := wfv1.MustUnmarshalWorkflow(wfWithStepRetry)
		wf, err := controller.wfclientset.ArgoprojV1alpha1().Workflows("default").Create(ctx, wf, metav1.CreateOptions{})
		require.NoError(t, err)
		woc := newWorkflowOperationCtx(wf, controller)
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

const pendingWfWithShutdownStrategy = `apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: synchronization-wf-level
  namespace: default
spec:
  entrypoint: whalesay
  onExit: whalesay
  synchronization:
    mutexes:
      - name:  test
  templates:
    - name: whalesay
      container:
        image: docker/whalesay:latest
        command: [sh, -c]
        args: ["sleep 99999"]`

func TestSynchronizationForPendingShuttingdownWfs(t *testing.T) {
	cancel, controller := newController()
	defer cancel()
	ctx := context.Background()
	controller.syncManager = sync.NewLockManager(ctx, controller.kubeclientset, controller.namespace, nil, getSyncLimitFunc(ctx, controller.kubeclientset), func(key string) {
	}, workflowExistenceFunc)

	t.Run("PendingShuttingdownTerminatingWf", func(t *testing.T) {
		// Create and acquire the lock for the first workflow
		wf := wfv1.MustUnmarshalWorkflow(pendingWfWithShutdownStrategy)
		wf.Name = "one-terminating"
		wf.Spec.Synchronization.Mutexes[0].Name = "terminating-test"
		wf, err := controller.wfclientset.ArgoprojV1alpha1().Workflows(wf.Namespace).Create(ctx, wf, metav1.CreateOptions{})
		require.NoError(t, err)
		woc := newWorkflowOperationCtx(wf, controller)
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
		wocTwo := newWorkflowOperationCtx(wfTwo, controller)
		wocTwo.operate(ctx)
		assert.Equal(t, wfv1.WorkflowPending, wocTwo.wf.Status.Phase)

		// Shutdown the second workflow that's pending.
		patchObj := map[string]interface{}{
			"spec": map[string]interface{}{
				"shutdown": wfv1.ShutdownStrategyTerminate,
			},
		}
		patch, err := json.Marshal(patchObj)
		require.NoError(t, err)
		wfTwo, err = controller.wfclientset.ArgoprojV1alpha1().Workflows(wf.Namespace).Patch(ctx, wfTwo.Name, types.MergePatchType, patch, metav1.PatchOptions{})
		require.NoError(t, err)

		// The pending workflow that's being shutdown should have succeeded and released the lock.
		wocTwo = newWorkflowOperationCtx(wfTwo, controller)
		wocTwo.operate(ctx)
		assert.Equal(t, wfv1.WorkflowSucceeded, wocTwo.execWf.Status.Phase)
		assert.Nil(t, wocTwo.wf.Status.Synchronization)
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
		woc := newWorkflowOperationCtx(wf, controller)
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
		wocTwo := newWorkflowOperationCtx(wfTwo, controller)
		wocTwo.operate(ctx)
		assert.Equal(t, wfv1.WorkflowPending, wocTwo.wf.Status.Phase)

		// Shutdown the second workflow that's pending.
		patchObj := map[string]interface{}{
			"spec": map[string]interface{}{
				"shutdown": wfv1.ShutdownStrategyStop,
			},
		}
		patch, err := json.Marshal(patchObj)
		require.NoError(t, err)
		wfTwo, err = controller.wfclientset.ArgoprojV1alpha1().Workflows(wf.Namespace).Patch(ctx, wfTwo.Name, types.MergePatchType, patch, metav1.PatchOptions{})
		require.NoError(t, err)

		// The pending workflow that's being shutdown should still be pending and waiting to acquire the lock.
		wocTwo = newWorkflowOperationCtx(wfTwo, controller)
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
	wf := wfv1.MustUnmarshalWorkflow(`apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: example-steps-simple
  namespace: default
spec:
  entrypoint: main
  templates:
    - name: main
      steps:
        - - name: job-1
            template: sleep
            arguments:
              parameters:
                - name: sleep_duration
                  value: 10
          - name: job-2
            template: sleep
            arguments:
              parameters:
                - name: sleep_duration
                  value: 5

    - name: sleep
      synchronization:
        mutexes:
          - name: mutex-example-steps-simple
      inputs:
        parameters:
          - name: sleep_duration
      script:
        image: alpine:latest
        command: [/bin/sh]
        source: |
          echo "Sleeping for {{ inputs.parameters.sleep_duration }}"
          sleep {{ inputs.parameters.sleep_duration }}
      memoize:
        key: "memo-key-1"
        cache:
          configMap:
            name: cache-example-steps-simple
    `)
	wf.Name = "example-steps-simple-gas12"
	cancel, controller := newController(wf)
	defer cancel()

	ctx := context.Background()
	woc := newWorkflowOperationCtx(wf, controller)
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
