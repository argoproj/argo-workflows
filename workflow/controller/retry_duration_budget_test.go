package controller

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/util/intstr"
	intstrutil "github.com/argoproj/argo-workflows/v4/util/intstr"
	"github.com/argoproj/argo-workflows/v4/util/logging"
)

// retryFixture builds a retry node with `children` fulfilled children in `childPhase`,
// and returns the parent node plus the child ids, ready for processNodeRetries.
func retryFixture(t *testing.T, children int, childPhase wfv1.NodePhase) (*wfOperationCtx, *wfv1.NodeStatus, []string) {
	t.Helper()
	cancel, controller := newController(logging.TestContext(t.Context()))
	t.Cleanup(cancel)
	wf := wfv1.MustUnmarshalWorkflow(helloWorldWf)
	ctx := logging.TestContext(t.Context())
	woc := newWorkflowOperationCtx(ctx, wf, controller)

	nodeName := "test-node"
	nodeID := woc.wf.NodeID(nodeName)
	ctx, node := woc.initializeNode(ctx, nodeName, wfv1.NodeTypeRetry, "", &wfv1.WorkflowStep{}, "", wfv1.NodeRunning, &wfv1.NodeFlag{}, true)
	woc.wf.Status.Nodes[nodeID] = *node

	var childIDs []string
	for i := range children {
		childName := fmt.Sprintf("%s(%d)", nodeName, i)
		_, child := woc.initializeNode(ctx, childName, wfv1.NodeTypePod, "", &wfv1.WorkflowStep{}, "", wfv1.NodeRunning, &wfv1.NodeFlag{Retried: true}, true)
		woc.addChildNode(ctx, nodeName, childName)
		woc.markNodePhase(ctx, childName, childPhase)
		childIDs = append(childIDs, child.ID)
	}
	n, err := woc.wf.GetNodeByName(nodeName)
	require.NoError(t, err)
	return woc, n, childIDs
}

// TestRetryAllowedIgnoringDurationBudget_AgreesWithRetryPath is the test the
// predicate's doc comment promises. retryAllowedIgnoringDurationBudget duplicates the
// policy / retryability / limit / expression conditions instead of restructuring the
// live retry path, so the duplication needs something that fails when the two drift.
//
// With no backoff configured there is no duration budget, so processNodeRetries'
// own verdict is exactly "would this retry go ahead" — which is what the predicate
// claims to answer. Any disagreement is the drift.
func TestRetryAllowedIgnoringDurationBudget_AgreesWithRetryPath(t *testing.T) {
	cases := []struct {
		name       string
		childPhase wfv1.NodePhase
		children   int
		strategy   wfv1.RetryStrategy
	}{
		{"onFailure policy, failed child, under limit", wfv1.NodeFailed, 1,
			wfv1.RetryStrategy{Limit: intstrutil.ParsePtr("3"), RetryPolicy: wfv1.RetryPolicyOnFailure}},
		{"onFailure policy, failed child, limit exhausted", wfv1.NodeFailed, 4,
			wfv1.RetryStrategy{Limit: intstrutil.ParsePtr("2"), RetryPolicy: wfv1.RetryPolicyOnFailure}},
		{"onError policy but child merely failed", wfv1.NodeFailed, 1,
			wfv1.RetryStrategy{Limit: intstrutil.ParsePtr("3"), RetryPolicy: wfv1.RetryPolicyOnError}},
		{"onError policy, errored child", wfv1.NodeError, 1,
			wfv1.RetryStrategy{Limit: intstrutil.ParsePtr("3"), RetryPolicy: wfv1.RetryPolicyOnError}},
		{"always policy, errored child", wfv1.NodeError, 1,
			wfv1.RetryStrategy{Limit: intstrutil.ParsePtr("3"), RetryPolicy: wfv1.RetryPolicyAlways}},
		{"expression refuses the retry", wfv1.NodeFailed, 1,
			wfv1.RetryStrategy{Limit: intstrutil.ParsePtr("3"), RetryPolicy: wfv1.RetryPolicyOnFailure, Expression: "false"}},
		{"expression permits the retry", wfv1.NodeFailed, 1,
			wfv1.RetryStrategy{Limit: intstrutil.ParsePtr("3"), RetryPolicy: wfv1.RetryPolicyOnFailure, Expression: "true"}},
		{"limit of zero", wfv1.NodeFailed, 1,
			wfv1.RetryStrategy{Limit: intstrutil.ParsePtr("0"), RetryPolicy: wfv1.RetryPolicyOnFailure}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			woc, node, childIDs := retryFixture(t, tc.children, tc.childPhase)
			lastChild, err := woc.wf.Status.Nodes.Get(childIDs[len(childIDs)-1])
			require.NoError(t, err)

			predicted := woc.retryAllowedIgnoringDurationBudget(node, lastChild, childIDs, tc.strategy)

			// No Backoff => no duration budget => the real path's verdict is purely
			// policy/retryability/limit/expression, the same question the predicate asks.
			ctx := logging.TestContext(t.Context())
			got, _, err := woc.processNodeRetries(ctx, node, tc.strategy, &executeTemplateOpts{})
			require.NoError(t, err)
			actual := !got.Phase.Fulfilled(got.TaskResultSynced)

			assert.Equal(t, actual, predicted,
				"predicate says %v but the retry path says %v — retryAllowedIgnoringDurationBudget has drifted from processNodeRetries",
				predicted, actual)
		})
	}
}

// A retry the limit would have rejected anyway must not be attributed to the duration
// budget. This is the requirement from #16856 that the counter report the reason the
// retry did not happen rather than whichever check the controller evaluated first.
func TestRetryAllowedIgnoringDurationBudget_FalseWhenLimitExhausted(t *testing.T) {
	woc, node, childIDs := retryFixture(t, 4, wfv1.NodeFailed)
	lastChild, err := woc.wf.Status.Nodes.Get(childIDs[len(childIDs)-1])
	require.NoError(t, err)

	strategy := wfv1.RetryStrategy{Limit: intstrutil.ParsePtr("2"), RetryPolicy: wfv1.RetryPolicyOnFailure}
	assert.False(t, woc.retryAllowedIgnoringDurationBudget(node, lastChild, childIDs, strategy),
		"limit is already exhausted, so the duration budget is not the reason this retry stops")
}

func TestRetryAllowedIgnoringDurationBudget_TrueWhenOnlyTheBudgetStopsIt(t *testing.T) {
	woc, node, childIDs := retryFixture(t, 1, wfv1.NodeFailed)
	lastChild, err := woc.wf.Status.Nodes.Get(childIDs[len(childIDs)-1])
	require.NoError(t, err)

	strategy := wfv1.RetryStrategy{Limit: intstrutil.ParsePtr("5"), RetryPolicy: wfv1.RetryPolicyOnFailure}
	assert.True(t, woc.retryAllowedIgnoringDurationBudget(node, lastChild, childIDs, strategy),
		"nothing but the duration budget stands in the way, so this is a budget termination")
}

var _ = intstr.Int32 // keep the intstr import honest if helpers change
