package pod

import (
	"context"
	"syscall"
	"testing"

	"github.com/stretchr/testify/require"
	apiv1 "k8s.io/api/core/v1"
	"k8s.io/client-go/rest"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
)

func cleanupCostInstall(h *cleanupCostHarness) {
	h.c.lookupWorkflow = h.lookup
	h.c.SetWorkflowHydrator(h.hydrateWorkflow)
	h.c.signalContainer = func(context.Context, *rest.Config, *apiv1.Pod, string, syscall.Signal) error {
		h.counts.Signals++
		return nil
	}
}
func cleanupCostRecoveryAvailable() bool { return true }
func cleanupCostActiveBudget(t *testing.T, c cleanupCostCounts) {
	t.Helper()
	require.LessOrEqual(t, c.Pod["get"], 1)
	require.LessOrEqual(t, c.Workflow["get"], 1, "reuse the authoritative owner representation when hydrating")
	require.LessOrEqual(t, c.Hydrate, 1)
}

func cleanupCostUsefulBudget(t *testing.T, scenario string, c cleanupCostCounts) {
	t.Helper()
	if scenario == "daemon" || scenario == "cancellation" {
		require.LessOrEqual(t, c.Pod["get"], 3)
		require.LessOrEqual(t, c.Workflow["get"], 1)
		require.LessOrEqual(t, c.Hydrate, 1)
		return
	}
	require.LessOrEqual(t, c.Pod["get"], 2)
	require.LessOrEqual(t, c.Workflow["get"], 2)
	require.LessOrEqual(t, c.Hydrate, 2)
	if scenario == "orphan" || scenario == "deleting-owner" || scenario == "deleting-pending" {
		require.Zero(t, c.Hydrate)
	}
}
func cleanupCostSweep(ctx context.Context, c *Controller, wf *wfv1.Workflow) {
	c.QueueWorkflowCleanup(ctx, wf)
}
