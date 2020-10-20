package controller

import (
	"strconv"
	"strings"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	"sigs.k8s.io/yaml"

	argoErr "github.com/argoproj/argo/errors"
	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo/workflow/sync"
)

const configMap = `
apiVersion: v1
kind: ConfigMap
metadata:
  name: my-config
data:
  workflow: "2"
  template: "1"
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

func GetSyncLimitFunc(kube kubernetes.Interface) func(string) (int, error) {
	syncLimitConfig := func(lockName string) (int, error) {
		items := strings.Split(lockName, "/")
		if len(items) < 4 {
			return 0, argoErr.New(argoErr.CodeBadRequest, "Invalid Config Map Key")
		}

		configMap, err := kube.CoreV1().ConfigMaps(items[0]).Get(items[2], metav1.GetOptions{})

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
	controller.syncManager = sync.NewLockManager(GetSyncLimitFunc(controller.kubeclientset), func(key string) {
	})
	var cm v1.ConfigMap
	err := yaml.Unmarshal([]byte(configMap), &cm)
	assert.NoError(t, err)
	_, err = controller.kubeclientset.CoreV1().ConfigMaps("default").Create(&cm)
	assert.NoError(t, err)

	t.Run("TmplLevelAcquireAndRelease", func(t *testing.T) {
		wf := unmarshalWF(wfWithSemaphore)
		wf.Name = "one"
		wf, err := controller.wfclientset.ArgoprojV1alpha1().Workflows(wf.Namespace).Create(wf)
		assert.NoError(t, err)
		woc := newWorkflowOperationCtx(wf, controller)

		// acquired the lock
		woc.operate()
		assert.NotNil(t, woc.wf.Status.Synchronization)
		assert.NotNil(t, woc.wf.Status.Synchronization.Semaphore)
		assert.Equal(t, 1, len(woc.wf.Status.Synchronization.Semaphore.Holding))

		for _, node := range woc.wf.Status.Nodes {
			assert.Equal(t, wfv1.NodePending, node.Phase)
		}

		// Try to Acquire the lock, But lock is not available
		wf_Two := wf.DeepCopy()
		wf_Two.Name = "two"
		wf_Two, err = controller.wfclientset.ArgoprojV1alpha1().Workflows(wf.Namespace).Create(wf_Two)
		assert.NoError(t, err)
		woc_two := newWorkflowOperationCtx(wf_Two, controller)
		// Try Acquire the lock
		woc_two.operate()

		// Check Node status
		err = woc_two.podReconciliation()
		assert.NoError(t, err)
		for _, node := range woc_two.wf.Status.Nodes {
			assert.Equal(t, wfv1.NodePending, node.Phase)
		}

		// Updating Pod state
		makePodsPhase(woc, v1.PodFailed)

		// Release the lock
		woc = newWorkflowOperationCtx(woc.wf, controller)
		woc.operate()
		assert.Nil(t, woc.wf.Status.Synchronization)

		// Try to acquired the lock
		woc_two = newWorkflowOperationCtx(woc_two.wf, controller)
		woc_two.operate()
		assert.NotNil(t, woc_two.wf.Status.Synchronization)
		assert.NotNil(t, woc_two.wf.Status.Synchronization.Semaphore)
		assert.Equal(t, 1, len(woc_two.wf.Status.Synchronization.Semaphore.Holding))

	})
}

func TestSemaphoreScriptTmplLevel(t *testing.T) {
	cancel, controller := newController()
	defer cancel()
	controller.syncManager = sync.NewLockManager(GetSyncLimitFunc(controller.kubeclientset), func(key string) {
	})
	var cm v1.ConfigMap
	err := yaml.Unmarshal([]byte(configMap), &cm)
	assert.NoError(t, err)
	_, err = controller.kubeclientset.CoreV1().ConfigMaps("default").Create(&cm)
	assert.NoError(t, err)

	t.Run("ScriptTmplLevelAcquireAndRelease", func(t *testing.T) {
		wf := unmarshalWF(ScriptWfWithSemaphore)
		wf.Name = "one"
		wf, err := controller.wfclientset.ArgoprojV1alpha1().Workflows(wf.Namespace).Create(wf)
		assert.NoError(t, err)
		woc := newWorkflowOperationCtx(wf, controller)

		// acquired the lock
		woc.operate()
		assert.NotNil(t, woc.wf.Status.Synchronization)
		assert.NotNil(t, woc.wf.Status.Synchronization.Semaphore)
		assert.Equal(t, 1, len(woc.wf.Status.Synchronization.Semaphore.Holding))

		for _, node := range woc.wf.Status.Nodes {
			assert.Equal(t, wfv1.NodePending, node.Phase)
		}

		// Try to Acquire the lock, But lock is not available
		wf_Two := wf.DeepCopy()
		wf_Two.Name = "two"
		wf_Two, err = controller.wfclientset.ArgoprojV1alpha1().Workflows(wf.Namespace).Create(wf_Two)
		assert.NoError(t, err)
		woc_two := newWorkflowOperationCtx(wf_Two, controller)
		// Try Acquire the lock
		woc_two.operate()

		// Check Node status
		err = woc_two.podReconciliation()
		assert.NoError(t, err)
		for _, node := range woc_two.wf.Status.Nodes {
			assert.Equal(t, wfv1.NodePending, node.Phase)
		}
		// Updating Pod state
		makePodsPhase(woc, v1.PodFailed)

		// Release the lock
		woc = newWorkflowOperationCtx(woc.wf, controller)
		woc.operate()
		assert.Nil(t, woc.wf.Status.Synchronization)

		// Try to acquired the lock
		woc_two = newWorkflowOperationCtx(woc_two.wf, controller)
		woc_two.operate()
		assert.NotNil(t, woc_two.wf.Status.Synchronization)
		assert.NotNil(t, woc_two.wf.Status.Synchronization.Semaphore)
		assert.Equal(t, 1, len(woc_two.wf.Status.Synchronization.Semaphore.Holding))

	})
}

func TestSemaphoreResourceTmplLevel(t *testing.T) {
	cancel, controller := newController()
	defer cancel()
	controller.syncManager = sync.NewLockManager(GetSyncLimitFunc(controller.kubeclientset), func(key string) {
	})
	var cm v1.ConfigMap
	err := yaml.Unmarshal([]byte(configMap), &cm)
	assert.NoError(t, err)
	_, err = controller.kubeclientset.CoreV1().ConfigMaps("default").Create(&cm)
	assert.NoError(t, err)

	t.Run("ResourceTmplLevelAcquireAndRelease", func(t *testing.T) {
		wf := unmarshalWF(ResourceWfWithSemaphore)
		wf.Name = "one"
		wf, err := controller.wfclientset.ArgoprojV1alpha1().Workflows(wf.Namespace).Create(wf)
		assert.NoError(t, err)
		woc := newWorkflowOperationCtx(wf, controller)

		// acquired the lock
		woc.operate()
		assert.NotNil(t, woc.wf.Status.Synchronization)
		assert.NotNil(t, woc.wf.Status.Synchronization.Semaphore)
		assert.Equal(t, 1, len(woc.wf.Status.Synchronization.Semaphore.Holding))

		for _, node := range woc.wf.Status.Nodes {
			assert.Equal(t, wfv1.NodePending, node.Phase)
		}

		// Try to Acquire the lock, But lock is not available
		wf_Two := wf.DeepCopy()
		wf_Two.Name = "two"
		wf_Two, err = controller.wfclientset.ArgoprojV1alpha1().Workflows(wf.Namespace).Create(wf_Two)
		assert.NoError(t, err)
		woc_two := newWorkflowOperationCtx(wf_Two, controller)
		// Try Acquire the lock
		woc_two.operate()

		// Check Node status
		err = woc_two.podReconciliation()
		assert.NoError(t, err)
		for _, node := range woc_two.wf.Status.Nodes {
			assert.Equal(t, wfv1.NodePending, node.Phase)
		}

		// Updating Pod state
		makePodsPhase(woc, v1.PodFailed)

		// Release the lock
		woc = newWorkflowOperationCtx(woc.wf, controller)
		woc.operate()
		assert.Nil(t, woc.wf.Status.Synchronization)

		// Try to acquired the lock
		woc_two = newWorkflowOperationCtx(woc_two.wf, controller)
		woc_two.operate()
		assert.NotNil(t, woc_two.wf.Status.Synchronization)
		assert.NotNil(t, woc_two.wf.Status.Synchronization.Semaphore)
		assert.Equal(t, 1, len(woc_two.wf.Status.Synchronization.Semaphore.Holding))

	})
}
func TestSemaphoreWithOutConfigMap(t *testing.T) {
	cancel, controller := newController()
	defer cancel()

	controller.syncManager = sync.NewLockManager(GetSyncLimitFunc(controller.kubeclientset), func(key string) {
	})

	t.Run("SemaphoreRefWithOutConfigMap", func(t *testing.T) {
		wf := unmarshalWF(wfWithSemaphore)
		wf.Name = "one"
		wf, err := controller.wfclientset.ArgoprojV1alpha1().Workflows(wf.Namespace).Create(wf)
		assert.NoError(t, err)
		woc := newWorkflowOperationCtx(wf, controller)
		err = woc.podReconciliation()
		assert.NoError(t, err)
		for _, node := range woc.wf.Status.Nodes {
			assert.Equal(t, wfv1.NodePending, node.Phase)
		}
		// Acquire the lock
		woc.operate()
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
	controller.syncManager = sync.NewLockManager(GetSyncLimitFunc(controller.kubeclientset), func(key string) {
	})
	t.Run("MutexWithDAG", func(t *testing.T) {
		wf := unmarshalWF(DAGWithMutex)
		wf, err := controller.wfclientset.ArgoprojV1alpha1().Workflows(wf.Namespace).Create(wf)
		assert.NoError(err)
		woc := newWorkflowOperationCtx(wf, controller)
		woc.operate()
		for _, node := range woc.wf.Status.Nodes {
			if node.Name == "dag-mutex.A" {
				assert.Equal(wfv1.NodePending, node.Phase)
			}
		}
		assert.Equal(wfv1.NodeRunning, woc.wf.Status.Phase)
		makePodsPhase(woc, v1.PodSucceeded)

		woc1 := newWorkflowOperationCtx(woc.wf, controller)
		woc1.operate()
		for _, node := range woc1.wf.Status.Nodes {
			if node.Name == "dag-mutex.B" {
				assert.Nil(node.SynchronizationStatus)
				assert.Equal(wfv1.NodePending, node.Phase)
			}
		}
	})
}
