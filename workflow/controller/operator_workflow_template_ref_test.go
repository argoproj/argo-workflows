package controller

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/util"
	"github.com/argoproj/argo-workflows/v4/util/logging"
)

func TestWorkflowTemplateRef(t *testing.T) {
	cancel, controller := newController(logging.TestContext(t.Context()), wfv1.MustUnmarshalWorkflow(wfWithTmplRef), wfv1.MustUnmarshalWorkflowTemplate(wfTmpl))
	defer cancel()

	ctx := logging.TestContext(t.Context())
	woc := newWorkflowOperationCtx(ctx, wfv1.MustUnmarshalWorkflow(wfWithTmplRef), controller)
	woc.operate(ctx)
	assert.Equal(t, wfv1.MustUnmarshalWorkflowTemplate(wfTmpl).Spec.Templates, woc.execWf.Spec.Templates)
	assert.Equal(t, woc.wf.Spec.Entrypoint, woc.execWf.Spec.Entrypoint)
	// verify we copy these values
	assert.Len(t, woc.volumes, 1, "volumes from workflow template")
	// and these
	assert.Equal(t, "my-sa", woc.globalParams()["workflow.serviceAccountName"])
	assert.Equal(t, "77", woc.globalParams()["workflow.priority"])
}

func TestWorkflowTemplateRefWithArgs(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow(wfWithTmplRef)
	wftmpl := wfv1.MustUnmarshalWorkflowTemplate(wfTmpl)

	ctx := logging.TestContext(t.Context())
	t.Run("CheckArgumentPassing", func(t *testing.T) {
		args := []wfv1.Parameter{
			{
				Name:  "param1",
				Value: wfv1.AnyStringPtr("test"),
			},
		}
		wf.Spec.Arguments.Parameters = util.MergeParameters(wf.Spec.Arguments.Parameters, args)
		cancel, controller := newController(ctx, wf, wftmpl)
		defer cancel()
		woc := newWorkflowOperationCtx(ctx, wf, controller)
		woc.operate(ctx)
		assert.Equal(t, "test", woc.globalParams()["workflow.parameters.param1"])
	})
}

func TestWorkflowTemplateRefWithWorkflowTemplateArgs(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow(wfWithTmplRef)
	wftmpl := wfv1.MustUnmarshalWorkflowTemplate(wfTmpl)

	ctx := logging.TestContext(t.Context())
	t.Run("CheckArgumentFromWFT", func(t *testing.T) {
		args := []wfv1.Parameter{
			{
				Name:  "param1",
				Value: wfv1.AnyStringPtr("test"),
			},
		}
		wftmpl.Spec.Arguments.Parameters = util.MergeParameters(wf.Spec.Arguments.Parameters, args)
		cancel, controller := newController(ctx, wf, wftmpl)
		defer cancel()
		woc := newWorkflowOperationCtx(ctx, wf, controller)
		woc.operate(ctx)
		assert.Equal(t, "test", woc.globalParams()["workflow.parameters.param1"])
	})

	t.Run("CheckMergingWFDefaults", func(t *testing.T) {
		wfDefaultActiveS := int64(5)
		cancel, controller := newController(ctx, wf, wftmpl)
		defer cancel()
		controller.Config.WorkflowDefaults = &wfv1.Workflow{
			Spec: wfv1.WorkflowSpec{
				ActiveDeadlineSeconds: &wfDefaultActiveS,
			},
		}
		woc := newWorkflowOperationCtx(ctx, wf, controller)
		woc.operate(ctx)
		assert.Equal(t, wfDefaultActiveS, *woc.execWf.Spec.ActiveDeadlineSeconds)
	})
	t.Run("CheckMergingWFTandWF", func(t *testing.T) {
		wfActiveS := int64(10)
		wftActiveS := int64(10)
		wfDefaultActiveS := int64(5)

		wftmpl.Spec.ActiveDeadlineSeconds = &wftActiveS
		cancel, controller := newController(ctx, wf, wftmpl)
		defer cancel()
		controller.Config.WorkflowDefaults = &wfv1.Workflow{
			Spec: wfv1.WorkflowSpec{
				ActiveDeadlineSeconds: &wfDefaultActiveS,
			},
		}
		wf.Spec.ActiveDeadlineSeconds = &wfActiveS
		woc := newWorkflowOperationCtx(ctx, wf, controller)
		woc.operate(ctx)
		assert.Equal(t, wfActiveS, *woc.execWf.Spec.ActiveDeadlineSeconds)

		wf.Spec.ActiveDeadlineSeconds = nil
		woc = newWorkflowOperationCtx(ctx, wf, controller)
		woc.operate(ctx)
		assert.Equal(t, wftActiveS, *woc.execWf.Spec.ActiveDeadlineSeconds)
	})
}

func TestWorkflowTemplateRefInvalidWF(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow("@testdata/operator_workflow_template_ref/invalid-wf.yaml")
	t.Run("ProcessWFWithStoredWFT", func(t *testing.T) {
		cancel, controller := newController(logging.TestContext(t.Context()), wf)
		defer cancel()
		ctx := logging.TestContext(t.Context())
		woc := newWorkflowOperationCtx(ctx, wf, controller)
		woc.operate(ctx)
		assert.Equal(t, wfv1.WorkflowError, woc.wf.Status.Phase)
	})
}

func TestWorkflowTemplateRefParamMerge(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow("@testdata/operator_workflow_template_ref/wf-with-param.yaml")
	wftmpl := wfv1.MustUnmarshalWorkflowTemplate("@testdata/operator_workflow_template_ref/wft-with-param.yaml")

	t.Run("CheckArgumentFromWF", func(t *testing.T) {
		ctx := logging.TestContext(t.Context())
		cancel, controller := newController(ctx, wf, wftmpl)
		defer cancel()
		woc := newWorkflowOperationCtx(ctx, wf, controller)
		woc.operate(ctx)
		assert.Equal(t, wf.Spec.Arguments.Parameters, woc.wf.Spec.Arguments.Parameters)
	})
}

// https://github.com/argoproj/argo-workflows/issues/14426
func TestWorkflowTemplateRefValueFromParamOverwrite(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow("@testdata/operator_workflow_template_ref/wf-with-value-param-override.yaml")
	wftmpl := wfv1.MustUnmarshalWorkflowTemplate("@testdata/operator_workflow_template_ref/wft-with-value-from-param.yaml")
	t.Run("CheckArgumentFromWFT", func(t *testing.T) {
		ctx := logging.TestContext(t.Context())
		cancel, controller := newController(ctx, wf, wftmpl)
		defer cancel()
		woc := newWorkflowOperationCtx(ctx, wf, controller)
		woc.operate(ctx)
		assert.Equal(t, wf.Spec.Arguments.Parameters, woc.execWf.Spec.Arguments.Parameters)
		assert.Equal(t, "configmap argument overwrite with argument", woc.execWf.Spec.Arguments.Parameters[0].Value.String())
	})
}

func TestWorkflowTemplateRefValueParamOverwrite(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow("@testdata/operator_workflow_template_ref/wf-with-value-from-param-override.yaml")
	wftmpl := wfv1.MustUnmarshalWorkflowTemplate("@testdata/operator_workflow_template_ref/wft-with-value-parameter.yaml")
	t.Run("CheckArgumentFromWFT", func(t *testing.T) {
		ctx := logging.TestContext(t.Context())
		cancel, controller := newController(ctx, wf, wftmpl)
		defer cancel()
		var cm apiv1.ConfigMap
		wfv1.MustUnmarshal("@testdata/operator_workflow_template_ref/config-map-message.yaml", &cm)
		_, err := controller.kubeclientset.CoreV1().ConfigMaps(cm.Namespace).Create(ctx, &cm, metav1.CreateOptions{})
		require.NoError(t, err)
		woc := newWorkflowOperationCtx(ctx, wf, controller)
		woc.operate(ctx)
		assert.Equal(t, wf.Spec.Arguments.Parameters, woc.execWf.Spec.Arguments.Parameters)
		assert.Equal(t, "config-properties", woc.execWf.Spec.Arguments.Parameters[0].ValueFrom.ConfigMapKeyRef.Name)
		assert.Equal(t, "message", woc.execWf.Spec.Arguments.Parameters[0].ValueFrom.ConfigMapKeyRef.Key)
	})
}

func TestWorkflowTemplateRefGetArtifactsFromTemplate(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow("@testdata/operator_workflow_template_ref/wf-with-template-with-artifact.yaml")
	wftmpl := wfv1.MustUnmarshalWorkflowTemplate("@testdata/operator_workflow_template_ref/wft-with-artifact.yaml")

	t.Run("CheckArtifactArgumentFromWF", func(t *testing.T) {
		ctx := logging.TestContext(t.Context())
		cancel, controller := newController(ctx, wf, wftmpl)
		defer cancel()
		woc := newWorkflowOperationCtx(ctx, wf, controller)
		woc.operate(ctx)
		assert.Len(t, woc.execWf.Spec.Arguments.Artifacts, 3)

		assert.Equal(t, "own-file", woc.execWf.Spec.Arguments.Artifacts[0].Name)
		assert.Equal(t, "binary-file", woc.execWf.Spec.Arguments.Artifacts[1].Name)
		assert.Equal(t, "data-file", woc.execWf.Spec.Arguments.Artifacts[2].Name)
	})
}

func TestWorkflowTemplateRefWithShutdownAndSuspend(t *testing.T) {
	t.Run("EntryPointMissingInStoredWfSpec", func(t *testing.T) {
		wf := wfv1.MustUnmarshalWorkflow(wfWithTmplRef)
		ctx := logging.TestContext(t.Context())
		cancel, controller := newController(ctx, wf, wfv1.MustUnmarshalWorkflowTemplate(wfTmpl))
		defer cancel()
		woc := newWorkflowOperationCtx(ctx, wf, controller)
		woc.operate(ctx)
		assert.Nil(t, woc.wf.Status.StoredWorkflowSpec.Suspend)
		wf1 := woc.wf.DeepCopy()
		// Updating Pod state
		makePodsPhase(ctx, woc, apiv1.PodPending)
		wf1.Status.StoredWorkflowSpec.Entrypoint = ""
		woc1 := newWorkflowOperationCtx(ctx, wf1, controller)
		woc1.operate(ctx)
		assert.NotNil(t, woc1.wf.Status.StoredWorkflowSpec.Entrypoint)
		assert.Equal(t, woc.wf.Spec.Entrypoint, woc1.wf.Status.StoredWorkflowSpec.Entrypoint)
	})

	t.Run("WorkflowTemplateRefWithSuspend", func(t *testing.T) {
		wf := wfv1.MustUnmarshalWorkflow(wfWithTmplRef)
		ctx := logging.TestContext(t.Context())
		cancel, controller := newController(ctx, wf, wfv1.MustUnmarshalWorkflowTemplate(wfTmpl))
		defer cancel()
		woc := newWorkflowOperationCtx(ctx, wf, controller)
		woc.operate(ctx)
		assert.Nil(t, woc.wf.Status.StoredWorkflowSpec.Suspend)
		wf1 := woc.wf.DeepCopy()
		// Updating Pod state
		makePodsPhase(ctx, woc, apiv1.PodPending)
		wf1.Spec.Suspend = new(true)
		woc1 := newWorkflowOperationCtx(ctx, wf1, controller)
		woc1.operate(ctx)
		assert.NotNil(t, woc1.wf.Status.StoredWorkflowSpec.Suspend)
		assert.True(t, *woc1.wf.Status.StoredWorkflowSpec.Suspend)
	})
	t.Run("WorkflowTemplateRefWithShutdownTerminate", func(t *testing.T) {
		wf := wfv1.MustUnmarshalWorkflow(wfWithTmplRef)
		ctx := logging.TestContext(t.Context())
		cancel, controller := newController(ctx, wf, wfv1.MustUnmarshalWorkflowTemplate(wfTmpl))
		defer cancel()
		woc := newWorkflowOperationCtx(ctx, wf, controller)
		woc.operate(ctx)
		assert.Empty(t, woc.wf.Status.StoredWorkflowSpec.Shutdown)
		wf1 := woc.wf.DeepCopy()
		// Updating Pod state
		makePodsPhase(ctx, woc, apiv1.PodPending)
		wf1.Spec.Shutdown = wfv1.ShutdownStrategyTerminate
		woc1 := newWorkflowOperationCtx(ctx, wf1, controller)
		woc1.operate(ctx)
		assert.NotEmpty(t, woc1.wf.Status.StoredWorkflowSpec.Shutdown)
		assert.Equal(t, wfv1.ShutdownStrategyTerminate, woc1.wf.Status.StoredWorkflowSpec.Shutdown)
		for _, node := range woc1.wf.Status.Nodes {
			require.NotNil(t, node)
			assert.Contains(t, node.Message, "workflow shutdown with strategy")
			assert.Contains(t, node.Message, "Terminate")
		}
	})
	t.Run("WorkflowTemplateRefWithShutdownStop", func(t *testing.T) {
		wf := wfv1.MustUnmarshalWorkflow(wfWithTmplRef)
		ctx := logging.TestContext(t.Context())
		cancel, controller := newController(ctx, wf, wfv1.MustUnmarshalWorkflowTemplate(wfTmpl))
		defer cancel()
		woc := newWorkflowOperationCtx(ctx, wf, controller)
		woc.operate(ctx)
		assert.Empty(t, woc.wf.Status.StoredWorkflowSpec.Shutdown)
		wf1 := woc.wf.DeepCopy()
		// Updating Pod state
		makePodsPhase(ctx, woc, apiv1.PodPending)
		wf1.Spec.Shutdown = wfv1.ShutdownStrategyStop
		woc1 := newWorkflowOperationCtx(ctx, wf1, controller)
		woc1.operate(ctx)
		assert.NotEmpty(t, woc1.wf.Status.StoredWorkflowSpec.Shutdown)
		assert.Equal(t, wfv1.ShutdownStrategyStop, woc1.wf.Status.StoredWorkflowSpec.Shutdown)
		for _, node := range woc1.wf.Status.Nodes {
			require.NotNil(t, node)
			assert.Contains(t, node.Message, "workflow shutdown with strategy")
			assert.Contains(t, node.Message, "Stop")
		}
	})
	t.Run("WorkflowTemplateRefWithSuspendWithShutdownTerminate", func(t *testing.T) {
		wf := wfv1.MustUnmarshalWorkflow(wfWithTmplRef)
		wf1 := wf.DeepCopy()
		wf1.Spec.Suspend = new(true)
		ctx := logging.TestContext(t.Context())
		cancel, controller := newController(ctx, wf1, wfv1.MustUnmarshalWorkflowTemplate(wfTmpl))
		defer cancel()

		woc := newWorkflowOperationCtx(ctx, wf1, controller)
		woc.operate(ctx)
		assert.Equal(t, wfv1.WorkflowRunning, woc.wf.Status.Phase)
		assert.Empty(t, woc.wf.Status.Nodes)
		require.NotNil(t, woc.wf.Status.StoredWorkflowSpec.Suspend)
		assert.True(t, *woc.wf.Status.StoredWorkflowSpec.Suspend)

		wf2 := woc.wf.DeepCopy()
		wf2.Spec.Shutdown = wfv1.ShutdownStrategyTerminate
		woc = newWorkflowOperationCtx(ctx, wf2, controller)
		woc.operate(ctx)

		node := woc.wf.Status.Nodes.FindByDisplayName(wf.Name)
		require.NotNil(t, node)
		assert.Equal(t, wfv1.NodeFailed, node.Phase)
		assert.Contains(t, node.Message, "workflow shutdown with strategy")
		assert.Contains(t, node.Message, "Terminate")

		assert.Equal(t, wfv1.WorkflowFailed, woc.wf.Status.Phase)
		assert.Equal(t, "Stopped with strategy 'Terminate'", woc.wf.Status.Message)
	})

	t.Run("WorkflowTemplateRefWithSuspendWithShutdownStop", func(t *testing.T) {
		wf := wfv1.MustUnmarshalWorkflow(wfWithTmplRef)
		wf1 := wf.DeepCopy()
		wf1.Spec.Suspend = new(true)
		ctx := logging.TestContext(t.Context())
		cancel, controller := newController(ctx, wf1, wfv1.MustUnmarshalWorkflowTemplate(wfTmpl))
		defer cancel()

		woc := newWorkflowOperationCtx(ctx, wf1, controller)
		woc.operate(ctx)
		assert.Equal(t, wfv1.WorkflowRunning, woc.wf.Status.Phase)
		assert.Empty(t, woc.wf.Status.Nodes)
		require.NotNil(t, woc.wf.Status.StoredWorkflowSpec.Suspend)
		assert.True(t, *woc.wf.Status.StoredWorkflowSpec.Suspend)

		wf2 := woc.wf.DeepCopy()
		wf2.Spec.Shutdown = wfv1.ShutdownStrategyStop
		woc = newWorkflowOperationCtx(ctx, wf2, controller)
		woc.operate(ctx)

		node := woc.wf.Status.Nodes.FindByDisplayName(wf.Name)
		require.NotNil(t, node)
		assert.Equal(t, wfv1.NodeFailed, node.Phase)
		assert.Contains(t, node.Message, "workflow shutdown with strategy")
		assert.Contains(t, node.Message, "Stop")

		assert.Equal(t, wfv1.WorkflowFailed, woc.wf.Status.Phase)
		assert.Equal(t, "Stopped with strategy 'Stop'", woc.wf.Status.Message)
	})
}

func TestSuspendResumeWorkflowTemplateRef(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow("@testdata/operator_workflow_template_ref/suspendwf.yaml")
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx, wf, wfv1.MustUnmarshalWorkflowTemplate(wfTmpl))
	defer cancel()
	woc := newWorkflowOperationCtx(ctx, wf, controller)
	woc.operate(ctx)
	assert.True(t, *woc.wf.Status.StoredWorkflowSpec.Suspend)
	woc.wf.Spec.Suspend = nil
	woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
	woc.operate(ctx)
	assert.Nil(t, woc.wf.Status.StoredWorkflowSpec.Suspend)
}

func TestWorkflowTemplateUpdateScenario(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow(wfWithTmplRef)
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx, wf, wfv1.MustUnmarshalWorkflowTemplate(wfTmpl))
	defer cancel()
	woc := newWorkflowOperationCtx(ctx, wf, controller)
	woc.operate(ctx)
	assert.NotEmpty(t, woc.wf.Status.StoredWorkflowSpec)
	assert.NotEmpty(t, woc.wf.Status.StoredWorkflowSpec.Templates[0].Container)

	cancel, controller = newController(ctx, woc.wf, wfv1.MustUnmarshalWorkflowTemplate("@testdata/operator_workflow_template_ref/wf-tmpl-upt.yaml"))
	defer cancel()

	woc1 := newWorkflowOperationCtx(ctx, woc.wf, controller)
	woc1.operate(ctx)
	assert.NotEmpty(t, woc1.wf.Status.StoredWorkflowSpec)
	assert.Equal(t, woc.wf.Status.StoredWorkflowSpec, woc1.wf.Status.StoredWorkflowSpec)
}

func TestWFTWithVol(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow("@testdata/operator_workflow_template_ref/wf-tmpl-with-vol.yaml")
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx, wf, wfv1.MustUnmarshalWorkflowTemplate(wfTmpl))
	defer cancel()
	woc := newWorkflowOperationCtx(ctx, wf, controller)
	woc.operate(ctx)
	pvc, err := controller.kubeclientset.CoreV1().PersistentVolumeClaims("default").List(ctx, metav1.ListOptions{})
	require.NoError(t, err)
	assert.Len(t, pvc.Items, 1)
	makePodsPhase(ctx, woc, apiv1.PodSucceeded)
	woc.operate(ctx)
	pvc, err = controller.kubeclientset.CoreV1().PersistentVolumeClaims("default").List(ctx, metav1.ListOptions{})
	require.NoError(t, err)
	assert.Empty(t, pvc.Items)
}

func TestSubmitWorkflowTemplateRefWithoutRBAC(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow("@testdata/operator_workflow_template_ref/wf-tmp.yaml")
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx, wf, wfv1.MustUnmarshalWorkflowTemplate(wfTmpl))
	defer cancel()
	woc := newWorkflowOperationCtx(ctx, wf, controller)
	woc.controller.cwftmplInformer = nil
	woc.operate(ctx)
	assert.Equal(t, wfv1.WorkflowError, woc.wf.Status.Phase)
}

func TestWorkflowTemplateWithDynamicRef(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx, wfv1.MustUnmarshalWorkflow("@testdata/operator_workflow_template_ref/wf-with-dynamic-ref.yaml"), wfv1.MustUnmarshalWorkflowTemplate("@testdata/operator_workflow_template_ref/wf-template-hello.yaml"))
	defer cancel()

	woc := newWorkflowOperationCtx(ctx, wfv1.MustUnmarshalWorkflow("@testdata/operator_workflow_template_ref/wf-with-dynamic-ref.yaml"), controller)
	woc.operate(ctx)
	assert.Equal(t, wfv1.WorkflowRunning, woc.wf.Status.Phase)
	pods, err := listPods(ctx, woc)
	require.NoError(t, err)
	assert.NotEmpty(t, pods.Items, "pod was not created successfully")
	pod := pods.Items[0]
	assert.Contains(t, pod.Name, "hello-world")
	assert.Equal(t, "docker/whalesay", pod.Spec.Containers[1].Image)
	assert.Contains(t, "hello", pod.Spec.Containers[1].Args[0])
	pod = pods.Items[1]
	assert.Contains(t, pod.Name, "hello-world")
	assert.Equal(t, "docker/whalesay", pod.Spec.Containers[1].Image)
	assert.Contains(t, "hello", pod.Spec.Containers[1].Args[0])
	makePodsPhase(ctx, woc, apiv1.PodSucceeded)
	woc.operate(ctx)
	assert.Equal(t, wfv1.WorkflowSucceeded, woc.wf.Status.Phase)
}

func TestWorkflowTemplateWithPodMetadata(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx, wfv1.MustUnmarshalWorkflow("@testdata/operator_workflow_template_ref/wf-with-template-ref.yaml"), wfv1.MustUnmarshalClusterWorkflowTemplate("@testdata/operator_workflow_template_ref/wf-template-with-pod-metadata.yaml"))
	defer cancel()

	woc := newWorkflowOperationCtx(ctx, wfv1.MustUnmarshalWorkflow("@testdata/operator_workflow_template_ref/wf-with-template-ref.yaml"), controller)
	woc.operate(ctx)
	assert.Equal(t, wfv1.WorkflowRunning, woc.wf.Status.Phase)
	pods, err := listPods(ctx, woc)
	require.NoError(t, err)
	assert.NotEmpty(t, len(pods.Items) > 0, "pod was not created successfully")
	pod := pods.Items[0]
	assert.Contains(t, pod.Labels, "caller-label")
	assert.Contains(t, pod.Labels, "workflow-template-label")
	assert.Contains(t, pod.Annotations, "all-pods-should-have-this")
}
