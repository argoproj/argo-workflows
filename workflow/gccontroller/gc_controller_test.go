package gccontroller

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	testingclock "k8s.io/utils/clock/testing"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	fakewfclientset "github.com/argoproj/argo-workflows/v3/pkg/client/clientset/versioned/fake"
	"github.com/argoproj/argo-workflows/v3/workflow/metrics"
	"github.com/argoproj/argo-workflows/v3/workflow/util"
)

var completedWf = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  creationTimestamp: 2018-08-27T20:41:38Z
  generateName: hello-world-
  generation: 1
  labels:
    workflows.argoproj.io/completed: "true"
    workflows.argoproj.io/phase: Succeeded
  name: hello-world-nrgbf
  namespace: default
  resourceVersion: "1063703"
  selfLink: /apis/argoproj.io/v1alpha1/namespaces/default/workflows/hello-world-nrgbf
  uid: 9866f345-aa39-11e8-b103-025000000001
spec:
  entrypoint: whalesay
  templates:
  - container:
      args:
      - hello world
      command:
      - cowsay
      image: docker/whalesay:latest
    name: whalesay
status:
  phase: Succeeded
  startedAt: 2018-08-27T20:41:38Z
  finishedAt: 2018-08-27T20:41:38Z
`

var succeededWf = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  creationTimestamp: 2018-08-27T20:41:38Z
  generateName: hello-world-
  generation: 1
  labels:
    workflows.argoproj.io/completed: "true"
    workflows.argoproj.io/phase: Succeeded
  name: hello-world-nrgbf
  namespace: default
  resourceVersion: "1063703"
  selfLink: /apis/argoproj.io/v1alpha1/namespaces/default/workflows/hello-world-nrgbf
  uid: 9866f345-aa39-11e8-b103-025000000001
spec:
  entrypoint: whalesay
  templates:
  - container:
      args:
      - hello world
      command:
      - cowsay
      image: docker/whalesay:latest
    name: whalesay
status:
  phase: Succeeded
  startedAt: 2018-08-27T20:41:38Z
`

var failedWf = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  creationTimestamp: 2018-08-27T20:41:38Z
  generateName: hello-world-
  generation: 1
  labels:
    workflows.argoproj.io/completed: "true"
    workflows.argoproj.io/phase: Succeeded
  name: hello-world-nrgbf
  namespace: default
  resourceVersion: "1063703"
  selfLink: /apis/argoproj.io/v1alpha1/namespaces/default/workflows/hello-world-nrgbf
  uid: 9866f345-aa39-11e8-b103-025000000001
spec:
  entrypoint: whalesay
  templates:
  - container:
      args:
      - hello world
      command:
      - cowsay
      image: docker/whalesay:latest
    name: whalesay
status:
  phase: Failed
  startedAt: 2018-08-27T20:41:38Z
`

var wftRefWithTTLinWFT = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  creationTimestamp: "2020-06-16T00:57:45Z"
  generateName: workflow-template-hello-world-
  generation: 6
  labels:
    workflows.argoproj.io/completed: "true"
    workflows.argoproj.io/phase: Succeeded
  name: workflow-template-hello-world-k4d26
  namespace: default
  resourceVersion: "564446"
  selfLink: /apis/argoproj.io/v1alpha1/namespaces/argo/workflows/workflow-template-hello-world-k4d26
  uid: e25cce2d-c71d-4f4e-b016-a0a2e10bf4d1
spec:
  arguments:
    parameters:
    - name: message
      value: hello world
  entrypoint: start
  templates: null
  workflowTemplateRef:
    name: workflow-template-submittable-2.9
status:
  conditions:
  - status: "True"
    type: Completed
  finishedAt: "2020-06-16T00:57:51Z"
  nodes:
    workflow-template-hello-world-k4d26:
      displayName: workflow-template-hello-world-k4d26
      finishedAt: "2020-06-16T00:57:49Z"
      hostNodeName: docker-desktop
      id: workflow-template-hello-world-k4d26
      inputs:
        parameters:
        - name: message
          value: hello world
      name: workflow-template-hello-world-k4d26
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
            key: workflow-template-hello-world-k4d26/workflow-template-hello-world-k4d26/main.log
            secretKeySecret:
              key: secretkey
              name: my-minio-cred
        exitCode: "0"
      phase: Succeeded
      resourcesDuration:
        cpu: 3
        memory: 1
      startedAt: "2020-06-16T00:57:45Z"
      templateRef:
        name: workflow-template-submittable-2.9
        template: start
      templateScope: local/workflow-template-hello-world-k4d26
      type: Pod
  phase: Succeeded
  resourcesDuration:
    cpu: 3
    memory: 1
  startedAt: "2020-06-16T00:57:45Z"
  storedTemplates:
    namespaced/workflow-template-submittable-2.9/start:
      
      container:
        args:
        - '{{inputs.parameters.message}}'
        command:
        - echo
        image: docker/whalesay:latest
        name: ""
        resources: {}
      inputs:
        parameters:
        - name: message
      metadata: {}
      name: start
      outputs: {}
  storedWorkflowTemplateSpec:
    arguments:
      parameters:
      - name: message
        value: hello world
    entrypoint: start
    templates:
    - 
      container:
        args:
        - '{{inputs.parameters.message}}'
        command:
        - echo
        image: docker/whalesay:latest
        name: ""
        resources: {}
      inputs:
        parameters:
        - name: message
      metadata: {}
      name: start
      outputs: {}
    ttlStrategy:
      secondsAfterCompletion: 10
`

var wftRefWithTTLinWF = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  creationTimestamp: "2020-06-16T01:00:14Z"
  generateName: workflow-template-hello-world-
  generation: 6
  labels:
    workflows.argoproj.io/completed: "true"
    workflows.argoproj.io/phase: Succeeded
  name: workflow-template-hello-world-jdkdw
  namespace: default
  resourceVersion: "564728"
  selfLink: /apis/argoproj.io/v1alpha1/namespaces/argo/workflows/workflow-template-hello-world-jdkdw
  uid: 57dac176-2e10-4f1d-b77c-db321d187d83
spec:
  arguments:
    parameters:
    - name: message
      value: hello world
  entrypoint: start
  templates: null
  ttlStrategy:
    secondsAfterCompletion: 10
  workflowTemplateRef:
    name: workflow-template-submittable-2.9
status:
  conditions:
  - status: "True"
    type: Completed
  finishedAt: "2020-06-16T01:00:19Z"
  nodes:
    workflow-template-hello-world-jdkdw:
      displayName: workflow-template-hello-world-jdkdw
      finishedAt: "2020-06-16T01:00:17Z"
      hostNodeName: docker-desktop
      id: workflow-template-hello-world-jdkdw
      inputs:
        parameters:
        - name: message
          value: hello world
      name: workflow-template-hello-world-jdkdw
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
            key: workflow-template-hello-world-jdkdw/workflow-template-hello-world-jdkdw/main.log
            secretKeySecret:
              key: secretkey
              name: my-minio-cred
        exitCode: "0"
      phase: Succeeded
      resourcesDuration:
        cpu: 2
        memory: 1
      startedAt: "2020-06-16T01:00:14Z"
      templateRef:
        name: workflow-template-submittable-2.9
        template: start
      templateScope: local/workflow-template-hello-world-jdkdw
      type: Pod
  phase: Succeeded
  resourcesDuration:
    cpu: 2
    memory: 1
  startedAt: "2020-06-16T01:00:14Z"
  storedTemplates:
    namespaced/workflow-template-submittable-2.9/start:
      
      container:
        args:
        - '{{inputs.parameters.message}}'
        command:
        - echo
        image: docker/whalesay:latest
        name: ""
        resources: {}
      inputs:
        parameters:
        - name: message
      metadata: {}
      name: start
      outputs: {}
  storedWorkflowTemplateSpec:
    arguments:
      parameters:
      - name: message
        value: hello world
    entrypoint: start
    templates:
    - 
      container:
        args:
        - '{{inputs.parameters.message}}'
        command:
        - echo
        image: docker/whalesay:latest
        name: ""
        resources: {}
      inputs:
        parameters:
        - name: message
      metadata: {}
      name: start
      outputs: {}
    ttlStrategy:
      secondsAfterCompletion: 60
`

func newTTLController() *Controller {
	clock := testingclock.NewFakeClock(time.Now())
	wfclientset := fakewfclientset.NewSimpleClientset()
	wfInformer := cache.NewSharedIndexInformer(nil, nil, 0, nil)
	return &Controller{
		wfclientset: wfclientset,
		wfInformer:  wfInformer,
		clock:       clock,
		workqueue:   workqueue.NewDelayingQueue(),
		metrics:     metrics.New(metrics.ServerConfig{}, metrics.ServerConfig{}),
	}
}

func enqueueWF(controller *Controller, un *unstructured.Unstructured) {
	controller.enqueueWF(un)
	time.Sleep(100*time.Millisecond + time.Second)
}

func TestEnqueueWF(t *testing.T) {
	var err error
	var un *unstructured.Unstructured

	controller := newTTLController()

	// Veirfy we do not enqueue if not completed
	wf := wfv1.MustUnmarshalWorkflow([]byte(completedWf))
	un, err = util.ToUnstructured(wf)
	assert.NoError(t, err)
	enqueueWF(controller, un)
	assert.Equal(t, 0, controller.workqueue.Len())
}

func TestTTLStrategySucceeded(t *testing.T) {
	var err error
	var un *unstructured.Unstructured
	var ten int32 = 10

	controller := newTTLController()

	// Veirfy we do not enqueue if not completed
	wf := wfv1.MustUnmarshalWorkflow([]byte(succeededWf))
	wf.Spec.TTLStrategy = &wfv1.TTLStrategy{SecondsAfterSuccess: &ten}
	wf.Status.FinishedAt = metav1.Time{Time: controller.clock.Now().Add(-5 * time.Second)}
	un, err = util.ToUnstructured(wf)
	assert.NoError(t, err)
	enqueueWF(controller, un)
	assert.Equal(t, 0, controller.workqueue.Len())

	wf1 := wfv1.MustUnmarshalWorkflow([]byte(succeededWf))
	wf1.Spec.TTLStrategy = &wfv1.TTLStrategy{SecondsAfterSuccess: &ten}
	wf1.Status.FinishedAt = metav1.Time{Time: controller.clock.Now().Add(-11 * time.Second)}
	un, err = util.ToUnstructured(wf1)
	assert.NoError(t, err)
	enqueueWF(controller, un)
	assert.Equal(t, 1, controller.workqueue.Len())

	wf2 := wfv1.MustUnmarshalWorkflow([]byte(wftRefWithTTLinWFT))
	wf2.Status.FinishedAt = metav1.Time{Time: controller.clock.Now().Add(-11 * time.Second)}
	un, err = util.ToUnstructured(wf2)
	assert.NoError(t, err)

	ctx := context.Background()
	_, err = controller.wfclientset.ArgoprojV1alpha1().Workflows("default").Create(ctx, wf2, metav1.CreateOptions{})
	assert.NoError(t, err)
	enqueueWF(controller, un)
	controller.processNextWorkItem(ctx)
	assert.Equal(t, 1, controller.workqueue.Len())

	wf3 := wfv1.MustUnmarshalWorkflow([]byte(wftRefWithTTLinWF))
	wf3.Status.FinishedAt = metav1.Time{Time: controller.clock.Now().Add(-11 * time.Second)}
	un, err = util.ToUnstructured(wf3)
	assert.NoError(t, err)

	_, err = controller.wfclientset.ArgoprojV1alpha1().Workflows("default").Create(ctx, wf3, metav1.CreateOptions{})
	assert.NoError(t, err)
	enqueueWF(controller, un)
	controller.processNextWorkItem(ctx)
	assert.Equal(t, 1, controller.workqueue.Len())
}

func TestTTLStrategyFailed(t *testing.T) {
	var err error
	var un *unstructured.Unstructured
	var ten int32 = 10

	controller := newTTLController()

	// Veirfy we do not enqueue if not completed
	wf := wfv1.MustUnmarshalWorkflow([]byte(failedWf))
	wf.Spec.TTLStrategy = &wfv1.TTLStrategy{SecondsAfterFailure: &ten}
	wf.Status.FinishedAt = metav1.Time{Time: controller.clock.Now().Add(-5 * time.Second)}
	un, err = util.ToUnstructured(wf)
	assert.NoError(t, err)
	enqueueWF(controller, un)
	assert.Equal(t, 0, controller.workqueue.Len())

	wf1 := wfv1.MustUnmarshalWorkflow([]byte(failedWf))
	wf1.Spec.TTLStrategy = &wfv1.TTLStrategy{SecondsAfterFailure: &ten}
	wf1.Status.FinishedAt = metav1.Time{Time: controller.clock.Now().Add(-11 * time.Second)}
	un, err = util.ToUnstructured(wf1)
	assert.NoError(t, err)
	enqueueWF(controller, un)
	assert.Equal(t, 1, controller.workqueue.Len())
}

func TestNoTTLStrategyFailed(t *testing.T) {
	var err error
	var un *unstructured.Unstructured
	controller := newTTLController()
	// Veirfy we do not enqueue if not completed
	wf := wfv1.MustUnmarshalWorkflow([]byte(failedWf))
	wf.Status.FinishedAt = metav1.Time{Time: controller.clock.Now().Add(-5 * time.Second)}
	un, err = util.ToUnstructured(wf)
	assert.NoError(t, err)
	enqueueWF(controller, un)
	assert.Equal(t, 0, controller.workqueue.Len())

	wf1 := wfv1.MustUnmarshalWorkflow([]byte(failedWf))
	wf1.Status.FinishedAt = metav1.Time{Time: controller.clock.Now().Add(-11 * time.Second)}
	un, err = util.ToUnstructured(wf1)
	assert.NoError(t, err)
	enqueueWF(controller, un)
	assert.Equal(t, 0, controller.workqueue.Len())
}

func TestTTLStrategyFromUnstructured(t *testing.T) {
	var err error
	var un *unstructured.Unstructured
	var five int32 = 5

	controller3 := newTTLController()
	wf3 := wfv1.MustUnmarshalWorkflow([]byte(failedWf))
	ttlstrategy3 := wfv1.TTLStrategy{SecondsAfterSuccess: &five}
	wf3.Spec.TTLStrategy = &ttlstrategy3
	wf3.Status.FinishedAt = metav1.Time{Time: controller3.clock.Now().Add(-6 * time.Second)}
	un, err = util.ToUnstructured(wf3)
	t.Log(wf3.Spec.TTLStrategy)
	assert.NoError(t, err)
	enqueueWF(controller3, un)
	assert.Equal(t, 0, controller3.workqueue.Len())
}

func TestTTLlExpired(t *testing.T) {
	controller := newTTLController()
	var ten int32 = 10

	wf := wfv1.MustUnmarshalWorkflow([]byte(failedWf))
	wf.Spec.TTLStrategy = &wfv1.TTLStrategy{SecondsAfterFailure: &ten}
	wf.Status.FinishedAt = metav1.Time{Time: controller.clock.Now().Add(-11 * time.Second)}
	assert.True(t, wf.Status.Failed())
	now := controller.clock.Now()
	assert.True(t, now.After(wf.Status.FinishedAt.Add(time.Second*time.Duration(*wf.Spec.TTLStrategy.SecondsAfterFailure))))
	assert.True(t, wf.Status.Failed() && wf.Spec.TTLStrategy.SecondsAfterFailure != nil)
	expiresIn, ok := controller.expiresIn(wf)
	assert.True(t, ok)
	assert.LessOrEqual(t, int(expiresIn), 0)

	wf1 := wfv1.MustUnmarshalWorkflow([]byte(failedWf))
	wf1.Spec.TTLStrategy = &wfv1.TTLStrategy{SecondsAfterFailure: &ten}
	wf1.Status.FinishedAt = metav1.Time{Time: controller.clock.Now().Add(-5 * time.Second)}
	expiresIn, ok = controller.expiresIn(wf1)
	assert.True(t, ok)
	assert.GreaterOrEqual(t, int(expiresIn), 0)

	wf2 := wfv1.MustUnmarshalWorkflow([]byte(failedWf))
	wf2.Spec.TTLStrategy = &wfv1.TTLStrategy{SecondsAfterFailure: &ten}
	wf2.Status.FinishedAt = metav1.Time{Time: controller.clock.Now().Add(-11 * time.Second)}
	expiresIn, ok = controller.expiresIn(wf2)
	assert.True(t, ok)
	assert.LessOrEqual(t, int(expiresIn), 0)

	wf3 := wfv1.MustUnmarshalWorkflow([]byte(failedWf))
	wf3.Spec.TTLStrategy = &wfv1.TTLStrategy{SecondsAfterCompletion: &ten}
	wf3.Status.FinishedAt = metav1.Time{Time: controller.clock.Now().Add(-5 * time.Second)}
	expiresIn, ok = controller.expiresIn(wf3)
	assert.True(t, ok)
	assert.GreaterOrEqual(t, int(expiresIn), 0)

	wf4 := wfv1.MustUnmarshalWorkflow([]byte(failedWf))
	wf4.Spec.TTLStrategy = &wfv1.TTLStrategy{SecondsAfterCompletion: &ten}
	wf4.Status.FinishedAt = metav1.Time{Time: controller.clock.Now().Add(-11 * time.Second)}
	expiresIn, ok = controller.expiresIn(wf4)
	assert.True(t, ok)
	assert.LessOrEqual(t, int(expiresIn), 0)

	wf5 := wfv1.MustUnmarshalWorkflow([]byte(succeededWf))
	wf5.Spec.TTLStrategy = &wfv1.TTLStrategy{SecondsAfterSuccess: &ten}
	wf5.Status.FinishedAt = metav1.Time{Time: controller.clock.Now().Add(-5 * time.Second)}
	expiresIn, ok = controller.expiresIn(wf5)
	assert.True(t, ok)
	assert.GreaterOrEqual(t, int(expiresIn), 0)

	wf6 := wfv1.MustUnmarshalWorkflow([]byte(succeededWf))
	wf6.Spec.TTLStrategy = &wfv1.TTLStrategy{SecondsAfterSuccess: &ten}
	wf6.Status.FinishedAt = metav1.Time{Time: controller.clock.Now().Add(-11 * time.Second)}
	expiresIn, ok = controller.expiresIn(wf6)
	assert.True(t, ok)
	assert.LessOrEqual(t, int(expiresIn), 0)
}

func TestGetTTLStrategy(t *testing.T) {
	var ten int32 = 10

	t.Run("TTLFromWorkflow", func(t *testing.T) {
		wf := wfv1.MustUnmarshalWorkflow([]byte(succeededWf))
		wf.Spec.TTLStrategy = &wfv1.TTLStrategy{
			SecondsAfterCompletion: &ten,
		}
		ttl := wf.GetTTLStrategy()
		assert.NotNil(t, ttl)
		assert.Equal(t, ten, *ttl.SecondsAfterCompletion)
	})

	t.Run("TTLInWfwithWorkflowTemplate", func(t *testing.T) {
		wf1 := wfv1.MustUnmarshalWorkflow([]byte(wftRefWithTTLinWF))
		ttl := wf1.GetTTLStrategy()
		assert.NotNil(t, ttl)
		assert.Equal(t, ten, *ttl.SecondsAfterCompletion)

		wf1.Spec.TTLStrategy = nil
		wf1.Status.StoredWorkflowSpec.TTLStrategy = nil
		ttl = wf1.GetTTLStrategy()
		assert.Nil(t, ttl)
	})
}
