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
  template: "2"
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

func TestSemaphoreTmplLevel(t *testing.T) {
	_, controller := newController()
	controller.concurrencyMgr = concurrency.NewConcurrencyManager(controller.kubeclientset, func(key string) {
	})
	var cm v1.ConfigMap
	err := yaml.Unmarshal([]byte(configMap), &cm)
	assert.NoError(t, err)
	_, err =controller.kubeclientset.CoreV1().ConfigMaps("default").Create(&cm)
	assert.NoError(t, err)
	t.Run("TmplLevelAcquireAndRelease", func(t *testing.T) {
		wf := unmarshalWF(wfWithSemaphore)
		wf.Name = "one"
		wf, err := controller.wfclientset.ArgoprojV1alpha1().Workflows(wf.Namespace).Create(wf)
		assert.NoError(t, err)
		woc := newWorkflowOperationCtx(wf, controller)
		// Acquire the lock
		woc.operate()
		assert.NotNil(t, woc.wf.Status.ConcurrencyLockStatus)
		assert.NotNil(t, woc.wf.Status.ConcurrencyLockStatus.SemaphoreHolders)
		assert.Equal(t, 1, len(woc.wf.Status.ConcurrencyLockStatus.SemaphoreHolders))
		for _, node := range woc.wf.Status.Nodes {
			node.Phase = wfv1.NodeSucceeded
		}
		// Release the lock
		woc.operate()
		assert.NotNil(t, woc.wf.Status.ConcurrencyLockStatus)
		assert.Empty(t, woc.wf.Status.ConcurrencyLockStatus.SemaphoreHolders)
		assert.Equal(t, 0, len(woc.wf.Status.ConcurrencyLockStatus.SemaphoreHolders))

	})
}
