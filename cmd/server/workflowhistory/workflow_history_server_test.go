package workflowhistory

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	authorizationv1 "k8s.io/api/authorization/v1"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	kubefake "k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"

	"github.com/argoproj/argo/persist/sqldb/mocks"
	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	argofake "github.com/argoproj/argo/pkg/client/clientset/versioned/fake"
)



func Test_workflowHistoryServer(t *testing.T) {
	repo := &mocks.WorkflowHistoryRepository{}
	kubeClient := &kubefake.Clientset{}
	wfClient := &argofake.Clientset{}
	w := NewWorkflowHistoryServer(wfClient, kubeClient, "", false, repo)
	allowed := true
	kubeClient.AddReactor("create", "selfsubjectaccessreviews", func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
		return true, &authorizationv1.SelfSubjectAccessReview{
			Status: authorizationv1.SubjectAccessReviewStatus{Allowed: allowed},
		}, nil
	})
	// two pages of results for limit 1
	repo.On("ListWorkflowHistory", "", 1, 0).Return([]wfv1.Workflow{{}}, nil)
	repo.On("ListWorkflowHistory", "", 1, 1).Return([]wfv1.Workflow{}, nil)
	repo.On("GetWorkflowHistory", "", "").Return(nil, nil)
	repo.On("GetWorkflowHistory", "my-ns", "my-uid").Return(&wfv1.Workflow{
		ObjectMeta: metav1.ObjectMeta{Name: "my-name"},
		Spec: wfv1.WorkflowSpec{
			Entrypoint: "my-entrypoint",
			Templates: []wfv1.Template{
				{Name: "my-entrypoint", Container: &apiv1.Container{}},
			},
		},
	}, nil)
	wfClient.AddReactor("create", "workflows", func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
		return true, &wfv1.Workflow{
			ObjectMeta: metav1.ObjectMeta{Name: "my-name-resubmitted"},
		}, nil
	})
	repo.On("DeleteWorkflowHistory", "my-ns", "my-uid").Return( nil)

	t.Run("ListWorkflowHistory", func(t *testing.T) {
		allowed = false
		history, err := w.ListWorkflowHistory(context.TODO(), &WorkflowHistoryListRequest{ListOptions: &metav1.ListOptions{Limit: 1}})
		if assert.NoError(t, err) {
			assert.Len(t, history.Items, 0)
		}
		allowed = true
		history, err = w.ListWorkflowHistory(context.TODO(), &WorkflowHistoryListRequest{ListOptions: &metav1.ListOptions{Limit: 1}})
		if assert.NoError(t, err) {
			assert.Len(t, history.Items, 1)
			assert.Equal(t, "1", history.Continue)
		}
		history, err = w.ListWorkflowHistory(context.TODO(), &WorkflowHistoryListRequest{ListOptions: &metav1.ListOptions{Continue: "1", Limit: 1}})
		if assert.NoError(t, err) {
			assert.Len(t, history.Items, 0)
			assert.Empty(t, history.Continue)
		}
	})
	t.Run("GetWorkflowHistory", func(t *testing.T) {
		allowed = false
		_, err := w.GetWorkflowHistory(context.TODO(), &WorkflowHistoryGetRequest{Namespace: "my-ns", Uid: "my-uid"})
		assert.Equal(t, err, status.Error(codes.PermissionDenied, "permission denied"))
		allowed = true
		_, err = w.GetWorkflowHistory(context.TODO(), &WorkflowHistoryGetRequest{})
		assert.Equal(t, err, status.Error(codes.NotFound, "not found"))
		wf, err := w.GetWorkflowHistory(context.TODO(), &WorkflowHistoryGetRequest{Namespace: "my-ns", Uid: "my-uid"})
		assert.NoError(t, err)
		assert.NotNil(t, wf)
	})
	t.Run("ResubmitWorkflowHistory", func(t *testing.T) {
		allowed = false
		wf, err := w.ResubmitWorkflowHistory(context.TODO(), &WorkflowHistoryUpdateRequest{Namespace: "my-ns", Uid: "my-uid"})
		assert.Equal(t, err, status.Error(codes.PermissionDenied, "permission denied"))
		allowed = true
		wf, err = w.ResubmitWorkflowHistory(context.TODO(), &WorkflowHistoryUpdateRequest{Namespace: "my-ns", Uid: "my-uid"})
		assert.NoError(t, err)
		assert.Equal(t, "my-name-resubmitted", wf.Name)
	})
	t.Run("DeleteWorkflowHistory", func(t *testing.T) {
		allowed = false
		_, err := w.DeleteWorkflowHistory(context.TODO(), &WorkflowHistoryDeleteRequest{Namespace: "my-ns", Uid: "my-uid"})
		assert.Equal(t, err, status.Error(codes.PermissionDenied, "permission denied"))
		allowed = true
		_, err = w.DeleteWorkflowHistory(context.TODO(), &WorkflowHistoryDeleteRequest{Namespace: "my-ns", Uid: "my-uid"})
		assert.NoError(t, err)
	})
}
