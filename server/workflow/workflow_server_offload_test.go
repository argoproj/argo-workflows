package workflow

import (
	"context"
	"testing"

	"github.com/go-jose/go-jose/v4/jwt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	authorizationv1 "k8s.io/api/authorization/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/fake"
	ktesting "k8s.io/client-go/testing"

	"github.com/argoproj/argo-workflows/v4/persist/sqldb"
	"github.com/argoproj/argo-workflows/v4/persist/sqldb/mocks"
	workflowpkg "github.com/argoproj/argo-workflows/v4/pkg/apiclient/workflow"
	"github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	v1alpha "github.com/argoproj/argo-workflows/v4/pkg/client/clientset/versioned/fake"
	"github.com/argoproj/argo-workflows/v4/server/auth"
	authtypes "github.com/argoproj/argo-workflows/v4/server/auth/types"
	"github.com/argoproj/argo-workflows/v4/server/clusterworkflowtemplate"
	"github.com/argoproj/argo-workflows/v4/server/workflow/store"
	"github.com/argoproj/argo-workflows/v4/server/workflowtemplate"
	"github.com/argoproj/argo-workflows/v4/util/instanceid"
	"github.com/argoproj/argo-workflows/v4/util/logging"
)

// offloadedWorkflow builds a workflow whose node status lives in the offload table.
func offloadedWorkflow(name, uid, version string) v1alpha1.Workflow {
	return v1alpha1.Workflow{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: "argo", UID: types.UID(uid)},
		Status:     v1alpha1.WorkflowStatus{OffloadNodeStatusVersion: version},
	}
}

// offloadTestServer builds the smallest server that can serve ListWorkflows, and hands back
// the offload mock so tests can assert on how it was called. It deliberately does not reuse
// getWorkflowServer, which is shared by many other tests and does not expose the mock.
func offloadTestServer(t *testing.T, wfs ...v1alpha1.Workflow) (Server, context.Context, *mocks.OffloadNodeStatusRepo) {
	t.Helper()

	offloadNodeStatusRepo := &mocks.OffloadNodeStatusRepo{}
	offloadNodeStatusRepo.On("IsEnabled", mock.Anything).Return(true)

	archivedRepo := &mocks.WorkflowArchive{}
	archivedRepo.On("CountWorkflows", mock.Anything, mock.Anything).Return(int64(0), nil)
	archivedRepo.On("ListWorkflows", mock.Anything, mock.Anything).Return(v1alpha1.Workflows{}, nil)
	archivedRepo.On("HasMoreWorkflows", mock.Anything, mock.Anything).Return(false, nil)

	kubeClientSet := fake.NewClientset()
	kubeClientSet.PrependReactor("create", "selfsubjectaccessreviews", func(action ktesting.Action) (handled bool, ret runtime.Object, err error) {
		return true, &authorizationv1.SelfSubjectAccessReview{
			Status: authorizationv1.SubjectAccessReviewStatus{Allowed: true},
		}, nil
	})

	wfClientset := v1alpha.NewClientset()
	ctx := logging.TestContext(t.Context())
	ctx = context.WithValue(context.WithValue(context.WithValue(ctx, auth.WfKey, wfClientset), auth.KubeKey, kubeClientSet), auth.ClaimsKey, &authtypes.Claims{Claims: jwt.Claims{Subject: "my-sub"}})

	// An empty instance ID keeps the store from requiring an instance-id label on the fixtures.
	instanceIDSvc := instanceid.NewService("")
	wfStore, err := store.NewSQLiteStore(instanceIDSvc)
	require.NoError(t, err)
	for i := range wfs {
		require.NoError(t, wfStore.Add(&wfs[i]))
	}

	namespace := "argo"
	server := NewServer(ctx, instanceIDSvc, offloadNodeStatusRepo, archivedRepo, wfClientset, wfStore, nil, workflowtemplate.NewClientStore(), clusterworkflowtemplate.NewClientStore(), nil, &namespace, nil)
	return server, ctx, offloadNodeStatusRepo
}

// TestListWorkflows_PassesOnlyPageKeys asserts that the offload query is scoped to the
// workflows on this page, rather than to the whole namespace.
func TestListWorkflows_PassesOnlyPageKeys(t *testing.T) {
	wantedA := offloadedWorkflow("offloaded-a", "uid-a", "v1")
	wantedB := offloadedWorkflow("offloaded-b", "uid-b", "v2")
	server, ctx, offloadNodeStatusRepo := offloadTestServer(t, wantedA, wantedB, offloadedWorkflow("inline", "uid-c", ""))

	offloadNodeStatusRepo.On("List", mock.Anything, mock.Anything).Return(map[sqldb.UUIDVersion]v1alpha1.Nodes{}, nil)

	_, err := server.ListWorkflows(ctx, &workflowpkg.WorkflowListRequest{Namespace: "argo"})
	require.NoError(t, err)

	offloadNodeStatusRepo.AssertNumberOfCalls(t, "List", 1)
	call := offloadNodeStatusRepo.Calls[len(offloadNodeStatusRepo.Calls)-1]
	require.Equal(t, "List", call.Method)
	assert.Equal(t, "argo", call.Arguments[0])
	assert.ElementsMatch(t, []sqldb.UUIDVersion{
		{UID: "uid-a", Version: "v1"},
		{UID: "uid-b", Version: "v2"},
	}, call.Arguments[1], "List must be given exactly the offloaded keys on this page")
}

// TestListWorkflows_SkipsQueryWhenPageHasNoOffload asserts that a page with nothing offloaded
// issues no offload query at all. This is the common case for most users.
func TestListWorkflows_SkipsQueryWhenPageHasNoOffload(t *testing.T) {
	server, ctx, offloadNodeStatusRepo := offloadTestServer(t, offloadedWorkflow("inline-a", "uid-a", ""), offloadedWorkflow("inline-b", "uid-b", ""))

	_, err := server.ListWorkflows(ctx, &workflowpkg.WorkflowListRequest{Namespace: "argo"})
	require.NoError(t, err)

	offloadNodeStatusRepo.AssertNotCalled(t, "List")
}
