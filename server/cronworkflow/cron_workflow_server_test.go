package cronworkflow

import (
	"context"
	"testing"

	"github.com/go-jose/go-jose/v3/jwt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	cronworkflowpkg "github.com/argoproj/argo-workflows/v3/pkg/apiclient/cronworkflow"
	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	wftFake "github.com/argoproj/argo-workflows/v3/pkg/client/clientset/versioned/fake"
	"github.com/argoproj/argo-workflows/v3/server/auth"
	"github.com/argoproj/argo-workflows/v3/server/auth/types"
	"github.com/argoproj/argo-workflows/v3/server/clusterworkflowtemplate"
	"github.com/argoproj/argo-workflows/v3/server/workflowtemplate"
	"github.com/argoproj/argo-workflows/v3/util/instanceid"
	"github.com/argoproj/argo-workflows/v3/util/logging"
	"github.com/argoproj/argo-workflows/v3/workflow/common"
	"github.com/argoproj/argo-workflows/v3/workflow/creator"
)

func Test_cronWorkflowServiceServer(t *testing.T) {
	var unlabelled, cronWf wfv1.CronWorkflow
	wfv1.MustUnmarshal(`apiVersion: argoproj.io/v1alpha1
kind: CronWorkflow
metadata:
  name: my-name
  namespace: my-ns
  labels:
    workflows.argoproj.io/controller-instanceid: my-instanceid
spec:
  schedules:
    - "* * * * *"
  concurrencyPolicy: "Allow"
  startingDeadlineSeconds: 0
  successfulJobsHistoryLimit: 4
  failedJobsHistoryLimit: 2
  workflowSpec:
    podGC:
      strategy: OnPodCompletion
    entrypoint: whalesay
    templates:
      - name: whalesay
        container:
          image: python:alpine3.6
          imagePullPolicy: IfNotPresent
          command: ["sh", -c]
          args: ["echo hello"]`, &cronWf)

	wfv1.MustUnmarshal(`apiVersion: argoproj.io/v1alpha1
kind: CronWorkflow
metadata:
  name: unlabelled
  namespace: my-ns
`, &unlabelled)

	wfClientset := wftFake.NewSimpleClientset(&unlabelled)
	wftmplStore := workflowtemplate.NewWorkflowTemplateClientStore()
	cwftmplStore := clusterworkflowtemplate.NewClusterWorkflowTemplateClientStore()
	server := NewCronWorkflowServer(instanceid.NewService("my-instanceid"), wftmplStore, cwftmplStore, nil)
	baseCtx := logging.WithLogger(context.TODO(), logging.NewSlogLogger(logging.GetGlobalLevel(), logging.GetGlobalFormat()))
	ctx := context.WithValue(context.WithValue(baseCtx, auth.WfKey, wfClientset), auth.ClaimsKey, &types.Claims{Claims: jwt.Claims{Subject: "my-sub"}, Email: "my-sub@your.org"})
	userEmailLabel := "my-sub.at.your.org"

	t.Run("CreateCronWorkflow", func(t *testing.T) {
		created, err := server.CreateCronWorkflow(ctx, &cronworkflowpkg.CreateCronWorkflowRequest{
			Namespace:    "my-ns",
			CronWorkflow: &cronWf,
		})
		require.NoError(t, err)
		assert.NotNil(t, created)
		assert.Contains(t, created.Labels, common.LabelKeyControllerInstanceID)
		assert.Contains(t, created.Labels, common.LabelKeyCreator)
	})
	t.Run("LintWorkflow", func(t *testing.T) {
		wf, err := server.LintCronWorkflow(ctx, &cronworkflowpkg.LintCronWorkflowRequest{
			Namespace:    "my-ns",
			CronWorkflow: &cronWf,
		})
		require.NoError(t, err)
		assert.NotNil(t, wf)
		assert.Contains(t, wf.Labels, common.LabelKeyControllerInstanceID)
		assert.Contains(t, wf.Labels, common.LabelKeyCreator)
		assert.Equal(t, userEmailLabel, wf.Labels[common.LabelKeyCreatorEmail])
	})
	t.Run("ListCronWorkflows", func(t *testing.T) {
		cronWfs, err := server.ListCronWorkflows(ctx, &cronworkflowpkg.ListCronWorkflowsRequest{Namespace: "my-ns"})
		require.NoError(t, err)
		assert.Len(t, cronWfs.Items, 1)
	})
	t.Run("GetCronWorkflow", func(t *testing.T) {
		t.Run("Labelled", func(t *testing.T) {
			cronWf, err := server.GetCronWorkflow(ctx, &cronworkflowpkg.GetCronWorkflowRequest{Namespace: "my-ns", Name: "my-name"})
			require.NoError(t, err)
			assert.NotNil(t, cronWf)
		})
		t.Run("Unlabelled", func(t *testing.T) {
			_, err := server.GetCronWorkflow(ctx, &cronworkflowpkg.GetCronWorkflowRequest{Namespace: "my-ns", Name: "unlabelled"})
			require.Error(t, err)
		})
	})
	t.Run("UpdateCronWorkflow", func(t *testing.T) {
		t.Run("Invalid", func(t *testing.T) {
			x := cronWf.DeepCopy()
			x.Spec.Schedules = []string{"invalid"}
			_, err := server.UpdateCronWorkflow(ctx, &cronworkflowpkg.UpdateCronWorkflowRequest{Namespace: "my-ns", CronWorkflow: x})
			require.Error(t, err)
		})
		t.Run("Labelled", func(t *testing.T) {
			cronWf, err := server.UpdateCronWorkflow(ctx, &cronworkflowpkg.UpdateCronWorkflowRequest{Namespace: "my-ns", CronWorkflow: &cronWf})
			assert.Contains(t, cronWf.Labels, common.LabelKeyActor)
			assert.Equal(t, string(creator.ActionUpdate), cronWf.Labels[common.LabelKeyAction])
			assert.Equal(t, userEmailLabel, cronWf.Labels[common.LabelKeyActorEmail])
			require.NoError(t, err)
			assert.NotNil(t, cronWf)
		})
		t.Run("Unlabelled", func(t *testing.T) {
			_, err := server.UpdateCronWorkflow(ctx, &cronworkflowpkg.UpdateCronWorkflowRequest{Namespace: "my-ns", CronWorkflow: &unlabelled})
			require.Error(t, err)
		})
	})
	t.Run("DeleteCronWorkflow", func(t *testing.T) {
		t.Run("Labelled", func(t *testing.T) {
			_, err := server.DeleteCronWorkflow(ctx, &cronworkflowpkg.DeleteCronWorkflowRequest{Name: "my-name", Namespace: "my-ns"})
			require.NoError(t, err)
		})
		t.Run("Unlabelled", func(t *testing.T) {
			_, err := server.DeleteCronWorkflow(ctx, &cronworkflowpkg.DeleteCronWorkflowRequest{Name: "unlabelled", Namespace: "my-ns"})
			require.Error(t, err)
		})
	})
}
