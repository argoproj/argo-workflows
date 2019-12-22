package workflowhistory

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	authorizationv1 "k8s.io/api/authorization/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"

	"github.com/argoproj/argo/persist/sqldb/mocks"
	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
)

func Test_workflowHistoryServer_ListWorkflowHistory(t *testing.T) {
	repo := mocks.WorkflowHistoryRepository{}
	fakeKubeClient := &fake.Clientset{}
	w := workflowHistoryServer{repo: &repo, kubeClientset: fakeKubeClient}

	fakeKubeClient.AddReactor("create", "selfsubjectaccessreviews", func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
		return true, &authorizationv1.SelfSubjectAccessReview{
			Status: authorizationv1.SubjectAccessReviewStatus{Allowed: true},
		}, nil
	})

	// two pages of results for limit 1
	repo.On("ListWorkflowHistory", 1, 0).Return([]wfv1.Workflow{{}}, nil)
	repo.On("ListWorkflowHistory", 1, 1).Return([]wfv1.Workflow{}, nil)

	history, err := w.ListWorkflowHistory(context.TODO(), &WorkflowHistoryListRequest{ListOptions: &metav1.ListOptions{Limit: 1}})
	if assert.NoError(t, err) {
		assert.Len(t, history.Items, 1)
		assert.Equal(t, "1", history.Continue)
	}
	history, err = w.ListWorkflowHistory(context.TODO(), &WorkflowHistoryListRequest{ListOptions: &metav1.ListOptions{Continue: "1", Limit: 1}})
	if assert.NoError(t, err) {
		assert.Len(t, history.Items, 0)
		assert.Empty(t, history.Continue)
	}
}
