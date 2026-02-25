package sync

import (
	"context"
	"fmt"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/argoproj/argo-workflows/v4/util/logging"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/utils/ptr"

	"github.com/argoproj/argo-workflows/v4/config"
	argoErr "github.com/argoproj/argo-workflows/v4/errors"
	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	fakewfclientset "github.com/argoproj/argo-workflows/v4/pkg/client/clientset/versioned/fake"
	"github.com/argoproj/argo-workflows/v4/util/sqldb"
	syncdb "github.com/argoproj/argo-workflows/v4/util/sync/db"
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
    semaphores:
      - configMapKeyRef:
          key: workflow
          name: my-config
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
        semaphore: default/ConfigMap/my-config/workflow
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
   semaphores:
     - configMapKeyRef:
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
  - inputs: {}
    metadata: {}
    name: semaphore-tmpl-level-example
    outputs: {}
    steps:
    - - name: generate
        template: gen-number-list
    - - arguments:
          parameters:
          - name: seconds
            value: '{{item}}'
        name: sleep
        template: sleep-n-sec
        withParam: '{{steps.generate.outputs.result}}'
  - inputs: {}
    metadata: {}
    name: gen-number-list
    outputs: {}
    script:
      command:
      - python
      image: python:alpine3.23
      name: ""
      resources: {}
      source: |
        import json
        import sys
        json.dump([i for i in range(1, 3)], sys.stdout)
  - container:
      args:
      - echo sleeping for {{inputs.parameters.seconds}} seconds; sleep 10; echo done
      command:
      - sh
      - -c
      image: alpine:3.23
      name: ""
      resources: {}
    inputs:
      parameters:
      - name: seconds
    metadata: {}
    name: sleep-n-sec
    outputs: {}
    synchronization:
      semaphores:
        - configMapKeyRef:
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
   mutexes:
     - name: my-mutex
 templates:
 - name: whalesay
   container:
     image: docker/whalesay:latest
     command: [cowsay]
     args: ["hello world"]
`

// Workflow with database semaphore
const wfWithDBSemaphore = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
 name: hello-world-db-sem
 namespace: default
spec:
 entrypoint: whalesay
 synchronization:
   semaphores:
     - database:
         key: my-database-sem
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

func GetSyncLimitFunc(kube *fake.Clientset) func(context.Context, string) (int, error) {
	return func(ctx context.Context, lockName string) (int, error) {
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
}

func TestSemaphoreWfLevel(t *testing.T) {
	kube := fake.NewSimpleClientset()
	var cm v1.ConfigMap
	wfv1.MustUnmarshal([]byte(configMap), &cm)

	ctx := logging.TestContext(t.Context())
	_, err := kube.CoreV1().ConfigMaps("default").Create(ctx, &cm, metav1.CreateOptions{})
	require.NoError(t, err)

	syncLimitFunc := GetSyncLimitFunc(kube)
	t.Run("InitializeSynchronization", func(t *testing.T) {
		syncManager := NewLockManager(ctx, kube, "", nil, syncLimitFunc, func(key string) {
		}, WorkflowExistenceFunc)
		wf := wfv1.MustUnmarshalWorkflow(wfWithStatus)
		wfclientset := fakewfclientset.NewSimpleClientset(wf)

		wfList, err := wfclientset.ArgoprojV1alpha1().Workflows("default").List(ctx, metav1.ListOptions{})
		require.NoError(t, err)
		syncManager.Initialize(ctx, wfList.Items)
		assert.Len(t, syncManager.syncLockMap, 1)
	})
	t.Run("InitializeSynchronizationWithInvalid", func(t *testing.T) {
		syncManager := NewLockManager(ctx, kube, "", nil, syncLimitFunc, func(key string) {
		}, WorkflowExistenceFunc)
		wf := wfv1.MustUnmarshalWorkflow(wfWithStatus)
		invalidSync := []wfv1.SemaphoreHolding{{Semaphore: "default/configmap/my-config1/workflow", Holders: []string{"hello-world-vcrg5"}}}
		wf.Status.Synchronization.Semaphore.Holding = invalidSync
		wfclientset := fakewfclientset.NewSimpleClientset(wf)
		wfList, err := wfclientset.ArgoprojV1alpha1().Workflows("default").List(ctx, metav1.ListOptions{})
		require.NoError(t, err)
		syncManager.Initialize(ctx, wfList.Items)
		assert.Empty(t, syncManager.syncLockMap)
	})
	t.Run("InitializeMultipleWorkflowsHolding", func(t *testing.T) {
		// This test verifies that when multiple workflows claim to hold the same semaphore
		// (which can happen with stale status after a controller restart), ALL of their
		// holders are registered during Initialize, not just those after the first workflow.
		// This was a bug caused by variable shadowing in Initialize (PR #3141).

		// Create a ConfigMap with semaphore limit of 3 to allow multiple holders
		kubeClient := fake.NewSimpleClientset()
		_, err := kubeClient.CoreV1().ConfigMaps("default").Create(ctx, &v1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{Name: "my-config"},
			Data:       map[string]string{"workflow": "3"},
		}, metav1.CreateOptions{})
		require.NoError(t, err)

		syncManager := NewLockManager(ctx, kubeClient, "", nil, GetSyncLimitFunc(kubeClient), func(key string) {
		}, WorkflowExistenceFunc)

		// Create first workflow claiming to hold the semaphore
		wf1 := wfv1.MustUnmarshalWorkflow(wfWithStatus)
		wf1.Name = "hello-world-one"
		wf1.Status.Synchronization.Semaphore.Holding[0].Holders = []string{"default/hello-world-one"}

		// Create second workflow also claiming to hold the same semaphore
		wf2 := wfv1.MustUnmarshalWorkflow(wfWithStatus)
		wf2.Name = "hello-world-two"
		wf2.Status.Synchronization.Semaphore.Holding[0].Holders = []string{"default/hello-world-two"}

		// Initialize with both workflows
		syncManager.Initialize(ctx, []wfv1.Workflow{*wf1, *wf2})

		// Verify the semaphore was created
		assert.Len(t, syncManager.syncLockMap, 1)

		// Verify BOTH holders are registered (the bug would only register the second one)
		sem := syncManager.syncLockMap["default/ConfigMap/my-config/workflow"]
		require.NotNil(t, sem)
		holders, err := sem.getCurrentHolders(ctx)
		require.NoError(t, err)
		assert.Len(t, holders, 2, "both workflows should be registered as holders")
		assert.Contains(t, holders, "default/hello-world-one")
		assert.Contains(t, holders, "default/hello-world-two")
	})

	t.Run("WfLevelAcquireAndRelease", func(t *testing.T) {
		var nextKey string
		syncManager := NewLockManager(ctx, kube, "", nil, syncLimitFunc, func(key string) {
			nextKey = key
		}, WorkflowExistenceFunc)
		wf := wfv1.MustUnmarshalWorkflow(wfWithSemaphore)
		wf1 := wf.DeepCopy()
		wf2 := wf.DeepCopy()
		wf3 := wf.DeepCopy()
		status, wfUpdate, msg, failedLockName, err := syncManager.TryAcquire(ctx, wf, "", wf.Spec.Synchronization)
		require.NoError(t, err)
		assert.Empty(t, msg)
		assert.Empty(t, failedLockName)
		assert.True(t, status)
		assert.True(t, wfUpdate)
		require.NotNil(t, wf.Status.Synchronization)
		require.NotNil(t, wf.Status.Synchronization.Semaphore)
		require.NotNil(t, wf.Status.Synchronization.Semaphore.Holding)
		key := getHolderKey(wf, "")
		assert.Equal(t, key, wf.Status.Synchronization.Semaphore.Holding[0].Holders[0])

		// Try to acquire again
		status, wfUpdate, msg, failedLockName, err = syncManager.TryAcquire(ctx, wf, "", wf.Spec.Synchronization)
		require.NoError(t, err)
		assert.True(t, status)
		assert.Empty(t, msg)
		assert.Empty(t, failedLockName)
		assert.False(t, wfUpdate)

		wf1.Name = "two"
		status, wfUpdate, msg, failedLockName, err = syncManager.TryAcquire(ctx, wf1, "", wf1.Spec.Synchronization)
		require.NoError(t, err)
		assert.NotEmpty(t, msg)
		assert.Equal(t, "default/ConfigMap/my-config/workflow", failedLockName)
		assert.False(t, status)
		assert.True(t, wfUpdate)

		wf2.Name = "three"
		wf2.Spec.Priority = ptr.To(int32(5))
		holderKey2 := getHolderKey(wf2, "")
		status, wfUpdate, msg, failedLockName, err = syncManager.TryAcquire(ctx, wf2, "", wf2.Spec.Synchronization)
		require.NoError(t, err)
		assert.NotEmpty(t, msg)
		assert.Equal(t, "default/ConfigMap/my-config/workflow", failedLockName)
		assert.False(t, status)
		assert.True(t, wfUpdate)

		wf3.Name = "four"
		status, wfUpdate, msg, failedLockName, err = syncManager.TryAcquire(ctx, wf3, "", wf3.Spec.Synchronization)
		require.NoError(t, err)
		assert.NotEmpty(t, msg)
		assert.Equal(t, "default/ConfigMap/my-config/workflow", failedLockName)
		assert.False(t, status)
		assert.True(t, wfUpdate)

		syncManager.Release(ctx, wf, "", wf.Spec.Synchronization)
		assert.Equal(t, holderKey2, nextKey)
		require.NotNil(t, wf.Status.Synchronization)
		assert.Empty(t, wf.Status.Synchronization.Semaphore.Holding[0].Holders)

		// Low priority workflow try to acquire the lock
		status, wfUpdate, msg, failedLockName, err = syncManager.TryAcquire(ctx, wf1, "", wf1.Spec.Synchronization)
		require.NoError(t, err)
		assert.NotEmpty(t, msg)
		assert.Equal(t, "default/ConfigMap/my-config/workflow", failedLockName)
		assert.False(t, status)
		assert.True(t, wfUpdate)

		// High Priority workflow acquires the lock
		status, wfUpdate, msg, failedLockName, err = syncManager.TryAcquire(ctx, wf2, "", wf2.Spec.Synchronization)
		require.NoError(t, err)
		assert.Empty(t, msg)
		assert.Empty(t, failedLockName)
		assert.True(t, status)
		assert.True(t, wfUpdate)
		require.NotNil(t, wf2.Status.Synchronization)
		require.NotNil(t, wf2.Status.Synchronization.Semaphore)
		key = getHolderKey(wf2, "")
		assert.Equal(t, key, wf2.Status.Synchronization.Semaphore.Holding[0].Holders[0])

		syncManager.ReleaseAll(ctx, wf2)
		assert.Nil(t, wf2.Status.Synchronization)

		sema := syncManager.syncLockMap["default/ConfigMap/my-config/workflow"].(*prioritySemaphore)
		require.NotNil(t, sema)
		assert.Len(t, sema.pending.items, 2)
		syncManager.ReleaseAll(ctx, wf1)
		assert.Len(t, sema.pending.items, 1)
		syncManager.ReleaseAll(ctx, wf3)
		assert.Empty(t, sema.pending.items)
	})

	t.Run("WorkflowLevelSemaphoreAcquireAndReleaseWithMultipleSemaphores", func(t *testing.T) {
		// Create ConfigMap with multiple semaphore limits
		cm := v1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "multiple-sema-config",
				Namespace: "default",
			},
			Data: map[string]string{
				"sem1": "1",
				"sem2": "1",
				"sem3": "1",
			},
		}
		_, err := kube.CoreV1().ConfigMaps("default").Create(ctx, &cm, metav1.CreateOptions{})
		require.NoError(t, err)

		syncManager := NewLockManager(ctx, kube, "", nil, syncLimitFunc, func(key string) {
			// nextKey = key
		}, WorkflowExistenceFunc)

		// Create two workflows that both need all semaphores
		wf1 := wfv1.MustUnmarshalWorkflow(wfWithSemaphore)
		wf1.Name = "wf1"
		wf1.Spec.Synchronization.Semaphores = []*wfv1.SemaphoreRef{
			{
				ConfigMapKeyRef: &v1.ConfigMapKeySelector{
					LocalObjectReference: v1.LocalObjectReference{Name: "multiple-sema-config"},
					Key:                  "sem1",
				},
			},
			{
				ConfigMapKeyRef: &v1.ConfigMapKeySelector{
					LocalObjectReference: v1.LocalObjectReference{Name: "multiple-sema-config"},
					Key:                  "sem2",
				},
			},
			{
				ConfigMapKeyRef: &v1.ConfigMapKeySelector{
					LocalObjectReference: v1.LocalObjectReference{Name: "multiple-sema-config"},
					Key:                  "sem3",
				},
			},
		}

		wf2 := wf1.DeepCopy()
		wf2.Name = "wf2"

		// First workflow should acquire all semaphores
		status, wfUpdate, msg, failedLockName, err := syncManager.TryAcquire(ctx, wf1, "", wf1.Spec.Synchronization)
		require.NoError(t, err)
		assert.Empty(t, msg)
		assert.Empty(t, failedLockName)
		assert.True(t, status)
		assert.True(t, wfUpdate)
		require.NotNil(t, wf1.Status.Synchronization)
		require.NotNil(t, wf1.Status.Synchronization.Semaphore)
		require.NotNil(t, wf1.Status.Synchronization.Semaphore.Holding)
		assert.Len(t, wf1.Status.Synchronization.Semaphore.Holding, 3)

		// Release all semaphores from first workflow
		syncManager.ReleaseAll(ctx, wf1)

		// Second workflow should now be able to acquire all semaphores
		status, wfUpdate, msg, failedLockName, err = syncManager.TryAcquire(ctx, wf2, "", wf2.Spec.Synchronization)
		require.NoError(t, err)
		assert.Empty(t, msg)
		assert.Empty(t, failedLockName)
		assert.True(t, status)
		assert.True(t, wfUpdate)
		require.NotNil(t, wf2.Status.Synchronization)
		require.NotNil(t, wf2.Status.Synchronization.Semaphore)
		require.NotNil(t, wf2.Status.Synchronization.Semaphore.Holding)
		assert.Len(t, wf2.Status.Synchronization.Semaphore.Holding, 3)

		// Clean up
		syncManager.ReleaseAll(ctx, wf2)
	})
}

func TestResizeSemaphoreSize(t *testing.T) {
	kube := fake.NewSimpleClientset()
	var cm v1.ConfigMap
	wfv1.MustUnmarshal([]byte(configMap), &cm)

	ctx := logging.TestContext(t.Context())
	_, err := kube.CoreV1().ConfigMaps("default").Create(ctx, &cm, metav1.CreateOptions{})
	require.NoError(t, err)

	syncLimitFunc := GetSyncLimitFunc(kube)
	t.Run("WfLevelAcquireAndRelease", func(t *testing.T) {
		syncManager := NewLockManager(ctx, kube, "", nil, syncLimitFunc, func(key string) {
		}, WorkflowExistenceFunc)
		wf := wfv1.MustUnmarshalWorkflow(wfWithSemaphore)
		wf.CreationTimestamp = metav1.Time{Time: time.Now()}
		wf1 := wf.DeepCopy()
		wf2 := wf.DeepCopy()
		status, wfUpdate, msg, failedLockName, err := syncManager.TryAcquire(ctx, wf, "", wf.Spec.Synchronization)
		require.NoError(t, err)
		assert.Empty(t, msg)
		assert.Empty(t, failedLockName)
		assert.True(t, status)
		assert.True(t, wfUpdate)
		require.NotNil(t, wf.Status.Synchronization)
		require.NotNil(t, wf.Status.Synchronization.Semaphore)
		key := getHolderKey(wf, "")
		assert.Equal(t, key, wf.Status.Synchronization.Semaphore.Holding[0].Holders[0])

		wf1.Name = "two"
		status, wfUpdate, msg, failedLockName, err = syncManager.TryAcquire(ctx, wf1, "", wf1.Spec.Synchronization)
		require.NoError(t, err)
		assert.NotEmpty(t, msg)
		assert.Equal(t, "default/ConfigMap/my-config/workflow", failedLockName)
		assert.False(t, status)
		assert.True(t, wfUpdate)

		wf2.Name = "three"
		status, wfUpdate, msg, failedLockName, err = syncManager.TryAcquire(ctx, wf2, "", wf2.Spec.Synchronization)
		require.NoError(t, err)
		assert.NotEmpty(t, msg)
		assert.Equal(t, "default/ConfigMap/my-config/workflow", failedLockName)
		assert.False(t, status)
		assert.True(t, wfUpdate)

		// Increase the semaphore Size
		cm, err := kube.CoreV1().ConfigMaps("default").Get(ctx, "my-config", metav1.GetOptions{})
		require.NoError(t, err)
		cm.Data["workflow"] = "3"
		_, err = kube.CoreV1().ConfigMaps("default").Update(ctx, cm, metav1.UpdateOptions{})
		require.NoError(t, err)

		status, wfUpdate, msg, failedLockName, err = syncManager.TryAcquire(ctx, wf1, "", wf1.Spec.Synchronization)
		require.NoError(t, err)
		assert.True(t, status)
		assert.Empty(t, msg)
		assert.Empty(t, failedLockName)
		assert.True(t, wfUpdate)
		require.NotNil(t, wf1.Status.Synchronization)
		require.NotNil(t, wf1.Status.Synchronization.Semaphore)
		key = getHolderKey(wf1, "")
		assert.Equal(t, key, wf1.Status.Synchronization.Semaphore.Holding[0].Holders[0])

		status, wfUpdate, msg, failedLockName, err = syncManager.TryAcquire(ctx, wf2, "", wf2.Spec.Synchronization)
		require.NoError(t, err)
		assert.Empty(t, msg)
		assert.Empty(t, failedLockName)
		assert.True(t, status)
		assert.True(t, wfUpdate)
		require.NotNil(t, wf2.Status.Synchronization)
		require.NotNil(t, wf2.Status.Synchronization.Semaphore)
		key = getHolderKey(wf2, "")
		assert.Equal(t, key, wf2.Status.Synchronization.Semaphore.Holding[0].Holders[0])
	})
}

func TestSemaphoreTmplLevel(t *testing.T) {
	kube := fake.NewSimpleClientset()
	var cm v1.ConfigMap
	wfv1.MustUnmarshal([]byte(configMap), &cm)

	ctx := logging.TestContext(t.Context())
	_, err := kube.CoreV1().ConfigMaps("default").Create(ctx, &cm, metav1.CreateOptions{})
	require.NoError(t, err)

	syncLimitFunc := GetSyncLimitFunc(kube)
	t.Run("TemplateLevelAcquireAndRelease", func(t *testing.T) {
		// var nextKey string
		syncManager := NewLockManager(ctx, kube, "", nil, syncLimitFunc, func(key string) {
			// nextKey = key
		}, WorkflowExistenceFunc)
		wf := wfv1.MustUnmarshalWorkflow(wfWithTmplSemaphore)
		tmpl := wf.Spec.Templates[2]

		status, wfUpdate, msg, failedLockName, err := syncManager.TryAcquire(ctx, wf, "semaphore-tmpl-level-xjvln-3448864205", tmpl.Synchronization)
		require.NoError(t, err)
		assert.Empty(t, msg)
		assert.Empty(t, failedLockName)
		assert.True(t, status)
		assert.True(t, wfUpdate)
		require.NotNil(t, wf.Status.Synchronization)
		require.NotNil(t, wf.Status.Synchronization.Semaphore)
		key := getHolderKey(wf, "semaphore-tmpl-level-xjvln-3448864205")
		assert.Equal(t, key, wf.Status.Synchronization.Semaphore.Holding[0].Holders[0])

		// Try to acquire again
		status, wfUpdate, msg, failedLockName, err = syncManager.TryAcquire(ctx, wf, "semaphore-tmpl-level-xjvln-3448864205", tmpl.Synchronization)
		require.NoError(t, err)
		assert.True(t, status)
		assert.Empty(t, failedLockName)
		assert.False(t, wfUpdate)
		assert.Empty(t, msg)

		status, wfUpdate, msg, failedLockName, err = syncManager.TryAcquire(ctx, wf, "semaphore-tmpl-level-xjvln-1607747183", tmpl.Synchronization)
		require.NoError(t, err)
		assert.NotEmpty(t, msg)
		assert.Equal(t, "default/ConfigMap/my-config/template", failedLockName)
		assert.True(t, wfUpdate)
		assert.False(t, status)

		syncManager.Release(ctx, wf, "semaphore-tmpl-level-xjvln-3448864205", tmpl.Synchronization)
		require.NotNil(t, wf.Status.Synchronization)
		require.NotNil(t, wf.Status.Synchronization.Semaphore)
		assert.Empty(t, wf.Status.Synchronization.Semaphore.Holding[0].Holders)

		status, wfUpdate, msg, failedLockName, err = syncManager.TryAcquire(ctx, wf, "semaphore-tmpl-level-xjvln-1607747183", tmpl.Synchronization)
		require.NoError(t, err)
		assert.Empty(t, msg)
		assert.Empty(t, failedLockName)
		assert.True(t, status)
		assert.True(t, wfUpdate)
		require.NotNil(t, wf.Status.Synchronization)
		require.NotNil(t, wf.Status.Synchronization.Semaphore)
		key = getHolderKey(wf, "semaphore-tmpl-level-xjvln-1607747183")
		assert.Equal(t, key, wf.Status.Synchronization.Semaphore.Holding[0].Holders[0])
	})
}

type mockGetSyncLimit struct {
	callCount  int
	outputSize int
	outputErr  error
}

func (m *mockGetSyncLimit) getSyncLimit(_ context.Context, s string) (int, error) {
	m.callCount++
	return m.outputSize, m.outputErr
}

func TestSemaphoreSizeCache(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	kube := fake.NewSimpleClientset()

	mockedNow := time.Now()
	nowFn = func() time.Time {
		return mockedNow
	}
	defer func() {
		nowFn = time.Now
	}()

	t.Run("WfLevelAcquireAndRelease", func(t *testing.T) {
		mock := mockGetSyncLimit{}
		mock.outputSize = 10
		config := config.SyncConfig{
			SemaphoreLimitCacheSeconds: ptr.To(int64(1)),
		}

		syncManager := NewLockManager(ctx, kube, "", &config, mock.getSyncLimit, func(key string) {
		}, WorkflowExistenceFunc)

		wf := wfv1.MustUnmarshalWorkflow(wfWithSemaphore)
		wf.CreationTimestamp = metav1.Time{Time: time.Now()}

		status, wfUpdate, msg, failedLockName, err := syncManager.TryAcquire(ctx, wf, "", wf.Spec.Synchronization)
		require.NoError(t, err)
		assert.Empty(t, msg)
		assert.Empty(t, failedLockName)
		assert.True(t, status)
		assert.True(t, wfUpdate)
		assert.Equal(t, 1, mock.callCount)

		semaphore := syncManager.syncLockMap["default/ConfigMap/my-config/workflow"]
		assert.Equal(t, 10, semaphore.getLimit(ctx))

		status, wfUpdate, msg, failedLockName, err = syncManager.TryAcquire(ctx, wf, "", wf.Spec.Synchronization)
		require.NoError(t, err)
		assert.True(t, status)
		assert.Empty(t, failedLockName)
		assert.False(t, wfUpdate)
		assert.Empty(t, msg)
		assert.Equal(t, 1, mock.callCount)

		semaphore = syncManager.syncLockMap["default/ConfigMap/my-config/workflow"]
		assert.Equal(t, 10, semaphore.getLimit(ctx))

		mockedNow = mockedNow.Add(1 * time.Second)

		status, wfUpdate, msg, failedLockName, err = syncManager.TryAcquire(ctx, wf, "", wf.Spec.Synchronization)
		require.NoError(t, err)
		assert.True(t, status)
		assert.Empty(t, failedLockName)
		assert.False(t, wfUpdate)
		assert.Empty(t, msg)
		assert.Equal(t, 2, mock.callCount)

		semaphore = syncManager.syncLockMap["default/ConfigMap/my-config/workflow"]
		assert.Equal(t, 10, semaphore.getLimit(ctx))

		// semaphore age should be updated to now
		status, wfUpdate, msg, failedLockName, err = syncManager.TryAcquire(ctx, wf, "", wf.Spec.Synchronization)
		require.NoError(t, err)
		assert.True(t, status)
		assert.Empty(t, failedLockName)
		assert.False(t, wfUpdate)
		assert.Empty(t, msg)
		assert.Equal(t, 2, mock.callCount)

		semaphore = syncManager.syncLockMap["default/ConfigMap/my-config/workflow"]
		assert.Equal(t, 10, semaphore.getLimit(ctx))

		mockedNow = mockedNow.Add(1 * time.Second)
		mock.outputSize = 20

		status, wfUpdate, msg, failedLockName, err = syncManager.TryAcquire(ctx, wf, "", wf.Spec.Synchronization)
		require.NoError(t, err)
		assert.True(t, status)
		assert.Empty(t, failedLockName)
		assert.False(t, wfUpdate)
		assert.Empty(t, msg)
		assert.Equal(t, 3, mock.callCount)

		semaphore = syncManager.syncLockMap["default/ConfigMap/my-config/workflow"]
		assert.Equal(t, 20, semaphore.getLimit(ctx))

		// semaphore age should be updated to now again
		status, wfUpdate, msg, failedLockName, err = syncManager.TryAcquire(ctx, wf, "", wf.Spec.Synchronization)
		require.NoError(t, err)
		assert.True(t, status)
		assert.Empty(t, failedLockName)
		assert.False(t, wfUpdate)
		assert.Empty(t, msg)
		assert.Equal(t, 3, mock.callCount)

		semaphore = syncManager.syncLockMap["default/ConfigMap/my-config/workflow"]
		assert.Equal(t, 20, semaphore.getLimit(ctx))
	})

	t.Run("TemplateLevelAcquireAndRelease", func(t *testing.T) {
		mock := mockGetSyncLimit{}
		mock.outputSize = 10

		config := config.SyncConfig{
			SemaphoreLimitCacheSeconds: ptr.To(int64(1)),
		}

		syncManager := NewLockManager(ctx, kube, "", &config, mock.getSyncLimit, func(key string) {
		}, WorkflowExistenceFunc)
		wf := wfv1.MustUnmarshalWorkflow(wfWithTmplSemaphore)
		tmpl := wf.Spec.Templates[2]

		status, wfUpdate, msg, failedLockName, err := syncManager.TryAcquire(ctx, wf, "semaphore-tmpl-level-xjvln-3448864205", tmpl.Synchronization)
		require.NoError(t, err)
		assert.Empty(t, msg)
		assert.Empty(t, failedLockName)
		assert.True(t, status)
		assert.True(t, wfUpdate)
		assert.Equal(t, 1, mock.callCount)

		semaphore := syncManager.syncLockMap["default/ConfigMap/my-config/template"]
		assert.Equal(t, 10, semaphore.getLimit(ctx))

		status, wfUpdate, msg, failedLockName, err = syncManager.TryAcquire(ctx, wf, "semaphore-tmpl-level-xjvln-3448864205", tmpl.Synchronization)
		require.NoError(t, err)
		assert.True(t, status)
		assert.Empty(t, failedLockName)
		assert.False(t, wfUpdate)
		assert.Empty(t, msg)
		assert.Equal(t, 1, mock.callCount)

		semaphore = syncManager.syncLockMap["default/ConfigMap/my-config/template"]
		assert.Equal(t, 10, semaphore.getLimit(ctx))

		mockedNow = mockedNow.Add(1 * time.Second)

		status, wfUpdate, msg, failedLockName, err = syncManager.TryAcquire(ctx, wf, "semaphore-tmpl-level-xjvln-3448864205", tmpl.Synchronization)
		require.NoError(t, err)
		assert.True(t, status)
		assert.Empty(t, failedLockName)
		assert.False(t, wfUpdate)
		assert.Empty(t, msg)
		assert.Equal(t, 2, mock.callCount)

		semaphore = syncManager.syncLockMap["default/ConfigMap/my-config/template"]
		assert.Equal(t, 10, semaphore.getLimit(ctx))

		// semaphore age should be updated to now
		status, wfUpdate, msg, failedLockName, err = syncManager.TryAcquire(ctx, wf, "semaphore-tmpl-level-xjvln-3448864205", tmpl.Synchronization)
		require.NoError(t, err)
		assert.True(t, status)
		assert.Empty(t, failedLockName)
		assert.False(t, wfUpdate)
		assert.Empty(t, msg)
		assert.Equal(t, 2, mock.callCount)

		semaphore = syncManager.syncLockMap["default/ConfigMap/my-config/template"]
		assert.Equal(t, 10, semaphore.getLimit(ctx))

		mockedNow = mockedNow.Add(1 * time.Second)
		mock.outputSize = 20

		status, wfUpdate, msg, failedLockName, err = syncManager.TryAcquire(ctx, wf, "semaphore-tmpl-level-xjvln-3448864205", tmpl.Synchronization)
		require.NoError(t, err)
		assert.True(t, status)
		assert.Empty(t, failedLockName)
		assert.False(t, wfUpdate)
		assert.Empty(t, msg)
		assert.Equal(t, 3, mock.callCount)

		semaphore = syncManager.syncLockMap["default/ConfigMap/my-config/template"]
		assert.Equal(t, 20, semaphore.getLimit(ctx))

		// semaphore age should be updated to now again
		status, wfUpdate, msg, failedLockName, err = syncManager.TryAcquire(ctx, wf, "semaphore-tmpl-level-xjvln-3448864205", tmpl.Synchronization)
		require.NoError(t, err)
		assert.True(t, status)
		assert.Empty(t, failedLockName)
		assert.False(t, wfUpdate)
		assert.Empty(t, msg)
		assert.Equal(t, 3, mock.callCount)

		semaphore = syncManager.syncLockMap["default/ConfigMap/my-config/template"]
		assert.Equal(t, 20, semaphore.getLimit(ctx))
	})
}

func TestTriggerWFWithAvailableLock(t *testing.T) {
	assert := assert.New(t)
	kube := fake.NewSimpleClientset()
	var cm v1.ConfigMap
	wfv1.MustUnmarshal([]byte(configMap), &cm)
	cm.Data["workflow"] = "3"

	ctx := logging.TestContext(t.Context())
	_, err := kube.CoreV1().ConfigMaps("default").Create(ctx, &cm, metav1.CreateOptions{})
	require.NoError(t, err)

	syncLimitFunc := GetSyncLimitFunc(kube)
	t.Run("TriggerWfsWithAvailableLocks", func(t *testing.T) {
		triggerCount := 0
		syncManager := NewLockManager(ctx, kube, "", nil, syncLimitFunc, func(key string) {
			triggerCount++
		}, WorkflowExistenceFunc)
		var wfs []wfv1.Workflow
		for i := 0; i < 3; i++ {
			wf := wfv1.MustUnmarshalWorkflow(wfWithSemaphore)
			wf.Name = fmt.Sprintf("%s-%d", "acquired", i)
			status, wfUpdate, msg, failedLockName, err := syncManager.TryAcquire(ctx, wf, "", wf.Spec.Synchronization)
			require.NoError(t, err)
			assert.Empty(msg)
			assert.Empty(failedLockName)
			assert.True(status)
			assert.True(wfUpdate)
			wfs = append(wfs, *wf)

		}
		for i := 0; i < 3; i++ {
			wf := wfv1.MustUnmarshalWorkflow(wfWithSemaphore)
			wf.Name = fmt.Sprintf("%s-%d", "wait", i)
			status, wfUpdate, msg, failedLockName, err := syncManager.TryAcquire(ctx, wf, "", wf.Spec.Synchronization)
			require.NoError(t, err)
			assert.NotEmpty(msg)
			assert.Equal("default/ConfigMap/my-config/workflow", failedLockName)
			assert.False(status)
			assert.True(wfUpdate)
		}
		syncManager.Release(ctx, &wfs[0], "", wfs[0].Spec.Synchronization)
		triggerCount = 0
		syncManager.Release(ctx, &wfs[1], "", wfs[1].Spec.Synchronization)
		assert.Equal(2, triggerCount)
	})
}

func TestMutexWfLevel(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	kube := fake.NewSimpleClientset()
	syncLimitFunc := GetSyncLimitFunc(kube)
	t.Run("WorkflowLevelMutexAcquireAndRelease", func(t *testing.T) {
		// var nextKey string
		syncManager := NewLockManager(ctx, kube, "", nil, syncLimitFunc, func(key string) {
			// nextKey = key
		}, WorkflowExistenceFunc)
		wf := wfv1.MustUnmarshalWorkflow(wfWithMutex)
		wf1 := wf.DeepCopy()
		wf2 := wf.DeepCopy()

		status, wfUpdate, msg, failedLockName, err := syncManager.TryAcquire(ctx, wf, "", wf.Spec.Synchronization)
		require.NoError(t, err)
		assert.Empty(t, msg)
		assert.Empty(t, failedLockName)
		assert.True(t, status)
		assert.True(t, wfUpdate)
		require.NotNil(t, wf.Status.Synchronization)
		require.NotNil(t, wf.Status.Synchronization.Mutex)
		require.NotNil(t, wf.Status.Synchronization.Mutex.Holding)

		wf1.Name = "two"
		status, wfUpdate, msg, failedLockName, err = syncManager.TryAcquire(ctx, wf1, "", wf1.Spec.Synchronization)
		require.NoError(t, err)
		assert.NotEmpty(t, msg)
		assert.Equal(t, "default/Mutex/my-mutex", failedLockName)
		assert.False(t, status)
		assert.True(t, wfUpdate)

		wf2.Name = "three"
		status, wfUpdate, msg, failedLockName, err = syncManager.TryAcquire(ctx, wf2, "", wf2.Spec.Synchronization)
		require.NoError(t, err)
		assert.NotEmpty(t, msg)
		assert.Equal(t, "default/Mutex/my-mutex", failedLockName)
		assert.False(t, status)
		assert.True(t, wfUpdate)

		mutex := syncManager.syncLockMap["default/Mutex/my-mutex"].(*prioritySemaphore)
		require.NotNil(t, mutex)
		assert.Len(t, mutex.pending.items, 2)
		syncManager.ReleaseAll(ctx, wf1)
		assert.Len(t, mutex.pending.items, 1)
		syncManager.ReleaseAll(ctx, wf2)
		assert.Empty(t, mutex.pending.items)
	})

	t.Run("WorkflowLevelMutexAcquireAndReleaseWithMultipleMutex", func(t *testing.T) {
		syncManager := NewLockManager(ctx, kube, "", nil, syncLimitFunc, func(key string) {
			// nextKey = key
		}, WorkflowExistenceFunc)
		wf := wfv1.MustUnmarshalWorkflow(wfWithMutex)
		mutexes := make([]*wfv1.Mutex, 0, 10)
		for i := range 10 {
			mutexes = append(mutexes, &wfv1.Mutex{Name: fmt.Sprintf("mutex%d", i)})
		}
		wf.Spec.Synchronization.Mutexes = mutexes
		wf1 := wf.DeepCopy()
		wf1.Name = "two"

		status, wfUpdate, msg, failedLockName, err := syncManager.TryAcquire(ctx, wf, "", wf.Spec.Synchronization)
		require.NoError(t, err)
		require.Empty(t, msg)
		require.Empty(t, failedLockName)
		require.True(t, status)
		require.True(t, wfUpdate)
		require.NotNil(t, wf.Status.Synchronization)
		require.NotNil(t, wf.Status.Synchronization.Mutex)
		require.NotNil(t, wf.Status.Synchronization.Mutex.Holding)
		syncManager.ReleaseAll(ctx, wf)

		status, wfUpdate, msg, failedLockName, err = syncManager.TryAcquire(ctx, wf1, "", wf1.Spec.Synchronization)
		require.NoError(t, err)
		require.Empty(t, msg)
		require.Empty(t, failedLockName)
		require.True(t, status)
		require.True(t, wfUpdate)
		require.NotNil(t, wf1.Status.Synchronization)
		require.NotNil(t, wf1.Status.Synchronization.Mutex)
		require.NotNil(t, wf1.Status.Synchronization.Mutex.Holding)
		syncManager.ReleaseAll(ctx, wf1)
	})
}

func TestCheckWorkflowExistence(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	kube := fake.NewSimpleClientset()
	var cm v1.ConfigMap
	wfv1.MustUnmarshal([]byte(configMap), &cm)
	cm.Data["workflow"] = "1"

	_, err := kube.CoreV1().ConfigMaps("default").Create(ctx, &cm, metav1.CreateOptions{})
	require.NoError(t, err)

	syncLimitFunc := GetSyncLimitFunc(kube)
	t.Run("WorkflowDeleted", func(t *testing.T) {
		syncManager := NewLockManager(ctx, kube, "", nil, syncLimitFunc, func(key string) {
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
		_, _, _, _, _ = syncManager.TryAcquire(ctx, wfMutex, "", wfMutex.Spec.Synchronization)
		_, _, _, _, _ = syncManager.TryAcquire(ctx, wfMutex1, "", wfMutex.Spec.Synchronization)
		_, _, _, _, _ = syncManager.TryAcquire(ctx, wfSema, "", wfSema.Spec.Synchronization)
		_, _, _, _, _ = syncManager.TryAcquire(ctx, wfSema1, "", wfSema.Spec.Synchronization)
		mutex := syncManager.syncLockMap["default/Mutex/my-mutex"].(*prioritySemaphore)
		semaphore := syncManager.syncLockMap["default/ConfigMap/my-config/workflow"]

		// Pre-state: mutex has 1 holder (hello-world) and 1 pending (test1)
		holders, err := mutex.getCurrentHolders(ctx)
		require.NoError(t, err)
		assert.Len(t, holders, 1)
		pending, err := mutex.getCurrentPending(ctx)
		require.NoError(t, err)
		assert.Len(t, pending, 1)

		// Pre-state: semaphore has 1 holder (hello-world) and 1 pending (test2)
		holders, err = semaphore.getCurrentHolders(ctx)
		require.NoError(t, err)
		assert.Len(t, holders, 1)
		pending, err = semaphore.getCurrentPending(ctx)
		require.NoError(t, err)
		assert.Len(t, pending, 1)

		syncManager.CheckWorkflowExistence(ctx)

		// Post-state: mutex holder (hello-world) removed, pending (test1) remains
		holders, err = mutex.getCurrentHolders(ctx)
		require.NoError(t, err)
		assert.Empty(t, holders)
		pending, err = mutex.getCurrentPending(ctx)
		require.NoError(t, err)
		assert.Len(t, pending, 1)

		// Post-state: semaphore holder (hello-world) and pending (test2) both removed
		holders, err = semaphore.getCurrentHolders(ctx)
		require.NoError(t, err)
		assert.Empty(t, holders)
		pending, err = semaphore.getCurrentPending(ctx)
		require.NoError(t, err)
		assert.Empty(t, pending)
	})
}

func TestTriggerWFWithSemaphoreAndMutex(t *testing.T) {
	assert := assert.New(t)
	kube := fake.NewSimpleClientset()
	var cm v1.ConfigMap
	wfv1.MustUnmarshal([]byte(configMap), &cm)
	cm.Data["test-sem"] = "1"

	ctx := logging.TestContext(t.Context())
	_, err := kube.CoreV1().ConfigMaps("default").Create(ctx, &cm, metav1.CreateOptions{})
	require.NoError(t, err)
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
      image: alpine:3.23
      name: ""
      resources:
        requests:
          cpu: 100m
          memory: 100Mi
    name: load-command
    synchronization:
      mutexes:
        - name: dag-2-task-1
  - container:
      args:
      - echo 'django command!'
      command:
      - sh
      - -c
      image: alpine:3.23
      name: ""
      resources:
        requests:
          cpu: 100m
          memory: 100Mi
    name: django-command
    synchronization:
      semaphores:
        - configMapKeyRef:
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
  synchronization:
    mutex:
      holding:
      - holder: synchronization-tmpl-level-sgg6t-928517240
        mutex: argo/Mutex/workflow
      waiting:
      - holder: argo/synchronization-tmpl-level-sgg6t/synchronization-tmpl-level-sgg6t-928517240
        mutex: argo/Mutex/workflow
  taskResultsCompletionStatus:
    synchronization-tmpl-level-sgg6t-928517240: false
`)
	syncLimitFunc := GetSyncLimitFunc(kube)

	syncManager := NewLockManager(ctx, kube, "", nil, syncLimitFunc, func(key string) {
		// nextKey = key
	}, WorkflowExistenceFunc)
	t.Run("InitializeMutex", func(t *testing.T) {
		tmpl := wf.Spec.Templates[1]
		status, wfUpdate, msg, failedLockName, err := syncManager.TryAcquire(ctx, wf, "synchronization-tmpl-level-sgg6t-1949670081", tmpl.Synchronization)
		require.NoError(t, err)
		assert.Empty(msg)
		assert.Empty(failedLockName)
		assert.True(status)
		assert.True(wfUpdate)
		require.NotNil(t, wf.Status.Synchronization)
		assert.NotNil(wf.Status.Synchronization.Mutex)
	})
	t.Run("InitializeSemaphore", func(t *testing.T) {
		tmpl := wf.Spec.Templates[2]
		status, wfUpdate, msg, failedLockName, err := syncManager.TryAcquire(ctx, wf, "synchronization-tmpl-level-sgg6t-1899337224", tmpl.Synchronization)
		require.NoError(t, err)
		assert.Empty(msg)
		assert.Empty(failedLockName)
		assert.True(status)
		assert.True(wfUpdate)
		require.NotNil(t, wf.Status.Synchronization)
		require.NotNil(t, wf.Status.Synchronization.Semaphore)
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
    mutexes:
      - name: my-mutex
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
      image: alpine:3.23
      name: ""
      resources: {}
    inputs: {}
    metadata: {}
    name: acquire-lock
    outputs: {}
    synchronization:
      mutexes:
        - name: workflow
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
	ctx := logging.TestContext(t.Context())
	assert := assert.New(t)
	require := require.New(t)
	kube := fake.NewSimpleClientset()

	syncLimitFunc := GetSyncLimitFunc(kube)

	syncMgr := NewLockManager(ctx, kube, "", nil, syncLimitFunc, func(key string) {
	}, WorkflowExistenceFunc)

	wfMutex := wfv1.MustUnmarshalWorkflow(wfWithMutex)

	t.Run("RunMigrationWorkflowLevel", func(t *testing.T) {
		syncMgr.syncLockMap = make(map[string]semaphore)
		wfMutex2 := wfv1.MustUnmarshalWorkflow(wfV2MutexMigrationWorkflowLevel)

		require.Len(wfMutex2.Status.Synchronization.Mutex.Holding, 1)
		holderKey := getHolderKey(wfMutex2, "")
		items := strings.Split(holderKey, "/")
		holdingName := items[len(items)-1]
		assert.Equal(wfMutex2.Status.Synchronization.Mutex.Holding[0].Holder, holdingName)

		syncMgr.syncLockMap = make(map[string]semaphore)
		wfs := []wfv1.Workflow{*wfMutex2.DeepCopy()}
		syncMgr.Initialize(ctx, wfs)

		syncItems, err := allSyncItems(wfMutex2.Spec.Synchronization)
		require.NoError(err)
		lockName, err := syncItems[0].lockName(wfMutex2.Namespace)
		require.NoError(err)

		sem, found := syncMgr.syncLockMap[lockName.String(ctx)]
		require.True(found)

		holders, err := sem.getCurrentHolders(ctx)
		require.NoError(err)
		require.Len(holders, 1)

		// PROVE: bug absent
		assert.Equal(holderKey, holders[0])

		// We should already have this lock since we acquired it above
		status, _, _, _, err := syncMgr.TryAcquire(ctx, wfMutex2, "", wfMutex.Spec.Synchronization)
		require.NoError(err)
		// BUG NOT PRESENT: https://github.com/argoproj/argo-workflows/issues/8684
		assert.True(status)
	})

	syncMgr = NewLockManager(ctx, kube, "", nil, syncLimitFunc, func(key string) {
	}, WorkflowExistenceFunc)

	t.Run("RunMigrationTemplateLevel", func(t *testing.T) {
		syncMgr.syncLockMap = make(map[string]semaphore)
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
		syncMgr.Initialize(ctx, wfs)

		syncItems, err := allSyncItems(wfMutex3.Spec.Templates[1].Synchronization)
		require.NoError(err)
		lockName, err := syncItems[0].lockName(wfMutex3.Namespace)
		require.NoError(err)

		sem, found := syncMgr.syncLockMap[lockName.String(ctx)]
		require.True(found)

		holders, err := sem.getCurrentHolders(ctx)
		require.NoError(err)
		require.Len(holders, 1)

		holderKey := getHolderKey(wfMutex3, foundNodeID)

		// PROVE: bug absent
		assert.Equal(holderKey, holders[0])

		status, _, _, _, err := syncMgr.TryAcquire(ctx, wfMutex3, foundNodeID, wfMutex.Spec.Synchronization)
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

func TestBackgroundNotifierClearsExpiredLocks(t *testing.T) {

	if runtime.GOOS == "windows" {
		t.Skip("Skipping test on Windows platforms")
	}

	for _, dbType := range testDBTypes {
		t.Run(string(dbType), func(t *testing.T) {
			ctx := logging.TestContext(t.Context())
			// Create database session and info
			info, deferfunc, _, err := createTestDBSession(ctx, t, dbType)
			require.NoError(t, err)
			defer deferfunc()

			// Set up two controllers, one active and one inactive
			activeController := "activeController"
			inactiveController := "inactiveController"

			// Insert controller records - one fresh, one stale
			now := time.Now()
			staleTime := now.Add(-info.Config.InactiveControllerTimeout * 2) // Double the inactive timeout

			_, err = info.Session.Collection(info.Config.ControllerTable).Insert(&syncdb.ControllerHealthRecord{
				Controller: activeController,
				Time:       now,
			})
			require.NoError(t, err)

			_, err = info.Session.Collection(info.Config.ControllerTable).Insert(&syncdb.ControllerHealthRecord{
				Controller: inactiveController,
				Time:       staleTime,
			})
			require.NoError(t, err)

			// Create a lock with an active controller and one with an inactive controller
			lockName1 := "test-lock-active"
			lockName2 := "test-lock-inactive"

			_, err = info.Session.Collection(info.Config.LockTable).Insert(&syncdb.LockRecord{
				Name:       lockName1,
				Controller: activeController,
				Time:       now,
			})
			require.NoError(t, err)

			_, err = info.Session.Collection(info.Config.LockTable).Insert(&syncdb.LockRecord{
				Name:       lockName2,
				Controller: inactiveController,
				Time:       now, // Time doesn't matter, controller is what matters
			})
			require.NoError(t, err)

			_, err = info.Session.SQL().Exec("INSERT INTO sync_limit (name, sizelimit) VALUES (?, ?)", "foo/test-semaphore", 100)
			require.NoError(t, err)
			// Initialize a semaphore so it gets added to the syncLockMap
			testsem, err := newDatabaseSemaphore(ctx, "test-semaphore", "foo/test-semaphore", func(key string) {}, info, 0)
			require.NoError(t, err)
			syncLockMap := make(map[string]semaphore)
			syncLockMap["sem/test-semaphore"] = testsem

			// Verify both lock records exist initially
			lockCount, err := info.Session.Collection(info.Config.LockTable).Count()
			require.NoError(t, err)
			assert.Equal(t, uint64(2), lockCount, "Should have two lock records initially")

			// Run the background notifier manually once
			for _, lock := range syncLockMap {
				lock.probeWaiting(ctx)
			}

			// Check that only the active controller's lock remains
			var remainingLocks []syncdb.LockRecord
			err = info.Session.SQL().Select("*").From(info.Config.LockTable).All(&remainingLocks)
			require.NoError(t, err)

			assert.Len(t, remainingLocks, 1, "Should have one lock record remaining")
			if len(remainingLocks) > 0 {
				assert.Equal(t, lockName1, remainingLocks[0].Name, "Active controller's lock should remain")
				assert.Equal(t, activeController, remainingLocks[0].Controller, "Active controller's lock should remain")
			}
		})
	}
}

func TestUnconfiguredSemaphores(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	kube := fake.NewSimpleClientset()
	t.Run("UnconfiguredConfigMapSemaphore", func(t *testing.T) {
		// Setup with a fake k8s client but no ConfigMap created
		syncLimitFunc := GetSyncLimitFunc(kube)
		syncManager := NewLockManager(ctx, kube, "", nil, syncLimitFunc, func(key string) {
		}, WorkflowExistenceFunc)

		// Create a workflow with a semaphore referencing a non-existent ConfigMap
		wf := wfv1.MustUnmarshalWorkflow(wfWithSemaphore)

		// Try to acquire the lock
		status, _, msg, failedLockName, err := syncManager.TryAcquire(ctx, wf, "", wf.Spec.Synchronization)

		// Assertions - expect an error because ConfigMap doesn't exist
		require.Error(t, err)
		assert.False(t, status)
		assert.NotEmpty(t, msg)
		assert.Equal(t, "default/ConfigMap/my-config/workflow", failedLockName)
		assert.Contains(t, err.Error(), "failed to initialize semaphore")
	})

	t.Run("UnavailableDatabaseSemaphore", func(t *testing.T) {
		// Don't use testDBTypes here, as we can test this on windows
		for _, dbType := range []sqldb.DBType{sqldb.Postgres, sqldb.MySQL} {
			t.Run(string(dbType), func(t *testing.T) {
				// Create appropriate invalid config for the database type
				var syncConfig *config.SyncConfig
				switch dbType {
				case sqldb.Postgres:
					syncConfig = &config.SyncConfig{
						DBConfig: config.DBConfig{
							PostgreSQL: &config.PostgreSQLConfig{
								DatabaseConfig: config.DatabaseConfig{
									Host:     "non-existent-host",
									Port:     5432,
									Database: "non-existent-db",
								},
								SSL: false,
							},
						},
					}
				case sqldb.MySQL:
					syncConfig = &config.SyncConfig{
						DBConfig: config.DBConfig{
							MySQL: &config.MySQLConfig{
								DatabaseConfig: config.DatabaseConfig{
									Host:     "non-existent-host",
									Port:     3306,
									Database: "non-existent-db",
								},
							},
						},
					}
				}

				syncManager := NewLockManager(ctx, kube, "", syncConfig, nil, func(key string) {
				}, WorkflowExistenceFunc)

				// Create a workflow with a database semaphore
				wf := wfv1.MustUnmarshalWorkflow(wfWithDBSemaphore)

				// Try to acquire the lock
				status, _, msg, failedLockName, err := syncManager.TryAcquire(ctx, wf, "", wf.Spec.Synchronization)

				// Assertions - expect it to fail because DB connection will fail
				require.Error(t, err)
				assert.False(t, status)
				assert.NotEmpty(t, msg)
				assert.Equal(t, "default/Database/my-database-sem", failedLockName)
				assert.Contains(t, err.Error(), "database session is not available")
			})
		}
	})

	t.Run("UnconfiguredDBSemaphore", func(t *testing.T) {
		// Setup a LockManager with no database configuration. This doesn't need to distinguish between Postgres and MySQL, neither are configured
		syncConfig := &config.SyncConfig{}

		syncManager := NewLockManager(ctx, kube, "", syncConfig, nil, func(key string) {
		}, WorkflowExistenceFunc)

		// Create a workflow with a database semaphore
		wf := wfv1.MustUnmarshalWorkflow(wfWithDBSemaphore)

		// Try to acquire the lock
		status, _, msg, failedLockName, err := syncManager.TryAcquire(ctx, wf, "", wf.Spec.Synchronization)

		// Assertions - expect it to fail because DB is not configured
		require.Error(t, err)
		assert.False(t, status)
		assert.NotEmpty(t, msg)
		assert.Equal(t, "default/Database/my-database-sem", failedLockName)
		assert.Contains(t, err.Error(), "database session is not available")
	})

	t.Run("MissingLimitSemaphore", func(t *testing.T) {
		for _, dbType := range testDBTypes {
			t.Run(string(dbType), func(t *testing.T) {
				// Setup test database using helper
				info, cleanup, syncConfig, err := createTestDBSession(ctx, t, dbType)
				require.NoError(t, err)
				defer cleanup()

				// Configure sync manager
				syncManager := createLockManager(ctx, info.Session, &syncConfig, nil, func(key string) {
				}, WorkflowExistenceFunc)
				require.NotNil(t, syncManager)
				require.NotNil(t, syncManager.dbInfo.Session)

				// Create a workflow with a database semaphore
				wf := wfv1.MustUnmarshalWorkflow(wfWithDBSemaphore)

				// Try to acquire the lock
				status, _, msg, failedLockName, err := syncManager.TryAcquire(ctx, wf, "", wf.Spec.Synchronization)

				// Assertions - expect it to fail because limit is not in the table
				require.Error(t, err)
				assert.False(t, status)
				assert.NotEmpty(t, msg)
				assert.Equal(t, "default/Database/my-database-sem", failedLockName)
				assert.Contains(t, err.Error(), "failed to initialize semaphore")
			})
		}
	})
}
