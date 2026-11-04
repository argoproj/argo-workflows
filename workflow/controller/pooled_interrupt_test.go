package controller

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
)

// updateStored changes the stored workflow, as another writer, or time
// passing, would.
func updateStored(ctx context.Context, t *testing.T, controller *WorkflowController, wf *wfv1.Workflow, change func(*wfv1.Workflow)) {
	t.Helper()
	wfs := controller.wfclientset.ArgoprojV1alpha1().Workflows(wf.Namespace)
	stored, err := wfs.Get(ctx, wf.Name, metav1.GetOptions{})
	require.NoError(t, err)
	change(stored)
	_, err = wfs.Update(ctx, stored, metav1.UpdateOptions{})
	require.NoError(t, err)
}
