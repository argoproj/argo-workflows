package clusterworkflowtemplate

import (
	"context"
	"testing"

	"github.com/go-jose/go-jose/v3/jwt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/client-go/kubernetes/fake"

	clusterwftmplpkg "github.com/argoproj/argo-workflows/v3/pkg/apiclient/clusterworkflowtemplate"
	"github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	wftFake "github.com/argoproj/argo-workflows/v3/pkg/client/clientset/versioned/fake"
	"github.com/argoproj/argo-workflows/v3/server/auth"
	"github.com/argoproj/argo-workflows/v3/server/auth/types"
	"github.com/argoproj/argo-workflows/v3/util/instanceid"
	"github.com/argoproj/argo-workflows/v3/util/logging"
	"github.com/argoproj/argo-workflows/v3/workflow/common"
	"github.com/argoproj/argo-workflows/v3/workflow/creator"
)

var unlabelled, cwftObj2, cwftObj3 v1alpha1.ClusterWorkflowTemplate

func init() {
	v1alpha1.MustUnmarshal(`{
    "apiVersion": "argoproj.io/v1alpha1",
    "kind": "ClusterWorkflowTemplate",
    "metadata": {
      "name": "cluster-workflow-template-whalesay-template"
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
}`, &unlabelled)

	v1alpha1.MustUnmarshal(`{
  "apiVersion": "argoproj.io/v1alpha1",
  "kind": "ClusterWorkflowTemplate",
  "metadata": {
    "name": "cluster-workflow-template-whalesay-template2",
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
}`, &cwftObj2)

	v1alpha1.MustUnmarshal(`{
  "apiVersion": "argoproj.io/v1alpha1",
  "kind": "ClusterWorkflowTemplate",
  "metadata": {
    "name": "cluster-workflow-template-whalesay-template3",
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
}`, &cwftObj3)
}

const userEmailLabel = "my-sub.at.your.org"

func getClusterWorkflowTemplateServer() (clusterwftmplpkg.ClusterWorkflowTemplateServiceServer, context.Context) {
	kubeClientSet := fake.NewSimpleClientset()
	wfClientset := wftFake.NewSimpleClientset(&unlabelled, &cwftObj2, &cwftObj3)
	baseCtx := logging.WithLogger(context.TODO(), logging.NewSlogLogger(logging.GetGlobalLevel(), logging.GetGlobalFormat()))
	ctx := context.WithValue(context.WithValue(context.WithValue(baseCtx, auth.WfKey, wfClientset), auth.KubeKey, kubeClientSet), auth.ClaimsKey, &types.Claims{Claims: jwt.Claims{Subject: "my-sub"}, Email: "my-sub@your.org"})
	return NewClusterWorkflowTemplateServer(instanceid.NewService("my-instanceid"), nil, nil), ctx
}

func TestWorkflowTemplateServer_CreateClusterWorkflowTemplate(t *testing.T) {
	server, ctx := getClusterWorkflowTemplateServer()
	t.Run("Without parameter values", func(t *testing.T) {
		tmpl := unlabelled.DeepCopy()
		tmpl.Name = "foo-without-param-values"
		tmpl.Spec.Arguments.Parameters[0].Value = nil
		req := clusterwftmplpkg.ClusterWorkflowTemplateCreateRequest{
			Template: tmpl,
		}
		resp, err := server.CreateClusterWorkflowTemplate(ctx, &req)
		require.NoError(t, err)
		assert.Equal(t, "message", resp.Spec.Arguments.Parameters[0].Name)
		assert.Nil(t, resp.Spec.Arguments.Parameters[0].Value)
	})
	t.Run("With parameter values", func(t *testing.T) {
		cwftReq := clusterwftmplpkg.ClusterWorkflowTemplateCreateRequest{
			Template: unlabelled.DeepCopy(),
		}
		cwftReq.Template.Name = "foo-with-param-values"
		assert.NotContains(t, cwftReq.Template.Labels, common.LabelKeyControllerInstanceID)
		cwftRsp, err := server.CreateClusterWorkflowTemplate(ctx, &cwftReq)
		require.NoError(t, err)
		assert.NotNil(t, cwftRsp)
		// ensure the label is added
		assert.Contains(t, cwftRsp.Labels, common.LabelKeyControllerInstanceID)
		assert.Contains(t, cwftRsp.Labels, common.LabelKeyCreator)
		assert.Equal(t, userEmailLabel, cwftRsp.Labels[common.LabelKeyCreatorEmail])
	})
}

func TestWorkflowTemplateServer_GetClusterWorkflowTemplate(t *testing.T) {
	server, ctx := getClusterWorkflowTemplateServer()
	t.Run("Labelled", func(t *testing.T) {
		cwftRsp, err := server.GetClusterWorkflowTemplate(ctx, &clusterwftmplpkg.ClusterWorkflowTemplateGetRequest{
			Name: "cluster-workflow-template-whalesay-template2",
		})
		require.NoError(t, err)
		assert.NotNil(t, cwftRsp)
		assert.Equal(t, "cluster-workflow-template-whalesay-template2", cwftRsp.Name)
		assert.Contains(t, cwftRsp.Labels, common.LabelKeyControllerInstanceID)
	})
	t.Run("Unlabelled", func(t *testing.T) {
		_, err := server.GetClusterWorkflowTemplate(ctx, &clusterwftmplpkg.ClusterWorkflowTemplateGetRequest{
			Name: "cluster-workflow-template-whalesay-template",
		})
		require.Error(t, err)
	})
}

func TestWorkflowTemplateServer_ListClusterWorkflowTemplates(t *testing.T) {
	server, ctx := getClusterWorkflowTemplateServer()
	cwftReq := clusterwftmplpkg.ClusterWorkflowTemplateListRequest{}
	cwftRsp, err := server.ListClusterWorkflowTemplates(ctx, &cwftReq)
	require.NoError(t, err)
	assert.Len(t, cwftRsp.Items, 2)
	for _, item := range cwftRsp.Items {
		assert.Contains(t, item.Labels, common.LabelKeyControllerInstanceID)
	}
}

func TestWorkflowTemplateServer_DeleteClusterWorkflowTemplate(t *testing.T) {
	server, ctx := getClusterWorkflowTemplateServer()
	t.Run("Labelled", func(t *testing.T) {
		_, err := server.DeleteClusterWorkflowTemplate(ctx, &clusterwftmplpkg.ClusterWorkflowTemplateDeleteRequest{
			Name: "cluster-workflow-template-whalesay-template2",
		})
		require.NoError(t, err)
	})
	t.Run("Unlabelled", func(t *testing.T) {
		_, err := server.DeleteClusterWorkflowTemplate(ctx, &clusterwftmplpkg.ClusterWorkflowTemplateDeleteRequest{
			Name: "cluster-workflow-template-whalesay-template",
		})
		require.Error(t, err)
	})
}

func TestWorkflowTemplateServer_LintClusterWorkflowTemplate(t *testing.T) {
	server, ctx := getClusterWorkflowTemplateServer()
	t.Run("Labelled", func(t *testing.T) {
		var cwftObj1 v1alpha1.ClusterWorkflowTemplate
		resp, err := server.LintClusterWorkflowTemplate(ctx, &clusterwftmplpkg.ClusterWorkflowTemplateLintRequest{
			Template: &cwftObj1,
		})
		require.NoError(t, err)
		assert.Contains(t, resp.Labels, common.LabelKeyControllerInstanceID)
		assert.Contains(t, resp.Labels, common.LabelKeyCreator)
		assert.Equal(t, userEmailLabel, resp.Labels[common.LabelKeyCreatorEmail])
	})

	t.Run("Without param values", func(t *testing.T) {
		tmpl := unlabelled.DeepCopy()
		tmpl.Name = "foo-without-param-values"
		tmpl.Spec.Arguments.Parameters[0].Value = nil
		req := clusterwftmplpkg.ClusterWorkflowTemplateLintRequest{
			Template: tmpl,
		}
		resp, err := server.LintClusterWorkflowTemplate(ctx, &req)
		require.NoError(t, err)
		assert.Equal(t, "message", resp.Spec.Arguments.Parameters[0].Name)
		assert.Nil(t, resp.Spec.Arguments.Parameters[0].Value)
	})
}

func TestWorkflowTemplateServer_UpdateClusterWorkflowTemplate(t *testing.T) {
	server, ctx := getClusterWorkflowTemplateServer()
	t.Run("Labelled", func(t *testing.T) {
		req := &clusterwftmplpkg.ClusterWorkflowTemplateUpdateRequest{
			Template: cwftObj2.DeepCopy(),
		}
		req.Template.Spec.Templates[0].Container.Image = "alpine:latest"
		cwftRsp, err := server.UpdateClusterWorkflowTemplate(ctx, req)
		require.NoError(t, err)
		assert.Contains(t, cwftRsp.Labels, common.LabelKeyActor)
		assert.Equal(t, string(creator.ActionUpdate), cwftRsp.Labels[common.LabelKeyAction])
		assert.Equal(t, userEmailLabel, cwftRsp.Labels[common.LabelKeyActorEmail])
		assert.Equal(t, "alpine:latest", cwftRsp.Spec.Templates[0].Container.Image)
	})
	t.Run("Unlabelled", func(t *testing.T) {
		_, err := server.UpdateClusterWorkflowTemplate(ctx, &clusterwftmplpkg.ClusterWorkflowTemplateUpdateRequest{
			Template: &unlabelled,
		})
		require.Error(t, err)
	})
}
