package controller

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func operateExample(t *testing.T, file string) *wfOperationCtx {
	t.Helper()
	ctx := context.Background()
	cancel, controller := newController()
	t.Cleanup(cancel)
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("")

	data, err := os.ReadFile(file)
	require.NoError(t, err)
	wf := wfv1.MustUnmarshalWorkflow(data)
	wf, err = wfcset.Create(ctx, wf, metav1.CreateOptions{})
	require.NoError(t, err)
	woc := newWorkflowOperationCtx(wf, controller)
	woc.operate(ctx)
	return woc
}

// TestExamplesSkippedOutputDefaults verifies the example workflows under
// examples/skipped-output-defaults/ resolve the consumer input as documented.
func TestExamplesSkippedOutputDefaults(t *testing.T) {
	cases := []struct {
		file string
		want string
	}{
		{"../../examples/skipped-output-defaults/producer-output-default.yaml", "hello-from-producer"},
		{"../../examples/skipped-output-defaults/consumer-input-default.yaml", "fallback-from-input"},
	}
	for _, tc := range cases {
		t.Run(tc.file, func(t *testing.T) {
			woc := operateExample(t, tc.file)

			producer := woc.wf.Status.Nodes.FindByDisplayName("producer")
			require.NotNil(t, producer)
			require.Equal(t, wfv1.NodeSkipped, producer.Phase)

			consumer := woc.wf.Status.Nodes.FindByDisplayName("consumer")
			require.NotNil(t, consumer)
			require.NotNil(t, consumer.Inputs)
			in := consumer.Inputs.GetParameterByName("in")
			require.NotNil(t, in)
			require.NotNil(t, in.Value)
			assert.Equal(t, tc.want, in.Value.String())
		})
	}
}

// TestExampleExpressionFallback verifies the expression-fallback example: a skipped output with no
// default resolves to nil so the DAG output's ValueFrom.Expression can fall back via `??`.
func TestExampleExpressionFallback(t *testing.T) {
	woc := operateExample(t, "../../examples/skipped-output-defaults/expression-fallback.yaml")

	// The DAG's root node carries the aggregated template outputs.
	var dag *wfv1.NodeStatus
	for _, n := range woc.wf.Status.Nodes {
		if n.Type == wfv1.NodeTypeDAG && n.BoundaryID == "" {
			node := n
			dag = &node
			break
		}
	}
	require.NotNil(t, dag)
	require.NotNil(t, dag.Outputs)
	var result *wfv1.Parameter
	for i := range dag.Outputs.Parameters {
		if dag.Outputs.Parameters[i].Name == "result" {
			result = &dag.Outputs.Parameters[i]
			break
		}
	}
	require.NotNil(t, result)
	require.NotNil(t, result.Value)
	assert.Equal(t, "expr-fallback", result.Value.String())
}
