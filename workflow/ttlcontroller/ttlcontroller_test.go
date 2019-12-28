package ttlcontroller

import (
	"testing"
	"time"

	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	fakewfclientset "github.com/argoproj/argo/pkg/client/clientset/versioned/fake"
	"github.com/argoproj/argo/test"
	"github.com/argoproj/argo/workflow/util"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/util/clock"
	"k8s.io/client-go/util/workqueue"
)

var completedWf = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  clusterName: ""
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
  phase: Running
  startedAt: 2018-08-27T20:41:38Z
`

var succeededWf = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  clusterName: ""
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
  clusterName: ""
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

func newTTLController() *Controller {
	clock := clock.NewFakeClock(time.Now())
	return &Controller{
		wfclientset: fakewfclientset.NewSimpleClientset(),
		//wfInformer:   informer,
		resyncPeriod: workflowTTLResyncPeriod,
		clock:        clock,
		workqueue:    workqueue.NewNamedDelayingQueue("workflow-ttl"),
	}
}

func TestEnqueueWF(t *testing.T) {
	var err error
	var un *unstructured.Unstructured
	var ten int32 = 10

	controller := newTTLController()

	// Veirfy we do not enqueue if not completed
	wf := test.LoadWorkflowFromBytes([]byte(completedWf))
	un, err = util.ToUnstructured(wf)
	assert.NoError(t, err)
	controller.enqueueWF(un)
	assert.Equal(t, 0, controller.workqueue.Len())

	// Veirfy we do not enqueue if workflow finished is not exceed the TTL
	wf.Spec.TTLSecondsAfterFinished = &ten
	wf.Status.FinishedAt = metav1.Time{Time: controller.clock.Now().Add(-5 * time.Second)}
	un, err = util.ToUnstructured(wf)
	assert.NoError(t, err)
	controller.enqueueWF(un)
	assert.Equal(t, 0, controller.workqueue.Len())

	// Verify we enqueue when ttl is expired
	wf.Spec.TTLSecondsAfterFinished = &ten
	wf.Status.FinishedAt = metav1.Time{Time: controller.clock.Now().Add(-11 * time.Second)}
	un, err = util.ToUnstructured(wf)
	assert.NoError(t, err)
	controller.enqueueWF(un)
	assert.Equal(t, 1, controller.workqueue.Len())
}

func TestTTLStrategySucceded(t *testing.T) {
	var err error
	var un *unstructured.Unstructured
	var ten int32 = 10

	controller := newTTLController()

	// Veirfy we do not enqueue if not completed
	wf := test.LoadWorkflowFromBytes([]byte(succeededWf))
	wf.Spec.TTLStrategy = &wfv1.TTLStrategy{SecondsAfterSuccess: &ten}
	wf.Status.FinishedAt = metav1.Time{Time: controller.clock.Now().Add(-5 * time.Second)}
	un, err = util.ToUnstructured(wf)
	assert.NoError(t, err)
	controller.enqueueWF(un)
	assert.Equal(t, 0, controller.workqueue.Len())

	wf1 := test.LoadWorkflowFromBytes([]byte(succeededWf))
	wf1.Spec.TTLStrategy = &wfv1.TTLStrategy{SecondsAfterSuccess: &ten}
	wf1.Status.FinishedAt = metav1.Time{Time: controller.clock.Now().Add(-11 * time.Second)}
	un, err = util.ToUnstructured(wf1)
	assert.NoError(t, err)
	controller.enqueueWF(un)
	assert.Equal(t, 1, controller.workqueue.Len())

}

func TestTTLStrategyFailed(t *testing.T) {
	var err error
	var un *unstructured.Unstructured
	var ten int32 = 10

	controller := newTTLController()

	// Veirfy we do not enqueue if not completed
	wf := test.LoadWorkflowFromBytes([]byte(failedWf))
	wf.Spec.TTLStrategy = &wfv1.TTLStrategy{SecondsAfterFailed: &ten}
	wf.Status.FinishedAt = metav1.Time{Time: controller.clock.Now().Add(-5 * time.Second)}
	un, err = util.ToUnstructured(wf)
	assert.NoError(t, err)
	controller.enqueueWF(un)
	assert.Equal(t, 0, controller.workqueue.Len())

	wf1 := test.LoadWorkflowFromBytes([]byte(failedWf))
	wf1.Spec.TTLStrategy = &wfv1.TTLStrategy{SecondsAfterFailed: &ten}
	wf1.Status.FinishedAt = metav1.Time{Time: controller.clock.Now().Add(-11 * time.Second)}
	un, err = util.ToUnstructured(wf1)
	assert.NoError(t, err)
	controller.enqueueWF(un)
	assert.Equal(t, 1, controller.workqueue.Len())

}

func TestNoTTLStrategyFailed(t *testing.T) {
	var err error
	var un *unstructured.Unstructured
	controller := newTTLController()
	// Veirfy we do not enqueue if not completed
	wf := test.LoadWorkflowFromBytes([]byte(failedWf))
	wf.Status.FinishedAt = metav1.Time{Time: controller.clock.Now().Add(-5 * time.Second)}
	un, err = util.ToUnstructured(wf)
	assert.NoError(t, err)
	controller.enqueueWF(un)
	assert.Equal(t, 0, controller.workqueue.Len())

	wf1 := test.LoadWorkflowFromBytes([]byte(failedWf))
	wf1.Status.FinishedAt = metav1.Time{Time: controller.clock.Now().Add(-11 * time.Second)}
	un, err = util.ToUnstructured(wf1)
	assert.NoError(t, err)
	controller.enqueueWF(un)
	assert.Equal(t, 0, controller.workqueue.Len())

}

func TestNoTTLStrategyFailedButTTLSecondsAfterFinished(t *testing.T) {
	var err error
	var un *unstructured.Unstructured
	var ten int32 = 10

	controller := newTTLController()

	// Veirfy we do not enqueue if not completed
	wf := test.LoadWorkflowFromBytes([]byte(failedWf))
	wf.Spec.TTLSecondsAfterFinished = &ten
	wf.Status.FinishedAt = metav1.Time{Time: controller.clock.Now().Add(-5 * time.Second)}
	un, err = util.ToUnstructured(wf)
	assert.NoError(t, err)
	controller.enqueueWF(un)
	assert.Equal(t, 0, controller.workqueue.Len())

	wf1 := test.LoadWorkflowFromBytes([]byte(failedWf))
	wf1.Spec.TTLSecondsAfterFinished = &ten
	ttlstrategy := wfv1.TTLStrategy{SecondsAfterFailed: &ten}
	wf1.Spec.TTLStrategy = &ttlstrategy
	wf1.Status.FinishedAt = metav1.Time{Time: controller.clock.Now().Add(-11 * time.Second)}
	un, err = util.ToUnstructured(wf1)
	assert.NoError(t, err)
	controller.enqueueWF(un)
	assert.Equal(t, 1, controller.workqueue.Len())
}

func TestTTLStrategyFromUnstructured(t *testing.T) {
	var err error
	var un *unstructured.Unstructured
	var ten int32 = 10
	var five int32 = 5
	controller := newTTLController()
	wf := test.LoadWorkflowFromBytes([]byte(failedWf))
	wf.Spec.TTLSecondsAfterFinished = &ten
	ttlstrategy := wfv1.TTLStrategy{SecondsAfterFailed: &five}
	wf.Spec.TTLStrategy = &ttlstrategy
	wf.Status.FinishedAt = metav1.Time{Time: controller.clock.Now().Add(-6 * time.Second)}
	un, err = util.ToUnstructured(wf)
	assert.NoError(t, err)
	controller.enqueueWF(un)
	assert.Equal(t, 1, controller.workqueue.Len())

	controller1 := newTTLController()
	wf1 := test.LoadWorkflowFromBytes([]byte(failedWf))
	wf1.Spec.TTLSecondsAfterFinished = &ten
	ttlstrategy1 := wfv1.TTLStrategy{SecondsAfterCompleted: &five}
	wf1.Spec.TTLStrategy = &ttlstrategy1
	wf1.Status.FinishedAt = metav1.Time{Time: controller1.clock.Now().Add(-6 * time.Second)}
	un, err = util.ToUnstructured(wf1)
	assert.NoError(t, err)
	controller1.enqueueWF(un)
	assert.Equal(t, 1, controller1.workqueue.Len())

	controller2 := newTTLController()
	wf2 := test.LoadWorkflowFromBytes([]byte(failedWf))
	wf2.Spec.TTLSecondsAfterFinished = &ten
	ttlstrategy2 := wfv1.TTLStrategy{SecondsAfterFailed: &five}
	wf2.Spec.TTLStrategy = &ttlstrategy2
	wf2.Status.FinishedAt = metav1.Time{Time: controller2.clock.Now().Add(-6 * time.Second)}
	un, err = util.ToUnstructured(wf2)
	assert.NoError(t, err)
	controller2.enqueueWF(un)
	assert.Equal(t, 1, controller2.workqueue.Len())

	controller3 := newTTLController()
	wf3 := test.LoadWorkflowFromBytes([]byte(failedWf))
	ttlstrategy3 := wfv1.TTLStrategy{SecondsAfterSuccess: &five}
	wf3.Spec.TTLStrategy = &ttlstrategy3
	wf3.Status.FinishedAt = metav1.Time{Time: controller3.clock.Now().Add(-6 * time.Second)}
	un, err = util.ToUnstructured(wf3)
	t.Log(wf3.Spec.TTLStrategy)
	assert.NoError(t, err)
	controller.enqueueWF(un)
	assert.Equal(t, 0, controller3.workqueue.Len())
}

func TestTTLlExpired(t *testing.T) {
	controller := newTTLController()
	var ten int32 = 10

	wf := test.LoadWorkflowFromBytes([]byte(failedWf))
	wf.Spec.TTLStrategy = &wfv1.TTLStrategy{SecondsAfterFailed: &ten}
	wf.Status.FinishedAt = metav1.Time{Time: controller.clock.Now().Add(-11 * time.Second)}
	assert.Equal(t, true, wf.Status.Failed())
	now := controller.clock.Now()
	assert.Equal(t, true, now.After(wf.Status.FinishedAt.Add(time.Second*time.Duration(*wf.Spec.TTLStrategy.SecondsAfterFailed))))
	assert.Equal(t, true, wf.Status.Failed() && wf.Spec.TTLStrategy.SecondsAfterFailed != nil)
	assert.Equal(t, true, controller.ttlExpired(wf))

	wf1 := test.LoadWorkflowFromBytes([]byte(failedWf))
	wf1.Spec.TTLStrategy = &wfv1.TTLStrategy{SecondsAfterFailed: &ten}
	wf1.Status.FinishedAt = metav1.Time{Time: controller.clock.Now().Add(-5 * time.Second)}
	assert.Equal(t, false, controller.ttlExpired(wf1))

	wf2 := test.LoadWorkflowFromBytes([]byte(failedWf))
	wf2.Spec.TTLStrategy = &wfv1.TTLStrategy{SecondsAfterFailed: &ten}
	wf2.Status.FinishedAt = metav1.Time{Time: controller.clock.Now().Add(-11 * time.Second)}
	assert.Equal(t, true, controller.ttlExpired(wf2))

	wf3 := test.LoadWorkflowFromBytes([]byte(failedWf))
	wf3.Spec.TTLStrategy = &wfv1.TTLStrategy{SecondsAfterCompleted: &ten}
	wf3.Status.FinishedAt = metav1.Time{Time: controller.clock.Now().Add(-5 * time.Second)}
	assert.Equal(t, false, controller.ttlExpired(wf3))

	wf4 := test.LoadWorkflowFromBytes([]byte(failedWf))
	wf4.Spec.TTLStrategy = &wfv1.TTLStrategy{SecondsAfterCompleted: &ten}
	wf4.Status.FinishedAt = metav1.Time{Time: controller.clock.Now().Add(-11 * time.Second)}
	assert.Equal(t, true, controller.ttlExpired(wf4))

	wf5 := test.LoadWorkflowFromBytes([]byte(succeededWf))
	wf5.Spec.TTLStrategy = &wfv1.TTLStrategy{SecondsAfterSuccess: &ten}
	wf5.Status.FinishedAt = metav1.Time{Time: controller.clock.Now().Add(-5 * time.Second)}
	assert.Equal(t, false, controller.ttlExpired(wf5))

	wf6 := test.LoadWorkflowFromBytes([]byte(succeededWf))
	wf6.Spec.TTLStrategy = &wfv1.TTLStrategy{SecondsAfterSuccess: &ten}
	wf6.Status.FinishedAt = metav1.Time{Time: controller.clock.Now().Add(-11 * time.Second)}
	assert.Equal(t, true, controller.ttlExpired(wf6))
}
