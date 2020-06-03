package concurrency

import (
	"encoding/json"
	"fmt"
	"github.com/argoproj/argo/workflow/common"
	"github.com/argoproj/argo/workflow/controller"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	"sigs.k8s.io/yaml"
	"testing"
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
  semaphore:
    configMapKeyRef:
      name: my-config
      key: workflow
  templates:
  - name: whalesay
    container:
      image: docker/whalesay:latest
      command: [cowsay]
      args: ["hello world"]
`
const wfWithTmplSemaphore = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: semaphore-tmpl-level
  namespace: default
spec:
  entrypoint: semaphore-tmpl-level-example
  templates:
  - name: semaphore-tmpl-level-example
    steps:
    - - name: sleep
        template: sleep-n-sec
      - name: sleep1
        template: sleep-n-sec
      - name: sleep2
        template: sleep-n-sec
      - name: sleep3
        template: sleep-n-sec
      - name: sleep4
        template: sleep-n-sec
      - name: sleep5
        template: sleep-n-sec
      - name: sleep6
        template: sleep-n-sec

  - name: sleep-n-sec
    semaphore:
      configMapKeyRef:
        name: my-config
        key: template
    container:
      image: alpine:latest
      command: [sh, -c]
      args: ["sleep 10; echo done"]
`


func TestSemaphoreWfLevel(t *testing.T) {
	_, controller := controller.NewController()
	controller.concurrencyMgr = NewConcurrencyManager(controller)
	var cm v1.ConfigMap
	err := yaml.Unmarshal([]byte(configMap), &cm)
	assert.NoError(t, err)
	_, err = controller.kubeclientset.CoreV1().ConfigMaps("default").Create(&cm)
	assert.NoError(t, err)
	wf := controller2.unmarshalWF(wfWithSemaphore)

	wf1 := wf.DeepCopy()
	wf2 := wf.DeepCopy()
	t.Run("First Lock acquired", func(t *testing.T) {
		wf.Name = "one"
		priority, createTime := controller.getWfPriority(wf)
		key := controller.concurrencyMgr.getHolderKey(wf, "")
		status, _, err := controller.concurrencyMgr.tryAcquire(key, wf.Namespace, priority, createTime, wf.Spec.Semaphore, wf)
		assert.NoError(t, err)
		assert.True(t, status)
		assert.NotNil(t, wf.Labels)
		assert.Equal(t, "true", wf.Labels[common.LabelKeySemaphore])
		assert.NotNil(t, wf.Annotations)
		assert.Contains(t, wf.Annotations[common.AnnotationKeySemaphoreHolder], key)

	})

	t.Run("Second Lock acquired", func(t *testing.T) {
		wf1.Name = "two"
		priority, createTime := controller2.getWfPriority(wf1)
		key := controller.concurrencyMgr.getHolderKey(wf1, "")
		status, _, err := controller.concurrencyMgr.tryAcquire(key, wf1.Namespace, priority, createTime, wf1.Spec.Semaphore, wf1)
		assert.NoError(t, err)
		assert.True(t, status)
		assert.NotNil(t, wf1.Labels)
		assert.Equal(t, "true", wf1.Labels[common.LabelKeySemaphore])
		assert.NotNil(t, wf1.Annotations)
		assert.Contains(t, wf1.Annotations[common.AnnotationKeySemaphoreHolder], key)
	})
	t.Run("Waiting for Lock", func(t *testing.T) {
		wf2.Name = "three"
		priority, createTime := controller2.getWfPriority(wf2)
		key := controller.concurrencyMgr.getHolderKey(wf2, "")
		status, msg, err := controller.concurrencyMgr.tryAcquire(key, wf.Namespace, priority, createTime, wf2.Spec.Semaphore, wf2)
		assert.NoError(t, err)
		assert.False(t, status)
		assert.NotEmpty(t, msg)
	})
	t.Run("release the Lock", func(t *testing.T) {
		wf.Name = "one"
		key := controller.concurrencyMgr.getHolderKey(wf, "")
		controller.concurrencyMgr.release(key, wf.Namespace, wf.Spec.Semaphore, wf)
		assert.NotNil(t, wf.Annotations)
		assert.NotContains(t, wf.Annotations[common.AnnotationKeySemaphoreHolder], key)
	})
	t.Run("Released Lock acquired", func(t *testing.T) {
		wf2.Name = "three"
		priority, createTime := controller2.getWfPriority(wf2)
		key := controller.concurrencyMgr.getHolderKey(wf2, "")
		status, _, err := controller.concurrencyMgr.tryAcquire(key, wf2.Namespace, priority, createTime, wf2.Spec.Semaphore, wf2)
		assert.NoError(t, err)
		assert.True(t, status)
		assert.NotNil(t, wf2.Labels)
		assert.Equal(t, "true", wf2.Labels[common.LabelKeySemaphore])
		assert.NotNil(t, wf2.Annotations)
		assert.Contains(t, wf2.Annotations[common.AnnotationKeySemaphoreHolder], key)
	})
}

func TestSemaphoreTmplLevel(t *testing.T) {
	_, controller := controller2.newController()
	controller.concurrencyMgr = NewConcurrencyManager(controller)
	var cm v1.ConfigMap
	err := yaml.Unmarshal([]byte(configMap), &cm)
	assert.NoError(t, err)
	_, err = controller.kubeclientset.CoreV1().ConfigMaps("default").Create(&cm)
	assert.NoError(t, err)
	t.Run("Acquire lock for nodes", func(t *testing.T) {
		wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("default")
		wf := controller2.unmarshalWF(wfWithTmplSemaphore)
		wf, err := wfcset.Create(wf)
		assert.NoError(t, err)
		woc := controller2.newWorkflowOperationCtx(wf, controller)
		woc.operate()

		b,_ := json.Marshal(woc.wf)
		fmt.Println(string(b))
	})

}