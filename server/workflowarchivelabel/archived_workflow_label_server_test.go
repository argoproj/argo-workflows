package workflowarchivelabel

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"gopkg.in/square/go-jose.v2/jwt"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	k8stesting "k8s.io/client-go/testing"

	"github.com/argoproj/argo-workflows/v3/persist/sqldb/mocks"
	workflowarchivelabelpkg "github.com/argoproj/argo-workflows/v3/pkg/apiclient/workflowarchivelabel"
	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	argofake "github.com/argoproj/argo-workflows/v3/pkg/client/clientset/versioned/fake"
	"github.com/argoproj/argo-workflows/v3/server/auth"
	"github.com/argoproj/argo-workflows/v3/server/auth/types"
)

func Test_archivedWorkflowLabelServer(t *testing.T) {
	repo := &mocks.WorkflowArchive{}
	wfClient := &argofake.Clientset{}
	w := NewWorkflowArchiveLabelServer(repo)
	repo.On("ListWorkflowsLabelKey").Return(&wfv1.LabelKeys{
		Items: []string{"foo", "bar"},
	}, nil)
	repo.On("GetWorkflowLabel", "my-key").Return(&wfv1.Labels{
		Items: []string{"my-key=foo", "my-key=bar"},
	}, nil)
	wfClient.AddReactor("create", "workflows", func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
		return true, &wfv1.Workflow{
			ObjectMeta: metav1.ObjectMeta{Name: "my-name-resubmitted"},
		}, nil
	})

	ctx := context.WithValue(context.WithValue(context.TODO(), auth.WfKey, wfClient), auth.KubeKey, &types.Claims{Claims: jwt.Claims{Subject: "my-sub"}})
	t.Run("ListArchivedWorkflowLabel", func(t *testing.T) {
		resp, err := w.ListArchivedWorkflowLabel(ctx, &workflowarchivelabelpkg.ListArchivedWorkflowLabelRequest{})
		assert.NoError(t, err)
		assert.Len(t, resp.Items, 2)
	})
	t.Run("GetArchivedWorkflowLabel", func(t *testing.T) {
		resp, err := w.GetArchivedWorkflowLabel(ctx, &workflowarchivelabelpkg.GetArchivedWorkflowLabelRequest{ListOptions: &metav1.ListOptions{FieldSelector: "key=my-key"}})
		assert.NoError(t, err)
		assert.Len(t, resp.Items, 2)
	})
}
