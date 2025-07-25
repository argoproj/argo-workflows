package workflowtemplate

import (
	"context"
	"testing"

	"github.com/argoproj/argo-workflows/v3/util/logging"

	"github.com/go-jose/go-jose/v3/jwt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"

	workflowtemplatepkg "github.com/argoproj/argo-workflows/v3/pkg/apiclient/workflowtemplate"
	"github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	wftFake "github.com/argoproj/argo-workflows/v3/pkg/client/clientset/versioned/fake"
	"github.com/argoproj/argo-workflows/v3/server/auth"
	"github.com/argoproj/argo-workflows/v3/server/auth/types"
	"github.com/argoproj/argo-workflows/v3/server/clusterworkflowtemplate"
	"github.com/argoproj/argo-workflows/v3/util/instanceid"
	"github.com/argoproj/argo-workflows/v3/workflow/common"
	"github.com/argoproj/argo-workflows/v3/workflow/creator"
)

const unlabelled = `{
    "apiVersion": "argoproj.io/v1alpha1",
    "kind": "WorkflowTemplate",
    "metadata": {
      "name": "unlabelled",
      "namespace": "default"
    }
}`

const wftStr1 = `{
  "namespace": "default",
  "template": {
    "apiVersion": "argoproj.io/v1alpha1",
    "kind": "WorkflowTemplate",
    "metadata": {
      "name": "workflow-template-whalesay-template",
      "labels": {
		"workflows.argoproj.io/controller-instanceid": "my-instanceid"
	  }
    },
    "spec": {
      "arguments": {
        "parameters": [
          {
            "name": "message",
            "value": "Hello Argo"
          }
        ]
      },
      "templates": [
        {
          "name": "whalesay-template",
          "inputs": {
            "parameters": [
              {
                "name": "message"
              }
            ]
          },
          "container": {
            "image": "docker/whalesay",
            "command": [
              "cowsay"
            ],
            "args": [
              "{{inputs.parameters.message}}"
            ]
          }
        }
      ]
    }
  }
}`

const wftStr2 = `{
  "apiVersion": "argoproj.io/v1alpha1",
  "kind": "WorkflowTemplate",
  "metadata": {
    "name": "workflow-template-whalesay-template2",
    "namespace": "default",
	"labels": {
		"workflows.argoproj.io/controller-instanceid": "my-instanceid"
  	}
  },
  "spec": {
	"arguments": {
	  "parameters": [
		{
			"name": "message",
			"value": "Hello Argo",
			"description": "message description"
		}
	  ]
	},
    "templates": [
      {
        "name": "whalesay-template",
        "inputs": {
          "parameters": [
            {
              "name": "message",
              "value": "Hello Argo"
            }
          ]
        },
        "container": {
          "image": "docker/whalesay",
          "command": [
            "cowsay"
          ],
          "args": [
            "{{inputs.parameters.message}}"
          ]
        }
      }
    ]
  }
}`

const wftStr3 = `{
  "apiVersion": "argoproj.io/v1alpha1",
  "kind": "WorkflowTemplate",
  "metadata": {
    "name": "workflow-template-whalesay-template3",
	"namespace": "default",
	"labels": {
		"workflows.argoproj.io/controller-instanceid": "my-instanceid"
  	}
  },
  "spec": {
	"arguments": {
	  "parameters": [
		{
			"name": "message",
			"value": "Hello Argo"
		}
	  ]
	},
    "templates": [
      {
        "name": "whalesay-template",
        "inputs": {
          "parameters": [
            {
              "name": "message"
            }
          ]
        },
        "container": {
          "image": "docker/whalesay",
          "command": [
            "cowsay"
          ],
          "args": [
            "{{inputs.parameters.message}}"
          ]
        }
      }
    ]
  }
}`

const userEmailLabel = "my-sub.at.your.org"

func getWorkflowTemplateServer(t *testing.T) (workflowtemplatepkg.WorkflowTemplateServiceServer, context.Context) {
	t.Helper()
	var unlabelledObj, wftObj1, wftObj2 v1alpha1.WorkflowTemplate
	v1alpha1.MustUnmarshal(unlabelled, &unlabelledObj)
	v1alpha1.MustUnmarshal(wftStr2, &wftObj1)
	v1alpha1.MustUnmarshal(wftStr3, &wftObj2)
	kubeClientSet := fake.NewSimpleClientset()
	wfClientset := wftFake.NewSimpleClientset(&unlabelledObj, &wftObj1, &wftObj2)
	ctx := context.WithValue(context.WithValue(context.WithValue(logging.TestContext(t.Context()), auth.WfKey, wfClientset), auth.KubeKey, kubeClientSet), auth.ClaimsKey, &types.Claims{Claims: jwt.Claims{Subject: "my-sub"}, Email: "my-sub@your.org"})
	wftmplStore := NewWorkflowTemplateClientStore()
	cwftmplStore := clusterworkflowtemplate.NewClusterWorkflowTemplateClientStore()
	return NewWorkflowTemplateServer(instanceid.NewService("my-instanceid"), wftmplStore, cwftmplStore), ctx
}

func TestWorkflowTemplateServer_CreateWorkflowTemplate(t *testing.T) {
	server, ctx := getWorkflowTemplateServer(t)
	t.Run("Without parameter values", func(t *testing.T) {
		var wftReq workflowtemplatepkg.WorkflowTemplateCreateRequest
		v1alpha1.MustUnmarshal(wftStr1, &wftReq)
		wftReq.Template.Name = "foo-without-param-values"
		wftReq.Template.Spec.Arguments.Parameters[0].Value = nil
		wftRsp, err := server.CreateWorkflowTemplate(ctx, &wftReq)
		require.NoError(t, err)
		assert.NotNil(t, wftRsp)
		assert.Equal(t, "message", wftRsp.Spec.Arguments.Parameters[0].Name)
		assert.Nil(t, wftRsp.Spec.Arguments.Parameters[0].Value)
	})
	t.Run("Labelled", func(t *testing.T) {
		var wftReq workflowtemplatepkg.WorkflowTemplateCreateRequest
		v1alpha1.MustUnmarshal(wftStr1, &wftReq)
		wftRsp, err := server.CreateWorkflowTemplate(ctx, &wftReq)
		require.NoError(t, err)
		assert.NotNil(t, wftRsp)
	})
	t.Run("Unlabelled", func(t *testing.T) {
		var wftReq workflowtemplatepkg.WorkflowTemplateCreateRequest
		v1alpha1.MustUnmarshal(unlabelled, &wftReq.Template)
		wftReq.Namespace = "default"
		wftReq.Template.Name = "foo"
		wftRsp, err := server.CreateWorkflowTemplate(ctx, &wftReq)
		require.NoError(t, err)
		assert.NotNil(t, wftRsp)
		assert.Contains(t, wftRsp.Labels, common.LabelKeyControllerInstanceID)
		assert.Contains(t, wftRsp.Labels, common.LabelKeyCreator)
		assert.Equal(t, userEmailLabel, wftRsp.Labels[common.LabelKeyCreatorEmail])
	})
}

func TestWorkflowTemplateServer_GetWorkflowTemplate(t *testing.T) {
	server, ctx := getWorkflowTemplateServer(t)
	t.Run("Labelled", func(t *testing.T) {
		wftRsp, err := server.GetWorkflowTemplate(ctx, &workflowtemplatepkg.WorkflowTemplateGetRequest{Name: "workflow-template-whalesay-template2", Namespace: "default"})
		require.NoError(t, err)
		assert.NotNil(t, wftRsp)
		assert.Equal(t, "workflow-template-whalesay-template2", wftRsp.Name)
		assert.Equal(t, "message", wftRsp.Spec.Arguments.Parameters[0].Name)
		assert.Equal(t, "message description", wftRsp.Spec.Arguments.Parameters[0].Description.String())
	})
	t.Run("Unlabelled", func(t *testing.T) {
		_, err := server.GetWorkflowTemplate(ctx, &workflowtemplatepkg.WorkflowTemplateGetRequest{Name: "unlabelled", Namespace: "default"})
		require.Error(t, err)
	})
}

func TestWorkflowTemplateServer_ListWorkflowTemplates(t *testing.T) {
	server, ctx := getWorkflowTemplateServer(t)
	wftRsp, err := server.ListWorkflowTemplates(ctx, &workflowtemplatepkg.WorkflowTemplateListRequest{Namespace: "default"})
	require.NoError(t, err)
	assert.Len(t, wftRsp.Items, 2)
	wftRsp, err = server.ListWorkflowTemplates(ctx, &workflowtemplatepkg.WorkflowTemplateListRequest{Namespace: "test"})
	require.NoError(t, err)
	assert.Empty(t, wftRsp.Items)
}

func TestWorkflowTemplateServer_DeleteWorkflowTemplate(t *testing.T) {
	server, ctx := getWorkflowTemplateServer(t)
	t.Run("Labelled", func(t *testing.T) {
		_, err := server.DeleteWorkflowTemplate(ctx, &workflowtemplatepkg.WorkflowTemplateDeleteRequest{Namespace: "default", Name: "workflow-template-whalesay-template2"})
		require.NoError(t, err)
	})
	t.Run("Unlabelled", func(t *testing.T) {
		_, err := server.DeleteWorkflowTemplate(ctx, &workflowtemplatepkg.WorkflowTemplateDeleteRequest{Namespace: "default", Name: "unlabelled"})
		require.Error(t, err)
	})
}

func TestWorkflowTemplateServer_LintWorkflowTemplate(t *testing.T) {
	server, ctx := getWorkflowTemplateServer(t)
	tmpl, err := server.LintWorkflowTemplate(ctx, &workflowtemplatepkg.WorkflowTemplateLintRequest{
		Template: &v1alpha1.WorkflowTemplate{},
	})
	require.NoError(t, err)
	assert.Contains(t, tmpl.Labels, common.LabelKeyControllerInstanceID)
}

func TestWorkflowTemplateServer_UpdateWorkflowTemplate(t *testing.T) {
	server, ctx := getWorkflowTemplateServer(t)
	t.Run("Labelled", func(t *testing.T) {
		var wftObj1 v1alpha1.WorkflowTemplate
		v1alpha1.MustUnmarshal(wftStr2, &wftObj1)
		wftObj1.Spec.Templates[0].Container.Image = "alpine:latest"
		wftRsp, err := server.UpdateWorkflowTemplate(ctx, &workflowtemplatepkg.WorkflowTemplateUpdateRequest{
			Namespace: "default",
			Template:  &wftObj1,
		})
		require.NoError(t, err)
		assert.Contains(t, wftRsp.Labels, common.LabelKeyActor)
		assert.Equal(t, string(creator.ActionUpdate), wftRsp.Labels[common.LabelKeyAction])
		assert.Equal(t, userEmailLabel, wftRsp.Labels[common.LabelKeyActorEmail])
		assert.Equal(t, "alpine:latest", wftRsp.Spec.Templates[0].Container.Image)
	})
	t.Run("Unlabelled", func(t *testing.T) {
		_, err := server.UpdateWorkflowTemplate(ctx, &workflowtemplatepkg.WorkflowTemplateUpdateRequest{
			Template: &v1alpha1.WorkflowTemplate{
				ObjectMeta: metav1.ObjectMeta{Name: "unlabelled"},
			},
		})
		require.Error(t, err)
	})
}
