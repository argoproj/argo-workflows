package controller

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	sqldbmocks "github.com/argoproj/argo-workflows/v4/persist/sqldb/mocks"
	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	fakewfclientset "github.com/argoproj/argo-workflows/v4/pkg/client/clientset/versioned/fake"
	"github.com/argoproj/argo-workflows/v4/util/logging"
	"github.com/argoproj/argo-workflows/v4/workflow/hydrator"
	"github.com/argoproj/argo-workflows/v4/workflow/packer"
)

// The cleanup callback must read the API's referenced node version, never a
// caller's in-memory nodes or a node version belonging to a previous Workflow.
func TestLookupCleanupHydratesPersistedCapture(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	for _, storage := range []string{"raw", "compressed", "offloaded"} {
		t.Run(storage, func(t *testing.T) {
			wf := &wfv1.Workflow{ObjectMeta: metav1.ObjectMeta{Name: "wf", Namespace: "default", UID: "wf-uid"}, Status: wfv1.WorkflowStatus{Nodes: wfv1.Nodes{"node": {ID: "node", Name: "wf", Phase: wfv1.NodeSucceeded, CapturedPodUID: "pod-uid"}}}}
			repo := &sqldbmocks.OffloadNodeStatusRepo{}
			switch storage {
			case "compressed":
				node := wf.Status.Nodes["node"]
				node.Message = strings.Repeat("capture-result", 200)
				wf.Status.Nodes["node"] = node
				restore := packer.SetMaxWorkflowSize(1024)
				defer restore()
				require.NoError(t, packer.CompressWorkflowIfNeeded(ctx, wf))
				require.NotEmpty(t, wf.Status.CompressedNodes)
			case "offloaded":
				repo.On("Get", "wf-uid", "api-referenced-version").Return(wf.Status.Nodes.DeepCopy(), nil).Once()
				wf.Status.Nodes = nil
				wf.Status.OffloadNodeStatusVersion = "api-referenced-version"
			}
			wfc := &WorkflowController{wfclientset: fakewfclientset.NewClientset(wf), hydrator: hydrator.New(repo)}
			result, err := wfc.lookupWorkflowForPodCleanup(ctx, wf.Namespace, wf.Name, true)
			require.NoError(t, err)
			assert.Equal(t, "pod-uid", result.Status.Nodes["node"].CapturedPodUID)
			assert.Empty(t, result.Status.CompressedNodes)
			assert.Empty(t, result.Status.OffloadNodeStatusVersion)
			repo.AssertExpectations(t)
		})
	}
}
