package cronworkflow

import (
	"context"
	"testing"

	"github.com/ghodss/yaml"
	"github.com/stretchr/testify/assert"

	cronworkflowpkg "github.com/argoproj/argo/pkg/apiclient/cronworkflow"
	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	wftFake "github.com/argoproj/argo/pkg/client/clientset/versioned/fake"
	"github.com/argoproj/argo/server/auth"
	"github.com/argoproj/argo/util/instanceid"
)

func Test_cronWorkflowServiceServer(t *testing.T) {
	cronWfRaw := `apiVersion: argoproj.io/v1alpha1
kind: CronWorkflow
metadata:
  name: my-name
  namespace: my-ns
spec:
  schedule: "* * * * *"
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
          args: ["echo hello"]`

	var cronWf wfv1.CronWorkflow
	err := yaml.Unmarshal([]byte(cronWfRaw), &cronWf)
	if err != nil {
		panic(err)
	}
	wfClientset := wftFake.NewSimpleClientset()
	server := NewCronWorkflowServer(instanceid.NewService("my-instanceid"))
	ctx := context.WithValue(context.TODO(), auth.WfKey, wfClientset)

	t.Run("CreateCronWorkflow", func(t *testing.T) {
		created, err := server.CreateCronWorkflow(ctx, &cronworkflowpkg.CreateCronWorkflowRequest{
			Namespace:    "my-ns",
			CronWorkflow: &cronWf,
		})
		if assert.NoError(t, err) {
			assert.NotNil(t, created)
		}
	})
	t.Run("LintWorkflow", func(t *testing.T) {
		wf, err := server.LintCronWorkflow(ctx, &cronworkflowpkg.LintCronWorkflowRequest{
			Namespace:    "my-ns",
			CronWorkflow: &cronWf,
		})
		if assert.NoError(t, err) {
			assert.NotNil(t, wf)
		}
	})
	t.Run("ListCronWorkflows", func(t *testing.T) {
		cronWfs, err := server.ListCronWorkflows(ctx, &cronworkflowpkg.ListCronWorkflowsRequest{Namespace: "my-ns"})
		if assert.NoError(t, err) {
			assert.Len(t, cronWfs.Items, 1)
		}
	})
	t.Run("GetCronWorkflow", func(t *testing.T) {
		cronWf, err := server.GetCronWorkflow(ctx, &cronworkflowpkg.GetCronWorkflowRequest{Namespace: "my-ns", Name: "my-name"})
		if assert.NoError(t, err) {
			assert.NotNil(t, cronWf)
		}
	})
	t.Run("UpdateCronWorkflow", func(t *testing.T) {
		cronWf, err := server.UpdateCronWorkflow(ctx, &cronworkflowpkg.UpdateCronWorkflowRequest{Namespace: "my-ns", Name: "my-name", CronWorkflow: &cronWf})
		if assert.NoError(t, err) {
			assert.NotNil(t, cronWf)
		}
	})
	t.Run("DeleteCronWorkflow", func(t *testing.T) {
		_, err := server.DeleteCronWorkflow(ctx, &cronworkflowpkg.DeleteCronWorkflowRequest{Name: "my-name", Namespace: "my-ns"})
		assert.NoError(t, err)
	})
}
