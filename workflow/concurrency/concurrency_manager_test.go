package concurrency

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes/fake"
	"sigs.k8s.io/yaml"

	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	fakewfclientset "github.com/argoproj/argo/pkg/client/clientset/versioned/fake"
)

const configMap = `
apiVersion: v1
kind: ConfigMap
metadata:
 name: my-config
data:
 workflow: "1"
 template: "2"
`
const wfWithStatus = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  creationTimestamp: "2020-06-03T23:06:35Z"
  generateName: hello-world-
  generation: 4
  labels:
    workflows.argoproj.io/phase: Running
  name: hello-world-vcrg5
  namespace: argo
spec:
  entrypoint: whalesay
  semaphore:
    configMapKeyRef:
      key: workflow
      name: my-config
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
  concurrencyLockStatus:
    semaphoreHolders:
      argo/hello-world-vcrg5: default/configmap/my-config/workflow
  finishedAt: null
  nodes:
    hello-world-vcrg5:
      displayName: hello-world-vcrg5
      finishedAt: null
      hostNodeName: k3d-k3s-default-server
      id: hello-world-vcrg5
      name: hello-world-vcrg5
      phase: Running
      startedAt: "2020-06-03T23:06:35Z"
      templateName: whalesay
      templateScope: local/hello-world-vcrg5
      type: Pod
  phase: Running
  startedAt: "2020-06-03T23:06:35Z"
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

func unmarshalWF(yamlStr string) *wfv1.Workflow {
	var wf wfv1.Workflow
	err := yaml.Unmarshal([]byte(yamlStr), &wf)
	if err != nil {
		panic(err)
	}
	return &wf
}

func TestSemaphoreWfLevel(t *testing.T) {
	kube := fake.NewSimpleClientset()
	var cm v1.ConfigMap
	err := yaml.Unmarshal([]byte(configMap), &cm)
	assert.NoError(t, err)
	_, err = kube.CoreV1().ConfigMaps("default").Create(&cm)
	assert.NoError(t, err)
	t.Run("InitializeConcurrency", func(t *testing.T) {
		concurrenyMgr := NewConcurrencyManager(kube, func(key string) {
		})
		wf := unmarshalWF(wfWithStatus)
		wfclientset := fakewfclientset.NewSimpleClientset(wf)
		concurrenyMgr.Initialize(wf.Namespace, wfclientset)

		assert.Equal(t, 1, len(concurrenyMgr.semaphoreMap))
	})
	t.Run("InitializeConcurrencyWithInvalid", func(t *testing.T) {
		concurrenyMgr := NewConcurrencyManager(kube, func(key string) {

		})
		wf := unmarshalWF(wfWithStatus)
		invalidMap := map[string]string{
			"argo/hello-world-vcrg5": "default/configmap/my-config1/workflow",
		}
		wf.Status.ConcurrencyLockStatus.SemaphoreHolders = invalidMap
		wfclientset := fakewfclientset.NewSimpleClientset(wf)
		concurrenyMgr.Initialize(wf.Namespace, wfclientset)
		assert.Equal(t, 0, len(concurrenyMgr.semaphoreMap))
	})
	t.Run("WfLevelAcquireAndRelease", func(t *testing.T) {
		var nextKey string
		concurrenyMgr := NewConcurrencyManager(kube, func(key string) {
			nextKey = key
		})
		wf := unmarshalWF(wfWithSemaphore)
		holderKey := concurrenyMgr.GetHolderKey(wf, "")
		SemaName := getSemaphoreRefKey(wf.Namespace, wf.Spec.Semaphore)
		status, msg, err := concurrenyMgr.TryAcquire(holderKey, wf.Namespace, 0, time.Now(), wf.Spec.Semaphore, wf)
		assert.NoError(t, err)
		assert.Empty(t, msg)
		assert.True(t, status)
		assert.NotNil(t, wf.Status.ConcurrencyLockStatus)
		assert.NotNil(t, wf.Status.ConcurrencyLockStatus.SemaphoreHolders)
		assert.Equal(t, SemaName, wf.Status.ConcurrencyLockStatus.SemaphoreHolders[holderKey])

		wf1 := wf.DeepCopy()
		wf1.Name = "two"
		holderKey1 := concurrenyMgr.GetHolderKey(wf1, "")
		status, msg, err = concurrenyMgr.TryAcquire(holderKey1, wf1.Namespace, 0, time.Now(), wf1.Spec.Semaphore, wf1)
		assert.NoError(t, err)
		assert.NotEmpty(t, msg)
		assert.False(t, status)

		concurrenyMgr.Release(holderKey, wf.Namespace, wf.Spec.Semaphore, wf)
		assert.Equal(t, holderKey1, nextKey)
		assert.NotNil(t, wf.Status.ConcurrencyLockStatus)
		assert.Equal(t, 0, len(wf.Status.ConcurrencyLockStatus.SemaphoreHolders))

		status, msg, err = concurrenyMgr.TryAcquire(holderKey, wf1.Namespace, 0, time.Now(), wf1.Spec.Semaphore, wf1)
		assert.NoError(t, err)
		assert.Empty(t, msg)
		assert.True(t, status)
		assert.NotNil(t, wf1.Status.ConcurrencyLockStatus)
		assert.NotNil(t, wf1.Status.ConcurrencyLockStatus.SemaphoreHolders)
		assert.Equal(t, SemaName, wf1.Status.ConcurrencyLockStatus.SemaphoreHolders[holderKey])

		concurrenyMgr.ReleaseAll(wf1)
		assert.NotNil(t, wf1.Status.ConcurrencyLockStatus)
		assert.Equal(t, 0, len(wf1.Status.ConcurrencyLockStatus.SemaphoreHolders))
	})

}
