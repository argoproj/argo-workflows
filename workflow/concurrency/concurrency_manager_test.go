package concurrency

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
 template: "1"
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

const wfWithTmplSemaphore = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: semaphore-tmpl-level-xjvln
  namespace: default
spec:
  arguments: {}
  entrypoint: semaphore-tmpl-level-example
  templates:
  - arguments: {}
    inputs: {}
    metadata: {}
    name: semaphore-tmpl-level-example
    outputs: {}
    steps:
    - - arguments: {}
        name: generate
        template: gen-number-list
    - - arguments:
          parameters:
          - name: seconds
            value: '{{item}}'
        name: sleep
        template: sleep-n-sec
        withParam: '{{steps.generate.outputs.result}}'
  - arguments: {}
    inputs: {}
    metadata: {}
    name: gen-number-list
    outputs: {}
    script:
      command:
      - python
      image: python:alpine3.6
      name: ""
      resources: {}
      source: |
        import json
        import sys
        json.dump([i for i in range(1, 3)], sys.stdout)
  - arguments: {}
    container:
      args:
      - echo sleeping for {{inputs.parameters.seconds}} seconds; sleep 10; echo done
      command:
      - sh
      - -c
      image: alpine:latest
      name: ""
      resources: {}
    inputs:
      parameters:
      - name: seconds
    metadata: {}
    name: sleep-n-sec
    outputs: {}
    semaphore:
      configMapKeyRef:
        key: template
        name: my-config
status:
  finishedAt: null
  nodes:
    semaphore-tmpl-level-xjvln:
      children:
      - semaphore-tmpl-level-xjvln-2790796867
      displayName: semaphore-tmpl-level-xjvln
      finishedAt: null
      id: semaphore-tmpl-level-xjvln
      name: semaphore-tmpl-level-xjvln
      phase: Running
      startedAt: "2020-06-04T19:55:11Z"
      templateName: semaphore-tmpl-level-example
      templateScope: local/semaphore-tmpl-level-xjvln
      type: Steps
    semaphore-tmpl-level-xjvln-5807216:
      boundaryID: semaphore-tmpl-level-xjvln
      children:
      - semaphore-tmpl-level-xjvln-2858054438
      displayName: generate
      finishedAt: "2020-06-04T19:55:25Z"
      hostNodeName: k3d-k3s-default-server
      id: semaphore-tmpl-level-xjvln-5807216
      name: semaphore-tmpl-level-xjvln[0].generate
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
            key: semaphore-tmpl-level-xjvln/semaphore-tmpl-level-xjvln-5807216/main.log
            secretKeySecret:
              key: secretkey
              name: my-minio-cred
        exitCode: "0"
        result: '[1, 2]'
      phase: Succeeded
      resourcesDuration:
        cpu: 9
        memory: 9
      startedAt: "2020-06-04T19:55:11Z"
      templateName: gen-number-list
      templateScope: local/semaphore-tmpl-level-xjvln
      type: Pod
    semaphore-tmpl-level-xjvln-1607747183:
      boundaryID: semaphore-tmpl-level-xjvln
      displayName: sleep(1:2)
      finishedAt: null
      hostNodeName: k3d-k3s-default-server
      id: semaphore-tmpl-level-xjvln-1607747183
      inputs:
        parameters:
        - name: seconds
          value: "2"
      message: ContainerCreating
      name: semaphore-tmpl-level-xjvln[1].sleep(1:2)
      phase: Pending
      startedAt: "2020-06-04T19:55:56Z"
      templateName: sleep-n-sec
      templateScope: local/semaphore-tmpl-level-xjvln
      type: Pod
    semaphore-tmpl-level-xjvln-2790796867:
      boundaryID: semaphore-tmpl-level-xjvln
      children:
      - semaphore-tmpl-level-xjvln-5807216
      displayName: '[0]'
      finishedAt: "2020-06-04T19:55:56Z"
      id: semaphore-tmpl-level-xjvln-2790796867
      name: semaphore-tmpl-level-xjvln[0]
      phase: Succeeded
      startedAt: "2020-06-04T19:55:11Z"
      templateName: semaphore-tmpl-level-example
      templateScope: local/semaphore-tmpl-level-xjvln
      type: StepGroup
    semaphore-tmpl-level-xjvln-2858054438:
      boundaryID: semaphore-tmpl-level-xjvln
      children:
      - semaphore-tmpl-level-xjvln-3448864205
      - semaphore-tmpl-level-xjvln-1607747183
      displayName: '[1]'
      finishedAt: null
      id: semaphore-tmpl-level-xjvln-2858054438
      name: semaphore-tmpl-level-xjvln[1]
      phase: Running
      startedAt: "2020-06-04T19:55:56Z"
      templateName: semaphore-tmpl-level-example
      templateScope: local/semaphore-tmpl-level-xjvln
      type: StepGroup
    semaphore-tmpl-level-xjvln-3448864205:
      boundaryID: semaphore-tmpl-level-xjvln
      displayName: sleep(0:1)
      finishedAt: null
      hostNodeName: k3d-k3s-default-server
      id: semaphore-tmpl-level-xjvln-3448864205
      inputs:
        parameters:
        - name: seconds
          value: "1"
      message: ContainerCreating
      name: semaphore-tmpl-level-xjvln[1].sleep(0:1)
      phase: Pending
      startedAt: "2020-06-04T19:55:56Z"
      templateName: sleep-n-sec
      templateScope: local/semaphore-tmpl-level-xjvln
      type: Pod
  phase: Running
  startedAt: "2020-06-04T19:55:11Z"
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

		// Try to acquire again
		status, msg, err = concurrenyMgr.TryAcquire(holderKey, wf.Namespace, 0, time.Now(), wf.Spec.Semaphore, wf)
		assert.NoError(t, err)
		assert.True(t, status)

		wf1 := wf.DeepCopy()
		wf1.Name = "two"
		holderKey1 := concurrenyMgr.GetHolderKey(wf1, "")
		status, msg, err = concurrenyMgr.TryAcquire(holderKey1, wf1.Namespace, 0, time.Now(), wf1.Spec.Semaphore, wf1)
		assert.NoError(t, err)
		assert.NotEmpty(t, msg)
		assert.False(t, status)

		wf2 := wf.DeepCopy()
		wf1.Name = "three"
		holderKey2 := concurrenyMgr.GetHolderKey(wf1, "")
		status, msg, err = concurrenyMgr.TryAcquire(holderKey2, wf2.Namespace, 5, time.Now(), wf2.Spec.Semaphore, wf2)
		assert.NoError(t, err)
		assert.NotEmpty(t, msg)
		assert.False(t, status)

		concurrenyMgr.Release(holderKey, wf.Namespace, wf.Spec.Semaphore, wf)
		assert.Equal(t, holderKey2, nextKey)
		assert.NotNil(t, wf.Status.ConcurrencyLockStatus)
		assert.Equal(t, 0, len(wf.Status.ConcurrencyLockStatus.SemaphoreHolders))

		// Low priority workflow try to acquire the lock
		status, msg, err = concurrenyMgr.TryAcquire(holderKey1, wf1.Namespace, 0, time.Now(), wf1.Spec.Semaphore, wf1)
		assert.NoError(t, err)
		assert.NotEmpty(t, msg)
		assert.False(t, status)

		// High Priority workflow acquires the lock
		status, msg, err = concurrenyMgr.TryAcquire(holderKey2, wf2.Namespace, 5, time.Now(), wf2.Spec.Semaphore, wf2)
		assert.NoError(t, err)
		assert.Empty(t, msg)
		assert.True(t, status)
		assert.NotNil(t, wf2.Status.ConcurrencyLockStatus)
		assert.NotNil(t, wf2.Status.ConcurrencyLockStatus.SemaphoreHolders)
		assert.Equal(t, SemaName, wf2.Status.ConcurrencyLockStatus.SemaphoreHolders[holderKey2])

		concurrenyMgr.ReleaseAll(wf2)
		assert.NotNil(t, wf2.Status.ConcurrencyLockStatus)
		assert.Equal(t, 0, len(wf2.Status.ConcurrencyLockStatus.SemaphoreHolders))
	})
}

func TestResizeSemaphoreSize(t *testing.T) {
	kube := fake.NewSimpleClientset()
	var cm v1.ConfigMap
	err := yaml.Unmarshal([]byte(configMap), &cm)
	assert.NoError(t, err)
	_, err = kube.CoreV1().ConfigMaps("default").Create(&cm)
	assert.NoError(t, err)
	t.Run("WfLevelAcquireAndRelease", func(t *testing.T) {
		concurrenyMgr := NewConcurrencyManager(kube, func(key string) {
		})
		createTime := time.Now()
		wf := unmarshalWF(wfWithSemaphore)
		holderKey := concurrenyMgr.GetHolderKey(wf, "")
		SemaName := getSemaphoreRefKey(wf.Namespace, wf.Spec.Semaphore)
		status, msg, err := concurrenyMgr.TryAcquire(holderKey, wf.Namespace, 0, createTime, wf.Spec.Semaphore, wf)
		assert.NoError(t, err)
		assert.Empty(t, msg)
		assert.True(t, status)
		assert.NotNil(t, wf.Status.ConcurrencyLockStatus)
		assert.NotNil(t, wf.Status.ConcurrencyLockStatus.SemaphoreHolders)
		assert.Equal(t, SemaName, wf.Status.ConcurrencyLockStatus.SemaphoreHolders[holderKey])

		wf1 := wf.DeepCopy()
		wf1.Name = "two"
		holderKey1 := concurrenyMgr.GetHolderKey(wf1, "")
		status, msg, err = concurrenyMgr.TryAcquire(holderKey1, wf1.Namespace, 0, createTime, wf1.Spec.Semaphore, wf1)
		assert.NoError(t, err)
		assert.NotEmpty(t, msg)
		assert.False(t, status)

		wf2 := wf.DeepCopy()
		wf1.Name = "three"
		holderKey2 := concurrenyMgr.GetHolderKey(wf1, "")
		status, msg, err = concurrenyMgr.TryAcquire(holderKey2, wf2.Namespace, 0, createTime.Add(1*time.Millisecond), wf2.Spec.Semaphore, wf2)
		assert.NoError(t, err)
		assert.NotEmpty(t, msg)
		assert.False(t, status)

		// Increase the Semaphore Size
		cm, err := kube.CoreV1().ConfigMaps("default").Get("my-config", metav1.GetOptions{})
		assert.NoError(t, err)
		cm.Data["workflow"] = "3"
		_, err = kube.CoreV1().ConfigMaps("default").Update(cm)
		assert.NoError(t, err)

		status, msg, err = concurrenyMgr.TryAcquire(holderKey1, wf1.Namespace, 0, createTime, wf1.Spec.Semaphore, wf1)
		assert.NoError(t, err)
		assert.True(t, status)
		assert.Empty(t, msg)
		assert.NotNil(t, wf.Status.ConcurrencyLockStatus)
		assert.NotNil(t, wf.Status.ConcurrencyLockStatus.SemaphoreHolders)
		assert.Equal(t, SemaName, wf.Status.ConcurrencyLockStatus.SemaphoreHolders[holderKey])

		status, msg, err = concurrenyMgr.TryAcquire(holderKey2, wf2.Namespace, 0, createTime.Add(1*time.Millisecond), wf2.Spec.Semaphore, wf2)
		assert.NoError(t, err)
		assert.Empty(t, msg)
		assert.True(t, status)
		assert.NotNil(t, wf2.Status.ConcurrencyLockStatus)
		assert.NotNil(t, wf2.Status.ConcurrencyLockStatus.SemaphoreHolders)
		assert.Equal(t, SemaName, wf2.Status.ConcurrencyLockStatus.SemaphoreHolders[holderKey2])

	})
}


func TestSemaphoreTmplLevel(t *testing.T) {
	kube := fake.NewSimpleClientset()
	var cm v1.ConfigMap
	err := yaml.Unmarshal([]byte(configMap), &cm)
	assert.NoError(t, err)
	_, err = kube.CoreV1().ConfigMaps("default").Create(&cm)
	assert.NoError(t, err)

	t.Run("TemplateLevelAcquireAndRelease", func(t *testing.T) {
		//var nextKey string
		concurrenyMgr := NewConcurrencyManager(kube, func(key string) {
			//nextKey = key
		})
		wf := unmarshalWF(wfWithTmplSemaphore)
		tmpl := wf.Spec.Templates[2]
		holderKey := concurrenyMgr.GetHolderKey(wf, "semaphore-tmpl-level-xjvln-3448864205")
		SemaName := getSemaphoreRefKey(wf.Namespace, tmpl.Semaphore)
		status, msg, err := concurrenyMgr.TryAcquire(holderKey, wf.Namespace, 0, time.Now(), tmpl.Semaphore, wf)
		assert.NoError(t, err)
		assert.Empty(t, msg)
		assert.True(t, status)
		assert.NotNil(t, wf.Status.ConcurrencyLockStatus)
		assert.NotNil(t, wf.Status.ConcurrencyLockStatus.SemaphoreHolders)
		assert.Equal(t, SemaName, wf.Status.ConcurrencyLockStatus.SemaphoreHolders[holderKey])

		// Try to acquire again
		status, msg, err = concurrenyMgr.TryAcquire(holderKey, wf.Namespace, 0, time.Now(), tmpl.Semaphore, wf)
		assert.NoError(t, err)
		assert.True(t, status)

		holderKey1 := concurrenyMgr.GetHolderKey(wf, "semaphore-tmpl-level-xjvln-1607747183")
		status, msg, err = concurrenyMgr.TryAcquire(holderKey1, wf.Namespace, 0, time.Now(), tmpl.Semaphore, wf)
		assert.NoError(t, err)
		assert.NotEmpty(t, msg)
		assert.False(t, status)

		concurrenyMgr.Release(holderKey, wf.Namespace, tmpl.Semaphore, wf)
		assert.NotNil(t, wf.Status.ConcurrencyLockStatus)
		assert.NotNil(t, wf.Status.ConcurrencyLockStatus.SemaphoreHolders)
		assert.Empty(t, wf.Status.ConcurrencyLockStatus.SemaphoreHolders[holderKey])

		status, msg, err = concurrenyMgr.TryAcquire(holderKey1, wf.Namespace, 0, time.Now(), tmpl.Semaphore, wf)
		assert.NoError(t, err)
		assert.Empty(t, msg)
		assert.True(t, status)
		assert.NotNil(t, wf.Status.ConcurrencyLockStatus)
		assert.NotNil(t, wf.Status.ConcurrencyLockStatus.SemaphoreHolders)
		assert.Equal(t, SemaName, wf.Status.ConcurrencyLockStatus.SemaphoreHolders[holderKey1])

	})
}