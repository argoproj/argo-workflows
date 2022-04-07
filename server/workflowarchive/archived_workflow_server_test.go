package workflowarchive

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	authorizationv1 "k8s.io/api/authorization/v1"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	kubefake "k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"

	"github.com/argoproj/argo-workflows/v3/persist/sqldb/mocks"
	workflowarchivepkg "github.com/argoproj/argo-workflows/v3/pkg/apiclient/workflowarchive"
	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	argofake "github.com/argoproj/argo-workflows/v3/pkg/client/clientset/versioned/fake"
	"github.com/argoproj/argo-workflows/v3/server/auth"
	"github.com/argoproj/argo-workflows/v3/workflow/common"
)

func Test_archivedWorkflowServer(t *testing.T) {
	repo := &mocks.WorkflowArchive{}
	kubeClient := &kubefake.Clientset{}
	wfClient := &argofake.Clientset{}
	w := NewWorkflowArchiveServer(repo)
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
	repo.On("ListWorkflows", "", "", "", time.Time{}, time.Time{}, labels.Requirements(nil), 2, 0).Return(wfv1.Workflows{{}, {}}, nil)
	repo.On("ListWorkflows", "", "", "", time.Time{}, time.Time{}, labels.Requirements(nil), 2, 1).Return(wfv1.Workflows{{}}, nil)
	minStartAt, _ := time.Parse(time.RFC3339, "2020-01-01T00:00:00Z")
	maxStartAt, _ := time.Parse(time.RFC3339, "2020-01-02T00:00:00Z")
	createdTime := metav1.Time{Time: time.Now().UTC()}
	finishedTime := metav1.Time{Time: createdTime.Add(time.Second * 2)}
	repo.On("ListWorkflows", "", "", "", minStartAt, maxStartAt, labels.Requirements(nil), 2, 0).Return(wfv1.Workflows{{}}, nil)
	repo.On("ListWorkflows", "", "my-name", "", minStartAt, maxStartAt, labels.Requirements(nil), 2, 0).Return(wfv1.Workflows{{}}, nil)
	repo.On("ListWorkflows", "", "", "my-", minStartAt, maxStartAt, labels.Requirements(nil), 2, 0).Return(wfv1.Workflows{{}}, nil)
	repo.On("ListWorkflows", "", "my-name", "my-", minStartAt, maxStartAt, labels.Requirements(nil), 2, 0).Return(wfv1.Workflows{{}}, nil)
	repo.On("GetWorkflow", "").Return(nil, nil)
	repo.On("GetWorkflow", "my-uid").Return(&wfv1.Workflow{
		ObjectMeta: metav1.ObjectMeta{Name: "my-name"},
		Spec: wfv1.WorkflowSpec{
			Entrypoint: "my-entrypoint",
			Templates: []wfv1.Template{
				{Name: "my-entrypoint", Container: &apiv1.Container{}},
			},
		},
	}, nil)
	repo.On("GetWorkflow", "failed-uid").Return(&wfv1.Workflow{
		ObjectMeta: metav1.ObjectMeta{
			Name: "failed-wf",
			Labels: map[string]string{
				common.LabelKeyCompleted:               "true",
				common.LabelKeyWorkflowArchivingStatus: "Pending",
			},
		},
		Status: wfv1.WorkflowStatus{
			Phase:      wfv1.WorkflowFailed,
			StartedAt:  createdTime,
			FinishedAt: finishedTime,
			Nodes: map[string]wfv1.NodeStatus{
				"failed-node":    {Name: "failed-node", StartedAt: createdTime, FinishedAt: finishedTime, Phase: wfv1.NodeFailed, Message: "failed"},
				"succeeded-node": {Name: "succeeded-node", StartedAt: createdTime, FinishedAt: finishedTime, Phase: wfv1.NodeSucceeded, Message: "succeeded"}},
		},
	}, nil)
	repo.On("GetWorkflow", "resubmit-uid").Return(&wfv1.Workflow{
		ObjectMeta: metav1.ObjectMeta{Name: "resubmit-wf"},
		Spec: wfv1.WorkflowSpec{
			Entrypoint: "my-entrypoint",
			Templates: []wfv1.Template{
				{Name: "my-entrypoint", Container: &apiv1.Container{Image: "docker/whalesay:latest"}},
			},
		},
	}, nil)
	wfClient.AddReactor("create", "workflows", func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
		return true, &wfv1.Workflow{
			ObjectMeta: metav1.ObjectMeta{Name: "my-name-resubmitted"},
		}, nil
	})
	repo.On("DeleteWorkflow", "my-uid").Return(nil)
	repo.On("ListWorkflowsLabelKeys").Return(&wfv1.LabelKeys{
		Items: []string{"foo", "bar"},
	}, nil)
	repo.On("ListWorkflowsLabelValues", "my-key").Return(&wfv1.LabelValues{
		Items: []string{"my-key=foo", "my-key=bar"},
	}, nil)
	repo.On("RetryWorkflow", "failed-uid").Return(&wfv1.Workflow{
		ObjectMeta: metav1.ObjectMeta{Name: "failed-wf"},
	}, nil)
	repo.On("ResubmitWorkflow", "my-uid").Return(&wfv1.Workflow{
		ObjectMeta: metav1.ObjectMeta{Name: "my-name"},
		Spec: wfv1.WorkflowSpec{
			Entrypoint: "my-entrypoint",
			Templates: []wfv1.Template{
				{Name: "my-entrypoint", Container: &apiv1.Container{}},
			},
		},
	}, nil)

	ctx := context.WithValue(context.WithValue(context.TODO(), auth.WfKey, wfClient), auth.KubeKey, kubeClient)
	t.Run("ListArchivedWorkflows", func(t *testing.T) {
		allowed = false
		_, err := w.ListArchivedWorkflows(ctx, &workflowarchivepkg.ListArchivedWorkflowsRequest{ListOptions: &metav1.ListOptions{Limit: 1}})
		assert.Equal(t, err, status.Error(codes.PermissionDenied, "Permission denied, you are not allowed to list workflows in namespace \"\". Maybe you want to specify a namespace with `listOptions.fieldSelector=metadata.namespace=your-ns`?"))
		allowed = true
		resp, err := w.ListArchivedWorkflows(ctx, &workflowarchivepkg.ListArchivedWorkflowsRequest{ListOptions: &metav1.ListOptions{Limit: 1}})
		if assert.NoError(t, err) {
			assert.Len(t, resp.Items, 1)
			assert.Equal(t, "1", resp.Continue)
		}
		resp, err = w.ListArchivedWorkflows(ctx, &workflowarchivepkg.ListArchivedWorkflowsRequest{ListOptions: &metav1.ListOptions{Continue: "1", Limit: 1}})
		if assert.NoError(t, err) {
			assert.Len(t, resp.Items, 1)
			assert.Empty(t, resp.Continue)
		}
		resp, err = w.ListArchivedWorkflows(ctx, &workflowarchivepkg.ListArchivedWorkflowsRequest{ListOptions: &metav1.ListOptions{FieldSelector: "spec.startedAt>2020-01-01T00:00:00Z,spec.startedAt<2020-01-02T00:00:00Z", Limit: 1}})
		if assert.NoError(t, err) {
			assert.Len(t, resp.Items, 1)
			assert.Empty(t, resp.Continue)
		}
		resp, err = w.ListArchivedWorkflows(ctx, &workflowarchivepkg.ListArchivedWorkflowsRequest{ListOptions: &metav1.ListOptions{FieldSelector: "metadata.name=my-name,spec.startedAt>2020-01-01T00:00:00Z,spec.startedAt<2020-01-02T00:00:00Z", Limit: 1}})
		if assert.NoError(t, err) {
			assert.Len(t, resp.Items, 1)
			assert.Empty(t, resp.Continue)
		}
		resp, err = w.ListArchivedWorkflows(ctx, &workflowarchivepkg.ListArchivedWorkflowsRequest{ListOptions: &metav1.ListOptions{FieldSelector: "spec.startedAt>2020-01-01T00:00:00Z,spec.startedAt<2020-01-02T00:00:00Z", Limit: 1}, NamePrefix: "my-"})
		if assert.NoError(t, err) {
			assert.Len(t, resp.Items, 1)
			assert.Empty(t, resp.Continue)
		}
		resp, err = w.ListArchivedWorkflows(ctx, &workflowarchivepkg.ListArchivedWorkflowsRequest{ListOptions: &metav1.ListOptions{FieldSelector: "metadata.name=my-name,spec.startedAt>2020-01-01T00:00:00Z,spec.startedAt<2020-01-02T00:00:00Z", Limit: 1}, NamePrefix: "my-"})
		if assert.NoError(t, err) {
			assert.Len(t, resp.Items, 1)
			assert.Empty(t, resp.Continue)
		}
	})
	t.Run("GetArchivedWorkflow", func(t *testing.T) {
		allowed = false
		_, err := w.GetArchivedWorkflow(ctx, &workflowarchivepkg.GetArchivedWorkflowRequest{Uid: "my-uid"})
		assert.Equal(t, err, status.Error(codes.PermissionDenied, "permission denied"))
		allowed = true
		_, err = w.GetArchivedWorkflow(ctx, &workflowarchivepkg.GetArchivedWorkflowRequest{})
		assert.Equal(t, err, status.Error(codes.NotFound, "not found"))
		wf, err := w.GetArchivedWorkflow(ctx, &workflowarchivepkg.GetArchivedWorkflowRequest{Uid: "my-uid"})
		assert.NoError(t, err)
		assert.NotNil(t, wf)
	})
	t.Run("DeleteArchivedWorkflow", func(t *testing.T) {
		allowed = false
		_, err := w.DeleteArchivedWorkflow(ctx, &workflowarchivepkg.DeleteArchivedWorkflowRequest{Uid: "my-uid"})
		assert.Equal(t, err, status.Error(codes.PermissionDenied, "permission denied"))
		allowed = true
		_, err = w.DeleteArchivedWorkflow(ctx, &workflowarchivepkg.DeleteArchivedWorkflowRequest{Uid: "my-uid"})
		assert.NoError(t, err)
	})
	t.Run("ListArchivedWorkflowLabelKeys", func(t *testing.T) {
		resp, err := w.ListArchivedWorkflowLabelKeys(ctx, &workflowarchivepkg.ListArchivedWorkflowLabelKeysRequest{})
		assert.NoError(t, err)
		assert.Len(t, resp.Items, 2)
	})
	t.Run("ListArchivedWorkflowLabelValues", func(t *testing.T) {
		resp, err := w.ListArchivedWorkflowLabelValues(ctx, &workflowarchivepkg.ListArchivedWorkflowLabelValuesRequest{ListOptions: &metav1.ListOptions{LabelSelector: "my-key"}})
		assert.NoError(t, err)
		assert.Len(t, resp.Items, 2)
	})
	t.Run("RetryArchivedWorkflow", func(t *testing.T) {
		_, err := w.RetryArchivedWorkflow(ctx, &workflowarchivepkg.RetryArchivedWorkflowRequest{Uid: "failed-uid"})
		assert.Equal(t, err, status.Error(codes.AlreadyExists, "Workflow already exists on cluster, use argo retry {name} instead"))
	})
	t.Run("ResubmitArchivedWorkflow", func(t *testing.T) {
		wf, err := w.ResubmitArchivedWorkflow(ctx, &workflowarchivepkg.ResubmitArchivedWorkflowRequest{Uid: "resubmit-uid", Memoized: false})
		assert.NoError(t, err)
		assert.NotNil(t, wf)
	})
}
