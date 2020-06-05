package controller

import (
	"testing"

	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	"sigs.k8s.io/yaml"

	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo/workflow/concurrency"
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


  - name: whalesay
    semaphore:
      configMapKeyRef:
        name: my-config
        key: template
    container:
      image: docker/whalesay:latest
      command: [cowsay]
      args: ["hello world"]
`

func TestGetNodeType(t *testing.T) {
	t.Run("getNodeType", func(t *testing.T) {
		assert.Equal(t, wfv1.NodeTypePod, getNodeType(wfv1.TemplateTypeScript))
		assert.Equal(t, wfv1.NodeTypePod, getNodeType(wfv1.TemplateTypeContainer))
		assert.Equal(t, wfv1.NodeTypePod, getNodeType(wfv1.TemplateTypeResource))
		assert.NotEqual(t, wfv1.NodeTypePod, getNodeType(wfv1.TemplateTypeSteps))
		assert.NotEqual(t, wfv1.NodeTypePod, getNodeType(wfv1.TemplateTypeDAG))
		assert.NotEqual(t, wfv1.NodeTypePod, getNodeType(wfv1.TemplateTypeSuspend))
	})
}

func TestSemaphoreTmplLevel(t *testing.T) {
	_, controller := newController()
	controller.concurrencyMgr = concurrency.NewConcurrencyManager(controller.kubeclientset, func(key string) {
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

		// Acquired the lock
		woc.operate()
		assert.NotNil(t, woc.wf.Status.ConcurrencyLockStatus)
		assert.NotNil(t, woc.wf.Status.ConcurrencyLockStatus.SemaphoreHolders)
		assert.Equal(t, 1, len(woc.wf.Status.ConcurrencyLockStatus.SemaphoreHolders))

		// Try to Acquire the lock, But lock is not available
		wf_Two := wf.DeepCopy()
		wf_Two.Name = "two"
		wf_Two, err = controller.wfclientset.ArgoprojV1alpha1().Workflows(wf.Namespace).Create(wf_Two)
		assert.NoError(t, err)
		woc_two := newWorkflowOperationCtx(wf_Two, controller)
		// Try Acquire the lock
		woc_two.operate()
		for _, node := range woc.wf.Status.Nodes {
			assert.Equal(t, wfv1.NodePending, node.Phase)
		}

		// Check Node status
		woc_two.podReconciliation()
		for _, node := range woc_two.wf.Status.Nodes {
			assert.Equal(t, wfv1.NodePending, node.Phase)
		}

		// Release the lock
		woc.operate()
		assert.NotNil(t, woc.wf.Status.ConcurrencyLockStatus)
		assert.Empty(t, woc.wf.Status.ConcurrencyLockStatus.SemaphoreHolders)
		assert.Equal(t, 0, len(woc.wf.Status.ConcurrencyLockStatus.SemaphoreHolders))

		// Try to Acquired the lock
		woc_two.operate()
		assert.NotNil(t, woc_two.wf.Status.ConcurrencyLockStatus)
		assert.NotNil(t, woc_two.wf.Status.ConcurrencyLockStatus.SemaphoreHolders)
		assert.Equal(t, 1, len(woc_two.wf.Status.ConcurrencyLockStatus.SemaphoreHolders))

	})
}

func TestSemaphoreWithOutConfigMap(t *testing.T) {
	_, controller := newController()
	controller.concurrencyMgr = concurrency.NewConcurrencyManager(controller.kubeclientset, func(key string) {
	})

	t.Run("SemaphoreRefWithOutConfigMap", func(t *testing.T) {
		wf := unmarshalWF(wfWithSemaphore)
		wf.Name = "one"
		wf, err := controller.wfclientset.ArgoprojV1alpha1().Workflows(wf.Namespace).Create(wf)
		assert.NoError(t, err)
		woc := newWorkflowOperationCtx(wf, controller)
		woc.podReconciliation()
		for _, node := range woc.wf.Status.Nodes {
			assert.Equal(t, wfv1.NodePending, node.Phase)
		}
		// Acquire the lock
		woc.operate()
		assert.Nil(t, woc.wf.Status.ConcurrencyLockStatus)
		for _, node := range woc.wf.Status.Nodes {
			assert.Equal(t, wfv1.NodeError, node.Phase)
		}

	})
}
