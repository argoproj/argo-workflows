package controller

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v3/workflow/common"
)

// TestStepsFailedRetries ensures a steps template will recognize exhausted retries
func TestStepsFailedRetries(t *testing.T) {
	ctx := context.Background()
	wf := wfv1.MustUnmarshalWorkflow("@testdata/steps_test/steps-failed-retries.yaml")
	woc := newWoc(*wf)
	woc.operate(ctx)
	assert.Equal(t, wfv1.WorkflowFailed, woc.wf.Status.Phase)
}

// Tests ability to reference workflow parameters from within top level spec fields (e.g. spec.volumes)
func TestArtifactResolutionWhenSkipped(t *testing.T) {
	cancel, controller := newController()
	defer cancel()
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("")

	ctx := context.Background()
	wf := wfv1.MustUnmarshalWorkflow("@testdata/steps_test/artifact-resolution-when-skipped.yaml")
	wf, err := wfcset.Create(ctx, wf, metav1.CreateOptions{})
	require.NoError(t, err)
	woc := newWorkflowOperationCtx(wf, controller)

	woc.operate(ctx)
	assert.Equal(t, wfv1.WorkflowSucceeded, woc.wf.Status.Phase)
}

func TestStepsWithParamAndGlobalParam(t *testing.T) {
	cancel, controller := newController()
	defer cancel()
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("")

	ctx := context.Background()
	wf := wfv1.MustUnmarshalWorkflow("@testdata/steps_test/steps-with-param-and-global-param.yaml")
	wf, err := wfcset.Create(ctx, wf, metav1.CreateOptions{})
	require.NoError(t, err)
	woc := newWorkflowOperationCtx(wf, controller)

	woc.operate(ctx)
	assert.Equal(t, wfv1.WorkflowRunning, woc.wf.Status.Phase)
}

func TestResourceDurationMetric(t *testing.T) {
	nodeStatus := `
      boundaryID: many-items-z26lj
      displayName: sleep(4:four)
      finishedAt: "2020-06-02T16:04:50Z"
      hostNodeName: minikube
      id: many-items-z26lj-3491220632
      name: many-items-z26lj[0].sleep(4:four)
      outputs:
        parameters:
        - name: pipeline_tid
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
            key: many-items-z26lj/many-items-z26lj-3491220632/main.log
            secretKeySecret:
              key: secretkey
              name: my-minio-cred
        exitCode: "0"
      phase: Succeeded
      resourcesDuration:
        cpu: 33
        memory: 24
      startedAt: "2020-06-02T16:04:21Z"
      templateName: sleep
      templateScope: local/many-items-z26lj
      type: Pod
`

	woc := wfOperationCtx{globalParams: make(common.Parameters)}
	var node wfv1.NodeStatus
	wfv1.MustUnmarshal([]byte(nodeStatus), &node)
	localScope, _ := woc.prepareMetricScope(&node)
	assert.Equal(t, "33", localScope["resourcesDuration.cpu"])
	assert.Equal(t, "24", localScope["resourcesDuration.memory"])
	assert.Equal(t, "0", localScope["exitCode"])
}

func TestResourceDurationMetricDefaultMetricScope(t *testing.T) {
	wf := wfv1.Workflow{Status: wfv1.WorkflowStatus{StartedAt: metav1.NewTime(time.Now())}}
	woc := wfOperationCtx{
		globalParams: make(common.Parameters),
		wf:           &wf,
	}

	localScope, realTimeScope := woc.prepareDefaultMetricScope()

	assert.Equal(t, "0", localScope["resourcesDuration.cpu"])
	assert.Equal(t, "0", localScope["resourcesDuration.memory"])
	assert.Equal(t, "0", localScope["duration"])
	assert.Equal(t, "Pending", localScope["status"])
	assert.Less(t, realTimeScope["workflow.duration"](), 1.0)
}

func TestOptionalArgumentAndParameter(t *testing.T) {
	cancel, controller := newController()
	defer cancel()
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("")

	ctx := context.Background()
	wf := wfv1.MustUnmarshalWorkflow("@testdata/steps_test/optional-argument-and-parameter.yaml")
	wf, err := wfcset.Create(ctx, wf, metav1.CreateOptions{})
	require.NoError(t, err)
	woc := newWorkflowOperationCtx(wf, controller)

	woc.operate(ctx)
	assert.Equal(t, wfv1.WorkflowRunning, woc.wf.Status.Phase)
}

func TestOptionalArgumentUseSubPathInLoop(t *testing.T) {
	cancel, controller := newController()
	defer cancel()
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("")

	ctx := context.Background()
	wf := wfv1.MustUnmarshalWorkflow("@testdata/steps_test/optional-argument-use-subpath-in-loop.yaml")
	wf, err := wfcset.Create(ctx, wf, metav1.CreateOptions{})
	require.NoError(t, err)
	woc := newWorkflowOperationCtx(wf, controller)

	woc.operate(ctx)
	assert.Equal(t, wfv1.WorkflowRunning, woc.wf.Status.Phase)
}
