package workflow

import (
	"context"
	"testing"
	"time"

	"github.com/go-jose/go-jose/v4/jwt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/argoproj/argo-workflows/v4/persist/sqldb"
	"github.com/argoproj/argo-workflows/v4/persist/sqldb/mocks"
	workflowpkg "github.com/argoproj/argo-workflows/v4/pkg/apiclient/workflow"
	"github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	v1alpha "github.com/argoproj/argo-workflows/v4/pkg/client/clientset/versioned/fake"
	authtypes "github.com/argoproj/argo-workflows/v4/server/auth/types"
	"github.com/argoproj/argo-workflows/v4/server/workflow/store"
	"github.com/argoproj/argo-workflows/v4/util/instanceid"
)

// offloadedWorkflow builds a workflow whose node status lives in the offload table. Pages are
// ordered by startedat descending, so startedAt decides which page a fixture lands on.
func offloadedWorkflow(name, uid, version string, startedAt time.Time) v1alpha1.Workflow {
	return v1alpha1.Workflow{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: "argo", UID: types.UID(uid)},
		Status: v1alpha1.WorkflowStatus{
			OffloadNodeStatusVersion: version,
			StartedAt:                metav1.NewTime(startedAt),
		},
	}
}

// inlineWorkflow builds a workflow that keeps its node status on the object itself, so the
// offload table has nothing for it.
func inlineWorkflow(name, uid string) v1alpha1.Workflow {
	return offloadedWorkflow(name, uid, "", time.Time{})
}

// offloadTestServer builds the smallest server that can serve ListWorkflows, and hands back
// the offload mock so tests can assert on how it was called.
func offloadTestServer(t *testing.T, wfs ...v1alpha1.Workflow) (Server, context.Context, *mocks.OffloadNodeStatusRepo) {
	t.Helper()

	offloadNodeStatusRepo := &mocks.OffloadNodeStatusRepo{}
	offloadNodeStatusRepo.On("IsEnabled", mock.Anything).Return(true)

	archivedRepo := &mocks.WorkflowArchive{}
	archivedRepo.On("CountWorkflows", mock.Anything, mock.Anything).Return(int64(0), nil)
	archivedRepo.On("ListWorkflows", mock.Anything, mock.Anything).Return(v1alpha1.Workflows{}, nil)
	archivedRepo.On("HasMoreWorkflows", mock.Anything, mock.Anything).Return(false, nil)

	// An empty instance ID keeps the store from requiring an instance-id label on the fixtures.
	instanceIDSvc := instanceid.NewService("")
	wfStore, err := store.NewSQLiteStore(instanceIDSvc)
	require.NoError(t, err)
	for i := range wfs {
		require.NoError(t, wfStore.Add(&wfs[i]))
	}

	server, ctx := newTestServer(t, testServerOpts{
		instanceIDSvc: instanceIDSvc,
		offloadRepo:   offloadNodeStatusRepo,
		archivedRepo:  archivedRepo,
		wfClientset:   v1alpha.NewClientset(),
		wfLister:      wfStore,
		namespace:     "argo",
		claims:        &authtypes.Claims{Claims: jwt.Claims{Subject: "my-sub"}},
	})
	return server, ctx, offloadNodeStatusRepo
}

// TestListWorkflows_PassesOnlyPageKeys asserts that the offload query is scoped to the
// workflows on this page, and that what comes back is attached to the right workflow.
func TestListWorkflows_PassesOnlyPageKeys(t *testing.T) {
	now := time.Now()
	first := offloadedWorkflow("offloaded-first", "uid-first", "v1", now)
	second := offloadedWorkflow("offloaded-second", "uid-second", "v2", now.Add(-time.Minute))
	// Offloaded as well, but a Limit of 2 puts it on the next page.
	nextPage := offloadedWorkflow("offloaded-next-page", "uid-next-page", "v3", now.Add(-2*time.Minute))
	server, ctx, offloadNodeStatusRepo := offloadTestServer(t, first, second, nextPage)

	secondNodes := v1alpha1.Nodes{"n": v1alpha1.NodeStatus{ID: "n", Name: "belongs-to-second"}}
	offloadNodeStatusRepo.On("List", mock.Anything, mock.Anything).
		Return(map[sqldb.UUIDVersion]v1alpha1.Nodes{{UID: "uid-second", Version: "v2"}: secondNodes}, nil)

	list, err := server.ListWorkflows(ctx, &workflowpkg.WorkflowListRequest{
		Namespace:   "argo",
		ListOptions: &metav1.ListOptions{Limit: 2},
	})
	require.NoError(t, err)

	offloadNodeStatusRepo.AssertNumberOfCalls(t, "List", 1)
	call := offloadNodeStatusRepo.Calls[len(offloadNodeStatusRepo.Calls)-1]
	assert.Equal(t, "argo", call.Arguments[0])
	assert.ElementsMatch(t, []sqldb.UUIDVersion{
		{UID: "uid-first", Version: "v1"},
		{UID: "uid-second", Version: "v2"},
	}, call.Arguments[1], "List must be given exactly the offloaded keys on this page")

	// The page is sorted before it is returned, so look the fixtures up by name rather than
	// by position.
	require.Len(t, list.Items, 2)
	nodesByName := map[string]v1alpha1.Nodes{}
	for _, wf := range list.Items {
		nodesByName[wf.Name] = wf.Status.Nodes
	}
	assert.Equal(t, secondNodes, nodesByName["offloaded-second"], "nodes must land on the workflow they were keyed by")
	assert.Empty(t, nodesByName["offloaded-first"], "a workflow with no offloaded row must not pick up another one's nodes")
}

// TestListWorkflows_SkipsQueryWhenPageHasNoOffload asserts that a page with nothing offloaded
// issues no offload query at all. This is the common case for most users.
func TestListWorkflows_SkipsQueryWhenPageHasNoOffload(t *testing.T) {
	server, ctx, offloadNodeStatusRepo := offloadTestServer(t, inlineWorkflow("inline-a", "uid-a"), inlineWorkflow("inline-b", "uid-b"))

	_, err := server.ListWorkflows(ctx, &workflowpkg.WorkflowListRequest{Namespace: "argo"})
	require.NoError(t, err)

	// AssertNotCalled without argument matchers can never fail: it looks for a recorded call
	// matching the (empty) argument list it was given, which no real two-argument call does.
	offloadNodeStatusRepo.AssertNumberOfCalls(t, "List", 0)
}
