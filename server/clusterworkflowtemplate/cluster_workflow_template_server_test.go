package clusterworkflowtemplate

import (
	"context"
	"encoding/json"
	"testing"

	clusterwftmplpkg "github.com/argoproj/argo/pkg/apiclient/clusterworkflowtemplate"
	"github.com/argoproj/argo/server/auth"

	"github.com/stretchr/testify/assert"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	wftFake "github.com/argoproj/argo/pkg/client/clientset/versioned/fake"
)

const cwftStr1 = `{
  "template": {
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
  }
}`

const cwftStr2 = `{
  "apiVersion": "argoproj.io/v1alpha1",
  "kind": "ClusterWorkflowTemplate",
  "metadata": {
    "name": "cluster-workflow-template-whalesay-template2"
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
}`

const cwftStr3 = `{
  "apiVersion": "argoproj.io/v1alpha1",
  "kind": "ClusterWorkflowTemplate",
  "metadata": {
    "name": "cluster-workflow-template-whalesay-template3"
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

func getClusterWorkflowTemplateServer() (clusterwftmplpkg.ClusterWorkflowTemplateServiceServer, context.Context) {
	var cwftObj1, cwftObj2 v1alpha1.ClusterWorkflowTemplate
	err := json.Unmarshal([]byte(cwftStr2), &cwftObj1)
	if err != nil {
		panic(err)
	}
	err = json.Unmarshal([]byte(cwftStr3), &cwftObj2)
	if err != nil {
		panic(err)
	}
	kubeClientSet := fake.NewSimpleClientset()
	wfClientset := wftFake.NewSimpleClientset(&cwftObj1, &cwftObj2)
	ctx := context.WithValue(context.WithValue(context.TODO(), auth.WfKey, wfClientset), auth.KubeKey, kubeClientSet)
	return NewClusterWorkflowTemplateServer(), ctx
}

func TestWorkflowTemplateServer_CreateClusterWorkflowTemplate(t *testing.T) {
	server, ctx := getClusterWorkflowTemplateServer()
	var cwftReq clusterwftmplpkg.ClusterWorkflowTemplateCreateRequest
	err := json.Unmarshal([]byte(cwftStr1), &cwftReq)
	if err != nil {
		panic(err)
	}
	cwftRsp, err := server.CreateClusterWorkflowTemplate(ctx, &cwftReq)
	if assert.NoError(t, err) {
		assert.NotNil(t, cwftRsp)
	}
}

func TestWorkflowTemplateServer_GetClusterWorkflowTemplate(t *testing.T) {
	server, ctx := getClusterWorkflowTemplateServer()
	cwftReq := clusterwftmplpkg.ClusterWorkflowTemplateGetRequest{
		Name:      "cluster-workflow-template-whalesay-template2",
	}
	cwftRsp, err := server.GetClusterWorkflowTemplate(ctx, &cwftReq)
	if assert.NoError(t, err) {
		assert.NotNil(t, cwftRsp)
		assert.Equal(t, "cluster-workflow-template-whalesay-template2", cwftRsp.Name)
	}
}

func TestWorkflowTemplateServer_ListClusterWorkflowTemplates(t *testing.T) {
	server, ctx := getClusterWorkflowTemplateServer()
	cwftReq := clusterwftmplpkg.ClusterWorkflowTemplateListRequest{
	}
	cwftRsp, err := server.ListClusterWorkflowTemplates(ctx, &cwftReq)
	if assert.NoError(t, err) {
		assert.Len(t, cwftRsp.Items, 2)
	}
}

func TestWorkflowTemplateServer_DeleteClusterWorkflowTemplate(t *testing.T) {
	server, ctx := getClusterWorkflowTemplateServer()
	cwftReq := clusterwftmplpkg.ClusterWorkflowTemplateDeleteRequest{
		Name: "cluster-workflow-template-whalesay-template2",
	}
	_, err := server.DeleteClusterWorkflowTemplate(ctx, &cwftReq)
	assert.NoError(t, err)

}

func TestWorkflowTemplateServer_UpdateClusterWorkflowTemplate(t *testing.T) {
	server, ctx := getClusterWorkflowTemplateServer()
	var cwftObj1 v1alpha1.ClusterWorkflowTemplate
	err := json.Unmarshal([]byte(cwftStr2), &cwftObj1)
	if err != nil {
		panic(err)
	}
	cwftObj1.Spec.Templates[0].Container.Image = "alpine:latest"
	cwftReq := clusterwftmplpkg.ClusterWorkflowTemplateUpdateRequest{
		Name:      "cluster-workflow-template-whalesay-template2",
		Template:  &cwftObj1,
	}
	cwftRsp, err := server.UpdateClusterWorkflowTemplate(ctx, &cwftReq)

	if assert.NoError(t, err) {
		assert.Equal(t, "alpine:latest", cwftRsp.Spec.Templates[0].Container.Image)
	}
}
