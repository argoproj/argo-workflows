package controller

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/util/logging"
	"github.com/argoproj/argo-workflows/v4/workflow/templateresolution"
)

func TestExecuteTemplate_NoSync_ReturnsNode(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx)
	defer cancel()

	wf := &wfv1.Workflow{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-wf",
			Namespace: "default",
		},
		Spec: wfv1.WorkflowSpec{
			Entrypoint: "main",
			Templates: []wfv1.Template{
				{
					Name: "main",
					Container: &apiv1.Container{
						Image: "alpine",
						Name:  "main",
					},
				},
			},
		},
	}

	woc := newWorkflowOperationCtx(ctx, wf, controller)

	tmplCtx := templateresolution.NewContext(nil, nil, wf, wf, woc.log)

	node, err := woc.reconcileTemplate(ctx, woc.wf.Name, &wfv1.WorkflowStep{Template: "main"}, tmplCtx, wfv1.Arguments{}, &executeTemplateOpts{})

	require.NoError(t, err)
	assert.NotNil(t, node, "Node should not be nil")
}
