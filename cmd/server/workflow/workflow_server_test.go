package workflow

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	v1alpha "github.com/argoproj/argo/pkg/client/clientset/versioned/fake"
)

const wf1 = `
{
    "apiVersion": "argoproj.io/v1alpha1",
    "kind": "Workflow",
    "metadata": {
        "creationTimestamp": "2019-12-13T23:36:32Z",
        "generateName": "hello-world-",
        "generation": 5,
        "labels": {
            "workflows.argoproj.io/completed": "true",
            "workflows.argoproj.io/phase": "Succeeded"
        },
        "name": "hello-world-9tql2",
        "namespace": "workflows",
        "resourceVersion": "53020772",
        "selfLink": "/apis/argoproj.io/v1alpha1/namespaces/workflows/workflows/hello-world-9tql2",
        "uid": "6522aff1-1e01-11ea-b443-42010aa80075"
    },
    "spec": {
        "arguments": {},
        "entrypoint": "whalesay",
        "templates": [
            {
                "arguments": {},
                "container": {
                    "args": [
                        "hello world"
                    ],
                    "command": [
                        "cowsay"
                    ],
                    "image": "docker/whalesay:latest",
                    "name": "",
                    "resources": {}
                },
                "inputs": {},
                "metadata": {},
                "name": "whalesay",
                "outputs": {}
            }
        ]
    },
    "status": {
        "finishedAt": "2019-12-13T23:36:40Z",
        "nodes": {
            "hello-world-9tql2": {
                "displayName": "hello-world-9tql2",
                "finishedAt": "2019-12-13T23:36:39Z",
                "id": "hello-world-9tql2",
                "name": "hello-world-9tql2",
                "phase": "Succeeded",
                "startedAt": "2019-12-13T23:36:32Z",
                "templateName": "whalesay",
                "type": "Pod"
            }
        },
        "phase": "Succeeded",
        "startedAt": "2019-12-13T23:36:32Z"
    }
}
`
const wf2 = `
{
    "apiVersion": "argoproj.io/v1alpha1",
    "kind": "Workflow",
    "metadata": {
        "creationTimestamp": "2019-12-13T19:12:55Z",
        "generateName": "hello-world-",
        "generation": 5,
        "labels": {
            "workflows.argoproj.io/completed": "true",
            "workflows.argoproj.io/phase": "Succeeded"
        },
        "name": "hello-world-b6h5m",
        "namespace": "workflows",
        "resourceVersion": "52919656",
        "selfLink": "/apis/argoproj.io/v1alpha1/namespaces/workflows/workflows/hello-world-b6h5m",
        "uid": "91066a6c-1ddc-11ea-b443-42010aa80075"
    },
    "spec": {
        "arguments": {},
        "entrypoint": "whalesay",
        "templates": [
            {
                "arguments": {},
                "container": {
                    "args": [
                        "hello world"
                    ],
                    "command": [
                        "cowsay"
                    ],
                    "image": "docker/whalesay:latest",
                    "name": "",
                    "resources": {}
                },
                "inputs": {},
                "metadata": {},
                "name": "whalesay",
                "outputs": {}
            }
        ]
    },
    "status": {
        "finishedAt": "2019-12-13T19:12:59Z",
        "nodes": {
            "hello-world-b6h5m": {
                "displayName": "hello-world-b6h5m",
                "finishedAt": "2019-12-13T19:12:58Z",
                "id": "hello-world-b6h5m",
                "name": "hello-world-b6h5m",
                "phase": "Succeeded",
                "startedAt": "2019-12-13T19:12:55Z",
                "templateName": "whalesay",
                "type": "Pod"
            }
        },
        "phase": "Succeeded",
        "startedAt": "2019-12-13T19:12:55Z"
    }
}
`
const wf3 = `
{
    "apiVersion": "argoproj.io/v1alpha1",
    "kind": "Workflow",
    "metadata": {
        "creationTimestamp": "2019-12-13T23:36:32Z",
        "generateName": "hello-world-",
        "generation": 5,
        "labels": {
            "workflows.argoproj.io/completed": "true",
            "workflows.argoproj.io/phase": "Succeeded"
        },
        "name": "hello-world-9tql2-test",
        "namespace": "test",
        "resourceVersion": "53020772",
        "selfLink": "/apis/argoproj.io/v1alpha1/namespaces/workflows/workflows/hello-world-9tql2",
        "uid": "6522aff1-1e01-11ea-b443-42010aa80075"
    },
    "spec": {
        "arguments": {},
        "entrypoint": "whalesay",
        "templates": [
            {
                "arguments": {},
                "container": {
                    "args": [
                        "hello world"
                    ],
                    "command": [
                        "cowsay"
                    ],
                    "image": "docker/whalesay:latest",
                    "name": "",
                    "resources": {}
                },
                "inputs": {},
                "metadata": {},
                "name": "whalesay",
                "outputs": {}
            }
        ]
    },
    "status": {
        "finishedAt": "2019-12-13T23:36:40Z",
        "nodes": {
            "hello-world-9tql2": {
                "displayName": "hello-world-9tql2",
                "finishedAt": "2019-12-13T23:36:39Z",
                "id": "hello-world-9tql2",
                "name": "hello-world-9tql2",
                "phase": "Succeeded",
                "startedAt": "2019-12-13T23:36:32Z",
                "templateName": "whalesay",
                "type": "Pod"
            }
        },
        "phase": "Succeeded",
        "startedAt": "2019-12-13T23:36:32Z"
    }
}
`
const wf4 = `
{
    "apiVersion": "argoproj.io/v1alpha1",
    "kind": "Workflow",
    "metadata": {
        "creationTimestamp": "2019-12-13T19:12:55Z",
        "generateName": "hello-world-",
        "generation": 5,
        "labels": {
            "workflows.argoproj.io/completed": "true",
            "workflows.argoproj.io/phase": "Succeeded"
        },
        "name": "hello-world-b6h5m-test",
        "namespace": "test",
        "resourceVersion": "52919656",
        "selfLink": "/apis/argoproj.io/v1alpha1/namespaces/workflows/workflows/hello-world-b6h5m",
        "uid": "91066a6c-1ddc-11ea-b443-42010aa80075"
    },
    "spec": {
        "arguments": {},
        "entrypoint": "whalesay",
        "templates": [
            {
                "arguments": {},
                "container": {
                    "args": [
                        "hello world"
                    ],
                    "command": [
                        "cowsay"
                    ],
                    "image": "docker/whalesay:latest",
                    "name": "",
                    "resources": {}
                },
                "inputs": {},
                "metadata": {},
                "name": "whalesay",
                "outputs": {}
            }
        ]
    },
    "status": {
        "finishedAt": "2019-12-13T19:12:59Z",
        "nodes": {
            "hello-world-b6h5m": {
                "displayName": "hello-world-b6h5m",
                "finishedAt": "2019-12-13T19:12:58Z",
                "id": "hello-world-b6h5m",
                "name": "hello-world-b6h5m",
                "phase": "Succeeded",
                "startedAt": "2019-12-13T19:12:55Z",
                "templateName": "whalesay",
                "type": "Pod"
            }
        },
        "phase": "Succeeded",
        "startedAt": "2019-12-13T19:12:55Z"
    }
}
`
const wf5 = `
{
    "apiVersion": "argoproj.io/v1alpha1",
    "kind": "Workflow",
    "metadata": {
        "creationTimestamp": "2019-12-13T23:36:32Z",
        "generateName": "hello-world-",
        "generation": 5,
        "labels": {
            "workflows.argoproj.io/completed": "false",
            "workflows.argoproj.io/phase": "Running"
        },
        "name": "hello-world-9tql2-run",
        "namespace": "workflows",
        "resourceVersion": "53020772",
        "selfLink": "/apis/argoproj.io/v1alpha1/namespaces/workflows/workflows/hello-world-9tql2",
        "uid": "6522aff1-1e01-11ea-b443-42010aa80075"
    },
    "spec": {
        "arguments": {},
        "entrypoint": "whalesay",
        "templates": [
            {
                "arguments": {},
                "container": {
                    "args": [
                        "hello world"
                    ],
                    "command": [
                        "cowsay"
                    ],
                    "image": "docker/whalesay:latest",
                    "name": "",
                    "resources": {}
                },
                "inputs": {},
                "metadata": {},
                "name": "whalesay",
                "outputs": {}
            }
        ]
    },
    "status": {
        "finishedAt": "2019-12-13T23:36:40Z",
        "nodes": {
            "hello-world-9tql2": {
                "displayName": "hello-world-9tql2-run",
                "finishedAt": "2019-12-13T23:36:39Z",
                "id": "hello-world-9tql2",
                "name": "hello-world-9tql2",
                "phase": "Running",
                "startedAt": "2019-12-13T23:36:32Z",
                "templateName": "whalesay",
                "type": "Pod"
            }
        },
        "phase": "Running",
        "startedAt": "2019-12-13T23:36:32Z"
    }
}

`
const workflow = `
{
  "namespace": "default",
  "workflow": {
    "apiVersion": "argoproj.io/v1alpha1",
    "kind": "Workflow",
    "metadata": {
      "generateName": "hello-world-"
    },
    "spec": {
      "entrypoint": "whalesay",
      "templates": [
        {
          "name": "whalesay",
          "container": {
            "image": "docker/whalesay:latest",
            "command": [
              "cowsay"
            ],
            "args": [
              "hello world"
            ]
          }
        }
      ]
    }
  }
}
`

func getWorkflowServer() *workflowServer {

	var wfObj1, wfObj2, wfObj3, wfObj4, wfObj5 v1alpha1.Workflow
	_ = json.Unmarshal([]byte(wf1), &wfObj1)
	_ = json.Unmarshal([]byte(wf2), &wfObj2)
	_ = json.Unmarshal([]byte(wf3), &wfObj3)
	_ = json.Unmarshal([]byte(wf4), &wfObj4)
	_ = json.Unmarshal([]byte(wf5), &wfObj5)
	kubeClientSet := fake.NewSimpleClientset()
	wfClientset := v1alpha.NewSimpleClientset(&wfObj1, &wfObj2, &wfObj3, &wfObj4, &wfObj5)
	server := NewWorkflowServer("Default", wfClientset, kubeClientSet, false, nil)
	return server
}

func getWorkflow(server *workflowServer, namespace string, wfName string) (*v1alpha1.Workflow, error) {

	req := WorkflowGetRequest{
		WorkflowName: wfName,
		Namespace:    namespace,
	}

	return server.GetWorkflow(context.TODO(), &req)
}

func getWorkflowList(server *workflowServer, namespace string) (*v1alpha1.WorkflowList, error) {
	req := WorkflowListRequest{
		Namespace: namespace,
	}
	return server.ListWorkflows(context.TODO(), &req)

}

func TestCreateWorkflow(t *testing.T) {

	server := getWorkflowServer()
	var req WorkflowCreateRequest
	_ = json.Unmarshal([]byte(workflow), &req)

	wf, err := server.CreateWorkflow(context.TODO(), &req)

	assert.NotNil(t, wf)
	assert.Nil(t, err)

}

func TestGetWorkflowWithFound(t *testing.T) {

	server := getWorkflowServer()

	wf, err := getWorkflow(server, "workflows", "hello-world-b6h5m")
	assert.NotNil(t, wf)
	assert.Nil(t, err)

	wf, err = getWorkflow(server, "test", "hello-world-b6h5m-test")
	assert.NotNil(t, wf)
	assert.Nil(t, err)
}

func TestGetWorkflowWithNotFound(t *testing.T) {

	server := getWorkflowServer()

	wf, err := getWorkflow(server, "test", "NotFound")
	assert.Nil(t, wf)
	assert.NotNil(t, err)

}

func TestListWorkflow(t *testing.T) {

	server := getWorkflowServer()

	wfl, err := getWorkflowList(server, "workflows")
	assert.NotNil(t, wfl)
	assert.Equal(t, 3, len(wfl.Items))
	assert.Nil(t, err)

	wfl, err = getWorkflowList(server, "test")
	assert.NotNil(t, wfl)
	assert.Equal(t, 2, len(wfl.Items))
	assert.Nil(t, err)
}

func TestDeleteWorkflow(t *testing.T) {

	server := getWorkflowServer()

	wf, err := getWorkflow(server, "workflows", "hello-world-b6h5m")
	assert.Nil(t, err)
	delReq := WorkflowDeleteRequest{
		WorkflowName: wf.Name,
		Namespace:    wf.Namespace,
	}

	delRsp, err := server.DeleteWorkflow(context.TODO(), &delReq)

	assert.NotNil(t, delRsp)
	assert.Equal(t, wf.Name, delRsp.WorkflowName)
	assert.Equal(t, "Deleted", delRsp.Status)
	assert.Nil(t, err)

	wfl, err := getWorkflowList(server, "workflows")
	assert.NotNil(t, wf)
	assert.Equal(t, 2, len(wfl.Items))
	assert.Nil(t, err)

}

func TestSuspendResumeWorkflow(t *testing.T) {
	server := getWorkflowServer()

	wf, err := getWorkflow(server, "workflows", "hello-world-9tql2-run")
	assert.Nil(t, err)
	rsmWfReq := WorkflowUpdateRequest{
		WorkflowName: wf.Name,
		Namespace:    wf.Namespace,
	}
	wf, err = server.SuspendWorkflow(context.TODO(), &rsmWfReq)
	assert.NotNil(t, wf)
	assert.Equal(t, true, *wf.Spec.Suspend)
	assert.Nil(t, err)

	wf, err = server.ResumeWorkflow(context.TODO(), &rsmWfReq)

	assert.NotNil(t, wf)
	assert.Nil(t, wf.Spec.Suspend)
	assert.Nil(t, err)
}

func TestSuspendResumeWorkflowWithNotFound(t *testing.T) {
	server := getWorkflowServer()

	rsmWfReq := WorkflowUpdateRequest{
		WorkflowName: "hello-world-9tql2-not",
		Namespace:    "workflows",
	}
	wf, err := server.SuspendWorkflow(context.TODO(), &rsmWfReq)
	assert.Nil(t, wf)
	assert.NotNil(t, err)
	wf, err = server.ResumeWorkflow(context.TODO(), &rsmWfReq)
	assert.Nil(t, wf)
	assert.NotNil(t, err)
}

func TestTerminateWorkflow(t *testing.T) {
	server := getWorkflowServer()

	wf, err := getWorkflow(server, "workflows", "hello-world-9tql2-run")
	assert.Nil(t, err)
	rsmWfReq := WorkflowUpdateRequest{
		WorkflowName: wf.Name,
		Namespace:    wf.Namespace,
	}
	wf, err = server.TerminateWorkflow(context.TODO(), &rsmWfReq)
	assert.NotNil(t, wf)
	assert.Equal(t, int64(0), *wf.Spec.ActiveDeadlineSeconds)
	assert.Nil(t, err)

	rsmWfReq = WorkflowUpdateRequest{
		WorkflowName: "hello-world-9tql2-not",
		Namespace:    "workflows",
	}
	wf, err = server.TerminateWorkflow(context.TODO(), &rsmWfReq)
	assert.Nil(t, wf)
	assert.NotNil(t, err)
}
