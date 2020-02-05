package workflowtemplate

import (
	"context"
	"encoding/json"
	"testing"

	workflowtemplatepkg "github.com/argoproj/argo/pkg/apiclient/workflowtemplate"
	"github.com/argoproj/argo/server/auth"

	"github.com/stretchr/testify/assert"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	wftFake "github.com/argoproj/argo/pkg/client/clientset/versioned/fake"
)

const wftStr1 = `
{
	"namespace": "default",
	"template":
	{
	  "apiVersion": "argoproj.io/v1alpha1",
	  "kind": "WorkflowTemplate",
	  "metadata": {
		"name": "workflow-template-whalesay-template"
	  },
	  "spec": {
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
}
`
const wftStr2 = `
{
  "apiVersion": "argoproj.io/v1alpha1",
  "kind": "WorkflowTemplate",
  "metadata": {
    "name": "workflow-template-whalesay-template2",
    "namespace": "default"

  },
  "spec": {
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
`

const wftStr3 = `
{
  "apiVersion": "argoproj.io/v1alpha1",
  "kind": "WorkflowTemplate",
  "metadata": {
    "name": "workflow-template-whalesay-template3",
	"namespace": "default"
  },
  "spec": {
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
`

func getWorkflowTemplateServer() (workflowtemplatepkg.WorkflowTemplateServiceServer, context.Context) {
	var wftObj1, wftObj2 v1alpha1.WorkflowTemplate
	_ = json.Unmarshal([]byte(wftStr2), &wftObj1)
	_ = json.Unmarshal([]byte(wftStr3), &wftObj2)
	kubeClientSet := fake.NewSimpleClientset()
	wfClientset := wftFake.NewSimpleClientset(&wftObj1, &wftObj2)
	ctx := context.WithValue(context.WithValue(context.TODO(), auth.WfKey, wfClientset), auth.KubeKey, kubeClientSet)
	return NewWorkflowTemplateServer(), ctx
}

func TestWorkflowTemplateServer_CreateWorkflowTemplate(t *testing.T) {
	server, ctx := getWorkflowTemplateServer()
	var wftReq workflowtemplatepkg.WorkflowTemplateCreateRequest
	err := json.Unmarshal([]byte(wftStr1), &wftReq)
	assert.Nil(t, err)
	wftRsp, err := server.CreateWorkflowTemplate(ctx, &wftReq)
	if assert.NoError(t, err) {
		assert.NotNil(t, wftRsp)
	}
}

func TestWorkflowTemplateServer_GetWorkflowTemplate(t *testing.T) {
	server, ctx := getWorkflowTemplateServer()
	wftReq := workflowtemplatepkg.WorkflowTemplateGetRequest{
		Name:      "workflow-template-whalesay-template2",
		Namespace: "default",
	}
	wftRsp, err := server.GetWorkflowTemplate(ctx, &wftReq)
	if assert.NoError(t, err) {
		assert.NotNil(t, wftRsp)
		assert.Equal(t, "workflow-template-whalesay-template2", wftRsp.Name)
	}
}

func TestWorkflowTemplateServer_ListWorkflowTemplates(t *testing.T) {
	server, ctx := getWorkflowTemplateServer()
	wftReq := workflowtemplatepkg.WorkflowTemplateListRequest{
		Namespace: "default",
	}
	wftRsp, err := server.ListWorkflowTemplates(ctx, &wftReq)
	if assert.NoError(t, err) {
		assert.Len(t, wftRsp.Items, 2)
	}

	wftReq = workflowtemplatepkg.WorkflowTemplateListRequest{
		Namespace: "test",
	}
	wftRsp, err = server.ListWorkflowTemplates(ctx, &wftReq)
	if assert.NoError(t, err) {
		assert.Empty(t, wftRsp.Items)
	}
}

func TestWorkflowTemplateServer_DeleteWorkflowTemplate(t *testing.T) {
	server, ctx := getWorkflowTemplateServer()
	wftReq := workflowtemplatepkg.WorkflowTemplateDeleteRequest{
		Namespace: "default",
		Name:      "workflow-template-whalesay-template2",
	}
	_, err := server.DeleteWorkflowTemplate(ctx, &wftReq)
	assert.NoError(t, err)

}

func TestWorkflowTemplateServer_UpdateWorkflowTemplate(t *testing.T) {
	server, ctx := getWorkflowTemplateServer()
	var wftObj1 v1alpha1.WorkflowTemplate
	err := json.Unmarshal([]byte(wftStr2), &wftObj1)
	assert.Nil(t, err)
	wftObj1.Spec.Templates[0].Container.Image = "alpine:latest"
	wftReq := workflowtemplatepkg.WorkflowTemplateUpdateRequest{
		Namespace: "default",
		Name:      "workflow-template-whalesay-template2",
		Template:  &wftObj1,
	}
	wftRsp, err := server.UpdateWorkflowTemplate(ctx, &wftReq)

	if assert.NoError(t, err) {
		assert.Equal(t, "alpine:latest", wftRsp.Spec.Templates[0].Container.Image)
	}
}
