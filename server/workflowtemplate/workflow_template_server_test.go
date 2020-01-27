package workflowtemplate

import (
	"context"
	"encoding/json"
	"github.com/argoproj/argo/server/auth"
	"testing"

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

func getWorkflowTemplateServer() (WorkflowTemplateServiceServer, context.Context) {
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
	var wftReq WorkflowTemplateCreateRequest
	err := json.Unmarshal([]byte(wftStr1), &wftReq)
	assert.Nil(t, err)
	wftRsp, err := CreateWorkflowTemplate(ctx, &wftReq)
	if assert.NoError(t, err) {
		assert.NotNil(t, wftRsp)
	}
}

func TestWorkflowTemplateServer_GetWorkflowTemplate(t *testing.T) {
	server, ctx := getWorkflowTemplateServer()
	wftReq := WorkflowTemplateGetRequest{
		Name:      "workflow-template-whalesay-template2",
		Namespace: "default",
	}
	wftRsp, err := GetWorkflowTemplate(ctx, &wftReq)
	if assert.NoError(t, err) {
		assert.NotNil(t, wftRsp)
		assert.Equal(t, "workflow-template-whalesay-template2", wftRsp.Name)
	}
}

func TestWorkflowTemplateServer_ListWorkflowTemplates(t *testing.T) {
	server, ctx := getWorkflowTemplateServer()
	wftReq := WorkflowTemplateListRequest{
		Namespace: "default",
	}
	wftRsp, err := ListWorkflowTemplates(ctx, &wftReq)
	if assert.NoError(t, err) {
		assert.Len(t, wftRsp.Items, 2)
	}

	wftReq = WorkflowTemplateListRequest{
		Namespace: "test",
	}
	wftRsp, err = ListWorkflowTemplates(ctx, &wftReq)
	if assert.NoError(t, err) {
		assert.Empty(t, wftRsp.Items)
	}
}

func TestWorkflowTemplateServer_DeleteWorkflowTemplate(t *testing.T) {
	server, ctx := getWorkflowTemplateServer()
	wftReq := WorkflowTemplateDeleteRequest{
		Namespace: "default",
		Name:      "workflow-template-whalesay-template2",
	}
	_, err := DeleteWorkflowTemplate(ctx, &wftReq)
	assert.NoError(t, err)

}

func TestWorkflowTemplateServer_UpdateWorkflowTemplate(t *testing.T) {
	server, ctx := getWorkflowTemplateServer()
	var wftObj1 v1alpha1.WorkflowTemplate
	err := json.Unmarshal([]byte(wftStr2), &wftObj1)
	assert.Nil(t, err)
	wftObj1.Spec.Templates[0].Container.Image = "alpine:latest"
	wftReq := WorkflowTemplateUpdateRequest{
		Namespace: "default",
		Name:      "workflow-template-whalesay-template2",
		Template:  &wftObj1,
	}
	wftRsp, err := UpdateWorkflowTemplate(ctx, &wftReq)

	if assert.NoError(t, err) {
		assert.Equal(t, "alpine:latest", wftRsp.Spec.Templates[0].Container.Image)
	}
}
