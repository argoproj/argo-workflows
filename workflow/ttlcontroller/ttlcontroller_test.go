package ttlcontroller

import (
	"testing"
	"time"

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
