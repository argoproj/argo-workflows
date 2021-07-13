package controller

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/yaml"

	"github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
)

var httpwf = `apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: hello-world
  namespace: default
spec:
  entrypoint: http
  templates:
    - name: http
      http:
        url: https://www.google.com/

`

var taskSet = `apiVersion: argoproj.io/v1alpha1
kind: WorkflowTaskSet
metadata:
  creationTimestamp: "2021-04-23T21:49:05Z"
  generation: 1
  name: hello-world
  namespace: default
  ownerReferences:
  - apiVersion: argoproj.io/v1alpha1
    kind: Workflow
    name: hello-world
    uid: 0b451726-8ddd-4ba3-8d69-c3b5b43e93a3
  resourceVersion: "11581184"
  selfLink: /apis/argoproj.io/v1alpha1/namespaces/default/workflowtasksets/hello-world
  uid: b80385b8-8b72-4f13-af6d-f429a2cad443
spec:
  tasks:
    http-template-nxvtg-1265710817:
      http:
        url: http://www.google.com

status:
  nodes:
    hello-world:
      phase: Succeed
      outputs:
        parameters:
        - name: test
          value: "welcome"
`

func TestHTTPTemplate(t *testing.T) {
	var ts v1alpha1.WorkflowTaskSet
	err := yaml.UnmarshalStrict([]byte(taskSet), &ts)
	wf := v1alpha1.MustUnmarshalWorkflow(httpwf)
	cancel, controller := newController(wf, ts)
	defer cancel()

	assert.NoError(t, err)
	t.Run("ExecuteHTTPTemplate", func(t *testing.T) {
		ctx := context.Background()
		woc := newWorkflowOperationCtx(wf, controller)
		woc.operate(ctx)
		pods, err := controller.kubeclientset.CoreV1().Pods(woc.wf.Namespace).List(ctx, metav1.ListOptions{})
		assert.NoError(t, err)
		for _, pod := range pods.Items {
			assert.Equal(t, pod.Name, "hello-world-1340600742-agent")
		}
		// tss, err :=controller.wfclientset.ArgoprojV1alpha1().WorkflowTaskSets(wf.Namespace).List(ctx, metav1.ListOptions{})
		ts, err := controller.wfclientset.ArgoprojV1alpha1().WorkflowTaskSets(wf.Namespace).Get(ctx, "hello-world", metav1.GetOptions{})
		assert.NoError(t, err)
		assert.NotNil(t, ts)
		assert.Len(t, ts.Spec.Tasks, 1)
	})
}
