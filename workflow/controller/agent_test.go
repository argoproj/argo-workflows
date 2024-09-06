package controller

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	apiv1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v3/workflow/common"
)

func TestExecuteTaskSet(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow(`apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: http-template
  namespace: default
spec:
  podSpecPatch: |
    nodeName: virtual-node
  entrypoint: main
  templates:
    - name: main
      steps:
        - - name: good
            template: http
            arguments:
              parameters: [{name: url, value: "https://raw.githubusercontent.com/argoproj/argo-workflows/4e450e250168e6b4d51a126b784e90b11a0162bc/pkg/apis/workflow/v1alpha1/generated.swagger.json"}]
        - - name: bad
            template: http
            continueOn:
              failed: true
            arguments:
              parameters: [{name: url, value: "http://openlibrary.org/people/george08/nofound.json"}]

    - name: http
      inputs:
        parameters:
          - name: url
      http:
       url: "{{inputs.parameters.url}}"

`)
	ctx := context.Background()
	var ts wfv1.WorkflowTaskSet
	wfv1.MustUnmarshal(`apiVersion: argoproj.io/v1alpha1
kind: WorkflowTaskSet
metadata:
  name: http-template-1
  namespace: default
spec:
  tasks:
    http-template-nxvtg-1265710817:
      http:
        url: http://openlibrary.org/people/george08/nofound.json
      inputs:
        parameters:
        - name: url
          value: http://openlibrary.org/people/george08/nofound.json
      name: http
status:
  nodes:
    http-template-1-3690327077:
      outputs:
        parameters:
        - name: result
          value: |
            {
              "swagger": "2.0",
              "info": {
                "title": "pkg/apis/workflow/v1alpha1/generated.proto",
                "version": "version not set"
              },
              "consumes": [
                "application/json"
              ],
              "produces": [
                "application/json"
              ],
              "paths": {},
              "definitions": {}
            }
      phase: Succeeded
    `, &ts)

	t.Run("CreateTaskSet", func(t *testing.T) {
		cancel, controller := newController(wf, ts, defaultServiceAccount)
		defer cancel()
		woc := newWorkflowOperationCtx(wf, controller)
		woc.operate(ctx)
		tslist, err := woc.controller.wfclientset.ArgoprojV1alpha1().WorkflowTaskSets("default").List(ctx, v1.ListOptions{})
		require.NoError(t, err)
		assert.NotEmpty(t, tslist.Items)
		assert.Len(t, tslist.Items, 1)
		for _, ts := range tslist.Items {
			assert.NotNil(t, ts)
			assert.Equal(t, ts.Name, wf.Name)
			assert.Equal(t, ts.Namespace, wf.Namespace)
			assert.Len(t, ts.Spec.Tasks, 1)
		}
		pods, err := woc.controller.kubeclientset.CoreV1().Pods("default").List(ctx, v1.ListOptions{})
		require.NoError(t, err)
		assert.NotEmpty(t, pods.Items)
		assert.Len(t, pods.Items, 1)
		for _, pod := range pods.Items {
			assert.NotNil(t, pod)
			assert.True(t, strings.HasSuffix(pod.Name, "-agent"))
			assert.Equal(t, "virtual-node", pod.Spec.NodeName)
		}
	})
	t.Run("CreateTaskSetWithInstanceID", func(t *testing.T) {
		cancel, controller := newController(wf, ts, defaultServiceAccount)
		defer cancel()
		controller.Config.InstanceID = "testID"
		woc := newWorkflowOperationCtx(wf, controller)
		woc.operate(ctx)
		tslist, err := woc.controller.wfclientset.ArgoprojV1alpha1().WorkflowTaskSets("default").List(ctx, v1.ListOptions{})
		require.NoError(t, err)
		assert.NotEmpty(t, tslist.Items)
		assert.Len(t, tslist.Items, 1)
		for _, ts := range tslist.Items {
			assert.NotNil(t, ts)
			assert.Equal(t, ts.Name, wf.Name)
			assert.Equal(t, ts.Namespace, wf.Namespace)
			assert.Len(t, ts.Spec.Tasks, 1)
		}
		pods, err := woc.controller.kubeclientset.CoreV1().Pods("default").List(ctx, v1.ListOptions{})
		require.NoError(t, err)
		assert.NotEmpty(t, pods.Items)
		assert.Len(t, pods.Items, 1)
		for _, pod := range pods.Items {
			assert.NotNil(t, pod)
			assert.True(t, strings.HasSuffix(pod.Name, "-agent"))
			assert.Equal(t, "testID", pod.ObjectMeta.Labels[common.LabelKeyControllerInstanceID])
			assert.Equal(t, "virtual-node", pod.Spec.NodeName)
		}
	})
}

func TestAssessAgentPodStatus(t *testing.T) {
	t.Run("Failed", func(t *testing.T) {
		pod1 := &apiv1.Pod{
			Status: apiv1.PodStatus{Phase: apiv1.PodFailed},
		}
		nodeStatus, msg := assessAgentPodStatus(pod1)
		assert.Equal(t, wfv1.NodeFailed, nodeStatus)
		assert.Equal(t, "", msg)
	})
	t.Run("Running", func(t *testing.T) {
		pod1 := &apiv1.Pod{
			Status: apiv1.PodStatus{Phase: apiv1.PodRunning},
		}

		nodeStatus, msg := assessAgentPodStatus(pod1)
		assert.Equal(t, wfv1.NodePhase(""), nodeStatus)
		assert.Equal(t, "", msg)
	})
	t.Run("Success", func(t *testing.T) {
		pod1 := &apiv1.Pod{
			Status: apiv1.PodStatus{Phase: apiv1.PodSucceeded},
		}
		nodeStatus, msg := assessAgentPodStatus(pod1)
		assert.Equal(t, wfv1.NodePhase(""), nodeStatus)
		assert.Equal(t, "", msg)
	})

}
