package controller

import (
	"context"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
        semaphore: 
          configMapKeyRef: 
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
      semaphore: 
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
      semaphore: 
        configMapKeyRef: 
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

func GetSyncLimitFunc(ctx context.Context, kube kubernetes.Interface) func(string) (int, error) {
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
	controller.syncManager = sync.NewLockManager(GetSyncLimitFunc(ctx, controller.kubeclientset), func(key string) {
	}, workflowExistenceFunc)
	var cm v1.ConfigMap
	wfv1.MustUnmarshal([]byte(configMap), &cm)
	_, err := controller.kubeclientset.CoreV1().ConfigMaps("default").Create(ctx, &cm, metav1.CreateOptions{})
	assert.NoError(t, err)

	t.Run("TmplLevelAcquireAndRelease", func(t *testing.T) {
		wf := wfv1.MustUnmarshalWorkflow(wfWithSemaphore)
		wf.Name = "one"
		wf, err := controller.wfclientset.ArgoprojV1alpha1().Workflows(wf.Namespace).Create(ctx, wf, metav1.CreateOptions{})
		assert.NoError(t, err)
		woc := newWorkflowOperationCtx(wf, controller)

		// acquired the lock
		woc.operate(ctx)
		assert.NotNil(t, woc.wf.Status.Synchronization)
		assert.NotNil(t, woc.wf.Status.Synchronization.Semaphore)
		assert.Equal(t, 1, len(woc.wf.Status.Synchronization.Semaphore.Holding))

		for _, node := range woc.wf.Status.Nodes {
			assert.Equal(t, wfv1.NodePending, node.Phase)
		}

		// Try to Acquire the lock, But lock is not available
		wf_Two := wf.DeepCopy()
		wf_Two.Name = "two"
		wf_Two, err = controller.wfclientset.ArgoprojV1alpha1().Workflows(wf.Namespace).Create(ctx, wf_Two, metav1.CreateOptions{})
		assert.NoError(t, err)
		woc_two := newWorkflowOperationCtx(wf_Two, controller)
		// Try Acquire the lock
		woc_two.operate(ctx)

		// Check Node status
		err = woc_two.podReconciliation(ctx)
		assert.NoError(t, err)
		for _, node := range woc_two.wf.Status.Nodes {
			assert.Equal(t, wfv1.NodePending, node.Phase)
		}

		// Updating Pod state
		makePodsPhase(ctx, woc, v1.PodFailed)

		// Release the lock
		woc = newWorkflowOperationCtx(woc.wf, controller)
		woc.operate(ctx)
		assert.Nil(t, woc.wf.Status.Synchronization)

		// Try to acquired the lock
		woc_two = newWorkflowOperationCtx(woc_two.wf, controller)
		woc_two.operate(ctx)
		assert.NotNil(t, woc_two.wf.Status.Synchronization)
		assert.NotNil(t, woc_two.wf.Status.Synchronization.Semaphore)
		assert.Equal(t, 1, len(woc_two.wf.Status.Synchronization.Semaphore.Holding))
	})
}

func TestSemaphoreScriptTmplLevel(t *testing.T) {
	cancel, controller := newController()
	defer cancel()
	ctx := context.Background()
	controller.syncManager = sync.NewLockManager(GetSyncLimitFunc(ctx, controller.kubeclientset), func(key string) {
	}, workflowExistenceFunc)
	var cm v1.ConfigMap
	wfv1.MustUnmarshal([]byte(configMap), &cm)
	_, err := controller.kubeclientset.CoreV1().ConfigMaps("default").Create(ctx, &cm, metav1.CreateOptions{})
	assert.NoError(t, err)

	t.Run("ScriptTmplLevelAcquireAndRelease", func(t *testing.T) {
		wf := wfv1.MustUnmarshalWorkflow(ScriptWfWithSemaphore)
		wf.Name = "one"
		wf, err := controller.wfclientset.ArgoprojV1alpha1().Workflows(wf.Namespace).Create(ctx, wf, metav1.CreateOptions{})
		assert.NoError(t, err)
		woc := newWorkflowOperationCtx(wf, controller)

		// acquired the lock
		woc.operate(ctx)
		assert.NotNil(t, woc.wf.Status.Synchronization)
		assert.NotNil(t, woc.wf.Status.Synchronization.Semaphore)
		assert.Equal(t, 1, len(woc.wf.Status.Synchronization.Semaphore.Holding))

		for _, node := range woc.wf.Status.Nodes {
			assert.Equal(t, wfv1.NodePending, node.Phase)
		}

		// Try to Acquire the lock, But lock is not available
		wf_Two := wf.DeepCopy()
		wf_Two.Name = "two"
		wf_Two, err = controller.wfclientset.ArgoprojV1alpha1().Workflows(wf.Namespace).Create(ctx, wf_Two, metav1.CreateOptions{})
		assert.NoError(t, err)
		woc_two := newWorkflowOperationCtx(wf_Two, controller)
		// Try Acquire the lock
		woc_two.operate(ctx)

		// Check Node status
		err = woc_two.podReconciliation(ctx)
		assert.NoError(t, err)
		for _, node := range woc_two.wf.Status.Nodes {
			assert.Equal(t, wfv1.NodePending, node.Phase)
		}
		// Updating Pod state
		makePodsPhase(ctx, woc, v1.PodFailed)

		// Release the lock
		woc = newWorkflowOperationCtx(woc.wf, controller)
		woc.operate(ctx)
		assert.Nil(t, woc.wf.Status.Synchronization)

		// Try to acquired the lock
		woc_two = newWorkflowOperationCtx(woc_two.wf, controller)
		woc_two.operate(ctx)
		assert.NotNil(t, woc_two.wf.Status.Synchronization)
		assert.NotNil(t, woc_two.wf.Status.Synchronization.Semaphore)
		assert.Equal(t, 1, len(woc_two.wf.Status.Synchronization.Semaphore.Holding))
	})
}

func TestSemaphoreResourceTmplLevel(t *testing.T) {
	cancel, controller := newController()
	defer cancel()
	ctx := context.Background()
	controller.syncManager = sync.NewLockManager(GetSyncLimitFunc(ctx, controller.kubeclientset), func(key string) {
	}, workflowExistenceFunc)
	var cm v1.ConfigMap
	wfv1.MustUnmarshal([]byte(configMap), &cm)
	_, err := controller.kubeclientset.CoreV1().ConfigMaps("default").Create(ctx, &cm, metav1.CreateOptions{})
	assert.NoError(t, err)

	t.Run("ResourceTmplLevelAcquireAndRelease", func(t *testing.T) {
		wf := wfv1.MustUnmarshalWorkflow(ResourceWfWithSemaphore)
		wf.Name = "one"
		wf, err := controller.wfclientset.ArgoprojV1alpha1().Workflows(wf.Namespace).Create(ctx, wf, metav1.CreateOptions{})
		assert.NoError(t, err)
		woc := newWorkflowOperationCtx(wf, controller)

		// acquired the lock
		woc.operate(ctx)
		assert.NotNil(t, woc.wf.Status.Synchronization)
		assert.NotNil(t, woc.wf.Status.Synchronization.Semaphore)
		assert.Equal(t, 1, len(woc.wf.Status.Synchronization.Semaphore.Holding))

		for _, node := range woc.wf.Status.Nodes {
			assert.Equal(t, wfv1.NodePending, node.Phase)
		}

		// Try to Acquire the lock, But lock is not available
		wf_Two := wf.DeepCopy()
		wf_Two.Name = "two"
		wf_Two, err = controller.wfclientset.ArgoprojV1alpha1().Workflows(wf.Namespace).Create(ctx, wf_Two, metav1.CreateOptions{})
		assert.NoError(t, err)
		woc_two := newWorkflowOperationCtx(wf_Two, controller)
		// Try Acquire the lock
		woc_two.operate(ctx)

		// Check Node status
		err = woc_two.podReconciliation(ctx)
		assert.NoError(t, err)
		for _, node := range woc_two.wf.Status.Nodes {
			assert.Equal(t, wfv1.NodePending, node.Phase)
		}

		// Updating Pod state
		makePodsPhase(ctx, woc, v1.PodFailed)

		// Release the lock
		woc = newWorkflowOperationCtx(woc.wf, controller)
		woc.operate(ctx)
		assert.Nil(t, woc.wf.Status.Synchronization)

		// Try to acquired the lock
		woc_two = newWorkflowOperationCtx(woc_two.wf, controller)
		woc_two.operate(ctx)
		assert.NotNil(t, woc_two.wf.Status.Synchronization)
		assert.NotNil(t, woc_two.wf.Status.Synchronization.Semaphore)
		assert.Equal(t, 1, len(woc_two.wf.Status.Synchronization.Semaphore.Holding))
	})
}

func TestSemaphoreWithOutConfigMap(t *testing.T) {
	cancel, controller := newController()
	defer cancel()

	ctx := context.Background()
	controller.syncManager = sync.NewLockManager(GetSyncLimitFunc(ctx, controller.kubeclientset), func(key string) {
	}, workflowExistenceFunc)

	t.Run("SemaphoreRefWithOutConfigMap", func(t *testing.T) {
		wf := wfv1.MustUnmarshalWorkflow(wfWithSemaphore)
		wf.Name = "one"
		wf, err := controller.wfclientset.ArgoprojV1alpha1().Workflows(wf.Namespace).Create(ctx, wf, metav1.CreateOptions{})
		assert.NoError(t, err)
		woc := newWorkflowOperationCtx(wf, controller)
		err = woc.podReconciliation(ctx)
		assert.NoError(t, err)
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
     mutex:
       name: welcome
   container:
     image: alpine:3.7
     command: [sh, -c, "exit 0"]
`

func TestMutexInDAG(t *testing.T) {
	assert := assert.New(t)

	cancel, controller := newController()
	defer cancel()
	ctx := context.Background()
	controller.syncManager = sync.NewLockManager(GetSyncLimitFunc(ctx, controller.kubeclientset), func(key string) {
	}, workflowExistenceFunc)
	t.Run("MutexWithDAG", func(t *testing.T) {
		wf := wfv1.MustUnmarshalWorkflow(DAGWithMutex)
		wf, err := controller.wfclientset.ArgoprojV1alpha1().Workflows(wf.Namespace).Create(ctx, wf, metav1.CreateOptions{})
		assert.NoError(err)
		woc := newWorkflowOperationCtx(wf, controller)
		woc.operate(ctx)
		for _, node := range woc.wf.Status.Nodes {
			if node.Name == "dag-mutex.A" {
				assert.Equal(wfv1.NodePending, node.Phase)
			}
		}
		assert.Equal(wfv1.WorkflowRunning, woc.wf.Status.Phase)
		makePodsPhase(ctx, woc, v1.PodSucceeded)

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
      daemon: true
      synchronization: 
        semaphore: 
          configMapKeyRef: 
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
	controller.syncManager = sync.NewLockManager(GetSyncLimitFunc(ctx, controller.kubeclientset), func(key string) {
	}, workflowExistenceFunc)
	var cm v1.ConfigMap
	wfv1.MustUnmarshal([]byte(configMap), &cm)
	_, err := controller.kubeclientset.CoreV1().ConfigMaps("default").Create(ctx, &cm, metav1.CreateOptions{})
	assert.NoError(err)
	t.Run("WorkflowWithRetry", func(t *testing.T) {
		wf := wfv1.MustUnmarshalWorkflow(RetryWfWithSemaphore)
		wf, err := controller.wfclientset.ArgoprojV1alpha1().Workflows(wf.Namespace).Create(ctx, wf, metav1.CreateOptions{})
		assert.NoError(err)
		woc := newWorkflowOperationCtx(wf, controller)
		woc.operate(ctx)
		for _, node := range woc.wf.Status.Nodes {
			if node.Name == "hello1" {
				assert.Equal(wfv1.NodePending, node.Phase)
			}
		}

		// Updating Pod state
		makePodsPhase(ctx, woc, v1.PodSucceeded)

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
		makePodsPhase(ctx, woc, v1.PodSucceeded)

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
      semaphore:
        configMapKeyRef:
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
      semaphore:
        configMapKeyRef:
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
	controller.syncManager = sync.NewLockManager(GetSyncLimitFunc(ctx, controller.kubeclientset), func(key string) {
	}, workflowExistenceFunc)
	var cm v1.ConfigMap
	wfv1.MustUnmarshal([]byte(configMap), &cm)
	_, err := controller.kubeclientset.CoreV1().ConfigMaps("default").Create(ctx, &cm, metav1.CreateOptions{})
	assert.NoError(err)

	t.Run("StepWithSychronization", func(t *testing.T) {
		// First workflow Acquire the lock
		wf := wfv1.MustUnmarshalWorkflow(StepWithSync)
		wf, err := controller.wfclientset.ArgoprojV1alpha1().Workflows("default").Create(ctx, wf, metav1.CreateOptions{})
		assert.NoError(err)
		woc := newWorkflowOperationCtx(wf, controller)
		woc.operate(ctx)
		assert.NotNil(woc.wf.Status.Synchronization)
		assert.NotNil(woc.wf.Status.Synchronization.Semaphore)
		assert.Len(woc.wf.Status.Synchronization.Semaphore.Holding, 1)

		// Second workflow try to acquire the lock and wait for lock
		wf1 := wfv1.MustUnmarshalWorkflow(StepWithSync)
		wf1.Name = "step2"
		wf1, err = controller.wfclientset.ArgoprojV1alpha1().Workflows("default").Create(ctx, wf1, metav1.CreateOptions{})
		assert.NoError(err)
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
