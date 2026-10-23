package controller

import (
	"testing"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/util/logging"
)

// TestWFDefaultsWithWorkflow checks that the configured workflow defaults are merged into a
// submitted Workflow, both when it sets its own entrypoint and when it leaves it to the default.
func TestWFDefaultsWithWorkflow(t *testing.T) {
	assert := assert.New(t)

	wfDefault := wfv1.MustUnmarshalWorkflow("@testdata/operator_wfdefault/workflow-defaults.yaml")
	wf := wfv1.MustUnmarshalWorkflow("@testdata/operator_wfdefault/simple-workflow.yaml")
	wf1 := wf.DeepCopy()
	wfResult := wfv1.MustUnmarshalWorkflow("@testdata/operator_wfdefault/simple-workflow-with-defaults.yaml")
	ctx := logging.TestContext(t.Context())
	cancel, controller := newControllerWithDefaults(ctx)
	defer cancel()

	controller.Config.WorkflowDefaults = wfDefault
	woc := newWorkflowOperationCtx(ctx, wf, controller)
	woc.operate(ctx)
	assert.Equal(woc.wf.Spec, wfResult.Spec)
	assert.Contains(woc.wf.Labels, "testLabel")
	assert.Contains(woc.wf.Annotations, "testAnnotation")

	wf1.Spec.Entrypoint = ""
	woc = newWorkflowOperationCtx(ctx, wf1, controller)
	woc.operate(ctx)
	assert.Equal(woc.wf.Spec, wfResult.Spec)
	assert.Contains(woc.wf.Labels, "testLabel")
	assert.Contains(woc.wf.Annotations, "testAnnotation")
}

// TestWFDefaultWithWFTAndWf checks that the workflow defaults are merged into the spec stored for
// a Workflow that only references a WorkflowTemplate, and that fields the Workflow sets itself
// still win over the defaults.
func TestWFDefaultWithWFTAndWf(t *testing.T) {
	assert := assert.New(t)
	wfDefault := wfv1.MustUnmarshalWorkflow("@testdata/operator_wfdefault/workflow-defaults.yaml")
	wft := wfv1.MustUnmarshalWorkflowTemplate("@testdata/operator_wfdefault/simple-workflow-template.yaml")
	var resultSpec wfv1.WorkflowSpec
	wfv1.MustUnmarshal("@testdata/operator_wfdefault/stored-workflow-spec.yaml", &resultSpec)

	t.Run("SubmitSimpleWorkflowRef", func(t *testing.T) {
		ctx := logging.TestContext(t.Context())
		cancel, controller := newController(ctx, wft)
		defer cancel()
		controller.Config.WorkflowDefaults = wfDefault

		wf := wfv1.Workflow{ObjectMeta: metav1.ObjectMeta{Namespace: "default"}, Spec: wfv1.WorkflowSpec{WorkflowTemplateRef: &wfv1.WorkflowTemplateRef{Name: "workflow-template-submittable"}}}
		woc := newWorkflowOperationCtx(ctx, &wf, controller)
		woc.operate(ctx)
		resultSpec.WorkflowTemplateRef = &wfv1.WorkflowTemplateRef{Name: "workflow-template-submittable"}
		assert.Equal(resultSpec, woc.execWf.Spec)
		assert.Equal(&resultSpec, woc.wf.Status.StoredWorkflowSpec)
	})

	t.Run("SubmitComplexWorkflowRef", func(t *testing.T) {
		ctx := logging.TestContext(t.Context())
		cancel, controller := newController(ctx, wft)
		defer cancel()
		controller.Config.WorkflowDefaults = wfDefault

		ttlStrategy := wfv1.TTLStrategy{
			SecondsAfterCompletion: new(int32(10)),
		}

		wf := wfv1.Workflow{
			ObjectMeta: metav1.ObjectMeta{Namespace: "default"},
			Spec: wfv1.WorkflowSpec{
				WorkflowTemplateRef: &wfv1.WorkflowTemplateRef{Name: "workflow-template-submittable"},
				Entrypoint:          "Test",
				TTLStrategy:         &ttlStrategy,
			},
		}
		resultSpec.Entrypoint = "Test"
		resultSpec.TTLStrategy = &ttlStrategy
		resultSpec.WorkflowTemplateRef = &wfv1.WorkflowTemplateRef{Name: "workflow-template-submittable"}

		woc := newWorkflowOperationCtx(ctx, &wf, controller)
		woc.operate(ctx)
		assert.Equal(resultSpec, woc.execWf.Spec)
		assert.Equal(&resultSpec, woc.wf.Status.StoredWorkflowSpec)
	})

	t.Run("SubmitComplexWorkflowRefWithArguments", func(t *testing.T) {
		ctx := logging.TestContext(t.Context())
		cancel, controller := newController(ctx, wft)
		defer cancel()
		controller.Config.WorkflowDefaults = wfDefault

		param := wfv1.Parameter{
			Name:  "Test",
			Value: wfv1.AnyStringPtr("welcome"),
		}
		art := wfv1.Artifact{
			Name: "TestA",
			Path: "tmp/test",
		}

		ttlStrategy := wfv1.TTLStrategy{
			SecondsAfterCompletion: new(int32(10)),
		}

		wf := wfv1.Workflow{
			ObjectMeta: metav1.ObjectMeta{Namespace: "default"},
			Spec: wfv1.WorkflowSpec{
				WorkflowTemplateRef: &wfv1.WorkflowTemplateRef{Name: "workflow-template-submittable"},
				Entrypoint:          "Test",
				TTLStrategy:         &ttlStrategy,
				Arguments: wfv1.Arguments{
					Parameters: []wfv1.Parameter{param},
					Artifacts:  wfv1.Artifacts{art},
				},
			},
		}
		// resultSpec.Arguments.Parameters = append(resultSpec.Arguments.Parameters, args.Parameters...)
		resultSpec.Entrypoint = "Test"
		resultSpec.TTLStrategy = &ttlStrategy
		resultSpec.WorkflowTemplateRef = &wfv1.WorkflowTemplateRef{Name: "workflow-template-submittable"}
		resultSpec.Arguments.Parameters = append(resultSpec.Arguments.Parameters, param)
		resultSpec.Arguments.Artifacts = append(resultSpec.Arguments.Artifacts, art)

		woc := newWorkflowOperationCtx(ctx, &wf, controller)
		woc.operate(ctx)
		assert.Contains(woc.execWf.Spec.Arguments.Parameters, param)
		assert.Contains(woc.wf.Status.StoredWorkflowSpec.Arguments.Artifacts, art)
	})
}
