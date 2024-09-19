package sync

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/utils/pointer"

	argoErr "github.com/argoproj/argo-workflows/v3/errors"
	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	fakewfclientset "github.com/argoproj/argo-workflows/v3/pkg/client/clientset/versioned/fake"
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
  creationTimestamp: "2020-06-19T17:37:05Z"
  generateName: hello-world-
  generation: 4
  labels:
    workflows.argoproj.io/phase: Running
  name: hello-world-prtl9
  namespace: default
  resourceVersion: "844854"
  selfLink: /apis/argoproj.io/v1alpha1/namespaces/default/workflows/hello-world-prtl9
  uid: 790f5c47-211f-4a3b-8949-514ae916633b
spec:

  entrypoint: whalesay
  synchronization:
    semaphore:
      configMapKeyRef:
        key: workflow
        name: my-config
  templates:
  -
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
  finishedAt: null
  nodes:
    hello-world-prtl9:
      displayName: hello-world-prtl9
      finishedAt: null
      hostNodeName: docker-desktop
      id: hello-world-prtl9
      message: ContainerCreating
      name: hello-world-prtl9
      phase: Pending
      startedAt: "2020-06-19T17:37:05Z"
      templateName: whalesay
      templateScope: local/hello-world-prtl9
      type: Pod
  phase: Running
  startedAt: "2020-06-19T17:37:05Z"
  synchronization:
    semaphore:
      holding:
      - holders:
        - hello-world-prtl9
        semaphore: default/configmap/my-config/workflow
`

const wfWithSemaphore = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
 name: hello-world
 namespace: default
spec:
 entrypoint: whalesay
 synchronization:
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

  entrypoint: semaphore-tmpl-level-example
  templates:
  -
    inputs: {}
    metadata: {}
    name: semaphore-tmpl-level-example
    outputs: {}
    steps:
    - -
        name: generate
        template: gen-number-list
    - - arguments:
          parameters:
          - name: seconds
            value: '{{item}}'
        name: sleep
        template: sleep-n-sec
        withParam: '{{steps.generate.outputs.result}}'
  -
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
  -
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
    synchronization:
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

const wfWithMutex = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
 name: hello-world
 namespace: default
spec:
 entrypoint: whalesay
 synchronization:
   mutex:
     name: my-mutex
 templates:
 - name: whalesay
   container:
     image: docker/whalesay:latest
     command: [cowsay]
     args: ["hello world"]
`

var WorkflowExistenceFunc = func(s string) bool {
	return false
}

func GetSyncLimitFunc(kube *fake.Clientset) func(string) (int, error) {
	syncLimitConfig := func(lockName string) (int, error) {
		items := strings.Split(lockName, "/")
		if len(items) < 4 {
			return 0, argoErr.New(argoErr.CodeBadRequest, "Invalid Config Map Key")
		}

		ctx := context.Background()
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

func TestSemaphoreWfLevel(t *testing.T) {
	kube := fake.NewSimpleClientset()
	var cm v1.ConfigMap
	wfv1.MustUnmarshal([]byte(configMap), &cm)

	ctx := context.Background()
	_, err := kube.CoreV1().ConfigMaps("default").Create(ctx, &cm, metav1.CreateOptions{})
	assert.NoError(t, err)

	syncLimitFunc := GetSyncLimitFunc(kube)
	t.Run("InitializeSynchronization", func(t *testing.T) {
		concurrenyMgr := NewLockManager(syncLimitFunc, func(key string) {
		}, WorkflowExistenceFunc)
		wf := wfv1.MustUnmarshalWorkflow(wfWithStatus)
		wfclientset := fakewfclientset.NewSimpleClientset(wf)

		wfList, err := wfclientset.ArgoprojV1alpha1().Workflows("default").List(ctx, metav1.ListOptions{})
		assert.NoError(t, err)
		concurrenyMgr.Initialize(wfList.Items)
		assert.Equal(t, 1, len(concurrenyMgr.syncLockMap))
	})
	t.Run("InitializeSynchronizationWithInvalid", func(t *testing.T) {
		concurrenyMgr := NewLockManager(syncLimitFunc, func(key string) {
		}, WorkflowExistenceFunc)
		wf := wfv1.MustUnmarshalWorkflow(wfWithStatus)
		invalidSync := []wfv1.SemaphoreHolding{{Semaphore: "default/configmap/my-config1/workflow", Holders: []string{"hello-world-vcrg5"}}}
		wf.Status.Synchronization.Semaphore.Holding = invalidSync
		wfclientset := fakewfclientset.NewSimpleClientset(wf)
		wfList, err := wfclientset.ArgoprojV1alpha1().Workflows("default").List(ctx, metav1.ListOptions{})
		assert.NoError(t, err)
		concurrenyMgr.Initialize(wfList.Items)
		assert.Equal(t, 0, len(concurrenyMgr.syncLockMap))
	})

	t.Run("WfLevelAcquireAndRelease", func(t *testing.T) {
		var nextKey string
		concurrenyMgr := NewLockManager(syncLimitFunc, func(key string) {
			nextKey = key
		}, WorkflowExistenceFunc)
		wf := wfv1.MustUnmarshalWorkflow(wfWithSemaphore)
		wf1 := wf.DeepCopy()
		wf2 := wf.DeepCopy()
		wf3 := wf.DeepCopy()
		status, wfUpdate, msg, err := concurrenyMgr.TryAcquire(wf, "", wf.Spec.Synchronization)
		assert.NoError(t, err)
		assert.Empty(t, msg)
		assert.True(t, status)
		assert.True(t, wfUpdate)
		assert.NotNil(t, wf.Status.Synchronization)
		assert.NotNil(t, wf.Status.Synchronization.Semaphore)
		assert.NotNil(t, wf.Status.Synchronization.Semaphore.Holding)
		key := getHolderKey(wf, "")
		assert.Equal(t, key, wf.Status.Synchronization.Semaphore.Holding[0].Holders[0])

		// Try to acquire again
		status, wfUpdate, msg, err = concurrenyMgr.TryAcquire(wf, "", wf.Spec.Synchronization)
		assert.NoError(t, err)
		assert.True(t, status)
		assert.Empty(t, msg)
		assert.False(t, wfUpdate)

		wf1.Name = "two"
		status, wfUpdate, msg, err = concurrenyMgr.TryAcquire(wf1, "", wf1.Spec.Synchronization)
		assert.NoError(t, err)
		assert.NotEmpty(t, msg)
		assert.False(t, status)
		assert.True(t, wfUpdate)

		wf2.Name = "three"
		wf2.Spec.Priority = pointer.Int32Ptr(5)
		holderKey2 := getHolderKey(wf2, "")
		status, wfUpdate, msg, err = concurrenyMgr.TryAcquire(wf2, "", wf2.Spec.Synchronization)
		assert.NoError(t, err)
		assert.NotEmpty(t, msg)
		assert.False(t, status)
		assert.True(t, wfUpdate)

		wf3.Name = "four"
		status, wfUpdate, msg, err = concurrenyMgr.TryAcquire(wf3, "", wf3.Spec.Synchronization)
		assert.NoError(t, err)
		assert.NotEmpty(t, msg)
		assert.False(t, status)
		assert.True(t, wfUpdate)

		concurrenyMgr.Release(wf, "", wf.Spec.Synchronization)
		assert.Equal(t, holderKey2, nextKey)
		assert.NotNil(t, wf.Status.Synchronization)
		assert.Equal(t, 0, len(wf.Status.Synchronization.Semaphore.Holding[0].Holders))

		// Low priority workflow try to acquire the lock
		status, wfUpdate, msg, err = concurrenyMgr.TryAcquire(wf1, "", wf1.Spec.Synchronization)
		assert.NoError(t, err)
		assert.NotEmpty(t, msg)
		assert.False(t, status)
		assert.True(t, wfUpdate)

		// High Priority workflow acquires the lock
		status, wfUpdate, msg, err = concurrenyMgr.TryAcquire(wf2, "", wf2.Spec.Synchronization)
		assert.NoError(t, err)
		assert.Empty(t, msg)
		assert.True(t, status)
		assert.True(t, wfUpdate)
		assert.NotNil(t, wf2.Status.Synchronization)
		assert.NotNil(t, wf2.Status.Synchronization.Semaphore)
		key = getHolderKey(wf2, "")
		assert.Equal(t, key, wf2.Status.Synchronization.Semaphore.Holding[0].Holders[0])

		concurrenyMgr.ReleaseAll(wf2)
		assert.Nil(t, wf2.Status.Synchronization)

		sema := concurrenyMgr.syncLockMap["default/ConfigMap/my-config/workflow"].(*PrioritySemaphore)
		assert.NotNil(t, sema)
		assert.Len(t, sema.pending.items, 2)
		concurrenyMgr.ReleaseAll(wf1)
		assert.Len(t, sema.pending.items, 1)
		concurrenyMgr.ReleaseAll(wf3)
		assert.Len(t, sema.pending.items, 0)
	})
}

func TestResizeSemaphoreSize(t *testing.T) {
	kube := fake.NewSimpleClientset()
	var cm v1.ConfigMap
	wfv1.MustUnmarshal([]byte(configMap), &cm)

	ctx := context.Background()
	_, err := kube.CoreV1().ConfigMaps("default").Create(ctx, &cm, metav1.CreateOptions{})
	assert.NoError(t, err)

	syncLimitFunc := GetSyncLimitFunc(kube)
	t.Run("WfLevelAcquireAndRelease", func(t *testing.T) {
		concurrenyMgr := NewLockManager(syncLimitFunc, func(key string) {
		}, WorkflowExistenceFunc)
		wf := wfv1.MustUnmarshalWorkflow(wfWithSemaphore)
		wf.CreationTimestamp = metav1.Time{Time: time.Now()}
		wf1 := wf.DeepCopy()
		wf2 := wf.DeepCopy()
		status, wfUpdate, msg, err := concurrenyMgr.TryAcquire(wf, "", wf.Spec.Synchronization)
		assert.NoError(t, err)
		assert.Empty(t, msg)
		assert.True(t, status)
		assert.True(t, wfUpdate)
		assert.NotNil(t, wf.Status.Synchronization)
		assert.NotNil(t, wf.Status.Synchronization.Semaphore)
		key := getHolderKey(wf, "")
		assert.Equal(t, key, wf.Status.Synchronization.Semaphore.Holding[0].Holders[0])

		wf1.Name = "two"
		status, wfUpdate, msg, err = concurrenyMgr.TryAcquire(wf1, "", wf1.Spec.Synchronization)
		assert.NoError(t, err)
		assert.NotEmpty(t, msg)
		assert.False(t, status)
		assert.True(t, wfUpdate)

		wf2.Name = "three"
		status, wfUpdate, msg, err = concurrenyMgr.TryAcquire(wf2, "", wf2.Spec.Synchronization)
		assert.NoError(t, err)
		assert.NotEmpty(t, msg)
		assert.False(t, status)
		assert.True(t, wfUpdate)

		// Increase the semaphore Size
		cm, err := kube.CoreV1().ConfigMaps("default").Get(ctx, "my-config", metav1.GetOptions{})
		assert.NoError(t, err)
		cm.Data["workflow"] = "3"
		_, err = kube.CoreV1().ConfigMaps("default").Update(ctx, cm, metav1.UpdateOptions{})
		assert.NoError(t, err)

		status, wfUpdate, msg, err = concurrenyMgr.TryAcquire(wf1, "", wf1.Spec.Synchronization)
		assert.NoError(t, err)
		assert.True(t, status)
		assert.Empty(t, msg)
		assert.True(t, wfUpdate)
		assert.NotNil(t, wf1.Status.Synchronization)
		assert.NotNil(t, wf1.Status.Synchronization.Semaphore)
		key = getHolderKey(wf1, "")
		assert.Equal(t, key, wf1.Status.Synchronization.Semaphore.Holding[0].Holders[0])

		status, wfUpdate, msg, err = concurrenyMgr.TryAcquire(wf2, "", wf2.Spec.Synchronization)
		assert.NoError(t, err)
		assert.Empty(t, msg)
		assert.True(t, status)
		assert.True(t, wfUpdate)
		assert.NotNil(t, wf2.Status.Synchronization)
		assert.NotNil(t, wf2.Status.Synchronization.Semaphore)
		key = getHolderKey(wf2, "")
		assert.Equal(t, key, wf2.Status.Synchronization.Semaphore.Holding[0].Holders[0])
	})
}

func TestSemaphoreTmplLevel(t *testing.T) {
	kube := fake.NewSimpleClientset()
	var cm v1.ConfigMap
	wfv1.MustUnmarshal([]byte(configMap), &cm)

	ctx := context.Background()
	_, err := kube.CoreV1().ConfigMaps("default").Create(ctx, &cm, metav1.CreateOptions{})
	assert.NoError(t, err)

	syncLimitFunc := GetSyncLimitFunc(kube)
	t.Run("TemplateLevelAcquireAndRelease", func(t *testing.T) {
		// var nextKey string
		concurrenyMgr := NewLockManager(syncLimitFunc, func(key string) {
			// nextKey = key
		}, WorkflowExistenceFunc)
		wf := wfv1.MustUnmarshalWorkflow(wfWithTmplSemaphore)
		tmpl := wf.Spec.Templates[2]

		status, wfUpdate, msg, err := concurrenyMgr.TryAcquire(wf, "semaphore-tmpl-level-xjvln-3448864205", tmpl.Synchronization)
		assert.NoError(t, err)
		assert.Empty(t, msg)
		assert.True(t, status)
		assert.True(t, wfUpdate)
		assert.NotNil(t, wf.Status.Synchronization)
		assert.NotNil(t, wf.Status.Synchronization.Semaphore)
		key := getHolderKey(wf, "semaphore-tmpl-level-xjvln-3448864205")
		assert.Equal(t, key, wf.Status.Synchronization.Semaphore.Holding[0].Holders[0])

		// Try to acquire again
		status, wfUpdate, msg, err = concurrenyMgr.TryAcquire(wf, "semaphore-tmpl-level-xjvln-3448864205", tmpl.Synchronization)
		assert.NoError(t, err)
		assert.True(t, status)
		assert.False(t, wfUpdate)
		assert.Empty(t, msg)

		status, wfUpdate, msg, err = concurrenyMgr.TryAcquire(wf, "semaphore-tmpl-level-xjvln-1607747183", tmpl.Synchronization)
		assert.NoError(t, err)
		assert.NotEmpty(t, msg)
		assert.True(t, wfUpdate)
		assert.False(t, status)

		concurrenyMgr.Release(wf, "semaphore-tmpl-level-xjvln-3448864205", tmpl.Synchronization)
		assert.NotNil(t, wf.Status.Synchronization)
		assert.NotNil(t, wf.Status.Synchronization.Semaphore)
		assert.Empty(t, wf.Status.Synchronization.Semaphore.Holding[0].Holders)

		status, wfUpdate, msg, err = concurrenyMgr.TryAcquire(wf, "semaphore-tmpl-level-xjvln-1607747183", tmpl.Synchronization)
		assert.NoError(t, err)
		assert.Empty(t, msg)
		assert.True(t, status)
		assert.True(t, wfUpdate)
		assert.NotNil(t, wf.Status.Synchronization)
		assert.NotNil(t, wf.Status.Synchronization.Semaphore)
		key = getHolderKey(wf, "semaphore-tmpl-level-xjvln-1607747183")
		assert.Equal(t, key, wf.Status.Synchronization.Semaphore.Holding[0].Holders[0])
	})
}

func TestTriggerWFWithAvailableLock(t *testing.T) {
	assert := assert.New(t)
	kube := fake.NewSimpleClientset()
	var cm v1.ConfigMap
	wfv1.MustUnmarshal([]byte(configMap), &cm)
	cm.Data["workflow"] = "3"

	ctx := context.Background()
	_, err := kube.CoreV1().ConfigMaps("default").Create(ctx, &cm, metav1.CreateOptions{})
	assert.NoError(err)

	syncLimitFunc := GetSyncLimitFunc(kube)
	t.Run("TriggerWfsWithAvailableLocks", func(t *testing.T) {
		triggerCount := 0
		concurrenyMgr := NewLockManager(syncLimitFunc, func(key string) {
			triggerCount++
		}, WorkflowExistenceFunc)
		var wfs []wfv1.Workflow
		for i := 0; i < 3; i++ {
			wf := wfv1.MustUnmarshalWorkflow(wfWithSemaphore)
			wf.Name = fmt.Sprintf("%s-%d", "acquired", i)
			status, wfUpdate, msg, err := concurrenyMgr.TryAcquire(wf, "", wf.Spec.Synchronization)
			assert.NoError(err)
			assert.Empty(msg)
			assert.True(status)
			assert.True(wfUpdate)
			wfs = append(wfs, *wf)

		}
		for i := 0; i < 3; i++ {
			wf := wfv1.MustUnmarshalWorkflow(wfWithSemaphore)
			wf.Name = fmt.Sprintf("%s-%d", "wait", i)
			status, wfUpdate, msg, err := concurrenyMgr.TryAcquire(wf, "", wf.Spec.Synchronization)
			assert.NoError(err)
			assert.NotEmpty(msg)
			assert.False(status)
			assert.True(wfUpdate)
		}
		concurrenyMgr.Release(&wfs[0], "", wfs[0].Spec.Synchronization)
		triggerCount = 0
		concurrenyMgr.Release(&wfs[1], "", wfs[1].Spec.Synchronization)
		assert.Equal(2, triggerCount)
	})
}

func TestMutexWfLevel(t *testing.T) {
	kube := fake.NewSimpleClientset()
	syncLimitFunc := GetSyncLimitFunc(kube)
	t.Run("WorkflowLevelMutexAcquireAndRelease", func(t *testing.T) {
		// var nextKey string
		concurrenyMgr := NewLockManager(syncLimitFunc, func(key string) {
			// nextKey = key
		}, WorkflowExistenceFunc)
		wf := wfv1.MustUnmarshalWorkflow(wfWithMutex)
		wf1 := wf.DeepCopy()
		wf2 := wf.DeepCopy()

		status, wfUpdate, msg, err := concurrenyMgr.TryAcquire(wf, "", wf.Spec.Synchronization)
		assert.NoError(t, err)
		assert.Empty(t, msg)
		assert.True(t, status)
		assert.True(t, wfUpdate)
		assert.NotNil(t, wf.Status.Synchronization)
		assert.NotNil(t, wf.Status.Synchronization.Mutex)
		assert.NotNil(t, wf.Status.Synchronization.Mutex.Holding)

		wf1.Name = "two"
		status, wfUpdate, msg, err = concurrenyMgr.TryAcquire(wf1, "", wf1.Spec.Synchronization)
		assert.NoError(t, err)
		assert.NotEmpty(t, msg)
		assert.False(t, status)
		assert.True(t, wfUpdate)

		wf2.Name = "three"
		status, wfUpdate, msg, err = concurrenyMgr.TryAcquire(wf2, "", wf2.Spec.Synchronization)
		assert.NoError(t, err)
		assert.NotEmpty(t, msg)
		assert.False(t, status)
		assert.True(t, wfUpdate)

		mutex := concurrenyMgr.syncLockMap["default/Mutex/my-mutex"].(*PriorityMutex)
		assert.NotNil(t, mutex)
		assert.Len(t, mutex.mutex.pending.items, 2)
		concurrenyMgr.ReleaseAll(wf1)
		assert.Len(t, mutex.mutex.pending.items, 1)
		concurrenyMgr.ReleaseAll(wf2)
		assert.Len(t, mutex.mutex.pending.items, 0)
	})
}

func TestCheckWorkflowExistence(t *testing.T) {
	assert := assert.New(t)
	kube := fake.NewSimpleClientset()
	var cm v1.ConfigMap
	wfv1.MustUnmarshal([]byte(configMap), &cm)
	cm.Data["workflow"] = "1"

	ctx := context.Background()
	_, err := kube.CoreV1().ConfigMaps("default").Create(ctx, &cm, metav1.CreateOptions{})
	assert.NoError(err)

	syncLimitFunc := GetSyncLimitFunc(kube)
	t.Run("WorkflowDeleted", func(t *testing.T) {
		concurrenyMgr := NewLockManager(syncLimitFunc, func(key string) {
			// nextKey = key
		}, func(s string) bool {
			return strings.Contains(s, "test1")
		})
		wfMutex := wfv1.MustUnmarshalWorkflow(wfWithMutex)
		wfMutex1 := wfMutex.DeepCopy()
		wfMutex1.Name = "test1"
		wfSema := wfv1.MustUnmarshalWorkflow(wfWithSemaphore)
		wfSema1 := wfSema.DeepCopy()
		wfSema1.Name = "test2"
		_, _, _, _ = concurrenyMgr.TryAcquire(wfMutex, "", wfMutex.Spec.Synchronization)
		_, _, _, _ = concurrenyMgr.TryAcquire(wfMutex1, "", wfMutex.Spec.Synchronization)
		_, _, _, _ = concurrenyMgr.TryAcquire(wfSema, "", wfSema.Spec.Synchronization)
		_, _, _, _ = concurrenyMgr.TryAcquire(wfSema1, "", wfSema.Spec.Synchronization)
		mutex := concurrenyMgr.syncLockMap["default/Mutex/my-mutex"].(*PriorityMutex)
		semaphore := concurrenyMgr.syncLockMap["default/ConfigMap/my-config/workflow"]

		assert.Len(mutex.getCurrentHolders(), 1)
		assert.Len(mutex.getCurrentPending(), 1)
		assert.Len(semaphore.getCurrentHolders(), 1)
		assert.Len(semaphore.getCurrentPending(), 1)
		concurrenyMgr.CheckWorkflowExistence()
		assert.Len(mutex.getCurrentHolders(), 0)
		assert.Len(mutex.getCurrentPending(), 1)
		assert.Len(semaphore.getCurrentHolders(), 0)
		assert.Len(semaphore.getCurrentPending(), 0)
	})
}

func TestTriggerWFWithSemaphoreAndMutex(t *testing.T) {
	assert := assert.New(t)
	kube := fake.NewSimpleClientset()
	var cm v1.ConfigMap
	wfv1.MustUnmarshal([]byte(configMap), &cm)
	cm.Data["test-sem"] = "1"

	ctx := context.Background()
	_, err := kube.CoreV1().ConfigMaps("default").Create(ctx, &cm, metav1.CreateOptions{})
	assert.NoError(err)
	wf := wfv1.MustUnmarshalWorkflow(`apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: synchronization-tmpl-level-sgg6t
  namespace: default
spec:
  entrypoint: main
  templates:
  - name: main
    steps:
    - - name: dag-1-task-1
        template: django-command
      - name: dag-1-task-2
        template: load-command
  - container:
      args:
      - S=$(shuf -i 2-2 -n 1); echo begin to sleep $S; sleep $S; echo acquired lock
      command:
      - sh
      - -c
      image: alpine:latest
      name: ""
      resources:
        requests:
          cpu: 100m
          memory: 100Mi
    name: load-command
    synchronization:
      mutex:
        name: dag-2-task-1
  - container:
      args:
      - echo 'django command!'
      command:
      - sh
      - -c
      image: alpine:latest
      name: ""
      resources:
        requests:
          cpu: 100m
          memory: 100Mi
    name: django-command
    synchronization:
      semaphore:
        configMapKeyRef:
          key: test-sem
          name: my-config
  ttlStrategy:
    secondsAfterCompletion: 600
status:
  conditions:
  - status: "True"
    type: PodRunning
  finishedAt: null
  nodes:
    synchronization-tmpl-level-sgg6t:
      children:
      - synchronization-tmpl-level-sgg6t-1530845388
      displayName: synchronization-tmpl-level-sgg6t
      finishedAt: null
      id: synchronization-tmpl-level-sgg6t
      name: synchronization-tmpl-level-sgg6t
      phase: Running
      progress: 1/2
      startedAt: "2022-02-28T18:13:00Z"
      templateName: main
      templateScope: local/synchronization-tmpl-level-sgg6t
      type: Steps
    synchronization-tmpl-level-sgg6t-1530845388:
      boundaryID: synchronization-tmpl-level-sgg6t
      children:
      - synchronization-tmpl-level-sgg6t-1899337224
      - synchronization-tmpl-level-sgg6t-1949670081
      displayName: '[0]'
      finishedAt: null
      id: synchronization-tmpl-level-sgg6t-1530845388
      name: synchronization-tmpl-level-sgg6t[0]
      phase: Running
      progress: 1/2
      startedAt: "2022-02-28T18:13:00Z"
      templateScope: local/synchronization-tmpl-level-sgg6t
      type: StepGroup
    synchronization-tmpl-level-sgg6t-1899337224:
      boundaryID: synchronization-tmpl-level-sgg6t
      displayName: dag-1-task-1
      finishedAt: "2022-02-28T18:13:04Z"
      hostNodeName: k3d-k3s-default-server-0
      id: synchronization-tmpl-level-sgg6t-1899337224
      name: synchronization-tmpl-level-sgg6t[0].dag-1-task-1
      outputs:
        artifacts:
        - name: main-logs
          s3:
            key: synchronization-tmpl-level-sgg6t/synchronization-tmpl-level-sgg6t-1899337224/main.log
        exitCode: "0"
      phase: Succeeded
      progress: 1/1
      resourcesDuration:
        cpu: 2
        memory: 1
      startedAt: "2022-02-28T18:13:00Z"
      templateName: django-command
      templateScope: local/synchronization-tmpl-level-sgg6t
      type: Pod
    synchronization-tmpl-level-sgg6t-1949670081:
      boundaryID: synchronization-tmpl-level-sgg6t
      displayName: dag-1-task-2
      finishedAt: null
      hostNodeName: k3d-k3s-default-server-0
      id: synchronization-tmpl-level-sgg6t-1949670081
      name: synchronization-tmpl-level-sgg6t[0].dag-1-task-2
      outputs:
        exitCode: "0"
      phase: Running
      progress: 0/1
      startedAt: "2022-02-28T18:13:00Z"
      templateName: load-command
      templateScope: local/synchronization-tmpl-level-sgg6t
      type: Pod
  phase: Running
  progress: 1/2
  resourcesDuration:
    cpu: 2
    memory: 1
  startedAt: "2022-02-28T18:13:00Z"
`)
	syncLimitFunc := GetSyncLimitFunc(kube)

	concurrenyMgr := NewLockManager(syncLimitFunc, func(key string) {
		// nextKey = key
	}, WorkflowExistenceFunc)
	t.Run("InitializeMutex", func(t *testing.T) {
		tmpl := wf.Spec.Templates[1]
		status, wfUpdate, msg, err := concurrenyMgr.TryAcquire(wf, "synchronization-tmpl-level-sgg6t-1949670081", tmpl.Synchronization)
		assert.NoError(err)
		assert.Empty(msg)
		assert.True(status)
		assert.True(wfUpdate)
		assert.NotNil(wf.Status.Synchronization)
		assert.NotNil(wf.Status.Synchronization.Mutex)
	})
	t.Run("InitializeSemaphore", func(t *testing.T) {
		tmpl := wf.Spec.Templates[2]
		status, wfUpdate, msg, err := concurrenyMgr.TryAcquire(wf, "synchronization-tmpl-level-sgg6t-1899337224", tmpl.Synchronization)
		assert.NoError(err)
		assert.Empty(msg)
		assert.True(status)
		assert.True(wfUpdate)
		assert.NotNil(wf.Status.Synchronization)
		assert.NotNil(wf.Status.Synchronization.Semaphore)
	})

}

const wfV2MutexMigrationWorkflowLevel = `apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  creationTimestamp: null
  name: test1
  namespace: default
spec:
  arguments: {}
  entrypoint: whalesay
  synchronization:
    mutex:
      name: my-mutex
  templates:
  - container:
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
  finishedAt: null
  startedAt: null
  synchronization:
    mutex:
      holding:
      - holder: test1
        mutex: default/Mutex/my-mutex

`

const wfV2MutexMigrationTemplateLevel = `apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  annotations:
    workflows.argoproj.io/pod-name-format: v2
  creationTimestamp: "2024-09-17T00:11:53Z"
  generateName: synchronization-tmpl-level-
  generation: 5
  labels:
    workflows.argoproj.io/completed: "false"
    workflows.argoproj.io/phase: Running
  name: synchronization-tmpl-level-xvzpt
  namespace: argo
  resourceVersion: "10182"
  uid: f2d4ac34-1495-48ba-8aab-25239880fef3
spec:
  activeDeadlineSeconds: 300
  arguments: {}
  entrypoint: synchronization-tmpl-level-example
  podSpecPatch: |
    terminationGracePeriodSeconds: 3
  templates:
  - inputs: {}
    metadata: {}
    name: synchronization-tmpl-level-example
    outputs: {}
    steps:
    - - arguments:
          parameters:
          - name: seconds
            value: '{{item}}'
        name: synchronization-acquire-lock
        template: acquire-lock
        withParam: '["1","2","3","4","5"]'
  - container:
      args:
      - sleep 60; echo acquired lock
      command:
      - sh
      - -c
      image: alpine:latest
      name: ""
      resources: {}
    inputs: {}
    metadata: {}
    name: acquire-lock
    outputs: {}
    synchronization:
      mutex:
        name: workflow
status:
  artifactGCStatus:
    notSpecified: true
  artifactRepositoryRef:
    artifactRepository:
      archiveLogs: true
      s3:
        accessKeySecret:
          key: accesskey
          name: my-minio-cred
        bucket: my-bucket
        endpoint: minio:9000
        insecure: true
        secretKeySecret:
          key: secretkey
          name: my-minio-cred
    configMap: artifact-repositories
    key: default-v1
    namespace: argo
  conditions:
  - status: "True"
    type: PodRunning
  finishedAt: null
  nodes:
    synchronization-tmpl-level-xvzpt:
      children:
      - synchronization-tmpl-level-xvzpt-2018718843
      displayName: synchronization-tmpl-level-xvzpt
      finishedAt: null
      id: synchronization-tmpl-level-xvzpt
      name: synchronization-tmpl-level-xvzpt
      phase: Running
      progress: 0/5
      startedAt: "2024-09-17T00:11:53Z"
      templateName: synchronization-tmpl-level-example
      templateScope: local/synchronization-tmpl-level-xvzpt
      type: Steps
    synchronization-tmpl-level-xvzpt-755731602:
      boundaryID: synchronization-tmpl-level-xvzpt
      displayName: synchronization-acquire-lock(1:2)
      finishedAt: null
      id: synchronization-tmpl-level-xvzpt-755731602
      message: 'Waiting for argo/Mutex/workflow lock. Lock status: 0/1'
      name: synchronization-tmpl-level-xvzpt[0].synchronization-acquire-lock(1:2)
      phase: Pending
      progress: 0/1
      startedAt: "2024-09-17T00:11:53Z"
      synchronizationStatus:
        waiting: argo/Mutex/workflow
      templateName: acquire-lock
      templateScope: local/synchronization-tmpl-level-xvzpt
      type: Pod
    synchronization-tmpl-level-xvzpt-928517240:
      boundaryID: synchronization-tmpl-level-xvzpt
      displayName: synchronization-acquire-lock(0:1)
      finishedAt: null
      hostNodeName: k3d-k3s-default-server-0
      id: synchronization-tmpl-level-xvzpt-928517240
      name: synchronization-tmpl-level-xvzpt[0].synchronization-acquire-lock(0:1)
      phase: Running
      progress: 0/1
      startedAt: "2024-09-17T00:11:53Z"
      templateName: acquire-lock
      templateScope: local/synchronization-tmpl-level-xvzpt
      type: Pod
    synchronization-tmpl-level-xvzpt-1018728496:
      boundaryID: synchronization-tmpl-level-xvzpt
      displayName: synchronization-acquire-lock(4:5)
      finishedAt: null
      id: synchronization-tmpl-level-xvzpt-1018728496
      message: 'Waiting for argo/Mutex/workflow lock. Lock status: 0/1'
      name: synchronization-tmpl-level-xvzpt[0].synchronization-acquire-lock(4:5)
      phase: Pending
      progress: 0/1
      startedAt: "2024-09-17T00:11:53Z"
      synchronizationStatus:
        waiting: argo/Mutex/workflow
      templateName: acquire-lock
      templateScope: local/synchronization-tmpl-level-xvzpt
      type: Pod
    synchronization-tmpl-level-xvzpt-2018718843:
      boundaryID: synchronization-tmpl-level-xvzpt
      children:
      - synchronization-tmpl-level-xvzpt-928517240
      - synchronization-tmpl-level-xvzpt-755731602
      - synchronization-tmpl-level-xvzpt-4037094368
      - synchronization-tmpl-level-xvzpt-3632956078
      - synchronization-tmpl-level-xvzpt-1018728496
      displayName: '[0]'
      finishedAt: null
      id: synchronization-tmpl-level-xvzpt-2018718843
      name: synchronization-tmpl-level-xvzpt[0]
      nodeFlag: {}
      phase: Running
      progress: 0/5
      startedAt: "2024-09-17T00:11:53Z"
      templateScope: local/synchronization-tmpl-level-xvzpt
      type: StepGroup
    synchronization-tmpl-level-xvzpt-3632956078:
      boundaryID: synchronization-tmpl-level-xvzpt
      displayName: synchronization-acquire-lock(3:4)
      finishedAt: null
      id: synchronization-tmpl-level-xvzpt-3632956078
      message: 'Waiting for argo/Mutex/workflow lock. Lock status: 0/1'
      name: synchronization-tmpl-level-xvzpt[0].synchronization-acquire-lock(3:4)
      phase: Pending
      progress: 0/1
      startedAt: "2024-09-17T00:11:53Z"
      synchronizationStatus:
        waiting: argo/Mutex/workflow
      templateName: acquire-lock
      templateScope: local/synchronization-tmpl-level-xvzpt
      type: Pod
    synchronization-tmpl-level-xvzpt-4037094368:
      boundaryID: synchronization-tmpl-level-xvzpt
      displayName: synchronization-acquire-lock(2:3)
      finishedAt: null
      id: synchronization-tmpl-level-xvzpt-4037094368
      message: 'Waiting for argo/Mutex/workflow lock. Lock status: 0/1'
      name: synchronization-tmpl-level-xvzpt[0].synchronization-acquire-lock(2:3)
      phase: Pending
      progress: 0/1
      startedAt: "2024-09-17T00:11:53Z"
      synchronizationStatus:
        waiting: argo/Mutex/workflow
      templateName: acquire-lock
      templateScope: local/synchronization-tmpl-level-xvzpt
      type: Pod
  phase: Running
  progress: 0/5
  startedAt: "2024-09-17T00:11:53Z"
  synchronization:
    mutex:
      holding:
      - holder: synchronization-tmpl-level-xvzpt-928517240
        mutex: argo/Mutex/workflow
      waiting:
      - holder: argo/synchronization-tmpl-level-xvzpt/synchronization-tmpl-level-xvzpt-928517240
        mutex: argo/Mutex/workflow
  taskResultsCompletionStatus:
    synchronization-tmpl-level-xvzpt-928517240: false
`

func TestMutexMigration(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)
	kube := fake.NewSimpleClientset()

	syncLimitFunc := GetSyncLimitFunc(kube)

	concurrenyMgr := NewLockManager(syncLimitFunc, func(key string) {
	}, WorkflowExistenceFunc)

	wfMutex := wfv1.MustUnmarshalWorkflow(wfWithMutex)

	t.Run("RunMigrationWorkflowLevel", func(t *testing.T) {
		concurrenyMgr.syncLockMap = make(map[string]Semaphore)
		wfMutex2 := wfv1.MustUnmarshalWorkflow(wfV2MutexMigrationWorkflowLevel)

		require.Len(wfMutex2.Status.Synchronization.Mutex.Holding, 1)
		holderKey := getHolderKey(wfMutex2, "")
		items := strings.Split(holderKey, "/")
		holdingName := items[len(items)-1]
		assert.Equal(wfMutex2.Status.Synchronization.Mutex.Holding[0].Holder, holdingName)

		concurrenyMgr.syncLockMap = make(map[string]Semaphore)
		wfs := []wfv1.Workflow{*wfMutex2.DeepCopy()}
		concurrenyMgr.Initialize(wfs)

		lockName, err := GetLockName(wfMutex2.Spec.Synchronization, wfMutex2.Namespace)
		require.NoError(err)

		sem, found := concurrenyMgr.syncLockMap[lockName.EncodeName()]
		require.True(found)

		holders := sem.getCurrentHolders()
		require.Len(holders, 1)

		// PROVE: bug absent
		assert.Equal(holderKey, holders[0])

		// We should already have this lock since we acquired it above
		status, _, _, err := concurrenyMgr.TryAcquire(wfMutex2, "", wfMutex.Spec.Synchronization)
		require.NoError(err)
		// BUG NOT PRESENT: https://github.com/argoproj/argo-workflows/issues/8684
		assert.True(status)
	})

	concurrenyMgr = NewLockManager(syncLimitFunc, func(key string) {
	}, WorkflowExistenceFunc)

	t.Run("RunMigrationTemplateLevel", func(t *testing.T) {
		concurrenyMgr.syncLockMap = make(map[string]Semaphore)
		wfMutex3 := wfv1.MustUnmarshalWorkflow(wfV2MutexMigrationTemplateLevel)
		require.Len(wfMutex3.Status.Synchronization.Mutex.Holding, 1)

		numFound := 0
		foundNodeID := ""
		for nodeID := range wfMutex3.Status.Nodes {
			holder := getHolderKey(wfMutex3, nodeID)
			if holder == getUpgradedKey(wfMutex3, wfMutex3.Status.Synchronization.Mutex.Holding[0].Holder, TemplateLevel) {
				foundNodeID = nodeID
				numFound++
			}
		}
		assert.Equal(1, numFound)

		wfs := []wfv1.Workflow{*wfMutex3.DeepCopy()}
		concurrenyMgr.Initialize(wfs)

		lockName, err := GetLockName(wfMutex3.Spec.Templates[1].Synchronization, wfMutex3.Namespace)
		require.NoError(err)

		sem, found := concurrenyMgr.syncLockMap[lockName.EncodeName()]
		require.True(found)

		holders := sem.getCurrentHolders()
		require.Len(holders, 1)

		holderKey := getHolderKey(wfMutex3, foundNodeID)

		// PROVE: bug absent
		assert.Equal(holderKey, holders[0])

		status, _, _, err := concurrenyMgr.TryAcquire(wfMutex3, foundNodeID, wfMutex.Spec.Synchronization)
		require.NoError(err)
		// BUG NOT PRESENT: https://github.com/argoproj/argo-workflows/issues/8684
		assert.True(status)
	})
}

// getHoldingNameV1 legacy code to get holding name.
func getHoldingNameV1(holderKey string) string {
	items := strings.Split(holderKey, "/")
	return items[len(items)-1]
}

func TestCheckHolderVersion(t *testing.T) {

	t.Run("CheckHolderKeyWithNodeName", func(t *testing.T) {
		assert := assert.New(t)
		wfMutex := wfv1.MustUnmarshalWorkflow(wfWithMutex)
		key := getHolderKey(wfMutex, wfMutex.Name)

		keyv2 := key
		version := wfv1.CheckHolderKeyVersion(keyv2)
		assert.Equal(wfv1.HoldingNameV2, version)

		keyv1 := getHoldingNameV1(key)
		version = wfv1.CheckHolderKeyVersion(keyv1)
		assert.Equal(wfv1.HoldingNameV1, version)

	})

	t.Run("CheckHolderKeyWithoutNodeName", func(t *testing.T) {
		assert := assert.New(t)
		wfMutex := wfv1.MustUnmarshalWorkflow(wfWithMutex)

		key := getHolderKey(wfMutex, "")
		keyv2 := key
		version := wfv1.CheckHolderKeyVersion(keyv2)
		assert.Equal(wfv1.HoldingNameV2, version)

		keyv1 := getHoldingNameV1(key)
		version = wfv1.CheckHolderKeyVersion(keyv1)
		assert.Equal(wfv1.HoldingNameV1, version)
	})
}
