package controller

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/tools/cache"

	"github.com/argoproj/argo-workflows/v4/pkg/apis/workflow"
	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/util/logging"
	"github.com/argoproj/argo-workflows/v4/workflow/common"
)

// indexWorkflow puts an unstructured Workflow into the controller's informer index,
// which is where workflowActive reads from.
func indexWorkflow(t *testing.T, wfc *WorkflowController, namespace, name string, labels map[string]string) string {
	t.Helper()
	un := &unstructured.Unstructured{}
	un.SetAPIVersion(wfv1.SchemeGroupVersion.String())
	un.SetKind(workflow.WorkflowKind)
	un.SetNamespace(namespace)
	un.SetName(name)
	if labels != nil {
		un.SetLabels(labels)
	}
	require.NoError(t, wfc.wfInformer.GetIndexer().Add(un))
	key, err := cache.MetaNamespaceKeyFunc(un)
	require.NoError(t, err)
	return key
}

// TestWorkflowActive covers the predicate that decides whether CheckWorkflowExistence
// may reclaim a synchronization lock. See #16772: a lock held by a Workflow that has
// completed is a leak, and before this it was reclaimed by nothing — the three
// status-gated release paths cannot see a lock whose status was never written, and the
// reconciler skipped it because the Workflow object still existed.
func TestWorkflowActive(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx)
	defer cancel()

	t.Run("running workflow still holds its lock", func(t *testing.T) {
		key := indexWorkflow(t, controller, "default", "running-wf",
			map[string]string{common.LabelKeyCompleted: "false"})
		assert.True(t, controller.workflowActive(ctx, key))
	})

	t.Run("workflow with no completed label is left alone", func(t *testing.T) {
		// Absent label is not evidence of completion. Releasing on a missing label
		// would hand the lock to a second workflow while the first is still running.
		key := indexWorkflow(t, controller, "default", "unlabelled-wf", nil)
		assert.True(t, controller.workflowActive(ctx, key))
	})

	t.Run("completed workflow cannot still be holding a lock", func(t *testing.T) {
		key := indexWorkflow(t, controller, "default", "completed-wf",
			map[string]string{common.LabelKeyCompleted: "true"})
		assert.False(t, controller.workflowActive(ctx, key),
			"a completed workflow's lock is leaked, not held — this is the #16772 case")
	})

	t.Run("absent workflow cannot be holding a lock", func(t *testing.T) {
		assert.False(t, controller.workflowActive(ctx, "default/never-existed"))
	})
}
