package workflowtemplate

import (
	"context"
	"encoding/json"
	"testing"

	v1alpha1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	wftFake "github.com/argoproj/argo/pkg/client/clientset/versioned/fake"
	"github.com/stretchr/testify/assert"
	"k8s.io/client-go/kubernetes/fake"

)

const wftStr1 =`
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
const wftStr2 =`
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

const wftStr3 =`
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


func getWorkflowTempplateServer()*WorkflowTemplateServer{
	var wftObj1, wftObj2 v1alpha1.WorkflowTemplate
	_ = json.Unmarshal([]byte(wftStr2), &wftObj1)
	_ = json.Unmarshal([]byte(wftStr3), &wftObj2)
	kubeClientSet := fake.NewSimpleClientset()
	wfClientset := wftFake.NewSimpleClientset(&wftObj1, &wftObj2)
	return NewWorkflowTemplateServer("default", wfClientset, kubeClientSet, false)
}

func TestWorkflowTemplateServer_CreateWorkflowTemplate(t *testing.T) {
	server := getWorkflowTempplateServer()
	var wftReq WorkflowTemplateCreateRequest
	err := json.Unmarshal([]byte(wftStr1), &wftReq)
	assert.Nil(t, err)
	wftRsp, err := server.CreateWorkflowTemplate(context.TODO(), &wftReq)
	assert.NotNil(t, wftRsp)
	assert.Nil(t, err)
}

func TestWorkflowTemplateServer_GetWorkflowTemplate(t *testing.T) {
	server := getWorkflowTempplateServer()
	wftReq := WorkflowTemplateGetRequest{
		TemplateName:         "workflow-template-whalesay-template2",
		Namespace:            "default",
	}
	wftRsp, err := server.GetWorkflowTemplate(context.TODO(), &wftReq)
	assert.NotNil(t, wftRsp)
	assert.Equal(t, "workflow-template-whalesay-template2", wftRsp.Name)
	assert.Nil(t, err)
}

func TestWorkflowTemplateServer_ListWorkflowTemplates(t *testing.T) {
	server := getWorkflowTempplateServer()
	wftReq := WorkflowTemplateListRequest{
		Namespace:            "default",
	}
	wftRsp, err := server.ListWorkflowTemplates(context.TODO(), &wftReq)
	assert.Equal(t, 2, len(wftRsp.Items))
	assert.Nil(t, err)

	wftReq = WorkflowTemplateListRequest{
		Namespace:            "test",
	}
	wftRsp, err = server.ListWorkflowTemplates(context.TODO(), &wftReq)
	assert.Equal(t, 0, len(wftRsp.Items))
	assert.Nil(t, err)
}

func TestWorkflowTemplateServer_DeleteWorkflowTemplate(t *testing.T) {
	server := getWorkflowTempplateServer()
	wftReq := WorkflowTemplateDeleteRequest{
		Namespace:            "default",
		TemplateName: "workflow-template-whalesay-template2",
	}
	wftRsp, err :=server.DeleteWorkflowTemplate(context.TODO(), &wftReq)

	assert.Equal(t, "Deleted", wftRsp.Status)
	assert.Nil(t, err)


}

func TestWorkflowTemplateServer_UpdateWorkflowTemplate(t *testing.T) {
	server := getWorkflowTempplateServer()
	var wftObj1 v1alpha1.WorkflowTemplate
	err := json.Unmarshal([]byte(wftStr2), &wftObj1)
	assert.Nil(t, err)
	wftObj1.Spec.Templates[0].Container.Image = "alpine:latest"
	wftReq := WorkflowTemplateUpdateRequest{
		Namespace:            "default",
		TemplateName:         "workflow-template-whalesay-template2",
		Template: &wftObj1,
		}
	wftRsp, err :=server.UpdateWorkflowTemplate(context.TODO(), &wftReq)

	assert.Equal(t, "alpine:latest", wftRsp.Spec.Templates[0].Container.Image)
	assert.Nil(t, err)


}