package workflowarchive

import (
	"context"
	"testing"
	"time"

	"github.com/argoproj/argo-workflows/v4/util/logging"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	authorizationv1 "k8s.io/api/authorization/v1"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	kubefake "k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"

	"github.com/argoproj/argo-workflows/v4/persist/sqldb"
	"github.com/argoproj/argo-workflows/v4/persist/sqldb/mocks"
	workflowarchivepkg "github.com/argoproj/argo-workflows/v4/pkg/apiclient/workflowarchive"
	"github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	argofake "github.com/argoproj/argo-workflows/v4/pkg/client/clientset/versioned/fake"
	"github.com/argoproj/argo-workflows/v4/server/auth"
	sutils "github.com/argoproj/argo-workflows/v4/server/utils"
	"github.com/argoproj/argo-workflows/v4/workflow/common"
)

func Test_archivedWorkflowServer(t *testing.T) {
	repo := &mocks.WorkflowArchive{}
	kubeClient := &kubefake.Clientset{}
	wfClient := &argofake.Clientset{}
	offloadNodeStatusRepo := &mocks.OffloadNodeStatusRepo{}
	offloadNodeStatusRepo.On("IsEnabled", mock.Anything).Return(true)
	offloadNodeStatusRepo.On("List", mock.Anything).Return(map[sqldb.UUIDVersion]v1alpha1.Nodes{}, nil)
	w := NewWorkflowArchiveServer(repo, offloadNodeStatusRepo, nil)
	allowed := true
	kubeClient.AddReactor("create", "selfsubjectaccessreviews", func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
		return true, &authorizationv1.SelfSubjectAccessReview{
			Status: authorizationv1.SubjectAccessReviewStatus{Allowed: allowed},
		}, nil
	})
	kubeClient.AddReactor("create", "selfsubjectrulesreviews", func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
		var rules []authorizationv1.ResourceRule
		if allowed {
			rules = append(rules, authorizationv1.ResourceRule{})
		}
		return true, &authorizationv1.SelfSubjectRulesReview{
			Status: authorizationv1.SubjectRulesReviewStatus{
				ResourceRules: rules,
			},
		}, nil
	})
	// two pages of results for limit 1
	repo.On("ListWorkflows", mock.Anything, sutils.ListOptions{Limit: 2, Offset: 0}).Return(v1alpha1.Workflows{{}, {}}, nil)
	repo.On("ListWorkflows", mock.Anything, sutils.ListOptions{Limit: 2, Offset: 1}).Return(v1alpha1.Workflows{{}}, nil)
	minStartAt, _ := time.Parse(time.RFC3339, "2020-01-01T00:00:00Z")
	maxStartAt, _ := time.Parse(time.RFC3339, "2020-01-02T00:00:00Z")
	createdTime := metav1.Time{Time: time.Now().UTC()}
	finishedTime := metav1.Time{Time: createdTime.Add(time.Second * 2)}
	repo.On("ListWorkflows", mock.Anything, sutils.ListOptions{Namespace: "", Name: "", NamePrefix: "", MinStartedAt: minStartAt, MaxStartedAt: maxStartAt, Limit: 2, Offset: 0}).Return(v1alpha1.Workflows{{}}, nil)
	repo.On("ListWorkflows", mock.Anything, sutils.ListOptions{Namespace: "", Name: "my-name", NamePrefix: "", MinStartedAt: minStartAt, MaxStartedAt: maxStartAt, Limit: 2, Offset: 0}).Return(v1alpha1.Workflows{{}}, nil)
	repo.On("ListWorkflows", mock.Anything, sutils.ListOptions{Namespace: "", Name: "", NamePrefix: "my-", MinStartedAt: minStartAt, MaxStartedAt: maxStartAt, Limit: 2, Offset: 0}).Return(v1alpha1.Workflows{{}}, nil)
	repo.On("ListWorkflows", mock.Anything, sutils.ListOptions{Namespace: "", Name: "my-name", NamePrefix: "my-", MinStartedAt: minStartAt, MaxStartedAt: maxStartAt, Limit: 2, Offset: 0}).Return(v1alpha1.Workflows{{}}, nil)
	repo.On("ListWorkflows", mock.Anything, sutils.ListOptions{Namespace: "", Name: "my-name", NamePrefix: "my-", MinStartedAt: minStartAt, MaxStartedAt: maxStartAt, Limit: 2, Offset: 0, ShowRemainingItemCount: true}).Return(v1alpha1.Workflows{{}}, nil)
	repo.On("ListWorkflows", mock.Anything, sutils.ListOptions{Namespace: "", Name: "excluded-name", NameFilter: "NotEquals", MinStartedAt: minStartAt, MaxStartedAt: maxStartAt, Limit: 2, Offset: 0}).Return(v1alpha1.Workflows{{}}, nil)
	repo.On("ListWorkflows", mock.Anything, sutils.ListOptions{Namespace: "", Name: "exact-name", NameFilter: "", MinStartedAt: minStartAt, MaxStartedAt: maxStartAt, Limit: 2, Offset: 0}).Return(v1alpha1.Workflows{{}}, nil)
	repo.On("ListWorkflows", mock.Anything, sutils.ListOptions{Namespace: "user-ns", Name: "", NamePrefix: "", MinStartedAt: time.Time{}, MaxStartedAt: time.Time{}, Limit: 2, Offset: 0}).Return(v1alpha1.Workflows{{}, {}}, nil)
	repo.On("CountWorkflows", mock.Anything, sutils.ListOptions{Namespace: "", Name: "my-name", NamePrefix: "my-", MinStartedAt: minStartAt, MaxStartedAt: maxStartAt, Limit: 2, Offset: 0}).Return(int64(5), nil)
	repo.On("CountWorkflows", mock.Anything, sutils.ListOptions{Namespace: "", Name: "my-name", NamePrefix: "my-", MinStartedAt: minStartAt, MaxStartedAt: maxStartAt, Limit: 2, Offset: 0, ShowRemainingItemCount: true}).Return(int64(5), nil)
	repo.On("GetWorkflow", mock.Anything, "", "", "").Return(nil, nil)
	repo.On("GetWorkflow", mock.Anything, "my-uid", "", "").Return(&v1alpha1.Workflow{
		ObjectMeta: metav1.ObjectMeta{Name: "my-name"},
		Spec: v1alpha1.WorkflowSpec{
			Entrypoint: "my-entrypoint",
			Templates: []v1alpha1.Template{
				{Name: "my-entrypoint", Container: &apiv1.Container{}},
			},
		},
	}, nil)
	repo.On("GetWorkflow", mock.Anything, "failed-uid", "", "").Return(&v1alpha1.Workflow{
		ObjectMeta: metav1.ObjectMeta{
			Name: "failed-wf",
			Labels: map[string]string{
				common.LabelKeyCompleted:               "true",
				common.LabelKeyWorkflowArchivingStatus: "Pending",
			},
		},
		Status: v1alpha1.WorkflowStatus{
			Phase:      v1alpha1.WorkflowFailed,
			StartedAt:  createdTime,
			FinishedAt: finishedTime,
			Nodes: map[string]v1alpha1.NodeStatus{
				"failed-node":    {Name: "failed-node", StartedAt: createdTime, FinishedAt: finishedTime, Phase: v1alpha1.NodeFailed, Message: "failed"},
				"succeeded-node": {Name: "succeeded-node", StartedAt: createdTime, FinishedAt: finishedTime, Phase: v1alpha1.NodeSucceeded, Message: "succeeded"}},
		},
	}, nil)
	repo.On("GetWorkflow", mock.Anything, "resubmit-uid", "", "").Return(&v1alpha1.Workflow{
		ObjectMeta: metav1.ObjectMeta{Name: "resubmit-wf"},
		Spec: v1alpha1.WorkflowSpec{
			Entrypoint: "my-entrypoint",
			Templates: []v1alpha1.Template{
				{Name: "my-entrypoint", Container: &apiv1.Container{Image: "docker/whalesay:latest"}},
			},
		},
	}, nil)
	wfClient.AddReactor("create", "workflows", func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
		return true, &v1alpha1.Workflow{
			ObjectMeta: metav1.ObjectMeta{Name: "my-name-resubmitted"},
		}, nil
	})
	repo.On("DeleteWorkflow", mock.Anything, "my-uid").Return(nil)
	repo.On("ListWorkflowsLabelKeys", mock.Anything).Return(&v1alpha1.LabelKeys{
		Items: []string{"foo", "bar"},
	}, nil)
	repo.On("ListWorkflowsLabelValues", mock.Anything, "my-key").Return(&v1alpha1.LabelValues{
		Items: []string{"my-key=foo", "my-key=bar"},
	}, nil)
	repo.On("RetryWorkflow", mock.Anything, "failed-uid").Return(&v1alpha1.Workflow{
		ObjectMeta: metav1.ObjectMeta{Name: "failed-wf"},
	}, nil)
	repo.On("ResubmitWorkflow", mock.Anything, "my-uid").Return(&v1alpha1.Workflow{
		ObjectMeta: metav1.ObjectMeta{Name: "my-name"},
		Spec: v1alpha1.WorkflowSpec{
			Entrypoint: "my-entrypoint",
			Templates: []v1alpha1.Template{
				{Name: "my-entrypoint", Container: &apiv1.Container{}},
			},
		},
	}, nil)

	ctx := context.WithValue(context.WithValue(logging.TestContext(t.Context()), auth.WfKey, wfClient), auth.KubeKey, kubeClient)
	t.Run("ListArchivedWorkflows", func(t *testing.T) {
		allowed = false
		_, err := w.ListArchivedWorkflows(ctx, &workflowarchivepkg.ListArchivedWorkflowsRequest{ListOptions: &metav1.ListOptions{Limit: 1}})
		assert.Equal(t, err, status.Error(codes.PermissionDenied, "Permission denied, you are not allowed to list workflows in namespace \"\". Maybe you want to specify a namespace with query parameter `.namespace=`?"))
		allowed = true
		resp, err := w.ListArchivedWorkflows(ctx, &workflowarchivepkg.ListArchivedWorkflowsRequest{ListOptions: &metav1.ListOptions{Limit: 1}})
		require.NoError(t, err)
		assert.Len(t, resp.Items, 1)
		assert.Equal(t, "1", resp.Continue)
		resp, err = w.ListArchivedWorkflows(ctx, &workflowarchivepkg.ListArchivedWorkflowsRequest{ListOptions: &metav1.ListOptions{Continue: "1", Limit: 1}})
		require.NoError(t, err)
		assert.Len(t, resp.Items, 1)
		assert.Empty(t, resp.Continue)
		resp, err = w.ListArchivedWorkflows(ctx, &workflowarchivepkg.ListArchivedWorkflowsRequest{ListOptions: &metav1.ListOptions{FieldSelector: "spec.startedAt>2020-01-01T00:00:00Z,spec.startedAt<2020-01-02T00:00:00Z", Limit: 1}})
		require.NoError(t, err)
		assert.Len(t, resp.Items, 1)
		assert.Empty(t, resp.Continue)
		resp, err = w.ListArchivedWorkflows(ctx, &workflowarchivepkg.ListArchivedWorkflowsRequest{ListOptions: &metav1.ListOptions{FieldSelector: "metadata.name=my-name,spec.startedAt>2020-01-01T00:00:00Z,spec.startedAt<2020-01-02T00:00:00Z", Limit: 1}})
		require.NoError(t, err)
		assert.Len(t, resp.Items, 1)
		assert.Empty(t, resp.Continue)
		resp, err = w.ListArchivedWorkflows(ctx, &workflowarchivepkg.ListArchivedWorkflowsRequest{ListOptions: &metav1.ListOptions{FieldSelector: "spec.startedAt>2020-01-01T00:00:00Z,spec.startedAt<2020-01-02T00:00:00Z", Limit: 1}, NamePrefix: "my-"})
		require.NoError(t, err)
		assert.Len(t, resp.Items, 1)
		assert.Empty(t, resp.Continue)
		resp, err = w.ListArchivedWorkflows(ctx, &workflowarchivepkg.ListArchivedWorkflowsRequest{ListOptions: &metav1.ListOptions{FieldSelector: "metadata.name=my-name,spec.startedAt>2020-01-01T00:00:00Z,spec.startedAt<2020-01-02T00:00:00Z", Limit: 1}, NamePrefix: "my-"})
		require.NoError(t, err)
		assert.Len(t, resp.Items, 1)
		assert.Empty(t, resp.Continue)
		resp, err = w.ListArchivedWorkflows(ctx, &workflowarchivepkg.ListArchivedWorkflowsRequest{ListOptions: &metav1.ListOptions{FieldSelector: "metadata.name=my-name,spec.startedAt>2020-01-01T00:00:00Z,spec.startedAt<2020-01-02T00:00:00Z,ext.showRemainingItemCount=true", Limit: 1}, NamePrefix: "my-"})
		require.NoError(t, err)
		assert.Len(t, resp.Items, 1)
		assert.Equal(t, int64(4), *resp.RemainingItemCount)
		assert.Empty(t, resp.Continue)
		resp, err = w.ListArchivedWorkflows(ctx, &workflowarchivepkg.ListArchivedWorkflowsRequest{ListOptions: &metav1.ListOptions{FieldSelector: "metadata.name!=excluded-name,spec.startedAt>2020-01-01T00:00:00Z,spec.startedAt<2020-01-02T00:00:00Z", Limit: 1}})
		require.NoError(t, err)
		assert.Len(t, resp.Items, 1)
		assert.Empty(t, resp.Continue)
		resp, err = w.ListArchivedWorkflows(ctx, &workflowarchivepkg.ListArchivedWorkflowsRequest{ListOptions: &metav1.ListOptions{FieldSelector: "metadata.name==exact-name,spec.startedAt>2020-01-01T00:00:00Z,spec.startedAt<2020-01-02T00:00:00Z", Limit: 1}})
		require.NoError(t, err)
		assert.Len(t, resp.Items, 1)
		assert.Empty(t, resp.Continue)
		/////// Currently, for the purpose of backward compatibility, namespace is supported both as its own query parameter and as part of the field selector
		/////// need to test both
		// pass namespace as its own query parameter
		resp, err = w.ListArchivedWorkflows(ctx, &workflowarchivepkg.ListArchivedWorkflowsRequest{Namespace: "user-ns", ListOptions: &metav1.ListOptions{Limit: 1}})
		require.NoError(t, err)
		assert.Len(t, resp.Items, 1)
		assert.Equal(t, "1", resp.Continue)
		// pass namespace as field selector and not query parameter
		resp, err = w.ListArchivedWorkflows(ctx, &workflowarchivepkg.ListArchivedWorkflowsRequest{ListOptions: &metav1.ListOptions{Limit: 1, FieldSelector: "metadata.namespace=user-ns"}})
		require.NoError(t, err)
		assert.Len(t, resp.Items, 1)
		assert.Equal(t, "1", resp.Continue)

		// pass namespace as field selector and query parameter both, where both match
		resp, err = w.ListArchivedWorkflows(ctx, &workflowarchivepkg.ListArchivedWorkflowsRequest{Namespace: "user-ns", ListOptions: &metav1.ListOptions{Limit: 1, FieldSelector: "metadata.namespace=user-ns"}})
		require.NoError(t, err)
		assert.Len(t, resp.Items, 1)
		assert.Equal(t, "1", resp.Continue)

		// pass namespace as field selector and query parameter both, where they don't match
		_, err = w.ListArchivedWorkflows(ctx, &workflowarchivepkg.ListArchivedWorkflowsRequest{Namespace: "user-ns", ListOptions: &metav1.ListOptions{Limit: 1, FieldSelector: "metadata.namespace=other-ns"}})
		assert.Equal(t, err, status.Error(codes.InvalidArgument, "'namespace' query param (\"user-ns\") and fieldselector 'metadata.namespace' (\"other-ns\") are both specified and contradict each other"))

	})
	t.Run("GetArchivedWorkflow", func(t *testing.T) {
		allowed = false
		_, err := w.GetArchivedWorkflow(ctx, &workflowarchivepkg.GetArchivedWorkflowRequest{Uid: "my-uid"})
		assert.Equal(t, err, status.Error(codes.PermissionDenied, "permission denied"))
		allowed = true
		_, err = w.GetArchivedWorkflow(ctx, &workflowarchivepkg.GetArchivedWorkflowRequest{})
		assert.Equal(t, err, status.Error(codes.NotFound, "not found"))
		wf, err := w.GetArchivedWorkflow(ctx, &workflowarchivepkg.GetArchivedWorkflowRequest{Uid: "my-uid"})
		require.NoError(t, err)
		assert.NotNil(t, wf)

		repo.On("GetWorkflow", mock.Anything, "", "my-ns", "my-name").Return(&v1alpha1.Workflow{
			ObjectMeta: metav1.ObjectMeta{Name: "my-name", Namespace: "my-ns"},
			Spec: v1alpha1.WorkflowSpec{
				Entrypoint: "my-entrypoint",
				Templates: []v1alpha1.Template{
					{Name: "my-entrypoint", Container: &apiv1.Container{}},
				},
			},
		}, nil)
		wf, err = w.GetArchivedWorkflow(ctx, &workflowarchivepkg.GetArchivedWorkflowRequest{Uid: "", Name: "my-name", Namespace: "my-ns"})
		require.NoError(t, err)
		assert.NotNil(t, wf)
	})
	t.Run("DeleteArchivedWorkflow", func(t *testing.T) {
		allowed = false
		_, err := w.DeleteArchivedWorkflow(ctx, &workflowarchivepkg.DeleteArchivedWorkflowRequest{Uid: "my-uid"})
		assert.Equal(t, err, status.Error(codes.PermissionDenied, "permission denied"))
		allowed = true
		_, err = w.DeleteArchivedWorkflow(ctx, &workflowarchivepkg.DeleteArchivedWorkflowRequest{Uid: "my-uid"})
		require.NoError(t, err)
	})
	t.Run("ListArchivedWorkflowLabelKeys", func(t *testing.T) {
		resp, err := w.ListArchivedWorkflowLabelKeys(ctx, &workflowarchivepkg.ListArchivedWorkflowLabelKeysRequest{})
		require.NoError(t, err)
		assert.Len(t, resp.Items, 2)
	})
	t.Run("ListArchivedWorkflowLabelValues", func(t *testing.T) {
		resp, err := w.ListArchivedWorkflowLabelValues(ctx, &workflowarchivepkg.ListArchivedWorkflowLabelValuesRequest{ListOptions: &metav1.ListOptions{LabelSelector: "my-key"}})
		require.NoError(t, err)
		assert.Len(t, resp.Items, 2)

		assert.False(t, matchLabelKeyPattern("my-key"))
		t.Setenv(disableValueListRetrievalKeyPattern, "my-key")
		assert.True(t, matchLabelKeyPattern("my-key"))
		assert.False(t, matchLabelKeyPattern("wrong key"))
		resp, err = w.ListArchivedWorkflowLabelValues(ctx, &workflowarchivepkg.ListArchivedWorkflowLabelValuesRequest{ListOptions: &metav1.ListOptions{LabelSelector: "my-key"}})
		require.NoError(t, err)
		assert.Empty(t, resp.Items)
	})
	t.Run("RetryArchivedWorkflow", func(t *testing.T) {
		_, err := w.RetryArchivedWorkflow(ctx, &workflowarchivepkg.RetryArchivedWorkflowRequest{Uid: "failed-uid"})
		assert.Equal(t, err, status.Error(codes.AlreadyExists, "Workflow already exists on cluster, use argo retry {name} instead"))
	})
	t.Run("ResubmitArchivedWorkflow", func(t *testing.T) {
		wf, err := w.ResubmitArchivedWorkflow(ctx, &workflowarchivepkg.ResubmitArchivedWorkflowRequest{Uid: "resubmit-uid", Memoized: false})
		require.NoError(t, err)
		assert.NotNil(t, wf)
	})
}
